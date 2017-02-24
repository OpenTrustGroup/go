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
	mx_handle_t rootClone;
	mx_handle_t stdioClones[3 * MXIO_MAX_HANDLES];
	uint32_t stdioCloneNumHandles[3];
	uint32_t stdioCloneTypes[3];
	mx_handle_t cwd;
	mx_handle_t processSelf;
	mx_handle_t vmarRootSelf;
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
	// mxio may have initialized std{in,out,err} with a single fd if the MXIO_FLAG_USE_FOR_STDIO
	// flag was set, or have separate in/out/err fds. We clone fds {0,1,2} to separate fds for use
	// in the Go runtime. The cloned fds aren't installed into mxio's fd table so they are only
	// usable from Go's table.
	for (int i = 0; i < 3; ++i) {
		uint32_t handle_types[MXIO_MAX_HANDLES];
		r = mxio_clone_fd(i, 0, &handles->stdioClones[i * MXIO_MAX_HANDLES], handle_types);
		if (r <= 0) {
			printf("runtime/cgo: mxio_clone_fd of stdio[%d] failed: %d\n", i, r);
			handles->stdioCloneNumHandles[i] = 0;
		} else {
			handles->stdioCloneNumHandles[i] = r;
			handles->stdioCloneTypes[i] = MX_HND_INFO_TYPE(handle_types[0]);
		}
	}
	type = 0;
	r = mxio_clone_cwd(&handles->cwd, &type);
	if (r <= 0) {
		printf("runtime/cgo: mxio_clone_cwd failed: %d\n", r);
		handles->cwd = 0;
	}

	handles->processSelf = mx_process_self();
	handles->vmarRootSelf = mx_vmar_root_self();

	for (char** env = environ; *env; env++) {
		handles->envlen++;
	}
	handles->environ = environ;
}
