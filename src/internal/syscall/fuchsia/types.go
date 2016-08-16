// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fuchsia

// Allow conversion from status to error
func (s Status) Error() string {
	return "errno(" + string(s) + ")"
}

const (
	ErrOk = Status(-iota)
	ErrInternal
	ErrNotSupported
	ErrNotFound
	ErrNoMemory
	ErrNoResources
	_ // -6
	_ // -7
	_ // -8
	_ // -9
	ErrInvalidArgs
	ErrBadSyscall
	ErrBadHandle
	ErrOutOfRange
	ErrNotEnoughBuffer
	ErrAlreadyExists
	ErrAlreadyBound
	_ // -17
	_ // -18
	_ // -19
	ErrBadState
	ErrFlowControl
	ErrNotReady
	ErrTimedOut
	ErrHandleClosed
	ErrChannelClosed
	ErrNotAvailable
	_ // -27
	_ // -28
	_ // -29
	ErrAccessDenied
	_ // -31
	_ // -32
	_ // -33
	_ // -34
	_ // -35
	_ // -36
	_ // -37
	_ // -38
	_ // -39
	ErrIO
	ErrI2CNack
	ErrChecksumFail
	_ // -43
	_ // -44
	_ // -45
	_ // -46
	_ // -47
	_ // -48
	_ // -49
	ErrBadPath
	ErrNotDir
	ErrNotFile
	ErrRecurseTooDeep
	ErrWrongType
)

const (
	SignalReadable = Signals(1 << iota)
	SignalWritable
	SignalPeerClosed
	SignalSignaled
	SignalUser0
	SignalUser1
	SignalUser2
	SignalUser3
	SignalUserAll = Signals(15 << 4)
	SignalNone    = Signals(0)
)

const (
	RightDuplicate = Rights(1 << iota)
	RightTransfer
	RightRead
	RightWrite
	RightExecute
	RightDebug
	RightSameRights = Rights(1 << 31)
	RightNone       = Rights(0)
)

const (
	TimensecInfinite = Time(0xFFFFFFFFFFFFFFFF)
)
