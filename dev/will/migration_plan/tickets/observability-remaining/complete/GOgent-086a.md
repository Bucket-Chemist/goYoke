---
id: GOgent-086a
title: Add config.GetGOgentDataDir() for XDG_DATA_HOME
description: Add XDG_DATA_HOME compliant directory helper for persistent data files (ML telemetry, training datasets, long-term logs)
type: implementation
status: pending
time_estimate: 30m
dependencies: []
priority: high
week: 4
tags: ["config", "xdg", "ml-optimization", "week-4"]
tests_required: true
acceptance_criteria_count: 7
---

### GOgent-086a: Add config.GetGOgentDataDir() for XDG_DATA_HOME

**Time**: 30 minutes
**Dependencies**: None

**Task**:
Add XDG_DATA_HOME compliant directory helper for persistent data files.

**Rationale**:
ML telemetry files are persistent training data, not cache. Per XDG spec:
- XDG_CACHE_HOME: Non-essential cached data (current GetGOgentDir)
- XDG_DATA_HOME: Portable user data (needed for ML logs)

**File**: `pkg/config/paths.go`

**Implementation**:
```go
// GetGOgentDataDir returns XDG-compliant data directory for persistent files.
// Priority: XDG_DATA_HOME > ~/.local/share/gogent
// Use for: ML telemetry, training datasets, long-term logs
func GetGOgentDataDir() string {
    if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        if err := os.MkdirAll(dir, 0755); err == nil {
            return dir
        }
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return filepath.Join(os.TempDir(), "gogent-data")
    }
    dir := filepath.Join(home, ".local", "share", "gogent")
    os.MkdirAll(dir, 0755)
    return dir
}

// GetMLToolEventsPath returns path for ML tool events log.
func GetMLToolEventsPath() string {
    return filepath.Join(GetGOgentDataDir(), "tool-events.jsonl")
}

// GetRoutingDecisionsPath returns path for routing decisions log.
func GetRoutingDecisionsPath() string {
    return filepath.Join(GetGOgentDataDir(), "routing-decisions.jsonl")
}

// GetCollaborationsPath returns path for agent collaborations log.
func GetCollaborationsPath() string {
    return filepath.Join(GetGOgentDataDir(), "agent-collaborations.jsonl")
}
```

**Tests**: `pkg/config/paths_test.go`

```go
func TestGetGOgentDataDir_XDGSet(t *testing.T) {
    origXDG := os.Getenv("XDG_DATA_HOME")
    defer os.Setenv("XDG_DATA_HOME", origXDG)

    testPath := t.TempDir()
    os.Setenv("XDG_DATA_HOME", testPath)

    dir := GetGOgentDataDir()
    expected := filepath.Join(testPath, "gogent")

    if dir != expected {
        t.Errorf("Expected %s, got %s", expected, dir)
    }
}

func TestGetGOgentDataDir_Fallback(t *testing.T) {
    origXDG := os.Getenv("XDG_DATA_HOME")
    defer os.Setenv("XDG_DATA_HOME", origXDG)

    os.Unsetenv("XDG_DATA_HOME")

    dir := GetGOgentDataDir()
    home, _ := os.UserHomeDir()
    expected := filepath.Join(home, ".local", "share", "gogent")

    if dir != expected {
        t.Errorf("Expected %s, got %s", expected, dir)
    }
}

func TestGetMLToolEventsPath(t *testing.T) {
    path := GetMLToolEventsPath()
    if !strings.HasSuffix(path, "tool-events.jsonl") {
        t.Errorf("Path should end with tool-events.jsonl, got %s", path)
    }
}
```

**Acceptance Criteria**:
- [x] GetGOgentDataDir() implemented
- [x] Respects XDG_DATA_HOME environment variable
- [x] Falls back to ~/.local/share/gogent
- [x] Creates directory if not exists
- [x] Separate from GetGOgentDir() (data vs cache)
- [x] Path helper functions for each log type (GetMLToolEventsPath, GetRoutingDecisionsPath, GetCollaborationsPath)
- [x] Unit tests cover both XDG and fallback paths with ≥90% coverage

**Why This Matters**: XDG compliance ensures ML telemetry data is stored in the correct location for persistent user data, separate from cache files that may be cleared.
