// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fuchsia

import (
	"unsafe"
)

// Allow conversion from status to error
func (s Status) Error() string {
	return "errno(" + string(s) + ")"
}

const (
	ErrOk       = Status(0)
	ErrInternal = Status(-iota)
	ErrNotSupported
	ErrNotFound
	ErrNoMemory
	ErrNoResources
	ErrInvalidArgs
	ErrBadSyscall
	ErrBadHandle
	ErrOutOfRange
	ErrNotEnoughBuffer
	ErrAlreadyExists
	ErrBadState
	ErrFlowControl
	ErrNotReady
	ErrTimedOut
	ErrHandleClosed
	ErrCancelled
	ErrChannelClosed
	ErrNotAvailable
	ErrAccessDenied
	ErrIO
	ErrChecksumFail
	ErrBadPath
	ErrNotDir
	ErrNotFile
	ErrRecurseTooDeep
)

const (
	SignalNone     = Signals(0)
	SignalReadable = Signals(1 << iota)
	SignalWritable
	SignalPeerClosed
	SignalSignaled
	SignalUser0
	SignalUser1
	SignalUser2
	SignalUser3
	SignalUserAll = Signals(15 << 4)
)

const (
	RightNone      = Rights(0)
	RightDuplicate = Rights(1 << iota)
	RightTransfer
	RightRead
	RightWrite
	RightExecute
	RightDebug
	RightSameRights = Rights(1 << 31)
)

const (
	TimensecInfinite = Time(0xFFFFFFFFFFFFFFFF)
)

const (
	MxioPacketSize = uint(unsafe.Sizeof(MxioPacket{}))
	// MxioMaxHandles is the maximum number of handles which can be transferred in an mxio op.
	MxioMaxHandles = 3
	// MxioChunkSize is the largest number of bytes which can be used in a single low-level op.
	MxioChunkSize = 8192
	// MxioIoctlMaxInput is the largest acceptable size for an Ioctl op input.
	MxioIoctlMaxInput = 1024
	// The operation number of an MXIO ioctl to request a handle.
	IoctlDeviceGetHandle = 0x7FFF0001
)

// MxioPacketHeader describes the following packet
type MxioPacketHeader struct {
	Key        uint64
	PacketType uint32
	Extra      uint32
}

// MxioPacket describes the unit of communication via an MXIO protocol
type MxioPacket struct {
	Hdr       MxioPacketHeader
	Timestamp Time
	Bytes     uintptr
	Signals   Signals
	Reserved  uint32
}

// All available MXIO protocols
const (
	ProtoclUndefined = iota
	ProtocolPipe
	ProtocolRemote
)

// Other MXIO related constants
const (
	// A flag to MessagePipeCreate, identifying that MessageWrite MUST include itself as the last
	// handle being transferred. Handles[1] will be created as a reply pipe, but Handles[0] will be
	// a regular pipe.
	FlagReplyPipe = 1 << 0
)
