// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import (
	"errors"
	"time"
)

type ProcessState struct {
	retCode int
}

func (p *ProcessState) userTime() time.Duration {
	panic("TODO os.userTime")
}

func (p *ProcessState) systemTime() time.Duration {
	panic("TODO os.systemTime")
}

func (p *ProcessState) exited() bool {
	panic("TODO os.exited")
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

func (p *ProcessState) String() string {
	panic("TODO os.ProcessState.String")
}

func (p *Process) release() error {
	panic("TODO os.Process.release")
}

func (p *Process) kill() error {
	panic("TODO os.Process.kill")
}

func (p *Process) signal(sig Signal) error {
	panic("TODO os.Process.signal")
}

func findProcess(pid int) (p *Process, err error) {
	panic("TODO os.findProcess")
}

func startProcess(name string, argv []string, attr *ProcAttr) (p *Process, err error) {
	h, err := launchpadLaunchMxio(name, argv)
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
