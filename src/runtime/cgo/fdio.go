// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package cgo

import _ "unsafe" // for go:linkname

//go:cgo_import_static x_cgo_get_fdio_root
//go:linkname x_cgo_get_fdio_root x_cgo_get_fdio_root
//go:linkname _cgo_get_fdio_root runtime._cgo_get_fdio_root
var x_cgo_get_fdio_root byte
var _cgo_get_fdio_root = &x_cgo_get_fdio_root
