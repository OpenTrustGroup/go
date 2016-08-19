// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fuchsia

import (
	"unsafe"
)

func HandleClose(handle Handle) Status {
	errno := Sys_handle_close(handle)
	if errno != 0 {
		return errno
	}
	return ErrOk
}

func HandleDuplicate(handle Handle, rights Rights) (Handle, Status) {
	dupHandle := Sys_handle_duplicate(handle, rights)
	if dupHandle <= 0 {
		return 0, Status(dupHandle)
	}
	return dupHandle, ErrOk
}

func HandleWaitOne(handle Handle, signals Signals, timeout Time) (SignalsState, Status) {
	var sigState SignalsState
	status := Sys_handle_wait_one(handle, signals, timeout, &sigState)
	return sigState, status
}

func HandleWaitMany(handles []Handle, signals []Signals, timeout Time) (uint32, []SignalsState, Status) {
	if len(handles) != len(signals) || len(handles) < 1 {
		return 0, nil, ErrInvalidArgs
	}
	var resultIndex uint32
	resultSignals := make([]SignalsState, len(handles))
	status := Sys_handle_wait_many(uint32(len(handles)), &handles[0], &signals[0], timeout, &resultIndex, &resultSignals[0])
	return resultIndex, resultSignals, status
}

func MessagePipeCreate(flags uint32) ([]Handle, Status) {
	handles := make([]Handle, 2)
	status := Sys_message_pipe_create(&handles[0], flags)
	if status != ErrOk {
		return nil, status
	}
	return handles, ErrOk
}

func MessageRead(handle Handle, data []byte, handles []Handle, flags uint32) (numBytes, numHandles uint32, status Status) {
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
	status = Sys_message_read(handle, dataPtr, &numBytes, handlePtr, &numHandles, flags)
	return
}

func MessageWrite(handle Handle, data []byte, handles []Handle, flags uint32) Status {
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	return Sys_message_write(handle, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)), flags)
}

func IoPortCreate(options uint32) (Handle, Status) {
	handle := Sys_io_port_create(options)
	if handle <= 0 {
		return 0, Status(handle)
	}
	return handle, ErrOk
}

func IoPortQueue(handle Handle, packet *MxioPacket, size uint) Status {
	return Sys_io_port_queue(handle, unsafe.Pointer(packet), size)
}

func IoPortWait(handle Handle, size uint) (*MxioPacket, Status) {
	var packet MxioPacket
	status := Sys_io_port_wait(handle, unsafe.Pointer(&packet), size)
	if status != ErrOk {
		return nil, status
	}
	return &packet, ErrOk
}

func IoPortBind(handle Handle, key uint64, source Handle, signals Signals) Status {
	return Sys_io_port_bind(handle, key, source, signals)
}
