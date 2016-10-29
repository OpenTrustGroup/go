// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !fuchsia

package runtime

import _ "unsafe" // for go:linkname

// TODO: move stub into each os_*.go file.

//go:linkname time_now time.now
func time_now() (sec int64, nsec int32)

func unixnanotime() int64 {
	sec, nsec := time_now()
	return sec*1e9 + int64(nsec)
}
