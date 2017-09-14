// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"syscall/mx"
)

type Logger struct {
	log            mx.Log
	logBufferArray [1024]byte
	logBufferN     int
}

func NewLogger(h mx.Handle) (*Logger, error) {
	return &Logger{
		log: mx.Log{Handle: h},
	}, nil
}

func (f *Logger) Read(data []byte) (n int, err error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
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
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}

func (f *Logger) WriteAt(data []byte, off int64) (int, error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}

func (f *Logger) Seek(offset int64, whence int) (int64, error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}

func (f *Logger) Close() error {
	if err := f.log.Close(); err != nil {
		return err
	}
	return nil
}

func (f *Logger) Open(path string, flags int32, mode uint32) (FDIO, error) {
	return nil, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}

func (f *Logger) Clone() ([]mx.Handle, error) {
	h, err := mx.Handle(f.log.Handle).Duplicate(mx.RightSameRights)
	if err != nil {
		return nil, err
	}
	return []mx.Handle{h}, nil
}

func (f *Logger) Ioctl(op uint32, in, out []byte) ([]mx.Handle, error) {
	return nil, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}
func (f *Logger) IoctlSetHandle(op uint32, in mx.Handle) error {
	return mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}

func (f *Logger) Misc(op uint32, off int64, in, out []byte, handles []mx.Handle) (n int, err error) {
	return 0, mx.Error{Status: mx.ErrNotSupported, Text: "mx.Logger"}
}
