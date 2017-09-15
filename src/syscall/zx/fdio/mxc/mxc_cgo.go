// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia
// +build cgo

package mxc

// #cgo fuchsia LDFLAGS: -lfdio
// #include <zircon/types.h>
// #include <zircon/syscalls.h>
// #include <stdlib.h>
// #include <fdio/util.h>
import "C"
import (
	"errors"
	"syscall/zx"
	"syscall/zx/fdio"
)

func transferFDIOFromCFD(fd int) (fdio.FDIO, error) {
	handlesC := make([]C.zx_handle_t, fdio.MaxHandles)
	typesC := make([]C.uint32_t, fdio.MaxHandles)
	status := C.fdio_transfer_fd(C.int(fd), 0, (*C.zx_handle_t)(&handlesC[0]), (*C.uint32_t)(&typesC[0]))
	if status < 0 {
		return nil, zx.Error{Status: zx.Status(status), Text: "mxc"}
	} else if status == 0 || fdio.MaxHandles < status {
		return nil, errors.New("Invalid handle from fdio_transfer_fd")
	}
	handles := make([]zx.Handle, int(status))
	for i := 0; i < int(status); i++ {
		handles[i] = zx.Handle(handlesC[i])
	}
	switch zx.HandleInfo(typesC[0]).Type() {
	case fdio.HandleTypeRemote:
		return fdio.NewRIO(handles)
	case fdio.HandleTypePipe:
		return fdio.NewPipe(handles)
	default:
		return nil, errors.New("Unexpected C handle type")
	}
}
