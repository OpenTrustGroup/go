#!/bin/sh
# This uses the Fuchsia clang and Magenta sysroot.
set -eu

if [ "$GOHOSTARCH" == "amd64" ]; then
	HOST_ARCH="x86_64"
elif [ "$GOHOSTARCH" == "arm64" ]; then
	HOST_ARCH="aarch64"
else
	echo "clangwrap.sh: missing/invalid GOHOSTARCH env variable" >&2
	exit 1
fi

if [ "$GOHOSTOS" == "linux" ]; then
	HOST_OS="linux"
elif [ "$GOHOSTOS" == "darwin" ]; then
	HOST_OS="darwin"
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

if [ "$SDK" == "" ]; then
	echo "clangwrap.sh: missing fuchsia SDK env variable" >&2
	exit 1
fi

exec ${SDK}/toolchain/clang+llvm-${HOST_ARCH}-${HOST_OS}/bin/clang \
	--target=${TARGET_ARCH}-fuchsia \
	--sysroot=${SDK}/../out/sysroot/${TARGET_ARCH}-fuchsia \
	"$@"
