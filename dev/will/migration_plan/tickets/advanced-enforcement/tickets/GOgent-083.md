---
id: GOgent-083
title: Build gogent-doc-theater CLI
description: **Task**:
status: pending
time_estimate: 1.5h
dependencies: ["GOgent-082"]
priority: high
week: 5
tags: ["doc-theater", "week-5"]
tests_required: true
acceptance_criteria_count: 6
---

### GOgent-083: Build gogent-doc-theater CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-082

**Task**:
Build CLI binary for detect-documentation-theater hook.

**File**: `cmd/gogent-doc-theater/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"io"
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
	// Parse PreToolUse event
	event, err := enforcement.ParsePreToolUseEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Only check Write/Edit operations on CLAUDE.md files
	if !event.IsClaudeMDFile() || !event.IsWriteOperation() {
		// Silent pass-through for other operations
		fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`)
		os.Exit(0)
	}

	// If content passed via environment, scan it
	content := os.Getenv("TOOL_INPUT_CONTENT")
	if content == "" {
		// Try to read from stdin (after event)
		data, err := io.ReadAll(os.Stdin)
		if err == nil && len(data) > 0 {
			content = string(data)
		}
	}

	// Detect patterns
	pd := enforcement.NewPatternDetector()
	results := pd.Detect(content)

	// Generate response
	response := generateDocTheaterResponse(event, results)

	// Output response
	fmt.Println(response)
}

func generateDocTheaterResponse(event *enforcement.PreToolUseEvent, results []enforcement.DetectionResult) string {
	if len(results) == 0 {
		// No patterns detected
		return `{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "allow"
  }
}`
	}

	// Patterns detected - inject warning
	pd := enforcement.NewPatternDetector()
	content := ""
	for _, result := range results {
		content += result.Description + "\n"
	}

	warning := enforcement.GenerateWarning(results, event.FilePath)

	return fmt.Sprintf(`{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "decision": "warn",
    "additionalContext": "%s"
  }
}`, escapeJSON(warning))
}

func escapeJSON(s string) string {
	// Minimal escaping for JSON output
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

Note: Add `import "strings"` to imports.

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
- [ ] CLI reads PreToolUse events
- [ ] Passes through non-CLAUDE.md operations
- [ ] Detects theater patterns in content
- [ ] Generates warning response (not blocking)
- [ ] Outputs valid hook response
- [ ] Build script creates executable

**Why This Matters**: CLI is detect-documentation-theater hook implementation. Prevents anti-pattern before commit.

---
