package harness

import (
	"context"
	"fmt"
	"io"

	"github.com/Bucket-Chemist/goYoke/internal/harness/installer"
	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
)

// RunLink creates or updates a harness link manifest for the given provider.
//
// Usage: goyoke harness link <provider>
//
// The provider must be registered in the built-in catalog. Link records a
// manifest with the provider's support level; no managed paths are installed
// by this command (the adapter is responsible for its own files). After
// linking, doctor checks run automatically and any failures are reported.
func RunLink(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goyoke harness link <provider>")
	}
	providerName := args[0]

	p, ok := defaultRegistry.Get(providerName)
	if !ok {
		return fmt.Errorf("unknown provider %q; run 'goyoke harness list' to see available providers", providerName)
	}

	if impl, ok := providerImpls[providerName]; ok {
		// Full installer pipeline: prerequisites → asset copy → manifest.
		if err := installer.Run(providerName, impl); err != nil {
			return fmt.Errorf("link provider %q: %w", providerName, err)
		}
	} else {
		// Fallback: bare manifest with no managed paths.
		opts := link.LinkOptions{
			SupportLevel: string(p.SupportLevel),
		}
		if err := link.Link(providerName, nil, opts); err != nil {
			return fmt.Errorf("link provider %q: %w", providerName, err)
		}
	}

	fmt.Fprintf(stdout, "Linked harness provider %q (support: %s)\n", providerName, p.SupportLevel)

	// Validate the link by running doctor for this provider.
	results := link.RunDoctor(providerName)
	hasFail := false
	for _, r := range results {
		if r.Status == link.StatusFail {
			hasFail = true
			break
		}
	}
	if hasFail {
		fmt.Fprintln(stdout, "\nDoctor found issues after linking:")
		printDiagnostics(stdout, results)
		return fmt.Errorf("provider %q linked but doctor reported failures", providerName)
	}

	return nil
}
