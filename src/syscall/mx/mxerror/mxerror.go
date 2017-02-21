// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mxerror

import (
	"fmt"
	"syscall/mx"
)

// Status extracts an mx.Status from an error.
func Status(err error) mx.Status {
	if err == nil {
		return mx.ErrOk
	}
	if s, hasStatus := err.(mx.Error); hasStatus {
		return s.Status
	}
	return mx.ErrInternal
}

// Prependf prepends a format string onto an error.
// If the error is an mx.Error, the status code is retained.
func Prependf(err error, format string, args ...interface{}) error {
	str := fmt.Sprintf(format, args...)
	if e, isError := err.(mx.Error); isError {
		if e.Text == "" {
			e.Text = str
		} else {
			e.Text = str + ": " + e.Text
		}
		return e
	}
	return fmt.Errorf("%s: %v", str, err)
}

// Errorf creates an error with an mx.Status code.
func Errorf(s mx.Status, format string, args ...interface{}) error {
	return &mx.Error{
		Status: s,
		Text:   fmt.Sprintf(format, args...),
	}
}
