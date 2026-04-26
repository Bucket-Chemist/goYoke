package harness

import (
	"context"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
)

// RunList prints all registered harness providers with their support levels
// and link status to stdout.
//
// Usage: goyoke harness list
func RunList(_ context.Context, _ []string, _ io.Reader, stdout io.Writer) error {
	providers := defaultRegistry.List()
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name < providers[j].Name
	})

	linked, err := link.ListLinkedProviders()
	if err != nil {
		return fmt.Errorf("list linked providers: %w", err)
	}
	linkedSet := make(map[string]bool, len(linked))
	for _, p := range linked {
		linkedSet[p] = true
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PROVIDER\tSUPPORT\tLINKED")
	fmt.Fprintln(tw, "--------\t-------\t------")
	for _, p := range providers {
		linkedStr := "no"
		if linkedSet[p.Name] {
			linkedStr = "yes"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Name, string(p.SupportLevel), linkedStr)
	}
	return tw.Flush()
}
