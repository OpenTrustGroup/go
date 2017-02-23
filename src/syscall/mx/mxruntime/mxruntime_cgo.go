// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build fuchsia

package mxruntime

import "syscall/mx"

// #include "magenta/process.h"
import "C"

func getStartupHandle(info HandleInfo) mx.Handle {
	id := uint32(info.Arg)<<16 | uint32(info.Type)
	h := C.mx_get_startup_handle(C.uint32_t(id))
	return mx.Handle(h)
}
