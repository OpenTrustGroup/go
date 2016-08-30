// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rio

import (
	"unsafe"

	"internal/syscall/fuchsia"
	"internal/syscall/fuchsia/mxio"
)

const (
	// MessageMagic is the magic number to verify the RIO message
	MessageMagic = 0x024F4952

	// MessageChunkSize is the largest size of 'data' in an RIO message
	MessageChunkSize = 8192

	// MessageSizeMax is the largest size of an MXIO message
	MessageSizeMax = uint32(unsafe.Sizeof(Msg{}))

	// MessageHeaderSize is the size of a RIO message without 'data'
	MessageHeaderSize = uint32(unsafe.Sizeof(MsgHeader{}))

	// MessageMaxHandles is the maximum number of handles used by open/clone/create
	MessageMaxHandles = 3
)

// Values for op. Each one will signify a request for a different operation.
const (
	OpStatus = uint32(iota)
	OpClose
	OpClone
	OpOpen
	OpMisc
	OpRead
	OpWrite
	OpSeek
	OpStat
	OpReaddir
	OpIoctl
	OpUnlink
	OpReadAt
	OpWriteAt
	OpNumOps // The total number of operations

	OpReplyPipe = 0x01000000
)

// TODO(smklein): Confirm padding compatibility when communicating with C code implementing RIO.
type MsgHeader struct {
	Magic    uint32
	op       uint32
	Datalen  uint32
	Arg      int32
	arg2     int64
	Reserved int32
	Hcount   uint32
	Handle   [MessageMaxHandles + 1]fuchsia.Handle // 3 handles + reply pipe
}

// Msg is the message sent in the Remote IO protocol
type Msg struct {
	MsgHeader
	Data [MessageChunkSize]uint8
}

// ReadMsg reads a message from a handle
func (m *Msg) ReadMsg(h fuchsia.Handle) (numBytes, numHandles uint32, err error) {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	numBytes, numHandles, err = fuchsia.MessageRead(h, msgBytes[:], m.Handle[:], 0)
	m.Hcount = numHandles
	return
}

// WriteMsg writes a message to a handle
func (m *Msg) WriteMsg(h fuchsia.Handle) error {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))

	data := msgBytes[:MessageHeaderSize+m.Datalen]
	var handles []fuchsia.Handle
	if m.Hcount > 0 {
		handles = m.Handle[:m.Hcount]
	}
	return fuchsia.MessageWrite(h, data, handles, 0)
}

// HasReplyPipe returns if the message contains a reply pipe
func (m *Msg) HasReplyPipe() bool {
	return (m.op&OpReplyPipe != 0)
}

// SetReplyPipe identifies that the message contains a reply pipe
func (m *Msg) SetReplyPipe() {
	m.op |= OpReplyPipe
}

func (m *Msg) Op() uint32 {
	return m.op & 0xFFFF
}
func (m *Msg) SetOp(v uint32) {
	m.op = (v & 0xFFFF)
}
func (m *Msg) Off() int64 {
	return m.arg2
}
func (m *Msg) SetOff(v int64) {
	m.arg2 = v
}
func (m *Msg) Mode() uint32 {
	return uint32(m.arg2)
}
func (m *Msg) SetMode(v uint32) {
	m.arg2 = int64(v)
}
func (m *Msg) Protocol() mxio.Protocol {
	return mxio.Protocol(m.arg2)
}
func (m *Msg) SetProtocol(v uint32) {
	m.arg2 = int64(v)
}
func (m *Msg) IoctlOp() uint32 {
	return uint32(m.arg2)
}
func (m *Msg) SetIoctlOp(v uint32) {
	m.arg2 = int64(v)
}

func (m *Msg) IsValid() bool {
	if m.Magic != MessageMagic || m.Datalen > MessageChunkSize || m.Hcount > MessageMaxHandles {
		return false
	}
	return true
}

func (m *Msg) IsValidIncoming(size uint32) bool {
	if (size < MessageHeaderSize) || m.Datalen != (size-MessageHeaderSize) {
		return false
	}
	return m.IsValid()
}

// DiscardHandles throws away any handles that might exist within the message
func (m *Msg) DiscardHandles() {
	for i := uint32(0); i < m.Hcount; i++ {
		fuchsia.HandleClose(m.Handle[i])
	}
}
