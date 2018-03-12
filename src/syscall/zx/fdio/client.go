// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"sync/atomic"
	"syscall/zx"
	"unsafe"
)

// RemoteIO is an object which manages file-like operations on a remote object, accessible through a
// handle.
type RemoteIO struct {
	h    zx.Channel // Channel handle for the RPC
	e    zx.Handle  // Event handle for device state signals
	txid uint32     // Transaction ID used for synchronous remoteio calls
}

// New returns a new RemoteIO object
func NewRIO(handle []zx.Handle) (*RemoteIO, error) {
	r := &RemoteIO{}
	if len(handle) > 0 {
		r.h = zx.Channel(handle[0])
	}
	if len(handle) > 1 {
		r.e = handle[1]
	}
	return r, nil
}

func NewRIOFromChannel(h zx.Channel) *RemoteIO {
	return &RemoteIO{h: h}
}

// txn completes the client-side part of the RIO transaction.
// It wraps the raw 'Call Message' syscall with some error checking.
func (r *RemoteIO) txn(msg *Msg) (uint32, error) {
	if !msg.IsValid() {
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
	}

	msg.Txid = atomic.AddUint32(&r.txid, 1) - 1
	numBytes, _, rs, err := msg.CallMsg(r.h)
	if err != nil {
		if mxerr, hasStatus := err.(zx.Error); hasStatus && mxerr.Status == zx.ErrCallFailed {
			// Read failure: Real error in rs
			msg.Hcount = 0
			return 0, zx.Error{Status: rs, Text: "RemoteIO"}
		} else {
			// Write failure
			msg.DiscardHandles()
			return 0, err
		}
	}

	if !msg.IsValidIncoming(numBytes) || msg.Op() != OpStatus {
		msg.DiscardHandles() // If these handles were transmitted poorly, drop them
		return 0, zx.Error{Status: zx.ErrIO, Text: "RemoteIO"}
	}

	// Check for remote error
	status := zx.Status(msg.Arg)
	if status < 0 {
		msg.DiscardHandles() // Throw away any remaining handles; no one else will see them
		return 0, zx.Error{Status: status, Text: "RemoteIO"}
	}
	return uint32(status), nil
}

func (r *RemoteIO) Read(out []byte) (int, error) {
	return r.read(OpRead, out, -1)
}

func (r *RemoteIO) ReadAt(out []byte, off int64) (int, error) {
	return r.read(OpReadAt, out, off)
}

func (r *RemoteIO) read(op uint32, out []byte, off int64) (int, error) {
	var status zx.Status
	size := uint32(len(out))
	xferTotal := 0
	for size > 0 {
		xfer := uint32(MessageChunkSize) // Read to out in chunks
		if size < xfer {
			xfer = size
		}

		var msg Msg
		msg.SetOp(op)
		msg.Arg = int32(xfer)
		if op == OpReadAt {
			msg.SetOff(off)
		}

		r, err := r.txn(&msg)
		if err != nil {
			if mxe, ok := err.(zx.Error); ok {
				status = mxe.Status
			} else {
				status = zx.ErrIO
			}
			break
		}
		msg.DiscardHandles()

		if r > msg.Datalen || r > xfer {
			status = zx.ErrIO
			break
		}
		copy(out[xferTotal:], msg.Data[:r])
		xferTotal += int(r)
		size -= r
		if op == OpReadAt {
			off += int64(r)
		}

		// stop at short read
		if r < xfer {
			break
		}
	}

	if status != 0 {
		return xferTotal, zx.Error{Status: status, Text: "RemoteIO.read"}
	}
	return xferTotal, nil
}

func (r *RemoteIO) Write(p []byte) (int, error) {
	return r.write(OpWrite, p, -1)
}
func (r *RemoteIO) WriteAt(p []byte, off int64) (int, error) {
	return r.write(OpWriteAt, p, off)
}
func (r *RemoteIO) write(op uint32, p []byte, off int64) (int, error) {
	var status zx.Status
	xferTotal := 0
	for len(p) > 0 {
		xfer := uint32(MessageChunkSize) // Write p in chunks
		if uint32(len(p)) < xfer {
			xfer = uint32(len(p))
		}

		var msg Msg
		msg.SetOp(op)
		msg.Datalen = xfer
		if op == OpWriteAt {
			msg.SetOff(off)
		}
		copy(msg.Data[:xfer], p[:xfer])

		transferred, err := r.txn(&msg)
		if err != nil {
			if mxe, ok := err.(zx.Error); ok {
				status = mxe.Status
			} else {
				status = zx.ErrIO
			}
			break
		}
		msg.DiscardHandles() // We shouldn't be receiving handles
		if transferred > xfer {
			status = zx.ErrIO
			break
		}
		xferTotal += int(transferred)
		p = p[transferred:]
		if op == OpWriteAt {
			off += int64(transferred)
		}
		if transferred < xfer { // Short write
			status = zx.ErrIO
			break
		}
	}

	if len(p) != 0 {
		return xferTotal, zx.Error{Status: status, Text: "RemoteIO.write"}
	}
	return xferTotal, nil
}

