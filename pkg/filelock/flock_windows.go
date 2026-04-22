//go:build windows

package filelock

import (
	"golang.org/x/sys/windows"
)

func Lock(fd int) error {
	h := windows.Handle(fd)
	var ol windows.Overlapped
	return windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &ol)
}

func LockShared(fd int) error {
	h := windows.Handle(fd)
	var ol windows.Overlapped
	return windows.LockFileEx(h, 0, 0, 1, 0, &ol)
}

func LockSharedNonBlock(fd int) error {
	h := windows.Handle(fd)
	var ol windows.Overlapped
	err := windows.LockFileEx(h, windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ol)
	if err == windows.ERROR_LOCK_VIOLATION {
		return ErrWouldBlock
	}
	return err
}

func Unlock(fd int) error {
	h := windows.Handle(fd)
	var ol windows.Overlapped
	return windows.UnlockFileEx(h, 0, 1, 0, &ol)
}
