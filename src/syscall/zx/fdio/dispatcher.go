// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"sync"

	"syscall/zx"
)

// A DispatcherCallback is called when the handle is readable or closed.
// Passing '0' to this callback as a handle indicates that the handle has been closed.
//
// A return value of err == nil causes another callback on the same handle.
// A return value of ErrNoWork causes the message draining to end.
// A return value of ErrDisconnectNoCallback disconnects the handler WITHOUT a callback.
// Any other error as a return value disconnects the handler WITH a final callback on an empty handle.
type DispatcherCallback func(h zx.Handle, callback interface{}, cookie int64) error

type handler struct {
	h        zx.Handle
	callback interface{}
	cookie   int64
}

// Dispatcher can read messages from a handle and assign them to a callback.
type Dispatcher struct {
	port     zx.Port
	callback DispatcherCallback

	mu            sync.Mutex
	handlers      map[uint64]*handler
	nextHandlerID uint64
}

// Custom dispatcher errors
var (
	// If returned from dispatcher callback, indicates that the dispatcher should immediately
	// disconnect with no additional callbacks.
	ErrDisconnectNoCallback = zx.Error{Status: 1, Text: "fdio.Dispatcher"}
	// If returned from dispatcher callback, indicates that there are no more messages to read.
	ErrNoWork = zx.Error{Status: -9999, Text: "fdio.Dispatcher.NoWork"}
	// If returned from dispatcher callback, indicates that the message was handed to another
	// server.
	ErrIndirect = zx.Error{Status: -9998, Text: "fdio.Dispatcher.Indirect"}
)

// New creates a new dispatcher.
func NewDispatcher(callback DispatcherCallback) (*Dispatcher, error) {
	port, err := zx.NewPort(0)
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		port:     port,
		callback: callback,
		handlers: make(map[uint64]*handler),
	}, nil
}

// Closes a dispatcher, shutting down all handles added to the dispatcher and preventing additional
// handles from being added.
func (d *Dispatcher) close() {
	d.port.Close()
	// TODO(abarth): We need to close all the handlers in d.handlers.
}

func (d *Dispatcher) destroyHandler(hr *handler, handlerID uint64, doCloseCallback bool) {
	// Prevent future dispatching to this handler.
	d.mu.Lock()
	delete(d.handlers, handlerID)
	d.mu.Unlock()

	// Issue the final callback to this handler, if required.
	if doCloseCallback {
		d.callback(0, hr.callback, hr.cookie)
	}

	// Close the underlying channel.
	hr.h.Close()
}

func (d *Dispatcher) dispatchHandler(hr *handler, handlerID uint64) {
	// Actually call the handler's callback.
	if err := d.callback(hr.h, hr.callback, hr.cookie); err != nil {
		if err != ErrNoWork {
			d.destroyHandler(hr, handlerID, err != nil)
			return
		}
	}
	// Rearm the handler
	err := d.port.WaitAsync(hr.h, handlerID, zx.SignalChannelReadable|zx.SignalChannelPeerClosed, zx.PortWaitAsyncOnce)
	if err != nil {
		d.mu.Lock()
		delete(d.handlers, handlerID)
		d.mu.Unlock()
	}
}

// Serve launches a dispatcher, which will continue serving 'forever'.
// The dispatcher waits on all handlers bound to it, and calls the callback of these handles as soon
// as any of them become readable or closed.
func (d *Dispatcher) Serve() {
	for {
		// Wait for any handler to be ready
		var packet zx.Packet
		err := d.port.Wait(&packet, zx.TimensecInfinite)
		if err != nil {
			d.close()
			return
		}
		handlerID := packet.Hdr.Key
		d.mu.Lock()
		hr, ok := d.handlers[handlerID]
		d.mu.Unlock()
		if !ok {
			// The handler was removed while this packet was in the queue.
			continue
		}
		// Dispatch to the appropriate handler.
		if (packet.Signal().Observed & zx.SignalChannelReadable) != 0 {
			go d.dispatchHandler(hr, handlerID)
			continue
		}
		// Destroy the handler and issue a final callback.
		if (packet.Signal().Observed & zx.SignalChannelPeerClosed) != 0 {
			go d.destroyHandler(hr, handlerID, true)
		}
	}
}

// AddHandler adds a handle to the dispatcher, with an associated callback.  The callback will be
// called by the dispatcher server, as soon as h becomes readable or closed.
func (d *Dispatcher) AddHandler(h zx.Handle, callback interface{}, cookie int64) error {
	hr := &handler{
		h:        h,
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

	err := d.port.WaitAsync(h, handlerID, zx.SignalChannelReadable|zx.SignalChannelPeerClosed, zx.PortWaitAsyncOnce)
	if err != nil {
		d.mu.Lock()
		delete(d.handlers, handlerID)
		d.mu.Unlock()
	}
	return err
}
