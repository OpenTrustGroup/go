// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package rio implements the Remote IO protocol, providing the MXIO interface remotely
package rio

import (
	"syscall/mx"
	"syscall/mx/mxio"
	"unsafe"
)

// RemoteIO is an object which manages file-like operations on a remote object, accessible through a
// handle.
type RemoteIO struct {
	h     mx.Handle // Message pipe handle for the RPC
	e     mx.Handle // Event handle for device state signals
	flags uint32
}

// New returns a new RemoteIO object
func New(handle []mx.Handle) (*RemoteIO, error) {
	r := &RemoteIO{}
	if len(handle) > 0 {
		r.h = handle[0]
	}
	if len(handle) > 1 {
		r.e = handle[1]
	}
	return r, nil
}

// txn completes the client-side part of the RIO transaction.
// It creates a message pipe, writes msg (+ reply pipe), waits for a response, reads the
// response back into msg, and signals an error if anything goes wrong along the way.
func (r *RemoteIO) txn(msg *Msg) (uint32, error) {
	msg.Magic = MessageMagic
	if !msg.IsValid() {
		return 0, mx.ErrInvalidArgs
	}
	rpipe, err := mx.MsgPipeCreate(mxio.FlagReplyPipe)
	if err != nil {
		return 0, err
	}

	// Package the reply pipe into the message itself
	msg.SetReplyPipe()
	msg.Handle[msg.Hcount] = rpipe[1]
	msg.Hcount++
	rh := rpipe[0]
	defer rh.Close() // Our end of the reply pipe -- close it before returning

	// Send the message (containing the reply pipe) over the RIO handle
	if err := msg.WriteMsg(r.h); err != nil {
		msg.DiscardHandles() // If we can't transmit these handles, drop them
		return 0, err
	}

	// Wait for a response on the reply pipe
	signals, err := rh.WaitOne(mx.SignalReadable|mx.SignalPeerClosed, mx.TimensecInfinite)
	if err != nil {
		return 0, err
	}
	if signals&mx.SignalPeerClosed != 0 && signals&mx.SignalReadable == 0 {
		return 0, mx.ErrRemoteClosed
	}

	// Set the Handle count to the maximum possible value. This is just a conservative mechanism; if
	// the message is poorly transmitted, this encourages 'DiscardHandles' to throw away as many
	// handles as possible.
	msg.Hcount = MessageMaxHandles + 1

	// Read the response which is ready on the handle
	numBytes, _, err := msg.ReadMsg(rh)
	if err != nil {
		return 0, err
	}
	// The kernel ensures that the reply pipe endpoint is the last handle returned
	msg.Hcount--
	msg.Handle[msg.Hcount].Close()

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
		copy(msg.Data[:xfer], p[xferTotal:uint32(xferTotal)+xfer])

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
	r.h = 0
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
func (r *RemoteIO) getObject(op uint32, name string, flags int32, mode uint32) ([]mx.Handle, mxio.Protocol, []byte, error) {
	if len(name) > MessageChunkSize {
		return nil, 0, nil, mx.ErrInvalidArgs
	}
	var msg Msg
	msg.SetOp(op)
	msg.Datalen = uint32(len(name))
	msg.Arg = flags
	msg.SetMode(mode)
	copy(msg.Data[:], name)

	if _, err := r.txn(&msg); err != nil {
		return nil, mxio.ProtocolUndefined, nil, err
	}
	var extra []byte
	if msg.Datalen != 0 {
		extra = append(extra, msg.Data[:msg.Datalen]...)
	}
	return msg.Handle[:msg.Hcount], msg.Protocol(), extra, nil
}

func (r *RemoteIO) Open(path string, flags int32, mode uint32) (mxio.MXIO, error) {
	handles, protocol, extra, err := r.getObject(OpOpen, path, flags, mode)
	if err != nil {
		return nil, err
	}
	switch protocol {
	case mxio.ProtocolPipe:
		// TODO(smklein): Call "New" on a PipeIO object here, once that exists in Go
		panic("Pipe Protocol: Unimplemented")
	case mxio.ProtocolRemote:
		return New(handles)
	case mxio.ProtocolVMOFile:
		return NewVMOFile(handles[0], extra)
	default:
		println("unknown rio protocol, ", protocol)
		// In the error case, the server may have given us handles -- close them
		for i := range handles {
			handles[i].Close()
		}
		return nil, mx.ErrNotSupported
	}
}

func (r *RemoteIO) Clone() ([]mx.Handle, error) {
	handles, _, _, err := r.getObject(OpClone, "", 0, 0)
	if err != nil {
		return nil, err
	}
	return handles, nil
}

func (r *RemoteIO) Ioctl(op uint32, in []byte) ([]byte, []mx.Handle, error) {
	var msg Msg
	if len(in) > mxio.IoctlMaxInput {
		return nil, nil, mx.ErrInvalidArgs
	}
	msg.SetOp(OpIoctl)
	msg.Datalen = uint32(len(in))
	if mxio.IoctlKind(op) == mxio.IoctlKindGetHandle {
		msg.Arg = int32(unsafe.Sizeof(mx.Handle(0)))
	} else {
		msg.Arg = MessageChunkSize
	}
	msg.SetIoctlOp(op)
	copy(msg.Data[:], in)

	_, err := r.txn(&msg)
	if err != nil {
		return nil, nil, err
	}

	switch mxio.IoctlKind(op) {
	case mxio.IoctlKindGetHandle:
		var outHandles []mx.Handle
		if msg.Hcount > 0 {
			outHandles = append(outHandles, msg.Handle[0]) // Give the first handle back to the Ioctl caller
			msg.Handle[0] = 0                              // Avoid closing this handle; we've put it in out
			msg.DiscardHandles()
		} else {
			outHandles = nil
		}
		return nil, outHandles, nil
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
