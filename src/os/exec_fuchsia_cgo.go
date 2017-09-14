// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package os

// #cgo fuchsia LDFLAGS: -llaunchpad
// #include <launchpad/launchpad.h>
// #include <launchpad/vmo.h>
// #include <zircon/process.h>
// #include <zircon/syscalls.h>
// #include <zircon/syscalls/object.h>
// #include <zircon/types.h>
// #include <stdlib.h>
import "C"
import (
	"errors"
	"syscall"
	"syscall/mx"
	"syscall/mx/fdio"
	"unsafe"
)

const (
	LaunchpadCloneFdioNamespace = 1
	LaunchpadCloneFdioCwd       = 2
	LaunchpadCloneFdioStdio     = 4
	LaunchpadCloneFdioAll       = 0xff
	LaunchpadCloneEnviron       = 0x100
	LaunchpadCloneDefaultJob    = 0x200
	LaunchpadCloneAll           = 0xffff
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

// Clone an fdio to a C array of handle values and types.
func cloneHandleToCHandleArray(m fdio.FDIO) (int, []C.zx_handle_t, []C.uint32_t, error) {
	handles, err := m.Clone()
	if err != nil {
		return 0, nil, nil, ErrInvalid
	}
	handlesC := make([]C.zx_handle_t, fdio.MaxHandles)
	typesC := make([]C.uint32_t, fdio.MaxHandles)
	fdioType := 0
	switch m.(type) {
	case *fdio.RemoteIO:
		fdioType = fdio.HandleTypeRemote
	case *fdio.Pipe:
		fdioType = fdio.HandleTypePipe
	default:
		for _, h := range handles {
			h.Close()
		}
		return 0, nil, nil, ErrInvalid
	}
	for i, h := range handles {
		handlesC[i] = C.zx_handle_t(h)
		typesC[i] = C.uint32_t(fdioType)
	}
	return len(handles), handlesC, typesC, nil
}

func closeAll(handles []C.zx_handle_t) {
	for _, h := range handles {
		if h != 0 {
			mx.Handle(h).Close()
		}
	}
}

func launchpadTransferFileToTargetFd(lp *C.launchpad_t, f *File, target int) error {
	if f == nil {
		return nil
	}
	m := syscall.FDIOForFD(int(f.Fd()))
	if m == nil {
		return ErrInvalid
	}
	numHandles, handlesC, typesC, err := cloneHandleToCHandleArray(m)
	if err != nil {
		return err
	}
	for i := 0; i < numHandles; i = i + 1 {
		typesC[i] = typesC[i] | C.uint32_t(target<<16)
	}

	status := mx.Status(C.launchpad_add_handles(lp, C.size_t(numHandles), &(handlesC[0]), &(typesC[0])))
	if status != mx.ErrOk {
		closeAll(handlesC)
		return errors.New("transfer file")
	}
	return nil
}

func launchpadSetWd(lp *C.launchpad_t, dir string) error {
	f, err := Open(dir)
	if err != nil {
		return err
	}
	m := syscall.FDIOForFD(int(f.Fd()))
	if m == nil {
		return ErrNotExist
	}
	numHandles, handlesC, typesC, err := cloneHandleToCHandleArray(m)
	if err != nil {
		return err
	}
	typesC[0] = fdio.HandleTypeCWD
	status := mx.Status(C.launchpad_add_handles(lp, C.size_t(numHandles), &(handlesC[0]), &(typesC[0])))
	if status != mx.ErrOk {
		closeAll(handlesC)
		return errors.New("transfer wd error: " + itoa(int(status)))
	}
	return nil
}

func launchpadLaunchFdio(name string, argv []string, attr *ProcAttr) (mx.Handle, error) {
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
	job := mx.Handle(C.zx_job_default())
	if job > 0 {
		err := error(nil)
		jobToChild, err = job.Duplicate(mx.RightSameRights)
		if err != nil {
			return 0, errors.New("duplicating job")
		}
	}
	lp := &C.launchpad_t{}
	status := mx.Status(C.launchpad_create(C.zx_handle_t(jobToChild), nameC, &lp))
	if status != mx.ErrOk {
		return 0, errors.New("launchpad_create")
	}
	defer C.launchpad_destroy(lp)
	status = mx.Status(C.launchpad_load_from_file(lp, nameC))
	if status != mx.ErrOk {
		return 0, errors.New("loading vdso")
	}
	status = mx.Status(C.launchpad_set_args(lp, C.int(len(argvC)-1), &(argvC[0])))
	if status != mx.ErrOk {
		return 0, errors.New("arguments")
	}
	status = mx.Status(C.launchpad_set_environ(lp, &(envC[0])))
	if status != mx.ErrOk {
		return 0, errors.New("environ")
	}
	status = mx.Status(C.launchpad_clone(lp, C.uint32_t(LaunchpadCloneFdioNamespace)))
	if status != mx.ErrOk {
		return 0, errors.New("launchpad_clone")
	}
	if attr.Dir != "" {
		err := launchpadSetWd(lp, attr.Dir)
		if err != nil {
			return 0, err
		}
	} else {
		status = mx.Status(C.launchpad_clone(lp, C.uint32_t(LaunchpadCloneFdioCwd))) // TODO: Go to Attr.Dir
		if status != mx.ErrOk {
			return 0, errors.New("clone_fdio_cwd")
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
	procInfo := C.zx_info_process_t{}
	status := mx.Sys_object_get_info(procHandle, mx.ObjectInfoProcess, unsafe.Pointer(&procInfo), C.sizeof_zx_info_process_t, nil, nil)
	if status != mx.ErrOk {
		return nil, errors.New("error retreiving process info: " + itoa(int(status)))
	}
	return &ProcessState{int(procInfo.return_code)}, nil
}
