// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"syscall"
)

func (c *IPConn) readFrom(b []byte) (int, *IPAddr, error) {
	return 0, nil, syscall.EFUCHSIA
}

func (c *IPConn) readMsg(b, oob []byte) (n, oobn, flags int, addr *IPAddr, err error) {
	return 0, 0, 0, nil, syscall.EFUCHSIA
}

func (c *IPConn) writeTo(b []byte, addr *IPAddr) (int, error) {
	return 0, syscall.EFUCHSIA
}

func (c *IPConn) writeMsg(b, oob []byte, addr *IPAddr) (n, oobn int, err error) {
	return 0, 0, syscall.EFUCHSIA
}

func dialIP(ctx context.Context, netProto string, laddr, raddr *IPAddr) (*IPConn, error) {
	return nil, syscall.EFUCHSIA
}

func listenIP(ctx context.Context, netProto string, laddr *IPAddr) (*IPConn, error) {
	return nil, syscall.EFUCHSIA
}
