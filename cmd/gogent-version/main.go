package main

import (
	"encoding/json"
	"fmt"
	"io"

	versionlib "github.com/Bucket-Chemist/GOgent-Fortress/internal/hooks/version"
)

const Version = "0.1.0"

// BuildTime can be overridden at build time:
//
//	go build -ldflags "-X main.BuildTime=$(date -u +%FT%TZ)"
var BuildTime = "development"

type versionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
}

func run(w io.Writer, jsonOutput bool) error {
	if jsonOutput {
		info := versionInfo{
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

	_, err := fmt.Fprintf(w, "GOgent-Fortress v%s (built: %s)\n", Version, BuildTime)
	return err
}

func main() {
	// Propagate cmd-level BuildTime (set via ldflags) to the library.
	versionlib.BuildTime = BuildTime
	versionlib.Main()
}
