---
id: GOgent-072
title: Build gogent-attention-gate CLI
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-071"]
priority: high
week: 4
tags: ["attention-gate", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-072: Build gogent-attention-gate CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-071

**Task**:
Build CLI binary for attention-gate hook.

**File**: `cmd/gogent-attention-gate/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/observability"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Get project directory
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}

	// Parse PostToolUse event
	event, err := observability.ParsePostToolUseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Increment tool counter
	counter := observability.NewToolCounter()
	currentCount, err := counter.Increment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to increment counter: %v\n", err)
		// Non-fatal - continue
		currentCount = 0
	}

	// Check if reminder should be injected
	var reminderMsg string
	if counter.ShouldRemind(currentCount) {
		// Load routing summary (simplified)
		summary := "haiku: find, search... sonnet: implement... (see routing-schema.json)"
		reminderMsg = observability.GenerateRoutingReminder(currentCount, summary)
	}

	// Check if flush should happen
	var flushMsg string
	if counter.ShouldFlush(currentCount) {
		shouldFlush, _, _ := observability.ShouldFlushLearnings(projectDir)
		if shouldFlush {
			ctx, err := observability.ArchivePendingLearnings(projectDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to flush learnings: %v\n", err)
			} else {
				flushMsg = observability.GenerateFlushNotification(ctx)
			}
		}
	}

	// Generate response
	response := observability.GenerateGateResponse(
		reminderMsg != "",
		flushMsg != "",
		reminderMsg,
		flushMsg,
	)

	// Output
	fmt.Println(response)
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build Script**: `scripts/build-attention-gate.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-attention-gate..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-attention-gate ./cmd/gogent-attention-gate

echo "✓ Built: bin/gogent-attention-gate"
```

**Acceptance Criteria**:
- [ ] CLI reads PostToolUse events from STDIN
- [ ] Increments tool counter
- [ ] Injects reminder every 10 tools
- [ ] Flushes pending learnings every 20 tools (if count >= 5)
- [ ] Generates valid hook response JSON
- [ ] Build script creates executable
- [ ] Manual test successful

**Why This Matters**: CLI is PostToolUse hook implementation. Fires after every tool call to maintain routing discipline and prevent data loss.

---
