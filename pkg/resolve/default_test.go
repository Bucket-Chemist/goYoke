package resolve

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestDefault_WithoutSetDefault(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	r := Default()
	if r == nil {
		t.Fatal("expected non-nil Resolver from Default()")
	}
}

func TestDefault_ReturnsSameInstance(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	r1 := Default()
	r2 := Default()
	if r1 != r2 {
		t.Fatal("expected Default() to return same instance on repeated calls")
	}
}

func TestSetDefault_ThenDefault(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	embedFS := fstest.MapFS{
		"embed.txt": &fstest.MapFile{Data: []byte("from embed")},
	}
	SetDefault(embedFS)

	r := Default()
	if r == nil {
		t.Fatal("expected non-nil Resolver")
	}
	data, err := r.ReadFile("embed.txt")
	if err != nil {
		t.Fatalf("expected embed.txt to be readable: %v", err)
	}
	if string(data) != "from embed" {
		t.Fatalf("got %q, want %q", data, "from embed")
	}
}

func TestSetDefault_AfterDefault_Panics(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	_ = Default() // triggers lazy init

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		SetDefault(fstest.MapFS{})
	}()

	if !panicked {
		t.Fatal("expected SetDefault to panic after Default() was lazily initialized")
	}
}

func TestSetDefault_CalledTwice_Panics(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	SetDefault(fstest.MapFS{})

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		SetDefault(fstest.MapFS{})
	}()

	if !panicked {
		t.Fatal("expected second SetDefault to panic")
	}
}

func TestGOYOKE_PROJECT_DIR(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	root := t.TempDir()
	t.Setenv("GOYOKE_PROJECT_DIR", root)
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "probe.txt"), []byte("goyoke"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Default()
	if !r.HasFile("probe.txt") {
		t.Fatal("expected probe.txt to be found via GOYOKE_PROJECT_DIR/.claude/")
	}
}

func TestCLAUDE_CONFIG_DIR(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	root := t.TempDir()
	t.Setenv("GOYOKE_PROJECT_DIR", "")
	t.Setenv("CLAUDE_CONFIG_DIR", root)

	if err := os.WriteFile(filepath.Join(root, "probe.txt"), []byte("claudecfg"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Default()
	if !r.HasFile("probe.txt") {
		t.Fatal("expected probe.txt to be found via CLAUDE_CONFIG_DIR")
	}
}

func TestDefault_FallbackHomeDir(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOYOKE_PROJECT_DIR", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "probe.txt"), []byte("home"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := Default()
	if !r.HasFile("probe.txt") {
		t.Fatal("expected probe.txt to be found via fallback ~/.claude/")
	}
}

func TestResetDefault_AllowsReinitialization(t *testing.T) {
	ResetDefault()
	defer ResetDefault()

	embedA := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("first")},
	}
	SetDefault(embedA)
	r1 := Default()

	ResetDefault()

	embedB := fstest.MapFS{
		"b.txt": &fstest.MapFile{Data: []byte("second")},
	}
	SetDefault(embedB)
	r2 := Default()

	if r1 == r2 {
		t.Fatal("expected different Resolver instances after ResetDefault")
	}
	if r2.HasFile("a.txt") {
		t.Fatal("r2 should not have a.txt from first init")
	}
	data, err := r2.ReadFile("b.txt")
	if err != nil {
		t.Fatalf("expected b.txt from second init: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("got %q, want %q", data, "second")
	}
}

func TestDiskFS_Open(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := diskFS{inner: os.DirFS(dir)}
	f, err := d.Open("f.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	f.Close()
}

func TestDiskFS_ReadFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := diskFS{inner: os.DirFS(dir)}
	data, err := d.ReadFile("f.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q, want hello", data)
	}
}
