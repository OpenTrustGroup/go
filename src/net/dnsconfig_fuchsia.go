// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

// TODO: use dnsconfig_unix when 127.0.0.1:53 is a valid DNS server.

package net

import (
	"os"
	"time"
)

var (
	defaultNS   = []string{"8.8.8.8:53"}
	getHostname = os.Hostname // variable for testing
)

type dnsConfig struct {
	servers    []string
	attempts   int
	timeout    time.Duration
	search     []string
	ndots      int
	rotate     bool
	unknownOpt bool
	lookup     []string
	err        error
	mtime      time.Time
	soffset    uint32
}

func dnsReadConfig(filename string) *dnsConfig {
	return &dnsConfig{
		ndots:    1,
		timeout:  5 * time.Second,
		attempts: 2,
		servers:  defaultNS,
	}
}

func (c *dnsConfig) serverOffset() uint32 { return 0 }
