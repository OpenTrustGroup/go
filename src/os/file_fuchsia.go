// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"runtime"
	"time"
)

type File struct {
	*file
}

type file struct {
	fd      int
	name    string
	dirinfo *dirInfo // nil unless directory being read
}

func (f *File) Fd() uintptr {
	if f == nil {
		return ^(uintptr(0))
	}
	return uintptr(f.fd)
}

func (f *File) readdirnames(n int) (names []string, err error) {
	panic("TODO os.readdirnames")
}

func (f *File) readdir(n int) (fi []FileInfo, err error) {
	panic("TODO os.readdir")
}

func (f *File) read(b []byte) (n int, err error) {
	panic("TODO os.read")
}

func (f *File) pread(b []byte, off int64) (n int, err error) {
	panic("TODO os.pread")
}

func (f *File) write(b []byte) (n int, err error) {
	panic("TODO os.write")
}

func (f *File) pwrite(b []byte, off int64) (n int, err error) {
	panic("TODO os.pwrite")
}

func (f *File) seek(offset int64, whence int) (ret int64, err error) {
	panic("TODO os.seek")
}

func (f *File) Chmod(mode FileMode) error {
	panic("TODO os.Chmod")
}

func (f *File) Chown(uid, gid int) error {
	panic("TODO os.Chown")
}

func (f *File) Truncate(size int64) error {
	panic("TODO os.Truncate")
}

func (f *File) Sync() error {
	panic("TODO os.Sync")
}

func (f *File) Close() error {
	if f == nil {
		return ErrInvalid
	}
	return f.file.close()
}

func (file *file) close() error {
	panic("TODO file.close")
}

func (f *File) Stat() (FileInfo, error) {
	panic("TODO os.Stat")
}

func NewFile(fd uintptr, name string) *File {
	fdi := int(fd)
	if fdi < 0 {
		return nil
	}
	f := &File{&file{fd: fdi, name: name}}
	runtime.SetFinalizer(f.file, (*file).close)
	return f
}

func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	panic("TODO os.OpenFile")
}

type dirInfo struct {
}

func sameFile(fs1, fs2 *fileStat) bool {
	panic("TODO os.sameFile")
}

func rename(oldname, newname string) error {
	panic("TODO os.rename")
}

func epipecheck(file *File, e error) {
	panic("TODO os.epipecheck")
}

func syscallMode(i FileMode) (o uint32) {
	panic("TODO os.syscallMode")
}

func Remove(name string) error {
	panic("TODO os.Remove")
}

func Chmod(name string, mode FileMode) error {
	panic("TODO os.Chmod")
}

func Chown(name string, uid, gid int) error {
	panic("TODO os.Chown")
}

func Chtimes(name string, atime time.Time, mtime time.Time) error {
	panic("TODO os.Chtimes")
}

func Stat(name string) (FileInfo, error) {
	panic("TODO os.Stat")
}

func Readlink(name string) (string, error) {
	panic("TODO os.Readlink")
}

func TempDir() string {
	panic("TODO os.TempDir")
}

func Pipe() (r *File, w *File, err error) {
	panic("TODO os.Pipe")
}

func Symlink(oldname, newname string) error {
	panic("TODO os.Symlink")
}

const DevNull = "/dev/null" // TODO
