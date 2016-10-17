#!/usr/bin/env python
# Copyright 2016 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

# Top-level build script for go.
# Build Go compiler capable of targetting Fuchsia.

import argparse
import os
import stat
import sys
from subprocess import call
from shutil import copyfile
from shutil import rmtree


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--fuchsia-root', help='Path to root of Fuchsia project',
                        required=True)
    parser.add_argument('--go-output-path', help='Path to go ouput binary',
                        required=True)
    parser.add_argument('--go-bootstrap-path', help='Path to bootstrap go',
                        required=True)
    parser.add_argument('--go-host-os', required=True)
    parser.add_argument('--go-host-arch', required=True)
    parser.add_argument('--go-target-arch', required=True)
    args = parser.parse_args()

    old_cwd = os.getcwd()
    go_dir = os.path.dirname(os.path.realpath(__file__))
    os.chdir(os.path.join(go_dir, 'src'))
    os.environ['GOROOT_BOOTSTRAP'] = args.go_bootstrap_path
    os.environ['CGO_ENABLED'] = '1'

    # TODO(smklein): Convert CC_FOR_TARGET from gccwrap to clangwrap.
    os.environ['MAGENTA'] = os.path.join(args.fuchsia_root, 'magenta')
    os.environ['CC_FOR_TARGET'] = os.path.join(go_dir, 'misc/fuchsia/gccwrap.sh')

    # Host
    os.environ['GOHOSTOS'] = args.go_host_os
    os.environ['GOHOSTARCH'] = args.go_host_arch

    # Target
    os.environ['GOOS'] = 'fuchsia'
    os.environ['GOARCH'] = args.go_target_arch

    # TODO(smklein): Remove this hack once crawshaw fixes the corresponding
    # 'cmd/dst' bug.
    rmtree(os.path.join(go_dir, 'pkg/fuchsia_' + args.go_target_arch), True)
    rmtree(os.path.join(go_dir, 'pkg/fuchsia_' + args.go_target_arch + '_shared'), True)

    # Actually compile Go
    r = call(['./make.bash'], env=os.environ)
    if r != 0:
        return r

    # Copy the output binary into a known location
    copyfile(os.path.join(go_dir, 'bin/go'), args.go_output_path)

    # Ensure the output binary is executable
    st = os.stat(args.go_output_path)
    os.chmod(args.go_output_path, st.st_mode | stat.S_IEXEC)

    os.chdir(old_cwd)
    return 0


if __name__ == '__main__':
    main()
