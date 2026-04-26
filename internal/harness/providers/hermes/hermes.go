// Package hermes implements the first-party Hermes harness provider.
//
// The adapter installs goYoke-side integration artifacts (README.md and a
// config template) into the goYoke-managed provider directory. It does not
// write into any Hermes-owned directory.
//
// Support level: experimental. The goYoke-side/local harness path is
// validated, but promotion waits for a true end-to-end run against a live
// Hermes instance.
package hermes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	harnessassets "github.com/Bucket-Chemist/goYoke/defaults/harnesses"
	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// lookPathFunc is the function used to locate the hermes binary in PATH.
// Replaced in tests to avoid requiring a real hermes binary.
var lookPathFunc func(string) (string, error) = exec.LookPath

// Adapter is the Hermes harness provider implementation.
type Adapter struct{}

// New returns a ready-to-use Adapter.
func New() *Adapter { return &Adapter{} }

// Name returns the canonical provider identifier.
func (a *Adapter) Name() string { return "hermes" }

// SupportLevel returns experimental — promotion waits for live Hermes
// interoperability verification, not just goYoke-side local validation.
func (a *Adapter) SupportLevel() registry.SupportLevel {
	return registry.SupportLevelExperimental
}

// CheckPrerequisites verifies that the hermes binary is available in PATH.
// Returns a single failing result with a clear explanation when hermes is absent.
func (a *Adapter) CheckPrerequisites() []link.DiagnosticResult {
	resolved, err := lookPathFunc("hermes")
	if err != nil {
		return []link.DiagnosticResult{{
			Check:  "hermes_binary",
			Status: link.StatusFail,
			Message: "hermes binary not found in PATH; " +
				"install Hermes and ensure it is on your PATH, then re-run 'goyoke harness link hermes'",
		}}
	}
	return []link.DiagnosticResult{{
		Check:   "hermes_binary",
		Status:  link.StatusPass,
		Message: "hermes binary found",
		Detail:  resolved,
	}}
}

// Install copies the embedded Hermes adapter templates into targetDir.
// Each file is written with 0600 permissions. If a file already exists at the
// target path and is not listed in existingManagedPaths, Install returns an
// error to prevent overwriting unmanaged content.
// Returns the exhaustive list of absolute paths written on success.
func (a *Adapter) Install(targetDir string, existingManagedPaths []string) ([]string, error) {
	managed := make(map[string]struct{}, len(existingManagedPaths))
	for _, p := range existingManagedPaths {
		managed[p] = struct{}{}
	}

	entries, err := fs.ReadDir(harnessassets.FS, "hermes")
	if err != nil {
		return nil, fmt.Errorf("hermes adapter: read embedded assets: %w", err)
	}

	var written []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		data, err := harnessassets.FS.ReadFile("hermes/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("hermes adapter: read embedded file %q: %w", e.Name(), err)
		}

		target := filepath.Join(targetDir, e.Name())

		if _, err := os.Stat(target); err == nil {
			if _, ok := managed[target]; !ok {
				return nil, fmt.Errorf(
					"hermes adapter: refusing to overwrite unmanaged file %q; "+
						"remove it manually or run 'goyoke harness unlink hermes' first",
					target,
				)
			}
		}

		if err := os.WriteFile(target, data, 0600); err != nil {
			return nil, fmt.Errorf("hermes adapter: write %q: %w", target, err)
		}
		written = append(written, target)
	}

	return written, nil
}

// Uninstall removes the given managed paths. Paths that no longer exist are
// silently ignored.
func (a *Adapter) Uninstall(managedPaths []string) error {
	for _, p := range managedPaths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("hermes adapter: remove %q: %w", p, err)
		}
	}
	return nil
}

// hermesConfigOutput is the JSON shape emitted by PrintConfig.
type hermesConfigOutput struct {
	Provider        string   `json:"provider"`
	SupportLevel    string   `json:"support_level"`
	Protocol        string   `json:"protocol"`
	ProtocolVersion string   `json:"protocol_version"`
	SocketPath      string   `json:"socket_path"`
	SetupNotes      []string `json:"setup_notes"`
}

// PrintConfig writes a JSON configuration snippet for Hermes harness wiring.
// socket_path is populated from active metadata when the harness is running
// and is empty otherwise — the operator must start goYoke before connecting.
func (a *Adapter) PrintConfig(w io.Writer) error {
	socketPath := ""
	if data, err := os.ReadFile(pkgconfig.GetHarnessActiveMetadataPath()); err == nil {
		var ep harnessproto.ActiveHarnessEndpoint
		if json.Unmarshal(data, &ep) == nil {
			socketPath = ep.SocketPath
		}
	}

	out := hermesConfigOutput{
		Provider:        "hermes",
		SupportLevel:    string(registry.SupportLevelExperimental),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		SocketPath:      socketPath,
		SetupNotes: []string{
			"Connect to socket_path using SOCK_STREAM and the harness-link JSON protocol.",
			"If socket_path is empty, start goYoke first; find the path via 'goyoke harness status'.",
			"Send Request envelopes and read Response envelopes as newline-delimited JSON.",
			"Hermes supports all seven protocol operations; see the installed README.md for details.",
			"This adapter is experimental — configuration details may change before stable release.",
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
