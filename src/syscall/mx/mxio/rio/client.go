// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package rio implements the Remote IO protocol, providing the MXIO interface remotely
package rio

import (
	"sync/atomic"
	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxio/pipe"
	"unsafe"
)

// RemoteIO is an object which manages file-like operations on a remote object, accessible through a
// handle.
type RemoteIO struct {
	h    *mx.Channel // Channel handle for the RPC
	e    mx.Handle   // Event handle for device state signals
	txid uint32      // Transaction ID used for synchronous remoteio calls
}

// New returns a new RemoteIO object
func New(handle []mx.Handle) (*RemoteIO, error) {
	r := &RemoteIO{}
	if len(handle) > 0 {
		r.h = &mx.Channel{handle[0]}
	}
	if len(handle) > 1 {
		r.e = handle[1]
	}
	return r, nil
}

// txn completes the client-side part of the RIO transaction.
// It wraps the raw 'Call Message' syscall with some error checking.
func (r *RemoteIO) txn(msg *Msg) (uint32, error) {
	if !msg.IsValid() {
		return 0, mx.ErrInvalidArgs
	}

	msg.Txid = atomic.AddUint32(&r.txid, 1) - 1
	numBytes, _, rs, err := msg.CallMsg(r.h)
	if err != nil {
		if err == mx.ErrCallFailed {
			// Read failure: Real error in rs
			msg.Hcount = 0
			return 0, rs
		} else {
			// Write failure
			msg.DiscardHandles()
			return 0, err
		}
	}

	if !msg.IsValidIncoming(numBytes) || msg.Op() != OpStatus {
		msg.DiscardHandles() // If these handles were transmitted poorly, drop them
		return 0, mx.ErrIO
	}

	// Check for remote error
	status := mx.Status(msg.Arg)
	if status < 0 {
		msg.DiscardHandles() // Throw away any remaining handles; no one else will see them
		return 0, status
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
	var status mx.Status
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
			break
		}
		msg.DiscardHandles()

		if r > msg.Datalen || r > xfer {
			status = mx.ErrIO
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
		return xferTotal, status
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
	var status mx.Status
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
			break
		}
		msg.DiscardHandles() // We shouldn't be receiving handles
		if transferred > xfer {
			status = mx.ErrIO
			break
		}
		xferTotal += int(transferred)
		p = p[transferred:]
		if op == OpWriteAt {
			off += int64(transferred)
		}
		if transferred < xfer { // Short write
			status = mx.ErrIO
			break
		}
	}

	if len(p) != 0 {
		return xferTotal, status
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
	r.h = nil
	if r.e > 0 {
		r.e.Close()
		r.e = 0
	}

	if err != nil {
		return err
	}
	return nil
}

// getObject requests an MXIO object remotely
func (r *RemoteIO) getObject(op uint32, name string, flags int32, mode uint32) (*RioObject, error) {
	if len(name) > MessageChunkSize {
		return nil, mx.ErrInvalidArgs
	}
	var msg Msg
	msg.SetOp(op)
	msg.Datalen = uint32(len(name))
	msg.Arg = flags
	msg.SetMode(mode)
	copy(msg.Data[:], name)

	return r.replyChannelCall(&msg)
}

func (r *RemoteIO) replyChannelCall(msg *Msg) (*RioObject, error) {
	// Create a new channel to receive the 'callback' on
	h, rh, err := mx.NewChannel(0)
	if err != nil {
		return nil, err
	}
	msg.Handle[0] = rh.Handle
	msg.Hcount = 1

	// Write the (one-way) OPEN or CLONE request message
	if err := msg.WriteMsg(r.h); err != nil {
		msg.DiscardHandles()
		h.Close()
		return nil, err
	}

	// Wait
	h.Handle.WaitOne(mx.SignalChannelReadable|mx.SignalChannelPeerClosed, mx.TimensecInfinite)

	// Attempt to read the callback response
	var ro RioObject
	if status := ro.Read(h.Handle, 0); status < 0 {
		return nil, status
	} else {
		return &ro, nil
	}
}

// Consumes handles, wrapping them in mxio object or closing them on error.
func createFromHandles(protocol mxio.Protocol, handles []mx.Handle, extra []byte) (mxio.MXIO, error) {
	switch protocol {
	case mxio.ProtocolPipe:
		return pipe.New(handles)
	case mxio.ProtocolRemote:
		return New(handles)
	case mxio.ProtocolVMOFile:
		return NewVMOFile(handles[0], extra)
	default:
		println("Unknown mxio protocol, ", protocol)
		// In the error case, the server may have given us handles -- close them
		for i := range handles {
			handles[i].Close()
		}
		return nil, mx.ErrNotSupported
	}
}

