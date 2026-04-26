package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// RunStatus reports the live harness endpoint state and lists linked providers.
//
// Usage: goyoke harness status
func RunStatus(_ context.Context, _ []string, _ io.Reader, stdout io.Writer) error {
	metaPath := pkgconfig.GetHarnessActiveMetadataPath()
	data, err := os.ReadFile(metaPath)
	switch {
	case os.IsNotExist(err):
		fmt.Fprintln(stdout, "Harness: not running (no active metadata)")
	case err != nil:
		fmt.Fprintf(stdout, "Harness: error reading metadata: %v\n", err)
	default:
		var ep harnessproto.ActiveHarnessEndpoint
		if jsonErr := json.Unmarshal(data, &ep); jsonErr != nil {
			fmt.Fprintf(stdout, "Harness: metadata is corrupt: %v\n", jsonErr)
		} else {
			fmt.Fprintln(stdout, "Harness: running")
			fmt.Fprintf(stdout, "  PID:              %d\n", ep.PID)
			fmt.Fprintf(stdout, "  Socket:           %s\n", ep.SocketPath)
			fmt.Fprintf(stdout, "  Protocol:         %s %s\n", ep.Protocol, ep.ProtocolVersion)
			if ep.SessionID != "" {
				fmt.Fprintf(stdout, "  Provider session: %s\n", ep.SessionID)
			}
			fmt.Fprintf(stdout, "  Started at:       %s\n", ep.StartedAt.UTC().Format(time.RFC3339))
		}
	}

	linked, err := link.ListLinkedProviders()
	if err != nil {
		fmt.Fprintf(stdout, "\nLinked providers: error: %v\n", err)
		return nil
	}

	fmt.Fprintf(stdout, "\nLinked providers: %d\n", len(linked))
	for _, p := range linked {
		fmt.Fprintf(stdout, "  %s\n", p)
	}

	return nil
}
