// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"syscall"
	"time"

	"syscall/mx/mxio/pipe"
)

func (f *File) Chmod(mode FileMode) error {
	return errors.New("Chmod unimplemented")
}

func (f *File) Chown(uid, gid int) error {
	return errors.New("Chown unimplemented")
}

func (f *File) Truncate(size int64) error {
	if f == nil {
		return ErrInvalid
	}
	return syscall.Ftruncate(f.fd, size)
}

func (f *File) Sync() error {
	if f == nil {
		return ErrInvalid
	}
	return syscall.Fsync(f.fd)
}

func syscallMode(i FileMode) (o uint32) {
	o |= uint32(i.Perm())
	// TODO i&ModeSetuid != 0
	// TODO i&ModeSetgid != 0
	// TODO i&ModeSticky != 0
	return o
}

func Chmod(name string, mode FileMode) error {
	return errors.New("Chmod unimplemented")
}

func Chown(name string, uid, gid int) error {
	return errors.New("Chown unimplemented")
}

func Chtimes(name string, atime time.Time, mtime time.Time) error {
	var utimes [2]syscall.Timespec
	utimes[0] = syscall.NsecToTimespec(atime.UnixNano())
	utimes[1] = syscall.NsecToTimespec(mtime.UnixNano())
	if e := syscall.UtimesNano(fixLongPath(name), utimes[0:]); e != nil {
		return &PathError{"chtimes", name, e}
	}
	return nil
}

func Readlink(name string) (string, error) {
	return "", errors.New("Readlink unimplemented")
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
