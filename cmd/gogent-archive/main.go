package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

const DEFAULT_TIMEOUT = 5 * time.Second

func main() {
	if err := run(); err != nil {
		outputError(err.Error())
		os.Exit(1)
	}
}

func run() error {
	// Determine project directory from env or cwd
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("[gogent-archive] Failed to get working directory: %w. Set GOGENT_PROJECT_DIR environment variable or run from project root.", err)
		}
		projectDir = cwd
	}

	// Parse SessionEnd event from STDIN with timeout
	event, err := session.ParseSessionEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to parse SessionEnd event: %w. Ensure hook provides valid JSON on STDIN.", err)
	}

	// Collect session metrics
	metrics, err := session.CollectSessionMetrics(event.SessionID)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to collect metrics for session %s: %w. Check temp files exist and are readable.", event.SessionID, err)
	}

	// Generate JSONL handoff
	handoffCfg := session.DefaultHandoffConfig(projectDir)
	if err := session.GenerateHandoff(handoffCfg, metrics); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to generate handoff: %w", err)
	}

	// Load generated handoff for markdown rendering
	handoff, err := session.LoadHandoff(handoffCfg.HandoffPath)
	if err != nil {
		return fmt.Errorf("[gogent-archive] Failed to load handoff from %s: %w. Verify GenerateHandoff succeeded.", handoffCfg.HandoffPath, err)
	}

	if handoff == nil {
		return fmt.Errorf("[gogent-archive] No handoff data in %s. This may be normal for first session. Cannot generate markdown for empty handoff.", handoffCfg.HandoffPath)
	}

	// Render markdown for human consumption
	mdPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
	if err := os.MkdirAll(filepath.Dir(mdPath), 0755); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to create directory for %s: %w", mdPath, err)
	}
	markdown := session.RenderHandoffMarkdown(handoff)
	if err := os.WriteFile(mdPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to write markdown to %s: %w", mdPath, err)
	}

	// Archive artifacts AFTER handoff generation
	if err := session.ArchiveArtifacts(*handoffCfg, event.SessionID); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to archive artifacts: %w", err)
	}

	// Output confirmation JSON matching bash hook format
	confirmation := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": fmt.Sprintf("📦 SESSION ARCHIVED: Handoff saved to %s. JSONL history at %s.", mdPath, handoffCfg.HandoffPath),
			"handoff_jsonl":     handoffCfg.HandoffPath,
			"handoff_md":        mdPath,
			"session_id":        event.SessionID,
			"metrics": map[string]int{
				"tool_calls": metrics.ToolCalls,
				"errors":     metrics.ErrorsLogged,
				"violations": metrics.RoutingViolations,
			},
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(confirmation); err != nil {
		return fmt.Errorf("[gogent-archive] Failed to encode confirmation JSON: %w. Check stdout is writable.", err)
	}

	return nil
}

// outputError writes error message in hook-compatible JSON format
func outputError(message string) {
	output := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":     "SessionEnd",
			"additionalContext": "🔴 " + message,
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	_ = encoder.Encode(output) // Best effort - if this fails, nothing we can do
}
