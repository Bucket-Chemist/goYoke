// Package providers defines the Provider interface for harness adapter implementations.
// A Provider augments the in-memory registry.Provider metadata with actual
// installation logic: asset copying, overwrite protection, and config emission.
//
// Not every registered provider has a Provider implementation. Providers without
// one fall back to the basic link.Link() behavior, which creates a manifest
// without installing any managed files.
package providers

import (
	"io"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
)

// Provider is implemented by harness adapters that support managed asset installation.
type Provider interface {
	// Name returns the unique provider identifier matching the registry entry.
	Name() string

	// SupportLevel returns the official support tier for this provider.
	SupportLevel() registry.SupportLevel

	// CheckPrerequisites returns diagnostic results for conditions that must be
	// satisfied before installation. An empty or nil slice means all prerequisites
	// pass. Any result with Status StatusFail blocks installation.
	CheckPrerequisites() []link.DiagnosticResult

	// Install copies managed assets into targetDir. existingManagedPaths lists
	// paths already owned by a previous installation of this provider — these may
	// be overwritten. Install must refuse to overwrite any existing file not in
	// existingManagedPaths, returning a descriptive error instead.
	// Returns the exhaustive list of absolute paths written on success.
	Install(targetDir string, existingManagedPaths []string) (managedPaths []string, err error)

	// Uninstall removes the given managed paths. Paths that no longer exist are
	// silently ignored.
	Uninstall(managedPaths []string) error

	// PrintConfig writes a machine-readable JSON configuration snippet to w,
	// including protocol information and setup instructions for this provider.
	PrintConfig(w io.Writer) error
}
