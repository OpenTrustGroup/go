// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"os"
	"syscall"
	"time"
)

// Network file descriptor.
type netFD struct {
	// locking/lifetime of sysfd + serialize access to Read and Write methods
	fdmu fdMutex

	// immutable until Close
	net          string
	laddr, raddr Addr
}

func (fd *netFD) destroy() {
	panic("TODO")
}

func sysInit() {
}

func (fd *netFD) Read(b []byte) (n int, err error) {
	panic("TODO")
}

func (fd *netFD) Write(b []byte) (n int, err error) {
	panic("TODO")
}

func (fd *netFD) closeRead() error {
	panic("TODO")
}

func (fd *netFD) closeWrite() error {
	panic("TODO")
}

func (fd *netFD) Close() error {
	panic("TODO")
}

func (fd *netFD) dup() (*os.File, error) {
	panic("TODO")
}

func (fd *netFD) setDeadline(t time.Time) error {
	return syscall.EFUCHSIA
}

func (fd *netFD) setReadDeadline(t time.Time) error {
	return syscall.EFUCHSIA
}

func (fd *netFD) setWriteDeadline(t time.Time) error {
	return syscall.EFUCHSIA
}
