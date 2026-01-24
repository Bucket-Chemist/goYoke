---
id: GOgent-079
title: Build gogent-orchestrator-guard CLI
description: **Task**: Build CLI binary for orchestrator-completion-guard hook using existing routing package functions.
status: pending
time_estimate: 2h
dependencies: ["GOgent-078"]
priority: high
week: 5
tags: ["orchestrator-guard", "week-5"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-079: Build gogent-orchestrator-guard CLI

**Time**: 2 hours
**Dependencies**: GOgent-078 (task tracking analysis functions)

**Task**:
Build CLI binary for orchestrator-completion-guard hook. This hook intercepts SubagentStop events for orchestrator/architect agents and verifies proper background task collection before allowing completion.

**CRITICAL FIXES FROM REVIEW**:
1. ✅ Use existing `routing.ParseSubagentStopEvent()` for event parsing
2. ✅ Use existing `routing.ParseTranscriptForMetadata()` for agent metadata extraction
3. ✅ Use existing `routing.GetAgentClass()` for agent type detection
4. ✅ Follow `cmd/gogent-agent-endstate/main.go` CLI patterns
5. ⚠️  Requires GOgent-078 task tracking functions (from `pkg/enforcement` package)

---

## Implementation

**File**: `cmd/gogent-orchestrator-guard/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/enforcement"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse SubagentStop event using EXISTING routing function
	event, err := routing.ParseSubagentStopEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Extract agent metadata from transcript using EXISTING routing function
	metadata, parseErr := routing.ParseTranscriptForMetadata(event.TranscriptPath)
	if parseErr != nil {
		// Non-fatal: graceful degradation (allow completion if can't parse)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Failed to parse transcript: %v\n", parseErr)
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Allowing completion (parsing failure is non-blocking)\n")
		outputAllow("Transcript parsing failed - allowing by default")
		return
	}

	// Check if orchestrator type using EXISTING GetAgentClass function
	agentClass := routing.GetAgentClass(metadata.AgentID)
	if agentClass != routing.ClassOrchestrator {
		// Silent pass-through for non-orchestrator agents
		outputAllow(fmt.Sprintf("Non-orchestrator agent (%s) - no guard needed", metadata.AgentID))
		return
	}

	// Analyze transcript for background task tracking
	// REQUIRES: GOgent-078 enforcement.NewTranscriptAnalyzer() implementation
	analyzer := enforcement.NewTranscriptAnalyzer(event.TranscriptPath)
	if err := analyzer.Analyze(); err != nil {
		fmt.Fprintf(os.Stderr, "[orchestrator-guard] Warning: Analysis failed: %v\n", err)
		outputAllow("Analysis failed - allowing by default")
		return
	}

	// Generate guard response (block if uncollected tasks)
	// REQUIRES: GOgent-077 enforcement.GenerateGuardResponse() implementation
	response := enforcement.GenerateGuardResponse(analyzer, metadata.AgentID)

	// Output response as JSON to stdout
	if err := response.Marshal(os.Stdout); err != nil {
		outputError(fmt.Sprintf("Failed to marshal response: %v", err))
		os.Exit(1)
	}
}

func outputAllow(reason string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "reason": "%s"
  }
}`, escapeJSON(reason))
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SubagentStop",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, escapeJSON(message))
}

func escapeJSON(s string) string {
	// Basic JSON escaping for embedded strings
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
```

**Missing import**: Add `"strings"` to imports for `escapeJSON`.

---

## Build Configuration

**Build Script**: `scripts/build-orchestrator-guard.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-orchestrator-guard..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-orchestrator-guard ./cmd/gogent-orchestrator-guard

if [[ $? -eq 0 ]]; then
    echo "✓ Built: bin/gogent-orchestrator-guard"
    echo "  Size: $(du -h bin/gogent-orchestrator-guard | cut -f1)"
else
    echo "✗ Build failed"
    exit 1
fi
```

**Makefile addition**:
```makefile
build-orchestrator-guard:
	@scripts/build-orchestrator-guard.sh

install-orchestrator-guard: build-orchestrator-guard
	@echo "Installing gogent-orchestrator-guard to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/gogent-orchestrator-guard ~/.local/bin/
	@chmod +x ~/.local/bin/gogent-orchestrator-guard
	@echo "✓ Installed: ~/.local/bin/gogent-orchestrator-guard"
```

---

## Hook Registration

**File**: `~/.config/claude/settings.json` (user configuration)

Add to hooks configuration:
```json
{
  "hooks": {
    "SubagentStop": {
      "command": "gogent-orchestrator-guard",
      "description": "Enforces background task collection before orchestrator completion"
    }
  }
}
```

**Alternative (absolute path)**:
```json
{
  "hooks": {
    "SubagentStop": {
      "command": "/home/username/.local/bin/gogent-orchestrator-guard"
    }
  }
}
```

**Note**: If using PATH-based lookup, ensure `~/.local/bin` is in PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

---

## Testing

**Manual test**:
```bash
# Test with mock SubagentStop event
echo '{
  "hook_event_name": "SubagentStop",
  "session_id": "test-session",
  "transcript_path": "/tmp/test-transcript.jsonl",
  "stop_hook_active": true
}' | ./bin/gogent-orchestrator-guard
```

**Expected behaviors**:
1. Non-orchestrator agents → `"decision": "allow"` (silent pass-through)
2. Orchestrator with all tasks collected → `"decision": "allow"`
3. Orchestrator with uncollected tasks → `"decision": "block"` + remediation steps
4. Transcript parsing failure → `"decision": "allow"` (graceful degradation)

---

## Acceptance Criteria

- [ ] CLI reads SubagentStop events using `routing.ParseSubagentStopEvent()`
- [ ] Extracts agent metadata using `routing.ParseTranscriptForMetadata()`
- [ ] Identifies orchestrator agents using `routing.GetAgentClass()`
- [ ] Silent pass-through for non-orchestrator agents
- [ ] Analyzes transcript for background task tracking (requires GOgent-078)
- [ ] Outputs valid hook response JSON
- [ ] Build script creates executable at `bin/gogent-orchestrator-guard`

**Blockers**:
- Requires GOgent-078 (`enforcement.NewTranscriptAnalyzer()`)
- Requires GOgent-077 (`enforcement.GenerateGuardResponse()`)

---

## Why This Matters

This CLI enforces the fan-out/fan-in pattern from `LLM-guidelines.md` § 2.2 MANDATORY:

> "If you spawn background tasks, you MUST call TaskOutput() before concluding."

**Without this hook**: Orchestrators can complete with orphaned background tasks, leaving work incomplete.

**With this hook**: System programmatically blocks completion until all background tasks are collected, preventing silent failures.

---

## References

- **Event parsing**: `pkg/routing/events.go` lines 220-244 (`ParseSubagentStopEvent`)
- **Metadata extraction**: `pkg/routing/events.go` lines 246-310 (`ParseTranscriptForMetadata`)
- **Agent classification**: `pkg/routing/events.go` lines 204-218 (`GetAgentClass`)
- **CLI pattern reference**: `cmd/gogent-agent-endstate/main.go`
- **Hook schema**: Claude Code SubagentStop event documentation

---

## Part 2: Detect-Documentation-Theater (GOgent-080 to 086)
