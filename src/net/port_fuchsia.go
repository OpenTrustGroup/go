// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

import "errors"

func goLookupPort(network, service string) (port int, err error) {
	return -1, errors.New("net: port lookup not supported yet on fuchsia")
}
