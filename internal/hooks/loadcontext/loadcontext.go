// Package loadcontext implements the gogent-load-context hook.
// It loads context at session start: routing schema, handoffs, git info.
package loadcontext

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

// DefaultTimeout is the read timeout for stdin events.
const DefaultTimeout = 5 * time.Second

// OutputError writes an error message in hook format to STDOUT.
func OutputError(message string) {
	fmt.Println(session.GenerateErrorResponse(message))
}

// Main is the entrypoint for the gogent-load-context hook.
func Main() {
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			OutputError(fmt.Sprintf("Failed to get working directory: %v", err))
			os.Exit(1)
		}
	}

	event, err := session.ParseSessionStartEvent(os.Stdin, DefaultTimeout)
	if err != nil {
		OutputError(fmt.Sprintf("Failed to parse SessionStart event: %v", err))
		os.Exit(1)
	}

	sessionDir, err := session.CreateSessionDir(projectDir, event.SessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: session dir: %v\n", err)
	} else {
		if err := session.WriteCurrentSession(projectDir, sessionDir); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: write current-session: %v\n", err)
		}
	}

	if sessionDir != "" {
		guardPath := filepath.Join(sessionDir, "active-skill.json")
		if _, err := os.Stat(guardPath); err == nil {
			os.Remove(guardPath)
			fmt.Fprintf(os.Stderr, "[gogent-load-context] Cleaned up stale active-skill.json\n")
		}
	}

	if err := config.InitializeToolCounter(); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to initialize tool counter: %v\n", err)
	}

	ctx := &session.ContextComponents{
		SessionType: event.Type,
		SessionDir:  sessionDir,
	}

	if summary, err := routing.LoadAndFormatSchemaSummary(); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: %v\n", err)
	} else {
		ctx.RoutingSummary = summary
	}

	if event.IsResume() {
		if handoff, err := session.LoadHandoffSummary(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to load handoff: %v\n", err)
		} else {
			ctx.HandoffSummary = handoff
		}
	}

	if pending, err := session.CheckPendingLearnings(projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-load-context] Warning: Failed to check pending learnings: %v\n", err)
	} else {
		ctx.PendingLearnings = pending
	}

	ctx.GitInfo = session.FormatGitInfo(projectDir)
	ctx.ProjectInfo = session.DetectProjectType(projectDir)

	response, err := session.GenerateSessionStartResponse(ctx)
	if err != nil {
		OutputError(fmt.Sprintf("Failed to generate response: %v", err))
		os.Exit(1)
	}

	fmt.Println(response)
}
