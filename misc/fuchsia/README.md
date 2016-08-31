Go on Fuchsia
=========

To build a toolchain on Linux for Fuchsia, first follow the
instructions on

  https://fuchsia.googlesource.com/manifest/

Then,
* Set the SDK environment variable to the location of the Fuchsia SDK:
  If you downloaded fuchsia to your ${HOME} directory:
  ```
  export SDK=${HOME}/fuchsia/buildtools
  ```

* Identify where a host version of go exists, with:
  ```
  export GOROOT_BOOTSTRAP=`path to directory containing go`
  ```
  On my machine, `which go` returned /usr/lib/golang/bin/go, so I set
  GOROOT_BOOTSTRAP=/usr/lib/golang/

* Build Go in `fuchsia/third_party/go/src` using
  ```
  GOOS=fuchsia CC_FOR_TARGET=../misc/fuchsia/clangwrap.sh ./make.bash
  ```

* To build a program, junk.go in go/src, use:
  ```
  GOOS=fuchsia ../bin/go build junk.go
  ```
  This program can be put in a bootfs and run on qemu with:
  ```
  echo bin/junk=junk > junk.manifest
  fuchsia/buildtools/mkbootfs -o junk.bootfs junk.manifest
  ./scripts/run-magenta-x86-64 -x path/to/fuchsia/third_party/go/src/junk.bootfs
  ```
  A cgo variant can be build by adding CGO_ENABLED=1 to both
  the make.bash command and the go build command.

* To build a program using just the magenta prebuild toolchain
  (useful for debugging if the fuchsia sysroot is out of sync
  with your magenta) set the environment variable MAGENTA to
  the location of your magenta and build with the gccwrap.sh
  script:
  ```
  MAGENTA=$HOME/magenta CGO_ENABLED=1 CC_FOR_TARGET=$(pwd)/../misc/fuchsia/gccwrap.sh GOOS=fuchsia ./make.bash
