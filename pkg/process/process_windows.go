//go:build windows

package process

import (
	"os"
	"syscall"
)

// Kill sends a kill signal to the process. On Windows, sig is ignored.
func Kill(pid int, sig syscall.Signal) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// KillGroup kills the process group. On Windows, this is equivalent to Kill
// (Job Objects are a Ticket B concern).
func KillGroup(pid int, sig syscall.Signal) error {
	return Kill(pid, sig)
}

// Exists reports whether a process with the given pid is alive.
func Exists(pid int) bool {
	_, err := os.FindProcess(pid)
	return err == nil
}

// GroupExists always returns false on Windows (stub; Ticket B adds real support).
func GroupExists(pid int) bool {
	return false
}

// NewProcessGroupAttr returns a SysProcAttr that creates a new process group
// via CREATE_NEW_PROCESS_GROUP.
func NewProcessGroupAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: 0x00000200} // CREATE_NEW_PROCESS_GROUP
}

// NewSessionAttr returns the same process-group attr on Windows (no Setsid concept).
func NewSessionAttr() *syscall.SysProcAttr {
	return NewProcessGroupAttr()
}

// Setsid is a no-op on Windows.
func Setsid() error {
	return nil
}
