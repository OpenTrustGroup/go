// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"bytes"
	"testing"

	"syscall/mx"
)

// Create a new dispatcher, running a RIO server. It is initialized with no clients.
func CreateRIOServer() *Dispatcher {
	d, err := NewDispatcher(Handler)
	if err != nil {
		panic(err)
	}
	go d.Serve()
	return d
}

// Create a pair of handles, and add one end to the RIO server, along with a RIO Server callback.
func CreateRIOClient(d *Dispatcher, cb ServerHandler, cookie int64) *RemoteIO {
	h0, h1, err := mx.NewChannel(0)
	if err != nil {
		panic(err)
	}

	if err := d.AddHandler(h0.Handle, cb, cookie); err != nil {
		panic(err)
	}

	// TODO(smklein): Add 'event' handle to clientHandles when testing 'wait'
	client, err := NewRIO([]mx.Handle{h1.Handle})
	if err != nil {
		panic(err)
	}
	return client
}

var testData = []struct {
	// Input messages
	inOp   uint32  // Operation (always used)
	inArg  int32   // 'arg', or 0 if ignored
	inArg2 int64   // 'arg2', or 0 if ignored
	inData []uint8 // 'data', or 0 if ignored

	// Output messages
	outRetval  mx.Status   // Expected return value from server
	outArg2    int64       // 'arg2', or 0 if ignored
	outData    []uint8     // 'data', or empty if ignored
	outHandles []mx.Handle // 'handles', or empty if ignored
}{
	// IN: Arg: Flags, Arg2: Mode, Data: Name
	// OUT: Retval: 0, Arg2: Protocol, Handles: Opened object (set elsewhere)
	{OpOpen, 12, 34, []uint8{'f', 'i', 'l', 'e'}, 0, int64(ProtocolRemote), []uint8{}, []mx.Handle{}},
	// IN: Nothing
	// OUT: Retval: 0, Arg2: Protocol, Handles: Opened object (set elsewhere)
	{OpClone, 0, 0, []uint8{}, 0, int64(ProtocolRemote), []uint8{}, []mx.Handle{}},
	// IN: Arg: Max bytes to read
	// OUT: Retval: Bytes read, Arg2: New offset, Data: Data read
	{OpRead, 4, 0, []uint8{}, 4, 5, []uint8{'d', 'a', 't', 'a'}, []mx.Handle{}},
	// IN: Data: Data to write
	// OUT: Retval: Bytes read, Arg2: New offset
	{OpWrite, 0, 0, []uint8{'i', 'n', 'p', 'u', 't'}, 5, 10, []uint8{}, []mx.Handle{}},
	// IN: Arg: Whence, Arg2: Offset
	// OUT: Retval: 0, Arg2: Offset
	{OpSeek, 1, 0x1234567, []uint8{}, 0, 0x7654321, []uint8{}, []mx.Handle{}},
	// IN: Arg: Max length of "out", Arg2: Opcode (arbitrary), Data: Arbitrary bytes
	// OUT: Retval: 0, Data: Arbitrary bytes
	{OpIoctl, MessageChunkSize, 1337, []uint8{'i', 'n'}, 0, 0, []uint8{'o', 'u', 't'}, []mx.Handle{}},
	// IN: Arg: Max length of "out", Arg2: Opcode (IoctlVFS)
	// OUT: Retval: 0, Handles: Opened object (set elsewhere)
	{OpIoctl, MessageChunkSize, int64(IoctlVFSMountFS), []uint8{}, 0, 0, []uint8{}, []mx.Handle{}},
	// IN: Nothing
	// OUT: Nothing
	{OpClose, 0, 0, []uint8{}, 0, 0, []uint8{}, []mx.Handle{}},
}

func checkOpen(client *RemoteIO, opIndex int) {
	// Create the expected return handle here, since dummy values would be rejected.
	h0, _, err := mx.NewChannel(0)
	if err != nil {
		panic(err)
	}
	testData[opIndex].outHandles = []mx.Handle{h0.Handle}
	_, err = client.Open(string(testData[opIndex].inData), testData[opIndex].inArg, uint32(testData[opIndex].inArg2))
	if err != nil {
		panic(err)
	}
}

func checkClone(client *RemoteIO, opIndex int) {
	// Create the expected return handle here, since dummy values would be rejected.
	h0, _, err := mx.NewChannel(0)
	if err != nil {
		panic(err)
	}
	testData[opIndex].outHandles = []mx.Handle{h0.Handle}
	handles, err := client.Clone()
	if err != nil {
		panic(err)
	} else if len(handles) != len(testData[opIndex].outHandles) {
		panic("Unexpected number of returned handles")
	} else if handles[0] != testData[opIndex].outHandles[0] {
		panic("Unexpected returned handle")
	}
}

