// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mx

import "unsafe"

// Process-wide MXIO handles.
var (
	RootHandle       Handle
	StdioHandles     [3][3]Handle // 3 stdio files, each with up to 3 (mxio.MaxHandles) handles
	StdioHandleTypes [3]int
	CwdHandle        Handle
	ProcHandle       Handle
	VmarRootHandle   Handle
)

type Socket Handle

func (s Socket) Read(data []byte, flags uint32) (n int, err error) {
	var actual uint
	status := sys_socket_read(Handle(s), 0, unsafe.Pointer(&data[0]), uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (s Socket) Write(data []byte, flags uint32) (n int, err error) {
	var actual uint
	status := sys_socket_write(Handle(s), 0, unsafe.Pointer(&data[0]), uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (s Socket) Close() error { return Handle(s).Close() }
func (s Socket) WaitOne(waitFor Signals, timeout Time) (Signals, error) {
	return Handle(s).WaitOne(waitFor, timeout)
}

func NewSocket(flags uint32) (s0, s1 Socket, err error) {
	var h0, h1 Handle
	if status := sys_socket_create(flags, &h0, &h1); status != ErrOk {
		return 0, 0, status
	}
	return Socket(h0), Socket(h1), nil
}

func (h Handle) Close() error {
	if status := sys_handle_close(h); status != ErrOk {
		return status
	}
	return nil
}

func (h Handle) Duplicate(rights Rights) (Handle, error) {
	var dup Handle
	if status := sys_handle_duplicate(h, rights, &dup); status != ErrOk {
		return 0, status
	}
	return dup, nil
}

func (h Handle) WaitOne(waitFor Signals, timeout Time) (observed Signals, err error) {
	if status := sys_handle_wait_one(h, waitFor, timeout, &observed); status != ErrOk {
		return observed, status
	}
	return observed, nil
}

func WaitMany(items []WaitItem, timeout Time) error {
	if status := sys_handle_wait_many(&items[0], uint32(len(items)), timeout); status != ErrOk {
		return status
	}
	return nil
}

type Channel struct {
	Handle Handle
}

func (s *Channel) Close() error {
	err := s.Handle.Close()
	s.Handle = 0
	return err
}

func (c *Channel) Read(data []byte, handles []Handle, flags uint32) (numBytes, numHandles uint32, err error) {
	numBytes = uint32(len(data))
	numHandles = uint32(len(handles))
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	if status := sys_channel_read(c.Handle, flags, dataPtr, numBytes, &numBytes, handlePtr, numHandles, &numHandles); status != ErrOk {
		err = status
	}
	return
}

func (c *Channel) Write(data []byte, handles []Handle, flags uint32) error {
	var dataPtr unsafe.Pointer
	if len(data) > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	var handlePtr *Handle
	if len(handles) > 0 {
		handlePtr = &handles[0]
	}
	status := sys_channel_write(c.Handle, flags, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)))
	if status != ErrOk {
		return status
	}
	return nil
}

func NewChannel(flags uint32) (c0, c1 *Channel, err error) {
	c0, c1 = new(Channel), new(Channel)
	if status := sys_channel_create(flags, &c0.Handle, &c1.Handle); status != ErrOk {
		return nil, nil, status
	}
	return c0, c1, nil
}

type Port struct {
	Handle Handle
}

func (p *Port) Close() error {
	err := p.Handle.Close()
	p.Handle = 0
	return err
}

func (p *Port) Queue(packet []byte) error {
	if status := Sys_port_queue(p.Handle, unsafe.Pointer(&packet[0]), uint(len(packet))); status != ErrOk {
		return status
	}
	return nil
}

func (p *Port) Wait(packet []byte, timeout Time) error {
	if status := Sys_port_wait(p.Handle, timeout, unsafe.Pointer(&packet[0]), uint(len(packet))); status != ErrOk {
		return status
	}
	return nil
}

func (p *Port) Bind(key uint64, source Handle, signals Signals) error {
	if status := Sys_port_bind(p.Handle, key, source, signals); status != ErrOk {
		return status
	}
	return nil
}

func NewPort(options uint32) (*Port, error) {
	p := new(Port)
	if status := Sys_port_create(options, &p.Handle); status != ErrOk {
		return nil, status
	}
	return p, nil
}

type VMO Handle

func (vmo VMO) Read(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_read(Handle(vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (vmo VMO) Write(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_write(Handle(vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}

func (vmo VMO) Size() (uint64, error) {
	var size uint64
	if status := sys_vmo_get_size(Handle(vmo), &size); status != ErrOk {
		return size, status
	}
	return size, nil
}

func (vmo VMO) SetSize(size uint64) error {
	if status := sys_vmo_set_size(Handle(vmo), size); status != ErrOk {
		return status
	}
	return nil
}

func (vmo VMO) OpRange(op uint32, offset, size uint64, b []byte) error {
	if status := sys_vmo_op_range(Handle(vmo), op, offset, size, unsafe.Pointer(&b[0]), uint(len(b))); status != ErrOk {
		return status
	}
	return nil
}

func (vmo VMO) Close() error {
	return Handle(vmo).Close()
}

func NewVMO(size uint64, options uint32) (VMO, error) {
	var h Handle
	if status := sys_vmo_create(size, options, &h); status != ErrOk {
		return 0, status
	}
	return VMO(h), nil
}

type Log struct {
	Handle Handle
}

func NewLog(options uint32) *Log {
	return &Log{Sys_log_create(options)}
}

func (l *Log) Write(b []byte) (n int, err error) {
	status := Sys_log_write(l.Handle, uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, status
	}
	return len(b), nil
}

func (l *Log) Read(b []byte) (n int, err error) {
	status := Sys_log_read(l.Handle, uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, status
	}
	return len(b), nil
}

func RandRead(b []byte) (n int, err error) {
	var actual uint
	if status := sys_cprng_draw(unsafe.Pointer(&b[0]), uint(len(b)), &actual); status != ErrOk {
		return int(actual), status
	}
	return int(actual), nil
}
