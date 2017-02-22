// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia
// +build cgo

package mxc

// #cgo fuchsia LDFLAGS: -lmxio
// #include <magenta/types.h>
// #include <magenta/syscalls.h>
// #include <stdlib.h>
// #include <mxio/util.h>
import "C"
import (
	"errors"
	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxio/pipe"
	"syscall/mx/mxio/rio"
)

func transferMXIOFromCFD(fd int) (mxio.MXIO, error) {
	handlesC := make([]C.mx_handle_t, mxio.MaxHandles)
	typesC := make([]C.uint32_t, mxio.MaxHandles)
	status := C.mxio_transfer_fd(C.int(fd), 0, (*C.mx_handle_t)(&handlesC[0]), (*C.uint32_t)(&typesC[0]))
	if status < 0 {
		return nil, mx.Error{Status: mx.Status(status), Text: "mxc"}
	} else if status == 0 || mxio.MaxHandles < status {
		return nil, errors.New("Invalid handle from mxio_transfer_fd")
	}
	handles := make([]mx.Handle, int(status))
	for i := 0; i < int(status); i++ {
		handles[i] = mx.Handle(handlesC[i])
	}
	switch mx.HandleInfo(typesC[0]).Type() {
	case mxio.HandleTypeRemote:
		return rio.New(handles)
	case mxio.HandleTypePipe:
		return pipe.New(handles)
	default:
		return nil, errors.New("Unexpected C handle type")
	}
}
