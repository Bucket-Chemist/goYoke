// Package installer orchestrates the managed asset installation pipeline for
// harness adapter providers. It is transport-agnostic: callers supply a
// providers.Provider implementation and the installer handles prerequisite
// checking, directory setup, overwrite-safe file installation, and manifest
// recording.
package installer

import (
	"fmt"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/providers"
	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
)

// Run orchestrates the full installation pipeline for a provider:
//
//  1. Fail fast if any prerequisite check reports StatusFail.
//  2. Resolve the target directory via config.GetHarnessProviderDir.
//  3. Read any existing manifest to collect previously managed paths (used by
//     the provider for overwrite-protection decisions).
//  4. Call impl.Install — the provider copies its assets and enforces overwrite
//     protection internally.
//  5. Write (or update) the manifest with the new managed paths.
//
// Run does not call link.RunDoctor; the caller is responsible for any
// post-installation health checks.
func Run(providerName string, impl providers.Provider) error {
	// 1. Check prerequisites.
	for _, r := range impl.CheckPrerequisites() {
		if r.Status == link.StatusFail {
			return fmt.Errorf("prerequisite check %q failed: %s", r.Check, r.Message)
		}
	}

	// 2. Resolve target directory (created by GetHarnessProviderDir if absent).
	targetDir, err := pkgconfig.GetHarnessProviderDir(providerName)
	if err != nil {
		return fmt.Errorf("resolve provider dir for %q: %w", providerName, err)
	}

	// 3. Collect existing managed paths so the provider can allow safe overwrites.
	var existingManagedPaths []string
	if existing, err := link.ReadManifest(providerName); err == nil {
		existingManagedPaths = existing.ManagedPaths
	}
	// A missing manifest is not an error — existingManagedPaths stays nil.

	// 4. Install assets.
	managedPaths, err := impl.Install(targetDir, existingManagedPaths)
	if err != nil {
		return fmt.Errorf("install provider %q: %w", providerName, err)
	}

	// 5. Write manifest.
	opts := link.LinkOptions{
		SupportLevel: string(impl.SupportLevel()),
	}
	if err := link.Link(providerName, managedPaths, opts); err != nil {
		return fmt.Errorf("record manifest for %q: %w", providerName, err)
	}

	return nil
}
