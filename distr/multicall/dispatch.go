package multicall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Dispatch is the top-level entry point for the gofortress multi-call binary.
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

	// Fall through: the binary was invoked as "gofortress [subcommand]".
	if err := buildRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// buildRootCmd constructs the Cobra command tree for the gofortress surface.
// All subcommands are placeholder stubs; real implementations land in later tickets.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gofortress",
		Short: "GOgent Fortress — unified hook and tooling binary",
		Long: `gofortress is the multi-call binary for GOgent Fortress.

When invoked via a symlink named after a hook (e.g. gogent-validate), it
dispatches to the corresponding hook implementation.

When invoked directly as 'gofortress', it exposes administrative subcommands
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
		Short: "Initialise GOgent Fortress in the current directory",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: gofortress init")
			return nil
		},
	}
}

func upgradeSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade GOgent Fortress binaries to the latest release",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: gofortress upgrade")
			return nil
		},
	}
}

func doctorSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check the GOgent Fortress installation for problems",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: gofortress doctor")
			return nil
		},
	}
}

func versionSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print gofortress version information",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: gofortress version")
			return nil
		},
	}
}

func agentsSubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agents",
		Short: "List or manage registered GOgent agents",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("not yet wired: gofortress agents")
			return nil
		},
	}
}
