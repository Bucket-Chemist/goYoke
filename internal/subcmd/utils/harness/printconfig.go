package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	pkgconfig "github.com/Bucket-Chemist/goYoke/pkg/config"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// printConfigOutput is the machine-readable output shape for print-config.
type printConfigOutput struct {
	Provider        string   `json:"provider"`
	SupportLevel    string   `json:"support_level"`
	Capabilities    []string `json:"capabilities"`
	Protocol        string   `json:"protocol"`
	ProtocolVersion string   `json:"protocol_version"`
	// SocketPath is the path to the live harness control socket.
	// Empty when the harness is not currently running.
	SocketPath string `json:"socket_path"`
}

// RunPrintConfig emits a machine-readable JSON configuration snippet for the
// given provider. Intended for external harnesses wiring themselves manually.
//
// Usage: goyoke harness print-config <provider>
//
// SocketPath is populated from active metadata when the harness is running;
// it is left empty otherwise — the external harness should wait for the TUI
// to start before connecting.
func RunPrintConfig(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: goyoke harness print-config <provider>")
	}
	providerName := args[0]

	p, ok := defaultRegistry.Get(providerName)
	if !ok {
		return fmt.Errorf("unknown provider %q; run 'goyoke harness list' to see available providers", providerName)
	}

	socketPath := ""
	metaPath := pkgconfig.GetHarnessActiveMetadataPath()
	if data, err := os.ReadFile(metaPath); err == nil {
		var ep harnessproto.ActiveHarnessEndpoint
		if json.Unmarshal(data, &ep) == nil {
			socketPath = ep.SocketPath
		}
	}

	out := printConfigOutput{
		Provider:        p.Name,
		SupportLevel:    string(p.SupportLevel),
		Capabilities:    p.Capabilities,
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		SocketPath:      socketPath,
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
