#!/bin/bash
# This uses the zircon prebuild C++ toolchain

if [ "$ZIRCON" == "" ]; then
	echo "gccwrap.sh: missing ZIRCON env variable" >&2
	exit 1
fi

if [ "$GOARCH" == "amd64" ]; then
	gccbin="$ZIRCON/prebuilt/downloads/x86_64-elf-6.3.0-$(uname)-x86_64/bin/x86_64-elf-gcc"
	zirconbuild="$ZIRCON/../out/build-zircon/build-zircon-pc-x86-64"
elif [ "$GOARCH" == "arm64" ]; then
	gccbin="$ZIRCON/prebuilt/downloads/aarch64-elf-6.3.0-$(uname)-x86_64/bin/aarch64-elf-gcc"
	zirconbuild="${ZIRCON_BUILD_DIR:-"$ZIRCON/../out/build-zircon/build-zircon-qemu-arm64"}"
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
	extra_args="-lc $zirconbuild/system/ulib/crt1.o $zirconbuild/system/ulib/zircon/libzircon.so.abi"
fi

exec $gccbin \
	--sysroot=$zirconbuild/sysroot \
	-I$zirconbuild/sysroot/include \
	-L$zirconbuild/sysroot/lib \
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

