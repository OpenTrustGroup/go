// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build fuchsia

package mxruntime

import "syscall/zx"

// #include "zircon/process.h"
import "C"

func getStartupHandle(info HandleInfo) zx.Handle {
	id := uint32(info.Arg)<<16 | uint32(info.Type)
	h := C.zx_get_startup_handle(C.uint32_t(id))
	return zx.Handle(h)
}
