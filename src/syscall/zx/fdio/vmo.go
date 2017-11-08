// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"unsafe"

	"syscall/zx"
)

type VMOFile struct {
	h   zx.VMO
	off int64
	len int64
	end int64
	at  int64
}

func NewVMOFile(handle zx.Handle, extra []byte) (*VMOFile, error) {
	// fdio_vmofile_create
	f := &VMOFile{
		h:   zx.VMO(handle),
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
		err = zx.EOF
	}
	if len(data) == 0 {
		return 0, err
	}
	n, err = f.h.Read(data, uint64(f.at))
	f.at += int64(n)
	return n, err
}

func (f *VMOFile) ReadAt(data []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "fdio.VMOFile.ReadAt"}
	}
	if off >= f.end {
		return 0, zx.EOF
	}
	if len(data) == 0 {
		return 0, zx.EOF
	}
	n = len(data)
	max := int(f.end - off)
	if max < n {
		n = max
		err = zx.EOF
	}
	data = data[:n]
	if n2, err2 := f.h.Read(data, uint64(off)); err2 != nil {
		return n2, err2
	}
	return
}

func (f *VMOFile) Write(data []byte) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
}

func (f *VMOFile) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
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
		return 0, zx.Error{Status: zx.ErrOutOfRange, Text: "rio.VMOFile"}
	}
	f.at = f.off + at
	return at, nil
}

func (f *VMOFile) Close() error {
	if err := f.h.Close(); err != nil {
		return err
	}
	f.h = 0
	return nil
}

func (f *VMOFile) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
}
func (f *VMOFile) Clone() ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
}
func (f *VMOFile) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
}
func (f *VMOFile) IoctlSetHandle(op uint32, in zx.Handle) error {
	return zx.Error{Status: zx.ErrNotSupported, Text: "rio.VMOFile"}
}

func (f *VMOFile) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	for _, h := range handles {
		h.Close()
	}

	switch op {
	case OpStat:
		attr := Vnattr{
			Size: uint64(f.end - f.off),
			Mode: VtypeFile | VtypeIRUSR,
		}
		if unsafe.Sizeof(attr) > uintptr(len(out)) {
			return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "rio.VMOFile"}
		}
		*(*Vnattr)(unsafe.Pointer(&out[0])) = attr
		return int(unsafe.Sizeof(attr)), nil
	default:
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "rio.VMOFile"}
	}
}
