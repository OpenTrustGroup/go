// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "go_asm.h"
#include "funcdata.h"
#include "textflag.h"

TEXT runtime·threadinit(SB),NOSPLIT,$0
	// Layout new m scheduler stack on os stack.
	// TODO: which os stack? do we need to make a new vmo?

	MOVD	m_g0(R0), g
	MOVD	R0, g_m(g)

	MOVD RSP, R7
	MOVD $(-64*1024)(R7), R0
	MOVD R0, g_stackguard0(g)
	MOVD R0, g_stackguard1(g)
	MOVD R0, (g_stack+stack_lo)(g)
	MOVD R7, (g_stack+stack_hi)(g)

	BL	runtime·mstart(SB)
	MOVD	$0, R0
	MOVD	R0, (R0)	// boom
	UNDEF

TEXT ·cgocallm0(SB),NOSPLIT,$0-20
	MOVD fn+0(FP), R1
	MOVD arg+8(FP), R0

	MOVD RSP, R19 // R19 is callee-save by C
	MOVD g, R21   // R21 is callee-save by C
	MOVD RSP, R7
	ADD  $-32, R7, R7
	AND  $~15, R7
	MOVD R7, RSP  // align stack for C
	BL   R1

	MOVD R19, RSP // restore SP
	MOVD R21, g   // restore g
	RET
