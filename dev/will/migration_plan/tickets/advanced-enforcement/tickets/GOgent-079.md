---
id: GOgent-079
title: Build gogent-orchestrator-guard CLI
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-078"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-079: Build gogent-orchestrator-guard CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-078

**Task**:
Build CLI binary for orchestrator-completion-guard hook.

**File**: `cmd/gogent-orchestrator-guard/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/enforcement"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event
	event, err := enforcement.ParseOrchestratorStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only process orchestrator/architect agents
	if !event.IsOrchestratorType() {
		// Silent pass-through for other agents
		fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow"
  }
}`)
		os.Exit(0)
	}

	// Analyze transcript if available
	var analyzer *enforcement.TranscriptAnalyzer
	if event.TranscriptPath != "" {
		analyzer = enforcement.NewTranscriptAnalyzer(event.TranscriptPath)
		if err := analyzer.Analyze(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to analyze transcript: %v\n", err)
			analyzer = &enforcement.TranscriptAnalyzer{
				tracker: &enforcement.TaskTracker{},
			}
		}
	}

	// Generate guard response
	response := enforcement.GenerateGuardResponse(analyzer, event)

	// Output response
	fmt.Println(response.FormatJSON())
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build Script**: `scripts/build-orchestrator-guard.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-orchestrator-guard..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-orchestrator-guard ./cmd/gogent-orchestrator-guard

echo "✓ Built: bin/gogent-orchestrator-guard"
```

**Acceptance Criteria**:
- [ ] CLI reads SubagentStop events
- [ ] Passes through non-orchestrator agents
- [ ] Analyzes transcript for orchestrator agents
- [ ] Generates block/allow decision
- [ ] Outputs valid hook response
- [ ] Build script creates executable

**Why This Matters**: CLI is orchestrator-completion-guard hook implementation.

---

## Part 2: Detect-Documentation-Theater (GOgent-080 to 086)
