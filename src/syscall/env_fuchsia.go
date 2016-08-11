// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

func Unsetenv(key string) error {
	panic("TODO")
}

func Getenv(key string) (value string, found bool) {
	panic("TODO")
}

func Setenv(key, value string) error {
	panic("TODO")
}

func Clearenv() {
	panic("TODO")
}

func Environ() []string {
	panic("TODO")
}
