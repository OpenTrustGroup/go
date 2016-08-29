// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// TODO(smklein): Convert this program to a go test once we can run `go test` on Magenta, and place
// it in the same package as the dispatcher.
package main

import (
	"unsafe"

	"internal/syscall/fuchsia"
	"internal/syscall/fuchsia/mxio/dispatcher"
)

// Test creating a dispatcher which both sums integers.
// Writing messages to one end of a handle added to the dispatcher should
// (a) Add the value of the message to a sum, and
// (b) Return the sum back on a channel
// Closing the handle will subtract 1 from the sum.
func main() {
	println("Dispatcher Test Executable Start...")
	c := make(chan int64)
	sum := int64(0)
	sumCb := func(val int64) {
		sum += val
		c <- sum
	}

	expectedCookie := int64(1234)

	dispatcherCb := func(h fuchsia.Handle, cb interface{}, cookie int64) error {
		if cookie != expectedCookie {
			panic("Unexpected cookie")
		}

		// The type of the handler callback is independent of the underlying dispatcher.
		// However, it must match the callback passed to AddHandler.
		handlerCb, ok := cb.(func(value int64))
		if !ok {
			panic("Cannot convert handler callback to func(value int64)")
		}
		if h == 0 {
			// Indicates the remote side was closed
			handlerCb(-1)
		} else {
			data := make([]byte, 8)
			numBytes, numHandles, err := fuchsia.MessageRead(h, data, nil, 0)
			if err != nil {
				if err == fuchsia.ErrBadState {
					// This error implies there are no messages currently in the pipe
					return dispatcher.ErrNoWork
				}
				return err
			} else if numBytes != 8 || numHandles != 0 {
				panic("Unexpected message read in dispatcher callback")
			}
			val := *(*int64)(unsafe.Pointer(&data[0]))
			handlerCb(val)
		}
		return nil
	}

	d, err := dispatcher.New(dispatcherCb)
	if err != nil {
		panic(err)
	}
	go d.Serve()
	handles, err := fuchsia.MessagePipeCreate(0)
	if err != nil {
		panic(err)
	} else if len(handles) != 2 {
		panic("Unexpected number of handles")
	}

	if err := d.AddHandler(handles[0], sumCb, expectedCookie); err != nil {
		panic(err)
	}

	if sum != 0 {
		panic("Unexpected starting sum value")
	}

	writeMessageVerifySum := func(val, expectedSum int64) {
		buf := (*[8]byte)(unsafe.Pointer(&val))[:]
		if err := fuchsia.MessageWrite(handles[1], buf, nil, 0); err != nil {
			panic(err)
		}
		seenSum := <-c
		if sum != seenSum || seenSum != expectedSum {
			panic("Unexpected sum value")
		}
	}
	writeMessageVerifySum(1, 1)
	writeMessageVerifySum(3, 4)
	writeMessageVerifySum(6, 10)
	writeMessageVerifySum(15, 25)

	if err := fuchsia.HandleClose(handles[1]); err != nil {
		panic(err)
	}

	// Closing the handle should decrease the sum by 1.
	seenSum := <-c
	if sum != seenSum || seenSum != 24 {
		panic("Unexpected closing sum value")
	}

	println("... Dispatcher test executable success")
}

// TODO(smklein): Test multiple handlers
// TODO(smklein): Test reply pipes
