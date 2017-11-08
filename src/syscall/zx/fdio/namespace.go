// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build fuchsia

package fdio

import (
	"path"
	"strings"
	"syscall/zx"
	"unsafe"
)

func splitPath(path string) (name string, remaining string) {
	slash := strings.IndexByte(path, '/')
	if slash == 0 || slash == len(path)-1 {
		return "", ""
	} else if slash == -1 {
		return path, ""
	} else {
		return path[:slash], path[slash+1:]
	}
}

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

func (ns *NS) OpenRoot() (FDIO, error) {
	root := &vnode{}
	for path, h := range ns.m {
		if path == "" || path[0] != '/' {
			return nil, zx.Error{Status: zx.ErrBadPath, Text: "fdio.OpenRoot"}
		}
		vn := root
		path = path[1:]
		for path != "" {
			name, remaining := splitPath(path)
			vn = vn.getOrAddChild(name)
			path = remaining
		}
		if vn.remote != nil {
			return nil, zx.Error{Status: zx.ErrAlreadyExists, Text: "fdio.OpenRoot"}
		}
		vn.remote = &zx.Channel{h}
	}
	return newDir(root)
}

type vnode struct {
	child  *vnode
	parent *vnode
	next   *vnode
	remote *zx.Channel
	name   string
}

func (vn *vnode) getOrAddChild(name string) *vnode {
	child := vn.getChild(name)
	if child == nil {
		child = &vnode{
			parent: vn,
			next:   vn.child,
			name:   name,
		}
		vn.child = child
	}
	return child
}

func (vn *vnode) getChild(name string) *vnode {
	for vn = vn.child; vn != nil; vn = vn.next {
		if vn.name == name {
			return vn
		}
	}
	return nil
}

type dir struct {
	vn  *vnode
	rio *RemoteIO
}

func newDir(vn *vnode) (*dir, error) {
	d := &dir{vn: vn}
	if vn.remote != nil {
		d.rio = NewRIOFromChannel(vn.remote)
	}
	return d, nil
}

func (d *dir) Read(data []byte) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) ReadAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) Write(data []byte) (n int, err error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) WriteAt(data []byte, off int64) (int, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) Seek(offset int64, whence int) (int64, error) {
	return 0, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) Close() error {
	d.vn = nil
	d.rio = nil
	return nil
}

func (d *dir) Open(path string, flags uint32, mode uint32) (FDIO, error) {
	vn := d.vn

	if path == "." || path == "" {
		return newDir(vn)
	}

	originalPath := path

	var savedVn *vnode
	var savedPath string

	for {
		name, remaining := splitPath(path)

		if name == "" {
			return nil, zx.Error{Status: zx.ErrBadPath, Text: path}
		}

		child := vn.getChild(name)

		if child != nil {
			if remaining == "" {
				// We've resolved all the path segments and found the vnode we're
				// looking for.
				if child.remote != nil {
					handles, err := CloneHandle(child.remote)
					if err != nil {
						return nil, err
					}
					return NewRIO(handles)
				} else {
					return newDir(child)
				}
			} else {
				// We've got more path to resolve.
				vn = child
				path = remaining

				// If this child has a remote file system, we might hand off to
				// this remote if we don't find anything more specific below
				// this vnode.
				if child.remote != nil {
					savedVn = vn
					savedPath = path
				}

				continue
			}
		}

		// We've reached the end of our local vnode structure with more
		// path to resolve.

		if vn.remote != nil {
			// We've got a remote file system at this vnode. We can hand off
			// directly to that file system.
			return OpenHandle(vn.remote, path, flags, mode)
		}

		// We saw a remote file system earlier in our walk that might have this
		// path. Let's hand off to it.
		if savedVn != nil {
			return OpenHandle(savedVn.remote, savedPath, flags, mode)
		}

		// There is no remote file system to resolve the path against,
		// which means we failed to find what we were looking for.
		return nil, zx.Error{Status: zx.ErrNotFound, Text: originalPath}
	}
}

func (d *dir) Clone() ([]zx.Handle, error) {
	if d.rio != nil {
		return nil, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
	}
	return d.rio.Clone()
}

func (d *dir) Ioctl(op uint32, in, out []byte) ([]zx.Handle, error) {
	return nil, zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) IoctlSetHandle(op uint32, in zx.Handle) error {
	return zx.Error{Status: zx.ErrNotSupported, Text: "fdio.dir"}
}

func (d *dir) Misc(op uint32, off int64, in, out []byte, handles []zx.Handle) (n int, err error) {
	switch op {
	// TODO(abarth): Implement OpReaddir
	case OpStat:
		attr := Vnattr{
			Mode:  VtypeDir | VtypeIRUSR,
			Inode: 1,
			Nlink: 1,
		}
		if unsafe.Sizeof(attr) > uintptr(len(out)) {
			return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "fdio.dir"}
		}
		*(*Vnattr)(unsafe.Pointer(&out[0])) = attr
		return int(unsafe.Sizeof(attr)), nil
	default:
		return 0, zx.Error{Status: zx.ErrInvalidArgs, Text: "fdio.dir"}
	}
}
