---
id: GOgent-062
title: CLI Binary - Main Orchestrator
description: Build CLI binary that orchestrates SessionStart workflow
status: pending
time_estimate: 1.5h
dependencies:
  - GOgent-056
  - GOgent-057
  - GOgent-058
  - GOgent-059
  - GOgent-060
  - GOgent-061
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 20
---

## GOgent-062: CLI Binary - Main Orchestrator

**Time**: 1.5 hours
**Dependencies**: GOgent-056 to 061
**Priority**: HIGH

**Task**:
Build CLI binary that orchestrates SessionStart workflow.

**File**: `cmd/gogent-load-context/main.go` (new file)

**Implementation**:
```go
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
```

**Build Target**: Add to `Makefile`

```makefile
# Add to existing Makefile

build-load-context:
	@echo "Building gogent-load-context..."
	go build -o bin/gogent-load-context ./cmd/gogent-load-context
	@echo "✓ Built: bin/gogent-load-context"

install-load-context: build-load-context
	@echo "Installing gogent-load-context..."
	@mkdir -p $(HOME)/.local/bin
	cp bin/gogent-load-context $(HOME)/.local/bin/
	chmod +x $(HOME)/.local/bin/gogent-load-context
	@echo "✓ Installed: $(HOME)/.local/bin/gogent-load-context"
```

**Tests**: `cmd/gogent-load-context/main_test.go` (new file)

```go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
)

func TestMain_Startup(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Prepare input
	input := `{"type":"startup","session_id":"test-123","hook_event_name":"SessionStart"}`

	// Run binary
	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	// Verify valid JSON output
	var response session.SessionStartResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("Invalid JSON output: %v. Output: %s", err, string(output))
	}

	// Verify content
	if response.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("Expected hookEventName 'SessionStart', got: %s", response.HookSpecificOutput.HookEventName)
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "startup") {
		t.Error("Response should indicate startup session")
	}
}

func TestMain_Resume(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Create temp project with handoff
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	handoffContent := "# Test Handoff\n\nPrevious session info."
	os.WriteFile(filepath.Join(memoryDir, "last-handoff.md"), []byte(handoffContent), 0644)

	// Prepare input
	input := `{"type":"resume","session_id":"test-456","hook_event_name":"SessionStart"}`

	// Run binary with project directory
	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)
	cmd.Env = append(os.Environ(), "GOGENT_PROJECT_DIR="+tmpDir)

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	// Verify response includes handoff
	var response session.SessionStartResponse
	json.Unmarshal(output, &response)

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "resume") {
		t.Error("Response should indicate resume session")
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "PREVIOUS SESSION HANDOFF") {
		t.Error("Resume response should include handoff")
	}
}

func TestMain_InvalidInput(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Invalid JSON input
	input := "not valid json"

	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)

	output, _ := cmd.Output() // Exit code 1 expected

	// Should still output valid JSON error
	var response session.SessionStartResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("Error response should be valid JSON: %v", err)
	}

	if !strings.Contains(response.HookSpecificOutput.AdditionalContext, "ERROR") {
		t.Error("Should indicate error")
	}
}

func TestMain_ToolCounterInitialized(t *testing.T) {
	// Build binary
	buildCmd := exec.Command("go", "build", "-o", "test-gogent-load-context", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("test-gogent-load-context")

	// Use temp XDG directory
	tmpDir := t.TempDir()

	input := `{"type":"startup","session_id":"test-789","hook_event_name":"SessionStart"}`

	cmd := exec.Command("./test-gogent-load-context")
	cmd.Stdin = bytes.NewBufferString(input)
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+tmpDir)

	if _, err := cmd.Output(); err != nil {
		t.Fatalf("Binary execution failed: %v", err)
	}

	// Verify tool counter was created
	counterPath := filepath.Join(tmpDir, "gogent", "tool-counter")
	if _, err := os.Stat(counterPath); os.IsNotExist(err) {
		t.Error("Tool counter file should be created")
	}
}
```

**Acceptance Criteria**:
- [x] CLI reads SessionStart events from STDIN
- [x] Parses event with 5s timeout
- [x] Initializes tool counter (non-fatal if fails)
- [x] Loads routing schema summary (non-fatal if missing)
- [x] Loads handoff for resume sessions only
- [x] Checks pending learnings
- [x] Gets git status
- [x] Detects project type
- [x] Outputs valid JSON context injection
- [x] Warnings go to stderr, response goes to stdout
- [x] `make build-load-context` builds binary
- [x] `make install-load-context` installs to ~/.local/bin
- [x] All tests pass

**Test Deliverables**:
- [x] Test file created: `cmd/gogent-load-context/main_test.go`
- [x] Test file size: ~160 lines (actual: 166 lines)
- [x] Number of test functions: 4
- [x] Tests passing: ✅
- [x] Race detector clean: ✅
- [x] **ECOSYSTEM TEST PASS REQUIRED**: `make test-ecosystem` (gogent-load-context tests passed)
- [x] Ecosystem test output saved to: `test/audit/GOgent-062/`

**Why This Matters**: This is the SessionStart hook binary. It's the first code to run in every Claude Code session and sets up context for all downstream hooks.

---
