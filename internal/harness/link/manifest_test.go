package link_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
)

// setTestXDGEnv isolates XDG base directories to a temp dir and returns the
// temp root. Env vars are restored on test cleanup.
func setTestXDGEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	origData := os.Getenv("XDG_DATA_HOME")
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	origCache := os.Getenv("XDG_CACHE_HOME")
	os.Setenv("XDG_DATA_HOME", tmp)
	os.Setenv("XDG_RUNTIME_DIR", tmp)
	os.Unsetenv("XDG_CACHE_HOME")
	t.Cleanup(func() {
		os.Setenv("XDG_DATA_HOME", origData)
		os.Setenv("XDG_RUNTIME_DIR", origRuntime)
		os.Setenv("XDG_CACHE_HOME", origCache)
	})
	return tmp
}

func TestWriteReadManifestRoundTrip(t *testing.T) {
	setTestXDGEnv(t)

	now := time.Now().UTC().Truncate(time.Second)
	m := &link.HarnessLinkManifest{
		Provider:        "hermes",
		SupportLevel:    "supported",
		InstalledAt:     now,
		UpdatedAt:       now,
		AdapterVersion:  "1.2.3",
		ProtocolVersion: "1.0.0",
		ManagedPaths:    []string{"/tmp/a.json", "/tmp/b.yaml"},
		Notes:           []string{"first-party"},
	}

	if err := link.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	got, err := link.ReadManifest("hermes")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if got.Provider != "hermes" {
		t.Errorf("Provider: got %q", got.Provider)
	}
	if got.SupportLevel != "supported" {
		t.Errorf("SupportLevel: got %q", got.SupportLevel)
	}
	if got.AdapterVersion != "1.2.3" {
		t.Errorf("AdapterVersion: got %q", got.AdapterVersion)
	}
	if got.ProtocolVersion != "1.0.0" {
		t.Errorf("ProtocolVersion: got %q", got.ProtocolVersion)
	}
	if len(got.ManagedPaths) != 2 {
		t.Errorf("ManagedPaths: got %d, want 2", len(got.ManagedPaths))
	}
	if len(got.Notes) != 1 || got.Notes[0] != "first-party" {
		t.Errorf("Notes: got %v", got.Notes)
	}
	if !got.InstalledAt.Equal(now) {
		t.Errorf("InstalledAt: got %v, want %v", got.InstalledAt, now)
	}
}

func TestWriteManifestEmptyProvider(t *testing.T) {
	setTestXDGEnv(t)
	err := link.WriteManifest(&link.HarnessLinkManifest{Provider: ""})
	if err == nil {
		t.Error("WriteManifest with empty provider should return error")
	}
}

func TestReadManifestNotFound(t *testing.T) {
	setTestXDGEnv(t)
	_, err := link.ReadManifest("nonexistent")
	if err == nil {
		t.Error("ReadManifest for nonexistent provider should return error")
	}
}

func TestManifestFilePermissions(t *testing.T) {
	tmp := setTestXDGEnv(t)

	m := &link.HarnessLinkManifest{
		Provider:     "perm-test",
		SupportLevel: "manual",
		ManagedPaths: []string{},
	}
	if err := link.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// Reconstruct the manifest path from the XDG env.
	// GetHarnessLinksDir() → {XDG_DATA_HOME}/goyoke/harness/links/
	manifestFile := filepath.Join(tmp, "goyoke", "harness", "links", "perm-test.json")
	info, err := os.Stat(manifestFile)
	if err != nil {
		t.Fatalf("stat manifest file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("manifest file permissions: got %04o, want 0600", perm)
	}
}

func TestRemoveManifest(t *testing.T) {
	setTestXDGEnv(t)

	m := &link.HarnessLinkManifest{Provider: "to-remove", SupportLevel: "manual", ManagedPaths: []string{}}
	if err := link.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	if err := link.RemoveManifest("to-remove"); err != nil {
		t.Fatalf("RemoveManifest: %v", err)
	}

	_, err := link.ReadManifest("to-remove")
	if err == nil {
		t.Error("ReadManifest after RemoveManifest should return error")
	}
}

func TestRemoveManifestNotExist(t *testing.T) {
	setTestXDGEnv(t)
	// Removing a non-existent manifest must not return an error.
	if err := link.RemoveManifest("ghost"); err != nil {
		t.Errorf("RemoveManifest of non-existent manifest should not error: %v", err)
	}
}

func TestListLinkedProviders(t *testing.T) {
	setTestXDGEnv(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		m := &link.HarnessLinkManifest{Provider: name, SupportLevel: "manual", ManagedPaths: []string{}}
		if err := link.WriteManifest(m); err != nil {
			t.Fatalf("WriteManifest %q: %v", name, err)
		}
	}

	providers, err := link.ListLinkedProviders()
	if err != nil {
		t.Fatalf("ListLinkedProviders: %v", err)
	}
	if len(providers) != 3 {
		t.Fatalf("ListLinkedProviders: got %d, want 3", len(providers))
	}

	sort.Strings(providers)
	expected := []string{"alpha", "beta", "gamma"}
	for i, want := range expected {
		if providers[i] != want {
			t.Errorf("providers[%d]: got %q, want %q", i, providers[i], want)
		}
	}
}

func TestListLinkedProvidersEmpty(t *testing.T) {
	setTestXDGEnv(t)
	providers, err := link.ListLinkedProviders()
	if err != nil {
		t.Fatalf("ListLinkedProviders on empty dir: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected empty list, got %v", providers)
	}
}

func TestListLinkedProvidersIgnoresDirs(t *testing.T) {
	tmp := setTestXDGEnv(t)

	// Create a legitimate manifest.
	m := &link.HarnessLinkManifest{Provider: "real", SupportLevel: "manual", ManagedPaths: []string{}}
	if err := link.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	// Create a subdirectory inside the links dir — should be ignored.
	linksDir := filepath.Join(tmp, "goyoke", "harness", "links")
	if err := os.Mkdir(filepath.Join(linksDir, "subdir.json"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	providers, err := link.ListLinkedProviders()
	if err != nil {
		t.Fatalf("ListLinkedProviders: %v", err)
	}
	if len(providers) != 1 || providers[0] != "real" {
		t.Errorf("ListLinkedProviders: got %v, want [real]", providers)
	}
}
