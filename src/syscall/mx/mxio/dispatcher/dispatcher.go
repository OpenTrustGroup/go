// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// Package dispatcher provides a mechanism to dispatch callbacks to readable handles
package dispatcher

import (
	"sync"

	"syscall/mx"
	"syscall/mx/mxio"
)

// A DispatcherCallback is called when the handle is readable or closed.
// Passing '0' to this callback as a handle indicates that the handle has been closed.
//
// A return value of err == nil causes another callback on the same handle.
// A return value of ErrNoWork causes the message draining to end.
// A return value of ErrDisconnectNoCallback disconnects the handler WITHOUT a callback.
// Any other error as a return value disconnects the handler WITH a final callback on an empty handle.
type DispatcherCallback func(h mx.Handle, callback interface{}, cookie int64) error

type handler struct {
	h        mx.Handle
	flags    uint32
	callback interface{}
	cookie   int64
}

// handler flag options
const (
	// flagDisconnected identifies that a handler is no longer usable.
	flagDisconnected = 1
)

// Dispatcher can read messages from a handle and assign them to a callback.
type Dispatcher struct {
	ioport   mx.Handle
	callback DispatcherCallback

	mu            sync.Mutex
	handlers      map[uint64]*handler
	nextHandlerID uint64
}

// Custom dispatcher errors
const (
	// If returned from dispatcher callback, indicates that the dispatcher should immediately
	// disconnect with no additional callbacks.
	ErrDisconnectNoCallback = mx.Status(1)
	// If returned from dispatcher callback, indicates that there are no more messages to read.
	ErrNoWork = mx.Status(-9999)
	// If returned from dispatcher callback, indicates that the message was handed to another
	// server.
	ErrIndirect = mx.Status(-9998)
)

// New creates a new dispatcher.
func New(callback DispatcherCallback) (*Dispatcher, error) {
	ioport, err := mxio.IoPortCreate(0)
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		ioport:   ioport,
		callback: callback,
		handlers: make(map[uint64]*handler),
	}, nil
}

// Closes a dispatcher, shutting down all handles added to the dispatcher and preventing additional
// handles from being added.
func (d *Dispatcher) close() {
	mx.HandleClose(d.ioport)
}

// Unbind a handler from the dispatcher to prevent further messages from arriving.
func (d *Dispatcher) disconnectHandler(hr *handler, handlerID uint64) {
	// Unbind handle from io port
	mxio.IoPortBind(d.ioport, handlerID, hr.h, 0)

	// Send a "SignalSignaled" close message to shut down the handler
	var packet mxio.Packet
	packet.Hdr.Key = handlerID
	packet.Signals = mx.SignalSignaled
	mxio.IoPortQueue(d.ioport, &packet, mxio.PacketSize)
	hr.flags |= flagDisconnected
}

func (d *Dispatcher) closeHandler(hr *handler, handlerID uint64) {
	mx.HandleClose(hr.h)
	d.mu.Lock()
	delete(d.handlers, handlerID)
	d.mu.Unlock()
}

// drainAllMessages calls the dispatcher callback on each readable message on the specific handle.
func (d *Dispatcher) drainAllMessages(hr *handler, handlerID uint64) (disconnected bool) {
	for {
		if err := d.callback(hr.h, hr.callback, hr.cookie); err != nil {
			if err == ErrNoWork {
				// No more messages to read
				return false
			}
			if err != ErrDisconnectNoCallback {
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
func (d *Dispatcher) Serve() {
	for {
		// Wait for any handler to be ready
		packet, err := mxio.IoPortWait(d.ioport, mxio.PacketSize)
		if err != nil {
			d.close()
			return
		}
		handlerID := packet.Hdr.Key
		d.mu.Lock()
		hr, ok := d.handlers[handlerID]
		d.mu.Unlock()
		if !ok {
			// The handler is concurrently being closed by another thread
			continue
		}
		if (hr.flags & flagDisconnected) != 0 {
			// Do not process packets from disconnected handlers.
			// Close handlers that receive the "SignalSignaled" close message.
			if (packet.Signals & mx.SignalSignaled) != 0 {
				d.closeHandler(hr, handlerID)
			}
			continue
		}
		// Read everything we can from this single handler
		if (packet.Signals & mx.SignalReadable) != 0 {
			if d.drainAllMessages(hr, handlerID) {
				// We disconnected from the handle -- don't bother considering a close
				continue
			}
		}
		// We haven't yet disconnected from the handler. Should we?
		if (packet.Signals & mx.SignalPeerClosed) != 0 {
			d.callback(0, hr.callback, hr.cookie)
			d.disconnectHandler(hr, handlerID)
		}
	}
}

// AddHandler adds a handle to the dispatcher, with an associated callback.  The callback will be
// called by the dispatcher server, as soon as h becomes readable or closed.
func (d *Dispatcher) AddHandler(h mx.Handle, callback interface{}, cookie int64) error {
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

	return mxio.IoPortBind(d.ioport, handlerID, h, mx.SignalReadable|mx.SignalPeerClosed)
}
