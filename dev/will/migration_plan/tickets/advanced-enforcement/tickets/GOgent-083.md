---
id: GOgent-083
title: Build gogent-doc-theater CLI
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-082", "GOgent-080"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-083: Build gogent-doc-theater CLI

**Time**: 2 hours
**Dependencies**: GOgent-082, GOgent-080

**Task**:
Build CLI binary for detect-documentation-theater hook. Uses helpers from GOgent-080 for file detection and content extraction.

**CRITICAL FIXES**:
1. Parse STDIN once (existing ParseToolEvent)
2. Extract content from tool_input field (NOT re-reading STDIN)
3. Add missing strings import
4. Support configurable blocking via environment variable
5. Consider integration into gogent-validate (alternative approach)

**File**: `cmd/gogent-doc-theater/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)
```

**Implementation**:
```go
const (
	DEFAULT_TIMEOUT = 5 * time.Second
)

func main() {
	// Parse ToolEvent ONCE from STDIN
	event, err := routing.ParseToolEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only check Write/Edit operations on CLAUDE.md files
	// Uses helpers from GOgent-080
	if !event.IsClaudeMDFile() || !event.IsWriteOperation() {
		outputAllow()
		return
	}

	// Extract content from tool_input (NOT re-reading STDIN)
	// Uses ExtractWriteContent() from GOgent-080
	content := event.ExtractWriteContent()
	if content == "" {
		outputAllow() // No content to analyze
		return
	}

	// Detect patterns
	pd := routing.NewPatternDetector()
	results := pd.Detect(content)

	// Generate response (blocking configurable via env)
	response := generateDocTheaterResponse(event, results)
	fmt.Println(response)
}

func generateDocTheaterResponse(event *routing.ToolEvent, results []routing.DetectionResult) string {
	if len(results) == 0 {
		// No patterns detected
		return allowResponse()
	}

	// Build pattern summary
	var descriptions []string
	hasCritical := false
	for _, result := range results {
		descriptions = append(descriptions, result.Description)
		if result.Severity == "critical" {
			hasCritical = true
		}
	}

	warning := routing.GenerateWarning(results, event.FilePath)

	// Check if blocking is enabled (default: warn only)
	blockEnabled := os.Getenv("GOGENT_DOC_THEATER_BLOCK") == "true"
	if blockEnabled && hasCritical {
		return blockResponse(warning)
	}

	return warnResponse(warning)
}

func allowResponse() string {
	return `{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`
}

func warnResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "warn",
    "additionalContext": "%s"
  }
}`, escapeJSON(message))
}

func blockResponse(message string) string {
	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "block",
    "additionalContext": "🚫 BLOCKED: %s"
  }
}`, escapeJSON(message))
}

func outputAllow() {
	fmt.Println(allowResponse())
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build Script**: `scripts/build-doc-theater.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-doc-theater..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-doc-theater ./cmd/gogent-doc-theater

echo "✓ Built: bin/gogent-doc-theater"
```

**Acceptance Criteria**:
- [ ] CLI parses STDIN once (using routing.ParseToolEvent)
- [ ] Passes through non-CLAUDE.md operations
- [ ] Extracts content from tool_input field (NOT re-reading STDIN)
- [ ] Detects theater patterns using routing.PatternDetector
- [ ] Generates warning response by default
- [ ] Supports GOGENT_DOC_THEATER_BLOCK=true for blocking mode
- [ ] Outputs valid hook response JSON
- [ ] Build script creates executable

**Integration Note**:
This CLI can be used standalone OR integrated into gogent-validate. If integrating:

```go
// In cmd/gogent-validate/main.go
if event.ToolName == "Task" {
    result := orchestrator.ValidateTask(event)
} else if event.IsClaudeMDFile() && event.IsWriteOperation() {
    result := routing.DetectDocTheater(event)
}
```

Trade-offs:
- **Standalone**: Separate hook configuration, dedicated binary
- **Integrated**: Single PreToolUse hook, unified validation logic

**Hook Configuration** (Standalone):

```toml
# May conflict with existing PreToolUse hook (gogent-validate)
# Consider chaining hooks or integration approach
[hooks.PreToolUse]
command = "gogent-doc-theater"
```

**Environment Variables**:
- `GOGENT_DOC_THEATER_BLOCK`: Set to "true" to block critical patterns (default: warn only)

**Why This Matters**: Detects documentation theater patterns before they reach CLAUDE.md, enforcing the principle that enforcement goes in hooks/code, not documentation.

---
