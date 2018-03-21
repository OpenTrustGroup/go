// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package zx

import "unsafe"

// Process-wide FDIO handles.
var (
	StdioHandles     [3][3]Handle // 3 stdio files, each with up to 3 (fdio.MaxHandles) handles
	StdioHandleTypes [3]int
	CwdHandle        Handle
	ProcHandle       Handle
	VMARRoot         VMAR
	RootNSMap        map[string]Handle
)

const _MX_CPRNG_DRAW_MAX_LEN = 256

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

func (s *Socket) Handle() *Handle {
	return (*Handle)(s)
}

func (s *Socket) Read(data []byte, flags uint32) (n int, err error) {
	var actual uint
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
	}
	status := sys_socket_read(Handle(*s), flags, p, uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), Error{Status: status, Text: "zx.Socket.Read"}
	}
	return int(actual), nil
}

func (s *Socket) Write(data []byte, flags uint32) (n int, err error) {
	var p unsafe.Pointer
	if len(data) > 0 {
		p = unsafe.Pointer(&data[0])
	}
	var actual uint
	status := sys_socket_write(Handle(*s), flags, p, uint(len(data)), &actual)
	if status != ErrOk {
		return int(actual), Error{Status: status, Text: "zx.Socket.Write"}
	}
	return int(actual), nil
}

func (s *Socket) Share(h Handle) error {
	if status := sys_socket_share(Handle(*s), h); status != ErrOk {
		return Error{Status: status, Text: "zx.Socket.Share"}
	}
	return nil
}

func (s *Socket) Accept(h *Handle) error {
	if status := sys_socket_accept(Handle(*s), h); status != ErrOk {
		return Error{Status: status, Text: "zx.Socket.Accept"}
	}
	return nil
}

func (s *Socket) Close() error {
	h := Handle(*s)
	err := h.Close()
	*s = Socket(HANDLE_INVALID)
	return err
}

func (s *Socket) WaitOne(waitFor Signals, timeout Time) (Signals, error) {
	h := Handle(*s)
	return h.WaitOne(waitFor, timeout)
}

func NewSocket(flags uint32) (s0, s1 Socket, err error) {
	var h0, h1 Handle
	if status := sys_socket_create(flags, &h0, &h1); status != ErrOk {
		return Socket(HANDLE_INVALID), Socket(HANDLE_INVALID), Error{Status: status, Text: "zx.Socket"}
	}
	return Socket(h0), Socket(h1), nil
}

type Event Handle

func NewEvent(options uint32) (e Event, err error) {
	var h Handle
	if status := Sys_event_create(options, &h); status != ErrOk {
		return Event(HANDLE_INVALID), Error{Status: status, Text: "zx.Event"}
	}
	return Event(h), nil
}

func (e *Event) Handle() *Handle {
	return (*Handle)(e)
}

func (e *Event) Close() error {
	h := Handle(*e)
	err := h.Close()
	*e = Event(HANDLE_INVALID)
	return err
}

func (e *Event) Duplicate(rights Rights) (Event, error) {
	h := Handle(*e)
	h, err := h.Duplicate(rights)
	if err != nil {
		return Event(HANDLE_INVALID), err
	}
	return Event(h), nil
}

func (h *Handle) Close() error {
	if status := sys_handle_close(*h); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.Close"}
	}
	*h = HANDLE_INVALID
	return nil
}

func (h *Handle) Duplicate(rights Rights) (Handle, error) {
	var dup Handle
	if status := sys_handle_duplicate(*h, rights, &dup); status != ErrOk {
		return HANDLE_INVALID, Error{Status: status, Text: "zx.Handle.Duplicate"}
	}
	return dup, nil
}

func (h *Handle) WaitOne(waitFor Signals, timeout Time) (observed Signals, err error) {
	if status := sys_object_wait_one(*h, waitFor, timeout, &observed); status != ErrOk {
		return observed, Error{Status: status, Text: "zx.Handle.WaitOne"}
	}
	return observed, nil
}

func (h *Handle) Signal(clearMask, setMask Signals) error {
	if status := Sys_object_signal(*h, uint32(clearMask), uint32(setMask)); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.Signal"}
	}
	return nil
}

