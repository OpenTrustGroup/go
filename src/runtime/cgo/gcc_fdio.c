// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build cgo
// +build fuchsia

#include "libcgo.h"

#include <stdlib.h>
#include <zircon/process.h>
#include <zircon/processargs.h>
#include <fdio/io.h>
#include <fdio/namespace.h>
#include <fdio/util.h>

extern char **environ;

// fdio_init matches a Go struct of the same name in os_fuchsia.go.
struct fdio_init {
	zx_handle_t stdioClones[3 * FDIO_MAX_HANDLES];
	uint32_t stdioCloneNumHandles[3];
	uint32_t stdioCloneTypes[3];
	zx_handle_t processSelf;
	zx_handle_t vmarRootSelf;
	int32_t unused;
	int32_t envlen;
	char**  environ;
	int32_t unused2;
	int32_t rootNSNumHandles;
	zx_handle_t *rootNSHandles;
	char** rootNSPaths;
};

void
x_cgo_get_initial_handles(struct fdio_init *handles)
{
	// When running with cgo support, there are two FDIO libraries in the
	// same process. We let the C FDIO start first, then clone its root
	// and use that for Go.
	// TODO: there is some small amount of fdio library state we
	//       have to share between Go and C, in particular, cwd.
	//       Work out how.

	// fdio may have initialized std{in,out,err} with a single fd if the FDIO_FLAG_USE_FOR_STDIO
	// flag was set, or have separate in/out/err fds. We clone fds {0,1,2} to separate fds for use
	// in the Go runtime. The cloned fds aren't installed into fdio's fd table so they are only
	// usable from Go's table.
	zx_status_t r;
	for (int i = 0; i < 3; ++i) {
		uint32_t handle_types[FDIO_MAX_HANDLES];
		r = fdio_clone_fd(i, 0, &handles->stdioClones[i * FDIO_MAX_HANDLES], handle_types);
		if (r <= 0) {
			printf("runtime/cgo: fdio_clone_fd of stdio[%d] failed: %d\n", i, r);
			handles->stdioCloneNumHandles[i] = 0;
		} else {
			handles->stdioCloneNumHandles[i] = r;
			handles->stdioCloneTypes[i] = PA_HND_TYPE(handle_types[0]);
		}
	}

	handles->processSelf = zx_process_self();
	handles->vmarRootSelf = zx_vmar_root_self();

	for (char** env = environ; *env; env++) {
		handles->envlen++;
	}
	handles->environ = environ;

	fdio_flat_namespace_t* rootNS;
	r = fdio_ns_export_root(&rootNS);
	if (r < 0) {
		handles->rootNSNumHandles = 0;
		handles->rootNSHandles = NULL;
		handles->rootNSPaths = NULL;
	} else {
		handles->rootNSNumHandles = (int32_t)rootNS->count;
		handles->rootNSHandles = rootNS->handle;
		handles->rootNSPaths = (char**)rootNS->path;
	}
}
