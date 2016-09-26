#!/bin/bash
# This uses the magenta prebuild C++ toolchain

if [ "$MAGENTA" == "" ]; then
	echo "clangwrap.sh: missing MAGENTA env variable" >&2
	exit 1
fi

gccbin="$MAGENTA/prebuilt/downloads/x86_64-elf-6.2.0-$(uname)-x86_64/bin/x86_64-elf-gcc"
magentabuild="$MAGENTA/build-magenta-pc-x86-64"

compiler=false
extra_args=""

for var in "$@"
do
	if [[ "$var" == "-r" ]]; then
		compiler=true
	fi
done

if ! $compiler; then
	extra_args="-lc $magentabuild/ulib/crt1.o"
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

