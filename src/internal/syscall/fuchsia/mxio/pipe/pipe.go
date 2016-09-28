// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package pipe

import (
	"unsafe"

	"internal/syscall/fuchsia"
	"internal/syscall/fuchsia/mxio"
)

type Pipe struct {
	h fuchsia.Handle
}

func NewPipe() (*Pipe, *Pipe, error) {
	var h [2]fuchsia.Handle
	status := fuchsia.Sys_socket_create(&h[0], 0)
	if status < 0 {
		return nil, nil, status
	}
	return &Pipe{h[0]}, &Pipe{h[1]}, nil
}

func (f *Pipe) Read(data []byte) (n int, err error) {
	for {
		r := fuchsia.Sys_socket_read(f.h, 0, uint(len(data)), unsafe.Pointer(&data[0]))
		if r > 0 {
			return r, nil
		}
		if fuchsia.Status(r) == fuchsia.ErrShouldWait {
			var pending fuchsia.SignalsState
			status := fuchsia.Sys_handle_wait_one(f.h, fuchsia.SignalReadable|fuchsia.SignalPeerClosed, fuchsia.TimensecInfinite, &pending)
			if status != 0 {
				return 0, status
			}
			if pending.Satisfied&fuchsia.SignalReadable != 0 {
				continue
			}
			if pending.Satisfied&fuchsia.SignalPeerClosed != 0 {
				return 0, fuchsia.ErrRemoteClosed
			}
			// impossible
			return 0, fuchsia.ErrInternal
		}
	}
}

func (f *Pipe) Write(data []byte) (int, error) {
	for {
		r := fuchsia.Sys_socket_write(f.h, 0, uint(len(data)), unsafe.Pointer(&data[0]))
		if r > 0 {
			return r, nil
		}
		if fuchsia.Status(r) == fuchsia.ErrShouldWait {
			var pending fuchsia.SignalsState
			status := fuchsia.Sys_handle_wait_one(f.h, fuchsia.SignalWritable|fuchsia.SignalPeerClosed, fuchsia.TimensecInfinite, &pending)
			if status != 0 {
				return 0, status
			}
			if pending.Satisfied&fuchsia.SignalWritable != 0 {
				continue
			}
			if pending.Satisfied&fuchsia.SignalPeerClosed != 0 {
				return 0, fuchsia.ErrRemoteClosed
			}
			// impossible
			return 0, fuchsia.ErrInternal
		}
		return 0, fuchsia.Status(r)
	}
}

func (f *Pipe) ReadAt(data []byte, off int64) (int, error) {
	return 0, fuchsia.ErrNotSupported
}

func (f *Pipe) WriteAt(data []byte, off int64) (int, error) {
	return 0, fuchsia.ErrNotSupported
}

func (f *Pipe) Seek(offset int64, whence int) (int64, error) {
	return 0, fuchsia.ErrNotSupported
}

func (f *Pipe) Close() error {
	if err := fuchsia.HandleClose(f.h); err != nil {
		return err
	}
	f.h = 0
	return nil
}

func (f *Pipe) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	return nil, fuchsia.ErrNotSupported
}

func (f *Pipe) Clone() ([]fuchsia.Handle, error) {
	h := fuchsia.Sys_handle_duplicate(f.h, fuchsia.RightSameRights)
	if h < 0 {
		return nil, fuchsia.Status(h)
	}
	return []fuchsia.Handle{h}, nil
}

func (f *Pipe) Ioctl(op uint32, in []byte) ([]byte, []fuchsia.Handle, error) {
	return nil, nil, fuchsia.ErrNotSupported
}

func (f *Pipe) Misc(op uint32, in, out []byte) (n int, err error) {
	return 0, fuchsia.ErrNotSupported
}
