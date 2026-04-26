package hermes

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
)

// withFakeLookPath replaces lookPathFunc for the duration of the test and
// restores it on cleanup. This avoids requiring a real hermes binary in CI.
func withFakeLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	orig := lookPathFunc
	lookPathFunc = fn
	t.Cleanup(func() { lookPathFunc = orig })
}

func TestAdapter_Name(t *testing.T) {
	a := New()
	if got := a.Name(); got != "hermes" {
		t.Errorf("Name: got %q, want %q", got, "hermes")
	}
}

func TestAdapter_SupportLevel(t *testing.T) {
	a := New()
	if got := a.SupportLevel(); got != registry.SupportLevelExperimental {
		t.Errorf("SupportLevel: got %q, want %q", got, registry.SupportLevelExperimental)
	}
}

func TestAdapter_CheckPrerequisites_Pass(t *testing.T) {
	withFakeLookPath(t, func(string) (string, error) {
		return "/usr/local/bin/hermes", nil
	})

	a := New()
	results := a.CheckPrerequisites()
	if len(results) == 0 {
		t.Fatal("expected at least one diagnostic result")
	}
	for _, r := range results {
		if r.Status == link.StatusFail {
			t.Errorf("CheckPrerequisites returned failure: check=%q message=%q", r.Check, r.Message)
		}
	}
	if results[0].Detail != "/usr/local/bin/hermes" {
		t.Errorf("Detail: got %q, want %q", results[0].Detail, "/usr/local/bin/hermes")
	}
}

func TestAdapter_CheckPrerequisites_Fail_BinaryMissing(t *testing.T) {
	withFakeLookPath(t, func(string) (string, error) {
		return "", errors.New("not found")
	})

	a := New()
	results := a.CheckPrerequisites()
	if len(results) == 0 {
		t.Fatal("expected at least one diagnostic result")
	}

	found := false
	for _, r := range results {
		if r.Check == "hermes_binary" && r.Status == link.StatusFail {
			found = true
			if !strings.Contains(r.Message, "not found in PATH") {
				t.Errorf("expected 'not found in PATH' in failure message, got: %q", r.Message)
			}
		}
	}
	if !found {
		t.Error("expected hermes_binary StatusFail result, not found")
	}
}

func TestAdapter_Install_CopiesFiles(t *testing.T) {
	dir := t.TempDir()
	a := New()

	paths, err := a.Install(dir, nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("Install returned no managed paths")
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("installed path %q does not exist: %v", p, err)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil || strings.HasPrefix(rel, "..") {
			t.Errorf("installed path %q is outside targetDir %q", p, dir)
		}
	}
}

func TestAdapter_Install_WritesExpectedFiles(t *testing.T) {
	dir := t.TempDir()
	a := New()

	paths, err := a.Install(dir, nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	names := make(map[string]bool)
	for _, p := range paths {
		names[filepath.Base(p)] = true
	}
	for _, want := range []string{"README.md", "config-template.json"} {
		if !names[want] {
			t.Errorf("Install did not write expected file %q; got paths: %v", want, paths)
		}
	}
}

func TestAdapter_Install_OverwriteRefused(t *testing.T) {
	dir := t.TempDir()

	// Pre-create README.md as unmanaged content.
	unmanaged := filepath.Join(dir, "README.md")
	if err := os.WriteFile(unmanaged, []byte("custom content"), 0644); err != nil {
		t.Fatal(err)
	}

	a := New()
	_, err := a.Install(dir, nil) // no existing managed paths
	if err == nil {
		t.Fatal("expected overwrite-refusal error, got nil")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Errorf("expected 'refusing to overwrite' in error, got: %v", err)
	}

	// The original unmanaged file must be untouched.
	got, readErr := os.ReadFile(unmanaged)
	if readErr != nil {
		t.Fatalf("cannot read unmanaged file after failed Install: %v", readErr)
	}
	if string(got) != "custom content" {
		t.Errorf("unmanaged file was modified: got %q, want %q", string(got), "custom content")
	}
}

func TestAdapter_Install_ReinstallAllowed(t *testing.T) {
	dir := t.TempDir()
	a := New()

	// First install.
	paths, err := a.Install(dir, nil)
	if err != nil {
		t.Fatalf("first Install: %v", err)
	}

	// Second install passing existing managed paths — must succeed (safe overwrite).
	paths2, err := a.Install(dir, paths)
	if err != nil {
		t.Fatalf("reinstall Install: %v", err)
	}
	if len(paths2) == 0 {
		t.Error("reinstall returned no managed paths")
	}
}

func TestAdapter_Uninstall_RemovesManagedPaths(t *testing.T) {
	dir := t.TempDir()
	a := New()

	paths, err := a.Install(dir, nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	if err := a.Uninstall(paths); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("path %q still exists after Uninstall", p)
		}
	}
}

func TestAdapter_Uninstall_IdempotentOnMissingPaths(t *testing.T) {
	a := New()
	err := a.Uninstall([]string{"/nonexistent/path/that/does/not/exist"})
	if err != nil {
		t.Errorf("Uninstall with missing paths returned error: %v", err)
	}
}

func TestAdapter_PrintConfig_ValidJSON(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	a := New()
	var buf bytes.Buffer
	if err := a.PrintConfig(&buf); err != nil {
		t.Fatalf("PrintConfig: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("PrintConfig output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	for _, key := range []string{"provider", "support_level", "protocol", "protocol_version", "setup_notes", "socket_path"} {
		if _, ok := out[key]; !ok {
			t.Errorf("PrintConfig output missing field %q", key)
		}
	}
}

func TestAdapter_PrintConfig_Fields(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	a := New()
	var buf bytes.Buffer
	if err := a.PrintConfig(&buf); err != nil {
		t.Fatalf("PrintConfig: %v", err)
	}

	var out hermesConfigOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Provider != "hermes" {
		t.Errorf("provider: got %q, want %q", out.Provider, "hermes")
	}
	if out.SupportLevel != "experimental" {
		t.Errorf("support_level: got %q, want %q", out.SupportLevel, "experimental")
	}
	if out.Protocol == "" {
		t.Error("protocol field is empty")
	}
	if out.ProtocolVersion == "" {
		t.Error("protocol_version field is empty")
	}
	if len(out.SetupNotes) == 0 {
		t.Error("setup_notes is empty")
	}
}
