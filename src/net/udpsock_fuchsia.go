// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"syscall"
	"syscall/mx/mxnet"
)

func (a *UDPAddr) encode() ([]byte, error) {
	b := make([]byte, mxnet.SockaddrLen)
	_, err := mxnet.EncodeSockaddr(b, mxnet.Addr(a.IP), uint16(a.Port))
	return b, err
}

func (a *UDPAddr) family() int {
	if a == nil || len(a.IP) <= IPv4len {
		return syscall.AF_INET
	}
	if a.IP.To4() != nil {
		return syscall.AF_INET
	}
	return syscall.AF_INET6
}

func (a *UDPAddr) sockaddr(family int) (addr mxnet.Addr, port uint16, err error) {
	if a == nil {
		return "", 0, nil
	}
	return ipToSockaddr(family, a.IP, a.Port, a.Zone)
}

func (c *UDPConn) readFrom(b []byte) (n int, addr *UDPAddr, err error) {
	panic("TODO readFrom")
}

func (c *UDPConn) readMsg(b, oob []byte) (n, oobn, flags int, addr *UDPAddr, err error) {
	n, flags, addrstr, port, err := c.fd.recvMsg(b)
	if err != nil {
		return 0, 0, 0, nil, err
	}
	addr = &UDPAddr{
		IP:   IP(addrstr),
		Port: int(port),
	}
	return n, 0, flags, addr, nil
}

func (c *UDPConn) writeTo(b []byte, addr *UDPAddr) (int, error) {
	panic("TODO writeTo")
}

func (c *UDPConn) writeMsg(b, oob []byte, addr *UDPAddr) (n, oobn int, err error) {
	if len(oob) > 0 {
		return 0, 0, syscall.EFUCHSIA
	}
	n, err = c.fd.sendMsg(b, string(addr.IP), uint16(addr.Port))
	return n, 0, err
}

func dialUDP(ctx context.Context, net string, laddr, raddr *UDPAddr) (*UDPConn, error) {
	switch net {
	case "udp", "udp4", "udp6":
	default:
		return nil, UnknownNetworkError(net)
	}
	if raddr == nil {
		return nil, errMissingAddress
	}
	var laddrif sockaddr
	if laddr != nil {
		laddrif = laddr
	}
	fd, err := dialFuchsia(ctx, net, laddrif, raddr)
	if err != nil {
		return nil, err
	}
	return newUDPConn(fd), nil
}

const udpHeaderSize = 16*3 + 2*2

func listenUDP(ctx context.Context, network string, laddr *UDPAddr) (*UDPConn, error) {
	panic("TODO listenUDP")
}

func listenMulticastUDP(ctx context.Context, network string, ifi *Interface, gaddr *UDPAddr) (*UDPConn, error) {
	return nil, syscall.EFUCHSIA
}
