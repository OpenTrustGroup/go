// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

// TODO(crawshaw): set this during init via syscall
var randomNumber uint32 = 7

//go:nosplit
func cputicks() int64 {
	// TODO: switch to zx.Sys_clock_get(MX_CLOCK_MONOTONIC) ??

	// Currently cputicks() is used in blocking profiler and to seed fastrand1().
	// nanotime() is a poor approximation of CPU ticks that is enough for the profiler.
	// randomNumber provides better seeding of fastrand1.
	return nanotime() + int64(randomNumber)
}
