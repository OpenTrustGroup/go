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
	msg.Hcount = MessageMaxHandles
	numBytes, _, err := msg.ReadMsg(&mx.Channel{h}) // TODO: take *mx.Channel as arugment
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

	isClose := (msg.Op() == OpClose)
	status := serverCb(&msg, h, cookie)
	msg.Arg = int32(status)
	if mx.Status(msg.Arg) == dispatcher.ErrIndirect {
		// The callback is handling the reply itself
		return nil
	} else if status < 0 || !msg.IsValid() {
		// mx.Error case
		msg.DiscardHandles()
		msg.Datalen = 0
		if msg.Arg >= 0 {
			// TODO(smklein): This error code may change. It currently is set to "ERR_FAULT" in
			// Magenta, but is scheduled to change.
			msg.Arg = int32(mx.ErrIO)
		}
	}

	msg.SetOp(OpStatus)
	err = msg.WriteMsg(&mx.Channel{h}) // TODO: make h an *mx.Channel
	if err != nil {
		msg.DiscardHandles()
	}
	if isClose {
		// Signals that a close callback is not necessary -- the op has already completed
		return dispatcher.ErrDisconnectNoCallback
	}
	return err
}

// TxnHandoff passes a message to srv along with a reply handle 'reply'
// The other server will reply using 'reply', and this call will return immediately.
func TxnHandoff(srv *mx.Channel, reply mx.Handle, msg *Msg) {
	msg.Txid = 0
	msg.Handle[0] = reply
	msg.Hcount = 1

	if !msg.IsValid() {
		IndirectError(reply, mx.ErrInvalidArgs)
	} else if err := msg.WriteMsg(srv); err != nil {
		IndirectError(reply, err.(mx.Status))
	}
}
