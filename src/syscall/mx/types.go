// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mx

import "unsafe"

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

const MX_MAX_NAME_LEN = 32

const (
	MX_RREC_DELETED = 0
	MX_RREC_SELF    = 1
	MX_RREC_DATA    = 2
	MX_RREC_IRQ     = 3
	MX_RREC_MMIO    = 4
	MX_RREC_IOPORT  = 5
)

const (
	MX_RACT_ENABLE  = 1
	MX_RACT_DISABLE = 2
)

const (
	MX_RREC_SELF_GENERIC = 0
	MX_RREC_SELF_ROOT    = 1
)

type RrecSelf struct {
	Type        uint16 // MX_RREC_SELF
	Subtype     uint16
	Options     uint32
	Koid        uint64
	RecordCount uint32
	ChildCount  uint32
	_           [2]uint32
	Name        [MX_MAX_NAME_LEN]uint8
} // mx_rrec_self_t

type RrecMMIO struct {
	Type     uint16 // MX_RREC_MMIO
	Subtype  uint16
	Options  uint32
	PhysBase uint64
	PhysSize uint64
	_        [10]uint32
} // mx_rrec_mmio_t

type RrecIRQ struct {
	Type     uint16 // MX_RREC_IRQ
	Subtype  uint16
	Options  uint32
	IRQBase  uint32
	IRQCount uint32
	_        [12]uint32
} // mx_rrec_irq_t

type RrecIOPort struct {
	Type      uint16 // MX_RREC_IOPORT
	Subtype   uint16
	Options   uint32
	PortBase  uint32
	PortCount uint32
	_         [12]uint32
} // mx_rrec_ioport_t

type RrecData struct {
	Type    uint16
	Subtype uint16
	Options uint32 // low 4 bits are count
	Data    [56]uint8
} // mx_rrec_data_t

type Rrec struct {
	typ uint16
	_   [62]uint8
} // mx_rrec_t

func (r *Rrec) Box() RrecValue {
	switch r.typ {
	case MX_RREC_DELETED:
		return nil
	case MX_RREC_SELF:
		return (*RrecSelf)(unsafe.Pointer(r))
	case MX_RREC_DATA:
		return (*RrecData)(unsafe.Pointer(r))
	case MX_RREC_IRQ:
		return (*RrecIRQ)(unsafe.Pointer(r))
	case MX_RREC_MMIO:
		return (*RrecMMIO)(unsafe.Pointer(r))
	case MX_RREC_IOPORT:
		return (*RrecIOPort)(unsafe.Pointer(r))
	default:
		return nil
	}
}

type RrecValue interface {
	rrec()
}

func (*RrecSelf) rrec()   {}
func (*RrecData) rrec()   {}
func (*RrecIRQ) rrec()    {}
func (*RrecMMIO) rrec()   {}
func (*RrecIOPort) rrec() {}

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
