// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package mime

func init() {
	osInitMime = initMimeFuchsia
}

func initMimeFuchsia() {
	// TODO: find an equivalent to /etc/mime.types.
}
