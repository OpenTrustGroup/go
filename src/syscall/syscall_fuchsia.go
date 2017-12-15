// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syscall

import (
	"errors"
	"io"
	"path"
	"strings"
	"sync"
	"unsafe"

	"syscall/zx"
	"syscall/zx/fdio"
)

var (
	Stdin  = 0
	Stdout = 1
	Stderr = 2
)

const (
	FsRightReadable = 0x00000001
	FsRightWritable = 0x00000002
	FsRightAdmin    = 0x00000004
	FsRightRight    = 0x0000FFFF
	FsFlagCreate    = 0x00010000
	FsFlagExclusive = 0x00020000
	FsFlagTruncate  = 0x00040000
	FsFlagDirectory = 0x00080000
	FsFlagAppend    = 0x00100000
	FsFlagNoRemote  = 0x00200000
	FsFlagPath      = 0x00400000
	FsFlagDescribe  = 0x00800000
)

const (
	O_RDONLY   = 0x0
	O_WRONLY   = 0x1
	O_RDWR     = 0x2
	O_PIPELINE = 0x80000000

	// Flags which align with ZXIO_FS_*
	O_ADMIN     = FsRightAdmin
	O_CREAT     = FsFlagCreate
	O_EXCL      = FsFlagExclusive
	O_TRUNC     = FsFlagTruncate
	O_DIRECTORY = FsFlagDirectory
	O_APPEND    = FsFlagAppend
	O_NOREMOTE  = FsFlagNoRemote
	O_PATH      = FsFlagPath

	FdioAlignedFlags = O_ADMIN | O_CREAT | O_EXCL | O_TRUNC | O_DIRECTORY | O_APPEND |
		O_NOREMOTE | O_PATH

	// Flags which do not align with ZXIO_FS_*
	O_NONBLOCK  = 0x00000010
	O_DSYNC     = 0x00000020
	O_SYNC      = 0x00000040 | O_DSYNC
	O_RSYNC     = O_SYNC
	O_NOFOLLOW  = 0x00000080
	O_CLOEXEC   = 0x00000100
	O_NOCTTY    = 0x00000200
	O_ASYNC     = 0x00000400
	O_DIRECT    = 0x00000800
	O_LARGEFILE = 0x00001000
	O_NOATIME   = 0x00002000
	O_TMPFILE   = 0x00004000
)

func FdioFlagsToZxio(flags uint32) (zflags uint32) {
	perms := uint32(O_RDONLY | O_WRONLY | O_RDWR)
	switch flags & perms {
	case O_RDONLY:
		zflags |= FsRightReadable
	case O_WRONLY:
		zflags |= FsRightWritable
	case O_RDWR:
		zflags |= FsRightReadable | FsRightWritable
	}

	if (flags & O_PIPELINE) == 0 {
		zflags |= FsFlagDescribe
	}

	zflags |= (flags & FdioAlignedFlags)
	return zflags
}

const (
	S_IFMT  = 0000170000
	S_IFDIR = 0000040000
	S_IFREG = 0000100000
)

// TODO(crawshaw): generate a zerrors file once cgo is working
const (
	EPERM         = Errno(0x01)
	ENOENT        = Errno(0x02)
	EINTR         = Errno(0x04)
	EIO           = Errno(0x05)
	EBADF         = Errno(0x09)
	EACCES        = Errno(0x0d)
	EEXIST        = Errno(0x11)
	ENOTDIR       = Errno(0x14)
	EISDIR        = Errno(0x15)
	EINVAL        = Errno(0x16)
	EMFILE        = Errno(0x18)
	ESPIPE        = Errno(0x1d)
	ENAMETOOLONG  = Errno(0x24)
	ENOTEMPTY     = Errno(0x27)
	ENOPROTOOPT   = Errno(0x5c)
	EOPNOTSUPP    = Errno(0x5f)
	ENOTSUP       = EOPNOTSUPP
	EADDRINUSE    = Errno(0x62)
	EADDRNOTAVAIL = Errno(0x63)
	ETIMEDOUT     = Errno(0x6e)
	ECONNABORTED  = Errno(0x67)
	EHOSTUNREACH  = Errno(0x71)
	EFUCHSIA      = Errno(0xfa)
)

