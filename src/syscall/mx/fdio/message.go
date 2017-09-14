// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"unsafe"

	"syscall/mx"
)

const (
	// MessageChunkSize is the largest size of 'data' in an RIO message
	MessageChunkSize = 8192

	// MessageSizeMax is the largest size of an FDIO message
	MessageSizeMax = uint32(unsafe.Sizeof(Msg{}))

	// MessageHeaderSize is the size of a RIO message without 'data'
	MessageHeaderSize = uint32(unsafe.Sizeof(MsgHeader{}))

	// MessageMaxHandles is the maximum number of handles used by open/clone/create
	MessageMaxHandles = 3
)

const (
	OpFlagOneHandle = uint32(0x100 + iota)
	OpFlagTwoHandles
	OpFlagThreeHandles
)

// Values for op. Each one will signify a request for a different operation.
const (
	OpStatus = uint32(iota)
	OpClose
	OpClone = uint32(iota) | OpFlagOneHandle
	OpOpen  = uint32(iota) | OpFlagOneHandle
	OpMisc  = uint32(iota)
	OpRead
	OpWrite
	OpSeek
	OpStat
	OpReaddir
	OpIoctl
	OpUnlink
	OpReadAt
	OpWriteAt
	OpTruncate
	OpRename  = uint32(iota) | OpFlagOneHandle
	OpConnect = uint32(iota)
	OpBind
	OpListen
	OpGetSockname
	OpGetPeerName
	OpGetSockOpt
	OpSetSockOpt
	OpGetAddrInfo
	OpSetAttr
	OpSync
	OpLink = uint32(iota) | OpFlagOneHandle
	OpMmap = uint32(iota)
	OpFcntl
	OpNumOps = uint32(iota) // The total number of operations
)

// Top 4 bits of open flags are reserved for the protocol
const (
	OpOpenFlagProtocolMask = uint32(0xf0000000)
	OpOpenFlagPipeline     = uint32(0x10000000)
)

const (
	OpOpenFlagRDOnly = uint32(iota)
	OpOpenFlagWROnly
	OpOpenFlagRDWR
)

// Ops which correspond to a previously listed operation, but have a different number of handles
const (
	OpIoctlOneHandle = OpIoctl | OpFlagOneHandle
)

const (
	OpFcntlCmdGetFL = uint32(iota + 3)
	OpFcntlCmdSetFL
)

const (
	RioObjectExtra   = 32
	RioObjectMinSize = uint32(unsafe.Sizeof(RioObjectHeader{}))
	RioObjectMaxSize = RioObjectMinSize + RioObjectExtra
)

type RioObjectHeader struct {
	Status mx.Status
	Type   uint32
}

type RioObject struct {
	// Required Header
	RioObjectHeader

	// Optional Extra Data
	Extra [RioObjectExtra]uint8

	// OOB Data
	Esize  uint32
	Hcount uint32
	Handle [MessageMaxHandles]mx.Handle
}

// Helper to write back a RIO object. Does not consume h.
func (ro *RioObject) Write(h mx.Handle, flags uint32) {
	c := &mx.Channel{h}

	// Prepare the raw data
	size := RioObjectMinSize + ro.Esize
	if size > RioObjectMaxSize {
		panic("Invalid RIO Object (too large): RIO Implementation error")
	}
	msgBytes := (*[RioObjectMaxSize]byte)(unsafe.Pointer(ro))
	data := msgBytes[:size]

	// Prepare the handles
	var handles []mx.Handle
	if ro.Hcount > 0 {
		handles = ro.Handle[:ro.Hcount]
	}
	c.Write(data, handles, flags)
}

