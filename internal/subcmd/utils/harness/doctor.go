package harness

import (
	"context"
	"fmt"
	"io"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
)

// RunDoctor runs harness health checks and reports results in human-readable form.
//
// Usage:
//
//	goyoke harness doctor              # global checks only
//	goyoke harness doctor <provider>   # global + provider-specific checks
//
// Exits with an error if any check has status "fail".
func RunDoctor(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	provider := ""
	if len(args) > 0 {
		provider = args[0]
	}

	results := link.RunDoctor(provider)

	if provider != "" {
		fmt.Fprintf(stdout, "Doctor results for provider %q:\n", provider)
	} else {
		fmt.Fprintln(stdout, "Doctor results (global checks):")
	}

	printDiagnostics(stdout, results)
	fmt.Fprintf(stdout, "\nSummary: %s\n", diagnosticSummary(results))

	for _, r := range results {
		if r.Status == link.StatusFail {
			return fmt.Errorf("doctor found failures (%s)", diagnosticSummary(results))
		}
	}

	return nil
}
