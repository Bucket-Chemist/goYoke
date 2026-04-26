// Package manual implements the manual harness provider — the universal fallback
// adapter for external harnesses that do not have a first-party goYoke integration.
//
// On install it copies embedded documentation and a config template into the
// provider directory so the operator has everything needed to wire a custom
// process to the running goYoke TUI.
package manual

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	harnessassets "github.com/Bucket-Chemist/goYoke/defaults/harnesses"
	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// Adapter is the manual harness provider implementation.
type Adapter struct{}

// New returns a ready-to-use Adapter.
func New() *Adapter { return &Adapter{} }

// Name returns the canonical provider identifier.
func (a *Adapter) Name() string { return "manual" }

// SupportLevel marks the manual adapter as fully supported — it is the universal
// fallback and always works regardless of what external harness the operator uses.
func (a *Adapter) SupportLevel() registry.SupportLevel {
	return registry.SupportLevelSupported
}

// CheckPrerequisites always passes: the manual adapter has no external dependencies.
func (a *Adapter) CheckPrerequisites() []link.DiagnosticResult { return nil }

// Install copies the embedded manual adapter templates into targetDir.
// Each file is written with 0600 permissions. If a file already exists at the
// target path and is not listed in existingManagedPaths, Install returns an
// error to prevent overwriting unmanaged content.
// Returns the exhaustive list of absolute paths written on success.
func (a *Adapter) Install(targetDir string, existingManagedPaths []string) ([]string, error) {
	managed := make(map[string]struct{}, len(existingManagedPaths))
	for _, p := range existingManagedPaths {
		managed[p] = struct{}{}
	}

	entries, err := fs.ReadDir(harnessassets.FS, "manual")
	if err != nil {
		return nil, fmt.Errorf("manual adapter: read embedded assets: %w", err)
	}

	var written []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		data, err := harnessassets.FS.ReadFile("manual/" + e.Name())
		if err != nil {
			return nil, fmt.Errorf("manual adapter: read embedded file %q: %w", e.Name(), err)
		}

		target := filepath.Join(targetDir, e.Name())

		if _, err := os.Stat(target); err == nil {
			if _, ok := managed[target]; !ok {
				return nil, fmt.Errorf(
					"manual adapter: refusing to overwrite unmanaged file %q; "+
						"remove it manually or run 'goyoke harness unlink manual' first",
					target,
				)
			}
		}

		if err := os.WriteFile(target, data, 0600); err != nil {
			return nil, fmt.Errorf("manual adapter: write %q: %w", target, err)
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
			return fmt.Errorf("manual adapter: remove %q: %w", p, err)
		}
	}
	return nil
}

// manualConfigOutput is the JSON shape emitted by PrintConfig.
type manualConfigOutput struct {
	Provider        string   `json:"provider"`
	SupportLevel    string   `json:"support_level"`
	Protocol        string   `json:"protocol"`
	ProtocolVersion string   `json:"protocol_version"`
	SocketPath      string   `json:"socket_path"`
	SetupNotes      []string `json:"setup_notes"`
}

// PrintConfig writes a JSON configuration snippet for manual harness wiring.
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

	out := manualConfigOutput{
		Provider:        "manual",
		SupportLevel:    string(registry.SupportLevelSupported),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		SocketPath:      socketPath,
		SetupNotes: []string{
			"Connect to socket_path using SOCK_STREAM and the harness-link JSON protocol.",
			"If socket_path is empty, start goYoke first; find the path via 'goyoke harness status'.",
			"Send Request envelopes and read Response envelopes as newline-delimited JSON.",
			"See the installed README.md in the provider directory for full setup instructions.",
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
