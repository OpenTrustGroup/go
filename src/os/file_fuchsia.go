// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"io"
	"syscall"
	"time"

	"syscall/mx/mxio/pipe"
)

func (f *File) readdir(n int) (fi []FileInfo, err error) {
	dirname := f.name
	if dirname == "" {
		dirname = "."
	}
	names, err := f.Readdirnames(n)
	fi = make([]FileInfo, 0, len(names))
	for _, filename := range names {
		fip, lerr := lstat(dirname + "/" + filename)
		if IsNotExist(lerr) {
			// File disappeared between readdir + stat.
			// Just treat it as if it didn't exist.
			continue
		}
		if lerr != nil {
			return fi, lerr
		}
		fi = append(fi, fip)
	}
	if len(fi) == 0 && err == nil && n > 0 {
		// Per File.Readdir, the slice must be non-empty or err
		// must be non-nil if n > 0.
		err = io.EOF
	}
	return fi, err
}

func (f *File) readdirnames(n int) (names []string, err error) {
	return syscall.Readdirnames(f.fd, n)
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
	if f == nil {
		return ErrInvalid
	}
	if e := syscall.Fsync(f.fd); e != nil {
		return e
	}
	return nil
}

func syscallMode(i FileMode) (o uint32) {
	o |= uint32(i.Perm())
	// TODO i&ModeSetuid != 0
	// TODO i&ModeSetgid != 0
	// TODO i&ModeSticky != 0
	return o
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

func Readlink(name string) (string, error) {
	panic("TODO os.Readlink")
}

func Pipe() (r *File, w *File, err error) {
	p0, p1, err := pipe.NewPipe()
	if err != nil {
		return nil, nil, err
	}
	r = NewFile(uintptr(syscall.OpenMXIO(p0)), "|0")
	w = NewFile(uintptr(syscall.OpenMXIO(p1)), "|1")
	// TODO CloseOnExec equivalent
	return r, w, nil
}

func sigpipe() // implemented in package runtime

var supportsCloseOnExec = true

func fillFileStatFromSys(fs *fileStat, name string) {
	fs.name = basename(name)
	fs.size = int64(fs.sys.Size)
	// For now, pretend files have all bits. This should really be:
	// fs.mode = FileMode(fs.sys.Dev & 0777)
	// but we don't really set file modes properly and testing for executables
	// in $PATH fails without the x bit set.
	fs.mode = FileMode(0766)

	switch fs.sys.Dev & 0xf000 {
	case 0x4000:
		fs.mode = fs.mode | ModeDir
	case 0x2000:
		fs.mode = fs.mode | ModeCharDevice
	case 0xa000:
		fs.mode = fs.mode | ModeSymlink
	case 0xc000:
		fs.mode = fs.mode | ModeSocket
	}
	fs.modTime = time.Unix(int64(fs.sys.ModifyTime), 0)
}

func atime(fi FileInfo) time.Time {
	// TODO: mxio doesn't have atime yet
	return time.Time{}
}

func executable() (string, error) {
	return "", errors.New("os.Executable not supported on fuchsia")
}