var EPIPE error = Errno(0x20)

const (
	SOMAXCONN = 0x80
)

type Stat_t struct {
	Dev        uint64
	Ino        uint64
	Size       uint64
	CreateTime uint64
	ModifyTime uint64
}

func Getpid() (pid int) {
	// TODO: is ProcHandle process-wide or OS-wide?
	return int(zx.ProcHandle)
}

func Getppid() (ppid int) {
	return 0
}

func Getuid() (uid int) {
	return 0
}

func Geteuid() (uid int) {
	return 0
}

func Getgid() (uid int) {
	return 0
}

func Getegid() (uid int) {
	return 0
}

func Getgroups() (gids []int, err error) {
	return nil, errors.New("Getgroups unimplemented")
}

func Exit(code int) {
	zx.Sys_process_exit(code)
}

const ImplementsGetwd = true

func Getwd() (wd string, err error) {
	cwdMu.Lock()
	wd = cwdStr
	cwdMu.Unlock()
	return wd, nil
}

func Chdir(dir string) (err error) {
	// TODO: if cgo enabled, change the working directory there too.
	f, err := OpenPath(dir, 0, 0)
	if err != nil {
		return err
	}
	cwdMu.Lock()
	cwd = f
	if path.IsAbs(dir) {
		cwdStr = path.Clean(dir)
	} else {
		cwdStr = path.Join(cwdStr, dir)
	}
	cwdMu.Unlock()
	return nil
}

