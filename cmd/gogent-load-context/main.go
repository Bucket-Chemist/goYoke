package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Get project directory (priority: GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > cwd)
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			outputError(fmt.Sprintf("Failed to get working directory: %v", err))
			os.Exit(1)
		}
	}

	// Parse SessionStart event from STDIN
	event, err := session.ParseSessionStartEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse SessionStart event: %v", err))
		os.Exit(1)
	}

	// Initialize tool counter for attention-gate hook
	if err := config.InitializeToolCounter(); err != nil {
		// Non-fatal - log warning and continue
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to initialize tool counter: %v\n", err)
	}

	// Build context components
	ctx := &session.ContextComponents{
		SessionType: event.Type,
	}

	// Load routing schema summary (non-fatal if missing)
	if summary, err := routing.LoadAndFormatSchemaSummary(); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: %v\n", err)
	} else {
		ctx.RoutingSummary = summary
	}

	// Load handoff for resume sessions only
	if event.IsResume() {
		if handoff, err := session.LoadHandoffSummary(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to load handoff: %v\n", err)
		} else {
			ctx.HandoffSummary = handoff
		}
	}

	// Check pending learnings
	if pending, err := session.CheckPendingLearnings(projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to check pending learnings: %v\n", err)
	} else {
		ctx.PendingLearnings = pending
	}

	// Get git info
	ctx.GitInfo = session.FormatGitInfo(projectDir)

	// Detect project type
	ctx.ProjectInfo = session.DetectProjectType(projectDir)

	// Generate response
	response, err := session.GenerateSessionStartResponse(ctx)
	if err != nil {
		outputError(fmt.Sprintf("Failed to generate response: %v", err))
		os.Exit(1)
	}

	// Output response to STDOUT
	fmt.Println(response)
}

// outputError writes error message in hook format to STDOUT
func outputError(message string) {
	fmt.Println(session.GenerateErrorResponse(message))
}
