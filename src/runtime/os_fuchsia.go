// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"syscall/zx"
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
	deadline := zx.Sys_deadline_after(zx.Duration(ns))
	if ns < 0 {
		deadline = zx.Time(zx.TimensecInfinite)
	}
	zx.Sys_futex_wait((*int)(unsafe.Pointer(addr)), int(val), deadline)
}

//go:nosplit
func futexwakeup(addr *uint32, cnt uint32) {
	zx.Sys_futex_wake((*int)(unsafe.Pointer(addr)), cnt)
}

//go:nosplit
func osyield() {
	zx.Sys_nanosleep(0)
}

// cgocallm0 calls a C function on the current stack.
//
// It is intended for use inside functions like osinit that run
// before mstart directly on an OS thread stack.
//
// If in doubt, do not use.
func cgocallm0(fn, arg unsafe.Pointer)

func osinit() {
	if _cgo_get_fdio_root != nil {
		cgocallm0(_cgo_get_fdio_root, unsafe.Pointer(&fdioHandles))
		zx.RootHandle = zx.Handle(fdioHandles.rootClone)
		for i := 0; i < 3; i = i + 1 {
			for j := 0; j < int(fdioHandles.stdioCloneNumHandles[i]); j = j + 1 {
				zx.StdioHandles[i][j] = fdioHandles.stdioClones[i*3+j]
			}
			zx.StdioHandleTypes[i] = int(fdioHandles.stdioCloneTypes[i])
		}
		zx.CwdHandle = zx.Handle(fdioHandles.cwd)
		zx.ProcHandle = zx.Handle(fdioHandles.processSelf)
		zx.VMARRoot = zx.VMAR(fdioHandles.vmarRootSelf)
	} else {
		// TODO: implement cgo-less init
		println("runtime: no fuchsia process handle without cgo yet")
		exit(2)
	}

	ncpu = int32(zx.Sys_system_get_num_cpus())
	physPageSize = 4096
}

func parseRootNS() {
	if fdioHandles.rootNSNumHandles > 0 {
		const maxHandleCount = 1 << 20 // arbitrary
		paths := (*(*[maxHandleCount]*byte)(unsafe.Pointer(fdioHandles.rootNSPaths)))[:fdioHandles.rootNSNumHandles]
		handles := (*(*[maxHandleCount]zx.Handle)(unsafe.Pointer(fdioHandles.rootNSHandles)))[:fdioHandles.rootNSNumHandles]
		zx.RootNSMap = make(map[string]zx.Handle)
		for i, p := range paths {
			zx.RootNSMap[gostring(p)] = handles[i]
		}
	}
}

// Filled in by runtime/cgo when linked into binary.
var _cgo_get_fdio_root unsafe.Pointer // pointer to C function

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
	*(*int32)(nil) = 0
}

func threadinit(mp *m)

var gothreadname = []byte("gothread")

//go:nowritebarrier
func newosproc(mp *m, stk unsafe.Pointer) {
	p := zx.Handle(0) // TODO: only works due to temporary hack in zircon
	var h zx.Handle
	status := zx.Sys_thread_create(p, &gothreadname[0], uint32(len(gothreadname)), 0, &h)
	if status < 0 {
		println("runtime: newosproc zx.Sys_thread_create failed:", h)
		exit(2)
	}

	status = zx.Sys_thread_start(h, funcPC(threadinit), uintptr(stk), uintptr(unsafe.Pointer(mp)), 0)
	if status < 0 {
		println("runtime: newosproc zx.Sys_thread_start failed:", h)
		exit(2)
	}
}

type zx_proc_args struct {
	protocol        uint32
	version         uint32
	handle_info_off uint32
	args_off        uint32
	args_num        uint32
}

type zx_proc_info struct {
	magic         uint32
	version       uint32
	next_tls_slot uint32
	proc_args     *zx_proc_args
	handle        *zx.Handle
	handle_info   *uint32
	handle_count  int32
	argv          **byte
	argc          int32
}

