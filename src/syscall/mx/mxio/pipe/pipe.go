// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package pipe

import (
	"unsafe"

	"syscall/mx"
	"syscall/mx/mxio"
)

type Pipe struct {
	h mx.Handle
}

func NewPipe() (*Pipe, *Pipe, error) {
	var h [2]mx.Handle
	status := mx.Sys_socket_create(&h[0], 0)
	if status < 0 {
		return nil, nil, status
	}
	return &Pipe{h[0]}, &Pipe{h[1]}, nil
}

func (f *Pipe) Read(data []byte) (n int, err error) {
	for {
		r := mx.Sys_socket_read(f.h, 0, uint(len(data)), unsafe.Pointer(&data[0]))
		if r > 0 {
			return r, nil
		}
		if mx.Status(r) == mx.ErrShouldWait {
			var pending mx.SignalsState
			status := mx.Sys_handle_wait_one(f.h, mx.SignalReadable|mx.SignalPeerClosed, mx.TimensecInfinite, &pending)
			if status != 0 {
				return 0, status
			}
			if pending.Satisfied&mx.SignalReadable != 0 {
				continue
			}
			if pending.Satisfied&mx.SignalPeerClosed != 0 {
				return 0, mx.ErrRemoteClosed
			}
			// impossible
			return 0, mx.ErrInternal
		}
	}
}

func (f *Pipe) Write(data []byte) (int, error) {
	for {
		r := mx.Sys_socket_write(f.h, 0, uint(len(data)), unsafe.Pointer(&data[0]))
		if r > 0 {
			return r, nil
		}
		if mx.Status(r) == mx.ErrShouldWait {
			var pending mx.SignalsState
			status := mx.Sys_handle_wait_one(f.h, mx.SignalWritable|mx.SignalPeerClosed, mx.TimensecInfinite, &pending)
			if status != 0 {
				return 0, status
			}
			if pending.Satisfied&mx.SignalWritable != 0 {
				continue
			}
			if pending.Satisfied&mx.SignalPeerClosed != 0 {
				return 0, mx.ErrRemoteClosed
			}
			// impossible
			return 0, mx.ErrInternal
		}
		return 0, mx.Status(r)
	}
}

func (f *Pipe) ReadAt(data []byte, off int64) (int, error) {
	return 0, mx.ErrNotSupported
}

func (f *Pipe) WriteAt(data []byte, off int64) (int, error) {
	return 0, mx.ErrNotSupported
}

func (f *Pipe) Seek(offset int64, whence int) (int64, error) {
	return 0, mx.ErrNotSupported
}

func (f *Pipe) Close() error {
	if err := mx.HandleClose(f.h); err != nil {
		return err
	}
	f.h = 0
	return nil
}

func (f *Pipe) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	return nil, mx.ErrNotSupported
}

func (f *Pipe) Clone() ([]mx.Handle, error) {
	h := mx.Sys_handle_duplicate(f.h, mx.RightSameRights)
	if h < 0 {
		return nil, mx.Status(h)
	}
	return []mx.Handle{h}, nil
}

func (f *Pipe) Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error) {
	return nil, nil, mx.ErrNotSupported
}

func (f *Pipe) Misc(op uint32, off int64, in, out []byte) (n int, err error) {
	return 0, mx.ErrNotSupported
}
