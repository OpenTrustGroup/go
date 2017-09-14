// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"testing"
	"unsafe"

	"syscall/mx"
)

// Test creating a dispatcher which both sums integers.
// Writing messages to one end of a handle added to the dispatcher should
// (a) Add the value of the message to a sum, and
// (b) Return the sum back on a channel
// Closing the handle will subtract 1 from the sum.
func TestDispatcher(t *testing.T) {
	c := make(chan int64)
	sum := int64(0)
	sumCb := func(val int64) {
		sum += val
		c <- sum
	}

	expectedCookie := int64(1234)

	dispatcherCb := func(h mx.Handle, cb interface{}, cookie int64) error {
		if cookie != expectedCookie {
			t.Fatalf("cookie=%q, want %q", cookie, expectedCookie)
		}

		// The type of the handler callback is independent of the underlying dispatcher.
		// However, it must match the callback passed to AddHandler.
		handlerCb, ok := cb.(func(value int64))
		if !ok {
			t.Fatal("cannot convert handler callback to func(value int64)")
		}
		if h == 0 {
			// Indicates the remote side was closed
			handlerCb(-1)
		} else {
			data := make([]byte, 8)
			ch := &mx.Channel{h}
			numBytes, numHandles, err := ch.Read(data, nil, 0)
			if err != nil {
				if err == mx.ErrBadState {
					// This error implies there are no messages currently in the pipe
					return ErrNoWork
				}
				return err
			} else if numBytes != 8 || numHandles != 0 {
				t.Fatal("unexpected message read in dispatcher callback")
			}
			val := *(*int64)(unsafe.Pointer(&data[0]))
			handlerCb(val)
		}
		return nil
	}

	d, err := NewDispatcher(dispatcherCb)
	if err != nil {
		panic(err)
	}
	go d.Serve()
	h0, h1, err := mx.NewChannel(0)
	if err != nil {
		panic(err)
	}

	if err := d.AddHandler(h0.Handle, sumCb, expectedCookie); err != nil {
		panic(err)
	}

	if sum != 0 {
		t.Fatal("unexpected starting sum value, %d", sum)
	}

	writeMessageVerifySum := func(val, expectedSum int64) {
		buf := (*[8]byte)(unsafe.Pointer(&val))[:]
		if err := h1.Write(buf, nil, 0); err != nil {
			panic(err)
		}
		seenSum := <-c
		if sum != seenSum || seenSum != expectedSum {
			t.Fatal("sum=%d, seenSum=%d, want %d", sum, seenSum, expectedSum)
		}
	}
	writeMessageVerifySum(1, 1)
	writeMessageVerifySum(3, 4)
	writeMessageVerifySum(6, 10)
	writeMessageVerifySum(15, 25)

	if err := h1.Close(); err != nil {
		t.Fatal(err)
	}

	// Closing the handle should decrease the sum by 1.
	seenSum := <-c
	if want := int64(24); sum != seenSum || seenSum != want {
		t.Fatal("closing sum=%d, seenSum=%d, want %d", sum, seenSum, want)
	}
}

// TODO(smklein): Test multiple handlers
