#!/bin/bash
# This uses the magenta prebuild C++ toolchain

if [ "$MAGENTA" == "" ]; then
	echo "gccwrap.sh: missing MAGENTA env variable" >&2
	exit 1
fi

if [ "$GOARCH" == "amd64" ]; then
	gccbin="$MAGENTA/prebuilt/downloads/x86_64-elf-6.3.0-$(uname)-x86_64/bin/x86_64-elf-gcc"
	magentabuild="$MAGENTA/../out/build-magenta/build-magenta-pc-x86-64"
elif [ "$GOARCH" == "arm64" ]; then
	gccbin="$MAGENTA/prebuilt/downloads/aarch64-elf-6.3.0-$(uname)-x86_64/bin/aarch64-elf-gcc"
	magentabuild="${MAGENTA_BUILD_DIR:-"$MAGENTA/../out/build-magenta/build-magenta-qemu-arm64"}"
else
	echo "gccwrap.sh: unsupported GOARCH: $GOARCH" >&2
	exit 1
fi

compiler=false
extra_args=""

for var in "$@"
do
	if [[ "$var" == "-r" || "$var" == "-c" ]]; then
		compiler=true
	fi
done

if ! $compiler; then
	extra_args="-lc $magentabuild/system/ulib/crt1.o $magentabuild/system/ulib/magenta/libmagenta.so.abi"
fi

exec $gccbin \
	--sysroot=$magentabuild/sysroot \
	-I$magentabuild/sysroot/include \
	-L$magentabuild/sysroot/lib \
	-nostdlib \
	-fuse-ld=gold \
	-fno-use-linker-plugin \
	-Wl,-nostdlib \
	-Wl,--build-id \
	-Wl,-z,max-page-size=4096 \
	-Wl,-z,combreloc \
	-Wl,-z,relro \
	-Wl,-z,now \
	-Wl,-z,text \
	-Wl,--hash-style=gnu \
	-Wl,--eh-frame-hdr \
	-Wl,-dynamic-linker=ld.so.1 \
	$extra_args \
	"$@"