func (h *Handle) SignalPeer(clearMask, setMask Signals) error {
	if status := Sys_object_signal_peer(*h, uint32(clearMask), uint32(setMask)); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.SignalPeer"}
	}
	return nil
}

func (h *Handle) SetCookie(scope Handle, cookie uint64) error {
	if status := Sys_object_set_cookie(*h, scope, cookie); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.SetCookie"}
	}
	return nil
}

func (h *Handle) GetCookie(scope Handle) (uint64, error) {
	var cookie uint64
	if status := Sys_object_get_cookie(*h, scope, &cookie); status != ErrOk {
		return 0, Error{Status: status, Text: "zx.Handle.GetCookie"}
	}
	return cookie, nil
}

func (h *Handle) GetProperty(property uint32, data []byte) error {
	var numBytes = uint(len(data))
	var dataPtr unsafe.Pointer
	if numBytes > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	if status := Sys_object_get_property(*h, property, dataPtr, numBytes); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.GetProperty"}
	}
	return nil
}

func (h *Handle) SetProperty(property uint32, data []byte) error {
	var numBytes = uint(len(data))
	var dataPtr unsafe.Pointer
	if numBytes > 0 {
		dataPtr = unsafe.Pointer(&data[0])
	}
	if status := Sys_object_set_property(*h, property, dataPtr, numBytes); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.SetProperty"}
	}
	return nil
}

func (h *Handle) IsValid() bool {
	return *h > HANDLE_INVALID
}

func WaitMany(items []WaitItem, timeout Time) error {
	if status := sys_object_wait_many(&items[0], uint32(len(items)), timeout); status != ErrOk {
		return Error{Status: status, Text: "zx.Handle.WaitMany"}
	}
	return nil
}

const (
	ChannelMaxMessageBytes = 65536
	ChannelMaxMessageHandles = 64
)

type Channel Handle

func (c *Channel) Handle() *Handle {
	return (*Handle)(c)
}

