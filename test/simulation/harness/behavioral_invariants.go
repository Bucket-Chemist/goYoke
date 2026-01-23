package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BehavioralInvariant represents a property that must hold across sessions.
// Unlike per-execution invariants (Invariant type), these validate system-level
// properties that emerge from multi-turn interactions.
type BehavioralInvariant struct {
	ID    string
	Name  string
	Check func(ctx *BehavioralContext) (bool, string)
}

// BehavioralContext provides access to session state for invariant checking.
// Populated by the runner after session replay completes.
type BehavioralContext struct {
	// TempDir is the session's isolated directory
	TempDir string

	// SharpEdges contains parsed sharp edge records from pending-learnings.jsonl
	SharpEdges []map[string]interface{}

	// FailureTrackerLog contains parsed failure records from failure-tracker.jsonl
	FailureTrackerLog []map[string]interface{}

	// Handoff contains the last handoff record, if any
	Handoff map[string]interface{}

	// Config holds system configuration values
	Config BehavioralConfig
}

// BehavioralConfig holds configuration values for invariant checking.
type BehavioralConfig struct {
	// MaxFailures is the threshold for sharp edge detection (default: 3)
	MaxFailures int

	// SchemaVersion is the expected schema version for handoffs
	SchemaVersion string
}

// DefaultBehavioralConfig returns sensible defaults.
func DefaultBehavioralConfig() BehavioralConfig {
	return BehavioralConfig{
		MaxFailures:   3,
		SchemaVersion: "1.2",
	}
}

// BehavioralInvariants defines properties that must hold across the system.
//
// NOTE: B2 (code_snippet) and B3 (Category) are DEFERRED to ticket 042b.
// Rationale:
// - B2 requires GOgent-037b (code snippet extraction feature)
// - B3 requires GOgent-041 (intent category classification)
// Implementing stubs for unreleased features creates false confidence.
var BehavioralInvariants = []BehavioralInvariant{
	{
		// B1: Sharp edges have required fields for the learning pipeline
		// These fields are essential for downstream processing (archival, review)
		ID:   "B1",
		Name: "sharp_edges_have_required_fields",
		Check: func(ctx *BehavioralContext) (bool, string) {
			for i, edge := range ctx.SharpEdges {
				// error_type is required for categorization
				errorType, _ := edge["error_type"].(string)
				if errorType == "" {
					return false, fmt.Sprintf("sharp edge %d missing error_type", i)
				}

				// consecutive_failures must be >= threshold (3)
				// JSON numbers unmarshal as float64
				failures, ok := edge["consecutive_failures"].(float64)
				if !ok {
					// Try int conversion for robustness
					if failInt, ok := edge["consecutive_failures"].(int); ok {
						failures = float64(failInt)
					} else {
						return false, fmt.Sprintf("sharp edge %d: consecutive_failures not a number", i)
					}
				}
				if int(failures) < ctx.Config.MaxFailures {
					return false, fmt.Sprintf("sharp edge %d has consecutive_failures=%.0f, want >= %d",
						i, failures, ctx.Config.MaxFailures)
				}

				// Timestamp is required for ordering and expiry
				// Accept both "timestamp" and "ts" field names
				timestamp, _ := edge["timestamp"].(float64)
				if timestamp == 0 {
					timestamp, _ = edge["ts"].(float64)
				}
				if timestamp == 0 {
					// Check string timestamp format
					if tsStr, ok := edge["timestamp"].(string); ok && tsStr != "" {
						// Has timestamp as string - acceptable
					} else if tsStr, ok := edge["ts"].(string); ok && tsStr != "" {
						// Has ts as string - acceptable
					} else {
						return false, fmt.Sprintf("sharp edge %d missing timestamp", i)
					}
				}
			}
			return true, ""
		},
	},
	{
		// B4: Cross-session handoffs preserve schema version
		// Schema version drift can break the memory system
		ID:   "B4",
		Name: "handoff_preserves_schema_version",
		Check: func(ctx *BehavioralContext) (bool, string) {
			if ctx.Handoff == nil || len(ctx.Handoff) == 0 {
				// No handoff in this session - invariant vacuously holds
				return true, ""
			}

			version, ok := ctx.Handoff["schema_version"].(string)
			if !ok {
				return false, "schema_version field missing or not a string"
			}
			if version != ctx.Config.SchemaVersion {
				return false, fmt.Sprintf("schema_version=%q, want %q", version, ctx.Config.SchemaVersion)
			}
			return true, ""
		},
	},
	{
		// B5: Failure tracker counts are accurate per composite key
		// The composite key is file:error_type - counts must match between tracker and edges
		ID:   "B5",
		Name: "failure_tracker_counts_accurate",
		Check: func(ctx *BehavioralContext) (bool, string) {
			// Count failures by composite key from tracker log
			keyCounts := make(map[string]int)
			for _, entry := range ctx.FailureTrackerLog {
				file, _ := entry["file"].(string)
				errorType, _ := entry["error_type"].(string)
				// Skip entries with missing required fields
				if file == "" || errorType == "" {
					continue
				}
				key := file + ":" + errorType
				keyCounts[key]++
			}

			// Verify sharp edges match expected counts
			for _, edge := range ctx.SharpEdges {
				file, _ := edge["file"].(string)
				errorType, _ := edge["error_type"].(string)
				if file == "" || errorType == "" {
					continue // Already caught by B1
				}
				key := file + ":" + errorType

				recorded, _ := edge["consecutive_failures"].(float64)
				tracked := keyCounts[key]

				// Sharp edge should be created exactly when threshold is reached
				// The recorded count in the sharp edge should match or exceed threshold
				if tracked >= ctx.Config.MaxFailures {
					if int(recorded) < ctx.Config.MaxFailures {
						return false, fmt.Sprintf("key %s: edge has %.0f failures but tracker shows %d (threshold: %d)",
							key, recorded, tracked, ctx.Config.MaxFailures)
					}
				}
			}
			return true, ""
		},
	},
	{
		// B6: Blocking occurs exactly at threshold, not before
		// Critical for UX - premature blocking frustrates users; late blocking misses the loop
		ID:   "B6",
		Name: "blocking_at_exact_threshold",
		Check: func(ctx *BehavioralContext) (bool, string) {
			// This invariant is primarily validated during ReplaySession execution
			// via per-event expected_decision checks.
			//
			// The post-hoc check here validates consistency:
			// - If we have sharp edges, we should have had blocking
			// - Sharp edges should not exist without reaching threshold

			for _, edge := range ctx.SharpEdges {
				failures, _ := edge["consecutive_failures"].(float64)
				if int(failures) < ctx.Config.MaxFailures {
					// Sharp edge exists but threshold not reached - violation
					return false, fmt.Sprintf("sharp edge exists with only %.0f failures (threshold: %d)",
						failures, ctx.Config.MaxFailures)
				}
			}
			return true, ""
		},
	},
	{
		// B7: JSONL files remain valid JSON after writes (no corruption)
		// Concurrent writes must not corrupt the file format
		ID:   "B7",
		Name: "jsonl_files_valid_after_writes",
		Check: func(ctx *BehavioralContext) (bool, string) {
			// Check all JSONL files the system writes to
			jsonlFiles := []string{
				".claude/memory/pending-learnings.jsonl",
				".claude/memory/handoffs.jsonl",
				".gogent/failure-tracker.jsonl",
			}

			for _, relPath := range jsonlFiles {
				fullPath := filepath.Join(ctx.TempDir, relPath)
				content, err := os.ReadFile(fullPath)
				if os.IsNotExist(err) {
					continue // File not created is acceptable
				}
				if err != nil {
					return false, fmt.Sprintf("cannot read %s: %v", relPath, err)
				}

				// Empty file is valid
				if len(strings.TrimSpace(string(content))) == 0 {
					continue
				}

				// Each non-empty line must be valid JSON
				lines := strings.Split(string(content), "\n")
				for i, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					var obj interface{}
					if err := json.Unmarshal([]byte(line), &obj); err != nil {
						// Include first 100 chars of invalid line for debugging
						preview := line
						if len(preview) > 100 {
							preview = preview[:100] + "..."
						}
						return false, fmt.Sprintf("%s line %d: invalid JSON: %v (content: %s)",
							relPath, i+1, err, preview)
					}
				}
			}
			return true, ""
		},
	},
}

