// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mx

import (
	"unsafe"
)

// Process-wide MXIO handles.
var (
	RootHandle  Handle
	StdioHandle Handle
	CwdHandle   Handle
	ProcHandle  Handle
)

func (h Handle) Close() error {
	if status := Sys_handle_close(h); status != ErrOk {
		return status
	}
	return nil
}

func (h Handle) Duplicate(rights Rights) (Handle, error) {
	var dup Handle
	if status := Sys_handle_duplicate(h, rights, &dup); status != ErrOk {
		return 0, status
	}
	return dup, nil
}

func (h Handle) WaitOne(waitFor Signals, timeout Time) (observed Signals, err error) {
	if status := Sys_handle_wait_one(h, waitFor, timeout, &observed); status != ErrOk {
		return observed, status
	}
	return observed, nil
}

func WaitMany(items []WaitItem, timeout Time) error {
	if status := Sys_handle_wait_many(&items[0], uint32(len(items)), timeout); status != ErrOk {
		return status
	}
	return nil
}

// TODO: s/MsgPipe/Channel/
func MsgPipeCreate(flags uint32) ([]Handle, error) {
	handles := make([]Handle, 2)
	status := Sys_channel_create(flags, &handles[0], &handles[1])
	if status != ErrOk {
		return nil, status
	}
	return handles, nil
}

func MsgPipeRead(handle Handle, data []byte, handles []Handle, flags uint32) (numBytes, numHandles uint32, err error) {
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
	if status := Sys_channel_read(handle, flags, dataPtr, numBytes, &numBytes, handlePtr, numHandles, &numHandles); status != ErrOk {
		err = status
	}
	return
}

func MsgPipeWrite(handle Handle, data []byte, handles []Handle, flags uint32) error {
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	status := Sys_channel_write(handle, flags, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)))
	if status != ErrOk {
		return status
	}
	return nil
}
