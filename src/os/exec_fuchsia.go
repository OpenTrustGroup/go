// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "time"

type ProcessState struct {
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
	panic("TODO os.success")
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

func (p *Process) wait() (ps *ProcessState, err error) {
	panic("TODO os.Process.wait")
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
	panic("TODO os.startProcess")
}

func hostname() (name string, err error) {
	panic("TODO os.hostname")
}

func isExist(err error) bool      { panic("TODO os.isExist") }
func isNotExist(err error) bool   { panic("TODO os.isNotExist") }
func isPermission(err error) bool { panic("TODO os.isPermission") }

func Lstat(name string) (FileInfo, error) { panic("TODO os.Lstat") }

type signal string

func (s signal) String() string { return string(s) }
func (s signal) Signal()        {}

var Interrupt = signal("signal_interrupt")
