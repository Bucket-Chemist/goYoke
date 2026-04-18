// Package testdata provides shared test fixtures and helpers for the
// goYoke TUI test suite. Test fixture files are embedded as
// JSON files in this directory and loaded via LoadFixture.
package testdata

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// LoadFixture reads a test fixture file from the testdata directory.
// It uses runtime.Caller to locate the directory relative to this source file,
// so callers from any package can load fixtures without path gymnastics.
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("load fixture %s: runtime.Caller failed", name)
	}
	dir := filepath.Dir(thisFile)
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}
	return data
}
