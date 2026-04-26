package link

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// LinkOptions configures a Link call with provider metadata beyond the managed paths.
type LinkOptions struct {
	// SupportLevel is the support tier for this provider. Defaults to "manual" if empty.
	SupportLevel string

	// AdapterVersion is the version string of the adapter being linked.
	AdapterVersion string

	// Notes are optional human-readable annotations stored in the manifest.
	Notes []string
}

// Link creates or updates a manifest for the given provider, recording
// managedPaths as the exhaustive list of files owned by this link.
//
// If a manifest already exists, InstalledAt is preserved and UpdatedAt is
// refreshed. Unlink will only remove paths listed in managedPaths.
func Link(provider string, managedPaths []string, opts LinkOptions) error {
	if provider == "" {
		return fmt.Errorf("provider must not be empty")
	}

	supportLevel := opts.SupportLevel
	if supportLevel == "" {
		supportLevel = "manual"
	}

	now := time.Now().UTC()

	existing, err := ReadManifest(provider)
	if err != nil {
		// No existing manifest — start fresh.
		existing = &HarnessLinkManifest{InstalledAt: now}
	}

	existing.Provider = provider
	existing.SupportLevel = supportLevel
	existing.UpdatedAt = now
	existing.AdapterVersion = opts.AdapterVersion
	existing.ProtocolVersion = harnessproto.ProtocolVersion
	existing.ManagedPaths = managedPaths
	existing.Notes = opts.Notes

	return WriteManifest(existing)
}

// Unlink removes only the files listed in the manifest's ManagedPaths, then
// removes the manifest file itself. It never performs recursive directory
// deletion; only individual files in ManagedPaths are removed.
//
// Unlink returns an error if the manifest cannot be read. If a managed path
// cannot be removed (and is not already absent), the first removal error is
// returned and the manifest is left in place to allow retry.
func Unlink(provider string) error {
	m, err := ReadManifest(provider)
	if err != nil {
		return fmt.Errorf("unlink %q: %w", provider, err)
	}

	for _, p := range m.ManagedPaths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("unlink %q: remove managed path %q: %w", provider, p, err)
		}
	}

	return RemoveManifest(provider)
}
