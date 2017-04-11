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
)

var (
	flagDepfile    = flag.String("depfile", "", "Stdlib depfile")
	flagRootOutDir = flag.String("root-out-dir", "", "GN variable")
	flagRootGenDir = flag.String("root-gen-dir", "", "GN variable")
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
	buf.WriteString(relGenDir() + "/goroot/stdlib.done:")
	if err := collectDeps(buf, "goroot/src"); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
	}
	if err := ioutil.WriteFile(path, buf.Bytes(), 0664); err != nil {
		log.Fatal("gen depfile stdlib: %v", err)
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

func main() {
	log.SetFlags(0)
	log.SetPrefix("makestdlib.go: ")
	flag.Parse()

	if *flagRootGenDir == "" {
		log.Fatal("must specify -root-gen-dir")
	}

	genDepfileStdlib()

	cmd := exec.Command(filepath.Join(*flagRootGenDir, "goroot/bin/go"), "install", "std")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal("go install std failed")
	}
	donePath := filepath.Join(*flagRootGenDir, "goroot/stdlib.done")
	os.Remove(donePath)
	if err := ioutil.WriteFile(donePath, []byte("done"), 0664); err != nil {
		log.Fatalf("writing done file failed: %v", err)
	}
	os.Exit(0)
}
