// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mx

type Status int32   // mx_status_t
type Handle int32   // mx_handle_t
type Time uint64    // mx_time_t, nanoseconds
type Signals uint32 // mx_signals_t
type Rights uint32  // mx_rights_t
type Paddr uintptr  // mx_paddr_t

type WaitSetResult struct {
	Cookie     uint64
	WaitResult Status
	Observed   Signals
} // mx_waitset_result_t

type WaitItem struct {
	Handle  Handle
	WaitFor Signals
	Pending Signals
} // mx_wait_item_t

// Status codes. See magenta/system/public/magenta/errors.h
const (
	ErrOk Status = -iota
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
	_ // -21
	_ // -22
	ErrTimedOut
	ErrHandleClosed
	ErrRemoteClosed
	ErrUnavailable
	ErrShouldWait
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
	ErrIORefused
	ErrIODataIntegrity
	ErrIODataLoss
	_ // -44
	_ // -45
	_ // -46
	_ // -47
	_ // -48
	_ // -49
	ErrBadPath
	ErrNotDir
	ErrNotFile
	_ // -53
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

// EOF is used like io.EOF.
//
// When package io is linked in, at initialization this value is
// rewritten to be equivalent to io.EOF, so that the io.Readers
// and io.Writers in this package meet their specs.
var EOF error = eof{}

type eof struct{}

func (eof) Error() string { return "mx.EOF" }
