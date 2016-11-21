// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rio

import (
	"unsafe"

	"syscall/mx"
	"syscall/mx/mxio"
)

type VMOFile struct {
	h   mx.VMO
	off int64
	len int64
	end int64
	at  int64
}

func NewVMOFile(handle mx.Handle, extra []byte) (*VMOFile, error) {
	// mxio_vmofile_create
	f := &VMOFile{
		h:   mx.VMO(handle),
		off: *(*int64)(unsafe.Pointer(&extra[0:8][0])),
		len: *(*int64)(unsafe.Pointer(&extra[8:16][0])),
	}
	f.end = f.off + f.len
	f.at = f.off
	return f, nil
}

func (f *VMOFile) Read(data []byte) (n int, err error) {
	if int64(len(data)) > f.end-f.at {
		data = data[:f.end-f.at]
		err = mx.EOF
	}
	if len(data) == 0 {
		return 0, err
	}
	n, err = f.h.Read(data, uint64(f.at))
	f.at += int64(n)
	return n, err
}

func (f *VMOFile) ReadAt(data []byte, off int64) (int, error) {
	panic("TODO")
}

func (f *VMOFile) Write(data []byte) (int, error) {
	return 0, mx.ErrNotSupported
}

func (f *VMOFile) WriteAt(data []byte, off int64) (int, error) {
	return 0, mx.ErrNotSupported
}

func (f *VMOFile) Seek(offset int64, whence int) (int64, error) {
	var at int64
	switch whence {
	case 0: // SEEK_SET
		at = offset
	case 1: // SEEK_CUR
		at = f.at - f.off + offset
	case 2: // SEEK_END
		at = f.end - f.off + offset
	}

	if at > f.end-f.off {
		return 0, mx.ErrOutOfRange
	}
	f.at = at
	return at, nil
}

func (f *VMOFile) Close() error {
	if err := f.h.Close(); err != nil {
		return err
	}
	f.h = 0
	return nil
}

func (f *VMOFile) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	return nil, mx.ErrNotSupported
}
func (f *VMOFile) Clone() ([]mx.Handle, error) {
	return nil, mx.ErrNotSupported
}
func (f *VMOFile) Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error) {
	return nil, nil, mx.ErrNotSupported
}

func (f *VMOFile) Misc(op uint32, off int64, in, out []byte) (n int, err error) {
	switch op {
	case OpStat:
		attr := mxio.Vnattr{
			Size: uint64(f.end - f.off),
			Mode: mxio.VtypeFile | mxio.VtypeIRUSR,
		}
		if unsafe.Sizeof(attr) > uintptr(len(out)) {
			return 0, mx.ErrInvalidArgs
		}
		*(*mxio.Vnattr)(unsafe.Pointer(&out[0])) = attr
		return int(unsafe.Sizeof(attr)), nil
	default:
		return 0, mx.ErrInvalidArgs
	}
}
