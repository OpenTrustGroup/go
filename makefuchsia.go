// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The makefuchsia.go script symlinks the src contents from third_party/go
// into the active GOROOT in ./buildtools/{ARCH}/go.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	flagFuchsiaRoot   = flag.String("fuchsia-root", "", "Path to root of the Fuchsia project")
	flagRootGenDir    = flag.String("root-gen-dir", "", "GN variable")
	flagBootstrapPath = flag.String("bootstrap-path", "", "Path to bootstrap Go installation")
)

func mkdir(path ...string) {
	if err := os.MkdirAll(join(path...), 0755); err != nil {
		log.Fatal(err)
	}
}

func join(path ...string) string { return filepath.Join(path...) }

func rm(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}
}

func symlink(src, dst string) {
	if err := os.Symlink(src, dst); err != nil {
		log.Fatal(err)
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("makefuchsia.go: ")
	flag.Parse()

	if *flagRootGenDir == "" {
		log.Fatal("must specify -root-gen-dir")
	}

	if *flagFuchsiaRoot == "" {
		log.Fatal("must specify -fuchsia-root when building toolchain")
	}

	srcGoroot := join(*flagFuchsiaRoot, "third_party/go")
	dstGoroot := *flagBootstrapPath
	src := join(srcGoroot, "src")
	dst := join(dstGoroot, "src")

	// The make.bash script adds these files to the tree.
	// We make a copy of them now, blow away the src tree,
	// shallow symlink what we can, and deep-symlink any directory
	// containing one of these generated files.
	genFiles := []string{
		"cmd/cgo/zdefaultcc.go",
		"cmd/go/zdefaultcc.go",
		"cmd/go/zosarch.go",
		"cmd/internal/obj/zbootstrap.go",
		"go/build/zcgo.go",
		"runtime/internal/sys/zversion.go",
	}
	deepSymlink := map[string]bool{}
	for _, file := range genFiles {
		deepSymlink[file[:strings.Index(file, "/")]] = true
	}
	genFilesContents := map[string][]byte{}
	for _, file := range genFiles {
		b, err := ioutil.ReadFile(join(dstGoroot, "src", file))
		if err != nil {
			log.Fatal(err)
		}
		genFilesContents[file] = b
	}

	rm(dst)
	mkdir(dst)
	rm(join(dstGoroot, "misc/fuchsia"))
	symlink(join(srcGoroot, "misc/fuchsia"), join(dstGoroot, "misc/fuchsia"))

	srcFiles, err := filepath.Glob(join(src, "*"))
	if err != nil {
		log.Fatalf("could not glob source files: %v", err)
	}

	var wg sync.WaitGroup
	for _, file := range srcFiles {
		base := filepath.Base(file)
		if !deepSymlink[base] {
			symlink(file, join(dst, base))
			continue
		}
		file := file
		deepSymlink := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.HasSuffix(path, "go/zdefaultcc.go") {
				return nil // created below
			}
			if filepath.Base(path) == "testdata" {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			dstPath := join(dst, path[len(src)+1:])
			mkdir(filepath.Dir(dstPath))
			symlink(path, dstPath)
			return nil
		}
		wg.Add(1)
		go func() {
			if err := filepath.Walk(file, deepSymlink); err != nil {
				log.Fatalf("goroot deep symlink failed: %v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	for file, contents := range genFilesContents {
		if err := ioutil.WriteFile(join(dstGoroot, "src", file), contents, 0664); err != nil {
			log.Fatal(err)
		}
	}

	donePath := join(*flagRootGenDir, "goroot.done")
	rm(donePath)
	if err := ioutil.WriteFile(donePath, []byte("done"), 0664); err != nil {
		log.Fatalf("writing goroot.done failed: %v", err)
	}
}