// FUTURE INVARIANTS (add in 042b when dependencies complete)
//
// B2: sharp_edges_have_code_snippet
// Requires: GOgent-037b (code snippet extraction)
// Check: edge["code_snippet"] exists OR edge["code_snippet_skip_reason"] exists
// Rationale: Code context is critical for learning review
//
// B3: user_intents_have_category
// Requires: GOgent-041 (intent category classification)
// Check: intent["category"] is non-empty string
// Rationale: Categories enable preference drift detection

// LoadBehavioralContext populates context from a session's temp directory.
func LoadBehavioralContext(tempDir string, config BehavioralConfig) (*BehavioralContext, error) {
	ctx := &BehavioralContext{
		TempDir: tempDir,
		Config:  config,
	}

	// Load sharp edges
	pendingPath := filepath.Join(tempDir, ".claude", "memory", "pending-learnings.jsonl")
	if edges, err := loadJSONLFile(pendingPath); err == nil {
		ctx.SharpEdges = edges
	}

	// Load failure tracker
	trackerPath := filepath.Join(tempDir, ".gogent", "failure-tracker.jsonl")
	if entries, err := loadJSONLFile(trackerPath); err == nil {
		ctx.FailureTrackerLog = entries
	}

	// Load last handoff
	handoffPath := filepath.Join(tempDir, ".claude", "memory", "handoffs.jsonl")
	if handoffs, err := loadJSONLFile(handoffPath); err == nil && len(handoffs) > 0 {
		ctx.Handoff = handoffs[len(handoffs)-1] // Last entry is most recent
	}

	return ctx, nil
}

// loadJSONLFile parses a JSONL file into a slice of maps.
func loadJSONLFile(path string) ([]map[string]interface{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// Skip invalid lines (may be partial writes)
			continue
		}
		results = append(results, obj)
	}
	return results, nil
}

// CheckBehavioralInvariants runs all behavioral invariants against a context.
func CheckBehavioralInvariants(ctx *BehavioralContext) []InvariantResult {
	var results []InvariantResult

	for _, inv := range BehavioralInvariants {
		passed, message := inv.Check(ctx)
		results = append(results, InvariantResult{
			InvariantID: inv.ID,
			Passed:      passed,
			Message:     message,
		})
	}

	return results
}

// BehavioralInvariantByID returns the invariant with the given ID, or nil if not found.
func BehavioralInvariantByID(id string) *BehavioralInvariant {
	for i := range BehavioralInvariants {
		if BehavioralInvariants[i].ID == id {
			return &BehavioralInvariants[i]
		}
	}
	return nil
}
