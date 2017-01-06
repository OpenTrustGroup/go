// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package os

// #cgo fuchsia LDFLAGS: -llaunchpad
// #include <launchpad/launchpad.h>
// #include <magenta/types.h>
// #include <magenta/syscalls.h>
// #include <magenta/syscalls/object.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"syscall/mx"
	"unsafe"
)

func launchpadLaunchMxio(name string, argv []string) (mx.Handle, error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argvC := make([]*C.char, len(argv)+1)
	for i, s := range argv {
		argvC[i] = (*C.char)(C.CString(s))
	}
	argvC[len(argvC)-1] = nil
	h := mx.Handle(C.launchpad_launch_mxio(nameC, 0, &(argvC[0])))
	for i := range argv {
		if argvC[i] != nil {
			C.free(unsafe.Pointer(argvC[i]))
		}
	}
	if h < 0 {
		return 0, errors.New("launchpad error")
	}
	return h, nil
}

// TODO: get/generate these in syscalls/mx
const signalTaskTerminated = mx.Signals(1 << 3)
const objectInfoProcess = 3

func (p *Process) wait() (ps *ProcessState, err error) {
	procHandle := mx.Handle(p.handle)
	_, err = procHandle.WaitOne(signalTaskTerminated, mx.TimensecInfinite)
	if err != nil {
		return nil, err
	}
	procInfo := C.mx_info_process_t{}
	status := mx.Sys_object_get_info(procHandle, objectInfoProcess, unsafe.Pointer(&procInfo), C.sizeof_mx_info_process_t, nil, nil)
	if status != mx.ErrOk {
		println("mx.Sys_object_get_info(", procHandle, "MX_INFO_PROCESS,...) returned", status)
		return nil, errors.New("error retreiving process info")
	}
	return &ProcessState{int(procInfo.return_code)}, nil
}
