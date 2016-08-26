// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "go_asm.h"
#include "go_tls.h"
#include "textflag.h"

// func now() (sec int64, nsec int32)
TEXT time·now(SB),NOSPLIT,$8-12
	MOVQ $0xff00ff00000004, AX // 4
	SYSCALL
	MOVQ AX, CX

	// generated code for
	//	func f(x uint64) (int64, uint32) { return int64(x / 1e6), uint32(x % 1e6) }
	MOVQ $0x431bde82d7b634db, AX
	MULQ CX
	SHRQ $0x12, DX
	MOVQ DX, sec+0(FP)
	IMULQ $0xf4240, DX
	SUBQ DX, CX
	MOVL CX, nsec+8(FP)
	RET

TEXT runtime·threadinit(SB),NOSPLIT,$0
	MOVQ	DI, CX

	MOVQ	m_g0(CX), DX		// g
	MOVQ	CX, g_m(DX)

	// Layout new m scheduler stack on os stack.
	LEAQ	(-8*1024+104)(SP), BX
	MOVQ	BX, g_stackguard0(DX)
	MOVQ	BX, g_stackguard1(DX)
	MOVQ	BX, (g_stack+stack_lo)(DX)
	MOVQ	SP, (g_stack+stack_hi)(DX)

	// Set up tls.
	PUSHQ 	CX
	LEAQ	m_tls(CX), DI
	PUSHQ 	DI
	CALL 	runtime·magenta_settls(SB)
	POPQ 	DI
	POPQ 	CX

	get_tls(CX)

	LEAQ	m_tls(CX), DI
	MOVQ	m_g0(CX), DX
	MOVQ	DX, g(DI)

	CLD				// convention is D is always left cleared

	CALL	runtime·stackcheck(SB)	// clobbers AX,CX
	CALL	runtime·mstart(SB)
	// TODO CALL	internal/syscall/fuchsia·Sys_magenta_thread_exit(SB)
	RET

// set tls base to DI
TEXT runtime·settls(SB),NOSPLIT,$8
	// TODO If there is no cgo, create the mx_tls_root object.
        //MOVD    _cgo_init(SB), R12
        //CMP     $0, R12
        //BEQ     nocgo

	//SUBQ	$0x88, DI // random slot, known to linker
	PUSHQ 	DI
	CALL runtime·magenta_settls(SB)
	POPQ 	DI
	RET
