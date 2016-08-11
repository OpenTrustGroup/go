// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"syscall"
)

func (c *UDPConn) readFrom(b []byte) (n int, addr *UDPAddr, err error) {
	panic("TODO")
}

func (c *UDPConn) readMsg(b, oob []byte) (n, oobn, flags int, addr *UDPAddr, err error) {
	return 0, 0, 0, nil, syscall.EFUCHSIA
}

func (c *UDPConn) writeTo(b []byte, addr *UDPAddr) (int, error) {
	panic("TODO")
}

func (c *UDPConn) writeMsg(b, oob []byte, addr *UDPAddr) (n, oobn int, err error) {
	return 0, 0, syscall.EFUCHSIA
}

func dialUDP(ctx context.Context, net string, laddr, raddr *UDPAddr) (*UDPConn, error) {
	panic("TODO")
}

const udpHeaderSize = 16*3 + 2*2

func listenUDP(ctx context.Context, network string, laddr *UDPAddr) (*UDPConn, error) {
	panic("TODO")
}

func listenMulticastUDP(ctx context.Context, network string, ifi *Interface, gaddr *UDPAddr) (*UDPConn, error) {
	return nil, syscall.EFUCHSIA
}
