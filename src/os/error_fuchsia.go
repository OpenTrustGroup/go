// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package os

import (
	"syscall"
	"syscall/mx"
)

func isExist(err error) bool {
	err = underlyingError(err)
	switch err := err.(type) {
	case mx.Error:
		return err.Status == mx.ErrAlreadyExists
	}
	return err == syscall.EEXIST || err == syscall.ENOTEMPTY || err == ErrExist
}

func isNotExist(err error) bool {
	err = underlyingError(err)
	switch err := err.(type) {
	case mx.Error:
		return err.Status == mx.ErrNotFound
	}
	return err == syscall.ENOENT || err == ErrNotExist
}

func isPermission(err error) bool {
	err = underlyingError(err)
	switch err := err.(type) {
	case mx.Error:
		return err.Status == mx.ErrAccessDenied
	}
	return err == syscall.EACCES || err == syscall.EPERM || err == ErrPermission
}
