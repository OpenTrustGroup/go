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

func (m *Msg) Op() uint32 {
	return m.AsFDIOMsg().Op()
}
func (m *Msg) SetOp(v uint32) {
	m.AsFDIOMsg().SetOp(v)
}
func (m *Msg) OpHandleCount() uint32 {
	return m.AsFDIOMsg().OpHandleCount()
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

// DiscardHandles throws away any handles that might exist within the message
func (m *Msg) DiscardHandles() {
	for i := uint32(0); i < m.Hcount; i++ {
		m.Handle[i].Close()
	}
	m.Hcount = 0
}

// AsFDIOMsg returns this message as an fdio.Msg. This is a temporary hack
// to make the transition away from fdio easier.
func (m *Msg) AsFDIOMsg() (*fdio.Msg) {
	return (*fdio.Msg)(unsafe.Pointer(m))
}