func (r *RemoteIO) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	ro, err := r.getObject(OpOpen, path, flags, mode)
	if err != nil {
		return nil, err
	}
	return createFromHandles(mxio.Protocol(ro.Type), ro.Handle[:ro.Hcount], ro.Extra[:ro.Esize])
}

func (r *RemoteIO) Clone() ([]mx.Handle, error) {
	ro, err := r.getObject(OpClone, "", 0, 0)
	if err != nil {
		return nil, err
	}
	return ro.Handle[:ro.Hcount], nil
}

func (r *RemoteIO) IoctlSetHandle(op uint32, in mx.Handle) error {
	if mxio.IoctlKind(op) != mxio.IoctlKindSetHandle {
		return mx.ErrInvalidArgs
	}
	var msg Msg
	msg.SetOp(OpIoctlOneHandle)
	msg.Handle[0] = in
	msg.Hcount = 1
	msg.SetIoctlOp(op)

	if _, err := r.txn(&msg); err != nil {
		return err
	}
	return nil
}

func (r *RemoteIO) Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error) {
	var msg Msg
	if len(in) > mxio.IoctlMaxInput {
		return nil, nil, mx.ErrInvalidArgs
	}
	msg.SetOp(OpIoctl)
	msg.Datalen = uint32(len(in))
	msg.Arg = MessageChunkSize
	msg.SetIoctlOp(op)
	copy(msg.Data[:], in)

	_, err := r.txn(&msg)
	if err != nil {
		return nil, nil, err
	}

	const handleSize = uint32(unsafe.Sizeof(mx.Handle(0)))
	switch mxio.IoctlKind(op) {
	case mxio.IoctlKindGetHandle:
		if msg.Hcount < 1 {
			return nil, nil, nil
		}
		out := []mx.Handle{msg.Handle[0]}
		msg.Handle[0] = 0
		msg.DiscardHandles()
		var data []byte
		if msg.Datalen > handleSize {
			data = msg.Data[handleSize:msg.Datalen]
		}
		return data, out, nil
	case mxio.IoctlKindGetTwoHandles:
		if msg.Hcount < 2 {
			return nil, nil, nil
		}
		out := []mx.Handle{msg.Handle[0], msg.Handle[1]}
		msg.Handle[0] = 0
		msg.Handle[1] = 0
		msg.DiscardHandles()
		var data []byte
		if msg.Datalen > 2*handleSize {
			data = msg.Data[2*handleSize : msg.Datalen]
		}
		return data, out, nil
	case mxio.IoctlKindGetThreeHandles:
		if msg.Hcount < 3 {
			return nil, nil, nil
		}
		out := []mx.Handle{msg.Handle[0], msg.Handle[1], msg.Handle[2]}
		msg.Handle[0] = 0
		msg.Handle[1] = 0
		msg.Handle[2] = 0
		msg.DiscardHandles()
		var data []byte
		if msg.Datalen > 3*handleSize {
			data = msg.Data[3*handleSize : msg.Datalen]
		}
		return data, out, nil
	default:
		msg.DiscardHandles()
		return msg.Data[:msg.Datalen], nil, nil
	}
}

func (r *RemoteIO) Misc(op uint32, off int64, in, out []byte) (n int, err error) {
	if len(in) > MessageChunkSize || len(out) > MessageChunkSize {
		return 0, mx.ErrInvalidArgs
	}

	var msg Msg
	msg.SetOp(op)
	msg.Arg = int32(len(out))
	msg.Datalen = uint32(len(in))
	msg.SetOff(off)
	copy(msg.Data[:], in)

	if op == OpRename {
		// Rename is a special snowflake that sends a reply channel, like open and clone.
		// Unlike open and close, it only cares about the 'status' response.
		if _, err := r.replyChannelCall(&msg); err != nil {
			return 0, err
		} else {
			msg.DiscardHandles()
			return 0, nil
		}
	}
	if _, err = r.txn(&msg); err != nil {
		return 0, err
	}
	msg.DiscardHandles()

	if int(msg.Datalen) > len(out) {
		return 0, mx.ErrIO
	}
	n = copy(out, msg.Data[:msg.Datalen])
	return n, nil
}

func Stat(f mxio.MXIO) (attr mxio.Vnattr, err error) {
	b := (*[unsafe.Sizeof(attr)]byte)(unsafe.Pointer(&attr))
	if _, err := f.Misc(OpStat, 0, nil, (*b)[:]); err != nil {
		return mxio.Vnattr{}, err
	}
	return attr, nil
}

func UnlinkAt(dir mxio.MXIO, name string) error {
	_, err := dir.Misc(OpUnlink, 0, []byte(name), nil)
	if err != nil {
		return err
	}
	return nil
}
