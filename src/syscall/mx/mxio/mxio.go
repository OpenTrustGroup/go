// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package mxio provides unix-style IO, implementing Magenta's "system/ulib/mxio"
package mxio

import (
	"unsafe"

	"syscall/mx"
)

// MXIO provides classic unix-style IO over various transports
type MXIO interface {
	Read(data []byte) (int, error)
	ReadAt(data []byte, off int64) (int, error)
	Write(data []byte) (int, error)
	WriteAt(data []byte, off int64) (int, error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
	Open(path string, flags int32, mode uint32) (MXIO, error)
	Clone() ([]mx.Handle, error)
	Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error)
	Misc(op uint32, off int64, in, out []byte) (n int, err error)
	// TODO(smklein): Add missing ops, like "Wait"
}

const (
	PacketSize = uint(unsafe.Sizeof(Packet{}))
	// MaxHandles is the maximum number of handles which can be transferred in an mxio op.
	MaxHandles = 3
	// ChunkSize is the largest number of bytes which can be used in a single low-level op.
	ChunkSize = 8192
	// IoctlMaxInput is the largest acceptable size for an Ioctl op input.
	IoctlMaxInput = 1024
)

const (
	PortPacketTypeKernel = iota
	PortPacketTypeIosn
	PortPacketTypeUser
	PortPacketTypeException
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
	Timestamp mx.Time
	Bytes     uintptr
	Signals   mx.Signals
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
	// The VMOFILE protocol accesses a file as a VMO.
	ProtocolVMOFile
)

// Other MXIO related constants
const (
	// A flag to MsgPipeCreate, identifying that MsgPipeWrite MUST include itself as the last
	// handle being transferred. Handles[1] will be created as a reply pipe, but Handles[0] will be
	// a regular pipe.
	FlagReplyPipe = 1 << 0
)

// MXIO Ioctls
func IoctlNum(kind, family, number uint32) uint32 {
	return ((kind & 0xF) << 20) | ((family & 0xFF) << 8) | (number & 0xFF)
}
func IoctlKind(ioctl uint32) uint32 {
	return (ioctl >> 20) & 0xF
}
func IoctlFamily(ioctl uint32) uint32 {
	return (ioctl >> 8) & 0xFF
}
func IoctlNumber(ioctl uint32) uint32 {
	return ioctl & 0xFF
}

const (
	IoctlKindDefault = iota
	IoctlKindGetHandle
)

const (
	IoctlFamilyReserved = iota
	IoctlFamilyDevice
	IoctlFamilyDevmgr
)

var IoctlDevmgrMountFS = IoctlNum(IoctlKindGetHandle, IoctlFamilyDevmgr, 0)
var IoctlDevmgrUnmountFS = IoctlNum(IoctlKindDefault, IoctlFamilyDevmgr, 1)

// IoPortCreate wraps the corresponding syscall
func IoPortCreate(options uint32) (mx.Handle, error) {
	var h mx.Handle
	if status := mx.Sys_port_create(options, &h); status != mx.ErrOk {
		return 0, status
	}
	return h, nil
}

// IoPortQueue wraps the corresponding syscall
func IoPortQueue(handle mx.Handle, packet *Packet, size uint) error {
	if status := mx.Sys_port_queue(handle, unsafe.Pointer(packet), size); status != mx.ErrOk {
		return status
	}
	return nil
}

// IoPortWait wraps the corresponding syscall
func IoPortWait(handle mx.Handle, timeout mx.Time, size uint) (*Packet, error) {
	var packet Packet
	status := mx.Sys_port_wait(handle, timeout, unsafe.Pointer(&packet), size)
	if status != mx.ErrOk {
		return nil, status
	}
	return &packet, nil
}

// IoPortBind wraps the corresponding syscall
func IoPortBind(handle mx.Handle, key uint64, source mx.Handle, signals mx.Signals) error {
	if status := mx.Sys_port_bind(handle, key, source, signals); status != mx.ErrOk {
		return status
	}
	return nil
}

type Vnattr struct {
	Valid      uint32
	Mode       Vtype
	Inode      uint64
	Size       uint64
	CreateTime uint64
	ModifyTime uint64
}

const (
	AttrCtime = 1 << iota
	AttrMtime
	AttrAtime
)

type Vtype uint32

// bit-compatible with POSIX stat
const (
	VtypeMask Vtype = 0170000
	VtypeSock Vtype = 0140000
	VtypeLink Vtype = 0120000
	VtypeFile Vtype = 0100000
	VtypeBdev Vtype = 0060000
	VtypeDir  Vtype = 0040000
	VtypeCdev Vtype = 0020000
	VtypePipe Vtype = 0010000

	VtypeISUID Vtype = 0004000
	VtypeISGID Vtype = 0002000
	VtypeISVTX Vtype = 0001000
	VtypeIRWXU Vtype = 0000700
	VtypeIRUSR Vtype = 0000400
	VtypeIWUSR Vtype = 0000200
	VtypeIXUSR Vtype = 0000100
	VtypeIRWXG Vtype = 0000070
	VtypeIRGRP Vtype = 0000040
	VtypeIWGRP Vtype = 0000020
	VtypeIXGRP Vtype = 0000010
	VtypeIRWXO Vtype = 0000007
	VtypeIROTH Vtype = 0000004
	VtypeIWOTH Vtype = 0000002
	VtypeIXOTH Vtype = 0000001
)
