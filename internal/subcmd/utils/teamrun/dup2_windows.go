//go:build windows

package teamrun

// dup2 is a no-op on Windows (Ticket B implements real fd redirection).
func dup2(oldFd, newFd int) error { return nil }
