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
	// TODO: enable unmapping when sysUsed is implemented.
	//off := uintptr(v) - sysVMARBaseAddr
	//sysVMAR.Unmap(sysVmar, off, uint(n))
}

func sysUsed(v unsafe.Pointer, n uintptr) {
}

//go:nosplit
func sysFree(v unsafe.Pointer, n uintptr, sysStat *uint64) {
	mSysStatDec(sysStat, n)
	println("TODO sysFree")
}

func sysMmap(ptr unsafe.Pointer, len uintptr, flags uint32) {
	vmo, err := mx.NewVMO(uint64(len), 0)
	if err != nil {
		println("runtime: mx.NewVMO failed, len=", len, " err=", err.Error())
		exit(2)
	}
	vmflags := mx.VMFlagPermRead | mx.VMFlagPermWrite
	res, err := mx.VMARRoot.Map(0, vmo, 0, uint64(len), vmflags)
	*(*uintptr)(ptr) = res
	vmo.Close()
	if err != nil {
		println("runtime: mx.Sys_vmar_map failed, err=", err.Error())
		exit(2)
	}
}

var (
	sysVMAR         mx.VMAR
	sysVMARBaseAddr uintptr
)

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	if sysVMAR != 0 {
		println("runtime: system vmar already reserved")
		exit(2)
	}

	var err error
	*reserved = true
	flags := mx.VMFlagCanMapSpecific | mx.VMFlagCanMapRead | mx.VMFlagCanMapWrite
	sysVMAR, sysVMARBaseAddr, err = mx.NewVMAR(mx.VMARRoot, 0, uint64(n), flags)
	if err != nil {
		println("runtime: vmar allocate failed, err=", err.Error())
		exit(2)
	}
	return unsafe.Pointer(sysVMARBaseAddr)
}

func sysFault(v unsafe.Pointer, n uintptr) {
	println("TODO sysFault")
}

func sysMap(v unsafe.Pointer, n uintptr, reserved bool, sysStat *uint64) {
	mSysStatInc(sysStat, n)

	vmo, err := mx.NewVMO(uint64(n), 0)
	if err != nil {
		println("runtime: mx.NewVMO failed, n=", n, " err=", err.Error())
		exit(2)
	}

	off := uint64(uintptr(v) - sysVMARBaseAddr)
	flags := mx.VMFlagPermRead | mx.VMFlagPermWrite | mx.VMFlagSpecificOverwrite
	res, err := sysVMAR.Map(off, vmo, 0, uint64(n), flags)
	if err != nil {
		println("runtime: vmar map failed, off=", off, " n=", n, " err=", err.Error())
		exit(2)
	}
	if res != uintptr(v) {
		println("runtime: vmar map resulted in an unexpected address ", res, "v=", uintptr(v))
		exit(2)
	}
	vmo.Close()
}
