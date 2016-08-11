// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"io"
	"os"
)

func (c *TCPConn) readFrom(r io.Reader) (int64, error) {
	return genericReadFrom(c, r)
}

func dialTCP(ctx context.Context, net string, laddr, raddr *TCPAddr) (*TCPConn, error) {
	panic("TODO")
}

func (ln *TCPListener) ok() bool {
	panic("TODO")
}

func (ln *TCPListener) accept() (*TCPConn, error) {
	panic("TODO")
}

func (ln *TCPListener) close() error {
	panic("TODO")
}

func (ln *TCPListener) file() (*os.File, error) {
	panic("TODO")
}

func listenTCP(ctx context.Context, network string, laddr *TCPAddr) (*TCPListener, error) {
	panic("TODO")
}
