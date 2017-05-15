// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"io"
	"os"
	"syscall"
	"syscall/mx/mxnet"
)

func (a *TCPAddr) encode() ([]byte, error) { // TODO remove
	b := make([]byte, mxnet.SockaddrLen)
	_, err := mxnet.EncodeSockaddr(b, mxnet.Addr(a.IP), uint16(a.Port))
	return b, err
}

func (a *TCPAddr) family() int {
	if a == nil || len(a.IP) <= IPv4len {
		return syscall.AF_INET
	}
	if a.IP.To4() != nil {
		return syscall.AF_INET
	}
	return syscall.AF_INET6
}

func (a *TCPAddr) sockaddr(family int) (addr mxnet.Addr, port uint16, err error) {
	if a == nil {
		return "", 0, nil
	}
	return ipToSockaddr(family, a.IP, a.Port, a.Zone)
}

func (c *TCPConn) readFrom(r io.Reader) (int64, error) {
	return genericReadFrom(c, r)
}

func dialTCP(ctx context.Context, net string, laddr, raddr *TCPAddr) (*TCPConn, error) {
	if testHookDialTCP != nil {
		return testHookDialTCP(ctx, net, laddr, raddr)
	}
	return doDialTCP(ctx, net, laddr, raddr)
}

func doDialTCP(ctx context.Context, net string, laddr, raddr *TCPAddr) (*TCPConn, error) {
	switch net {
	case "tcp", "tcp4", "tcp6":
	default:
		return nil, UnknownNetworkError(net)
	}
	if raddr == nil {
		return nil, errMissingAddress
	}
	fd, err := dialFuchsia(ctx, net, laddr, raddr)
	if err != nil {
		return nil, err
	}
	return newTCPConn(fd), nil
}

func (ln *TCPListener) ok() bool { return ln != nil && ln.fd != nil }

func (ln *TCPListener) accept() (*TCPConn, error) {
	fd, err := ln.fd.accept()
	if err != nil {
		return nil, err
	}
	return newTCPConn(fd), nil
}

func (ln *TCPListener) close() error { return ln.fd.Close() }

func (ln *TCPListener) file() (*os.File, error) { return ln.fd.dup() }

func listenTCP(ctx context.Context, network string, laddr *TCPAddr) (*TCPListener, error) {
	fd, err := dialFuchsia(ctx, network, laddr, nil)
	if err != nil {
		return nil, err
	}
	return &TCPListener{fd}, nil
}
