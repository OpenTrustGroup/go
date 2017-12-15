// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The makefuchsia.go script builds a GOROOT in the out/ build directory
// using sources from third_party/go.
//
// When the GOROOT is first built, this script also calls make.bash.
// On subsquent modifications to source files in third_party/go, the
// source files are copied into the out/ directory but the toolchain
// is not rebuilt as the content-based hashing in the Go tool will
// rebuild any stale packages.
package main

import (
	"bytes"
	"crypto/sha256"
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

func rm(path string) {
	didSomething = true
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}
}

func hashfile(path string) string {
	h := sha256.New()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "does_not_exist(" + path + ")"
		}
		log.Fatal(err)
	}
	defer f.Close()

	if _, err := io.Copy(h, f); err != nil {
		log.Fatalf("hash %s: %v", path, err)
	}

	return string(h.Sum(nil))
}

var didSomething = false

// copyfile copies src to dst if the two files have different contents.
func copyfile(src, dst string) {
	if hashfile(src) == hashfile(dst) {
		return
	}
	didSomething = true

	rm(dst)
	srcfile, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcfile.Close()

	f, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(f, srcfile); err != nil {
		log.Fatalf("cp %s %s: %v", src, dst, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("cp %s %s: close: %v", src, dst, err)
	}

	fi, err := os.Stat(src)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(dst, fi.Mode()); err != nil {
		log.Fatal(err)
	}
}

func collectDeps(w io.Writer) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fmt.Fprintf(w, " %s", path)
		return err
	}
	srcGoroot := join(*flagFuchsiaGoroot, "src")
	return filepath.Walk(srcGoroot, fn)
}

func genDepfileStdlib() {
	// Change anything under src/*, and we copy those changes into out/.
	path := *flagDepfile
	if path == "" {
		return
	}
	os.Remove(path)
	buf := new(bytes.Buffer)
	buf.WriteString(relGenDir() + "/goroot.done:")
	if err := collectDeps(buf); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
	}
	depbytes := buf.Bytes()

	h := sha256.New()
	h.Write(depbytes)
	newhash := string(h.Sum(nil))

	if hashfile(path) == newhash {
		return // don't overwrite an accurate file
	}

	if err := ioutil.WriteFile(path, depbytes, 0664); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
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

	genDepfileStdlib()

	srcGoroot := *flagFuchsiaGoroot
	dstGoroot := *flagOutGoroot

	src := join(srcGoroot, "src")
	dst := join(dstGoroot, "src")

	mkdir(dst)
	mkdir(join(dstGoroot, "misc"))
	mkdir(join(dstGoroot, "pkg"))
	mkdir(join(dstGoroot, "bin"))
	mkdir(join(dstGoroot, "lib"))

	mkdir(join(dstGoroot, "misc/fuchsia"))
	miscFiles := []string{
		"clangwrap.sh",
		"gccwrap.sh",
		"go_stdlib_tests.json",
	}
	for _, file := range miscFiles {
		copyfile(join(srcGoroot, "misc/fuchsia", file), join(dstGoroot, "misc/fuchsia", file))
	}

	versionFile := join(dstGoroot, "VERSION")
	if _, err := os.Stat(versionFile); err != nil {
		if !os.IsNotExist(err) {
			log.Fatal(err)
		}
		err := ioutil.WriteFile(versionFile, []byte("devel third_party/go fuchsia"), 0664)
		if err != nil {
			log.Fatal(err)
		}
	}

	srcFiles, err := filepath.Glob(join(src, "*"))
	if err != nil {
		log.Fatalf("could not glob source files: %v", err)
	}
	dstFiles, err := filepath.Glob(join(dst, "*"))
	if err != nil {
		log.Fatalf("could not glob destination files: %v", err)
	}

	srcRelNames := make(map[string]bool, len(srcFiles))
	for _, file := range srcFiles {
		srcRelNames[file[len(src)+1:]] = true
	}
	for _, file := range dstFiles {
		if !srcRelNames[file[len(dst)+1:]] {
			rm(file)
		}
	}

	var wg sync.WaitGroup
	for _, file := range srcFiles {
		file := file
		deepCopy := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, "_test.go") && !strings.Contains(path, "os/") {
				return nil
			}
			if filepath.Base(path) == "testdata" {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			dstPath := join(dst, path[len(src)+1:])
			mkdir(filepath.Dir(dstPath))
			copyfile(path, dstPath)
			return nil
		}
		wg.Add(1)
		go func() {
			if err := filepath.Walk(file, deepCopy); err != nil {
				log.Fatalf("goroot deep copy failed: %v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if _, err := os.Stat(join(dstGoroot, "bin", "go")); err != nil {
		cmd := exec.Command(*flagBuiltoolsGo, "env", "GOROOT")
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("cannot find bootstrap GOROOT: %v", err)
		}

		env := os.Environ()
		env = append(env, "GOROOT_BOOTSTRAP="+string(out))
		env = append(env, "CC_FOR_fuchsia_"+goarchTarget+"="+dstGoroot+"/misc/fuchsia/gccwrap.sh")

		// Fresh copy, we need to build the compiler.
		// On subsequent copies, we can rely on "go build" to
		// rebuild any standard library changes.
		cmd = exec.Command(join(dst, "make.bash"))
		cmd.Dir = dst
		cmd.Env = env
		buf := new(bytes.Buffer)
		cmd.Stdout = buf
		cmd.Stderr = buf
		if err := cmd.Run(); err != nil {
			log.Fatalf("make.bash failed: %v\n%s", err, buf.String())
		}
	}

	if didSomething {
		donePath := join(*flagRootGenDir, "goroot.done")
		rm(donePath)
		if err := ioutil.WriteFile(donePath, []byte("done"), 0664); err != nil {
			log.Fatalf("writing goroot.done failed: %v", err)
		}
	}
}
