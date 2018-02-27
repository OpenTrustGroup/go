// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"errors"
	"io"
	"syscall/zx"
	"syscall/zx/mxnet"
)

// Socket is an FDIO socket.
type Socket struct {
	ctrl   zx.Channel
	socket zx.Socket

	dgram bool

	// r has the same handles as the socket, so we can borrow
	// the typical RemoteIO method implementations.
	r *RemoteIO
}

// NewSocket creates a new FDIO Socket.
func NewSocket(handles []zx.Handle) (*Socket, error) {
	r, err := NewRIO(handles)
	if err != nil {
		return nil, err
	}
	s := &Socket{
		ctrl: zx.Channel{Handle: handles[0]},
		r:    r,
	}
	if len(handles) > 1 {
		s.socket = zx.Socket(handles[1])
	}
	return s, nil
}

func (s *Socket) Read(data []byte) (n int, err error) {
	if s.dgram {
		b, _, _, _, err := s.RecvMsg(len(data))
		n = copy(data, b)
		return n, err
	}
	// c.f. mxsio_read_stream in zircon/system/ulib/fdio/remoteio.c
	for {
		n, err = zx.Socket(s.socket).Read(data, 0)
		switch mxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrPeerClosed:
			return 0, io.EOF
		case zx.ErrShouldWait:
			obs, err := s.socket.WaitOne(zx.SignalSocketReadable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			switch mxStatus(err) {
			case zx.ErrOk:
				switch {
				case obs&zx.SignalSocketReadable != 0:
					continue
				case obs&zx.SignalSocketPeerClosed != 0:
					return 0, io.EOF
				}
			case zx.ErrBadHandle, zx.ErrCanceled:
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
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "rio.Socket.ReadAt"}
}

func (s *Socket) Write(data []byte) (int, error) {
	if s.dgram {
		n, err := s.SendMsg(data, "", 0)
		return n, err
	}
	// c.f. mxsio_write_stream
	for {
		n, err := zx.Socket(s.socket).Write(data, 0)
		switch mxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrPeerClosed:
			return 0, io.EOF
		case zx.ErrShouldWait:
			obs, err := s.socket.WaitOne(zx.SignalSocketWritable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			switch mxStatus(err) {
			case zx.ErrOk:
				switch {
				case obs&zx.SignalSocketPeerClosed != 0:
					return 0, io.EOF
				case obs&zx.SignalSocketWritable != 0:
					continue
				}
			case zx.ErrBadHandle, zx.ErrCanceled:
				return 0, io.EOF
			default:
				return 0, err
			}
		default:
			return 0, err
		}
	}
}

func (s *Socket) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "rio.Socket.WriteAt"}
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
	for {
		n, err := s.socket.Read(data, 0)
		switch mxStatus(err) {
		case zx.ErrOk:
			return int(n), err
		case zx.ErrShouldWait:
			obs, err := s.socket.WaitOne(zx.SignalChannelReadable|zx.SignalChannelPeerClosed, zx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if obs&zx.SignalChannelReadable != 0 {
				continue
			}
			if obs&zx.SignalChannelPeerClosed != 0 {
				return 0, zx.Error{Status: zx.ErrPeerClosed, Text: "rio.Socket.recvMsg"}
			}
			return 0, zx.Error{Status: zx.ErrInternal, Text: "rio.Socket.recvMsg"}
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
	_, err = s.socket.Write(data, 0)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Socket) Seek(offset int64, whence int) (int64, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "rio.Socket.Seek"}
}

func (s *Socket) Close() error {
	s.ctrl.Close()
	if err := s.socket.Close(); err != nil {
		return err
	}
	s.ctrl = zx.Channel{}
	s.socket = 0
	return nil
}

func (s *Socket) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	return s.r.Open(path, flags, mode)
}
func (*Socket) Clone() ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "rio.Socket.Clone"}
}
func (s *Socket) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	return s.r.Ioctl(op, in, out)
}
func (s *Socket) IoctlSetHandle(op uint32, in zx.Handle) error {
	return s.r.IoctlSetHandle(op, in)
}
func (s *Socket) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	return s.r.Misc(op, off, in, out, handles)
}

// SetDgram marks a *Socket as a datagram (UDP) socket.
// A different protocol is used internally.
func SetDgram(m FDIO) error {
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
func Bind(m FDIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("rio.Bind: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	addrlen, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}
	b = b[:addrlen]

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpBind, 0, b, nil, nil)
	return err
}

// Connect is BSD Sockets connect on a *Socket.
func Connect(m FDIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("rio.Connect: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	addrlen, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}
	b = b[:addrlen]

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpConnect, 0, b, nil, nil)
	if err != nil && mxStatus(err) == zx.ErrShouldWait {
		obs, err := s.socket.WaitOne(mxnet.MXSIO_SIGNAL_OUTGOING, zx.TimensecInfinite)
		switch mxStatus(err) {
		case zx.ErrOk:
			switch {
			case obs&mxnet.MXSIO_SIGNAL_CONNECTED != 0:
				return nil
			default:
				// TODO: Use opSetSockOpt to get the reason of the error.
				return errors.New("rio.Connect: OpConnect was not successful")
			}
		default:
			return err
		}
	}
	return err
}

func ListenStream(m FDIO, backlog int) error {
	if m == nil {
		return errors.New("rio.ListenStream: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return errors.New("rio.ListenStream: argument is not a *Socket")
	}
	b := make([]byte, 4)
	b[0] = byte(backlog)
	b[1] = byte(backlog >> 8)
	b[2] = byte(backlog >> 16)
	b[3] = byte(backlog >> 24)
	_, err := s.Misc(OpListen, 0, b, nil, nil)
	return err
}

func Accept(m FDIO) (newm FDIO, err error) {
	if m == nil {
		return nil, errors.New("rio.Accept: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return nil, errors.New("rio.Accept: argument is not a *Socket")
	}

	for {
		newm, err = m.Open("accept", 0, 0)
		if err == nil {
			break
		}
		if mxStatus(err) != zx.ErrShouldWait {
			return nil, err
		}
		// TODO: non-blocking support, pass this to the poller
		const MXSIO_SIGNAL_INCOMING = zx.SignalUser0
		_, err = s.socket.WaitOne(mxnet.MXSIO_SIGNAL_INCOMING, zx.TimensecInfinite)
		if err != nil {
			return nil, err
		}
	}

	return newm, nil
}

func getName(m FDIO, op uint32, debug string) (addr mxnet.Addr, port uint16, err error) {
	if m == nil {
		return "", 0, errors.New("rio." + debug + ": nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return "", 0, errors.New("rio." + debug + ": argument is not a *Socket")
	}
	out := make([]byte, mxnet.SockaddrReplyLen)
	_, err = s.Misc(OpGetSockname, 0, nil, out, nil)
	if err != nil {
		return "", 0, err
	}
	addr, port, err = mxnet.DecodeSockaddr(out)
	if err != nil {
		return "", 0, err
	}

	return addr, port, nil
}

func GetSockName(m FDIO) (addr mxnet.Addr, port uint16, err error) {
	return getName(m, OpGetSockname, "GetSockName")
}

func GetPeerName(m FDIO) (addr mxnet.Addr, port uint16, err error) {
	return getName(m, OpGetPeerName, "GetPeerName")
}

func mxStatus(err error) zx.Status {
	if err == nil {
		return zx.ErrOk
	}
	if status, ok := err.(zx.Error); ok {
		return status.Status
	}
	return zx.ErrInternal
}
