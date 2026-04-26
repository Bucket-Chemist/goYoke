package link_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

func TestLinkCreatesManifest(t *testing.T) {
	setTestXDGEnv(t)
	tmp := t.TempDir()

	// Create the managed files in a temp location.
	fileA := filepath.Join(tmp, "skill.yaml")
	fileB := filepath.Join(tmp, "config.json")
	for _, f := range []string{fileA, fileB} {
		if err := os.WriteFile(f, []byte("{}"), 0600); err != nil {
			t.Fatalf("create %s: %v", f, err)
		}
	}

	opts := link.LinkOptions{
		SupportLevel:   "supported",
		AdapterVersion: "2.0.0",
		Notes:          []string{"test link"},
	}
	if err := link.Link("myprovider", []string{fileA, fileB}, opts); err != nil {
		t.Fatalf("Link: %v", err)
	}

	m, err := link.ReadManifest("myprovider")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if m.Provider != "myprovider" {
		t.Errorf("Provider: got %q", m.Provider)
	}
	if m.SupportLevel != "supported" {
		t.Errorf("SupportLevel: got %q", m.SupportLevel)
	}
	if m.AdapterVersion != "2.0.0" {
		t.Errorf("AdapterVersion: got %q", m.AdapterVersion)
	}
	if m.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion: got %q, want %q", m.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if len(m.ManagedPaths) != 2 {
		t.Errorf("ManagedPaths: got %d, want 2", len(m.ManagedPaths))
	}
	if m.InstalledAt.IsZero() {
		t.Error("InstalledAt should not be zero")
	}
	if m.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
	if len(m.Notes) != 1 || m.Notes[0] != "test link" {
		t.Errorf("Notes: got %v", m.Notes)
	}
}

func TestLinkDefaultSupportLevel(t *testing.T) {
	setTestXDGEnv(t)

	if err := link.Link("default-prov", []string{}, link.LinkOptions{}); err != nil {
		t.Fatalf("Link: %v", err)
	}

	m, err := link.ReadManifest("default-prov")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.SupportLevel != "manual" {
		t.Errorf("default SupportLevel: got %q, want manual", m.SupportLevel)
	}
}

func TestLinkPreservesInstalledAt(t *testing.T) {
	setTestXDGEnv(t)

	if err := link.Link("preserve-prov", []string{}, link.LinkOptions{SupportLevel: "manual"}); err != nil {
		t.Fatalf("first Link: %v", err)
	}

	m1, err := link.ReadManifest("preserve-prov")
	if err != nil {
		t.Fatalf("ReadManifest after first Link: %v", err)
	}
	installedAt := m1.InstalledAt

	// Brief sleep to ensure UpdatedAt will differ.
	time.Sleep(2 * time.Millisecond)

	if err := link.Link("preserve-prov", []string{}, link.LinkOptions{SupportLevel: "supported"}); err != nil {
		t.Fatalf("second Link: %v", err)
	}

	m2, err := link.ReadManifest("preserve-prov")
	if err != nil {
		t.Fatalf("ReadManifest after second Link: %v", err)
	}

	if !m2.InstalledAt.Equal(installedAt) {
		t.Errorf("InstalledAt changed on re-link: was %v, now %v", installedAt, m2.InstalledAt)
	}
	if !m2.UpdatedAt.After(installedAt) {
		t.Errorf("UpdatedAt should be after InstalledAt on re-link: UpdatedAt=%v InstalledAt=%v", m2.UpdatedAt, installedAt)
	}
	if m2.SupportLevel != "supported" {
		t.Errorf("SupportLevel not updated: got %q", m2.SupportLevel)
	}
}

func TestLinkEmptyProvider(t *testing.T) {
	setTestXDGEnv(t)
	if err := link.Link("", []string{}, link.LinkOptions{}); err == nil {
		t.Error("Link with empty provider should return error")
	}
}

func TestUnlinkRemovesManagedPaths(t *testing.T) {
	setTestXDGEnv(t)
	tmp := t.TempDir()

	fileA := filepath.Join(tmp, "adapter.json")
	fileB := filepath.Join(tmp, "skill.yaml")
	unrelated := filepath.Join(tmp, "unrelated.txt")
	for _, f := range []string{fileA, fileB, unrelated} {
		if err := os.WriteFile(f, []byte("data"), 0600); err != nil {
			t.Fatalf("create %s: %v", f, err)
		}
	}

	if err := link.Link("unlink-prov", []string{fileA, fileB}, link.LinkOptions{}); err != nil {
		t.Fatalf("Link: %v", err)
	}

	if err := link.Unlink("unlink-prov"); err != nil {
		t.Fatalf("Unlink: %v", err)
	}

	// Managed paths must be gone.
	for _, f := range []string{fileA, fileB} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("managed path %s should be removed after Unlink", f)
		}
	}

	// Manifest must be gone.
	if _, err := link.ReadManifest("unlink-prov"); err == nil {
		t.Error("manifest should be removed after Unlink")
	}

	// Unrelated file must NOT be touched.
	if _, err := os.Stat(unrelated); err != nil {
		t.Errorf("unrelated file was unexpectedly removed: %v", err)
	}
}

func TestUnlinkToleratesAbsentManagedPaths(t *testing.T) {
	setTestXDGEnv(t)

	// List a path that does not exist — Unlink should still succeed.
	if err := link.Link("ghost-prov", []string{"/tmp/does-not-exist-xyz"}, link.LinkOptions{}); err != nil {
		t.Fatalf("Link: %v", err)
	}

	if err := link.Unlink("ghost-prov"); err != nil {
		t.Errorf("Unlink with absent managed path should not error: %v", err)
	}
}

func TestUnlinkNoManifest(t *testing.T) {
	setTestXDGEnv(t)
	if err := link.Unlink("no-manifest-prov"); err == nil {
		t.Error("Unlink without manifest should return error")
	}
}