func (r *RemoteIO) Seek(offset int64, whence int) (int64, error) {
	var msg Msg
	msg.SetOp(OpSeek)
	msg.SetOff(offset)
	msg.Arg = int32(whence)

	if _, err := r.txn(&msg); err != nil {
		return 0, err
	}
	msg.DiscardHandles()
	return msg.Off(), nil
}

func (r *RemoteIO) Close() error {
	var msg Msg
	msg.SetOp(OpClose)

	ret, err := r.txn(&msg)
	if ret >= 0 {
		msg.DiscardHandles()
	}

	r.h.Close()
	if r.e.IsValid() {
		r.e.Close()
	}

	if err != nil {
		return err
	}
	return nil
}

// GetObject requests an FDIO object remotely
func GetObject(rio zx.Channel, op uint32, name string, flags uint32, mode uint32) (zx.Channel, *RioDescription, error) {
	if len(name) > MessageChunkSize {
		return zx.Channel(zx.HANDLE_INVALID), nil, zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
	}
	var msg Msg
	msg.SetOp(op)
	msg.Datalen = uint32(len(name))
	msg.Arg = int32(flags)
	msg.SetMode(mode)
	copy(msg.Data[:], name)

	return replyChannelCall(rio, &msg)
}

func replyChannelCall(rio zx.Channel, msg *Msg) (zx.Channel, *RioDescription, error) {
	// Create a new channel to receive the 'callback' on
	h, rh, err := zx.NewChannel(0)
	if err != nil {
		return zx.Channel(zx.HANDLE_INVALID), nil, err
	}
	msg.Arg |= OpenFlagDescribe
	msg.Handle[0] = zx.Handle(rh)
	msg.Hcount = 1

	// Write the (one-way) OPEN or CLONE request message
	if err := msg.WriteMsg(rio); err != nil {
		msg.DiscardHandles()
		h.Close()
		return zx.Channel(zx.HANDLE_INVALID), nil, err
	}

	// Wait
	h.Handle().WaitOne(zx.SignalChannelReadable|zx.SignalChannelPeerClosed, zx.TimensecInfinite)

	// Attempt to read the callback response
	var ro RioDescription
	if status := ro.Read(zx.Handle(h), 0); status < 0 {
		return zx.Channel(zx.HANDLE_INVALID), nil, zx.Error{Status: status, Text: "RemoteIO.replyChannelCall"}
	} else {
		return h, &ro, nil
	}
}

func connect(svc zx.Handle, cnxn zx.Handle, op uint32, name string) error {
	if len(name) >= PathMax {
		return zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO.connect"}
	}

	msg := Msg{}
	msg.SetOp(op)
	msg.Datalen = uint32(len(name))
	msg.Arg = OpOpenFlagRDWR
	msg.SetMode(0755)
	msg.Hcount = 1
	msg.Handle[0] = cnxn
	copy(msg.Data[:], name)

	if err := msg.WriteMsg(zx.Channel(svc)); err != nil {
		cnxn.Close()
		return err
	}

	return nil
}

func ServiceConnect(svcpath string, h zx.Handle) error {
	// Otherwise attempt to connect through the root namespace.
	ns, err := NewNSFromMap(zx.RootNSMap)
	if err == nil {
		return ns.Connect(svcpath, h)
	}
	h.Close()
	return zx.Error{Status: zx.ErrNotFound, Text: "RemoteIO.ServiceConnect"}
}

func ServiceConnectAt(dir zx.Handle, path string, h zx.Handle) error {
	if !dir.IsValid() || !h.IsValid() {
		return zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO.ServiceConnectAt"}
	}
	return connect(dir, h, OpOpen, path)
}

func closeAll(handles []zx.Handle) {
	for i := range handles {
		handles[i].Close()
	}
}

// Consumes handles, wrapping them in fdio object or closing them on error.
func createFromHandles(handle zx.Handle, info ObjectInfo) (FDIO, error) {
	handles := append([]zx.Handle{handle}, info.GetHandles()...)
	switch info.Tag {
	case ProtocolService:
		return NewRIO(handles)
	case ProtocolFile:
		return NewRIO(handles)
	case ProtocolDirectory:
		return NewRIO(handles)
	case ProtocolPipe:
		return NewPipe(handles)
	case ProtocolVMOFile:
		handles[0].Close()
		vInfo := info.VMOFile()
		return NewVMOFile(handles[1], vInfo.Offset, vInfo.Length)
	case ProtocolDevice:
		return NewRIO(handles)
	case ProtocolSocket:
		return NewSocket(handles)
	default:
		println("Unknown fdio protocol, ", info.Tag)
		// In the error case, the server may have given us handles -- close them
		closeAll(handles)
		return nil, zx.Error{Status: zx.ErrNotSupported, Text: "RemoteIO"}
	}
}

func OpenHandle(rio zx.Channel, path string, flags uint32, mode uint32) (FDIO, error) {
	c, ro, err := GetObject(rio, OpOpen, path, flags, mode)
	if err != nil {
		return nil, err
	}
	return createFromHandles(zx.Handle(c), ro.Info)
}

