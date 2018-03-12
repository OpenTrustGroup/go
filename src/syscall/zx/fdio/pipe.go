// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"syscall/zx"
)

type Pipe struct {
	s zx.Socket
}

func NewPipes() (*Pipe, *Pipe, error) {
	s0, s1, err := zx.NewSocket(0)
	if err != nil {
		return nil, nil, err
	}
	return &Pipe{s0}, &Pipe{s1}, nil
}

func NewPipe(h []zx.Handle) (*Pipe, error) {
	return &Pipe{zx.Socket(h[0])}, nil
}

func (f *Pipe) Read(data []byte) (n int, err error) {
	for {
		n, err = f.s.Read(data, 0)
		if err == nil {
			return n, nil
		}
		mxerr, ok := err.(zx.Error)
		if !ok {
			return n, err
		}
		switch mxerr.Status {
		case zx.ErrPeerClosed:
			return 0, zx.EOF
		case zx.ErrShouldWait:
			pending, err := f.s.WaitOne(zx.SignalSocketReadable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if pending&zx.SignalSocketReadable != 0 {
				continue
			}
			if pending&zx.SignalSocketPeerClosed != 0 {
				return 0, zx.EOF
			}
			// impossible
			return 0, zx.Error{Status: zx.ErrInternal, Text: "fdio.Pipe"}
		}
		return 0, err
	}
}

func (f *Pipe) Write(data []byte) (n int, err error) {
	for {
		n, err = f.s.Write(data, 0)
		if err == nil {
			return n, nil
		}
		mxerr, ok := err.(zx.Error)
		if !ok {
			return n, err
		}
		switch mxerr.Status {
		case zx.ErrPeerClosed:
			return 0, zx.EOF
			return 0, zx.EPIPE
		case zx.ErrShouldWait:
			pending, err := f.s.WaitOne(zx.SignalSocketWritable|zx.SignalSocketPeerClosed, zx.TimensecInfinite)
			if err != nil {
				return 0, err
			}
			if pending&zx.SignalSocketWritable != 0 {
				continue
			}
			if pending&zx.SignalSocketPeerClosed != 0 {
				return 0, zx.EPIPE
			}
			// impossible
			return 0, zx.Error{Status: zx.ErrInternal, Text: "fdio.Pipe"}
		}
		return 0, err
	}
}

func (f *Pipe) ReadAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}

func (f *Pipe) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}

func (f *Pipe) Seek(offset int64, whence int) (int64, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}

func (f *Pipe) Close() error {
	if err := f.s.Close(); err != nil {
		return err
	}
	f.s = 0
	return nil
}

func (f *Pipe) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}

func (f *Pipe) Clone() ([]zx.Handle, error) {
	h, err := f.s.Handle().Duplicate(zx.RightSameRights)
	if err != nil {
		return nil, err
	}
	return []zx.Handle{h}, nil
}

func (f *Pipe) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}
func (f *Pipe) IoctlSetHandle(op uint32, in zx.Handle) error {
	return zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}

func (f *Pipe) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.Pipe"}
}
