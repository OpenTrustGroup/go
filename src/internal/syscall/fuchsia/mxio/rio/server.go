// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rio

import (
	"internal/syscall/fuchsia"
	"internal/syscall/fuchsia/mxio/dispatcher"
)

// ServerHandler is the type of the callback which contains the remoteio "server". It is called from
// the server handler function.
// - msg.Datalen represents the size of the incoming/outgoing message in msg.Data.
// - Return values are placed in msg.Arg. Negative is error.
type ServerHandler func(msg *Msg, rh fuchsia.Handle, cookie int64) fuchsia.Status

// Handler reads a single RIO message from the handle h. If the message is valid, the callback
// cb is invoked. If the callback is successful, a response may be sent back on handle h.
func Handler(h fuchsia.Handle, cb interface{}, cookie int64) fuchsia.Status {
	var msg Msg

	serverCb := cb.(ServerHandler)
	if h == 0 {
		// The remote side has been closed. Send our final callback and leave.
		msg.SetOp(OpClose)
		msg.Arg = 0
		msg.Datalen = 0
		msg.Hcount = 0
		serverCb(&msg, 0, cookie)
		return fuchsia.ErrOk
	}
	msg.Hcount = MessageMaxHandles + 1
	numBytes, _, status := msg.ReadMsg(h)
	if status < 0 {
		if status == fuchsia.ErrBadState {
			return dispatcher.ErrNoWork
		}
		return status
	}
	if !msg.IsValidIncoming(numBytes) {
		msg.DiscardHandles()
		return fuchsia.ErrInvalidArgs
	}

	rh := h
	if msg.HasReplyPipe() {
		if msg.Hcount == 0 {
			msg.DiscardHandles()
			return fuchsia.ErrInvalidArgs
		}
		msg.Hcount--
		rh = msg.Handle[msg.Hcount]
	}
	isClose := (msg.Op() == OpClose)
	handleToSend := fuchsia.Handle(0)
	if rh != h {
		handleToSend = rh
	}
	status = serverCb(&msg, handleToSend, cookie)
	msg.Arg = int32(status)
	if msg.Arg == dispatcher.ErrIndirect {
		// The callback is handling the reply itself, and it took ownership of the reply handle
		return fuchsia.ErrOk
	} else if status < 0 || !msg.IsValid() {
		// fuchsia.Error case
		msg.DiscardHandles()
		msg.Datalen = 0
		msg.Hcount = 0
		if msg.Arg >= 0 {
			// TODO(smklein): This error code may change. It currently is set to "ERR_FAULT" in
			// Magenta, but is scheduled to change.
			msg.Arg = int32(fuchsia.ErrIO)
		}
	}

	// The kernel requires that a reply pipe endpoint be returned as the last handle attached to a
	// write to that endpoint
	if rh != h {
		msg.Handle[msg.Hcount] = rh
		msg.Hcount++
	}
	msg.SetOp(OpStatus)
	if status = msg.WriteMsg(rh); status < 0 {
		msg.DiscardHandles()
	}
	if isClose {
		// Signals that a close callback is not necessary -- the op has already completed
		return 1
	}
	return status
}

// TxnHandoff passes a message to srv along with a reply handle rh.
// The other server will reply using rh, and this call will return immediately.
// rh is not closed on error.
func TxnHandoff(srv, rh fuchsia.Handle, msg *Msg) fuchsia.Status {
	msg.Magic = MessageMagic
	msg.Handle[0] = rh
	msg.Hcount = 1
	msg.SetReplyPipe()

	if !msg.IsValid() {
		return fuchsia.ErrInvalidArgs
	}

	if r := msg.WriteMsg(srv); r < 0 {
		return r
	}
	return 0
}
