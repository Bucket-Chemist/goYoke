---
id: GOgent-067
title: Build gogent-agent-endstate CLI
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-066"]
priority: high
week: 4
tags: ["agent-endstate", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-067: Build gogent-agent-endstate CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-066

**Task**:
Build CLI binary that reads SubagentStop events and generates follow-up responses.

**File**: `cmd/gogent-agent-endstate/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent/pkg/workflow"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event
	event, err := workflow.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Generate response
	response := workflow.GenerateEndstateResponse(event)

	// Log decision
	if err := workflow.LogEndstate(event, response); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to log endstate: %v\n", err)
		// Don't exit - non-fatal
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

**Build Script**: `scripts/build-agent-endstate.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-agent-endstate..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-agent-endstate ./cmd/gogent-agent-endstate

echo "✓ Built: bin/gogent-agent-endstate"
```

**Acceptance Criteria**:
- [ ] CLI reads SubagentStop events from STDIN
- [ ] Generates tier-specific responses
- [ ] Logs decisions to JSONL
- [ ] Outputs valid JSON response
- [ ] Build script creates executable
- [ ] Warnings logged to stderr, not stdout
- [ ] Manual test: `echo '{"agent_id":"orchestrator",...}' | ./bin/gogent-agent-endstate`

**Why This Matters**: CLI is SubagentStop hook implementation. Must generate appropriate follow-up for each agent type.

---

## Part 2: Attention-Gate Hook (GOgent-068 to 074)
