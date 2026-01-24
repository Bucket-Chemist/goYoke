---
id: GOgent-067
title: Build gogent-agent-endstate CLI
description: "Build CLI binary that reads SubagentStop events, parses transcript for agent metadata, and generates follow-up responses with graceful degradation."
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-066"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 12
---

### GOgent-067: Build gogent-agent-endstate CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-066

**Task**:
Build CLI binary that reads SubagentStop events, parses transcript for agent metadata, and generates follow-up responses with graceful degradation.

**File**: `cmd/gogent-agent-endstate/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/workflow"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event (uses ACTUAL schema)
	event, err := workflow.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Parse transcript for agent metadata (CRITICAL: metadata not in event directly)
	metadata, parseErr := workflow.ParseTranscriptForMetadata(event.TranscriptPath)
	if parseErr != nil {
		// Non-fatal: transcript parsing failure
		// Continue with nil metadata - GenerateEndstateResponse will use defaults
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "Continuing with default agent metadata...\n")
	}

	// Generate response (accepts event + metadata, handles nil gracefully)
	response := workflow.GenerateEndstateResponse(event, metadata)

	// Log decision (non-blocking if fails)
	if err := workflow.LogEndstate(event, response); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log endstate: %v\n", err)
		// Don't exit - logging failure is non-fatal
	}

	// Output response
	fmt.Println(response.FormatJSON())
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "silent",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Makefile Integration**: Add to `Makefile`

```makefile
build-agent-endstate:
	go build -o bin/gogent-agent-endstate ./cmd/gogent-agent-endstate

build-all: build-validate build-archive build-sharp-edge build-agent-endstate

install: build-all
	cp bin/gogent-* ~/.local/bin/
```

**Test Verification**:
```bash
# Test with mock SubagentStop event (ACTUAL schema)
echo '{
  "hook_event_name": "SubagentStop",
  "session_id": "test-session-001",
  "transcript_path": "/tmp/test-transcript.jsonl",
  "stop_hook_active": true
}' | ./bin/gogent-agent-endstate

# Expected: Valid JSON response with decision and recommendations
# Warning messages to stderr are acceptable if transcript doesn't exist
```

**Acceptance Criteria**:
- [x] CLI reads SubagentStop events from STDIN (ACTUAL schema: session_id, transcript_path)
- [x] Parses transcript file for agent metadata (agent_id, model, tier, etc.)
- [x] Handles transcript parsing failures gracefully (warning to stderr, continue with defaults)
- [x] Generates tier-specific responses based on parsed metadata
- [x] Logs decisions to XDG-compliant JSONL path
- [x] Outputs valid JSON response to stdout
- [x] Makefile target `build-agent-endstate` added
- [x] Makefile target `build-all` includes agent-endstate
- [x] Warnings logged to stderr, not stdout (non-blocking)
- [x] Transcript parsing errors are non-fatal (graceful degradation)
- [x] Manual test passes with ACTUAL SubagentStop schema
- [x] `go build ./cmd/gogent-agent-endstate` succeeds

**Why This Matters**: CLI is SubagentStop hook implementation. Must generate appropriate follow-up for each agent type.

---

## Part 2: Attention-Gate Hook (GOgent-068 to 074)
