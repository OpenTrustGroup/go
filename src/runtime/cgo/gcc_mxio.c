// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build cgo
// +build fuchsia

#include "libcgo.h"

#include <stdlib.h>
#include <magenta/processargs.h>
#include <mxio/io.h>
#include <mxio/util.h>

void
x_cgo_get_mxio_root(mx_handle_t *h)
{
	// When running with cgo support, there are two MXIO libraries in the
	// same process. We let the C MXIO start first, then clone its root
	// and use that for Go.
	// TODO: there is some small amount of mxio library state we
	//       have to share between Go and C, in particular, cwd.
	//       Work out how.

	uint32_t type;
	mx_status_t r;
	r = mxio_clone_root(h, &type);
	if (r <= 0 || type != MX_HND_TYPE_MXIO_ROOT) {
		printf("runtime/cgo: mxio_clone_root failed: %d\n", r);
		*h = 0;
	}
}
