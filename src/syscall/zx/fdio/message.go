// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"unsafe"

	"syscall/zx"
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
	OpAccept
	OpNumOps = uint32(iota) // The total number of operations
)

// Control ordinal operations. Sent from server to client without
// necessarily having an accompanying request message.
const (
	OpOnOpen = uint32(0x80000007)
)

const (
	OpenFlagDescribe = 0x00800000
)

const (
	OpOpenFlagReadable = 1
	OpOpenFlagWritable = 2
	OpOpenFlagRDWR     = OpOpenFlagReadable | OpOpenFlagWritable
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
	RioDescribeExtra   = 24
	RioDescriptionSize = uint32(unsafe.Sizeof(RioDescription{}))
)

// FIDL2 message header
type FIDLHeader struct {
	Txid      uint32
	reserved0 uint32
	flags     uint32
	op        uint32
}

// ObjectInfo represents the union of "extra", identified
// by "Tag".
type ObjectInfo struct {
	Tag      Protocol
	reserved uint32
	extra    [RioDescribeExtra]uint8
}

type ServiceInfo struct {
}

func (o *ObjectInfo) checkProtocol(p Protocol) {
	if o.Tag != p {
		panic("Unexpected FDIO protocol")
	}
}

func (o *ObjectInfo) Service() *ServiceInfo {
	o.checkProtocol(ProtocolService)
	return (*ServiceInfo)(unsafe.Pointer(&o.extra[0]))
}

type FileInfo struct {
	Handle zx.Handle
}

func (o *ObjectInfo) File() *FileInfo {
	o.checkProtocol(ProtocolFile)
	return (*FileInfo)(unsafe.Pointer(&o.extra[0]))
}

type DirectoryInfo struct {
}

func (o *ObjectInfo) Directory() *DirectoryInfo {
	o.checkProtocol(ProtocolDirectory)
	return (*DirectoryInfo)(unsafe.Pointer(&o.extra[0]))
}

type PipeInfo struct {
	Handle zx.Handle
}

func (o *ObjectInfo) Pipe() *PipeInfo {
	o.checkProtocol(ProtocolPipe)
	return (*PipeInfo)(unsafe.Pointer(&o.extra[0]))
}

type VMOFileInfo struct {
	Handle zx.Handle
	Offset uint64
	Length uint64
}

func (o *ObjectInfo) VMOFile() *VMOFileInfo {
	o.checkProtocol(ProtocolVMOFile)
	return (*VMOFileInfo)(unsafe.Pointer(&o.extra[0]))
}

type DeviceInfo struct {
	Handle zx.Handle
}

func (o *ObjectInfo) Device() *DeviceInfo {
	o.checkProtocol(ProtocolDevice)
	return (*DeviceInfo)(unsafe.Pointer(&o.extra[0]))
}

type SocketInfo struct {
	Handle zx.Handle
}

func (o *ObjectInfo) Socket() *SocketInfo {
	o.checkProtocol(ProtocolSocket)
	return (*SocketInfo)(unsafe.Pointer(&o.extra[0]))
}

func (o *ObjectInfo) GetHandles() []zx.Handle {
	switch o.Tag {
	case ProtocolService:
		return []zx.Handle{}
	case ProtocolFile:
		h := o.File().Handle
		if h != zx.HANDLE_INVALID {
			return []zx.Handle{h}
		}
	case ProtocolDirectory:
		return []zx.Handle{}
	case ProtocolPipe:
		return []zx.Handle{o.Pipe().Handle}
	case ProtocolVMOFile:
		return []zx.Handle{o.VMOFile().Handle}
	case ProtocolDevice:
		h := o.Device().Handle
		if h != zx.HANDLE_INVALID {
			return []zx.Handle{h}
		}
	case ProtocolSocket:
		h := o.Socket().Handle
		if h != zx.HANDLE_INVALID {
			return []zx.Handle{h}
		}
	}
	return []zx.Handle{}
}

type RioDescription struct {
	FIDLHeader
	Status  zx.Status
	infoPtr *uint8
	Info    ObjectInfo
}

func (ro *RioDescription) Op() uint32 {
	return ro.op
}
func (ro *RioDescription) SetOp(v uint32) {
	ro.op = v
}

