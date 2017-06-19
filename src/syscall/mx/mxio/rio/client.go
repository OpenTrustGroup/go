// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package rio implements the Remote IO protocol, providing the MXIO interface remotely
package rio

import (
	"strings"
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
		return 0, mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
	}

	msg.Txid = atomic.AddUint32(&r.txid, 1) - 1
	numBytes, _, rs, err := msg.CallMsg(r.h)
	if err != nil {
		if mxerr, hasStatus := err.(mx.Error); hasStatus && mxerr.Status == mx.ErrCallFailed {
			// Read failure: Real error in rs
			msg.Hcount = 0
			return 0, mx.Error{Status: rs, Text: "RemoteIO"}
		} else {
			// Write failure
			msg.DiscardHandles()
			return 0, err
		}
	}

	if !msg.IsValidIncoming(numBytes) || msg.Op() != OpStatus {
		msg.DiscardHandles() // If these handles were transmitted poorly, drop them
		return 0, mx.Error{Status: mx.ErrIO, Text: "RemoteIO"}
	}

	// Check for remote error
	status := mx.Status(msg.Arg)
	if status < 0 {
		msg.DiscardHandles() // Throw away any remaining handles; no one else will see them
		return 0, mx.Error{Status: status, Text: "RemoteIO"}
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
			if mxe, ok := err.(mx.Error); ok {
				status = mxe.Status
			} else {
				status = mx.ErrIO
			}
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
		return xferTotal, mx.Error{Status: status, Text: "RemoteIO.read"}
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
			if mxe, ok := err.(mx.Error); ok {
				status = mxe.Status
			} else {
				status = mx.ErrIO
			}
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
		return xferTotal, mx.Error{Status: status, Text: "RemoteIO.write"}
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
		return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
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
		return nil, mx.Error{Status: status, Text: "RemoteIO.replyChannelCall"}
	} else {
		return &ro, nil
	}
}

func connect(svc mx.Handle, cnxn mx.Handle, op uint32, name string) error {
	if len(name) >= mxio.PathMax {
		return mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO.connect"}
	}

	msg := Msg{}
	msg.SetOp(op)
	msg.Datalen = uint32(len(name))
	msg.Arg = int32(OpOpenFlagPipeline | OpOpenFlagRDWR)
	msg.SetMode(0755)
	msg.Hcount = 1
	msg.Handle[0] = cnxn
	copy(msg.Data[:], name)

	if err := msg.WriteMsg(&mx.Channel{svc}); err != nil {
		cnxn.Close()
		return err
	}

	return nil
}

func ServiceConnect(svcpath string, h mx.Handle) error {
	if mx.SVCRootHandle.IsValid() {
		// If there's an explicit svcroot handle, attempt to connect
		// through that.
		if svcpath == "/svc" {
			// TODO: may want to pass "" instead of "."
			//       once svcfs accepts it.
			return ServiceConnectAt(mx.SVCRootHandle, ".", h)
		} else if strings.HasPrefix(svcpath, "/svc/") {
			return ServiceConnectAt(mx.SVCRootHandle,
				strings.TrimPrefix(svcpath, "/svc/"), h)
		}
	} else {
		// Otherwise attempt to connect through the root namespace.
		ns, err := NewNSFromMap(mx.RootNSMap)
		if err == nil {
			return ns.Connect(svcpath, h)
		}
	}
	h.Close()
	return mx.Error{Status: mx.ErrNotFound, Text: "RemoteIO.ServiceConnect"}
}

func ServiceConnectAt(dir mx.Handle, path string, h mx.Handle) error {
	if !dir.IsValid() || !h.IsValid() {
		return mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO.ServiceConnectAt"}
	}
	return connect(dir, h, OpOpen, path)
}

func closeAll(handles []mx.Handle) {
	for i := range handles {
		handles[i].Close()
	}
}

