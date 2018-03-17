// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The makefuchsia.go script builds a GOROOT in the out/ build directory
// using sources from third_party/go.
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
)

var (
	flagBuiltoolsGo   = flag.String("buildtools-go", "", "Path to go tool used for bootstrap")
	flagFuchsiaGoroot = flag.String("fuchsia-goroot", "", "Path to Fuchsia's source GOROOT")
	flagDepfile       = flag.String("depfile", "", "Stdlib depfile")
	flagRootOutDir    = flag.String("root-out-dir", "", "GN variable")
	flagRootGenDir    = flag.String("root-gen-dir", "", "GN variable")
	flagTargetCPU     = flag.String("target-cpu", "", "GN variable")
	flagOutGoroot     = flag.String("out-goroot", "", "Path to the output GOROOT")
)

func relGenDir() string {
	return (*flagRootGenDir)[len(*flagRootOutDir)+1:]
}

func mkdir(path ...string) {
	if err := os.MkdirAll(join(path...), 0755); err != nil {
		log.Fatal(err)
	}
}

func join(path ...string) string { return filepath.Join(path...) }

// copyfile copies src to dst if the two files have different contents.
func copyfile(src, dst string) {
	srcfile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcfile.Close()

	fi, err := srcfile.Stat()
	if err != nil {
		log.Fatal(err)
	}

	mkdir(filepath.Dir(dst))
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, fi.Mode())
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(f, srcfile); err != nil {
		log.Fatalf("cp %s %s: %v", src, dst, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("cp %s %s: close: %v", src, dst, err)
	}
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

// allFilesIn recursively walks a set of files, returning the entire tree in
// paths relative to the given root. A map is returned for convenience. If
// prefix is given, only files that match the prefix are considered.
func allFilesIn(root string, paths map[string]struct{}, prefix ...string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		for _, pfx := range prefix {
			if strings.HasPrefix(path, join(root, pfx)) {
				relPath := strings.TrimPrefix(path, root)[1:]
				paths[relPath] = struct{}{}
				return err
			}
		}
		return err
	})
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("makefuchsia.go: ")
	flag.Parse()

	if *flagRootGenDir == "" {
		log.Fatal("must specify -root-gen-dir")
	}

	if *flagFuchsiaGoroot == "" {
		log.Fatal("must specify -fuchsia-goroot when building toolchain")
	}

	goarchTarget := convCPU(*flagTargetCPU)
	if goarchTarget == "" {
		log.Fatalf("unsupported target CPU: %s", *flagTargetCPU)
	}

	srcGoroot := *flagFuchsiaGoroot
	dstGoroot := *flagOutGoroot

	srcFiles := map[string]struct{}{}
	if err := allFilesIn(join(srcGoroot), srcFiles, "src/", "misc/fuchsia/"); err != nil {
		log.Fatal(err)
	}

	dstFiles := map[string]struct{}{}
	if _, err := os.Stat(join(dstGoroot)); err == nil {
		if err := allFilesIn(join(dstGoroot), dstFiles, ""); err != nil {
			log.Fatal(err)
		}
	}

	for f := range dstFiles {
		if _, ok := srcFiles[f]; !ok {
			// try to preserve the bin/ and pkg/ trees, as they can be used to do faster
			// rebuilds of the toolchain.
			if strings.HasPrefix(f, "src/") || strings.HasPrefix(f, "misc/fuchsia/") {
				os.Remove(filepath.Join(dstGoroot, f))
			}
		}
	}

	for f := range srcFiles {
		src, dst := join(srcGoroot, f), join(dstGoroot, f)
		copyfile(src, dst)
	}

	versionFile := join(dstGoroot, "VERSION")
	if err := ioutil.WriteFile(versionFile, []byte("devel third_party/go fuchsia"), 0664); err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(*flagBuiltoolsGo, "env", "GOROOT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cannot find bootstrap GOROOT: %v", err)
	}

	env := os.Environ()
	env = append(env, "GOROOT_BOOTSTRAP="+strings.TrimSpace(string(out)))
	env = append(env, "CC_FOR_fuchsia_"+goarchTarget+"="+dstGoroot+"/misc/fuchsia/gccwrap.sh")

	cmd = exec.Command(join(dstGoroot, "src", "make.bash"))
	cmd.Args = []string{"--no-clean"}
	cmd.Dir = join(dstGoroot, "src")
	cmd.Env = env
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		log.Fatalf("make.bash failed: %v\n%s", err, buf.String())
	}

	depfile, err := os.Create(*flagDepfile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(depfile, "%s/goroot.done:", relGenDir())
	for f := range srcFiles {
		fmt.Fprintf(depfile, " %s", join(srcGoroot, f))
	}
	fmt.Fprintf(depfile, " %s", join(srcGoroot, "makefuchsia.go"))

	donePath := join(*flagRootGenDir, "goroot.done")
	if err := ioutil.WriteFile(donePath, []byte("done"), 0664); err != nil {
		log.Fatalf("writing goroot.done failed: %v", err)
	}
}
