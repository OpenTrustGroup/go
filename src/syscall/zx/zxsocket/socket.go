// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package zxsocket

import (
	"errors"
	"io"
	"syscall/zx"
	"syscall/zx/fdio"
	"syscall/zx/mxnet"
)

// Socket is an FDIO socket.
type Socket struct {
	socket zx.Socket

	dgram bool
	flags uint32
}

// Socket flags
const (
	SocketFlagsDidListen = 1 << iota // zxsocket._DID_LISTEN
)

// NewSocket creates a new zxsocket Socket.
func NewSocket(handle zx.Handle) (*Socket, error) {
	s := &Socket{
		socket: zx.Socket(handle),
	}
	return s, nil
}

func ReadControl(s zx.Socket, data []byte) (n int, err error) {
	// c.f. zxsocket._read_control in zircon/system/ulib/fdio/remoteio.c
	for {
		n, err = s.Read(data, zx.SocketControl)
		switch zxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrPeerClosed:
			return 0, io.EOF
		case zx.ErrShouldWait:
			obs, err := s.WaitOne(zx.SignalSocketControlReadable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			switch zxStatus(err) {
			case zx.ErrOk:
				switch {
				case obs&zx.SignalSocketControlReadable != 0:
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

func (s *Socket) Read(data []byte) (n int, err error) {
	if s.dgram {
		b, _, _, _, err := s.RecvMsg(len(data))
		n = copy(data, b)
		return n, err
	}
	// c.f. mxsio_read_stream in zircon/system/ulib/fdio/remoteio.c
	for {
		n, err = s.socket.Read(data, 0)
		switch zxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrPeerClosed:
			return 0, io.EOF
		case zx.ErrShouldWait:
			obs, err := s.socket.WaitOne(zx.SignalSocketReadable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			switch zxStatus(err) {
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
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.ReadAt"}
}

func WriteControl(s zx.Socket, data []byte) (int, error) {
	// c.f. zxsocket._write_control
	for {
		n, err := s.Write(data, zx.SocketControl)
		switch zxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrShouldWait:
			_, err := s.WaitOne(zx.SignalSocketControlWriteable, zx.TimensecInfinite)
			if err != nil {
				return n, err
			}
			continue
		}
	}
}

func (s *Socket) Write(data []byte) (int, error) {
	if s.dgram {
		n, err := s.SendMsg(data, "", 0)
		return n, err
	}
	// c.f. mxsio_write_stream
	for {
		n, err := s.socket.Write(data, 0)
		switch zxStatus(err) {
		case zx.ErrOk:
			return n, nil
		case zx.ErrShouldWait:
			_, err := s.socket.WaitOne(zx.SignalSocketWritable, zx.TimensecInfinite)
			if err != nil {
				return n, err
			}
			continue
		}
	}
}

func (s *Socket) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.WriteAt"}
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
		switch zxStatus(err) {
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
				return 0, zx.Error{Status: zx.ErrPeerClosed, Text: "zxsocket.Socket.recvMsg"}
			}
			return 0, zx.Error{Status: zx.ErrInternal, Text: "zxsocket.Socket.recvMsg"}
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
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.Seek"}
}

func (s *Socket) Open(path string, flags uint32, mode uint32) (fdio.FDIO, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.Open"}
}

func (*Socket) Clone() ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zxsocket.Socket.Clone"}
}

// SetDgram marks a *Socket as a datagram (UDP) socket.
// A different protocol is used internally.
func SetDgram(m fdio.FDIO) error {
	if m == nil {
		return errors.New("zxsocket.SetDgram: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return errors.New("zxsocket.SetDgram: argument is not a *Socket")
	}
	s.dgram = true
	return nil
}

// Bind is BSD Sockets bind on a *Socket.
func Bind(m fdio.FDIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("zxsocket.Bind: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	addrlen, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}
	b = b[:addrlen]

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("zxsocket.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpBind, 0, b, nil, nil)
	return err
}

// Connect is BSD Sockets connect on a *Socket.
func Connect(m fdio.FDIO, addr mxnet.Addr, port uint16) error {
	if m == nil {
		return errors.New("zxsocket.Connect: nil argument")
	}
	b := make([]byte, mxnet.SockaddrLen)
	addrlen, err := mxnet.EncodeSockaddr(b, addr, port)
	if err != nil {
		return err
	}
	b = b[:addrlen]

	s, ok := m.(*Socket)
	if !ok {
		return errors.New("zxsocket.Connect: argument is not a *Socket")
	}
	_, err = s.Misc(OpConnect, 0, b, nil, nil)
	if err != nil && zxStatus(err) == zx.ErrShouldWait {
		obs, err := s.socket.WaitOne(mxnet.MXSIO_SIGNAL_OUTGOING, zx.TimensecInfinite)
		switch zxStatus(err) {
		case zx.ErrOk:
			switch {
			case obs&mxnet.MXSIO_SIGNAL_CONNECTED != 0:
				return nil
			default:
				// TODO: Use opSetSockOpt to get the reason of the error.
				return errors.New("zxsocket.Connect: OpConnect was not successful")
			}
		default:
			return err
		}
	}
	return err
}

func ListenStream(m fdio.FDIO, backlog int) error {
	if m == nil {
		return errors.New("zxsocket.ListenStream: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return errors.New("zxsocket.ListenStream: argument is not a *Socket")
	}
	b := make([]byte, 4)
	b[0] = byte(backlog)
	b[1] = byte(backlog >> 8)
	b[2] = byte(backlog >> 16)
	b[3] = byte(backlog >> 24)
	_, err := s.Misc(OpListen, 0, b, nil, nil)
	return err
}

func Accept(m fdio.FDIO) (newm fdio.FDIO, err error) {
	if m == nil {
		return nil, errors.New("zxsocket.Accept: nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return nil, errors.New("zxsocket.Accept: argument is not a *Socket")
	}
	if s.flags&SocketFlagsDidListen == 0 {
		return nil, errors.New("zxsocket.Accept: listen was not called")
	}

	var h zx.Handle
	for {
		err = s.socket.Accept(&h)
		if err == nil {
			break
		}
		if zxStatus(err) != zx.ErrShouldWait {
			return nil, err
		}
		s.socket.Handle().SignalPeer(0, zx.SignalUser7)
		// TODO: non-blocking support, pass this to the poller
		const ZX_SOCKET_ACCEPT = zx.SignalSocketAccept
		const ZX_SOCKET_PEER_CLOSED = zx.SignalSocketPeerClosed
		pending, err := s.socket.WaitOne(ZX_SOCKET_ACCEPT|ZX_SOCKET_PEER_CLOSED, zx.TimensecInfinite)
		if err != nil {
			return nil, err
		}
		if pending&ZX_SOCKET_ACCEPT != 0 {
			continue
		}
		if pending&ZX_SOCKET_PEER_CLOSED != 0 {
			return nil, zx.Error{Status: zx.ErrPeerClosed, Text: "zxsocket"}
		}
	}

	return NewSocket(h)
}

func getName(m fdio.FDIO, op uint32, debug string) (addr mxnet.Addr, port uint16, err error) {
	if m == nil {
		return "", 0, errors.New("zxsocket." + debug + ": nil argument")
	}
	s, ok := m.(*Socket)
	if !ok {
		return "", 0, errors.New("zxsocket." + debug + ": argument is not a *Socket")
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

func GetSockName(m fdio.FDIO) (addr mxnet.Addr, port uint16, err error) {
	return getName(m, OpGetSockname, "GetSockName")
}

func GetPeerName(m fdio.FDIO) (addr mxnet.Addr, port uint16, err error) {
	return getName(m, OpGetPeerName, "GetPeerName")
}

func zxStatus(err error) zx.Status {
	if err == nil {
		return zx.ErrOk
	}
	if status, ok := err.(zx.Error); ok {
		return status.Status
	}
	return zx.ErrInternal
}
