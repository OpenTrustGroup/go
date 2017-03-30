// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rio

import (
	"errors"
	"io"
	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxnet"
)

// Socket is an MXIO socket.
type Socket struct {
	ctrl   mx.Channel
	socket mx.Handle // mx.Socket for TCP, mx.Channel for UDP

	dgram bool

	// r has the same handles as the socket, so we can borrow
	// the typical RemoteIO method implementations.
	r *RemoteIO
}

// NewSocket creates a new MXIO Socket.
func NewSocket(handles []mx.Handle) (*Socket, error) {
	r, err := New(handles)
	if err != nil {
		return nil, err
	}
	s := &Socket{
		ctrl: mx.Channel{Handle: handles[0]},
		r:    r,
	}
	if len(handles) > 1 {
		s.socket = handles[1]
	}
	return s, nil
}

func (s *Socket) Read(data []byte) (n int, err error) {
	if s.dgram {
		b, _, _, _, err := s.RecvMsg(len(data))
		n = copy(data, b)
		return n, err
	}
	// c.f. mxsio_read_stream in magenta/system/ulib/mxio/remoteio.c
	for {
		n, err = mx.Socket(s.socket).Read(data, 0)
		switch mxStatus(err) {
		case mx.ErrOk:
			return n, nil
		case mx.ErrRemoteClosed:
			return 0, io.EOF
		case mx.ErrShouldWait:
			obs, err := s.socket.WaitOne(mx.SignalSocketReadable|mx.SignalSocketPeerClosed, mx.TimensecInfinite)
			switch mxStatus(err) {
			case mx.ErrOk:
				switch {
				case obs&mx.SignalSocketReadable != 0:
					continue
				case obs&mx.SignalSocketPeerClosed != 0:
					return 0, io.EOF
				}
			case mx.ErrBadHandle, mx.ErrHandleClosed:
				return 0, io.EOF
			default:
				return 0, err
			}
		default:
			return 0, err
		}
	}
}

func (s *Socket) ReadAt(data []byte, off int64) (int, error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "rio.Socket.ReadAt"}
}

func (s *Socket) Write(data []byte) (int, error) {
	if s.dgram {
		n, err := s.SendMsg(data, "", 0)
		return n, err
	}
	// c.f. mxsio_write_stream
	for {
		n, err := mx.Socket(s.socket).Write(data, 0)
		switch mxStatus(err) {
		case mx.ErrOk:
			return n, nil
		case mx.ErrShouldWait:
			_, err := s.socket.WaitOne(mx.SignalSocketWritable, mx.TimensecInfinite)
			if err != nil {
				return n, err
			}
			continue
		}
	}
}

func (s *Socket) WriteAt(data []byte, off int64) (int, error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "rio.Socket.WriteAt"}
}

func (s *Socket) RecvMsg(maxLen int) (b []byte, flags int, addr string, port uint16, err error) {
	mlen := maxLen + mxnet.SockmsgHdrLen
	msgdata := make([]byte, mlen)
	n, err := s.recvMsg(msgdata)
	if err != nil {
		return nil, 0, "", 0, err
	}
	msgdata = msgdata[:n]
	addrstr, port, flags, err := mxnet.DecodeSockmsgHdr(msgdata)
	if err != nil {
		return nil, 0, "", 0, err
	}
	return msgdata[mxnet.SockmsgHdrLen:], flags, string(addrstr), port, nil
}

func (s *Socket) recvMsg(data []byte) (n int, err error) {
	ch := mx.Channel{s.socket}
	for {
		n, _, err := ch.Read(data, nil, 0)
		switch mxStatus(err) {
		case mx.ErrOk:
			return int(n), err
		case mx.ErrShouldWait:
			obs, err := s.socket.WaitOne(mx.SignalChannelReadable|mx.SignalChannelPeerClosed, mx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if obs&mx.SignalChannelReadable != 0 {
				continue
			}
			if obs&mx.SignalChannelPeerClosed != 0 {
				return 0, mx.Error{Status: mx.ErrRemoteClosed, Text: "rio.Socket.recvMsg"}
			}
			return 0, mx.Error{Status: mx.ErrInternal, Text: "rio.Socket.recvMsg"}
		default:
			return 0, err
		}
	}
}

func (s *Socket) SendMsg(b []byte, addr string, port uint16) (n int, err error) {
	data := make([]byte, len(b)+mxnet.SockmsgHdrLen)
	err = mxnet.EncodeSockmsgHdr(data, mxnet.Addr(addr), port, 0)
	if err != nil {
		return 0, err
	}
	copy(data[mxnet.SockmsgHdrLen:], b)
	ch := mx.Channel{s.socket}
	err = ch.Write(data, nil, 0)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Socket) Seek(offset int64, whence int) (int64, error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "rio.Socket.Seek"}
}

func (s *Socket) Close() error {
	s.ctrl.Close()
	if err := s.socket.Close(); err != nil {
		return err
	}
	s.ctrl = mx.Channel{}
	s.socket = 0
	return nil
}

func (*Socket) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	return nil, mx.Error{Status: mx.ErrNotSupported, Text: "rio.Socket.Open"}
}
func (*Socket) Clone() ([]mx.Handle, error) {
	return nil, mx.Error{Status: mx.ErrNotSupported, Text: "rio.Socket.Clone"}
}
func (s *Socket) Ioctl(op uint32, in, out []byte) ([]mx.Handle, error) {
	return s.r.Ioctl(op, in, out)
}
func (s *Socket) IoctlSetHandle(op uint32, in mx.Handle) error {
	return s.r.IoctlSetHandle(op, in)
}
func (s *Socket) Misc(op uint32, off int64, in, out []byte, handles []mx.Handle) (n int, err error) {
	return s.r.Misc(op, off, in, out, handles)
}

// SetDgram marks a *Socket as a datagram (UDP) socket.
// A different protocol is used internally.
func SetDgram(m mxio.MXIO) error {
	if m == nil {
		return errors.New("rio.SetDgram: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.SetDgram: argument is not a *Socket")
	}
	s.dgram = true
	return nil
}

// Bind is BSD Sockets bind on a *Socket.
func Bind(m mxio.MXIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("rio.Bind: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	_, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpBind, 0, b, nil, nil)
	return err
}

// Connect is BSD Sockets connect on a *Socket.
func Connect(m mxio.MXIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("rio.Connect: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	_, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpConnect, 0, b, nil, nil)
	if err != nil && mxStatus(err) != mx.ErrShouldWait {
		return err
	}
	if !s.dgram {
		_, err = s.socket.WaitOne(mxnet.MXSIO_SIGNAL_CONNECTED, 0)
		return err
	}
	return nil
}

func mxStatus(err error) mx.Status {
	if err == nil {
		return mx.ErrOk
	}
	if status, ok := err.(mx.Error); ok {
		return status.Status
	}
	return mx.ErrInternal
}
