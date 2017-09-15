// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mxerror

import (
	"fmt"
	"syscall/zx"
)

// Status extracts an zx.Status from an error.
func Status(err error) zx.Status {
	if err == nil {
		return zx.ErrOk
	}
	if s, hasStatus := err.(zx.Error); hasStatus {
		return s.Status
	}
	return zx.ErrInternal
}

// Prependf prepends a format string onto an error.
// If the error is an zx.Error, the status code is retained.
func Prependf(err error, format string, args ...interface{}) error {
	str := fmt.Sprintf(format, args...)
	if e, isError := err.(zx.Error); isError {
		if e.Text == "" {
			e.Text = str
		} else {
			e.Text = str + ": " + e.Text
		}
		return e
	}
	return fmt.Errorf("%s: %v", str, err)
}

// Errorf creates an error with an zx.Status code.
func Errorf(s zx.Status, format string, args ...interface{}) error {
	return &zx.Error{
		Status: s,
		Text:   fmt.Sprintf(format, args...),
	}
}
