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
	MOVQ	DI, R8

	MOVQ	m_g0(R8), DX		// g
	MOVQ	R8, g_m(DX)

	// Layout new m scheduler stack on os stack.
	// TODO: which os stack? do we need to make a new vmo?
	LEAQ	(-8*1024+104)(SP), BX
	MOVQ	BX, g_stackguard0(DX)
	MOVQ	BX, g_stackguard1(DX)
	MOVQ	BX, (g_stack+stack_lo)(DX)
	MOVQ	SP, (g_stack+stack_hi)(DX)

	PUSHQ	R8

	// Set up tls.
	LEAQ	m_tls(R8), DI
	CALL 	runtime·settls(SB)

	get_tls(CX)

	POPQ	R8
	MOVQ	m_g0(R8), DX
	MOVQ	DX, g(CX)

	CLD				// convention is D is always left cleared
	CALL	runtime·stackcheck(SB)	// clobbers AX,CX
	CALL	runtime·mstart(SB)
	// TODO CALL	internal/syscall/fuchsia·Sys_magenta_thread_exit(SB)
	RET

// set tls base to DI
// Note that this code is only used when CGO_ENABLED=0.
// Not using internal/syscall/fuchsia so we can avoid setting up a G.
//
// TODO: make VSDO call directly here, avoid the syscall.
TEXT runtime·settls(SB),NOSPLIT,$8
	ADDQ   $8, DI	// ELF wants to use -8(FS)
	PUSHQ  DI
	LEAQ   0(SP), DX // prctl on magenta takes a pointer, not the value
	MOVQ $0, DI
	MOVQ $0, SI
	MOVQ $0xff00ff00000035, AX // MAGENTA_SYSCALL_MAGIC | thread_arch_prctl (53)
	SYSCALL
	CMPQ	AX, $0xfffffffffffff001
	JLS	2(PC)
	MOVL	$0xf1, 0xf1  // crash
	POPQ  DI
	RET
