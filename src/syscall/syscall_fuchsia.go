// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import (
	"io"
	"sync"
	"unsafe"

	"syscall/mx"
	"syscall/mx/mxio"
	"syscall/mx/mxio/rio"
)

var (
	Stdin  = 0
	Stdout = 1
	Stderr = 2
)

const (
	O_RDONLY = 0x0
	O_WRONLY = 0x1
	O_RDWR   = 0x2

	O_CREAT     = 00000100
	O_EXCL      = 00000200
	O_NOCTTY    = 00000400
	O_TRUNC     = 00001000
	O_APPEND    = 00002000
	O_NONBLOCK  = 00004000
	O_DSYNC     = 00010000
	O_SYNC      = 04010000
	O_RSYNC     = 04010000
	O_DIRECTORY = 00200000
	O_NOFOLLOW  = 00400000
	O_CLOEXEC   = 02000000
)

const (
	S_IFMT  = 0000170000
	S_IFDIR = 0000040000
	S_IFREG = 0000100000
)

// TODO(crawshaw): generate a zerrors file once cgo is working
const (
	EPERM        = Errno(0x01)
	ENOENT       = Errno(0x02)
	EINTR        = Errno(0x04)
	EIO          = Errno(0x05)
	EACCES       = Errno(0x0d)
	EEXIST       = Errno(0x11)
	ENOTDIR      = Errno(0x14)
	EISDIR       = Errno(0x15)
	EINVAL       = Errno(0x16)
	ESPIPE       = Errno(0x1d)
	EPIPE        = Errno(0x20)
	ENAMETOOLONG = Errno(0x24)
	ENOTEMPTY    = Errno(0x27)
	ENOPROTOOPT  = Errno(0x5c)
	EOPNOTSUPP   = Errno(0x5f)
	ENOTSUP      = EOPNOTSUPP
	EFUCHSIA     = Errno(0xfa)
)

const (
	SOMAXCONN = 0x80
)

type Stat_t struct {
	Dev  uint64
	Ino  uint64
	Size uint64
}

func Getpid() (pid int) {
	// TODO: is ProcHandle process-wide or OS-wide?
	return int(mx.ProcHandle)
}

func Getppid() (ppid int) {
	panic("syscall.Getppid TODO")
}

func Getuid() (uid int) {
	panic("syscall.Getuid TODO")
}

func Geteuid() (uid int) {
	panic("syscall.Geteuid TODO")
}

func Getgid() (uid int) {
	panic("syscall.Getgid TODO")
}

func Getegid() (uid int) {
	panic("syscall.Getegid TODO")
}

func Getgroups() (gids []int, err error) {
	panic("syscall.Getgroups TODO")
}

func Exit(code int) {
	mx.Sys_exit(code)
}

func Getpagesize() int { return 4096 }

const ImplementsGetwd = false

func Getwd() (wd string, err error) {
	panic("syscall.Getwd TODO")
}

func Chdir(path string) (err error) {
	// TODO: if cgo enabled, change the working directory there too.
	f, err := openPath(path, 0, 0)
	if err != nil {
		return err
	}
	cwdMu.Lock()
	cwd = f
	cwdMu.Unlock()
	return nil
}

