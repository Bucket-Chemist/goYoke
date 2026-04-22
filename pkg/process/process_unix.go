//go:build !windows

package process

import (
	"os"
	"syscall"
)

// Kill sends sig to a single process.
func Kill(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}

// KillGroup sends sig to the process group of pid (i.e. -pid).
func KillGroup(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
}

// Exists reports whether the process with the given pid is alive.
func Exists(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return p.Signal(syscall.Signal(0)) == nil
}

// GroupExists reports whether the process group with the given pgid is alive.
func GroupExists(pid int) bool {
	return syscall.Kill(-pid, 0) == nil
}

// NewProcessGroupAttr returns a SysProcAttr that places the child in a new
// process group (Setpgid). Use KillGroup to signal the whole group.
func NewProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// NewSessionAttr returns a SysProcAttr that places the child in a new session
// (Setsid). Used for long-lived agent subprocesses.
func NewSessionAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

// Setsid creates a new session for the calling process.
func Setsid() error {
	_, err := syscall.Setsid()
	return err
}