func Mkdir(path string, mode uint32) (err error) {
	f, err := OpenPath(path, O_CREAT|O_EXCL|O_RDWR, mode&0777|S_IFDIR)
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

	f, err := OpenPath(dir, 0, 0)
	if err != nil {
		return err
	}
	err = fdio.UnlinkAt(f, file)
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

func Rename(oldpath, newpath string) error {
	return twoPathOp(oldpath, newpath, fdio.OpRename)
}

func Link(oldpath, newpath string) error {
	return twoPathOp(oldpath, newpath, fdio.OpLink)
}

func twoPathOp(oldpath, newpath string, op uint32) (err error) {
	if oldpath == "" || newpath == "" {
		return EINVAL
	}

	olddir, oldfile := splitpath(oldpath)
	newdir, newfile := splitpath(newpath)

	old, err := OpenPath(olddir, O_DIRECTORY, 0)
	if err != nil {
		return err
	}
	defer old.Close()
	new, err := OpenPath(newdir, O_DIRECTORY, 0)
	if err != nil {
		return err
	}
	defer new.Close()

	token, err := new.Ioctl(fdio.IoctlVFSGetTokenFS, nil, nil)
	if err != nil {
		return err
	}
	buf := make([]byte, len(oldfile)+len(newfile)+2)
	copy(buf, oldfile)
	copy(buf[len(oldfile)+1:], newfile)
	_, err = old.Misc(op, 0, buf, nil, token)
	return err
}

func Fsync(fd int) (err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	_, err = f.Misc(fdio.OpSync, 0, nil, nil, nil)
	return err
}

func Fchdir(fd int) (err error) {
	return errors.New("Fchdir unimplemented")
}

func CloseOnExec(r int) {
	panic("syscall.CloseOnExec TODO")
}

var (
	rootMu sync.Mutex
	root   fdio.FDIO
)

var (
	cwdMu  sync.Mutex
	cwd    fdio.FDIO
	cwdStr string
)

var (
	nsMu sync.Mutex
	ns   *fdio.NS
)

var (
	stdioMu sync.Mutex
	stdin   fdio.FDIO
	stdout  fdio.FDIO
	stderr  fdio.FDIO
)

const fdsOff = 1e6 // move outside the fd range of unistd.c  FDIO

var (
	fdsMu sync.Mutex
	fds   []fdio.FDIO
)

func Read(fd int, p []byte) (n int, err error) {
	if fd == Stdin {
		stdioMu.Lock()
		n, err = stdin.Read(p)
		stdioMu.Unlock()
	} else {
		if fd < fdsOff {
			return 0, EINVAL
		}
		fdsMu.Lock()
		f := fds[fd-fdsOff]
		fdsMu.Unlock()

		n, err = f.Read(p)
	}
	// An io.Reader can return n < len(p) and io.EOF.
	// But this is the POSIX syscall read(3), which
	// returns nil on eof.
	if err == io.EOF {
		err = nil
	}
	return n, err
}

func init() {
	EPIPE = zx.EPIPE

	var err error

	nsMu.Lock()
	ns, err = fdio.NewNSFromMap(zx.RootNSMap)
	nsMu.Unlock()
	if err != nil {
		println("syscall: failed to create namespace: ", err.Error())
	}

	rootMu.Lock()
	root, err = ns.OpenRoot()
	rootMu.Unlock()
	if err != nil {
		println("syscall: failed to create root directory from namespace: ", err.Error())
	}

	cwdMu.Lock()
	var found bool
	cwdStr, found = Getenv("PWD")
	if !found {
		cwdStr = "/"
	} else {
		// Ensure the incoming cwd is absolute and cleaned
		cwdStr = path.Join("/", cwdStr)
	}
	cwd = root
	if cwdStr != "/" {
		cwd, err = OpenPath(cwdStr, 0, 0)
	}
	cwdMu.Unlock()
	if err != nil {
		println("syscall: failed to create fdio cwd: ", err.Error())
	}

	initStdio := func(i int) (fdio.FDIO, error) {
		switch zx.StdioHandleTypes[i] {
		case 0:
			return nil, EINVAL
		case fdio.HandleTypeRemote:
			return fdio.NewRIO(zx.StdioHandles[i][:])
		case fdio.HandleTypePipe:
			return fdio.NewPipe([]zx.Handle{zx.StdioHandles[i][0]})
		case fdio.HandleTypeLogger:
			return fdio.NewLogger(zx.StdioHandles[i][0])
		default:
			println("syscall: unknown handle type for stdio: " + itoa(zx.StdioHandleTypes[i]))
			return nil, EINVAL
		}
	}
	stdioMu.Lock()
	stdin, err = initStdio(0)
	if err != nil {
		println("syscall: failed to create fdio stdin: ", err.Error())
	}
	stdout, err = initStdio(1)
	if err != nil {
		println("syscall: failed to create fdio stdiout: ", err.Error())
	}
	stderr, err = initStdio(2)
	if err != nil {
		println("syscall: failed to create fdio stderr: ", err.Error())
	}
	stdioMu.Unlock()
}

func Write(fd int, p []byte) (n int, err error) {
	switch fd {
	case Stdout:
		stdioMu.Lock()
		n, err := stdout.Write(p)
		stdioMu.Unlock()
		return n, err
	case Stderr:
		stdioMu.Lock()
		n, err := stderr.Write(p)
		stdioMu.Unlock()
		return n, err
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
	if fd < fdsOff {
		return EINVAL
	}
	fdsMu.Lock()
	if fd-fdsOff > len(fds) {
		fdsMu.Unlock()
		return EINVAL
	}
	f := fds[fd-fdsOff]
	fds[fd-fdsOff] = nil
	fdsMu.Unlock()
	return f.Close()
}

func OpenPath(p string, mode int, perm uint32) (f fdio.FDIO, err error) {
	if strings.Contains(p, "\x00") {
		return nil, EINVAL
	}

	p = path.Clean(p)

	if p[0] != '/' {
		cwdMu.Lock()
		c := cwd
		cwdMu.Unlock()
		f, err = c.Open(p, FdioFlagsToZxio(uint32(mode)), perm)
	} else {
		rootMu.Lock()
		r := root
		rootMu.Unlock()
		f, err = r.Open(p[1:], FdioFlagsToZxio(uint32(mode)), perm)
	}
	return f, err
}

func Open(path string, mode int, perm uint32) (fd int, err error) {
	if path == "" {
		return -1, EINVAL
	}
	f, err := OpenPath(path, mode, perm)
	if err != nil {
		return -1, err
	}
	return OpenFDIO(f), nil
}

func OpenAt(fdParent int, path string, mode int, perm uint32) (fd int, err error) {
	parent := FDIOForFD(fdParent)
	if parent == nil {
		return -1, EBADF
	}
	f, err := parent.Open(path, FdioFlagsToZxio(uint32(mode)), perm)
	if err != nil {
		return -1, err
	}
	return OpenFDIO(f), nil
}

func FDIOForFD(fd int) fdio.FDIO {
	switch fd {
	case Stdin:
		return stdin
	case Stdout:
		return stdout
	case Stderr:
		return stderr
	}
	if fd < fdsOff {
		return nil
	}
	fdsMu.Lock()
	if fd-fdsOff > len(fds) {
		fdsMu.Unlock()
		return nil
	}
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f
}

func OpenFDIO(f fdio.FDIO) (fd int) {
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

func Ioctl(fd int, op uint32, in, out []byte) ([]zx.Handle, error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f.Ioctl(op, in, out)
}

func IoctlSetHandle(fd int, op uint32, in zx.Handle) error {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()
	return f.IoctlSetHandle(op, in)
}

func Fstat(fd int, stat *Stat_t) (err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	attr, err := fdio.Stat(f)
	if err != nil {
		return err
	}
	stat.Dev = uint64(attr.Mode)
	stat.Ino = attr.Inode
	stat.Size = attr.Size
	stat.CreateTime = attr.CreateTime
	stat.ModifyTime = attr.ModifyTime
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

func Ftruncate(fd int, length int64) (err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	_, err = f.Misc(fdio.OpTruncate, length, nil, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func Truncate(path string, length int64) (err error) {
	fd, err := Open(path, O_WRONLY, 0)
	if err != nil {
		return err
	}
	err = Ftruncate(fd, length)
	if err2 := Close(fd); err == nil {
		err = err2
	}
	return err
}

func Symlink(oldpath string, newpath string) (err error) {
	return EOPNOTSUPP // no fuchsia support yet
}

func UtimesNano(path string, ts []Timespec) (err error) {
	f, err := OpenPath(path, 0, 0)
	if err != nil {
		return err
	}
	vn := &fdio.Vnattr{
		Valid:      fdio.AttrMtime,
		ModifyTime: uint64(TimespecToNsec(ts[1])),
	}
	_, err = f.Misc(fdio.OpSetAttr, 0, (*[unsafe.Sizeof(fdio.Vnattr{})]byte)(unsafe.Pointer(vn))[:], nil, nil)
	return err
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

func ReadDirent(fd int, buf []byte) (n int, err error) {
	fdsMu.Lock()
	f := fds[fd-fdsOff]
	fdsMu.Unlock()

	return f.Misc(fdio.OpReaddir, 0, nil, buf, nil)
}

func direntIno(buf []byte) (uint64, bool) {
	// Inodes aren't exposed through Fuchsia; just lie.
	// Use '1' instead of '0', since '0' means "entry not filled" on some operating systems.
	return 1, true
}

func direntReclen(buf []byte) (uint64, bool) {
	return readInt(buf, unsafe.Offsetof(Dirent{}.Size), unsafe.Sizeof(Dirent{}.Size))
}

func direntNamlen(buf []byte) (uint64, bool) {
	reclen, ok := direntReclen(buf)
	if !ok {
		return 0, false
	}
	return reclen - uint64(unsafe.Offsetof(Dirent{}.Name)), true
}

type Dirent struct {
	Size uint32
	Type uint32
	Name [1]byte
}
