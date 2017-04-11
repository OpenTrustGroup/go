// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The makefuchsia.go script builds a GOROOT for cross compiling Fuchsia
// binaries from the host. It is used as part of the GN+ninja build.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	flagFuchsiaRoot   = flag.String("fuchsia-root", "", "Path to root of the Fuchsia project")
	flagDepfile       = flag.String("depfile", "", "Toolchain depfile")
	flagRootOutDir    = flag.String("root-out-dir", "", "GN variable")
	flagRootGenDir    = flag.String("root-gen-dir", "", "GN variable")
	flagHostOS        = flag.String("host-os", "", "GN variable")
	flagHostCPU       = flag.String("host-cpu", "", "GN variable")
	flagTargetCPU     = flag.String("target-cpu", "", "GN variable")
	flagBootstrapPath = flag.String("bootstrap-path", "", "Path to bootstrap Go installation")
)

func genDepfileToolchain() {
	// Change any of the src/cmd/* files, and we rebuild the entire toolchain.
	// This is not perfect. The toolchain does depend on other stdlib packages.
	// But in practice it works well, changes to the rest of the stdlib don't
	// have big semantic effects on the compiler/linker/assembler, because
	// these programs are designed to be built with old versions of the stdlib
	// for bootstrapping.
	path := *flagDepfile
	if path == "" {
		return
	}
	os.Remove(path)
	buf := new(bytes.Buffer)
	buf.WriteString(relGenDir() + "/goroot/bin/go:")
	if err := collectDeps(buf, "goroot/src/cmd"); err != nil {
		log.Fatal("gen depfile toolchain: %v", err)
	}
	if err := ioutil.WriteFile(path, buf.Bytes(), 0664); err != nil {
		log.Fatal("gen depfile toolchain: %v", err)
	}
}

func relGenDir() string {
	return (*flagRootGenDir)[len(*flagRootOutDir)+1:]
}

func collectDeps(w io.Writer, path string) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fmt.Fprintf(w, " %s", path)
		return err
	}
	return filepath.Walk(filepath.Join(*flagRootGenDir, path), fn)
}

func genGoroot() {
	if *flagFuchsiaRoot == "" {
		log.Fatal("must specify -fuchsia-root when building toolchain")
	}

	// Setup a minimal GOROOT in a target-specific generated files
	// directory. This ensures that the ninja build for a target
	// does not touch the source directories, an important invariant
	// for building against multiple architectures.
	dstGoroot := filepath.Join(*flagRootGenDir, "goroot")
	if err := os.RemoveAll(dstGoroot); err != nil {
		log.Fatal(err)
	}
	dst := filepath.Join(dstGoroot, "src")
	if err := os.MkdirAll(dst, 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dstGoroot, "misc"), 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(*flagFuchsiaRoot, "third_party/go/misc/fuchsia"), filepath.Join(dstGoroot, "misc/fuchsia")); err != nil {
		log.Fatalf("goroot misc symlink failed: %v", err)
	}
	src := filepath.Join(*flagFuchsiaRoot, "third_party/go/src")
	cmd := exec.Command("git", "log", "-n", "1", "--format=format:fuchsia:%h %cd", "HEAD")
	cmd.Dir = src
	verData, err := cmd.Output()
	if err != nil {
		log.Fatalf("could not get third_party/go git version: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(dstGoroot, "VERSION"), verData, 0664); err != nil {
		log.Fatalf("could not write VERSION: %v", err)
	}

	// The make.bash script adds files to the packages:
	//
	//	src/cmd
	//	src/go/build
	//	src/runtime
	//
	// So we deep symlink these subtrees in the minimal GOROOT.
	// For the rest of the packages, symlinking the top-level
	// directories is enough.
	deepSymlink := map[string]bool{"cmd": true, "go": true, "runtime": true}
	srcFiles, err := filepath.Glob(filepath.Join(src, "*"))
	if err != nil {
		log.Fatalf("could not glob source files: %v", err)
	}

	var wg sync.WaitGroup
	for _, file := range srcFiles {
		base := filepath.Base(file)
		if !deepSymlink[base] {
			//log.Printf("shallow link %s (base=%s)", file, base)
			if err := os.Symlink(file, filepath.Join(dst, base)); err != nil {
				log.Fatalf("goroot symlink failed: %v", err)
			}
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
			if filepath.Base(path) == "testdata" {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			dstPath := filepath.Join(dst, path[len(src)+1:])
			os.MkdirAll(filepath.Dir(dstPath), 0775)
			//log.Printf("deepSymlink path=%s -> dst=%s", path, dstPath)
			return os.Symlink(path, dstPath)
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
}

func convCPU(cpu string) (arch string) {
	switch cpu {
	case "x64":
		return "amd64"
	case "arm64":
		return "arm64"
	}
	return ""
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("makefuchsia.go: ")
	flag.Parse()

	if *flagRootGenDir == "" {
		log.Fatal("must specify -root-gen-dir")
	}

	genGoroot()
	genDepfileToolchain()

	var gohostos string
	switch *flagHostOS {
	case "linux":
		gohostos = "linux"
	case "mac":
		gohostos = "darwin"
	default:
		log.Fatalf("unsupported host OS: %s", *flagHostOS)
	}

	gohostarch := convCPU(*flagHostCPU)
	if gohostarch == "" {
		log.Fatalf("unsupported host CPU: %s", *flagHostCPU)
	}

	goarch := convCPU(*flagTargetCPU)
	if gohostarch == "" {
		log.Fatalf("unsupported target CPU: %s", *flagTargetCPU)
	}

	cmd := exec.Command(filepath.Join(*flagRootGenDir, "goroot/src/make.bash"))
	cmd.Dir = filepath.Join(*flagRootGenDir, "goroot/src")
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"GOROOT_BOOTSTRAP=" + *flagBootstrapPath,
		"CGO_ENABLED=1",
		"FUCHSIA=" + *flagFuchsiaRoot,
		"MAGENTA=" + filepath.Join(*flagFuchsiaRoot, "magenta"),
		"CC_FOR_TARGET=" + filepath.Join(*flagRootGenDir, "goroot/misc/fuchsia/gccwrap.sh"),
		"GOHOSTOS=" + gohostos,
		"GOHOSTARCH=" + gohostarch,
		"GOOS=fuchsia",
		"GOARCH=" + goarch,
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("make.bash failed: %s", string(out))
		os.Exit(1)
	}
}
