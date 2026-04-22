//go:build windows

package filelock

// Lock is a no-op stub on Windows (Ticket B implements real LockFileEx).
func Lock(fd int) error { return nil }

// LockShared is a no-op stub on Windows.
func LockShared(fd int) error { return nil }

// LockSharedNonBlock is a no-op stub on Windows.
func LockSharedNonBlock(fd int) error { return nil }

// Unlock is a no-op stub on Windows.
func Unlock(fd int) error { return nil }