func CloneHandle(rio zx.Channel) ([]zx.Handle, error) {
	c, _, err := GetObject(rio, OpClone, "", 0, 0)
	if err != nil {
		return nil, err
	}
	return []zx.Handle{zx.Handle(c)}, nil
}

func (r *RemoteIO) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	return OpenHandle(r.h, path, flags, mode)
}

func (r *RemoteIO) Clone() ([]zx.Handle, error) {
	return CloneHandle(r.h)
}

const handleSize = uint32(unsafe.Sizeof(zx.Handle(0)))

// TODO(smklein): Remove; this function is a subset of "Ioctl".
func (r *RemoteIO) IoctlSetHandle(op uint32, in zx.Handle) error {
	if IoctlKind(op) != IoctlKindSetHandle {
		return zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
	}
	var msg Msg
	msg.SetOp(OpIoctlOneHandle)
	msg.Handle[0] = in
	msg.Hcount = 1
	msg.Datalen = handleSize
	msg.SetIoctlOp(op)

	if _, err := r.txn(&msg); err != nil {
		return err
	}
	return nil
}

func (r *RemoteIO) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	var msg Msg
	if len(in) > IoctlMaxInput {
		return nil, zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
	}
	msg.SetOp(OpIoctl)
	msg.Datalen = uint32(len(in))
	switch IoctlKind(op) {
	case IoctlKindGetHandle:
		msg.Arg = int32(handleSize)
	case IoctlKindGetTwoHandles:
		msg.Arg = int32(2 * handleSize)
	case IoctlKindGetThreeHandles:
		msg.Arg = int32(3 * handleSize)
	case IoctlKindSetHandle:
		msg.SetOp(OpIoctlOneHandle)
		if len(in) < int(handleSize) {
			return nil, zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
		}
		msg.Handle[0] = zx.Handle(*(*zx.Handle)(unsafe.Pointer(&in[0])))
		msg.Hcount = 1
	}
	msg.Arg += int32(len(out))
	msg.SetIoctlOp(op)
	copy(msg.Data[:], in)

	_, err := r.txn(&msg)
	if err != nil {
		return nil, err
	}

	switch IoctlKind(op) {
	case IoctlKindGetHandle:
		if msg.Hcount < 1 {
			return nil, zx.Error{Status: zx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []zx.Handle{msg.Handle[0]}
		msg.Handle[0] = 0
		msg.DiscardHandles()
		if msg.Datalen > handleSize {
			copy(out, msg.Data[handleSize:msg.Datalen])
		}
		return outh, nil
	case IoctlKindGetTwoHandles:
		if msg.Hcount < 2 {
			msg.DiscardHandles()
			return nil, zx.Error{Status: zx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []zx.Handle{msg.Handle[0], msg.Handle[1]}
		msg.Handle[0] = 0
		msg.Handle[1] = 0
		msg.DiscardHandles()
		if msg.Datalen > 2*handleSize {
			copy(out, msg.Data[2*handleSize:msg.Datalen])
		}
		return outh, nil
	case IoctlKindGetThreeHandles:
		if msg.Hcount < 3 {
			msg.DiscardHandles()
			return nil, zx.Error{Status: zx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []zx.Handle{msg.Handle[0], msg.Handle[1], msg.Handle[2]}
		msg.Handle[0] = 0
		msg.Handle[1] = 0
		msg.Handle[2] = 0
		msg.DiscardHandles()
		if msg.Datalen > 3*handleSize {
			copy(out, msg.Data[3*handleSize:msg.Datalen])
		}
		return outh, nil
	default:
		msg.DiscardHandles()
		copy(out, msg.Data[:msg.Datalen])
		return nil, nil
	}
}

func (r *RemoteIO) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	if len(in) > MessageChunkSize || len(out) > MessageChunkSize {
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "RemoteIO"}
	}

	var msg Msg
	msg.SetOp(op)
	msg.Arg = int32(len(out))
	msg.Datalen = uint32(len(in))
	msg.SetOff(off)
	copy(msg.Data[:], in)

	if len(handles) > 0 {
		copy(msg.Handle[:], handles)
		msg.Hcount = uint32(len(handles))
	}
	if _, err = r.txn(&msg); err != nil {
		return 0, err
	}
	msg.DiscardHandles()

	if int(msg.Datalen) > len(out) {
		return 0, zx.Error{Status: zx.ErrIO, Text: "RemoteIO"}
	}
	n = copy(out, msg.Data[:msg.Datalen])
	return n, nil
}

func Stat(f FDIO) (attr Vnattr, err error) {
	b := (*[unsafe.Sizeof(attr)]byte)(unsafe.Pointer(&attr))
	if _, err := f.Misc(OpStat, 0, nil, (*b)[:], nil); err != nil {
		return Vnattr{}, err
	}
	return attr, nil
}

func UnlinkAt(dir FDIO, name string) error {
	_, err := dir.Misc(OpUnlink, 0, []byte(name), nil, nil)
	if err != nil {
		return err
	}
	return nil
}
