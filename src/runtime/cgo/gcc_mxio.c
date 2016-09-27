// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build cgo
// +build fuchsia

#include "libcgo.h"

#include <stdlib.h>
#include <magenta/process.h>
#include <magenta/processargs.h>
#include <mxio/io.h>
#include <mxio/util.h>

extern char **environ;

// mxio_init matches a Go struct of the same name in os_fuchsia.go.
struct mxio_init {
	int32_t rootClone;
	int32_t stdioClone;
	int32_t cwd;
	int32_t processSelf;
	int32_t unused;
	int32_t envlen;
	char**  environ;
};

void
x_cgo_get_mxio_root(struct mxio_init *handles)
{
	// When running with cgo support, there are two MXIO libraries in the
	// same process. We let the C MXIO start first, then clone its root
	// and use that for Go.
	// TODO: there is some small amount of mxio library state we
	//       have to share between Go and C, in particular, cwd.
	//       Work out how.

	uint32_t type;
	mx_status_t r;
	r = mxio_clone_root(&handles->rootClone, &type);
	if (r <= 0 || type != MX_HND_TYPE_MXIO_ROOT) {
		printf("runtime/cgo: mxio_clone_root failed: %d\n", r);
		handles->rootClone = 0;
	}
	type = 0;
	r = mxio_clone_fd(1, 0, &handles->stdioClone, &type);
	if (r <= 0) {
		printf("runtime/cgo: mxio_clone_fd of stdio failed: %d\n", r);
		handles->stdioClone = 0;
	}
	type = 0;
	r = mxio_clone_cwd(&handles->cwd, &type);
	if (r <= 0) {
		printf("runtime/cgo: mxio_clone_cwd failed: %d\n", r);
		handles->cwd = 0;
	}

	handles->processSelf = mx_process_self();

	for (char** env = environ; env; env++) {
		handles->envlen++;
	}
	handles->environ = environ;
}
