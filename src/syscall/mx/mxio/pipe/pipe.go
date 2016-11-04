// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package pipe

import (
	"syscall/mx"
	"syscall/mx/mxio"
)

type Pipe struct {
	h mx.Handle
}

func NewPipe() (*Pipe, *Pipe, error) {
	h0, h1, err := mx.NewSocket(0)
	if err != nil {
		return nil, nil, err
	}
	return &Pipe{h0}, &Pipe{h1}, nil
}

func (f *Pipe) Read(data []byte) (n int, err error) {
	for {
		n, err = f.h.ReadSocket(data, 0)
		if err == nil {
			return n, nil
		}
		if err == mx.ErrShouldWait {
			pending, err := f.h.WaitOne(mx.SignalReadable|mx.SignalPeerClosed, mx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if pending&mx.SignalReadable != 0 {
				continue
			}
			if pending&mx.SignalPeerClosed != 0 {
				return 0, mx.ErrRemoteClosed
			}
			// impossible
			return 0, mx.ErrInternal
		}
		return 0, err
	}
}

func (f *Pipe) Write(data []byte) (n int, err error) {
	for {
		n, err = f.h.WriteSocket(data, 0)
		if err == nil {
			return n, nil
		}
		if err == mx.ErrShouldWait {
			pending, err := f.h.WaitOne(mx.SignalWritable|mx.SignalPeerClosed, mx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if pending&mx.SignalWritable != 0 {
				continue
			}
			if pending&mx.SignalPeerClosed != 0 {
				return 0, mx.ErrRemoteClosed
			}
			// impossible
			return 0, mx.ErrInternal
		}
		return 0, err
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
	if err := f.h.Close(); err != nil {
		return err
	}
	f.h = 0
	return nil
}

func (f *Pipe) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	return nil, mx.ErrNotSupported
}

func (f *Pipe) Clone() ([]mx.Handle, error) {
	h, err := f.h.Duplicate(mx.RightSameRights)
	if err != nil {
		return nil, err
	}
	return []mx.Handle{h}, nil
}

func (f *Pipe) Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error) {
	return nil, nil, mx.ErrNotSupported
}

func (f *Pipe) Misc(op uint32, off int64, in, out []byte) (n int, err error) {
	return 0, mx.ErrNotSupported
}
