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
  GOOS=fuchsia CC=`pwd`/../misc/fuchsia/clangwrap.sh ../bin/go build junk.go
  ```
  This program can be put in a bootfs and run on qemu with:
  ```
  echo bin/junk=junk > junk.manifest
  fuchsia/buildtools/mkbootfs -o junk.bootfs junk.manifest
  ./scripts/run-magenta-x86-64 -x path/to/fuchsia/third_party/go/src/junk.bootfs
  ```
