package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
)

// RunUnlink removes the harness link manifest for the given provider and
// deletes any managed paths recorded at link time.
//
// Usage: goyoke harness unlink <provider>
//
// Only paths listed in the manifest's ManagedPaths field are removed.
// No recursive directory deletion is performed.
func RunUnlink(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goyoke harness unlink <provider>")
	}
	providerName := args[0]

	m, err := link.ReadManifest(providerName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("provider %q is not linked", providerName)
		}
		return fmt.Errorf("read manifest for %q: %w", providerName, err)
	}

	removed := make([]string, len(m.ManagedPaths))
	copy(removed, m.ManagedPaths)

	if err := link.Unlink(providerName); err != nil {
		return fmt.Errorf("unlink provider %q: %w", providerName, err)
	}

	fmt.Fprintf(stdout, "Unlinked harness provider %q\n", providerName)
	if len(removed) > 0 {
		fmt.Fprintln(stdout, "Removed managed paths:")
		for _, p := range removed {
			fmt.Fprintf(stdout, "  %s\n", p)
		}
	}

	return nil
}
