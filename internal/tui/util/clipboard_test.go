package util_test

import (
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// TestCopyToClipboard_FunctionExists verifies that CopyToClipboard is
// callable. Real clipboard access is not available in headless CI
// environments, so we only assert the function exists and returns an
// error value (which may be non-nil in a headless environment).
func TestCopyToClipboard_FunctionExists(t *testing.T) {
	// CopyToClipboard must compile and be callable.
	err := util.CopyToClipboard("test string")
	// In a headless environment the clipboard is unavailable; an error is
	// acceptable. We only ensure the function does not panic.
	_ = err
}

// TestReadFromClipboard_FunctionExists verifies that ReadFromClipboard is
// callable. Same caveat about headless CI environments applies.
func TestReadFromClipboard_FunctionExists(t *testing.T) {
	// ReadFromClipboard must compile and be callable.
	text, err := util.ReadFromClipboard()
	// In a headless environment the clipboard is unavailable; an error is
	// acceptable. We only ensure the function does not panic.
	_ = text
	_ = err
}

// TestCopyThenRead_RoundTrip attempts a write-then-read round-trip when the
// clipboard is available. The test is skipped rather than failed in headless
// environments where the clipboard is unavailable.
func TestCopyThenRead_RoundTrip(t *testing.T) {
	const want = "hello clipboard"

	writeErr := util.CopyToClipboard(want)
	if writeErr != nil {
		t.Skipf("clipboard unavailable (write error): %v", writeErr)
	}

	got, readErr := util.ReadFromClipboard()
	if readErr != nil {
		t.Skipf("clipboard unavailable (read error): %v", readErr)
	}

	if got != want {
		t.Errorf("ReadFromClipboard() = %q; want %q", got, want)
	}
}
