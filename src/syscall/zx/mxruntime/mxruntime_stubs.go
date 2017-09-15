// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build fuchsia
// +build !cgo

package mxruntime

func getStartupHandle(info HandleInfo) zx.Handle {
	panic("mxruntime.GetStartupHandle not implemented")
}
