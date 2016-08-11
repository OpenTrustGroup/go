// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package os

import "time"

type ProcessState struct {
}

func (p *ProcessState) userTime() time.Duration {
	panic("TODO")
}

func (p *ProcessState) systemTime() time.Duration {
	panic("TODO")
}

func (p *ProcessState) exited() bool {
	panic("TODO")
}

func (p *ProcessState) success() bool {
	panic("TODO")
}

func (p *ProcessState) sys() interface{} {
	panic("TODO")
}

func (p *ProcessState) sysUsage() interface{} {
	panic("TODO")
}

func (p *ProcessState) String() string {
	panic("TODO")
}

func (p *Process) wait() (ps *ProcessState, err error) {
	panic("TODO")
}

func (p *Process) release() error {
	panic("TODO")
}

func (p *Process) kill() error {
	panic("TODO")
}

func (p *Process) signal(sig Signal) error {
	panic("TODO")
}

func findProcess(pid int) (p *Process, err error) {
	panic("TODO")
}

func startProcess(name string, argv []string, attr *ProcAttr) (p *Process, err error) {
	panic("TODO")
}

func hostname() (name string, err error) {
	panic("TODO")
}

func isExist(err error) bool      { panic("TODO") }
func isNotExist(err error) bool   { panic("TODO") }
func isPermission(err error) bool { panic("TODO") }

func Lstat(name string) (FileInfo, error) { panic("TODO") }

type signal string

func (s signal) String() string { return string(s) }
func (s signal) Signal()        {}

var Interrupt = signal("signal_interrupt")