// Helper to read a RIO object. Consumes h.
func (ro *RioObject) Read(h mx.Handle, flags uint32) mx.Status {
	c := &mx.Channel{h}
	msgBytes := (*[RioObjectMaxSize]byte)(unsafe.Pointer(ro))
	numBytes, numHandles, err := c.Read(msgBytes[:], ro.Handle[1:], flags)
	if err != nil {
		c.Close()
		return err.(mx.Error).Status
	}
	ro.Handle[0] = c.Handle
	numHandles++
	if numBytes < RioObjectMinSize {
		ro.DiscardHandles()
		return mx.ErrIO
	}
	ro.Hcount = numHandles
	ro.Esize = numBytes - RioObjectMinSize
	r := ro.Status
	if r < 0 {
		ro.DiscardHandles()
	}
	return r
}

// DiscardHandles throws away any handles that might exist within the object
func (ro *RioObject) DiscardHandles() {
	for i := uint32(0); i < ro.Hcount; i++ {
		ro.Handle[i].Close()
	}
	ro.Hcount = 0
}

// Helper wrapping the common case of 'write back only a status'
func IndirectError(h mx.Handle, status mx.Status) mx.Status {
	ro := &RioObject{
		RioObjectHeader: RioObjectHeader{
			Status: status,
		},
		Esize:  0,
		Hcount: 0,
	}
	ro.Write(h, 0)
	h.Close()
	return ErrIndirect.Status
}

type MsgHeader struct {
	Txid     uint32
	op       uint32
	Datalen  uint32
	Arg      int32
	arg2     int64
	Reserved int32
	Hcount   uint32
	Handle   [MessageMaxHandles + 1]mx.Handle // 3 handles + reply pipe
}

// Msg is the message sent in the Remote IO protocol
type Msg struct {
	MsgHeader
	Data [MessageChunkSize]uint8
}

func (m *Msg) CallMsg(c *mx.Channel) (numBytes, numHandles uint32, callStatus mx.Status, err error) {
	wData := (*[MessageSizeMax]byte)(unsafe.Pointer(m))[:MessageHeaderSize+m.Datalen]
	var wHandles []mx.Handle
	if m.Hcount > 0 {
		wHandles = m.Handle[:m.Hcount]
	}
	rData := (*[MessageSizeMax]byte)(unsafe.Pointer(m))[:]
	rHandles := m.Handle[:]
	numBytes, numHandles, callStatus, err = c.Call(0, mx.TimensecInfinite, wData, wHandles, rData, rHandles)
	m.Hcount = numHandles
	return
}

// ReadMsg reads a message from a handle
func (m *Msg) ReadMsg(c *mx.Channel) (numBytes, numHandles uint32, err error) {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	numBytes, numHandles, err = c.Read(msgBytes[:], m.Handle[:], 0)
	m.Hcount = numHandles
	return
}

// WriteMsg writes a message to a handle
func (m *Msg) WriteMsg(c *mx.Channel) error {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	data := msgBytes[:MessageHeaderSize+m.Datalen]
	var handles []mx.Handle
	if m.Hcount > 0 {
		handles = m.Handle[:m.Hcount]
	}
	return c.Write(data, handles, 0)
}

func (m *Msg) Op() uint32 {
	return m.op & 0x3FFF
}
func (m *Msg) SetOp(v uint32) {
	m.op = (v & 0x3FFF)
}
func (m *Msg) OpHandleCount() uint32 {
	return (m.op >> 8) & 3
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
func (m *Msg) Protocol() Protocol {
	return Protocol(m.arg2)
}
func (m *Msg) SetProtocol(p Protocol) {
	m.arg2 = int64(p)
}
func (m *Msg) IoctlOp() uint32 {
	return uint32(m.arg2)
}
func (m *Msg) SetIoctlOp(v uint32) {
	m.arg2 = int64(v)
}

func (m *Msg) FcntlFlags() uint32 {
	return uint32(m.arg2)
}

func (m *Msg) SetFcntlFlags(v uint32) {
	m.arg2 = int64(v)
}

func (m *Msg) IsValid() bool {
	if m.Datalen > MessageChunkSize || m.Hcount > MessageMaxHandles {
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
		m.Handle[i].Close()
	}
	m.Hcount = 0
}
