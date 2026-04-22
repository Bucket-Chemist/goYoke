//go:build !windows

package teamrun

import "syscall"

func dup2(oldFd, newFd int) error {
	return syscall.Dup2(oldFd, newFd)
}
