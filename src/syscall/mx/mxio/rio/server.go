// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package rio

import (
	"syscall/mx"
	"syscall/mx/mxio/dispatcher"
)

// ServerHandler is the type of the callback which contains the remoteio "server". It is called from
// the server handler function.
// - msg.Datalen represents the size of the incoming/outgoing message in msg.Data.
// - Return values are placed in msg.Arg. Negative is error.
type ServerHandler func(msg *Msg, rh mx.Handle, cookie int64) mx.Status

// Handler reads a single RIO message from the handle h. If the message is valid, the callback
// cb is invoked. If the callback is successful, a response may be sent back on handle h.
func Handler(h mx.Handle, cb interface{}, cookie int64) error {
	var msg Msg

	serverCb := cb.(ServerHandler)
	if h == 0 {
		// The remote side has been closed. Send our final callback and leave.
		msg.SetOp(OpClose)
		msg.Arg = 0
		msg.Datalen = 0
		msg.Hcount = 0
		serverCb(&msg, 0, cookie)
		return nil
	}
	msg.Hcount = MessageMaxHandles + 1
	numBytes, _, err := msg.ReadMsg(h)
	if err != nil {
		if err == mx.ErrBadState {
			return dispatcher.ErrNoWork
		}
		return err
	}
	if !msg.IsValidIncoming(numBytes) {
		msg.DiscardHandles()
		return mx.ErrInvalidArgs
	}

	rh := h
	if msg.HasReplyPipe() {
		if msg.Hcount == 0 {
			msg.DiscardHandles()
			return mx.ErrInvalidArgs
		}
		msg.Hcount--
		rh = msg.Handle[msg.Hcount]
	}
	isClose := (msg.Op() == OpClose)
	handleToSend := mx.Handle(0)
	if rh != h {
		handleToSend = rh
	}
	status := serverCb(&msg, handleToSend, cookie)
	msg.Arg = int32(status)
	if mx.Status(msg.Arg) == dispatcher.ErrIndirect {
		// The callback is handling the reply itself, and it took ownership of the reply handle
		return nil
	} else if status < 0 || !msg.IsValid() {
		// mx.Error case
		msg.DiscardHandles()
		msg.Datalen = 0
		msg.Hcount = 0
		if msg.Arg >= 0 {
			// TODO(smklein): This error code may change. It currently is set to "ERR_FAULT" in
			// Magenta, but is scheduled to change.
			msg.Arg = int32(mx.ErrIO)
		}
	}

	// The kernel requires that a reply pipe endpoint be returned as the last handle attached to a
	// write to that endpoint
	if rh != h {
		msg.Handle[msg.Hcount] = rh
		msg.Hcount++
	}
	msg.SetOp(OpStatus)
	err = msg.WriteMsg(rh)
	if err != nil {
		msg.DiscardHandles()
	}
	if isClose {
		// Signals that a close callback is not necessary -- the op has already completed
		return dispatcher.ErrDisconnectNoCallback
	}
	return err
}

// TxnHandoff passes a message to srv along with a reply handle rh.
// The other server will reply using rh, and this call will return immediately.
// rh is not closed on error.
func TxnHandoff(srv, rh mx.Handle, msg *Msg) error {
	msg.Magic = MessageMagic
	msg.Handle[0] = rh
	msg.Hcount = 1
	msg.SetReplyPipe()

	if !msg.IsValid() {
		return mx.ErrInvalidArgs
	}
	return msg.WriteMsg(srv)
}
