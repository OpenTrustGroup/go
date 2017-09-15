// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package mxnet contains types for talking to Fuchsia's fdio net interface.
package mxnet

import (
	"syscall/zx"
	"unsafe"
)

const MX_SOCKET_HALF_CLOSE = 1
const MX_SOCKET_READABLE = zx.SignalObject0
const MX_SOCKET_WRITABLE = zx.SignalObject1
const MX_SOCKET_PEER_CLOSED = zx.SignalObject2
const MXSIO_SIGNAL_INCOMING = zx.SignalUser0
const MXSIO_SIGNAL_CONNECTED = zx.SignalUser3
const MXSIO_SIGNAL_HALFCLOSED = zx.SignalUser4

// Addr is an IP address.
// It has either len == 4 (if it is IPv4) or len == 16 (IPv6).
type Addr string

// TODO(crawshaw): consider cleaning up these interfaces by migrating to
// an address type that captures everything:
//
//	type Address struct {
//		Addr [16]byte
//		IPv4 bool
//		Port uint16
//	}

const SockaddrLen = c_sockaddr_in6_len
const SockmsgHdrLen = c_fdio_socket_msg_hdr_len
const SockaddrReplyLen = c_mxrio_sockaddr_reply_len

func EncodeSockmsgHdr(dst []byte, addr Addr, port uint16, flags int) (err error) {
	if len(dst) < SockmsgHdrLen {
		return errString("mxnet.EncodeSockmsgHdr: dst too short")
	}
	var n int
	if addr != "" || port != 0 {
		n, err = EncodeSockaddr(dst, addr, port)
		if err != nil {
			return err
		}
	}
	hdr := (*c_fdio_socket_msg_hdr)(unsafe.Pointer(&dst[0]))
	hdr.addrlen = c_socklen(n)
	hdr.flags = int32(flags)
	return nil
}

func DecodeSockmsgHdr(src []byte) (addr Addr, port uint16, flags int, err error) {
	if len(src) < SockmsgHdrLen {
		return "", 0, 0, errString("mxnet.DecodeSockmsgHdr: src too short")
	}
	addr, port, err = DecodeSockaddr(src)
	if err != nil {
		return "", 0, 0, err
	}
	hdr := (*c_fdio_socket_msg_hdr)(unsafe.Pointer(&src[0]))
	flags = int(hdr.flags)
	return addr, port, flags, nil
}

func DecodeSockaddr(data []byte) (addr Addr, port uint16, err error) {
	if len(data) < 2 {
		return "", 0, errString("mxnet:DecodeSockaddr: data too short")
	}
	family := uint16(data[0]) | uint16(data[1])<<8
	switch family {
	case AF_INET:
		if len(data) < int(unsafe.Sizeof(c_sockaddr_in{})) {
			return "", 0, errString("mxnet:DecodeSockaddr: data too short for ipv4")
		}
		v := (*c_sockaddr_in)(unsafe.Pointer(&data[0]))
		port = uint16(data[3]) | uint16(data[2])<<8
		if !isZeros(v.sin_addr[:]) {
			addr = Addr(v.sin_addr[:])
		}
		return addr, port, nil
	case AF_INET6:
		if len(data) < int(unsafe.Sizeof(c_sockaddr_in6{})) {
			return "", 0, errString("mxnet:DecodeSockaddr: data too short for ipv6")
		}
		v := (*c_sockaddr_in6)(unsafe.Pointer(&data[0]))
		port = uint16(data[3]) | uint16(data[2])<<8
		if !isZeros(v.sin6_addr[:]) {
			addr = Addr(v.sin6_addr[:])
		}
		return addr, port, nil
	default:
		return "", 0, errString("mxnet:DecodeSockaddr: unknown family")
	}
}

func EncodeSockaddr(dst []byte, addr Addr, port uint16) (int, error) {
	switch len(addr) {
	case 0, 4:
		if len(dst) < c_sockaddr_in_len {
			return 0, errString("mxnet: dst too small for c_sockaddr_in")
		}
		sockaddr := c_sockaddr_in{sin_family: AF_INET}
		sockaddr.sin_port.setPort(port)
		copy(sockaddr.sin_addr[:], addr)
		srcb := (*[unsafe.Sizeof(sockaddr)]byte)(unsafe.Pointer(&sockaddr))[:]
		return copy(dst, srcb), nil
	case 16:
		if len(dst) < c_sockaddr_in_len {
			return 0, errString("mxnet: dst too small for c_sockaddr_in6")
		}
		sockaddr := c_sockaddr_in6{sin6_family: AF_INET6}
		sockaddr.sin6_port.setPort(port)
		copy(sockaddr.sin6_addr[:], addr)
		srcb := (*[unsafe.Sizeof(sockaddr)]byte)(unsafe.Pointer(&sockaddr))[:]
		return copy(dst, srcb), nil
	default:
		return 0, errString("mxnet: bad address length")
	}
}

type errString string

func (err errString) Error() string { return string(err) }

const (
	c_sockaddr_in_len          = int(unsafe.Sizeof(c_sockaddr_in{}))
	c_sockaddr_in6_len         = int(unsafe.Sizeof(c_sockaddr_in6{}))
	c_fdio_socket_msg_hdr_len  = int(unsafe.Sizeof(c_fdio_socket_msg_hdr{}))
	c_mxrio_sockaddr_reply_len = int(unsafe.Sizeof(c_mxrio_sockaddr_reply{}))
)

func (v c_in_port) port() uint16 {
	return uint16(v[0])<<8 | uint16(v[1])
}

func (v *c_in_port) setPort(port uint16) {
	v[0] = uint8(port >> 8)
	v[1] = uint8(port)
}

func isZeros(buf []byte) bool {
	for i := 0; i < len(buf); i++ {
		if buf[i] != 0 {
			return false
		}
	}
	return true
}
