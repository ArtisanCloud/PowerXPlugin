//go:build darwin || linux

package publisher

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

// Every operation stays relative to pinned directory descriptors. No symlink
// component may redirect state, temporary files, credentials, or locks.
type directory struct {
	file *os.File
	path string
}

func openDirectory(root string) (*directory, error) {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	return &directory{file: os.NewFile(uintptr(fd), root), path: root}, nil
}
func (d *directory) child(name string, private bool) (*directory, error) {
	if err := unix.Mkdirat(int(d.file.Fd()), name, 0700); err != nil && !errors.Is(err, unix.EEXIST) {
		return nil, failure(StorageFailure, err)
	}
	return d.existing(name, private)
}
func (d *directory) existing(name string, private bool) (*directory, error) {
	fd, err := unix.Openat(int(d.file.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, failure(NotFound, err)
	}
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	f := os.NewFile(uintptr(fd), name)
	info, err := f.Stat()
	if err != nil || (private && info.Mode().Perm()&0077 != 0) {
		f.Close()
		return nil, failure(StorageFailure, err)
	}
	return &directory{file: f, path: filepath.Join(d.path, name)}, nil
}
func (d *directory) close() error { return d.file.Close() }
func (d *directory) read(name string, limit int64) ([]byte, error) {
	fd, err := unix.Openat(int(d.file.Fd()), name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, failure(NotFound, err)
	}
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	f := os.NewFile(uintptr(fd), name)
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > limit {
		return nil, failure(InvalidRecord, err)
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, failure(InvalidRecord, err)
	}
	return data, nil
}
func (d *directory) names() ([]string, error) {
	fd, err := unix.Openat(int(d.file.Fd()), ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	f := os.NewFile(uintptr(fd), ".")
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	return names, nil
}
func (d *directory) write(name string, data []byte, replace bool) error {
	tmp := ".pending-" + uuid.NewString()
	fd, err := unix.Openat(int(d.file.Fd()), tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if err != nil {
		return failure(StorageFailure, err)
	}
	f := os.NewFile(uintptr(fd), tmp)
	defer unix.Unlinkat(int(d.file.Fd()), tmp, 0)
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return failure(StorageFailure, err)
	}
	if replace {
		err = unix.Renameat(int(d.file.Fd()), tmp, int(d.file.Fd()), name)
	} else {
		err = unix.Linkat(int(d.file.Fd()), tmp, int(d.file.Fd()), name, 0)
	}
	if errors.Is(err, unix.EEXIST) {
		return failure(Conflict, err)
	}
	if err != nil {
		return failure(StorageFailure, err)
	}
	if err = d.file.Sync(); err != nil {
		return failure(StorageFailure, err)
	}
	return nil
}
func (d *directory) remove(name string) error {
	if err := unix.Unlinkat(int(d.file.Fd()), name, 0); err != nil && !errors.Is(err, unix.ENOENT) {
		return failure(StorageFailure, err)
	}
	return d.file.Sync()
}
func (d *directory) lock() (func(), error) {
	return d.lockFile("executor.lock")
}
func (d *directory) lockFile(name string) (func(), error) {
	fd, err := unix.Openat(int(d.file.Fd()), name, unix.O_RDWR|unix.O_CREAT|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, failure(StorageFailure, err)
	}
	f := os.NewFile(uintptr(fd), name)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, failure(StorageFailure, err)
	}
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, unix.EWOULDBLOCK) {
			return nil, failure(Conflict, err)
		}
		return nil, failure(StorageFailure, err)
	}
	return func() { _ = unix.Flock(fd, unix.LOCK_UN); _ = f.Close() }, nil
}

func (d *directory) openRead(name string) (*os.File, error) {
	fd, err := unix.Openat(int(d.file.Fd()), name, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, failure(ArtifactChanged, err)
	}
	f := os.NewFile(uintptr(fd), name)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		f.Close()
		return nil, failure(ArtifactChanged, err)
	}
	return f, nil
}
func (d *directory) current() bool {
	path, err := filepath.EvalSymlinks(d.path)
	if err != nil || path != d.path {
		return false
	}
	a, err := os.Stat(d.path)
	if err != nil {
		return false
	}
	b, err := d.file.Stat()
	return err == nil && os.SameFile(a, b)
}
