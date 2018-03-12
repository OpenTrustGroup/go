// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package zxsocket

import (
	"syscall/zx"
	"syscall/zx/fdio"
	"unsafe"
)

// txn completes the client-side part of the RIO transaction.
func (s *Socket) txn(msg *Msg) (uint32, error) {
	if !msg.IsValid() {
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "zxsocket.txn"}
	}

	msgBytes := (*[MessageSizeMax]byte)(unsafe.Pointer(msg))
	numBytes, err := WriteControl(s.socket, msgBytes[:MessageHeaderSize+msg.Datalen])
	if err != nil {
		return 0, err
	}
	numBytes, err = ReadControl(s.socket, msgBytes[:])
	if err != nil {
		return 0, err
	}

	if !msg.IsValidIncoming(uint32(numBytes)) || msg.Op() != OpStatus {
		return 0, zx.Error{Status: zx.ErrIO, Text: "zxsocket.txn"}
	}

	// Check for remote error
	status := zx.Status(msg.Arg)
	if status < 0 {
		return 0, zx.Error{Status: status, Text: "zxsocket.txn"}
	}
	return uint32(status), nil
}

func (s *Socket) Close() error {
	var msg Msg
	msg.SetOp(OpClose)

	ret, err := s.txn(&msg)
	if ret >= 0 {
		msg.DiscardHandles()
	}

	s.socket.Close()
	s.socket = 0

	if err != nil {
		return err
	}
	return nil
}

func closeAll(handles []zx.Handle) {
	for i := range handles {
		handles[i].Close()
	}
}

// Consumes handles, wrapping them in fdio object or closing them on error.
func createFromHandles(handle zx.Handle, info fdio.ObjectInfo) (fdio.FDIO, error) {
	handles := append([]zx.Handle{handle}, info.GetHandles()...)
	switch info.Tag {
	case fdio.ProtocolSocket:
		handles[0].Close()
		return NewSocket(handles[1])
	default:
		println("Unknown fdio protocol, ", info.Tag)
		// In the error case, the server may have given us handles -- close them
		closeAll(handles)
		return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket"}
	}
}

func OpenHandle(rio zx.Channel, path string, flags uint32, mode uint32) (fdio.FDIO, error) {
	c, ro, err := fdio.GetObject(rio, OpOpen, path, flags, mode)
	if err != nil {
		return nil, err
	}
	return createFromHandles(zx.Handle(c), ro.Info)
}

// TODO(smklein): Remove; this function is a subset of "Ioctl".
func (s *Socket) IoctlSetHandle(op uint32, in zx.Handle) error {
	return zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.IoctlSetHandle"}
}

func (s *Socket) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	var msg Msg
	if len(in) > fdio.IoctlMaxInput {
		return nil, zx.Error{Status: zx.ErrInvalidArgs, Text: "zxsocket.Ioctl"}
	}

	if fdio.IoctlKind(op) != fdio.IoctlKindDefault {
		return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Ioctl"}
	}

	msg.SetOp(OpIoctl)
	msg.Datalen = uint32(len(in))
	msg.Arg = int32(len(out))
	msg.SetIoctlOp(op)
	copy(msg.Data[:], in)

	_, err := s.txn(&msg)
	if err != nil {
		return nil, err
	}

	copy(out, msg.Data[:msg.Datalen])
	return nil, nil
}

func (s *Socket) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	if len(in) > MessageChunkSize || len(out) > MessageChunkSize {
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "zxsocket.Misc"}
	}

	var msg Msg
	msg.SetOp(op)
	msg.Arg = int32(len(out))
	msg.Datalen = uint32(len(in))
	msg.SetOff(off)
	copy(msg.Data[:], in)

	if len(handles) > 0 {
		copy(msg.Handle[:], handles)
		msg.Hcount = uint32(len(handles))
	}
	if _, err = s.txn(&msg); err != nil {
		return 0, err
	}
	msg.DiscardHandles()

	if int(msg.Datalen) > len(out) {
		return 0, zx.Error{Status: zx.ErrIO, Text: "zxsocket.Misc"}
	}
	n = copy(out, msg.Data[:msg.Datalen])

	if op == OpListen {
		s.flags |= SocketFlagsDidListen
	}

	return n, nil
}
