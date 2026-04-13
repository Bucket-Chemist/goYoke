// Package updatereviewoutcome implements the gogent-update-review-outcome hook.
// It updates the outcome of a review finding in the telemetry system.
package updatereviewoutcome

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// Main is the entrypoint for the gogent-update-review-outcome hook.
func Main() {
	findingID := flag.String("finding-id", "", "Finding ID to update (required)")
	resolution := flag.String("resolution", "", "Resolution: fixed, wontfix, false_positive, deferred (required)")
	ticketID := flag.String("ticket-id", "", "Associated ticket ID (optional)")
	commit := flag.String("commit", "", "Commit hash that fixed it (optional)")
	flag.Parse()

	if *findingID == "" || *resolution == "" {
		fmt.Fprintln(os.Stderr, "Usage: gogent-update-review-outcome --finding-id=ID --resolution=TYPE [--ticket-id=ID] [--commit=HASH]")
		os.Exit(1)
	}

	// Validate resolution
	validResolutions := map[string]bool{
		"fixed": true, "wontfix": true, "false_positive": true, "deferred": true,
	}
	if !validResolutions[*resolution] {
		fmt.Fprintf(os.Stderr, "Invalid resolution: %s\n", *resolution)
		os.Exit(1)
	}

	// Calculate accurate resolution time by looking up original finding
	var resolutionMs int64 = 0
	if origTimestamp, err := telemetry.LookupFindingTimestamp(*findingID); err == nil {
		resolutionMs = (time.Now().Unix() - origTimestamp) * 1000
	} else {
		fmt.Fprintf(os.Stderr, "Warning: could not lookup finding timestamp: %v\n", err)
	}

	err := telemetry.UpdateReviewFindingOutcome(*findingID, *resolution, *ticketID, *commit, resolutionMs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated finding %s: resolution=%s, resolution_time=%dms\n", *findingID, *resolution, resolutionMs)
}
