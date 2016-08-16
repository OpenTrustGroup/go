// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package mxio provides unix-style IO, implementing Magenta's "system/ulib/mxio"
package mxio

import (
	"unsafe"

	"internal/syscall/fuchsia"
)

// MXIO provides classic unix-style IO over various transports
type MXIO interface {
	Read(data []byte) (int, error)
	Write(data []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
	Open(path string, flags int32, mode uint32) (MXIO, error)
	Clone() ([]fuchsia.Handle, error)
	Ioctl(op uint32, in []byte) ([]byte, []fuchsia.Handle, error)
	// TODO(smklein): Add missing ops, like "Misc" and "Wait"
}

const (
	PacketSize = uint(unsafe.Sizeof(Packet{}))
	// MaxHandles is the maximum number of handles which can be transferred in an mxio op.
	MaxHandles = 3
	// ChunkSize is the largest number of bytes which can be used in a single low-level op.
	ChunkSize = 8192
	// IoctlMaxInput is the largest acceptable size for an Ioctl op input.
	IoctlMaxInput = 1024
	// The operation number of an MXIO ioctl to request a handle.
	IoctlDeviceGetHandle = 0x7FFF0001
)

// PacketHeader describes the following packet
type PacketHeader struct {
	Key        uint64
	PacketType uint32
	Extra      uint32
}

// Packet describes the unit of communication via an MXIO protocol
type Packet struct {
	Hdr       PacketHeader
	Timestamp fuchsia.Time
	Bytes     uintptr
	Signals   fuchsia.Signals
	Reserved  uint32
}

type Protocol uint32

// All available MXIO protocols
const (
	ProtocolUndefined = Protocol(iota)
	// The PIPE protocol uses message ports as simple, no-flow-control io pipes.
	ProtocolPipe
	// The REMOTEIO protocol uses message ports to implement synchronous remoting of
	// read/write/close operations.
	ProtocolRemote
)

// Other MXIO related constants
const (
	// A flag to MessagePipeCreate, identifying that MessageWrite MUST include itself as the last
	// handle being transferred. Handles[1] will be created as a reply pipe, but Handles[0] will be
	// a regular pipe.
	FlagReplyPipe = 1 << 0
)

// IoPortCreate wraps the corresponding syscall
func IoPortCreate(options uint32) (fuchsia.Handle, fuchsia.Status) {
	handle := fuchsia.Sys_io_port_create(options)
	if handle <= 0 {
		return 0, fuchsia.Status(handle)
	}
	return handle, fuchsia.ErrOk
}

// IoPortQueue wraps the corresponding syscall
func IoPortQueue(handle fuchsia.Handle, packet *Packet, size uint) fuchsia.Status {
	return fuchsia.Sys_io_port_queue(handle, unsafe.Pointer(packet), size)
}

// IoPortWait wraps the corresponding syscall
func IoPortWait(handle fuchsia.Handle, size uint) (*Packet, fuchsia.Status) {
	var packet Packet
	status := fuchsia.Sys_io_port_wait(handle, unsafe.Pointer(&packet), size)
	if status != fuchsia.ErrOk {
		return nil, status
	}
	return &packet, fuchsia.ErrOk
}

// IoPortBind wraps the corresponding syscall
func IoPortBind(handle fuchsia.Handle, key uint64, source fuchsia.Handle, signals fuchsia.Signals) fuchsia.Status {
	return fuchsia.Sys_io_port_bind(handle, key, source, signals)
}
