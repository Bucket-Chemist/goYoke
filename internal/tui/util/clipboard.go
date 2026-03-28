// Package util provides shared utility functions for the GOgent-Fortress TUI.
package util

import "github.com/atotto/clipboard"

// CopyToClipboard copies text to the system clipboard.
// Returns an error if the clipboard is unavailable or the write fails.
func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}

// ReadFromClipboard reads text from the system clipboard.
// Returns an error if the clipboard is unavailable or the read fails.
func ReadFromClipboard() (string, error) {
	return clipboard.ReadAll()
}
