// Package logreview implements the gogent-log-review hook.
// It reads a review summary from stdin and logs findings to the telemetry system.
package logreview

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// ReviewInput represents the JSON input from /review skill.
type ReviewInput struct {
	SessionID     string         `json:"session_id"`
	ReviewScope   string         `json:"review_scope"`
	FilesReviewed int            `json:"files_reviewed"`
	Findings      []FindingInput `json:"findings"`
}

// FindingInput represents a single finding in the review input.
type FindingInput struct {
	Severity       string `json:"severity"`
	Reviewer       string `json:"reviewer"`
	Category       string `json:"category"`
	File           string `json:"file"`
	Line           int    `json:"line"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
	SharpEdgeID    string `json:"sharp_edge_id"`
}

// LogOutput represents the JSON output after logging findings.
type LogOutput struct {
	Logged         int      `json:"logged"`
	FindingIDs     []string `json:"finding_ids"`
	SharpEdgeHits  int      `json:"sharp_edge_hits"`
	InvalidEdgeIDs []string `json:"invalid_edge_ids,omitempty"`
}

// Main is the entrypoint for the gogent-log-review hook.
func Main() {
	// Read JSON from stdin
	var input ReviewInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input: %v\n", err)
		os.Exit(1)
	}

	// Load sharp edge registry for validation
	if err := telemetry.LoadSharpEdgeIDs(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load sharp edge registry: %v\n", err)
	}

	output := LogOutput{
		FindingIDs:     make([]string, 0, len(input.Findings)),
		InvalidEdgeIDs: make([]string, 0),
	}

	for _, f := range input.Findings {
		// Create and log finding
		finding := telemetry.NewReviewFinding(
			input.SessionID,
			f.Reviewer,
			f.Severity,
			f.Category,
			f.File,
			f.Line,
			f.Message,
		)
		finding.ReviewScope = input.ReviewScope
		finding.FilesReviewed = input.FilesReviewed
		finding.Recommendation = f.Recommendation
		finding.SharpEdgeID = f.SharpEdgeID

		if err := telemetry.LogReviewFinding(finding); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to log finding: %v\n", err)
			continue
		}

		output.FindingIDs = append(output.FindingIDs, finding.FindingID)
		output.Logged++

		// Log sharp edge hit if correlated (with validation)
		if f.SharpEdgeID != "" {
			hit, err := telemetry.NewSharpEdgeHit(
				input.SessionID,
				f.SharpEdgeID,
				f.Reviewer, // agent_id
				f.Reviewer, // reviewer_id
				finding.FindingID,
				f.File,
				f.Line,
			)
			if err != nil {
				output.InvalidEdgeIDs = append(output.InvalidEdgeIDs, f.SharpEdgeID)
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				continue
			}
			if err := telemetry.LogSharpEdgeHit(hit); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log sharp edge hit: %v\n", err)
			} else {
				output.SharpEdgeHits++
			}
		}
	}

	// Output summary
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(output)
}
