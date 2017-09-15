// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"path"
	"strings"
	"syscall/zx"
)

type NS struct {
	m map[string]zx.Handle
}

func NewNS() *NS {
	return &NS{m: make(map[string]zx.Handle)}
}

func NewNSFromMap(m map[string]zx.Handle) (*NS, error) {
	if m == nil {
		return nil, zx.Error{Status: zx.ErrInvalidArgs, Text: "namespace.NewFromNSMAP"}
	}
	return &NS{m: m}, nil
}

func (ns *NS) Bind(p string, h zx.Handle) {
	ns.m[p] = h
}

func (ns *NS) Connect(p string, h zx.Handle) error {
	if p[0] != '/' {
		return zx.Error{Status: zx.ErrNotFound, Text: "namespace.Connect"}
	}
	node := p
	for {
		svcHandle := ns.m[node]
		if svcHandle.IsValid() {
			rest := strings.TrimPrefix(p, node)
			rest = strings.TrimPrefix(rest, "/")
			return ServiceConnectAt(svcHandle, rest, h)
		}
		if node == "/" {
			break
		}
		node = path.Dir(node)
	}
	return zx.Error{Status: zx.ErrNotFound, Text: "namespace.Connect"}
}

// TODO: support directory creation
