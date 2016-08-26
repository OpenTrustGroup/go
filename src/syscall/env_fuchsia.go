// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

func Unsetenv(key string) error {
	panic("syscall.Unsetenv TODO")
}

func Getenv(key string) (value string, found bool) {
	println("syscall.Getenv TODO")
	return "", false
}

func Setenv(key, value string) error {
	panic("syscall.Setenv TODO")
}

func Clearenv() {
	panic("syscall.Clearenv TODO")
}

func Environ() []string {
	panic("syscall.Environ TODO")
}
