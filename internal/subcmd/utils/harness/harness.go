// Package harness implements the "goyoke harness" command family, exposing
// Harness Link operations (list, link, unlink, status, doctor, print-config)
// through the subcmd registry.
//
// Commands in this package write exclusively to the io.Writer passed by the
// dispatcher; they never import TUI packages and run cleanly outside Bubbletea.
package harness

import (
	"fmt"
	"io"
	"strings"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/harness/providers"
	hermesprovider "github.com/Bucket-Chemist/goYoke/internal/harness/providers/hermes"
	manualprovider "github.com/Bucket-Chemist/goYoke/internal/harness/providers/manual"
	"github.com/Bucket-Chemist/goYoke/internal/harness/registry"
	"github.com/Bucket-Chemist/goYoke/internal/subcmd"
)

// defaultRegistry is the built-in provider catalog used by all harness subcommands.
var defaultRegistry = buildDefaultRegistry()

// providerImpls maps provider names to their managed-asset installer implementations.
// Providers in this map route through installer.Run in RunLink; others fall back to
// the bare link.Link call that records a manifest with no managed paths.
var providerImpls = buildProviderImpls()

func buildProviderImpls() map[string]providers.Provider {
	return map[string]providers.Provider{
		"manual": manualprovider.New(),
		"hermes": hermesprovider.New(),
	}
}

func buildDefaultRegistry() *registry.Registry {
	r := registry.New()
	r.Register(registry.Provider{
		Name:         "hermes",
		SupportLevel: registry.SupportLevelExperimental,
		Capabilities: []string{
			"submit_prompt",
			"get_snapshot",
			"respond_modal",
			"respond_permission",
			"interrupt",
			"set_model",
			"set_effort",
		},
	})
	r.Register(registry.Provider{
		Name:         "manual",
		SupportLevel: registry.SupportLevelSupported,
		Capabilities: []string{"submit_prompt", "get_snapshot"},
	})
	return r
}

// Commands returns the map of harness subcommands for use with Registry.RegisterGroup.
func Commands() map[string]subcmd.RunFunc {
	return map[string]subcmd.RunFunc{
		"list":         RunList,
		"link":         RunLink,
		"unlink":       RunUnlink,
		"status":       RunStatus,
		"doctor":       RunDoctor,
		"print-config": RunPrintConfig,
	}
}

// printDiagnostics writes diagnostic results to w using a human-readable format.
// Each result is prefixed with a status indicator: ✓ pass, ⚠ warn, ✗ fail.
func printDiagnostics(w io.Writer, results []link.DiagnosticResult) {
	for _, r := range results {
		var indicator string
		switch r.Status {
		case link.StatusPass:
			indicator = "  [pass]"
		case link.StatusWarn:
			indicator = "  [warn]"
		case link.StatusFail:
			indicator = "  [FAIL]"
		default:
			indicator = "  [    ]"
		}
		fmt.Fprintf(w, "%s %s: %s\n", indicator, r.Check, r.Message)
		if r.Detail != "" {
			fmt.Fprintf(w, "         %s\n", r.Detail)
		}
	}
}

// diagnosticSummary returns a one-line summary of diagnostic results.
func diagnosticSummary(results []link.DiagnosticResult) string {
	pass, warn, fail := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case link.StatusPass:
			pass++
		case link.StatusWarn:
			warn++
		case link.StatusFail:
			fail++
		}
	}
	parts := []string{fmt.Sprintf("%d pass", pass)}
	if warn > 0 {
		parts = append(parts, fmt.Sprintf("%d warn", warn))
	}
	if fail > 0 {
		parts = append(parts, fmt.Sprintf("%d fail", fail))
	}
	return strings.Join(parts, ", ")
}
