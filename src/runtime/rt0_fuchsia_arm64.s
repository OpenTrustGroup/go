// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include "textflag.h"

TEXT _rt0_arm64_fuchsia(SB),NOSPLIT,$-8
	MOVD 0(RSP),  R0 // argc
	ADD  $8, RSP, R1 // argv
	BL   main(SB)

TEXT main(SB),NOSPLIT,$-8
	MOVD $runtime·rt0_go(SB), R2
	BL   R2
