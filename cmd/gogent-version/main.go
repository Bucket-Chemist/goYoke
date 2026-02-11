package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

const Version = "0.1.0"

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
	jsonFlag := flag.Bool("json", false, "Output version information as JSON")
	flag.Parse()

	if err := run(os.Stdout, *jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