// Consumes handles, wrapping them in mxio object or closing them on error.
func createFromHandles(protocol mxio.Protocol, handles []mx.Handle, extra []byte) (mxio.MXIO, error) {
	switch protocol {
	case mxio.ProtocolPipe:
		if len(handles) != 1 {
			closeAll(handles)
			return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "createFromHandles: Pipe"}
		}
		return pipe.New(handles)
	case mxio.ProtocolRemote:
		if len(handles) == 0 || len(handles) > 2 {
			closeAll(handles)
			return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "createFromHandles: RemoteIO"}
		}
		return New(handles)
	case mxio.ProtocolVMOFile:
		if len(handles) != 2 || len(extra) != 16 {
			closeAll(handles)
			return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "createFromHandles: VMOFile"}
		}
		handles[0].Close()
		return NewVMOFile(handles[1], extra)
	case mxio.ProtocolSocket:
		return NewSocket(handles)
	default:
		println("Unknown mxio protocol, ", protocol)
		// In the error case, the server may have given us handles -- close them
		closeAll(handles)
		return nil, mx.Error{Status: mx.ErrNotSupported, Text: "RemoteIO"}
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

const handleSize = uint32(unsafe.Sizeof(mx.Handle(0)))

// TODO(smklein): Remove; this function is a subset of "Ioctl".
func (r *RemoteIO) IoctlSetHandle(op uint32, in mx.Handle) error {
	if mxio.IoctlKind(op) != mxio.IoctlKindSetHandle {
		return mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
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

func (r *RemoteIO) Ioctl(op uint32, in, out []byte) ([]mx.Handle, error) {
	var msg Msg
	if len(in) > mxio.IoctlMaxInput {
		return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
	}
	msg.SetOp(OpIoctl)
	msg.Datalen = uint32(len(in))
	switch mxio.IoctlKind(op) {
	case mxio.IoctlKindGetHandle:
		msg.Arg = int32(handleSize)
	case mxio.IoctlKindGetTwoHandles:
		msg.Arg = int32(2 * handleSize)
	case mxio.IoctlKindGetThreeHandles:
		msg.Arg = int32(3 * handleSize)
	case mxio.IoctlKindSetHandle:
		msg.SetOp(OpIoctlOneHandle)
		if len(in) < int(handleSize) {
			return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
		}
		msg.Handle[0] = mx.Handle(*(*mx.Handle)(unsafe.Pointer(&in[0])))
		msg.Hcount = 1
	}
	msg.Arg += int32(len(out))
	msg.SetIoctlOp(op)
	copy(msg.Data[:], in)

	_, err := r.txn(&msg)
	if err != nil {
		return nil, err
	}

	switch mxio.IoctlKind(op) {
	case mxio.IoctlKindGetHandle:
		if msg.Hcount < 1 {
			return nil, mx.Error{Status: mx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []mx.Handle{msg.Handle[0]}
		msg.Handle[0] = 0
		msg.DiscardHandles()
		if msg.Datalen > handleSize {
			copy(out, msg.Data[handleSize:msg.Datalen])
		}
		return outh, nil
	case mxio.IoctlKindGetTwoHandles:
		if msg.Hcount < 2 {
			msg.DiscardHandles()
			return nil, mx.Error{Status: mx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []mx.Handle{msg.Handle[0], msg.Handle[1]}
		msg.Handle[0] = 0
		msg.Handle[1] = 0
		msg.DiscardHandles()
		if msg.Datalen > 2*handleSize {
			copy(out, msg.Data[2*handleSize:msg.Datalen])
		}
		return outh, nil
	case mxio.IoctlKindGetThreeHandles:
		if msg.Hcount < 3 {
			msg.DiscardHandles()
			return nil, mx.Error{Status: mx.ErrIO, Text: "RemoteIO.Ioctl too few handles"}
		}
		outh := []mx.Handle{msg.Handle[0], msg.Handle[1], msg.Handle[2]}
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

func (r *RemoteIO) Misc(op uint32, off int64, in, out []byte, handles []mx.Handle) (n int, err error) {
	if len(in) > MessageChunkSize || len(out) > MessageChunkSize {
		return 0, mx.Error{Status: mx.ErrInvalidArgs, Text: "RemoteIO"}
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
		return 0, mx.Error{Status: mx.ErrIO, Text: "RemoteIO"}
	}
	n = copy(out, msg.Data[:msg.Datalen])
	return n, nil
}

func Stat(f mxio.MXIO) (attr mxio.Vnattr, err error) {
	b := (*[unsafe.Sizeof(attr)]byte)(unsafe.Pointer(&attr))
	if _, err := f.Misc(OpStat, 0, nil, (*b)[:], nil); err != nil {
		return mxio.Vnattr{}, err
	}
	return attr, nil
}

func UnlinkAt(dir mxio.MXIO, name string) error {
	_, err := dir.Misc(OpUnlink, 0, []byte(name), nil, nil)
	if err != nil {
		return err
	}
	return nil
}
