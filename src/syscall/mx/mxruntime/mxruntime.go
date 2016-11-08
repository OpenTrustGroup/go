// Copyright 2016 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// +build fuchsia

// Package mxruntime provides access to MXIO process state.
package mxruntime

// TODO: move process handle globals from mx into mxruntime.

import "syscall/mx"

type HandleType uint16

const (
	HandleUser0 HandleType = 0xFFF0
	HandleUser1 HandleType = 0xFFF1
	HandleUser2 HandleType = 0xFFF2
)

type HandleInfo struct {
	Type HandleType
	Arg  uint16
}

func GetStartupHandle(info HandleInfo) mx.Handle {
	return getStartupHandle(info)
}
