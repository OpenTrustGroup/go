// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import "internal/syscall/fuchsia"

var (
	Stdin  = 0
	Stdout = 1
	Stderr = 2
)

const (
	O_RDONLY = 0x0
	O_WRONLY = 0x1
	O_RDWR   = 0x2

	O_APPEND = 0x8
	O_CREAT  = 0x200
	O_EXCL   = 0x800
	O_SYNC   = 0x80
	O_TRUNC  = 0x400
)

// TODO(crawshaw): generate a zerrors file once cgo is working
const (
	ENOTDIR      = Errno(0x14)
	EISDIR       = Errno(0x15)
	EINVAL       = Errno(0x16)
	EPIPE        = Errno(0x20)
	ENAMETOOLONG = Errno(0x24)
	ENOPROTOOPT  = Errno(0x5c)
	EFUCHSIA     = Errno(0xfa)
)

const (
	SOMAXCONN = 0x80
)

type Stat_t struct {
}

func Getpid() (pid int) {
	panic("syscall.Getpid TODO")
}

func Getppid() (ppid int) {
	panic("syscall.Getppid TODO")
}

func Getuid() (uid int) {
	panic("syscall.Getuid TODO")
}

func Geteuid() (uid int) {
	panic("syscall.Geteuid TODO")
}

func Getgid() (uid int) {
	panic("syscall.Getgid TODO")
}

func Getegid() (uid int) {
	panic("syscall.Getegid TODO")
}

func Getgroups() (gids []int, err error) {
	panic("syscall.Getgroups TODO")
}

func Exit(code int) {
	fuchsia.Sys_exit(code)
}

func Getpagesize() int { return 4096 }

const ImplementsGetwd = true

func Getwd() (wd string, err error) {
	panic("syscall.Getwd TODO")
}

func Chdir(path string) (err error) {
	panic("syscall.Chdir TODO")
}
func Mkdir(path string, mode uint32) (err error) {
	panic("syscall.Mkdir TODO")
}
func Rmdir(path string) (err error) {
	panic("syscall.Rmdir TODO")
}
func Unlink(path string) (err error) {
	panic("syscall.Unlink TODO")
}
func Rename(oldpath, newpath string) (err error) {
	panic("syscall.Rename TODO")
}
func Fchdir(fd int) (err error) {
	panic("syscall.Fchdir TODO")
}
