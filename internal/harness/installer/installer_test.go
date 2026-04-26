package installer

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
)

// stubProvider is a minimal Provider implementation for installer tests.
// It writes the configured files to targetDir during Install.
type stubProvider struct {
	name       string
	files      map[string][]byte
	prereqFail bool
	installErr error
}

func (s *stubProvider) Name() string                    { return s.name }
func (s *stubProvider) SupportLevel() registry.SupportLevel { return registry.SupportLevelSupported }

func (s *stubProvider) CheckPrerequisites() []link.DiagnosticResult {
	if s.prereqFail {
		return []link.DiagnosticResult{{
			Check:   "prereq",
			Status:  link.StatusFail,
			Message: "forced prerequisite failure",
		}}
	}
	return nil
}

func (s *stubProvider) Install(targetDir string, _ []string) ([]string, error) {
	if s.installErr != nil {
		return nil, s.installErr
	}
	var written []string
	for name, content := range s.files {
		p := filepath.Join(targetDir, name)
		if err := os.WriteFile(p, content, 0600); err != nil {
			return nil, err
		}
		written = append(written, p)
	}
	return written, nil
}

func (s *stubProvider) Uninstall(managedPaths []string) error {
	for _, p := range managedPaths {
		os.Remove(p)
	}
	return nil
}

func (s *stubProvider) PrintConfig(_ io.Writer) error { return nil }

// isolate redirects XDG_DATA_HOME to a temp directory so manifest operations
// do not touch the real user data directory.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
}

func TestRun_WritesManifestWithManagedPaths(t *testing.T) {
	isolate(t)

	impl := &stubProvider{
		name:  "test-provider",
		files: map[string][]byte{"hello.txt": []byte("hello")},
	}

	if err := Run("test-provider", impl); err != nil {
		t.Fatalf("Run: %v", err)
	}

	m, err := link.ReadManifest("test-provider")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if len(m.ManagedPaths) == 0 {
		t.Error("manifest has no managed paths")
	}

	for _, p := range m.ManagedPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("managed path %q does not exist: %v", p, err)
		}
	}
}

func TestRun_ManifestRecordsProviderAndSupportLevel(t *testing.T) {
	isolate(t)

	impl := &stubProvider{
		name:  "test-provider",
		files: map[string][]byte{"file.txt": []byte("data")},
	}

	if err := Run("test-provider", impl); err != nil {
		t.Fatalf("Run: %v", err)
	}

	m, err := link.ReadManifest("test-provider")
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Provider != "test-provider" {
		t.Errorf("manifest Provider: got %q, want %q", m.Provider, "test-provider")
	}
	if m.SupportLevel != string(registry.SupportLevelSupported) {
		t.Errorf("manifest SupportLevel: got %q, want %q", m.SupportLevel, registry.SupportLevelSupported)
	}
}

func TestRun_PrerequisiteFailureBlocked(t *testing.T) {
	isolate(t)

	impl := &stubProvider{
		name:       "test-provider",
		prereqFail: true,
	}

	err := Run("test-provider", impl)
	if err == nil {
		t.Fatal("expected error from failed prerequisite, got nil")
	}
	if !strings.Contains(err.Error(), "prerequisite") {
		t.Errorf("expected 'prerequisite' in error, got: %v", err)
	}

	// No manifest should be written when prerequisites fail.
	if _, readErr := link.ReadManifest("test-provider"); readErr == nil {
		t.Error("manifest was written despite prerequisite failure")
	}
}

func TestRun_PropagatesInstallError(t *testing.T) {
	isolate(t)

	impl := &stubProvider{
		name:       "test-provider",
		installErr: errors.New("disk full"),
	}

	err := Run("test-provider", impl)
	if err == nil {
		t.Fatal("expected error from failed install, got nil")
	}
	if !strings.Contains(err.Error(), "install") {
		t.Errorf("expected 'install' in error, got: %v", err)
	}
}

func TestRun_ExistingManifestPassedToInstall(t *testing.T) {
	isolate(t)

	var receivedExisting []string
	type capturingProvider struct {
		stubProvider
	}

	// Use a real stub but capture existingManagedPaths via a wrapper.
	// We verify this by running twice: the second run must receive the
	// first run's managed paths as existingManagedPaths.

	impl1 := &stubProvider{
		name:  "round-trip",
		files: map[string][]byte{"asset.txt": []byte("v1")},
	}
	if err := Run("round-trip", impl1); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	m, err := link.ReadManifest("round-trip")
	if err != nil {
		t.Fatalf("ReadManifest after first run: %v", err)
	}
	_ = receivedExisting

	// Second run: stub captures existingManagedPaths.
	capturedExisting := []string(nil)
	impl2 := &captureProvider{
		stubProvider: stubProvider{
			name:  "round-trip",
			files: map[string][]byte{"asset.txt": []byte("v2")},
		},
		capture: &capturedExisting,
	}
	if err := Run("round-trip", impl2); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if len(capturedExisting) == 0 {
		t.Error("second Run did not pass existing managed paths to Install")
	}
	// The captured paths should match the first manifest's managed paths.
	if len(capturedExisting) != len(m.ManagedPaths) {
		t.Errorf("captured existing paths len %d != manifest paths len %d",
			len(capturedExisting), len(m.ManagedPaths))
	}
}

// captureProvider wraps stubProvider and records the existingManagedPaths
// argument passed to Install.
type captureProvider struct {
	stubProvider
	capture *[]string
}

func (c *captureProvider) Install(targetDir string, existingManagedPaths []string) ([]string, error) {
	*c.capture = append(*c.capture, existingManagedPaths...)
	return c.stubProvider.Install(targetDir, existingManagedPaths)
}
