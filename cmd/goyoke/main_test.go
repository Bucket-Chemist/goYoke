package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/subcmd"
)

func TestDispatchByArgv0ForwardsArgs(t *testing.T) {
	reg := subcmd.NewRegistry()
	var gotArgs []string
	reg.Register("sample", func(_ context.Context, args []string, _ io.Reader, _ io.Writer) error {
		gotArgs = append([]string(nil), args...)
		return nil
	})

	handled, err := dispatchByArgv0(
		context.Background(),
		[]string{"goyoke-sample", "--format", "json"},
		reg,
		bytes.NewReader(nil),
		io.Discard,
	)
	if err != nil {
		t.Fatalf("dispatchByArgv0 returned error: %v", err)
	}
	if !handled {
		t.Fatal("expected argv0 dispatch to be handled")
	}
	if len(gotArgs) != 2 || gotArgs[0] != "--format" || gotArgs[1] != "json" {
		t.Fatalf("expected forwarded args [--format json], got %v", gotArgs)
	}
}

func TestWriteRuntimeTempFileUsesCreateTemp(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	firstPath, err := writeRuntimeTempFile("goyoke-hooks-*.json", []byte("first"))
	if err != nil {
		t.Fatalf("writeRuntimeTempFile first call failed: %v", err)
	}
	defer os.Remove(firstPath)

	secondPath, err := writeRuntimeTempFile("goyoke-hooks-*.json", []byte("second"))
	if err != nil {
		t.Fatalf("writeRuntimeTempFile second call failed: %v", err)
	}
	defer os.Remove(secondPath)

	if firstPath == secondPath {
		t.Fatalf("expected unique temp file names, both calls returned %q", firstPath)
	}
	if filepath.Dir(firstPath) != tmpDir || filepath.Dir(secondPath) != tmpDir {
		t.Fatalf("expected temp files under %q, got %q and %q", tmpDir, firstPath, secondPath)
	}

	data, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first temp file: %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("first temp file content = %q, want %q", string(data), "first")
	}

	info, err := os.Stat(firstPath)
	if err != nil {
		t.Fatalf("stat first temp file: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("temp file permissions = %o, want 0600", info.Mode().Perm())
	}
}
