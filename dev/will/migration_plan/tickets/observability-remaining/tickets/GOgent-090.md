---
id: GOgent-090
title: Build gogent-benchmark-logger CLI
description: **Task**:
status: pending
time_estimate: 1h
dependencies: ["GOgent-089"]
priority: high
week: 4
tags: ["benchmark-logger", "week-4"]
tests_required: true
acceptance_criteria_count: 5
---

### GOgent-090: Build gogent-benchmark-logger CLI

**Time**: 1 hour
**Dependencies**: GOgent-089

**Task**:
Build CLI binary for benchmark-logger hook.

**File**: `cmd/gogent-benchmark-logger/main.go`

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
	// Parse PostToolUse event
	event, err := observability.ParseBenchmarkEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		// Non-fatal - just skip logging
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse event: %v\n", err)
		os.Exit(0)
	}

	// Log metrics
	if err := observability.LogBenchmark(event); err != nil {
		// Non-fatal - just warn
		fmt.Fprintf(os.Stderr, "Warning: Failed to log benchmark: %v\n", err)
	}

	// Silent - no response needed
	// Benchmark logging is purely observational
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse"
  }
}`)
}
```

**Build Script**: `scripts/build-benchmark-logger.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-benchmark-logger..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-benchmark-logger ./cmd/gogent-benchmark-logger

echo "✓ Built: bin/gogent-benchmark-logger"
```

**Acceptance Criteria**:
- [ ] CLI reads PostToolUse events
- [ ] Logs metrics silently (no response needed)
- [ ] Handles missing/malformed events gracefully
- [ ] Build script creates executable
- [ ] Warnings logged to stderr

**Why This Matters**: CLI is benchmark-logger hook implementation. Passively logs metrics for later analysis.

---

## Part 2: Stop-Gate Investigation (GOgent-091 to 093)
