#!/bin/bash
# This uses the Fuchsia clang and Magenta sysroot.
set -eu

if [ "$GOHOSTARCH" == "amd64" ]; then
	HOST_ARCH="x64"
elif [ "$GOHOSTARCH" == "arm64" ]; then
	HOST_ARCH="arm64"
else
	echo "clangwrap.sh: missing/invalid GOHOSTARCH env variable" >&2
	exit 1
fi

if [ "$GOHOSTOS" == "linux" ]; then
	HOST_OS="linux"
elif [ "$GOHOSTOS" == "darwin" ]; then
	HOST_OS="mac"
else
	echo "clangwrap.sh: missing/invalid GOHOSTOS env variable" >&2
	exit 1
fi

if [ "$GOARCH" == "amd64" ]; then
	TARGET_ARCH="x86_64"
elif [ "$GOHOSTARCH" == "arm64" ]; then
	TARGET_ARCH="aarch64"
else
	echo "clangwrap.sh: missing/invalid GOARCH env variable" >&2
	exit 1
fi

if [ "$FUCHSIA" == "" ]; then
	echo "clangwrap.sh: missing FUCHSIA env variable" >&2
	exit 1
fi

exec ${FUCHSIA}/buildtools/${HOST_OS}-${HOST_ARCH}/clang/bin/clang \
	--target=${TARGET_ARCH}-fuchsia \
	--sysroot=${FUCHSIA}/out/sysroot/${TARGET_ARCH}-fuchsia \
	"$@"
