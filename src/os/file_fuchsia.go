// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "time"

type File struct {
	fd      int
	name    string
	dirinfo *dirInfo // nil unless directory being read
}

type dirInfo struct {
}

func (f *File) readdirnames(n int) (names []string, err error) {
	panic("TODO")
}

func (f *File) readdir(n int) (fi []FileInfo, err error) {
	panic("TODO")
}

func (f *File) read(b []byte) (n int, err error) {
	panic("TODO")
}

func (f *File) pread(b []byte, off int64) (n int, err error) {
	panic("TODO")
}

func (f *File) write(b []byte) (n int, err error) {
	panic("TODO")
}

func (f *File) pwrite(b []byte, off int64) (n int, err error) {
	panic("TODO")
}

func (f *File) seek(offset int64, whence int) (ret int64, err error) {
	panic("TODO")
}

func (f *File) Chmod(mode FileMode) error {
	panic("TODO")
}

func (f *File) Chown(uid, gid int) error {
	panic("TODO")
}

func (f *File) Truncate(size int64) error {
	panic("TODO")
}

func (f *File) Sync() error {
	panic("TODO")
}

func (f *File) Close() error {
	panic("TODO")
}

func (f *File) Stat() (FileInfo, error) {
	panic("TODO")
}

func NewFile(fd uintptr, name string) *File {
	panic("TODO")
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	panic("TODO")
}

func sameFile(fs1, fs2 *fileStat) bool {
	panic("TODO")
}

func rename(oldname, newname string) error {
	panic("TODO")
}

func epipecheck(file *File, e error) {
	panic("TODO")
}

func syscallMode(i FileMode) (o uint32) {
	panic("TODO")
}

func Remove(name string) error {
	panic("TODO")
}

func Chmod(name string, mode FileMode) error {
	panic("TODO")
}

func Chown(name string, uid, gid int) error {
	panic("TODO")
}

func Chtimes(name string, atime time.Time, mtime time.Time) error {
	panic("TODO")
}

func Stat(name string) (FileInfo, error) {
	panic("TODO")
}

func Readlink(name string) (string, error) {
	panic("TODO")
}

func TempDir() string {
	panic("TODO")
}

func Pipe() (r *File, w *File, err error) {
	panic("TODO")
}

const DevNull = "/dev/null" // TODO
