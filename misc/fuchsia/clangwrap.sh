#!/bin/sh
# This uses the Fuchsia clang and Magenta sysroot.

if [ "$SDK" == "" ]; then
	echo "clangwrap.sh: missing fuchsia SDK env variable" >&2
	exit 1
fi

exec $SDK/toolchain/clang+llvm-x86_64-linux/bin/clang \
	--target=x86_64-fuchsia \
	--sysroot=$SDK/sysroot/x86_64-fuchsia \
	"$@"
