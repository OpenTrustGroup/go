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
	"syscall/mx/mxnet"
	"time"
)

func probeIPv4Stack() bool {
	return true
}

func probeIPv6Stack() (supportsIPv6, supportsIPv4map bool) {
	return true, true
}

// A sockaddr represents a TCP, UDP, IP or Unix network endpoint
// address that can be converted into an mxio sockaddr message.
type sockaddr interface {
	Addr

	family() int
	isWildcard() bool
	sockaddr(family int) (addr mxnet.Addr, port uint16, err error)
}

func favoriteAddrFamily(net string, laddr, raddr sockaddr, mode string) (family int, ipv6only bool) {
	switch net[len(net)-1] {
	case '4':
		return syscall.AF_INET, false
	case '6':
		return syscall.AF_INET6, true
	}

	if mode == "listen" && (laddr == nil || laddr.isWildcard()) {
		if supportsIPv4map || !supportsIPv4 {
			return syscall.AF_INET6, false
		}
		if laddr == nil {
			return syscall.AF_INET, false
		}
		return laddr.family(), false
	}

	if (laddr == nil || laddr.family() == syscall.AF_INET) &&
		(raddr == nil || raddr.family() == syscall.AF_INET) {
		return syscall.AF_INET, false
	}
	return syscall.AF_INET6, false
}

func dialFuchsia(ctx context.Context, net string, laddr, raddr sockaddr) (fd *netFD, err error) {
	family, _ := favoriteAddrFamily(net, laddr, raddr, "dial")

	sotype := syscall.SOCK_STREAM
	if len(net) >= 3 && net[:3] == "udp" {
		sotype = syscall.SOCK_DGRAM
	}
	proto := syscall.IPPROTO_IP

	var f *os.File
	// Wait for the network stack to publish the socket device.
	// See similar logic in magenta/system/ulib/mxio/bsdsocket.c.
	for i := 0; i < 40; i++ {
		path := "/dev/socket/socket/" + strconv.Itoa(family) + "/" + strconv.Itoa(sotype) + "/" + strconv.Itoa(proto)
		f, err = os.Open(path)
		if !os.IsNotExist(err) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		return nil, err
	}
	m := syscall.MXIOForFD(int(f.Fd()))
	if sotype == syscall.SOCK_DGRAM {
		mxio.SetDgram(m)
	}

	fd = &netFD{
		hsrc:   f,
		m:      m,
		family: family,
		sotype: sotype,
		net:    net,
	}
	if laddr != nil {
		addr, port, err := laddr.sockaddr(family)
		if err != nil {
			return nil, err
		}
		if addr != "" || port != 0 {
			if err := mxio.Bind(fd.m, addr, port); err != nil {
				return nil, err
			}
		}
		if raddr == nil {
			switch sotype {
			case syscall.SOCK_STREAM:
				if err := mxio.ListenStream(m, listenerBacklog); err != nil {
					return nil, err
				}
			case syscall.SOCK_DGRAM:
				return nil, errors.New("net: TODO listen datagram")
			}
		}
	}
	if raddr != nil {
		addr, port, err := raddr.sockaddr(family)
		if err != nil {
			return nil, err
		}
		if err := mxio.Connect(fd.m, addr, port); err != nil {
			return nil, err
		}
		fd.isConnected = true
	}
	fd.setAddr()

	return fd, nil
}

func ipToSockaddr(family int, ip IP, port int, zone string) (addr mxnet.Addr, portres uint16, err error) {
	switch family {
	case syscall.AF_INET:
		if len(ip) == 0 {
			ip = IPv4zero
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return "", 0, &AddrError{Err: "non-IPv4 address", Addr: ip.String()}
		}
		return mxnet.Addr(ip4)[:4], uint16(port), nil
	case syscall.AF_INET6:
		if len(ip) == 0 || ip.Equal(IPv4zero) {
			ip = IPv6zero
		}
		ip6 := ip.To16()
		if ip6 == nil {
			return "", 0, &AddrError{Err: "non-IPv6 address", Addr: ip.String()}
		}
		return mxnet.Addr(ip6), uint16(port), nil
	}
	return "", 0, &AddrError{Err: "invalid address family", Addr: ip.String()}
}
