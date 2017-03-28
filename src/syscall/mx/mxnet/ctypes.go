// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mxnet

// Types from <mxio/socket.h>.
// These were extracted from cgo by building the program:
//
//	package typesextract
//
//	// #include <arpa/inet.h>
//	// #include <netdb.h>
//	// #include <netinet/in.h>
//	// #include <netinet/tcp.h>
//	// #include <sys/ioctl.h>
//	// #include <sys/socket.h>
//	// #include <sys/types.h>
//	// #include <mxio/socket.h>
//	import "C"
//
//	func F() {
//		_ = C.struct_addrinfo{}
//		_ = C.struct_sockaddr_in{}
//		_ = C.struct_sockaddr_in6{}
//		_ = C.struct_mxrio_gai_req{}
//		_ = C.struct_mxrio_gai_reply{}
//		_ = C.struct_mxrio_sockaddr_reply{}
//		_ = C.struct_mxrio_sockopt_req_reply{}
//		_ = C.struct_mxio_socket_msg{}
//		_ = C.AF_INET
//		_ = C.AF_INET6
//		_ = C.MXRIO_GAI_REQ_NODE_MAXLEN
//		_ = C.MXRIO_GAI_REQ_SERVICE_MAXLEN
//		_ = C.MXRIO_GAI_REPLY_MAX
//	}
//
// Then the names were cleaned up manually, with commands like:
//	s/_t$//
//	s/_Ctype_struct_/c_/
//	s/_Ctype_/c_/
//	...
//
// Two big changes are c_in_port and c_in_addr, which for historcical
// UNIX reasons have terrible type representations, so we treat them
// as byte arrays.
//
// This will probably all be FIDL one day.

const SOCK_STREAM = 1
const SOCK_DGRAM = 2

const AF_INET = 2
const AF_INET6 = 10

const MXRIO_GAI_REQ_NODE_MAXLEN = 256
const MXRIO_GAI_REQ_SERVICE_MAXLEN = 256
const MXRIO_GAI_REPLY_MAX = 4

type c_socklen uint32
type c_in_port [2]byte // uint16 in C, but stored in network order
type c_sa_family uint16
type c_in_addr [4]byte // uint32 in C, but stored in network order
type c_ulong uint64

type c_addrinfo struct {
	ai_flags    int32
	ai_family   int32
	ai_socktype int32
	ai_protocol int32
	ai_addrlen  c_socklen
	_           [4]byte
	// The following pointers are unused on the wire.
	ai_addr      uintptr // *c_sockaddr
	ai_canonname uintptr // *int8
	ai_next      uintptr // *c_addrinfo
}

type c_mxrio_gai_reply struct {
	res [MXRIO_GAI_REPLY_MAX]struct {
		ai   c_addrinfo
		addr c_sockaddr_storage
	}
	nres   int32
	retval int32
}

type c_mxrio_gai_req struct {
	node_is_null    uint8
	service_is_null uint8
	hints_is_null   uint8
	reserved        uint8
	reserved2       uint32
	node            [MXRIO_GAI_REQ_NODE_MAXLEN]uint8
	service         [MXRIO_GAI_REQ_SERVICE_MAXLEN]uint8
	hints           c_addrinfo
}

type c_mxrio_sockaddr_reply struct {
	addr c_sockaddr_storage
	len  c_socklen
	_    [4]byte
}

type c_mxrio_sockopt_req_reply struct {
	level   int32
	optname int32
	optval  [8]uint8
	optlen  c_socklen
}

type c_sockaddr struct {
	sa_family c_sa_family
	sa_data   [14]uint8
}

type c_sockaddr_in struct {
	sin_family c_sa_family
	sin_port   c_in_port // network order
	sin_addr   c_in_addr
	sin_zero   [8]uint8
}

type c_in6_addr [16]byte

type c_sockaddr_in6 struct {
	sin6_family   c_sa_family
	sin6_port     c_in_port // network order
	sin6_flowinfo uint32
	sin6_addr     c_in6_addr
	sin6_scope_id uint32
}

type c_sockaddr_storage struct {
	ss_family    c_sa_family
	_            [6]byte
	__ss_align   c_ulong
	__ss_padding [112]uint8
}

type c_mxio_socket_msg_hdr struct {
	addr    c_sockaddr_storage
	addrlen c_socklen
	flags   int32
}

type c_mxio_socket_msg struct {
	hdr  c_mxio_socket_msg_hdr
	data [1]uint8
	_    [7]byte
}
