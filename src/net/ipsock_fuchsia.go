// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import (
	"context"
	"errors"
	"os"
	"strconv"
	"syscall"
	"syscall/mx/mxio/rio"
	"syscall/mx/mxnet"
)

func probeIPv4Stack() bool {
	return false
}

func probeIPv6Stack() (supportsIPv6, supportsIPv4map bool) {
	return false, false
}

// A sockaddr represents a TCP, UDP, IP or Unix network endpoint
// address that can be converted into an mxio sockaddr message.
type sockaddr interface {
	Addr

	sockaddr() (addr string, port uint16)
}

func dialFuchsia(ctx context.Context, net string, laddr, raddr sockaddr) (fd *netFD, err error) {
	domain := syscall.AF_INET6
	if net[len(net)-1] == '4' {
		domain = syscall.AF_INET
	}

	sotype := syscall.SOCK_STREAM
	if len(net) >= 3 && net[:3] == "udp" {
		sotype = syscall.SOCK_DGRAM
	}
	proto := syscall.IPPROTO_IP
	path := "/dev/socket/socket/" + strconv.Itoa(domain) + "/" + strconv.Itoa(sotype) + "/" + strconv.Itoa(proto)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	m := syscall.MXIOForFD(int(f.Fd()))
	if sotype == syscall.SOCK_DGRAM {
		rio.SetDgram(m)
	}

	fd = &netFD{
		hsrc:  f,
		m:     m,
		net:   net,
		laddr: laddr,
		raddr: raddr,
	}
	if laddr != nil {
		addr, port := laddr.sockaddr()
		if err := rio.Bind(fd.m, mxnet.Addr(addr), port); err != nil {
			return nil, err
		}
		if raddr == nil {
			switch sotype {
			case syscall.SOCK_STREAM:
				if err := rio.ListenStream(m, listenerBacklog); err != nil {
					return nil, err
				}
			case syscall.SOCK_DGRAM:
				return nil, errors.New("net: TODO listen datagram")
			}
		}
	}
	if raddr != nil {
		addr, port := raddr.sockaddr()
		if err := rio.Connect(fd.m, mxnet.Addr(addr), port); err != nil {
			return nil, err
		}
		fd.isConnected = true
	}
	// TODO: setAddr

	return fd, nil
}
