// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"syscall/zx"
	"time"
)

type ProcessState struct {
	retCode int
}

func (p *ProcessState) exited() bool {
	panic("TODO os.Exited")
}

func (p *ProcessState) Pid() int {
	panic("TODO os.Pid")
}

func (p *ProcessState) String() string {
	if p == nil {
		return "<nil>"
	}
	return "exit status " + itoa(p.retCode)
}

func (p *ProcessState) success() bool {
	return p.retCode == 0
}

func (p *ProcessState) sys() interface{} {
	panic("TODO os.sys")
}

func (p *ProcessState) sysUsage() interface{} {
	panic("TODO os.sysUsage")
}

func (p *ProcessState) systemTime() time.Duration {
	panic("TODO os.systemTime")
}

func (p *ProcessState) userTime() time.Duration {
	panic("TODO os.userTime")
}

func (p *Process) release() error {
	procHandle := zx.Handle(p.handle)
	return procHandle.Close()
}

func (p *Process) signal(sig Signal) error {
	panic("TODO os.Process.signal")
}

func findProcess(pid int) (p *Process, err error) {
	panic("TODO os.findProcess")
}

func startProcess(name string, argv []string, attr *ProcAttr) (p *Process, err error) {
	h, err := launchpadLaunchFdio(name, argv, attr)
	if err != nil {
		return nil, err
	}
	pid := 1 // TODO: what does this even mean for us?
	return newProcess(pid, uintptr(h)), nil
}

func hostname() (name string, err error) {
	return "", errors.New("os.hostname unimplemented")
}

type signal string

func (s signal) String() string { return string(s) }
func (s signal) Signal()        {}

var Interrupt = signal("signal_interrupt")
