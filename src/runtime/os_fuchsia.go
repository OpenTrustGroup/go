// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"syscall/mx"
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
	timeout := mx.Time(ns)
	if ns < 0 {
		timeout = mx.TimensecInfinite
	}
	mx.Sys_futex_wait((*int)(unsafe.Pointer(addr)), int(val), timeout)
}

//go:nosplit
func futexwakeup(addr *uint32, cnt uint32) {
	mx.Sys_futex_wake((*int)(unsafe.Pointer(addr)), cnt)
}

//go:nosplit
func osyield() {
	mx.Sys_nanosleep(0)
}

// cgocallm0 calls a C function on the current stack.
//
// It is intended for use inside functions like osinit that run
// before mstart directly on an OS thread stack.
//
// If in doubt, do not use.
func cgocallm0(fn, arg unsafe.Pointer)

func osinit() {
	if _cgo_get_mxio_root != nil {
		cgocallm0(_cgo_get_mxio_root, unsafe.Pointer(&mxioHandles))
		mx.RootHandle = mx.Handle(mxioHandles.rootClone)
		mx.StdioHandle = mx.Handle(mxioHandles.stdioClone)
		mx.CwdHandle = mx.Handle(mxioHandles.cwd)
		mx.ProcHandle = mx.Handle(mxioHandles.processSelf)
		mx.VmarRootHandle = mx.Handle(mxioHandles.vmarRootSelf)
	} else {
		// TODO: implement cgo-less init
		println("runtime: no fuchsia process handle without cgo yet")
		exit(2)
	}

	ncpu = int32(mx.Sys_num_cpus())
}

// Filled in by runtime/cgo when linked into binary.
var _cgo_get_mxio_root unsafe.Pointer // pointer to C function

func minit() {
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
	p := mx.Handle(0) // TODO: only works due to temporary hack in magenta
	var h mx.Handle
	status := mx.Sys_thread_create(p, &gothreadname[0], uint32(len(gothreadname)), 0, &h)
	if status < 0 {
		println("runtime: newosproc mx.Sys_thread_create failed:", h)
		exit(2)
	}

	status = mx.Sys_thread_start(h, funcPC(threadinit), uintptr(stk), uintptr(unsafe.Pointer(mp)), 0)
	if status < 0 {
		println("runtime: newosproc mx.Sys_thread_start failed:", h)
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
	handle        *mx.Handle
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

// mxio_init matches a Go struct of the same name in gcc_mxio.c.
type mxio_init struct {
	rootClone    mx.Handle
	stdioClone   mx.Handle
	cwd          mx.Handle
	processSelf  mx.Handle
	vmarRootSelf mx.Handle
	_            int32
	envlen       int32
	environ      **byte
}

var mxioHandles mxio_init

func goenvs() {
	if _cgo_get_mxio_root != nil {
		const maxEnvSize = 1 << 20 // arbitrary
		envp := (*(*[maxEnvSize]*byte)(unsafe.Pointer(mxioHandles.environ)))[:mxioHandles.envlen]
		envs = make([]string, len(envp))
		for i, e := range envp {
			envs[i] = gostring(e)
		}
	} else {
		// TODO: implement cgo-less init
		println("runtime: no fuchsia process handle without cgo yet")
		exit(2)
	}
}

const _NSIG = 65 // TODO

//go:nosplit
func nanotime() int64 {
	const MX_CLOCK_MONOTONIC = 1
	return int64(mx.Sys_time_get(MX_CLOCK_MONOTONIC))
}

//go:linkname time_now time.now
//go:nosplit
func time_now() (sec int64, nsec int32) {
	const MX_CLOCK_UTC = 1
	x := int64(mx.Sys_time_get(MX_CLOCK_UTC))
	return int64(x / 1e6), int32(x % 1e6)
}

func unixnanotime() int64 {
	return nanotime()
}

var logger mx.Handle
var logBufferArray [1024]byte
var logBuffer []byte

//go:nosplit
func write(fd uintptr, buf unsafe.Pointer, n int32) int32 {
	if fd == 1 || fd == 2 {
		// This initialization mechanism is not thread-safe, but if the only calls to 'write' with
		// these file descriptors come from print, they will already be serialized by a lock in
		// print's implementation.
		if logger <= 0 {
			logger = mx.Sys_log_create(0)
			logBuffer = logBufferArray[:0]
		}
		if logger > 0 {
			// Sys_log_write adds a newline to output buffers. To prevent fragmented logs, we buffer
			// output until a newline character is encountered.
			logBuffer = append(logBuffer, (*[1 << 30]byte)(buf)[:n]...)
			if logBuffer[len(logBuffer)-1] == '\n' {
				status := mx.Sys_log_write(logger, uint32(len(logBuffer)), unsafe.Pointer(&logBuffer[0]), 0)
				// Flush buffer, regardless of status returned from write
				logBuffer = logBuffer[:0]
				if status < 0 {
					return int32(status)
				}
			}
		}
	}
	// TODO
	return n
}

//go:nosplit
func exit(code int32) {
	mx.Sys_process_exit(int(code))
}

//go:nosplit
func usleep(usec uint32) {
	mx.Sys_nanosleep(mx.Time(usec * 1000))
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
	status := mx.Sys_thread_arch_prctl(0, ARCH_SET_FS, &val)
	if status < 0 {
		println("runtime: thread_arch_prctl failed err=", status)
		exit(2)
	}
}

//go:linkname os_sigpipe os.sigpipe
func os_sigpipe() {
	// TODO
}
