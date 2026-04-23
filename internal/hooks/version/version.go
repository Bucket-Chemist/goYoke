// Package version implements the goyoke-version hook.
// It reports the binary version and build time.
package version

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// Version is set at build time via -ldflags "-X .../internal/hooks/version.Version=...".
var Version = "dev"

// BuildTime is set at build time via -ldflags "-X .../internal/hooks/version.BuildTime=...".
var BuildTime = "development"

// VersionInfo is the JSON representation of version information.
type VersionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
}

// Run writes version info to w using the package-level BuildTime.
func Run(w io.Writer, jsonOutput bool) error {
	if jsonOutput {
		info := VersionInfo{
			Version:   Version,
			BuildTime: BuildTime,
		}
		data, err := json.Marshal(info)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = w.Write(append(data, '\n'))
		return err
	}

	ver := strings.TrimPrefix(Version, "v")
	_, err := fmt.Fprintf(w, "goYoke v%s (built: %s)\n", ver, BuildTime)
	return err
}

// Main is the entrypoint for the goyoke-version hook.
func Main() {
	jsonFlag := flag.Bool("json", false, "Output version information as JSON")
	flag.Parse()

	if err := Run(os.Stdout, *jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
