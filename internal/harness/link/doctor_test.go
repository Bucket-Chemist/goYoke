package link

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// setDoctorTestEnv isolates XDG dirs to a temp root and restores on cleanup.
func setDoctorTestEnv(t *testing.T) string {
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

// findResult finds the first DiagnosticResult with the given check name.
func findResult(results []DiagnosticResult, check string) (DiagnosticResult, bool) {
	for _, r := range results {
		if r.Check == check {
			return r, true
		}
	}
	return DiagnosticResult{}, false
}

func TestRunDoctorNoActiveMetadata(t *testing.T) {
	setDoctorTestEnv(t)

	results := RunDoctor("")
	r, ok := findResult(results, "stale_pid")
	if !ok {
		t.Fatal("stale_pid check missing from results")
	}
	if r.Status != StatusPass {
		t.Errorf("stale_pid with no active.json: got status %q, want pass; message: %s", r.Status, r.Message)
	}
}

func TestRunDoctorStalePID(t *testing.T) {
	setDoctorTestEnv(t)

	// Override pidRunning to simulate a dead process.
	old := pidRunning
	pidRunning = func(int) bool { return false }
	t.Cleanup(func() { pidRunning = old })

	// Write a fake active metadata file with a non-zero PID.
	metaPath := config.GetHarnessActiveMetadataPath()
	ep := harnessproto.ActiveHarnessEndpoint{
		PID:             99999,
		StartedAt:       time.Now().UTC(),
		SocketPath:      "/tmp/fake.sock",
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
	}
	data, _ := json.Marshal(ep)
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		t.Fatalf("write active metadata: %v", err)
	}

	results := RunDoctor("")
	r, ok := findResult(results, "stale_pid")
	if !ok {
		t.Fatal("stale_pid check missing from results")
	}
	if r.Status != StatusFail {
		t.Errorf("stale_pid with dead PID: got status %q, want fail; message: %s", r.Status, r.Message)
	}
}

func TestRunDoctorLivePID(t *testing.T) {
	setDoctorTestEnv(t)

	// Override pidRunning to simulate a live process.
	old := pidRunning
	pidRunning = func(int) bool { return true }
	t.Cleanup(func() { pidRunning = old })

	metaPath := config.GetHarnessActiveMetadataPath()
	ep := harnessproto.ActiveHarnessEndpoint{
		PID:             12345,
		StartedAt:       time.Now().UTC(),
		SocketPath:      "/tmp/live.sock",
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
	}
	data, _ := json.Marshal(ep)
	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		t.Fatalf("write active metadata: %v", err)
	}

	results := RunDoctor("")
	r, ok := findResult(results, "stale_pid")
	if !ok {
		t.Fatal("stale_pid check missing from results")
	}
	if r.Status != StatusPass {
		t.Errorf("stale_pid with live PID: got status %q, want pass; message: %s", r.Status, r.Message)
	}
}

func TestRunDoctorInvalidActiveMetadataJSON(t *testing.T) {
	setDoctorTestEnv(t)

	metaPath := config.GetHarnessActiveMetadataPath()
	if err := os.WriteFile(metaPath, []byte("not json{"), 0600); err != nil {
		t.Fatalf("write bad metadata: %v", err)
	}

	results := RunDoctor("")
	r, ok := findResult(results, "stale_pid")
	if !ok {
		t.Fatal("stale_pid check missing from results")
	}
	if r.Status != StatusWarn {
		t.Errorf("stale_pid with bad JSON: got status %q, want warn", r.Status)
	}
}

func TestRunDoctorDirPermissionsPass(t *testing.T) {
	setDoctorTestEnv(t)

	results := RunDoctor("")

	// GetHarnessRuntimeDir and GetHarnessDataDir are created with 0700 by config.
	for _, check := range []string{"harness_runtime_dir", "harness_data_dir"} {
		r, ok := findResult(results, check)
		if !ok {
			t.Fatalf("%s check missing from results", check)
		}
		if r.Status != StatusPass {
			t.Errorf("%s: got status %q, want pass; message: %s", check, r.Status, r.Message)
		}
	}
}

