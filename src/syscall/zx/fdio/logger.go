// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"syscall/zx"
)

type Logger struct {
	log            zx.Log
	logBufferArray [1024]byte
	logBufferN     int
}

func NewLogger(h zx.Handle) (*Logger, error) {
	return &Logger{
		log: zx.Log{Handle: h},
	}, nil
}

func (f *Logger) Read(data []byte) (n int, err error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) Write(data []byte) (n int, err error) {
	// Sys_log_write adds a newline to output buffers. To prevent fragmented logs, we buffer
	// output until a newline character is encountered.
	f.logBufferN += copy(f.logBufferArray[f.logBufferN:], data)
	if f.logBufferArray[f.logBufferN-1] == '\n' {
		n, err := f.log.Write(f.logBufferArray[:f.logBufferN])
		// Flush buffer, regardless of status returned from write
		f.logBufferN = 0
		if err != nil {
			return len(data), nil
		}
		return n, err
	}
	return len(data), nil
}

func (f *Logger) ReadAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) Seek(offset int64, whence int) (int64, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) Close() error {
	if err := f.log.Close(); err != nil {
		return err
	}
	return nil
}

func (f *Logger) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) Clone() ([]zx.Handle, error) {
	h, err := zx.Handle(f.log.Handle).Duplicate(zx.RightSameRights)
	if err != nil {
		return nil, err
	}
	return []zx.Handle{h}, nil
}

func (f *Logger) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}
func (f *Logger) IoctlSetHandle(op uint32, in zx.Handle) error {
	return zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}

func (f *Logger) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "zx.Logger"}
}
