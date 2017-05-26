// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package mxio provides unix-style IO, implementing Magenta's "system/ulib/mxio"
package mxio

import (
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
	Clone() ([]mx.Handle, error) // TODO(smklein): Pass type back from Clone
	Ioctl(op uint32, in, out []byte) ([]mx.Handle, error)
	IoctlSetHandle(op uint32, in mx.Handle) error
	Misc(op uint32, off int64, in, out []byte, handles []mx.Handle) (n int, err error)
	// TODO(smklein): Add missing ops, like "Wait"
}

const (
	// MaxHandles is the maximum number of handles which can be transferred in an mxio op.
	MaxHandles = 3
	// ChunkSize is the largest number of bytes which can be used in a single low-level op.
	ChunkSize = 8192
	// IoctlMaxInput is the largest acceptable size for an Ioctl op input.
	IoctlMaxInput = 1024
	// maximum number of bytes in a path
	PathMax = 4096
)

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
	// The SOCKET protocol provides socket access.
	ProtocolSocket
	// The SERVICE protocol provides generic services.
	ProtocolService
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
	IoctlKindDefault         = 0x0 // IOCTL_KIND_DEFAULT
	IoctlKindGetHandle       = 0x1 // IOCTL_KIND_GET_HANDLE
	IoctlKindGetTwoHandles   = 0x2 // IOCTL_KIND_GET_TWO_HANDLES
	IoctlKindSetHandle       = 0x3 // IOCTL_KIND_SET_HANDLE
	IoctlKindGetThreeHandles = 0x4 // IOCTL_KIND_GET_THREE_HANDLES
)

const (
	IoctlFamilyReserved = 0x00 // IOCTL_FAMILY_RESERVED
	IoctlFamilyDevice   = 0x01 // IOCTL_FAMILY_DEVICE
	IoctlFamilyDevmgr   = 0x02 // IOCTL_FAMILY_VFS, TODO rename to IoctlFamilyVFS
	IoctlFamilyDMCTL    = 0x03 // IOCTL_FAMILY_DMCTL
	IoctlFamilyTest     = 0x04 // IOCTL_FAMILY_TEST
)

const (
	IoctlFamilyBlock     = 0x13 // IOCTL_FAMILY_BLOCK
	IoctlFamilyNetconfig = 0x26 // IOCTL_FAMILY_NETCONFIG
)

var IoctlDeviceBind = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 0)
var IoctlDeviceWatchDir = IoctlNum(IoctlKindGetHandle, IoctlFamilyDevice, 1)
var IoctlDeviceGetEventHandle = IoctlNum(IoctlKindGetHandle, IoctlFamilyDevice, 2)
var IoctlDeviceGetDriverName = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 3)
var IoctlDeviceGetDeviceName = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 4)
var IoctlDeviceDebugSuspend = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 5)
var IoctlDeviceDebugResume = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 6)
var IoctlDeviceSync = IoctlNum(IoctlKindDefault, IoctlFamilyDevice, 7)

var IoctlDevmgrMountFS = IoctlNum(IoctlKindSetHandle, IoctlFamilyDevmgr, 0)
var IoctlDevmgrUnmountFS = IoctlNum(IoctlKindDefault, IoctlFamilyDevmgr, 1)
var IoctlDevmgrGetTokenFS = IoctlNum(IoctlKindGetHandle, IoctlFamilyDevmgr, 5)

type Vnattr struct {
	Valid      uint32
	Mode       Vtype
	Inode      uint64
	Size       uint64
	Blksize    uint64
	Blkcount   uint64
	Nlink      uint64
	CreateTime uint64
	ModifyTime uint64
}

// VnattrBlksize is the size of block used in "Blockcount", which may be distinct from Blksize.
const VnattrBlksize = 512

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

const (
	HandleTypeCWD    = 0x31
	HandleTypeRemote = 0x32
	HandleTypePipe   = 0x33
	HandleTypeEvent  = 0x34
	HandleTypeLogger = 0x35
)
