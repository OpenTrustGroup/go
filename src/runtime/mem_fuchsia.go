// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"syscall/mx"
	"unsafe"
)

//go:nosplit
func sysAlloc(n uintptr, sysStat *uint64) unsafe.Pointer {
	var p unsafe.Pointer
	sysMmap(unsafe.Pointer(&p), n, 0)
	if uintptr(p) < 4096 {
		println("runtime: sysMmap failed n=", n)
		exit(2)
	}
	mSysStatInc(sysStat, n)
	return p
}

func sysUnused(v unsafe.Pointer, n uintptr) {
}

func sysUsed(v unsafe.Pointer, n uintptr) {
}

//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatDec(sysStat, n)
	println("TODO sysFree")
}

func sysMmap(ptr unsafe.Pointer, len uintptr, flags uint32) {
	var h mx.Handle
	if status := mx.Sys_vmo_create(uint64(len), 0, &h); status != mx.ErrOk {
		println("runtime: mx.Sys_vmo_create failed, len=", len, " err=", status)
		exit(2)
	}
	const (
		MX_VM_FLAG_PERM_READ  = 1 << 1
		MX_VM_FLAG_PERM_WRITE = 1 << 2
	)
	mxFlags := uint32(MX_VM_FLAG_PERM_READ | MX_VM_FLAG_PERM_WRITE)
	status := mx.Sys_process_map_vm(mx.ProcHandle, h, 0, uint(len), (*uintptr)(ptr), mxFlags)
	mx.Sys_handle_close(h)
	if status != mx.ErrOk {
		println("runtime: mx.Sys_process_map_vm failed, err=", status)
		exit(2)
	}
}

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	if int(n) > 407<<20 {
		// Magenta does not like extreme overcommit, it consumes a
		// lot of kernel resources. So reject the larger reservations.
		return nil
	}
	var p unsafe.Pointer
	sysMmap(unsafe.Pointer(&p), n, 0)
	if p != nil {
		*reserved = true
	}
	return p
}

func sysFault(v unsafe.Pointer, n uintptr) {
	println("TODO sysFault")
}

func sysMap(v unsafe.Pointer, n uintptr, reserved bool, sysStat *uint64) {
	mSysStatInc(sysStat, n)
	if !reserved {
		var p unsafe.Pointer
		sysMmap(unsafe.Pointer(&p), n, 0)
		if p != v {
			print("runtime: address space conflict: map(", v, ") = ", p, "\n")
			throw("runtime: address space conflict")
		}
	}
}
