// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"path"
	"strings"
	"syscall/mx"
)

type NS struct {
	m map[string]mx.Handle
}

func NewNS() *NS {
	return &NS{m: make(map[string]mx.Handle)}
}

func NewNSFromMap(m map[string]mx.Handle) (*NS, error) {
	if m == nil {
		return nil, mx.Error{Status: mx.ErrInvalidArgs, Text: "namespace.NewFromNSMAP"}
	}
	return &NS{m: m}, nil
}

func (ns *NS) Bind(p string, h mx.Handle) {
	ns.m[p] = h
}

func (ns *NS) Connect(p string, h mx.Handle) error {
	if p[0] != '/' {
		return mx.Error{Status: mx.ErrNotFound, Text: "namespace.Connect"}
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
	return mx.Error{Status: mx.ErrNotFound, Text: "namespace.Connect"}
}

// TODO: support directory creation
