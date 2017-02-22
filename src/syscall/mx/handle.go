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

type HandleInfo uint32

func NewHandleInfo(t, arg uint32) HandleInfo {
	return HandleInfo((t & 0xFFFF) | ((arg & 0xFFFF) << 16))
}

func (hi HandleInfo) Type() uint32 {
	return uint32(hi) & 0xFFFF
}

func (hi HandleInfo) Arg() uint32 {
	return (uint32(hi) >> 16) & 0xFFFF
}

type Socket Handle

func (s Socket) Read(data []byte, flags uint32) (n int, err error) {
	var actual uint
	status := sys_socket_read(Handle(s), 0, unsafe.Pointer(&data[0]), uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), Error{Status: status, Text: "mx.Socket.Read"}
	}
	return int(actual), nil
}

func (s Socket) Write(data []byte, flags uint32) (n int, err error) {
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
	}
	var actual uint
	status := sys_socket_write(Handle(s), flags, p, uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), Error{Status: status, Text: "mx.Socket.Write"}
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
		return 0, 0, Error{Status: status, Text: "mx.Socket"}
	}
	return Socket(h0), Socket(h1), nil
}

func (h Handle) Close() error {
	if status := sys_handle_close(h); status != ErrOk {
		return Error{Status: status, Text: "mx.Handle.Close"}
	}
	return nil
}

func (h Handle) Duplicate(rights Rights) (Handle, error) {
	var dup Handle
	if status := sys_handle_duplicate(h, rights, &dup); status != ErrOk {
		return 0, Error{Status: status, Text: "mx.Handle.Duplicate"}
	}
	return dup, nil
}

func (h Handle) WaitOne(waitFor Signals, timeout Time) (observed Signals, err error) {
	if status := sys_object_wait_one(h, waitFor, timeout, &observed); status != ErrOk {
		return observed, Error{Status: status, Text: "mx.Handle.WaitOne"}
	}
	return observed, nil
}

func (h Handle) SignalPeer(clearMask, setMask Signals) error {
	if status := Sys_object_signal_peer(h, uint32(clearMask), uint32(setMask)); status != ErrOk {
		return Error{Status: status, Text: "mx.Handle.SignalPeer"}
	}
	return nil
}

func WaitMany(items []WaitItem, timeout Time) error {
	if status := sys_object_wait_many(&items[0], uint32(len(items)), timeout); status != ErrOk {
		return Error{Status: status, Text: "mx.Handle.WaitMany"}
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
		err = Error{Status: status, Text: "mx.Channel.Read"}
	}
	return numBytes, numHandles, err
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
		return Error{Status: status, Text: "mx.Channel.Write"}
	}
	return nil
}

func (c *Channel) Call(flags uint32, timeout Time, wData []byte, wHandles []Handle, rData []byte, rHandles []Handle) (actualBytes, actualHandles uint32, actualStatus Status, err error) {
	var writeDataPtr uintptr
	if len(wData) > 0 {
		writeDataPtr = uintptr(unsafe.Pointer(&wData[0]))
	}
	var writeHandlePtr *Handle
	if len(wHandles) > 0 {
		writeHandlePtr = &wHandles[0]
	}
	var readDataPtr uintptr
	if len(rData) > 0 {
		readDataPtr = uintptr(unsafe.Pointer(&rData[0]))
	}
	var readHandlePtr *Handle
	if len(rHandles) > 0 {
		readHandlePtr = &rHandles[0]
	}
	args := &ChannelCallArgs{
		WriteBytes:      writeDataPtr,
		WriteHandles:    writeHandlePtr,
		ReadBytes:       readDataPtr,
		ReadHandles:     readHandlePtr,
		WriteNumBytes:   uint32(len(wData)),
		WriteNumHandles: uint32(len(wHandles)),
		ReadNumBytes:    uint32(len(rData)),
		ReadNumHandles:  uint32(len(rHandles)),
	}

	status := Sys_channel_call(c.Handle, flags, timeout, args, &actualBytes, &actualHandles, &actualStatus)
	if status != ErrOk {
		err = Error{Status: status, Text: "mx.Channel.Call"}
	}
	return actualBytes, actualHandles, actualStatus, err
}

