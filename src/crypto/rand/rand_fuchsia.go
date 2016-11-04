// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rand

import (
	"syscall/mx"
	"unsafe"
)

func init() {
	Reader = new(cprngReader)
}

// cprngReader uses the PRNG exposed by magenta syscall.
//
// TODO(crawshaw): does it perform as well as /dev/urandom?
// If not, borrow newReader from rand_unix.go and use this
// as the entropy source.
type cprngReader struct{}

func (r *cprngReader) Read(b []byte) (n int, err error) {
	var actual uint
	status := mx.Sys_cprng_draw(unsafe.Pointer(&b[0]), uint(len(b)), &actual)
	if status != mx.ErrOk {
		return 0, status
	}
	return int(actual), nil
}
