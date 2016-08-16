// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fuchsia

import "sync"

// A DispatcherCallback is called when the handle is readable or closed.
// Passing '0' to this callback as a handle indicates that the handle has been closed.
//
// A return value of |ErrOk| causes another callback on the same handle.
// A return value of |ErrDispatcherNoWork| causes the message draining to end.
// A POSITIVE return value disconnects the handler WITHOUT a callback.
// A NEGATIVE return value disconnects the handler WITH a final callback on an empty handle.
type DispatcherCallback func(h Handle, callback interface{}, cookie int64) Status

type handler struct {
	h        Handle
	flags    uint32
	callback interface{}
	cookie   int64
}

// handler flag options
const (
	// flagDisconnected identifies that a handler is no longer usable.
	flagDisconnected = 1
)

// MxioDispatcher can read messages from a handle and assign them to a callback.
type MxioDispatcher struct {
	ioport   Handle
	callback DispatcherCallback

	mu            sync.Mutex
	handlers      map[uint64]*handler
	nextHandlerID uint64
}

// Custom dispatcher errors
const (
	// If returned from dispatcher callback, indicates that there are no more messages to read.
	ErrDispatcherNoWork = -9999
	// If returned from dispatcher callback, indicates that the message was handed to another
	// server.
	ErrDispatcherIndirect = -9998
)

// New creates a new dispatcher.
func New(callback DispatcherCallback) (*MxioDispatcher, Status) {
	ioport, status := IoPortCreate(0)
	if status != ErrOk {
		return nil, status
	}
	return &MxioDispatcher{
		ioport:   ioport,
		callback: callback,
		handlers: make(map[uint64]*handler),
	}, ErrOk
}

// Closes a dispatcher, shutting down all handles added to the dispatcher and preventing additional
// handles from being added.
func (d *MxioDispatcher) close() {
	HandleClose(d.ioport)
}

// Unbind a handler from the dispatcher to prevent further messages from arriving.
func (d *MxioDispatcher) disconnectHandler(hr *handler, handlerID uint64) {
	// Unbind handle from io port
	IoPortBind(d.ioport, handlerID, hr.h, 0)

	// Send a "SignalSignaled" close message to shut down the handler
	var packet MxioPacket
	packet.Hdr.Key = handlerID
	packet.Signals = SignalSignaled
	IoPortQueue(d.ioport, &packet, MxioPacketSize)
	hr.flags |= flagDisconnected
}

func (d *MxioDispatcher) closeHandler(hr *handler, handlerID uint64) {
	HandleClose(hr.h)
	d.mu.Lock()
	delete(d.handlers, handlerID)
	d.mu.Unlock()
}

// drainAllMessages calls the dispatcher callback on each readable message on the specific handle.
func (d *MxioDispatcher) drainAllMessages(hr *handler, handlerID uint64) (disconnected bool) {
	for {
		if r := d.callback(hr.h, hr.callback, hr.cookie); r != ErrOk {
			if r == ErrDispatcherNoWork {
				// No more messages to read
				return false
			}
			if r < 0 {
				// Synthesize a close
				d.callback(0, hr.callback, hr.cookie)
			}
			d.disconnectHandler(hr, handlerID)
			return true
		}
	}
}

// Serve launches a dispatcher, which will continue serving 'forever'.
// The dispatcher waits on all handlers bound to it, and calls the callback of these handles as soon
// as any of them become readable or closed.
func (d *MxioDispatcher) Serve() {
	for {
		// Wait for any handler to be ready
		packet, status := IoPortWait(d.ioport, MxioPacketSize)
		if status != ErrOk {
			d.close()
			return
		}
		handlerID := packet.Hdr.Key
		d.mu.Lock()
		hr := d.handlers[handlerID]
		d.mu.Unlock()
		if (hr.flags & flagDisconnected) != 0 {
			// Do not process packets from disconnected handlers.
			// Close handlers that receive the "SignalSignaled" close message.
			if (packet.Signals & SignalSignaled) != 0 {
				d.closeHandler(hr, handlerID)
			}
			continue
		}
		// Read everything we can from this single handler
		if (packet.Signals & SignalReadable) != 0 {
			if d.drainAllMessages(hr, handlerID) {
				// We disconnected from the handle -- don't bother considering a close
				continue
			}
		}
		// We haven't yet disconnected from the handler. Should we?
		if (packet.Signals & SignalPeerClosed) != 0 {
			d.callback(0, hr.callback, hr.cookie)
			d.disconnectHandler(hr, handlerID)
		}
	}
}

// AddHandler adds a handle to the dispatcher, with an associated callback.  The callback will be
// called by the dispatcher server, as soon as |h| becomes readable or closed.
func (d *MxioDispatcher) AddHandler(h Handle, callback interface{}, cookie int64) Status {
	hr := &handler{
		h:        h,
		flags:    0,
		callback: callback,
		cookie:   cookie,
	}

	d.mu.Lock()
	var handlerID uint64
	for {
		// Find a handlerID which is not already in use
		handlerID = d.nextHandlerID
		d.nextHandlerID++
		if _, ok := d.handlers[handlerID]; !ok {
			break
		}
	}
	d.handlers[handlerID] = hr
	d.mu.Unlock()

	return IoPortBind(d.ioport, handlerID, h, SignalReadable|SignalPeerClosed)
}
