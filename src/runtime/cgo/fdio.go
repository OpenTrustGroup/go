// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package cgo

import _ "unsafe" // for go:linkname

//go:cgo_import_static x_cgo_get_initial_handles
//go:linkname x_cgo_get_initial_handles x_cgo_get_initial_handles
//go:linkname _cgo_get_initial_handles runtime._cgo_get_initial_handles
var x_cgo_get_initial_handles byte
var _cgo_get_initial_handles = &x_cgo_get_initial_handles
