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

func HandleClose(handle Handle) error {
	if status := Sys_handle_close(handle); status != ErrOk {
		return status
	}
	return nil
}

func HandleDuplicate(handle Handle, rights Rights) (Handle, error) {
	dupHandle := Sys_handle_duplicate(handle, rights)
	if dupHandle <= 0 {
		return 0, Status(dupHandle)
	}
	return dupHandle, nil
}

func HandleWaitOne(handle Handle, signals Signals, timeout Time) (SignalsState, error) {
	var sigState SignalsState
	status := Sys_handle_wait_one(handle, signals, timeout, &sigState)
	if status != ErrOk {
		return sigState, status
	}
	return sigState, nil
}

func HandleWaitMany(handles []Handle, signals []Signals, timeout Time) (uint32, []SignalsState, error) {
	if len(handles) != len(signals) || len(handles) < 1 {
		return 0, nil, ErrInvalidArgs
	}
	var resultIndex uint32
	resultSignals := make([]SignalsState, len(handles))
	status := Sys_handle_wait_many(uint32(len(handles)), &handles[0], &signals[0], timeout, &resultIndex, &resultSignals[0])
	if status != ErrOk {
		return resultIndex, resultSignals, status
	}
	return resultIndex, resultSignals, nil
}

func MsgPipeCreate(flags uint32) ([]Handle, error) {
	handles := make([]Handle, 2)
	status := Sys_msgpipe_create(&handles[0], flags)
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
	if status := Sys_msgpipe_read(handle, dataPtr, &numBytes, handlePtr, &numHandles, flags); status != ErrOk {
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
	status := Sys_msgpipe_write(handle, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)), flags)
	if status != ErrOk {
		return status
	}
	return nil
}
