// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rand

import "syscall/zx"

func init() {
	Reader = new(cprngReader)
}

// cprngReader uses the PRNG exposed by zircon syscall.
//
// TODO(crawshaw): does it perform as well as /dev/urandom?
// If not, borrow newReader from rand_unix.go and use this
// as the entropy source.
type cprngReader struct{}

func (r *cprngReader) Read(b []byte) (n int, err error) {
	return zx.RandRead(b)
}