func TestRunDoctorMissingProvider(t *testing.T) {
	setDoctorTestEnv(t)

	results := RunDoctor("no-such-provider")
	r, ok := findResult(results, "manifest")
	if !ok {
		t.Fatal("manifest check missing when provider has no manifest")
	}
	if r.Status != StatusFail {
		t.Errorf("manifest check for missing provider: got %q, want fail", r.Status)
	}
}

func TestRunDoctorManagedPathsAllExist(t *testing.T) {
	setDoctorTestEnv(t)
	tmp := t.TempDir()

	fileA := filepath.Join(tmp, "a.json")
	fileB := filepath.Join(tmp, "b.yaml")
	for _, f := range []string{fileA, fileB} {
		if err := os.WriteFile(f, []byte("{}"), 0600); err != nil {
			t.Fatalf("create %s: %v", f, err)
		}
	}

	m := &HarnessLinkManifest{
		Provider:        "ok-provider",
		SupportLevel:    "manual",
		ProtocolVersion: harnessproto.ProtocolVersion,
		ManagedPaths:    []string{fileA, fileB},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	results := RunDoctor("ok-provider")
	for _, r := range results {
		if r.Check == "managed_paths" && r.Status != StatusPass {
			t.Errorf("managed_paths for existing file %s: got %q, want pass; message: %s", r.Detail, r.Status, r.Message)
		}
	}
}

func TestRunDoctorManagedPathMissing(t *testing.T) {
	setDoctorTestEnv(t)
	tmp := t.TempDir()

	exists := filepath.Join(tmp, "exists.json")
	if err := os.WriteFile(exists, []byte("{}"), 0600); err != nil {
		t.Fatalf("create file: %v", err)
	}
	missing := filepath.Join(tmp, "missing.json")

	m := &HarnessLinkManifest{
		Provider:        "partial-provider",
		SupportLevel:    "manual",
		ProtocolVersion: harnessproto.ProtocolVersion,
		ManagedPaths:    []string{exists, missing},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	results := RunDoctor("partial-provider")

	var failCount int
	for _, r := range results {
		if r.Check == "managed_paths" && r.Status == StatusFail {
			failCount++
		}
	}
	if failCount != 1 {
		t.Errorf("expected 1 managed_paths fail, got %d", failCount)
	}
}

func TestRunDoctorProtocolVersionCompatible(t *testing.T) {
	setDoctorTestEnv(t)

	m := &HarnessLinkManifest{
		Provider:        "compat-prov",
		SupportLevel:    "supported",
		ProtocolVersion: harnessproto.ProtocolVersion,
		ManagedPaths:    []string{},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	results := RunDoctor("compat-prov")
	r, ok := findResult(results, "protocol_version")
	if !ok {
		t.Fatal("protocol_version check missing from results")
	}
	if r.Status != StatusPass {
		t.Errorf("protocol_version compatible: got %q, want pass; message: %s", r.Status, r.Message)
	}
}

func TestRunDoctorProtocolVersionMismatch(t *testing.T) {
	setDoctorTestEnv(t)

	m := &HarnessLinkManifest{
		Provider:        "incompat-prov",
		SupportLevel:    "supported",
		ProtocolVersion: "9.0.0", // different major
		ManagedPaths:    []string{},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	results := RunDoctor("incompat-prov")
	r, ok := findResult(results, "protocol_version")
	if !ok {
		t.Fatal("protocol_version check missing from results")
	}
	if r.Status != StatusFail {
		t.Errorf("protocol_version mismatch: got %q, want fail; message: %s", r.Status, r.Message)
	}
}

func TestRunDoctorEmptyProviderSkipsProviderChecks(t *testing.T) {
	setDoctorTestEnv(t)

	results := RunDoctor("")
	for _, r := range results {
		switch r.Check {
		case "manifest", "managed_paths", "protocol_version":
			t.Errorf("provider-specific check %q should not run when provider is empty", r.Check)
		}
	}
}

func TestMajorVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1.0.0", "1"},
		{"2.3.4", "2"},
		{"10.0.0", "10"},
		{"", ""},
		{"noDots", "noDots"},
	}
	for _, tc := range tests {
		got := majorVersion(tc.input)
		if got != tc.want {
			t.Errorf("majorVersion(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}
