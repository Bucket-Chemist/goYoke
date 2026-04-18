package multicall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Dispatch is the top-level entry point for the goyoke multi-call binary.
//
// Resolution order:
//  1. Derive invocation name from filepath.Base(os.Args[0]), stripping ".exe".
//  2. If the name matches a registered hook, call it and return.
//  3. Otherwise fall through to the Cobra subcommand tree.
//
// Symlink dispatch must complete before Cobra parses os.Args so that hook
// binaries never see flags meant for the TUI command surface.
func Dispatch() {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")

	if fn, ok := Lookup(name); ok {
		fn()
		return
	}

	// Fall through: the binary was invoked as "goyoke [subcommand]".
	if err := buildRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// buildRootCmd constructs the Cobra command tree for the goyoke surface.
// All subcommands are placeholder stubs; real implementations land in later tickets.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "goyoke",
		Short: "goYoke — unified hook and tooling binary",
		Long: `goyoke is the multi-call binary for goYoke.

When invoked via a symlink named after a hook (e.g. goyoke-validate), it
dispatches to the corresponding hook implementation.

When invoked directly as 'goyoke', it exposes administrative subcommands
for managing the installation.`,
	}

	root.AddCommand(initSubCmd())
	root.AddCommand(upgradeSubCmd())
	root.AddCommand(doctorSubCmd())
	root.AddCommand(versionSubCmd())
	root.AddCommand(agentsSubCmd())

	return root
}

func initSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialise goYoke in the current directory",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: goyoke init")
			return nil
		},
	}
}

func upgradeSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade goYoke binaries to the latest release",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: goyoke upgrade")
			return nil
		},
	}
}

func doctorSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the goYoke installation for problems",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: goyoke doctor")
			return nil
		},
	}
}

func versionSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print goyoke version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: goyoke version")
			return nil
		},
	}
}

func agentsSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List or manage registered goYoke agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: goyoke agents")
			return nil
		},
	}
}
