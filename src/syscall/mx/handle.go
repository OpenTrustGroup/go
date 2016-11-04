// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mx

import "unsafe"

// Process-wide MXIO handles.
var (
	RootHandle  Handle
	StdioHandle Handle
	CwdHandle   Handle
	ProcHandle  Handle
)

func (h Handle) Close() error {
	if status := sys_handle_close(h); status != ErrOk {
		return status
	}
	return nil
}

func (h Handle) Duplicate(rights Rights) (Handle, error) {
	var dup Handle
	if status := sys_handle_duplicate(h, rights, &dup); status != ErrOk {
		return 0, status
	}
	return dup, nil
}

func (h Handle) WaitOne(waitFor Signals, timeout Time) (observed Signals, err error) {
	if status := sys_handle_wait_one(h, waitFor, timeout, &observed); status != ErrOk {
		return observed, status
	}
	return observed, nil
}

func WaitMany(items []WaitItem, timeout Time) error {
	if status := sys_handle_wait_many(&items[0], uint32(len(items)), timeout); status != ErrOk {
		return status
	}
	return nil
}

func NewChannel(flags uint32) (h0, h1 Handle, err error) {
	if status := sys_channel_create(flags, &h0, &h1); status != ErrOk {
		return 0, 0, status
	}
	return h0, h1, nil
}

func NewSocket(flags uint32) (h0, h1 Handle, err error) {
	if status := sys_socket_create(flags, &h0, &h1); status != ErrOk {
		return 0, 0, status
	}
	return h0, h1, nil
}

func (h Handle) ReadSocket(data []byte, flags uint32) (n int, err error) {
	var actual uint
	status := sys_socket_read(h, 0, unsafe.Pointer(&data[0]), uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (h Handle) WriteSocket(data []byte, flags uint32) (n int, err error) {
	var actual uint
	status := sys_socket_write(h, 0, unsafe.Pointer(&data[0]), uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (h Handle) ReadChannel(data []byte, handles []Handle, flags uint32) (numBytes, numHandles uint32, err error) {
	numBytes = uint32(len(data))
	numHandles = uint32(len(handles))
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	if status := sys_channel_read(h, flags, dataPtr, numBytes, &numBytes, handlePtr, numHandles, &numHandles); status != ErrOk {
		err = status
	}
	return
}

func (h Handle) WriteChannel(data []byte, handles []Handle, flags uint32) error {
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	status := sys_channel_write(h, flags, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)))
	if status != ErrOk {
		return status
	}
	return nil
}
