// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package os

// #cgo fuchsia LDFLAGS: -llaunchpad
// #include <launchpad/launchpad.h>
// #include <launchpad/vmo.h>
// #include <magenta/types.h>
// #include <magenta/syscalls.h>
// #include <magenta/syscalls/object.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"syscall"
	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxio/pipe"
	"syscall/mx/mxio/rio"
	"unsafe"
)

func makeCStringArray(s []string) []*C.char {
	ret := make([]*C.char, len(s)+1)
	for i, s := range s {
		ret[i] = (*C.char)(C.CString(s))
	}
	ret[len(ret)-1] = nil
	return ret
}

func freeCStringArray(a []*C.char) {
	for i := range a {
		if a[i] != nil {
			C.free(unsafe.Pointer(a[i]))
		}
	}
}

func launchpadTransferFileToTargetFd(lp *C.launchpad_t, f *File, target int) error {
	if f == nil {
		return nil
	}
	m := syscall.MXIOForFD(int(f.Fd()))
	if m == nil {
		return ErrNotExist
	}
	handles, err := m.Clone()
	if err != nil {
		return ErrInvalid
	}
	handlesC := make([]C.mx_handle_t, mxio.MaxHandles)
	typesC := make([]C.uint32_t, mxio.MaxHandles)
	switch m.(type) {
	case *rio.RemoteIO:
		handlesC[0] = C.mx_handle_t(handles[0])
		handlesC[1] = C.mx_handle_t(handles[1])
		typesC[0] = C.uint32_t(mxio.HandleTypeRemote | target<<16)
		typesC[1] = C.uint32_t(mxio.HandleTypeRemote | target<<16)
	case *pipe.Pipe:
		handlesC[0] = C.mx_handle_t(handles[0])
		typesC[0] = C.uint32_t(mxio.HandleTypePipe | target<<16)
	default:
		for _, h := range handles {
			h.Close()
		}
		return ErrInvalid
	}
	status := mx.Status(C.launchpad_add_handles(lp, C.size_t(len(handles)), &(handlesC[0]), &(typesC[0])))
	if status != mx.ErrOk {
		for _, h := range handles {
			h.Close()
		}
		return errors.New("transfer file")
	}
	return nil
}

func launchpadSetWd(lp *C.launchpad_t, dir string) error {
	f, err := Open(dir)
	if err != nil {
		return err
	}
	m := syscall.MXIOForFD(int(f.Fd()))
	if m == nil {
		return ErrNotExist
	}
	handles, err := m.Clone()
	if err != nil {
		return ErrInvalid
	}
	handlesC := make([]C.mx_handle_t, mxio.MaxHandles)
	typesC := make([]C.uint32_t, mxio.MaxHandles)
	for i, h := range handles {
		handlesC[i] = C.mx_handle_t(h)
	}
	typesC[0] = mxio.HandleTypeCWD
	if _, ok := m.(*rio.RemoteIO); ok {
		typesC[1] = mxio.HandleTypeRemote
	}
	status := mx.Status(C.launchpad_add_handles(lp, C.size_t(len(handles)), &(handlesC[0]), &(typesC[0])))
	if status != mx.ErrOk {
		for _, h := range handles {
			h.Close()
		}
		return errors.New("transfer wd error: " + itoa(int(status)))
	}
	return nil
}

func launchpadLaunchMxio(name string, argv []string, attr *ProcAttr) (mx.Handle, error) {
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	argvC := makeCStringArray(argv)
	defer freeCStringArray(argvC)
	env := Environ()
	if attr.Env != nil {
		env = attr.Env
	}
	envC := makeCStringArray(env)
	defer freeCStringArray(envC)
	jobToChild := mx.Handle(0)
	job := mx.Handle(C.launchpad_get_mxio_job())
	if job > 0 {
		err := error(nil)
		jobToChild, err = job.Duplicate(mx.RightSameRights)
		if err != nil {
			return 0, errors.New("duplicating job")
		}
	}
	lp := &C.launchpad_t{}
	status := mx.Status(C.launchpad_create(C.mx_handle_t(jobToChild), argvC[0], &lp))
	if status != mx.ErrOk {
		return 0, errors.New("launchpad_create")
	}
	defer C.launchpad_destroy(lp)
	vmo := mx.Handle(C.launchpad_vmo_from_file(argvC[0]))
	status = mx.Status(C.launchpad_elf_load(lp, C.mx_handle_t(vmo)))
	if status != mx.ErrOk {
		vmo.Close()
		return 0, errors.New("loading elf")
	}
	status = mx.Status(C.launchpad_load_vdso(lp, C.mx_handle_t(0)))
	if status != mx.ErrOk {
		return 0, errors.New("loading vdso")
	}
	status = mx.Status(C.launchpad_arguments(lp, C.int(len(argvC)-1), &(argvC[0])))
	if status != mx.ErrOk {
		return 0, errors.New("arguments")
	}
	status = mx.Status(C.launchpad_environ(lp, &(envC[0])))
	if status != mx.ErrOk {
		return 0, errors.New("environ")
	}
	status = mx.Status(C.launchpad_clone_mxio_root(lp))
	if status != mx.ErrOk {
		return 0, errors.New("clone_mxio_root")
	}
	if attr.Dir != "" {
		err := launchpadSetWd(lp, attr.Dir)
		if err != nil {
			return 0, err
		}
	} else {
		status = mx.Status(C.launchpad_clone_mxio_cwd(lp)) // TODO: Go to Attr.Dir
		if status != mx.ErrOk {
			return 0, errors.New("clone_mxio_cwd")
		}
	}
	for i, f := range attr.Files {
		if err := launchpadTransferFileToTargetFd(lp, f, i); err != nil {
			return 0, err
		}
	}
	h := mx.Handle(C.launchpad_start(lp))
	if h < 0 {
		return 0, errors.New("launchpad error")
	}
	return h, nil
}

func (p *Process) kill() error {
	procHandle := mx.Handle(p.handle)
	status := mx.Sys_task_kill(procHandle)
	if status != mx.ErrOk {
		return errors.New("kill error: " + itoa(int(status)))
	}
	return nil
}

func (p *Process) wait() (ps *ProcessState, err error) {
	procHandle := mx.Handle(p.handle)
	_, err = procHandle.WaitOne(mx.SignalTaskTerminated, mx.TimensecInfinite)
	if err != nil {
		return nil, err
	}
	procInfo := C.mx_info_process_t{}
	status := mx.Sys_object_get_info(procHandle, mx.ObjectInfoProcess, unsafe.Pointer(&procInfo), C.sizeof_mx_info_process_t, nil, nil)
	if status != mx.ErrOk && status != mx.ErrNotEnoughBuffer {
		return nil, errors.New("error retreiving process info: " + itoa(int(status)))
	}
	return &ProcessState{int(procInfo.return_code)}, nil
}