func (c *Channel) Close() error {
	h := Handle(*c)
	err := h.Close()
	*c = Channel(HANDLE_INVALID)
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
	if status := sys_channel_read(Handle(*c), flags, dataPtr, handlePtr, numBytes, numHandles, &numBytes, &numHandles); status != ErrOk {
		err = Error{Status: status, Text: "zx.Channel.Read"}
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
	status := sys_channel_write(Handle(*c), flags, dataPtr, uint32(len(data)), handlePtr, uint32(len(handles)))
	if status != ErrOk {
		return Error{Status: status, Text: "zx.Channel.Write"}
	}
	return nil
}

func (c *Channel) Call(flags uint32, deadline Time, wData []byte, wHandles []Handle, rData []byte, rHandles []Handle) (actualBytes, actualHandles uint32, actualStatus Status, err error) {
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

	status := Sys_channel_call(Handle(*c), flags, deadline, args, &actualBytes, &actualHandles, &actualStatus)
	if status != ErrOk {
		err = Error{Status: status, Text: "zx.Channel.Call"}
	}
	return actualBytes, actualHandles, actualStatus, err
}

func NewChannel(flags uint32) (c0, c1 Channel, err error) {
	var h0, h1 Handle
	if status := sys_channel_create(flags, &h0, &h1); status != ErrOk {
		return Channel(HANDLE_INVALID), Channel(HANDLE_INVALID), Error{Status: status, Text: "zx.Channel"}
	}
	return Channel(h0), Channel(h1), nil
}

const (
	PortWaitAsyncOnce = iota
	PortWaitAsyncRepeating
)

const (
	PortPacketTypeUser = iota
	PortPacketTypeSignalOne
	PortPacketTypeSignalRepeating
)

const (
	PacketHeaderSize = uint(unsafe.Sizeof(PacketHeader{}))
	PacketMaxSize    = uint(unsafe.Sizeof(Packet{}))
)

type PacketHeader struct {
	Key   uint64
	Type  uint32
	Extra uint32
}

// Packet describes the unit of communication via an FDIO protocol
type Packet struct {
	Hdr   PacketHeader
	Bytes [32]uint8
}

func (p *Packet) Signal() *PacketSignal {
	return (*PacketSignal)(unsafe.Pointer(p))
}

type PortPacket interface {
	Header() *PacketHeader
	Length() int
}

// PacketSignal describes packets of type PortPacketTypeSignal
type PacketSignal struct {
	Hdr      PacketHeader
	Trigger  Signals
	Observed Signals
	Count    uint64
}

func (s *PacketSignal) Header() *PacketHeader {
	return &s.Hdr
}

func (s *PacketSignal) Length() int {
	return int(unsafe.Sizeof(*s))
}

type Port Handle

func (p *Port) Handle() *Handle {
	return (*Handle)(p)
}

func (p *Port) Close() error {
	h := Handle(*p)
	err := h.Close()
	*p = Port(HANDLE_INVALID)
	return err
}

// Queue a PortPacket, which may have a varying size.
func (p *Port) Queue(packet PortPacket) error {
	if status := Sys_port_queue(Handle(*p), (*int)(unsafe.Pointer(packet.Header())), uint(packet.Length())); status != ErrOk {
		return Error{Status: status, Text: "zx.Port.Queue"}
	}
	return nil
}

// Wait for a Packet, which must be as large as the largest
// possible packet that may be received.
func (p *Port) Wait(packet *Packet, deadline Time) error {
	if status := Sys_port_wait(Handle(*p), deadline, (*int)(unsafe.Pointer(packet)), 0); status != ErrOk {
		return Error{Status: status, Text: "zx.Port.Wait"}
	}
	return nil
}

func (p *Port) Cancel(source Handle, key uint64) error {
	if status := Sys_port_cancel(Handle(*p), source, key); status != ErrOk {
		return Error{Status: status, Text: "zx.Port.Cancel"}
	}
	return nil
}

func (p *Port) WaitAsync(handle Handle, key uint64, signals Signals, options uint32) error {
	if status := Sys_object_wait_async(handle, Handle(*p), key, signals, options); status != ErrOk {
		return Error{Status: status, Text: "fdio.Port.WaitAsync"}
	}
	return nil
}

func NewPort(options uint32) (Port, error) {
	var h Handle
	if status := Sys_port_create(options, &h); status != ErrOk {
		return Port(HANDLE_INVALID), Error{Status: status, Text: "zx.Port"}
	}
	return Port(h), nil
}

type VMO Handle

func (vmo *VMO) Handle() *Handle {
	return (*Handle)(vmo)
}

func (vmo *VMO) Clone(options uint32, offset, size uint64) (VMO, error) {
	var h Handle
	if status := Sys_vmo_clone(Handle(*vmo), options, offset, size, &h); status != ErrOk {
		return VMO(HANDLE_INVALID), Error{Status: status, Text: "zx.VMO.Clone"}
	}
	return VMO(h), nil
}

func (vmo *VMO) Read(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_read_old(Handle(*vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), Error{Status: status, Text: "zx.VMO.Read"}
	}
	return int(actual), nil
}

func (vmo *VMO) Write(b []byte, offset uint64) (n int, err error) {
	var actual uint
	if status := sys_vmo_write_old(Handle(*vmo), unsafe.Pointer(&b[0]), offset, uint(len(b)), &actual); status != ErrOk {
		return int(actual), Error{Status: status, Text: "zx.VMO.Write"}
	}
	return int(actual), nil
}

func (vmo *VMO) Size() (uint64, error) {
	var size uint64
	if status := sys_vmo_get_size(Handle(*vmo), &size); status != ErrOk {
		return size, Error{Status: status, Text: "zx.VMO.Size"}
	}
	return size, nil
}

func (vmo *VMO) SetSize(size uint64) error {
	if status := sys_vmo_set_size(Handle(*vmo), size); status != ErrOk {
		return Error{Status: status, Text: "zx.VMO.SetSize"}
	}
	return nil
}

func (vmo *VMO) OpRange(op uint32, offset, size uint64, b []byte) error {
	if status := sys_vmo_op_range(Handle(*vmo), op, offset, size, unsafe.Pointer(&b[0]), uint(len(b))); status != ErrOk {
		return Error{Status: status, Text: "zx.VMO.OpRange"}
	}
	return nil
}

func (vmo *VMO) Close() error {
	h := Handle(*vmo)
	err := h.Close()
	*vmo = VMO(HANDLE_INVALID)
	return err
}

func NewVMO(size uint64, options uint32) (VMO, error) {
	var h Handle
	if status := sys_vmo_create(size, options, &h); status != ErrOk {
		return VMO(HANDLE_INVALID), Error{Status: status, Text: "zx.VMO"}
	}
	return VMO(h), nil
}

type VMAR Handle

func (v VMAR) Destroy() error {
	if status := sys_vmar_destroy(Handle(v)); status != ErrOk {
		return Error{Status: status, Text: "zx.VMAR.Destroy"}
	}
	return nil
}

func (v VMAR) Map(vmarOffset uint64, vmo VMO, vmoOffset, len uint64, flags VMFlag) (addr uintptr, err error) {
	status := sys_vmar_map(Handle(v), uint(vmarOffset), Handle(vmo), vmoOffset, uint(len), uint32(flags), &addr)
	if status != ErrOk {
		return 0, Error{Status: status, Text: "zx.VMAR.Map"}
	}
	return addr, nil
}

func (v VMAR) Unmap(addr uintptr, len uint64) error {
	status := sys_vmar_unmap(Handle(v), addr, uint(len))
	if status != ErrOk {
		return Error{Status: status, Text: "zx.VMAR.Unmap"}
	}
	return nil
}

func (v VMAR) Protect(addr uintptr, len uint64, flags VMFlag) error {
	status := sys_vmar_protect(Handle(v), addr, uint(len), uint32(flags))
	if status != ErrOk {
		return Error{Status: status, Text: "zx.VMAR.Protect"}
	}
	return nil
}

func NewVMAR(parent VMAR, offset, size uint64, flags VMFlag) (child VMAR, addr uintptr, err error) {
	var childHandle Handle
	status := sys_vmar_allocate(Handle(parent), uint(offset), uint(size), uint32(flags), &childHandle, &addr)
	if status != ErrOk {
		return 0, 0, Error{Status: status, Text: "zx.NewVMAR"}
	}
	return VMAR(childHandle), addr, nil
}

type Log Handle

func NewLog(options uint32) Log {
	var h Handle
	status := Sys_log_create(options, &h)
	if status != ErrOk {
		return Log(HANDLE_INVALID)
	}
	return Log(h)
}

func (l *Log) Handle() *Handle {
	return (*Handle)(l)
}

func (l Log) Write(b []byte) (n int, err error) {
	status := Sys_log_write(Handle(l), uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, Error{Status: status, Text: "zx.Log.Write"}
	}
	return len(b), nil
}

func (l Log) Read(b []byte) (n int, err error) {
	status := Sys_log_read(Handle(l), uint32(len(b)), unsafe.Pointer(&b[0]), 0)
	if status != ErrOk {
		return 0, Error{Status: status, Text: "zx.Log.Read"}
	}
	return len(b), nil
}

func (l *Log) Close() (err error) {
	h := Handle(*l)
	err = h.Close()
	*l = Log(HANDLE_INVALID)
	return err
}

func RandRead(b []byte) (n int, err error) {
	var total uint

	toRead := _MX_CPRNG_DRAW_MAX_LEN
	for start := 0; start < len(b); start += _MX_CPRNG_DRAW_MAX_LEN {
		if start+toRead > len(b) {
			toRead = len(b) - start
		}
		var actual uint
		if status := sys_cprng_draw(unsafe.Pointer(&b[start]), uint(toRead), &actual); status != ErrOk {
			total += actual
			return int(total), Error{Status: status, Text: "zx.RandRead"}
		}
		total += actual
	}

	return int(total), nil
}

// Error is a Status with associated error text.
// It is used as a Go error.
type Error struct {
	Status Status
	Text   string
}

func (e Error) Error() string {
	if e.Text == "" {
		return "zx.Status: " + e.Status.String()
	}
	return e.Status.String() + ": " + e.Text
}
