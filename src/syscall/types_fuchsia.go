// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

type Errno uintptr

func (e Errno) Error() string {
	return "errno(" + itoa(int(e)) + ")"
}

type Timespec struct {
	Sec  int64
	Nsec int64
}

type Timeval struct {
	Sec  int64
	Usec int64
}