func NewChannel(flags uint32) (c0, c1 *Channel, err error) {
	c0, c1 = new(Channel), new(Channel)
	if status := sys_channel_create(flags, &c0.Handle, &c1.Handle); status != ErrOk {
		return nil, nil, Error{Status: status, Text: "mx.Channel"}
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
		return Error{Status: status, Text: "mx.Port.Queue"}
	}
	return nil
}

func (p *Port) Wait(packet []byte, timeout Time) error {
	if status := Sys_port_wait(p.Handle, timeout, unsafe.Pointer(&packet[0]), uint(len(packet))); status != ErrOk {
		return Error{Status: status, Text: "mx.Port.Wait"}
	}
	return nil
}

func (p *Port) Bind(key uint64, source Handle, signals Signals) error {
	if status := Sys_port_bind(p.Handle, key, source, signals); status != ErrOk {
		return Error{Status: status, Text: "mx.Port.Bind"}
	}
	return nil
}

func NewPort(options uint32) (*Port, error) {
	p := new(Port)
	if status := Sys_port_create(options, &p.Handle); status != ErrOk {
		return nil, Error{Status: status, Text: "mx.Port"}
	}
	return p, nil
}

type VMO Handle

func (vmo VMO) Read(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_read(Handle(vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), Error{Status: status, Text: "mx.VMO.Read"}
	}
	return int(actual), nil
}

func (vmo VMO) Write(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_write(Handle(vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), Error{Status: status, Text: "mx.VMO.Write"}
	}
	return int(actual), nil
}

func (vmo VMO) Size() (uint64, error) {
	var size uint64
	if status := sys_vmo_get_size(Handle(vmo), &size); status != ErrOk {
		return size, Error{Status: status, Text: "mx.VMO.Size"}
	}
	return size, nil
}

func (vmo VMO) SetSize(size uint64) error {
	if status := sys_vmo_set_size(Handle(vmo), size); status != ErrOk {
		return Error{Status: status, Text: "mx.VMO.SetSize"}
	}
	return nil
}

func (vmo VMO) OpRange(op uint32, offset, size uint64, b []byte) error {
	if status := sys_vmo_op_range(Handle(vmo), op, offset, size, unsafe.Pointer(&b[0]), uint(len(b))); status != ErrOk {
		return Error{Status: status, Text: "mx.VMO.OpRange"}
	}
	return nil
}

func (vmo VMO) Close() error {
	return Handle(vmo).Close()
}

func NewVMO(size uint64, options uint32) (VMO, error) {
	var h Handle
	if status := sys_vmo_create(size, options, &h); status != ErrOk {
		return 0, Error{Status: status, Text: "mx.VMO"}
	}
	return VMO(h), nil
}

type Log struct {
	Handle Handle
}

func NewLog(options uint32) *Log {
	log := &Log{}
	status := Sys_log_create(options, &log.Handle)
	if status != ErrOk {
		return nil
	}
	return log
}

func (l *Log) Write(b []byte) (n int, err error) {
	status := Sys_log_write(l.Handle, uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, Error{Status: status, Text: "mx.Log.Write"}
	}
	return len(b), nil
}

func (l *Log) Read(b []byte) (n int, err error) {
	status := Sys_log_read(l.Handle, uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, Error{Status: status, Text: "mx.Log.Read"}
	}
	return len(b), nil
}

func (l *Log) Close() (err error) {
	return l.Handle.Close()
}

func RandRead(b []byte) (n int, err error) {
	var actual uint
	if status := sys_cprng_draw(unsafe.Pointer(&b[0]), uint(len(b)), &actual); status != ErrOk {
		return int(actual), Error{Status: status, Text: "mx.RandRead"}
	}
	return int(actual), nil
}

// Error is a Status with associated error text.
// It is used as a Go error.
type Error struct {
	Status Status
	Text   string
}

func (e Error) Error() string {
	if e.Text == "" {
		return "mx.Status: " + e.Status.String()
	}
	return e.Status.String() + ": " + e.Text
}
