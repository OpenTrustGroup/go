// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mxc

import "syscall"

func ExtractCMXIO(fd int) (uintptr, error) {
	// Extract the 'C' mxio
	f, err := transferMXIOFromCFD(fd)
	if err != nil {
		return 0, err
	}
	// Convert it to a 'Go' mxio
	return uintptr(syscall.OpenMXIO(f)), nil
}
