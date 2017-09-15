// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"syscall/zx"
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
	err := zx.VMARRoot.Unmap(uintptr(v), uint64(n))
	if err != nil {
		println("runtime: zx.VMAR.Unmap failed, err=", err.Error())
	}
}

func sysMmap(ptr unsafe.Pointer, len uintptr, flags uint32) {
	vmo, err := zx.NewVMO(uint64(len), 0)
	if err != nil {
		println("runtime: zx.NewVMO failed, len=", len, " err=", err.Error())
		exit(2)
	}
	(zx.Handle)(vmo).SetProperty(zx.PropName, []byte("go-sys-mmap"))
	vmflags := zx.VMFlagPermRead | zx.VMFlagPermWrite
	res, err := zx.VMARRoot.Map(0, vmo, 0, uint64(len), vmflags)
	*(*uintptr)(ptr) = res
	vmo.Close()
	if err != nil {
		println("runtime: zx.VMAR.Map failed, err=", err.Error())
		exit(2)
	}
}

var (
	sysVMAR         zx.VMAR
	sysVMARBaseAddr uintptr
)

func sysReserve(v unsafe.Pointer, n uintptr, reserved *bool) unsafe.Pointer {
	if sysVMAR != 0 {
		println("runtime: system vmar already reserved")
		exit(2)
	}

	var err error
	*reserved = true
	flags := zx.VMFlagCanMapSpecific | zx.VMFlagCanMapRead | zx.VMFlagCanMapWrite
	sysVMAR, sysVMARBaseAddr, err = zx.NewVMAR(zx.VMARRoot, 0, uint64(n), flags)
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

	vmo, err := zx.NewVMO(uint64(n), 0)
	if err != nil {
		println("runtime: zx.NewVMO failed, n=", n, " err=", err.Error())
		exit(2)
	}
	(zx.Handle)(vmo).SetProperty(zx.PropName, []byte("go-sys-map"))

	off := uint64(uintptr(v) - sysVMARBaseAddr)
	flags := zx.VMFlagPermRead | zx.VMFlagPermWrite | zx.VMFlagSpecificOverwrite
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