func Mkdir(path string, mode uint32) (err error) {
	f, err := openPath(path, O_CREAT|O_EXCL|O_RDWR, mode&0777|S_IFDIR)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func Rmdir(path string) (err error) {
	return Unlink(path)
}

func Unlink(path string) (err error) {
	dir, file := splitpath(path)

	f, err := openPath(dir, 0, 0)
	if err != nil {
		return err
	}
	err = rio.UnlinkAt(f, file)
	f.Close()
	return err
}

func splitpath(path string) (dir, file string) {
	i := len(path) - 1
	for i >= 0 && path[i] != '/' {
		i--
	}
	dir = path[:i+1]
	file = path[i+1:]
	return dir, file
}

func Rename(oldpath, newpath string) (err error) {
	if oldpath == "" || newpath == "" {
		return EINVAL
	}

	var f mxio.MXIO
	if oldpath[0] == '/' && newpath[0] == '/' {
		// both paths are absolute: rename relative to root
		rootMu.Lock()
		defer rootMu.Unlock()
		f = root
	} else if oldpath[0] != '/' && newpath[0] != '/' {
		// both paths are relative: rename relative to cwd
		cwdMu.Lock()
		defer cwdMu.Unlock()
		f = cwd
	} else {
		// mixed absolute & relative paths: unsupported
		return EOPNOTSUPP
	}

	buf := make([]byte, len(oldpath)+len(newpath)+2)
	copy(buf, oldpath)
	copy(buf[len(oldpath)+1:], newpath)
	_, err = f.Misc(rio.OpRename, buf, nil)
	return err
}

func Fchdir(fd int) (err error) {
	panic("syscall.Fchdir TODO")
}

func CloseOnExec(r int) {
	panic("syscall.CloseOnExec TODO")
}

var (
	rootMu sync.Mutex
	root   mxio.MXIO
)

var (
	cwdMu sync.Mutex
	cwd   mxio.MXIO
)

var (
	stdioMu sync.Mutex
	stdio   mxio.MXIO
)

const fdsOff = 1e6 // move outside the fd range of unistd.c  MXIO

var (
	fdsMu sync.Mutex
	fds   []mxio.MXIO
)

func Read(fd int, p []byte) (n int, err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	// An io.Reader can return n < len(p) and io.EOF.
	// But this is the POSIX syscall read(3), which
	// returns nil on eof.
	n, err = f.Read(p)
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func init() {
	var err error

	rootMu.Lock()
	root, err = rio.New([]mx.Handle{mx.RootHandle})
	if err != nil {
		println("syscall: failed to create mxio root: ", err.Error())
	}
	rootMu.Unlock()

	cwdMu.Lock()
	cwd, err = rio.New([]mx.Handle{mx.CwdHandle})
	if err != nil {
		println("syscall: failed to create mxio cwd: ", err.Error())
	}
	cwdMu.Unlock()

	stdioMu.Lock()
	stdio, err = rio.New([]mx.Handle{mx.StdioHandle})
	if err != nil {
		println("syscall: failed to create mxio stdio: ", err.Error())
	}
	stdioMu.Unlock()
}

func Write(fd int, p []byte) (n int, err error) {
	switch fd {
	case Stdout, Stderr:
		stdioMu.Lock()
		stdio.Write(p)
		stdioMu.Unlock()
	default:
		fdsMu.Lock()
		f := fds[fd-fdsOff]
		fdsMu.Unlock()
		return f.Write(p)
	}
	return 0, EINVAL
}

func Seek(fd int, offset int64, whence int) (off int64, err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f.Seek(offset, whence)
}

func Close(fd int) (err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fds[fd-fdsOff] = nil
	fdsMu.Unlock()
	return f.Close()
}

func openPath(path string, mode int, perm uint32) (f mxio.MXIO, err error) {
	if path == "" || path[0] != '/' {
		cwdMu.Lock()
		f, err = cwd.Open(path, int32(mode), perm)
		cwdMu.Unlock()
	} else {
		rootMu.Lock()
		f, err = root.Open(path, int32(mode), perm)
		rootMu.Unlock()
	}
	return f, err
}

func Open(path string, mode int, perm uint32) (fd int, err error) {
	f, err := openPath(path, mode, perm)
	if err != nil {
		println("syscall.Open: path=", path, ", err=", err.Error())
		return -1, err
	}
	return OpenMXIO(f), nil
}

func OpenMXIO(f mxio.MXIO) (fd int) {
	fdsMu.Lock()
	i := -1
	for i = 0; i < len(fds); i++ {
		if fds[i] == nil {
			fds[i] = f
			break
		}
	}
	if i == len(fds) {
		fds = append(fds, f)
	}
	fdsMu.Unlock()

	return i + fdsOff
}

func Ioctl(fd int, op uint32, in []byte) ([]byte, []mx.Handle, error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f.Ioctl(op, in)
}

func Fstat(fd int, stat *Stat_t) (err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	attr, err := rio.Stat(f)
	if err != nil {
		return err
	}
	stat.Dev = 0
	stat.Ino = attr.Inode
	stat.Size = attr.Size
	return nil
}

func Stat(path string, stat *Stat_t) (err error) {
	fd, err := Open(path, O_RDONLY, 0)
	if err != nil {
		return err
	}
	err = Fstat(fd, stat)
	if err2 := Close(fd); err == nil {
		err = err2
	}
	return err
}

func Lstat(path string, stat *Stat_t) (err error) {
	// TODO: adjust when there are symlinks
	return Stat(path, stat)
}

func Pread(fd int, p []byte, offset int64) (n int, err error) {
	if fd == Stdout || fd == Stderr {
		return 0, ESPIPE
	}

	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	// An io.Reader can return n < len(p) and io.EOF.
	// But this is the POSIX syscall read(3), which
	// returns nil on eof.
	n, err = f.ReadAt(p, offset)
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func Pwrite(fd int, p []byte, offset int64) (n int, err error) {
	if fd == Stdout || fd == Stderr {
		return 0, ESPIPE
	}

	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f.WriteAt(p, offset)
}

func Truncate(path string, length int64) (err error) {
	panic("syscall.Truncate TODO")
}
func Link(oldpath string, newpath string) (err error) {
	return EOPNOTSUPP // no fuchsia support yet
}
func Symlink(oldpath string, newpath string) (err error) {
	return EOPNOTSUPP // no fuchsia support yet
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

type errDirentEmpty struct{}

func (errDirentEmpty) Error() string {
	return "syscall.Readdirnames: remoteio readdir returned empty dirent"
}

type errDirentTooLarge struct{}

func (errDirentTooLarge) Error() string {
	return "syscall.Readdirnames: remoteio readdir returned oversized dirent"
}

func Readdirnames(fd int, max int) (names []string, err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	const dirBufsize = rio.MessageChunkSize
	var bufback [dirBufsize]byte

	for max != 0 {
		buf := bufback[:]
		n, err := f.Misc(rio.OpReaddir, nil, buf)
		if err != nil {
			return nil, err
		}
		buf = buf[:n:n]
		if n <= 0 {
			break // EOF
		}

		const sizeofSize = 4
		for max != 0 && len(buf) >= sizeofSize {
			d := (*Dirent)(unsafe.Pointer(&buf[0]))
			if d.Size == 0 {
				return nil, errDirentEmpty{}
			}
			if int(d.Size) > len(buf) {
				return nil, errDirentTooLarge{}
			}
			buf = buf[d.Size:]
			bytes := (*[dirBufsize]byte)(unsafe.Pointer(&d.Name[0]))[:d.Size-8]
			name := string(bytes[:clen(bytes)])
			if name == "." || name == ".." {
				continue
			}
			max--
			names = append(names, name)
		}
	}

	return names, nil
}

type Dirent struct {
	Size uint32
	Type uint32
	Name [1]byte
}
