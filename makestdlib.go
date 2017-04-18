// Copyright 2017 The Fuchsia Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

// The makestdlib.go script builds the standard library for GOROOT.
// It is used as part of the GN+ninja build.
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
	flagDepfile       = flag.String("depfile", "", "Stdlib depfile")
	flagRootOutDir    = flag.String("root-out-dir", "", "GN variable")
	flagRootGenDir    = flag.String("root-gen-dir", "", "GN variable")
	flagTargetCPU     = flag.String("target-cpu", "", "GN variable")
	flagBootstrapPath = flag.String("bootstrap-path", "", "Path to bootstrap Go installation")
)

func genDepfileStdlib() {
	// Change anything under src/*, and we rebuild the standard library.
	// The command we use, "go install std", is very efficient about what
	// packages it rebuilds.
	path := *flagDepfile
	if path == "" {
		return
	}
	os.Remove(path)
	buf := new(bytes.Buffer)
	buf.WriteString(relGenDir() + "/gostdlib.done:")
	if err := collectDeps(buf); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
	}
	if err := ioutil.WriteFile(path, buf.Bytes(), 0664); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
	}
}

func relGenDir() string {
	return (*flagRootGenDir)[len(*flagRootOutDir)+1:]
}

func collectDeps(w io.Writer) error {
	fn := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		fmt.Fprintf(w, " %s", path)
		return err
	}
	return filepath.Walk(filepath.Join(*flagBootstrapPath, "src"), fn)
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
	log.SetPrefix("makestdlib.go: ")
	flag.Parse()

	if *flagRootGenDir == "" {
		log.Fatal("must specify -root-gen-dir")
	}

	goarchTarget := convCPU(*flagTargetCPU)
	if goarchTarget == "" {
		log.Fatalf("unsupported target CPU: %s", *flagTargetCPU)
	}

	genDepfileStdlib()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		// Generate the stdlib for the host.
		// It is changed by third_party/go srcs being symlinked into buildtools.
		cmd := exec.Command(filepath.Join(*flagBootstrapPath, "bin/go"), "install", "std")
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"GOROOT=" + *flagBootstrapPath,
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal("go install std for host failed")
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		// Now generate the stdlib for the Fuchsia target.
		cmd := exec.Command(filepath.Join(*flagBootstrapPath, "bin/go"), "install", "std")
		cmd.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"GOOS=fuchsia",
			"GOARCH=" + goarchTarget,
			"GOROOT=" + *flagBootstrapPath,
			"CGO_ENABLED=1",
			"CC=" + filepath.Join(*flagBootstrapPath, "misc/fuchsia/gccwrap.sh"),
			"MAGENTA=" + filepath.Join(*flagFuchsiaRoot, "magenta"), // used by gccwrap.sh
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatal("go install std for target failed")
		}
		wg.Done()
	}()

	wg.Wait()

	donePath := filepath.Join(*flagRootGenDir, "gostdlib.done")
	os.Remove(donePath)
	if err := ioutil.WriteFile(donePath, []byte("done"), 0664); err != nil {
		log.Fatalf("writing done file failed: %v", err)
	}
	os.Exit(0)
}
