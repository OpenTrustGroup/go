// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package zxsocket

import (
	"unsafe"

	"syscall/zx"
	"syscall/zx/fdio"
)

const (
	// MessageChunkSize is the largest size of 'data' in an zxsocket message
	MessageChunkSize = 900

	// MessageSizeMax is the largest size of a zxsocket message
	MessageSizeMax = uint32(unsafe.Sizeof(Msg{}))

	// MessageHeaderSize is the size of zxsocket message without 'data'
	MessageHeaderSize = fdio.MessageHeaderSize
)

const (
	OpFlagOneHandle = uint32(0x100 + iota)
)

// Values for op. Each one will signify a request for a different operation.
// This is a copy of the list in fdio/message.go, with the non-socket-related
// options removed. Note that these are deprecated. We will be migrating to
// FIDL2 ordinals soon.
const (
	OpStatus = uint32(iota)
	OpClose
	OpClone = uint32(iota) | OpFlagOneHandle
	OpOpen  = uint32(iota) | OpFlagOneHandle
	_       = uint32(iota) // OpMisc
	OpRead
	OpWrite
	_ // OpSeek
	_ // OpStat
	_ // OpReaddir
	OpIoctl
	_                                          // OpUnlink
	_                                          // OpReadAt
	_                                          // OpWriteAt
	_                                          // OpTruncate
	_         = uint32(iota) | OpFlagOneHandle // OpRename
	OpConnect = uint32(iota)
	OpBind
	OpListen
	OpGetSockname
	OpGetPeerName
	OpGetSockOpt
	OpSetSockOpt
	OpGetAddrInfo
	_                                  // OpSetAttr
	_                                  // OpSync
	_ = uint32(iota) | OpFlagOneHandle // OpLink
	_ = uint32(iota)                   // OpMmap
	OpFcntl
	OpNumOps = uint32(iota) // The total number of operations
)

const (
	OpenFlagDescribe = 0x00800000
)

// Msg is the message sent in the zxsocket protocol.
// Its format must mirror fdio.Msg, though its Data field is a different size.
type Msg struct {
	fdio.MsgHeader
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

// ReadMsg reads a message from a socket.
func (m *Msg) ReadMsg(s zx.Socket) (numBytes int, numHandles uint32, err error) {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	// TODO: why is numBytes int?
	numBytes, err = s.Read(msgBytes[:], zx.SocketControl)
	numHandles = 0
	m.Hcount = numHandles
	return
}

// WriteMsg writes a message to a socket.
func (m *Msg) WriteMsg(s zx.Socket) (int, error) {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	data := msgBytes[:MessageHeaderSize+m.Datalen]
	return s.Write(data, zx.SocketControl)
}

// WriteMsg writes a message to a channel.
func (m *Msg) WriteChannelMsg(c *zx.Channel) error {
	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(m))
	data := msgBytes[:MessageHeaderSize+m.Datalen]
	var handles []zx.Handle
	if m.Hcount > 0 {
		handles = m.Handle[:m.Hcount]
	}
	return c.Write(data, handles, 0)
}

func (m *Msg) IsValid() bool {
	if m.Datalen > MessageChunkSize || m.Hcount != 0 {
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
