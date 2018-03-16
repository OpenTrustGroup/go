// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package dispatch_test

import (
	"sync/atomic"
	"syscall/zx"
	. "syscall/zx/dispatch"
	"testing"
	"time"
)

func dispatcherTest(t *testing.T, name string, f func(d *Dispatcher) error) {
	t.Run(name, func(t *testing.T) {
		d, err := NewDispatcher()
		if err != nil {
			t.Fatal("creating dispatcher: ", err)
		}
		go d.Serve()
		if err := f(d); err != nil {
			t.Fatal("executing test: ", err)
		}
		d.Close()
	})
}

func TestWait(t *testing.T) {
	t.Parallel()
	dispatcherTest(t, "WaitOnce", func(d *Dispatcher) error {
		h0, h1, err := zx.NewChannel(0)
		if err != nil {
			return err
		}
		defer h0.Close()

		done := make(chan struct{}, 1)
		defer close(done)

		handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
			done <- struct{}{}
			return WaitFinished
		}

		_, err = d.BeginWait(zx.Handle(h0), zx.SignalChannelPeerClosed, 0, handler)
		if err != nil {
			return err
		}

		h1.Close()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			t.Fatal("timed out, handler never executed")
		}
		return nil
	})
	dispatcherTest(t, "WaitMultipleTimes", func(d *Dispatcher) error {
		h0, h1, err := zx.NewChannel(0)
		if err != nil {
			return err
		}
		defer h0.Close()
		defer h1.Close()

		done := make(chan struct{}, 1)
		defer close(done)

		handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
			var b [zx.ChannelMaxMessageBytes]byte
			var h [zx.ChannelMaxMessageHandles]zx.Handle
			bn, hn, err := h0.Read(b[:], h[:], 0)
			if err != nil {
				t.Fatal("error reading: ", err)
			}
			if bn != 1 || hn != 0 {
				t.Fatalf("unexpected read %d bytes %d handles", bn, hn)
			}
			if b[0] == 1 {
				return WaitAgain
			}
			done <- struct{}{}
			return WaitFinished
		}

		_, err = d.BeginWait(zx.Handle(h0), zx.SignalChannelReadable, 0, handler)
		if err != nil {
			return err
		}

		for i := 0; i < 5; i++ {
			if err := h1.Write([]byte{1}, nil, 0); err != nil {
				return err
			}
		}
		if err := h1.Write([]byte{0}, nil, 0); err != nil {
			return err
		}

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			t.Fatal("timed out, handler never executed")
		}
		return nil
	})
	dispatcherTest(t, "CancelWaitHandleShutdown", func(d *Dispatcher) error {
		h0, h1, err := zx.NewChannel(0)
		if err != nil {
			return err
		}
		defer h0.Close()
		defer h1.Close()

		var i uint32
		handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
			atomic.AddUint32(&i, 1)
			return WaitFinished
		}

		id, err := d.BeginWait(zx.Handle(h0), zx.SignalChannelPeerClosed, HandleShutdown, handler)
		if err != nil {
			return err
		}

		if err := d.CancelWait(id); err != nil {
			return err
		}
		d.Close()
		if i != 0 {
			t.Fatal("handler ran unexpectedly")
		}
		return nil
	})
	dispatcherTest(t, "Shutdown", func(d *Dispatcher) error {
		h0, h1, err := zx.NewChannel(0)
		if err != nil {
			return err
		}
		defer h0.Close()
		defer h1.Close()

		handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
			return WaitFinished
		}
		d.Close()

		_, err = d.BeginWait(zx.Handle(h0), zx.SignalChannelPeerClosed, 0, handler)
		if err == nil {
			t.Fatal("unexpected success for BeginWait on shut down dispatcher")
		}
		return nil
	})
	dispatcherTest(t, "HandleShutdown", func(d *Dispatcher) error {
		h0, h1, err := zx.NewChannel(0)
		if err != nil {
			return err
		}
		defer h0.Close()
		defer h1.Close()

		done := make(chan struct{}, 1)
		defer close(done)

		handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
			done <- struct{}{}
			return WaitFinished
		}

		_, err = d.BeginWait(zx.Handle(h0), zx.SignalChannelPeerClosed, HandleShutdown, handler)
		if err != nil {
			return err
		}
		d.Close()

		select {
		case <-done:
			return nil
		case <-time.After(5 * time.Second):
			t.Fatal("timed out, handler never executed")
		}
		return nil
	})
	dispatcherTest(t, "MultipleGoroutines", func(d *Dispatcher) error {
		// Spin up 5 goroutines all serving the dispatcher.
		for i := 0; i < 5; i++ {
			go d.Serve()
		}

		done := make(chan struct{}, 20)
		defer close(done)

		var ends []zx.Channel
		for i := 0; i < 20; i++ {
			h0, h1, err := zx.NewChannel(0)
			if err != nil {
				return err
			}
			defer h0.Close()

			handler := func(d *Dispatcher, s zx.Status, p *zx.PacketSignal) WaitResult {
				done <- struct{}{}
				return WaitFinished
			}

			_, err = d.BeginWait(zx.Handle(h0), zx.SignalChannelPeerClosed, 0, handler)
			if err != nil {
				return err
			}
			ends = append(ends, h1)
		}
		for _, e := range ends {
			e.Close()
		}
		for i := 0; i < 20; i++ {
			select {
			case <-done:
				continue
			case <-time.After(3 * time.Second):
				t.Fatal("timed out, some handler never executed")
			}
		}
		return nil
	})
}
