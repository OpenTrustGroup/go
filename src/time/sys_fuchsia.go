// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package time

func initLocal() {
	localLoc.name = "UTC"
}

func loadLocation(name string) (*Location, error) {
	panic("TODO")
}

func readFile(name string) ([]byte, error) {
	panic("TODO")
}

func open(name string) (uintptr, error) {
	panic("TODO")
}

func closefd(fd uintptr) {
	panic("TODO")
}

func preadn(fd uintptr, buf []byte, off int) error {
	panic("TODO")
}

func isNotExist(err error) bool {
	panic("TODO")
}