type zx_tls_root struct {
	self     *zx_tls_root
	proc     *zx_proc_info
	magic    uint32
	flags    uint16
	maxslots uint16
	slots    [8]uintptr // has length maxslots
}

func resetcpuprofiler(hz int32) {
	// TODO
}

// fdio_init matches a Go struct of the same name in gcc_fdio.c.
type fdio_init struct {
	rootClone            zx.Handle
	stdioClones          [9]zx.Handle // 3 * fdio.MaxHandles
	stdioCloneNumHandles [3]uint32
	stdioCloneTypes      [3]uint32
	cwd                  zx.Handle
	processSelf          zx.Handle
	vmarRootSelf         zx.Handle
	_                    int32
	envlen               int32
	environ              **byte
	_                    int32
	rootNSNumHandles     int32
	rootNSHandles        *zx.Handle
	rootNSPaths          **byte
}

var fdioHandles fdio_init

func goenvs() {
	if _cgo_get_fdio_root != nil {
		const maxEnvSize = 1 << 20 // arbitrary
		envp := (*(*[maxEnvSize]*byte)(unsafe.Pointer(fdioHandles.environ)))[:fdioHandles.envlen]
		envs = make([]string, len(envp))
		for i, e := range envp {
			envs[i] = gostring(e)
		}
		// TODO: find a better place to call this
		parseRootNS()
	} else {
		// TODO: implement cgo-less init
		println("runtime: no fuchsia process handle without cgo yet")
		exit(2)
	}
}

const _NSIG = 65 // TODO

//go:nosplit
func nanotime() int64 {
	const MX_CLOCK_MONOTONIC = 0
	return int64(zx.Sys_time_get(MX_CLOCK_MONOTONIC))
}

//go:linkname time_now time.now
//go:nosplit
func time_now() (sec int64, nsec int32) {
	const MX_CLOCK_UTC = 1
	x := int64(zx.Sys_time_get(MX_CLOCK_UTC))
	return int64(x / 1e9), int32(x % 1e9)
}

func unixnanotime() int64 {
	return nanotime()
}

var logger zx.Handle
var logBufferArray [1024]byte
var logBufferN int

//go:nosplit
func write(fd uintptr, buf unsafe.Pointer, n int32) int32 {
	if fd == 1 || fd == 2 {
		// This initialization mechanism is not thread-safe, but if the only calls to 'write' with
		// these file descriptors come from print, they will already be serialized by a lock in
		// print's implementation.
		if logger <= 0 {
			zx.Sys_log_create(0, &logger)
			logBufferN = 0
		}
		if logger > 0 {
			// Sys_log_write adds a newline to output buffers. To prevent fragmented logs, we buffer
			// output until a newline character is encountered.
			logBufferN += copy(logBufferArray[logBufferN:], (*[1 << 30]byte)(buf)[:n])
			if logBufferArray[logBufferN-1] == '\n' {
				status := zx.Sys_log_write(logger, uint32(logBufferN), unsafe.Pointer(&logBufferArray[0]), 0)
				// Flush buffer, regardless of status returned from write
				logBufferN = 0
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
	zx.Sys_process_exit(int(code))
}

//go:nosplit
func usleep(usec uint32) {
	zx.Sys_nanosleep(zx.Sys_deadline_after(zx.Duration(usec * 1000)))
}

func signame(sig uint32) string {
	return "unknown_sig" // TODO
}

// zircon_mktls
//go:nosplit
func zircon_mktls() {
	// TODO
}

//go:nosplit
func zircon_settls(val uintptr) {
	// TODO: should call zx_set_object_property with property MX_PROP_REGISTER_FS in amd64 or
	// MX_PROP_REGISTER_CP15 on arm.
}

//go:linkname os_sigpipe os.sigpipe
func os_sigpipe() {
	// TODO
}
