// Package filelock provides platform-abstracted file locking.
package filelock

import "errors"

// ErrWouldBlock is returned by LockSharedNonBlock when the lock is held by
// another process and cannot be acquired without blocking.
var ErrWouldBlock = errors.New("filelock: would block")
