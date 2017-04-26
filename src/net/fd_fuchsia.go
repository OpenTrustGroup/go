// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"errors"
	"os"
	"syscall"
	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxio/rio"
	"time"
)

func mxStatus(err error) mx.Status {
	if err == nil {
		return mx.ErrOk
	}
	if status, ok := err.(mx.Error); ok {
		return status.Status
	}
	return mx.ErrInternal
}

// Network file descriptor.
type netFD struct {
	// locking/lifetime of sysfd + serialize access to Read and Write methods
	fdmu fdMutex
	hsrc *os.File
	m    mxio.MXIO

	isConnected bool

	// immutable until Close
	net          string
	laddr, raddr Addr
}

func (fd *netFD) destroy() {
	panic("TODO")
}

func sysInit() {
}

func (fd *netFD) isTCP() bool {
	return len(fd.net) >= 3 && fd.net[:3] == "tcp"
}

func (fd *netFD) Read(b []byte) (n int, err error) {
	return fd.m.Read(b)
}

func (fd *netFD) Write(b []byte) (n int, err error) {
	return fd.m.Write(b)
}

func (fd *netFD) recvMsg(b []byte) (n, flags int, addr string, port uint16, err error) {
	data, flags, addr, port, err := fd.m.(*rio.Socket).RecvMsg(len(b))
	n = copy(b, data)
	return n, flags, addr, port, err
}

func (fd *netFD) sendMsg(b []byte, addr string, port uint16) (n int, err error) {
	return fd.m.(*rio.Socket).SendMsg(b, addr, port)
}

func (fd *netFD) closeRead() error {
	return errors.New("net: closeRead not implemented on fuchsia")
}

func (fd *netFD) closeWrite() error {
	return errors.New("net: closeWrite not implemented on fuchsia")
}

func (fd *netFD) Close() error {
	return fd.hsrc.Close()
}

func (fd *netFD) dup() (*os.File, error) {
	return nil, errors.New("net: dup not implemented on fuchsia")
}

func (fd *netFD) setDeadline(t time.Time) error {
	return errors.New("net: setDeadline not implemented on fuchsia")
}

func (fd *netFD) setReadDeadline(t time.Time) error {
	return errors.New("net: setReadDeadline not implemented on fuchsia")
}

func (fd *netFD) setWriteDeadline(t time.Time) error {
	return errors.New("net: setWriteDeadline not implemented on fuchsia")
}

func (fd *netFD) accept() (netfd *netFD, err error) {
	if err := fd.readLock(); err != nil {
		return nil, err
	}
	defer fd.readUnlock()

	m, raddrip, raddrport, err := rio.Accept(fd.m)
	if err != nil {
		return nil, err
	}
	netfd = &netFD{
		hsrc:  os.NewFile(uintptr(syscall.OpenMXIO(m)), "socket"),
		m:     m,
		net:   fd.net,
		laddr: fd.laddr, // TODO not right
		raddr: &TCPAddr{
			IP:   IP(raddrip),
			Port: int(raddrport),
		},
	}
	return netfd, nil
}
