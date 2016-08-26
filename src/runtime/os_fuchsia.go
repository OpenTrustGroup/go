// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"internal/syscall/fuchsia"
	"unsafe"
)

type mOS struct{}

type sigset struct{}

//go:nosplit
func getRandomData(r []byte) {
	// TODO
}

//go:nosplit
func futexsleep(addr *uint32, val uint32, ns int64) {
	timeout := fuchsia.Time(ns)
	if ns < 0 {
		timeout = fuchsia.TimensecInfinite
	}
	fuchsia.Sys_futex_wait((*int)(unsafe.Pointer(addr)), int(val), timeout)
}

//go:nosplit
func futexwakeup(addr *uint32, cnt uint32) {
	fuchsia.Sys_futex_wake((*int)(unsafe.Pointer(addr)), cnt)
}

//go:nosplit
func osyield() {
	fuchsia.Sys_nanosleep(0)
}

func osinit() {
	ncpu = int32(fuchsia.Sys_num_cpus())
}

func minit() {
	// TODO
}

//go:nosplit
func unminit() {
	// TODO
}

func sigpanic() {
	// TODO
}

func mpreinit(mp *m) {
	// TODO
}

//go:nosplit
func msigsave(mp *m) {
	// TODO
}

//go:nosplit
func msigrestore(sigmask sigset) {
	// TODO
}

func initsig(preinit bool) {
	// TODO
}

//go:nosplit
func sigblock() {
	// TODO
}

func sigenable(sig uint32) {
	// TODO
}

func sigdisable(sig uint32) {
	// TODO
}

func sigignore(sig uint32) {
	// TODO
}

func crash() {
	// TODO
}

func threadinit(mp *m)

var gothreadname = []byte("gothread")

//go:nowritebarrier
func newosproc(mp *m, stk unsafe.Pointer) {
	p := fuchsia.Handle(0) // TODO: only works due to temporary hack in magenta
	h := fuchsia.Sys_thread_create(p, &gothreadname[0], uint32(len(gothreadname)), 0)
	if h < 0 {
		println("runtime: newosproc fuchsia.Sys_thread_create failed:", h)
		exit(2)
	}

	status := fuchsia.Sys_thread_start(h, funcPC(threadinit), uintptr(stk), uintptr(unsafe.Pointer(mp)))
	if status < 0 {
		println("runtime: newosproc fuchsia.Sys_thread_start failed:", h)
		exit(2)
	}
}

type mx_proc_args struct {
	protocol        uint32
	version         uint32
	handle_info_off uint32
	args_off        uint32
	args_num        uint32
}

type mx_proc_info struct {
	magic         uint32
	version       uint32
	next_tls_slot uint32
	proc_args     *mx_proc_args
	handle        *fuchsia.Handle
	handle_info   *uint32
	handle_count  int32
	argv          **byte
	argc          int32
}

type mx_tls_root struct {
	self     *mx_tls_root
	proc     *mx_proc_info
	magic    uint32
	flags    uint16
	maxslots uint16
	slots    [8]uintptr // has length maxslots
}

func resetcpuprofiler(hz int32) {
	// TODO
}

func goenvs() {
	// TODO
}

const _NSIG = 65 // TODO

var _cgo_setenv unsafe.Pointer   // pointer to C function
var _cgo_unsetenv unsafe.Pointer // pointer to C function

//go:nosplit
func nanotime() int64 {
	return int64(fuchsia.Sys_current_time())
}

//go:nosplit
func write(fd uintptr, buf unsafe.Pointer, n int32) int32 {
	if fd == 1 || fd == 2 {
		fuchsia.Sys_debug_write(buf, uint32(n))
	}
	// TODO
	return n
}

//go:nosplit
func exit(code int32) {
	fuchsia.Sys_exit(int(code))
}

//go:nosplit
func usleep(usec uint32) {
	fuchsia.Sys_nanosleep(fuchsia.Time(usec * 1000))
}

func signame(sig uint32) string {
	return "unknown_sig" // TODO
}

// magenta_mktls
//go:nosplit
func magenta_mktls() {
	// TODO
}

//go:nosplit
func magenta_settls(val uintptr) {
	const ARCH_SET_FS = 0
	status := fuchsia.Sys_thread_arch_prctl(0, ARCH_SET_FS, &val)
	if status < 0 {
		println("runtime: thread_arch_prctl failed err=", status)
		exit(2)
	}
}
