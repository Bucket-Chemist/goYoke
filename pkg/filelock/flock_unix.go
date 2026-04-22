//go:build !windows

package filelock

import (
	"errors"
	"syscall"
)

// Lock acquires an exclusive lock on the file descriptor fd, blocking until
// the lock is available.
func Lock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_EX)
}

// LockShared acquires a shared lock on the file descriptor fd.
func LockShared(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_SH)
}

// LockSharedNonBlock attempts a non-blocking shared lock. Returns ErrWouldBlock
// if the lock is held exclusively by another process.
func LockSharedNonBlock(fd int) error {
	err := syscall.Flock(fd, syscall.LOCK_SH|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return ErrWouldBlock
	}
	return err
}

// Unlock releases the lock on the file descriptor fd.
func Unlock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}
