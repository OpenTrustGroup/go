#!/bin/bash
# This uses the zircon prebuild C++ toolchain

if [ -z "${ZIRCON}" ]; then
	echo "gccwrap.sh: missing ZIRCON env variable" >&2
	exit 1
fi

# TODO(mcgrathr): Remove this fallback when everything has percolated through
# to passing sysroot instead of build-dir.
if [ -z "$ZIRCON_SYSROOT" -a -n "$ZIRCON_BUILD_DIR" ]; then
	ZIRCON_SYSROOT="${ZIRCON_BUILD_DIR}/sysroot"
fi

zirconsysroot="${ZIRCON_SYSROOT}"
if [ -z "${zirconsysroot}" ]; then
	echo "gccwrap.sh: missing ZIRCON_SYSROOT env variable" >&2
	exit 1
fi

if [ "$GOARCH" == "amd64" ]; then
	gccbin="$ZIRCON/prebuilt/downloads/gcc/bin/x86_64-elf-gcc"
elif [ "$GOARCH" == "arm64" ]; then
	gccbin="$ZIRCON/prebuilt/downloads/gcc/bin/aarch64-elf-gcc"
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
	extra_args="-lc $zirconsysroot/lib/Scrt1.o $zirconsysroot/lib/libzircon.so"
fi

exec $gccbin \
	--sysroot=$zirconsysroot \
	-I$zirconsysroot/include \
	-L$zirconsysroot/lib \
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
