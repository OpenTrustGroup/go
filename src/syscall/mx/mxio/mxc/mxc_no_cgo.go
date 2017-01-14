// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia
// +build !cgo

package mxc

import (
	"syscall/mx"
	"syscall/mx/mxio"
)

func transferMXIOFromCFD(fd int) (mxio.MXIO, error) {
	return nil, mx.ErrNotSupported
}