// Helper to write back a RIO object. Does not consume h.
func (ro *RioDescription) Write(h zx.Handle, flags uint32) {
	c := &zx.Channel{h}

	data := (*[RioDescriptionSize]byte)(unsafe.Pointer(ro))
	c.Write(data[:], ro.Info.GetHandles(), flags)
}

// Helper to read a RIO object. Does not consume h.
func (ro *RioDescription) Read(h zx.Handle, flags uint32) (r zx.Status) {
	c := &zx.Channel{h}
	msgBytes := (*[RioDescriptionSize]byte)(unsafe.Pointer(ro))
	var handles [1]zx.Handle
	numBytes, numHandles, err := c.Read(msgBytes[:], handles[:], flags)
	if err != nil {
		c.Close()
		return err.(zx.Error).Status
	}

	// Helper to throw away handles before they're embedded in the description
	// object.
	defer func() {
		if r != zx.ErrOk {
			for i := 0; i < int(numHandles); i++ {
				handles[i].Close()
			}
		}
	}()

	if numBytes != RioDescriptionSize {
		println("RioDescription.Read: Did not read message of expected description size")
		return zx.ErrIO
	}

	if ro.Status != zx.ErrOk {
		return ro.Status
	}

	switch ro.Info.Tag {
	case ProtocolService:
		if numHandles != 0 {
			return zx.ErrIO
		}
	case ProtocolFile:
		if numHandles == 1 {
			ro.Info.File().Handle = handles[0]
		} else {
			ro.Info.File().Handle = zx.HANDLE_INVALID
		}
	case ProtocolDirectory:
		if numHandles != 0 {
			return zx.ErrIO
		}
	case ProtocolPipe:
		if numHandles != 1 {
			return zx.ErrIO
		}
		ro.Info.Pipe().Handle = handles[0]
	case ProtocolVMOFile:
		if numHandles != 1 {
			return zx.ErrIO
		}
		ro.Info.VMOFile().Handle = handles[0]
	case ProtocolDevice:
		if numHandles == 1 {
			ro.Info.Device().Handle = handles[0]
		} else {
			ro.Info.Device().Handle = zx.HANDLE_INVALID
		}
	case ProtocolSocket:
		if numHandles == 1 {
			ro.Info.Socket().Handle = handles[0]
		} else {
			ro.Info.Socket().Handle = zx.HANDLE_INVALID
		}
	default:
		println("Unexpected Protocol")
		return zx.ErrIO
	}

	return ro.Status
}

// DiscardHandles throws away any handles that might exist within the object
func (ro *RioDescription) DiscardHandles() {
	handles := ro.Info.GetHandles()
	for i := 0; i < len(handles); i++ {
		handles[i].Close()
	}
}

type MsgHeader struct {
	FIDLHeader
	Datalen   uint32
	Arg       int32
	arg2      int64
	reserved1 int32
	Hcount    uint32
	Handle    [MessageMaxHandles + 1]zx.Handle // 3 handles + reply pipe
}

// Msg is the message sent in the Remote IO protocol
// Note: if you change this structure in any way, make sure to make the
// same change to zxsocket.Msg (though size of Data can be different).
type Msg struct {
	MsgHeader
	Data [MessageChunkSize]uint8
}

func (m *Msg) CallMsg(c *zx.Channel) (numBytes, numHandles uint32, callStatus zx.Status, err error) {
	wData := (*[MessageSizeMax]byte)(unsafe.Pointer(m))[:MessageHeaderSize+m.Datalen]
	var wHandles []zx.Handle
	if m.Hcount > 0 {
		wHandles = m.Handle[:m.Hcount]
	}
	rData := (*[MessageSizeMax]byte)(unsafe.Pointer(m))[:]
	rHandles := m.Handle[:]
	numBytes, numHandles, callStatus, err = c.Call(0, zx.TimensecInfinite, wData, wHandles, rData, rHandles)
	m.Hcount = numHandles
	return
}

// ReadMsg reads a message from a handle
func (m *Msg) ReadMsg(c *zx.Channel) (numBytes, numHandles uint32, err error) {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	numBytes, numHandles, err = c.Read(msgBytes[:], m.Handle[:], 0)
	m.Hcount = numHandles
	return
}

// WriteMsg writes a message to a handle
func (m *Msg) WriteMsg(c *zx.Channel) error {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	data := msgBytes[:MessageHeaderSize+m.Datalen]
	var handles []zx.Handle
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
