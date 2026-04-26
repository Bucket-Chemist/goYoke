// Package link manages harness link manifests, link/unlink operations, and
// doctor diagnostics.
//
// The manifest layer is the persistent counterpart to the in-memory registry:
// manifests record what was installed, where managed files live, and which
// protocol version the adapter was linked against.
package link

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/config"
)

// HarnessLinkManifest records the state of a linked harness adapter.
// It is stored as a 0600 JSON file at GetHarnessLinksDir()/{provider}.json.
type HarnessLinkManifest struct {
	Provider        string    `json:"provider"`
	SupportLevel    string    `json:"support_level"`
	InstalledAt     time.Time `json:"installed_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	AdapterVersion  string    `json:"adapter_version"`
	ProtocolVersion string    `json:"protocol_version"`
	// ManagedPaths is the exhaustive list of file paths created during Link.
	// Unlink ONLY removes paths listed here; it never performs recursive deletes.
	ManagedPaths []string `json:"managed_paths"`
	Notes        []string `json:"notes,omitempty"`
}

// manifestPath returns the on-disk path for the given provider's manifest.
func manifestPath(provider string) string {
	return filepath.Join(config.GetHarnessLinksDir(), provider+".json")
}

// ReadManifest reads and parses the manifest file for the given provider.
// Returns an error wrapping os.ErrNotExist when no manifest exists.
func ReadManifest(provider string) (*HarnessLinkManifest, error) {
	path := manifestPath(provider)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest for %q: %w", provider, err)
	}
	var m HarnessLinkManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest for %q: %w", provider, err)
	}
	return &m, nil
}

// WriteManifest serializes m and writes it to disk with 0600 permissions.
// m.Provider must not be empty.
func WriteManifest(m *HarnessLinkManifest) error {
	if m.Provider == "" {
		return fmt.Errorf("manifest provider name must not be empty")
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest for %q: %w", m.Provider, err)
	}
	path := manifestPath(m.Provider)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write manifest for %q: %w", m.Provider, err)
	}
	return nil
}

// RemoveManifest deletes the manifest file for the given provider.
// It is not an error if the file does not exist.
func RemoveManifest(provider string) error {
	path := manifestPath(provider)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove manifest for %q: %w", provider, err)
	}
	return nil
}

// ListLinkedProviders returns the provider names of all existing manifests by
// scanning GetHarnessLinksDir() for *.json files.
func ListLinkedProviders() ([]string, error) {
	dir := config.GetHarnessLinksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list linked providers: %w", err)
	}
	var providers []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if stem, ok := strings.CutSuffix(e.Name(), ".json"); ok {
			providers = append(providers, stem)
		}
	}
	return providers, nil
}
