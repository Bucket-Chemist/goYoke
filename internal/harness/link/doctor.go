package link

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// DiagnosticResult captures the outcome of a single doctor health check.
type DiagnosticResult struct {
	// Check is the machine-readable check identifier (e.g., "stale_pid").
	Check string
	// Status is "pass", "warn", or "fail".
	Status string
	// Message is a human-readable description of the check result.
	Message string
	// Detail carries optional technical context (file path, version strings, etc.).
	Detail string
}

const (
	StatusPass = "pass"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// pidRunning is a function variable so tests can substitute a fake implementation.
var pidRunning = defaultPidRunning

func defaultPidRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal(0) on POSIX systems checks process existence without delivering a signal.
	return proc.Signal(syscall.Signal(0)) == nil
}

// RunDoctor runs a set of health checks and returns one DiagnosticResult per check.
// Global checks (PID metadata, directory permissions) always run.
// Provider-specific checks (managed paths, protocol version) run only when
// provider is non-empty.
func RunDoctor(provider string) []DiagnosticResult {
	var results []DiagnosticResult

	results = append(results, checkStalePID())
	results = append(results, checkDirPerm("harness_runtime_dir", config.GetHarnessRuntimeDir(), 0700))
	results = append(results, checkDirPerm("harness_data_dir", config.GetHarnessDataDir(), 0700))

	if provider == "" {
		return results
	}

	m, err := ReadManifest(provider)
	if err != nil {
		results = append(results, DiagnosticResult{
			Check:   "manifest",
			Status:  StatusFail,
			Message: fmt.Sprintf("cannot read manifest for provider %q", provider),
			Detail:  err.Error(),
		})
		return results
	}

	results = append(results, checkManagedPaths(m)...)
	results = append(results, checkProtocolVersion(m))

	return results
}

func checkStalePID() DiagnosticResult {
	metaPath := config.GetHarnessActiveMetadataPath()
	data, err := os.ReadFile(metaPath)
	if os.IsNotExist(err) {
		return DiagnosticResult{
			Check:   "stale_pid",
			Status:  StatusPass,
			Message: "no active harness metadata (harness is not running)",
		}
	}
	if err != nil {
		return DiagnosticResult{
			Check:   "stale_pid",
			Status:  StatusWarn,
			Message: "cannot read active harness metadata",
			Detail:  err.Error(),
		}
	}

	var ep harnessproto.ActiveHarnessEndpoint
	if err := json.Unmarshal(data, &ep); err != nil {
		return DiagnosticResult{
			Check:   "stale_pid",
			Status:  StatusWarn,
			Message: "active harness metadata is not valid JSON",
			Detail:  err.Error(),
		}
	}

	if ep.PID <= 0 {
		return DiagnosticResult{
			Check:   "stale_pid",
			Status:  StatusWarn,
			Message: fmt.Sprintf("active harness metadata has invalid PID: %d", ep.PID),
		}
	}

	if !pidRunning(ep.PID) {
		return DiagnosticResult{
			Check:   "stale_pid",
			Status:  StatusFail,
			Message: fmt.Sprintf("active metadata references PID %d which is not running (stale)", ep.PID),
			Detail:  metaPath,
		}
	}

	return DiagnosticResult{
		Check:   "stale_pid",
		Status:  StatusPass,
		Message: fmt.Sprintf("harness process PID %d is running", ep.PID),
	}
}

func checkDirPerm(check, dir string, want os.FileMode) DiagnosticResult {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return DiagnosticResult{
				Check:   check,
				Status:  StatusFail,
				Message: fmt.Sprintf("directory does not exist: %s", dir),
				Detail:  dir,
			}
		}
		return DiagnosticResult{
			Check:   check,
			Status:  StatusFail,
			Message: fmt.Sprintf("cannot stat directory: %s", dir),
			Detail:  err.Error(),
		}
	}
	got := info.Mode().Perm()
	if got != want {
		return DiagnosticResult{
			Check:   check,
			Status:  StatusFail,
			Message: fmt.Sprintf("directory %s has permissions %04o, want %04o", dir, got, want),
			Detail:  dir,
		}
	}
	return DiagnosticResult{
		Check:   check,
		Status:  StatusPass,
		Message: fmt.Sprintf("directory %s has correct permissions %04o", dir, want),
	}
}

func checkManagedPaths(m *HarnessLinkManifest) []DiagnosticResult {
	if len(m.ManagedPaths) == 0 {
		return []DiagnosticResult{{
			Check:   "managed_paths",
			Status:  StatusPass,
			Message: "no managed paths recorded in manifest",
		}}
	}

	results := make([]DiagnosticResult, 0, len(m.ManagedPaths))
	for _, p := range m.ManagedPaths {
		if _, err := os.Stat(p); err != nil {
			status := StatusFail
			msg := fmt.Sprintf("managed path does not exist: %s", p)
			if !os.IsNotExist(err) {
				status = StatusWarn
				msg = fmt.Sprintf("cannot stat managed path: %s", p)
			}
			results = append(results, DiagnosticResult{
				Check:   "managed_paths",
				Status:  status,
				Message: msg,
				Detail:  p,
			})
		} else {
			results = append(results, DiagnosticResult{
				Check:   "managed_paths",
				Status:  StatusPass,
				Message: fmt.Sprintf("managed path exists: %s", p),
			})
		}
	}
	return results
}

func checkProtocolVersion(m *HarnessLinkManifest) DiagnosticResult {
	manifestMajor := majorVersion(m.ProtocolVersion)
	currentMajor := majorVersion(harnessproto.ProtocolVersion)

	if manifestMajor == "" || currentMajor == "" {
		return DiagnosticResult{
			Check:   "protocol_version",
			Status:  StatusWarn,
			Message: fmt.Sprintf("cannot parse protocol versions: manifest=%q current=%q", m.ProtocolVersion, harnessproto.ProtocolVersion),
		}
	}

	if manifestMajor != currentMajor {
		return DiagnosticResult{
			Check:   "protocol_version",
			Status:  StatusFail,
			Message: fmt.Sprintf("protocol version mismatch: manifest=%q current=%q", m.ProtocolVersion, harnessproto.ProtocolVersion),
			Detail:  fmt.Sprintf("major: manifest=%s current=%s", manifestMajor, currentMajor),
		}
	}

	return DiagnosticResult{
		Check:   "protocol_version",
		Status:  StatusPass,
		Message: fmt.Sprintf("protocol version %q is compatible with current %q", m.ProtocolVersion, harnessproto.ProtocolVersion),
	}
}

func majorVersion(v string) string {
	major, _, _ := strings.Cut(v, ".")
	return major
}
