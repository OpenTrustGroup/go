// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#include <pthread.h>
#include <errno.h>
#include <string.h> // strerror
#include <signal.h>
#include <stdlib.h>
#include "libcgo.h"

// TODO(teisenbe): Remove this when the VMAR compat syscalls go away
#define WITHOUT_COMPAT_SYSCALLS 1
#include <magenta/syscalls.h>

static void* threadentry(void*);
static void (*setg_gcc)(void*);

void
x_cgo_init(G* g, void (*setg)(void*))
{
	setg_gcc = setg;
}

void
_cgo_sys_thread_start(ThreadStart *ts)
{
	// TODO: switch to threads.h.
	pthread_attr_t attr;
	pthread_t p;
	size_t size;
	int err;

	pthread_attr_init(&attr);
	pthread_attr_getstacksize(&attr, &size);
	// leave stacklo=0 and set stackhi=size; mstack will do the rest.
	ts->g->stackhi = size;
	err = pthread_create(&p, &attr, threadentry, ts);

	if (err != 0) {
		fatalf("pthread_create failed: %s", strerror(err));
	}
}

static void*
threadentry(void *v)
{
	ThreadStart ts;

	ts = *(ThreadStart*)v;
	free(v);
	setg_gcc((void*)ts.g);

	crosscall_amd64(ts.fn);
	return 0;
}