func checkRead(client *RemoteIO, opIndex int) {
	buf := make([]byte, len(testData[opIndex].outData))
	count, err := client.Read(buf)
	if err != nil {
		panic(err)
	} else if count != len(buf) {
		panic("Unexpected read byte count")
	} else if !bytes.Equal(buf, testData[opIndex].outData) {
		panic("Unexpected read data")
	}
}

func checkWrite(client *RemoteIO, opIndex int) {
	count, err := client.Write(testData[opIndex].inData)
	if err != nil {
		panic(err)
	} else if count != len(testData[opIndex].inData) {
		panic("Unexpected write byte count")
	}
}

func checkSeek(client *RemoteIO, opIndex int) {
	off, err := client.Seek(testData[opIndex].inArg2, int(testData[opIndex].inArg))
	if err != nil {
		panic(err)
	} else if off != testData[opIndex].outArg2 {
		panic("Unexpected result offset from seek")
	}
}

func checkIoctl(client *RemoteIO, opIndex int) {
	if testData[opIndex].inArg2 == int64(IoctlVFSMountFS) {
		// Create the expected return handle here, since dummy values would be rejected.
		h0, _, err := mx.NewChannel(0)
		if err != nil {
			panic(err)
		}
		testData[opIndex].outHandles = []mx.Handle{h0.Handle}
	}

	outData, outHandles, err := client.Ioctl(uint32(testData[opIndex].inArg2), testData[opIndex].inData)
	if err != nil {
		panic(err)
	} else if !bytes.Equal(testData[opIndex].outData, outData) {
		panic("Unexpected ioctl output data")
	} else if len(outHandles) != len(testData[opIndex].outHandles) {
		panic("Unexpected ioctl output handle count")
	}
	for i := range outHandles {
		if outHandles[i] != testData[opIndex].outHandles[i] {
			panic("Unexpected handle")
		}
	}
}

func checkClose(client *RemoteIO) {
	if err := client.Close(); err != nil {
		panic(err)
	}
}

func TestRIO(t *testing.T) {
	testCookie := int64(0x12345678) // Checks callback plumbing
	opIndex := 0                    // Checks that client / server are in sync

	// This function will be called on the "fake" server. Check that everything is plumbed through
	// correctly. In a realistic use of RIO, this function might exist in another process
	// completely.
	serverCb := func(msg *Msg, rh mx.Handle, cookie int64) mx.Status {
		// Check incoming message
		if cookie != testCookie {
			t.Fatalf("%d: unexpected cookie", opIndex)
		}
		if rh != 0 && msg.Magic != MessageMagic {
			// The 'magic' value should only be checked if the message was actually sent from the
			// remote side, and not sent as a 'close' notification.
			t.Fatalf("%d: unexpected magic", opIndex)
		}
		if msg.Hcount != 0 {
			t.Fatalf("%d: unexpectedly sent a handle to the server", opIndex)
		}
		if got, want := testData[opIndex].inOp, msg.Op(); got != want {
			t.Fatalf("%d: inOp=%v, want %v", got, want)
		}
		if got, want := testData[opIndex].inArg, msg.Arg; got != want {
			t.Fatalf("%d: inArg=%v, want %v", got, want)
		}
		if got, want := testData[opIndex].inArg2, msg.Off(); got != want {
			t.Fatalf("%d: inArg2=%v, want %v", got, want)
		}
		if got, want := len(testData[opIndex].inData), int(msg.Datalen); got != want {
			t.Fatalf("%d: len(inData)=%d, want %d", got, want)
		}
		if got, want := testData[opIndex].inData, msg.Data[:msg.Datalen]; !bytes.Equal(got, want) {
			t.Fatalf("%d: inData=%q, want %q", got, want)
		}

		// Prepare outgoing message
		msg.SetOff(testData[opIndex].outArg2)
		copy(msg.Data[:], testData[opIndex].outData)
		msg.Datalen = uint32(len(testData[opIndex].outData))
		copy(msg.Handle[:], testData[opIndex].outHandles)
		msg.Hcount = uint32(len(testData[opIndex].outHandles))
		return testData[opIndex].outRetval
	}

	d := CreateRIOServer()
	client := CreateRIOClient(d, serverCb, testCookie)
	for range testData {
		switch testData[opIndex].inOp {
		case OpOpen:
			checkOpen(client, opIndex)
		case OpClone:
			checkClone(client, opIndex)
		case OpRead:
			checkRead(client, opIndex)
		case OpWrite:
			checkWrite(client, opIndex)
		case OpSeek:
			checkSeek(client, opIndex)
		case OpIoctl:
			checkIoctl(client, opIndex)
		case OpClose:
			checkClose(client)
		default:
			t.Fatalf("%d: unexpected test op: %v", opIndex, testData[opIndex].inOp)
		}
		opIndex++
	}
}
