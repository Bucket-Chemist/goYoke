//go:build windows

package process

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"golang.org/x/sys/windows"
)

// jobs tracks Job Object handles by PID for group kill support.
var jobs sync.Map

// Kill sends a kill signal to the process. On Windows, sig is ignored.
func Kill(pid int, sig syscall.Signal) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// KillGroup kills the process group via Job Object if tracked, falling back to Kill.
func KillGroup(pid int, sig syscall.Signal) error {
	if handle, ok := jobs.LoadAndDelete(pid); ok {
		h := handle.(windows.Handle)
		err := windows.TerminateJobObject(h, 1)
		windows.CloseHandle(h) //nolint:errcheck
		return err
	}
	return Kill(pid, sig)
}

// Exists reports whether a process with the given pid is alive.
func Exists(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	windows.CloseHandle(h) //nolint:errcheck
	return true
}

// GroupExists reports whether a tracked job object exists for pid.
func GroupExists(pid int) bool {
	_, ok := jobs.Load(pid)
	return ok
}

// TrackProcess creates a Job Object for pid and stores it for later group kill.
// Best-effort: callers should ignore the returned error.
func TrackProcess(pid int) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("CreateJobObject: %w", err)
	}

	h, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		windows.CloseHandle(job) //nolint:errcheck
		return fmt.Errorf("OpenProcess: %w", err)
	}
	defer windows.CloseHandle(h) //nolint:errcheck

	if err := windows.AssignProcessToJobObject(job, h); err != nil {
		windows.CloseHandle(job) //nolint:errcheck
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}

	jobs.Store(pid, job)
	return nil
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
