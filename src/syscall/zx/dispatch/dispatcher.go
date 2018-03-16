// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package dispatch

import (
	"fmt"
	"sync"
	"syscall/zx"
)

// WaitResult represents a wait result that is returned by the callback.
// WaitResult determines whether the wait should be re-queued.
type WaitResult int

const (
	WaitFinished WaitResult = iota
	WaitAgain
)

// WaitID is a monotonically increasing ID which corresponds to a particular
// call to BeginWait. Note that it may become invalidated if a wait is dequeued.
type WaitID uint64

// WaitFlags represents the collection of certain options for waiting.
type WaitFlags uint32

const (
	HandleShutdown WaitFlags = 1 << 0 // Calls the callback on shutdown.
)

// Handler is the callback that will be called when the wait is complete.
type Handler func(*Dispatcher, zx.Status, *zx.PacketSignal) WaitResult

// waitContext is a bookkeeping structure for in-progress waits.
type waitContext struct {
	object   zx.Handle
	callback Handler
	trigger  zx.Signals
	flags    WaitFlags
}

// Dispatcher can read messages from a handle and assign them to a callback.
type Dispatcher struct {
	// port is the underlying port used to wait on signals and dispatch.
	port zx.Port

	// objects is a map that manages wait contexts.
	mu         sync.RWMutex
	objects    map[WaitID]*waitContext
	nextWaitID WaitID

	// shutdown is whether the dispatcher has shut down.
	shutdown bool
}

// New creates a new dispatcher.
func NewDispatcher() (*Dispatcher, error) {
	port, err := zx.NewPort(0)
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		port:    port,
		objects: make(map[WaitID]*waitContext),
	}, nil
}

func assertWaitResult(result WaitResult, status zx.Status) {
	if !(result == WaitFinished || (result == WaitAgain && status == zx.ErrOk)) {
		panic(fmt.Sprintf("expected %d for status %d", result, status))
	}
}

// Closes a dispatcher, shutting down all handles added to the dispatcher.
func (d *Dispatcher) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.port.Close()
	for _, obj := range d.objects {
		if obj != nil && (obj.flags&HandleShutdown) != 0 {
			// Ignore the result; we won't be waiting again.
			result := obj.callback(d, zx.ErrCanceled, nil)
			assertWaitResult(result, zx.ErrCanceled)
		}
	}
	d.objects = make(map[WaitID]*waitContext)
	d.shutdown = true
}

func (d *Dispatcher) getWaitID() WaitID {
	_, ok := d.objects[d.nextWaitID]
	for ok {
		d.nextWaitID++
		_, ok = d.objects[d.nextWaitID]
	}
	return d.nextWaitID
}

// BeginWait creates a new wait on the handle h for signals t. The new wait
// will respect the flags f, and call the handler c on wait completion.
func (d *Dispatcher) BeginWait(h zx.Handle, t zx.Signals, f WaitFlags, c Handler) (WaitID, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If we've shut down, just notify that we're in a bad state.
	if d.shutdown {
		return 0, zx.Error{Status: zx.ErrBadState}
	}

	// Register the wait.
	id := d.getWaitID()
	d.objects[id] = &waitContext{
		object:   h,
		callback: c,
		trigger:  t,
		flags:    f,
	}

	// Schedule the wait on the port.
	err := d.port.WaitAsync(h, uint64(id), t, zx.PortWaitAsyncOnce)

	// If we fail, make sure to de-register the wait.
	if err != nil {
		delete(d.objects, id)
	}
	return id, err
}

// CancelWait cancels the wait with the given WaitID.
func (d *Dispatcher) CancelWait(id WaitID) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Look up the wait context.
	wc := d.objects[id]
	if wc == nil {
		return &zx.Error{Status: zx.ErrNotFound}
	}

	// Deregister no matter what here. Due to stack semantics of defer, this will always
	// execute before the lock is released.
	defer delete(d.objects, id)

	// Cancel the actual wait.
	return d.port.Cancel(wc.object, uint64(id))
}

func (d *Dispatcher) dispatch(id WaitID, wc *waitContext, signals *zx.PacketSignal) {
	// Deregister the handler before invoking it.
	d.mu.Lock()
	delete(d.objects, id)
	d.mu.Unlock()

	// Call the handler.
	result := wc.callback(d, zx.ErrOk, signals)
	assertWaitResult(result, zx.ErrOk)
	if result == WaitAgain {
		// Re-arm the handler, as the handler requested. Use a fresh ID since there's a
		// chance the old ID has already been re-used.
		d.mu.Lock()
		newID := d.getWaitID()
		d.objects[newID] = wc
		err := d.port.WaitAsync(wc.object, uint64(newID), wc.trigger, zx.PortWaitAsyncOnce)
		d.mu.Unlock()

		// If we fail to re-arm, notify the handler of what happened.
		if err != nil {
			zxErr := err.(zx.Error)
			result = wc.callback(d, zxErr.Status, nil)
			assertWaitResult(result, zxErr.Status)
		}
	}
}

// Serve runs indefinitely, waiting for the port to return a packet and dispatches
// the relevant handlers.
func (d *Dispatcher) Serve() {
	for {
		// Wait for any handler to be ready
		var packet zx.Packet
		err := d.port.Wait(&packet, zx.TimensecInfinite)
		if err != nil {
			d.Close()
			return
		}
		id := WaitID(packet.Hdr.Key)

		// Get the object for dispatching.
		d.mu.RLock()
		wc := d.objects[id]
		d.mu.RUnlock()

		// If handler was removed while this packet was in the queue,
		// just go back to waiting.
		if wc == nil {
			continue
		}

		// Dispatch to the appropriate handler.
		if packet.Hdr.Type == zx.PortPacketTypeSignalOne {
			d.dispatch(id, wc, packet.Signal())
		}
	}
}
