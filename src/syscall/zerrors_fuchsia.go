// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

const (
	SOCK_STREAM = 1
	SOCK_DGRAM  = 2
)

const (
	IPPROTO_IP     = 0
	IPPROTO_ICMP   = 1
	IPPROTO_TCP    = 6
	IPPROTO_UDP    = 17
	IPPROTO_ICMPV6 = 58
)

const (
	AF_INET  = 0x2
	AF_INET6 = 0xa
)

const (
	SOL_SOCKET = 0x1
	SOL_TCP    = 0x6
)

const (
	SO_ACCEPTCONN  = 0x1e
	SO_BROADCAST   = 0x6
	SO_BSDCOMPAT   = 0xe
	SO_DEBUG       = 0x1
	SO_DOMAIN      = 0x27
	SO_DONTROUTE   = 0x5
	SO_ERROR       = 0x4
	SO_KEEPALIVE   = 0x9
	SO_LINGER      = 0xd
	SO_NO_CHECK    = 0xb
	SO_PASSCRED    = 0x10
	SO_PEERCRED    = 0x11
	SO_PRIORITY    = 0xc
	SO_PROTOCOL    = 0x26
	SO_RCVBUF      = 0x8
	SO_RCVBUFFORCE = 0x21
	SO_RCVLOWAT    = 0x12
	SO_RCVTIMEO    = 0x14
	SO_REUSEADDR   = 0x2
	SO_REUSEPORT   = 0xf
	SO_SNDBUF      = 0x7
	SO_SNDBUFFORCE = 0x20
	SO_SNDLOWAT    = 0x13
	SO_SNDTIMEO    = 0x15
	SO_TYPE        = 0x3
)

const (
	TCP_NODELAY      = 1
	TCP_MAXSEG       = 2
	TCP_CORK         = 3
	TCP_KEEPIDLE     = 4
	TCP_KEEPINTVL    = 5
	TCP_KEEPCNT      = 6
	TCP_SYNCNT       = 7
	TCP_LINGER2      = 8
	TCP_DEFER_ACCEPT = 9
	TCP_WINDOW_CLAMP = 10
	TCP_INFO         = 11
	TCP_QUICKACK     = 12
)
