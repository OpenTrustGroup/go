// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package net

func probeIPv4Stack() bool {
	return false
}

func probeIPv6Stack() (supportsIPv6, supportsIPv4map bool) {
	return false, false
}
