# Week 1 Day 3-5: Escape Hatches, Complexity Routing, Tool Permissions

**File**: `02-week1-overrides-permissions.md`
**Tickets**: GOgent-010 to 019 (10 tickets)
**Total Time**: ~13 hours
**Phase**: Week 1 Day 3-5

---

## Navigation

- **Previous**: [01-week1-foundation-events.md](01-week1-foundation-events.md) - GOgent-001 to 009
- **Next**: [03-week1-validation-cli.md](03-week1-validation-cli.md) - GOgent-020 to 025
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure

---

## Summary

This file covers three critical areas:

1. **Escape Hatches (Day 3)**: Override flags, XDG path compliance, violation logging
2. **Complexity Routing (Day 4)**: Scout metrics, freshness checks, tier updates
3. **Tool Permissions (Day 5)**: Permission checks, wildcard handling, comprehensive tests

**Critical Dependencies**:
- GOgent-007 (ParseToolEvent) must be complete for override parsing
- GOgent-010 (XDG paths) used by all subsequent tickets
- GOgent-013 (scout metrics) enables complexity-based routing

---

## Day 3: Escape Hatches (GOgent-010 to 012)

### GOgent-010: Implement Override Flags and XDG Paths

**Time**: 1.5 hours
**Dependencies**: GOgent-007

**Task**:
Parse `--force-tier=X` and `--force-delegation=Y` flags from Task prompts to allow override of routing rules.

**File**: `pkg/routing/overrides.go`

**Imports**:
```go
package routing

import (
	"regexp"
	"strings"
)
```

**Implementation**:
```go
// OverrideFlags represents parsed override flags from prompt
type OverrideFlags struct {
	ForceTier       string // e.g., "haiku", "sonnet", "opus"
	ForceDelegation string // e.g., "haiku", "sonnet"
}

// ParseOverrides extracts --force-* flags from Task prompt
func ParseOverrides(prompt string) *OverrideFlags {
	flags := &OverrideFlags{}

	// Match --force-tier=VALUE
	tierRe := regexp.MustCompile(`--force-tier=(\w+)`)
	if match := tierRe.FindStringSubmatch(prompt); len(match) > 1 {
		flags.ForceTier = match[1]
	}

	// Match --force-delegation=VALUE
	delegationRe := regexp.MustCompile(`--force-delegation=(\w+)`)
	if match := delegationRe.FindStringSubmatch(prompt); len(match) > 1 {
		flags.ForceDelegation = match[1]
	}

	return flags
}

// HasOverrides returns true if any overrides are present
func (o *OverrideFlags) HasOverrides() bool {
	return o.ForceTier != "" || o.ForceDelegation != ""
}
```

**File**: `pkg/config/paths.go` (XDG compliance - fixes M-2)

**Path Resolution**:
```go
package config

import (
	"os"
	"path/filepath"
)

// GetGOgentDir returns XDG-compliant gogent directory
// Priority: XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache
func GetGOgentDir() string {
	// Try XDG_RUNTIME_DIR (systemd standard)
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		os.MkdirAll(dir, 0755)
		return dir
	}

	// Try XDG_CACHE_HOME
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		dir := filepath.Join(xdg, "gogent")
		os.MkdirAll(dir, 0755)
		return dir
	}

	// Fallback: ~/.cache/gogent
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".cache", "gogent")
	os.MkdirAll(dir, 0755)
	return dir
}

// GetTierFilePath returns path to current-tier file
func GetTierFilePath() string {
	return filepath.Join(GetGOgentDir(), "current-tier")
}

// GetMaxDelegationPath returns path to max_delegation file
func GetMaxDelegationPath() string {
	return filepath.Join(GetGOgentDir(), "max_delegation")
}

// GetViolationsLogPath returns path to routing violations log
func GetViolationsLogPath() string {
	return filepath.Join(GetGOgentDir(), "routing-violations.jsonl")
}
```

**Tests**: `pkg/routing/overrides_test.go`

```go
package routing

import (
	"testing"
)

func TestParseOverrides_ForceTier(t *testing.T) {
	prompt := "--force-tier=opus\n\nAGENT: einstein\n\nAnalyze this problem"
	flags := ParseOverrides(prompt)

	if flags.ForceTier != "opus" {
		t.Errorf("Expected force-tier opus, got: %s", flags.ForceTier)
	}
}

func TestParseOverrides_ForceDelegation(t *testing.T) {
	prompt := "--force-delegation=sonnet\n\nTask requires reasoning"
	flags := ParseOverrides(prompt)

	if flags.ForceDelegation != "sonnet" {
		t.Errorf("Expected force-delegation sonnet, got: %s", flags.ForceDelegation)
	}
}

func TestParseOverrides_Both(t *testing.T) {
	prompt := "--force-tier=haiku --force-delegation=sonnet\n\nSpecial case"
	flags := ParseOverrides(prompt)

	if flags.ForceTier != "haiku" || flags.ForceDelegation != "sonnet" {
		t.Errorf("Expected both flags, got: tier=%s delegation=%s",
			flags.ForceTier, flags.ForceDelegation)
	}
}

func TestParseOverrides_None(t *testing.T) {
	prompt := "AGENT: python-pro\n\nImplement function"
	flags := ParseOverrides(prompt)

	if flags.HasOverrides() {
		t.Error("Expected no overrides")
	}
}
```

**Acceptance Criteria**:
- [ ] `ParseOverrides()` extracts force-tier flag correctly
- [ ] `ParseOverrides()` extracts force-delegation flag correctly
- [ ] Regex handles flags anywhere in prompt
- [ ] `GetGOgentDir()` uses XDG_RUNTIME_DIR if available
- [ ] Falls back to XDG_CACHE_HOME, then ~/.cache/gogent
- [ ] All tests pass: `go test ./pkg/routing ./pkg/config`

**Why This Matters**: Override flags are critical escape hatches. Must parse reliably. XDG compliance fixes M-2 (hardcoded /tmp paths).

---

### GOgent-011: Implement Violation Logging to JSONL

**Time**: 1.5 hours
**Dependencies**: GOgent-010

**Task**:
Log routing violations to JSONL file for audit trail and debugging.

**File**: `pkg/routing/violations.go`

**Imports**:
```go
package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent/pkg/config"
)
```

**Implementation**:
```go
// Violation represents a routing rule violation
type Violation struct {
	Timestamp     string `json:"timestamp"`
	SessionID     string `json:"session_id"`
	ViolationType string `json:"violation_type"`
	Agent         string `json:"agent,omitempty"`
	Model         string `json:"model,omitempty"`
	Tool          string `json:"tool,omitempty"`
	Reason        string `json:"reason"`
	Allowed       string `json:"allowed,omitempty"`
	Override      string `json:"override,omitempty"`
}

// LogViolation appends violation to JSONL log file
func LogViolation(v *Violation) error {
	v.Timestamp = time.Now().Format(time.RFC3339)

	// Open log file (append mode)
	logPath := config.GetViolationsLogPath()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[violations] Failed to open log: %w", err)
	}
	defer f.Close()

	// Write JSONL entry
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("[violations] Failed to marshal violation: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("[violations] Failed to write log: %w", err)
	}

	return nil
}
```

**Tests**: `pkg/routing/violations_test.go`

```go
package routing

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/yourusername/gogent/pkg/config"
)

func TestLogViolation(t *testing.T) {
	// Create temp log file
	tmpLog := "/tmp/test-violations.jsonl"
	defer os.Remove(tmpLog)

	// Override log path for testing
	oldPath := config.GetViolationsLogPath()
	config.SetViolationsLogPathForTest(tmpLog)
	defer config.SetViolationsLogPathForTest(oldPath)

	// Log violation
	v := &Violation{
		SessionID:     "test-123",
		ViolationType: "tool_permission",
		Tool:          "Write",
		Reason:        "Tier haiku cannot use Write",
		Allowed:       "Read, Glob, Grep",
	}

	if err := LogViolation(v); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Read log file
	data, err := os.ReadFile(tmpLog)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	// Parse JSONL
	var logged Violation
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to parse logged violation: %v", err)
	}

	if logged.SessionID != "test-123" {
		t.Errorf("Expected session_id test-123, got: %s", logged.SessionID)
	}

	if logged.Timestamp == "" {
		t.Error("Expected timestamp to be populated")
	}
}
```

**Acceptance Criteria**:
- [ ] `LogViolation()` writes JSONL to violations log
- [ ] Log file created if doesn't exist
- [ ] Each violation appended as new line
- [ ] Timestamp auto-populated in RFC3339 format
- [ ] Tests verify JSONL format and content
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Audit trail for debugging routing issues. JSONL format allows jq analysis.

---

### GOgent-011a: Enhance Violation Schema and Add Dual-Write

**Time**: 1 hour
**Dependencies**: GOgent-011
**Created**: Post-orchestrator architectural analysis (2026-01-17)

**Task**:
Enhance the `Violation` struct with additional context fields required for weekly review agent analysis and root cause pattern detection. Implement dual-write pattern to log violations to both XDG cache (global audit trail) and `.claude/memory/` (project-scoped, session-integrated).

**Background**:
The orchestrator agent identified critical gaps in the GOgent-011 schema that prevent effective violation pattern analysis:
- Missing file context → can't correlate violations to code locations
- Missing tier context → can't analyze tier-mismatch patterns
- Missing task context → can't understand user intent
- Missing enforcement outcome → can't assess hook effectiveness
- Single write location → violations not integrated with session archive/handoff

**Files to Modify**:
- `pkg/routing/violations.go` - Add fields, implement dual-write
- `pkg/routing/violations_test.go` - Test new fields and dual-write
- `pkg/config/paths.go` - Add `GetProjectViolationsLogPath()` helper

**Enhanced Schema**:
```go
// Violation represents a routing rule violation.
// Logged to both XDG cache (global) and .claude/memory/ (project-scoped).
type Violation struct {
	// Existing fields from GOgent-011
	Timestamp     string `json:"timestamp"`
	SessionID     string `json:"session_id"`
	ViolationType string `json:"violation_type"`
	Agent         string `json:"agent,omitempty"`
	Model         string `json:"model,omitempty"`
	Tool          string `json:"tool,omitempty"`
	Reason        string `json:"reason"`
	Allowed       string `json:"allowed,omitempty"`
	Override      string `json:"override,omitempty"`

	// NEW: File context (critical for correlation with sharp edges)
	File          string `json:"file,omitempty"`

	// NEW: Tier context (critical for pattern analysis)
	CurrentTier   string `json:"current_tier,omitempty"`
	RequiredTier  string `json:"required_tier,omitempty"`

	// NEW: Task context (critical for understanding user intent)
	TaskDescription string `json:"task_description,omitempty"` // First 200 chars of prompt

	// NEW: Enforcement outcome (critical for effectiveness analysis)
	HookDecision  string `json:"hook_decision,omitempty"` // "allow", "warn", "block"

	// NEW: Project context (enables cross-project pattern detection)
	ProjectDir    string `json:"project_dir,omitempty"`
}
```

**Dual-Write Implementation**:
```go
// LogViolation appends violation to BOTH:
// 1. Global XDG cache: ~/.cache/gogent/routing-violations.jsonl (survives project deletion)
// 2. Project memory: <project>/.claude/memory/routing-violations.jsonl (session integration)
func LogViolation(v *Violation, projectDir string) error {
	// Auto-populate timestamp
	v.Timestamp = time.Now().Format(time.RFC3339)

	// Populate project directory if provided
	if projectDir != "" {
		v.ProjectDir = projectDir
	}

	// Marshal once, write twice
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("[violations] Failed to marshal violation: %w", err)
	}
	data = append(data, '\n') // JSONL format

	// WRITE 1: Global XDG cache (primary, required)
	globalPath := config.GetViolationsLogPath()
	if err := appendToFile(globalPath, data); err != nil {
		return fmt.Errorf("[violations] Failed to write global log: %w", err)
	}

	// WRITE 2: Project memory (secondary, optional)
	if projectDir != "" {
		projectPath := config.GetProjectViolationsLogPath(projectDir)
		if err := appendToFile(projectPath, data); err != nil {
			// Log warning but don't fail - global write succeeded
			fmt.Fprintf(os.Stderr, "[violations] Warning: Failed project log: %v\n", err)
		}
	}

	return nil
}

// Helper: append data to file (create if not exists)
func appendToFile(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	// Open/create file in append mode
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}
```

**Config Path Helper** (`pkg/config/paths.go`):
```go
// GetProjectViolationsLogPath returns project-scoped violation log path.
// Used for dual-write pattern - integrates with session archive.
func GetProjectViolationsLogPath(projectDir string) string {
	return filepath.Join(projectDir, ".claude", "memory", "routing-violations.jsonl")
}
```

**Tests** (`pkg/routing/violations_test.go`):
```go
// Test new fields are marshaled correctly
func TestViolation_EnhancedFields(t *testing.T) {
	v := &Violation{
		SessionID:       "test-123",
		ViolationType:   "tier_mismatch",
		Agent:           "tech-docs-writer",
		File:            "docs/system-guide.md",
		CurrentTier:     "haiku",
		RequiredTier:    "haiku_thinking",
		TaskDescription: "Update system guide with new routing rules",
		HookDecision:    "block",
		ProjectDir:      "/home/user/my-project",
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify all new fields present in JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	expectedFields := []string{
		"file", "current_tier", "required_tier",
		"task_description", "hook_decision", "project_dir",
	}
	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("Missing field: %s", field)
		}
	}
}

// Test dual-write pattern
func TestLogViolation_DualWrite(t *testing.T) {
	// Create temp directories for both logs
	tmpGlobal := t.TempDir()
	tmpProject := t.TempDir()

	// Mock config paths
	oldGetViolationsLogPath := config.GetViolationsLogPath
	config.GetViolationsLogPath = func() string {
		return filepath.Join(tmpGlobal, "routing-violations.jsonl")
	}
	defer func() { config.GetViolationsLogPath = oldGetViolationsLogPath }()

	// Log violation with project directory
	v := &Violation{
		SessionID:     "test-dual",
		ViolationType: "tool_permission",
		Tool:          "Write",
	}

	if err := LogViolation(v, tmpProject); err != nil {
		t.Fatalf("LogViolation failed: %v", err)
	}

	// Verify global log exists
	globalPath := filepath.Join(tmpGlobal, "routing-violations.jsonl")
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Global log not created")
	}

	// Verify project log exists
	projectPath := filepath.Join(tmpProject, ".claude", "memory", "routing-violations.jsonl")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		t.Error("Project log not created")
	}

	// Verify both logs contain same entry
	globalData, _ := os.ReadFile(globalPath)
	projectData, _ := os.ReadFile(projectPath)

	if string(globalData) != string(projectData) {
		t.Error("Global and project logs diverged")
	}
}

// Test dual-write handles project log failure gracefully
func TestLogViolation_ProjectLogFailureGraceful(t *testing.T) {
	tmpGlobal := t.TempDir()

	// Mock config paths
	oldGetViolationsLogPath := config.GetViolationsLogPath
	config.GetViolationsLogPath = func() string {
		return filepath.Join(tmpGlobal, "routing-violations.jsonl")
	}
	defer func() { config.GetViolationsLogPath = oldGetViolationsLogPath }()

	// Use invalid project directory (write will fail)
	invalidProjectDir := "/dev/null/invalid"

	v := &Violation{
		SessionID:     "test-graceful",
		ViolationType: "test",
	}

	// Should NOT return error even if project log fails
	if err := LogViolation(v, invalidProjectDir); err != nil {
		t.Errorf("LogViolation should not fail when project log fails: %v", err)
	}

	// Global log should still be written
	globalPath := filepath.Join(tmpGlobal, "routing-violations.jsonl")
	if _, err := os.Stat(globalPath); os.IsNotExist(err) {
		t.Error("Global log should be created even if project log fails")
	}
}
```

**Acceptance Criteria**:
- [ ] Enhanced `Violation` struct includes all 6 new fields (file, current_tier, required_tier, task_description, hook_decision, project_dir)
- [ ] All new fields marked `omitempty` for backward compatibility
- [ ] `LogViolation()` signature updated to accept `projectDir string`
- [ ] Dual-write implemented: writes to both XDG cache and project memory
- [ ] Project log failure does NOT fail the entire operation (graceful degradation)
- [ ] `GetProjectViolationsLogPath()` helper added to `pkg/config/paths.go`
- [ ] All tests pass: `go test ./pkg/routing ./pkg/config`

**Why This Matters**:
- **Weekly review agent** can analyze violation patterns by file, tier, and task type
- **Session handoff** includes violations from `.claude/memory/` for context continuity
- **Cross-project analysis** enabled via global log with project_dir field
- **Root cause diagnosis** possible with file + tier + task context
- **Hook effectiveness** measurable via hook_decision tracking
- **Graceful degradation** ensures global audit trail survives even if project write fails

**Integration Points**:
This enhancement enables future tickets:
- GOgent-026-033 (Session archive): Read violations from `.claude/memory/`
- GOgent-037 (Sharp edge capture): Correlate violations with sharp edges via `file` field
- Weekly review agent (future): Detect patterns like "3 violations on same file → sharp edge candidate"

---

### GOgent-012: Escape Hatch Integration Tests

**Time**: 1 hour
**Dependencies**: GOgent-011

**Task**:
Test override flags and violation logging end-to-end.

**File**: `test/integration/overrides_test.go`

**Implementation**:
```go
package integration

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/gogent/pkg/routing"
	"github.com/yourusername/gogent/pkg/config"
)

func TestOverrideWorkflow(t *testing.T) {
	// Create temp violations log
	tmpLog := "/tmp/test-overrides.jsonl"
	defer os.Remove(tmpLog)
	config.SetViolationsLogPathForTest(tmpLog)

	// Parse event with override
	eventJSON := `{
		"tool_name": "Task",
		"tool_input": {
			"model": "sonnet",
			"prompt": "--force-delegation=sonnet\n\nAGENT: architect\n\nCreate plan"
		},
		"session_id": "test-override",
		"hook_event_name": "PreToolUse"
	}`

	reader := strings.NewReader(eventJSON)
	event, err := routing.ParseToolEvent(reader, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse Task input
	taskInput, err := routing.ParseTaskInput(event.ToolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Parse overrides
	overrides := routing.ParseOverrides(taskInput.Prompt)
	if overrides.ForceDelegation != "sonnet" {
		t.Errorf("Expected force-delegation sonnet, got: %s", overrides.ForceDelegation)
	}

	// Log a violation (simulated ceiling check)
	violation := &routing.Violation{
		SessionID:     event.SessionID,
		ViolationType: "delegation_ceiling",
		Agent:         "architect",
		Model:         "sonnet",
		Reason:        "Ceiling is haiku, agent requires sonnet",
		Override:      "force-delegation=sonnet",
	}

	if err := routing.LogViolation(violation); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Verify log
	data, err := os.ReadFile(tmpLog)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var logged routing.Violation
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	if logged.Override != "force-delegation=sonnet" {
		t.Errorf("Expected override logged, got: %s", logged.Override)
	}

	t.Logf("✓ Override workflow complete: parsed, logged, verified")
}
```

**Acceptance Criteria**:
- [ ] Test parses event with override flag
- [ ] Test logs violation with override info
- [ ] Test verifies JSONL contains override field
- [ ] `go test ./test/integration` passes
- [ ] Test demonstrates end-to-end override workflow

**Why This Matters**: Escape hatches are critical for unblocking users. Must work reliably.

---

## Day 4: Complexity Routing (GOgent-013 to 016)

### GOgent-013: Implement Scout Metrics Loading

**Time**: 2 hours
**Dependencies**: GOgent-010

**Task**:
Load scout metrics from JSON file and validate structure. Scout metrics contain file counts, LoC, complexity signals written by `haiku-scout` agent.

**File**: `pkg/routing/metrics.go`

**Imports**:
```go
package routing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)
```

**Implementation**:
```go
// ScoutMetrics represents output from haiku-scout agent
type ScoutMetrics struct {
	FileCount       int      `json:"file_count"`
	TotalLines      int      `json:"total_lines"`
	ComplexityScore float64  `json:"complexity_score"`
	RecommendedTier string   `json:"recommended_tier"`
	Timestamp       int64    `json:"timestamp"`
	ScannedPaths    []string `json:"scanned_paths,omitempty"`
}

// LoadScoutMetrics reads scout_metrics.json from project tmp directory
func LoadScoutMetrics(projectDir string) (*ScoutMetrics, error) {
	metricsPath := filepath.Join(projectDir, ".claude", "tmp", "scout_metrics.json")

	data, err := os.ReadFile(metricsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No metrics file is OK - scout hasn't run
		}
		return nil, fmt.Errorf("[metrics] Failed to read scout metrics at %s: %w. Check file permissions.", metricsPath, err)
	}

	var metrics ScoutMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("[metrics] Failed to parse scout metrics: %w. Check JSON format in %s", err, metricsPath)
	}

	// Validate required fields
	if metrics.ComplexityScore < 0 {
		return nil, fmt.Errorf("[metrics] Invalid complexity_score: %f. Must be >= 0.", metrics.ComplexityScore)
	}

	if metrics.RecommendedTier == "" {
		return nil, fmt.Errorf("[metrics] Missing required field: recommended_tier. Scout must specify tier.")
	}

	return &metrics, nil
}

// IsFresh returns true if metrics are less than ttlSeconds old
func (m *ScoutMetrics) IsFresh(ttlSeconds int) bool {
	if m == nil {
		return false
	}
	age := time.Now().Unix() - m.Timestamp
	return age < int64(ttlSeconds)
}

// Age returns metrics age in seconds
func (m *ScoutMetrics) Age() int64 {
	if m == nil {
		return -1
	}
	return time.Now().Unix() - m.Timestamp
}
```

**Tests**: `pkg/routing/metrics_test.go`

```go
package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadScoutMetrics_Valid(t *testing.T) {
	// Create temp project dir
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write valid metrics
	metrics := ScoutMetrics{
		FileCount:       42,
		TotalLines:      3500,
		ComplexityScore: 38.5,
		RecommendedTier: "sonnet",
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Load metrics
	loaded, err := LoadScoutMetrics(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load metrics: %v", err)
	}

	if loaded.FileCount != 42 {
		t.Errorf("Expected file_count 42, got: %d", loaded.FileCount)
	}

	if loaded.RecommendedTier != "sonnet" {
		t.Errorf("Expected recommended_tier sonnet, got: %s", loaded.RecommendedTier)
	}
}

func TestLoadScoutMetrics_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Load when file doesn't exist (should return nil, not error)
	metrics, err := LoadScoutMetrics(tmpDir)
	if err != nil {
		t.Errorf("Expected no error when file missing, got: %v", err)
	}

	if metrics != nil {
		t.Error("Expected nil metrics when file missing")
	}
}

func TestLoadScoutMetrics_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	metricsDir := filepath.Join(tmpDir, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Write invalid JSON
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, []byte("{invalid json}"), 0644)

	// Load should fail
	_, err := LoadScoutMetrics(tmpDir)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestIsFresh(t *testing.T) {
	now := time.Now().Unix()

	tests := []struct {
		name       string
		timestamp  int64
		ttlSeconds int
		expected   bool
	}{
		{"Fresh (1 min old, 5 min TTL)", now - 60, 300, true},
		{"Stale (6 min old, 5 min TTL)", now - 360, 300, false},
		{"Exactly at TTL", now - 300, 300, false},
		{"Zero TTL", now, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := &ScoutMetrics{Timestamp: tt.timestamp}
			if fresh := metrics.IsFresh(tt.ttlSeconds); fresh != tt.expected {
				t.Errorf("IsFresh() = %v, expected %v", fresh, tt.expected)
			}
		})
	}
}

func TestIsFresh_Nil(t *testing.T) {
	var metrics *ScoutMetrics
	if metrics.IsFresh(300) {
		t.Error("Nil metrics should not be fresh")
	}
}
```

**Acceptance Criteria**:
- [ ] `LoadScoutMetrics()` reads and parses scout_metrics.json
- [ ] Returns nil (no error) when file doesn't exist
- [ ] Validates required fields (complexity_score, recommended_tier)
- [ ] `IsFresh()` correctly checks TTL against timestamp
- [ ] Tests cover valid, missing, and invalid JSON cases
- [ ] `go test ./pkg/routing` passes with ≥80% coverage

**Why This Matters**: Scout metrics enable complexity-based routing. Freshness check prevents stale routing decisions.

---

### GOgent-014: Implement Metrics Freshness Check

**Time**: 1.5 hours
**Dependencies**: GOgent-013

**Task**:
Add TTL-based validation for scout metrics. If metrics are stale (>5 minutes), fall back to default tier.

**File**: `pkg/routing/metrics.go` (extend existing)

**Add to existing file**:
```go
// MetricsConfig defines TTL and fallback behavior
type MetricsConfig struct {
	TTLSeconds  int    // Default: 300 (5 minutes)
	FallbackTier string // Default: "sonnet"
}

// DefaultMetricsConfig returns standard config
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}
}

// GetActiveTier returns tier based on metrics freshness
// If fresh: returns recommended_tier from metrics
// If stale: returns fallback tier
func (m *ScoutMetrics) GetActiveTier(config *MetricsConfig) string {
	if m == nil {
		return config.FallbackTier
	}

	if m.IsFresh(config.TTLSeconds) {
		return m.RecommendedTier
	}

	return config.FallbackTier
}
```

**Tests**: `pkg/routing/metrics_test.go` (extend existing)

**Add to existing test file**:
```go
func TestGetActiveTier_Fresh(t *testing.T) {
	now := time.Now().Unix()
	metrics := &ScoutMetrics{
		RecommendedTier: "haiku",
		Timestamp:       now - 60, // 1 minute old
	}

	config := &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}

	tier := metrics.GetActiveTier(config)
	if tier != "haiku" {
		t.Errorf("Expected fresh tier 'haiku', got: %s", tier)
	}
}

func TestGetActiveTier_Stale(t *testing.T) {
	now := time.Now().Unix()
	metrics := &ScoutMetrics{
		RecommendedTier: "haiku",
		Timestamp:       now - 400, // 6.7 minutes old
	}

	config := &MetricsConfig{
		TTLSeconds:   300, // 5 minutes
		FallbackTier: "sonnet",
	}

	tier := metrics.GetActiveTier(config)
	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet', got: %s", tier)
	}
}

func TestGetActiveTier_Nil(t *testing.T) {
	var metrics *ScoutMetrics
	config := DefaultMetricsConfig()

	tier := metrics.GetActiveTier(config)
	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet' for nil metrics, got: %s", tier)
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()

	if config.TTLSeconds != 300 {
		t.Errorf("Expected default TTL 300s, got: %d", config.TTLSeconds)
	}

	if config.FallbackTier != "sonnet" {
		t.Errorf("Expected default fallback 'sonnet', got: %s", config.FallbackTier)
	}
}
```

**Acceptance Criteria**:
- [ ] `GetActiveTier()` returns recommended tier when fresh
- [ ] Returns fallback tier when stale (age > TTL)
- [ ] Returns fallback tier when metrics is nil
- [ ] Default config has 300s TTL and "sonnet" fallback
- [ ] Tests cover fresh, stale, and nil cases
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Stale metrics can cause incorrect routing. TTL ensures routing decisions use current complexity.

---

### GOgent-015: Implement Tier Update from Complexity

**Time**: 2 hours
**Dependencies**: GOgent-014

**Task**:
Read scout metrics, determine active tier, and update current-tier file.

**File**: `pkg/routing/tier_update.go`

**Imports**:
```go
package routing

import (
	"fmt"
	"os"

	"github.com/yourusername/gogent/pkg/config"
)

```

**Implementation**:
```go
// UpdateTierFromMetrics reads scout metrics and updates current-tier file
func UpdateTierFromMetrics(projectDir string) error {
	// Load scout metrics
	metrics, err := LoadScoutMetrics(projectDir)
	if err != nil {
		return fmt.Errorf("[tier-update] Failed to load metrics: %w", err)
	}

	// If no metrics, nothing to update
	if metrics == nil {
		return nil
	}

	// Get active tier based on freshness
	config := DefaultMetricsConfig()
	activeTier := metrics.GetActiveTier(config)

	// Write to current-tier file
	tierPath := config.GetTierFilePath()
	if err := os.WriteFile(tierPath, []byte(activeTier), 0644); err != nil {
		return fmt.Errorf("[tier-update] Failed to write tier file at %s: %w. Check permissions.", tierPath, err)
	}

	return nil
}

// GetCurrentTier reads current-tier file, returns default if missing
func GetCurrentTier() (string, error) {
	tierPath := config.GetTierFilePath()

	data, err := os.ReadFile(tierPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "sonnet", nil // Default tier
		}
		return "", fmt.Errorf("[tier-read] Failed to read tier file at %s: %w", tierPath, err)
	}

	tier := string(data)
	if tier == "" {
		return "sonnet", nil
	}

	return tier, nil
}
```

**Tests**: `pkg/routing/tier_update_test.go`

```go
package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/gogent/pkg/config"
)

func TestUpdateTierFromMetrics_Fresh(t *testing.T) {
	// Setup temp directories
	tmpProject := t.TempDir()
	tmpGOgent := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	// Override gogent dir for testing
	config.SetGOgentDirForTest(tmpGOgent)
	defer config.ResetGOgentDir()

	// Write fresh metrics recommending "haiku"
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: 12.5,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix(),
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Update tier
	if err := UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("UpdateTierFromMetrics failed: %v", err)
	}

	// Verify tier file
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("GetCurrentTier failed: %v", err)
	}

	if tier != "haiku" {
		t.Errorf("Expected tier 'haiku', got: %s", tier)
	}
}

func TestUpdateTierFromMetrics_Stale(t *testing.T) {
	tmpProject := t.TempDir()
	tmpGOgent := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	config.SetGOgentDirForTest(tmpGOgent)
	defer config.ResetGOgentDir()

	// Write stale metrics (7 minutes old)
	metrics := ScoutMetrics{
		FileCount:       10,
		TotalLines:      500,
		ComplexityScore: 12.5,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix() - 420, // 7 minutes
	}

	data, _ := json.Marshal(metrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, data, 0644)

	// Update tier
	if err := UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("UpdateTierFromMetrics failed: %v", err)
	}

	// Should fall back to "sonnet"
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("GetCurrentTier failed: %v", err)
	}

	if tier != "sonnet" {
		t.Errorf("Expected fallback tier 'sonnet', got: %s", tier)
	}
}

func TestGetCurrentTier_NoFile(t *testing.T) {
	tmpGOgent := t.TempDir()
	config.SetGOgentDirForTest(tmpGOgent)
	defer config.ResetGOgentDir()

	// No tier file exists
	tier, err := GetCurrentTier()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tier != "sonnet" {
		t.Errorf("Expected default tier 'sonnet', got: %s", tier)
	}
}
```

**Acceptance Criteria**:
- [ ] `UpdateTierFromMetrics()` reads scout metrics
- [ ] Writes recommended tier to current-tier file when fresh
- [ ] Writes fallback tier when metrics stale
- [ ] `GetCurrentTier()` reads tier file, defaults to "sonnet"
- [ ] Tests verify fresh, stale, and missing file cases
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Tier updates enable dynamic routing. Current tier used by validation hook to enforce permissions.

---

### GOgent-016: Complexity Routing Tests

**Time**: 1.5 hours
**Dependencies**: GOgent-015

**Task**:
Integration tests for metrics → complexity → tier workflow.

**File**: `test/integration/complexity_routing_test.go`

**Implementation**:
```go
package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yourusername/gogent/pkg/routing"
	"github.com/yourusername/gogent/pkg/config"
)

func TestComplexityRoutingWorkflow(t *testing.T) {
	// Setup temp directories
	tmpProject := t.TempDir()
	tmpGOgent := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	config.SetGOgentDirForTest(tmpGOgent)
	defer config.ResetGOgentDir()

	// Step 1: Scout writes metrics (simulated)
	scoutMetrics := routing.ScoutMetrics{
		FileCount:       85,
		TotalLines:      7200,
		ComplexityScore: 42.8,
		RecommendedTier: "sonnet",
		Timestamp:       time.Now().Unix(),
		ScannedPaths:    []string{"src/", "pkg/"},
	}

	metricsData, _ := json.Marshal(scoutMetrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, metricsData, 0644)

	// Step 2: Load and verify metrics
	loaded, err := routing.LoadScoutMetrics(tmpProject)
	if err != nil {
		t.Fatalf("Failed to load scout metrics: %v", err)
	}

	if !loaded.IsFresh(300) {
		t.Error("Metrics should be fresh")
	}

	// Step 3: Update tier from metrics
	if err := routing.UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("Failed to update tier: %v", err)
	}

	// Step 4: Verify tier updated correctly
	currentTier, err := routing.GetCurrentTier()
	if err != nil {
		t.Fatalf("Failed to get current tier: %v", err)
	}

	if currentTier != "sonnet" {
		t.Errorf("Expected tier 'sonnet', got: %s", currentTier)
	}

	t.Logf("✓ Complexity routing workflow complete: scout → metrics → tier")
}

func TestComplexityRoutingStaleMetrics(t *testing.T) {
	tmpProject := t.TempDir()
	tmpGOgent := t.TempDir()

	metricsDir := filepath.Join(tmpProject, ".claude", "tmp")
	os.MkdirAll(metricsDir, 0755)

	config.SetGOgentDirForTest(tmpGOgent)
	defer config.ResetGOgentDir()

	// Write stale metrics (10 minutes old)
	staleMetrics := routing.ScoutMetrics{
		FileCount:       5,
		TotalLines:      200,
		ComplexityScore: 8.2,
		RecommendedTier: "haiku",
		Timestamp:       time.Now().Unix() - 600, // 10 minutes
	}

	metricsData, _ := json.Marshal(staleMetrics)
	metricsPath := filepath.Join(metricsDir, "scout_metrics.json")
	os.WriteFile(metricsPath, metricsData, 0644)

	// Update tier
	if err := routing.UpdateTierFromMetrics(tmpProject); err != nil {
		t.Fatalf("Failed to update tier: %v", err)
	}

	// Should fall back to "sonnet"
	currentTier, err := routing.GetCurrentTier()
	if err != nil {
		t.Fatalf("Failed to get current tier: %v", err)
	}

	if currentTier != "sonnet" {
		t.Errorf("Expected fallback to 'sonnet' for stale metrics, got: %s", currentTier)
	}

	t.Logf("✓ Stale metrics correctly fall back to default tier")
}
```

**Acceptance Criteria**:
- [ ] Test simulates scout writing metrics
- [ ] Test loads metrics and verifies freshness
- [ ] Test updates tier from fresh metrics
- [ ] Test verifies tier file contains recommended tier
- [ ] Stale metrics test verifies fallback behavior
- [ ] `go test ./test/integration` passes

**Why This Matters**: End-to-end test ensures scout metrics actually drive routing decisions.

---

## Day 5: Tool Permissions (GOgent-017 to 019)

### GOgent-017: Implement Tool Permission Checks

**Time**: 2.5 hours
**Dependencies**: GOgent-015

**Task**:
Check if current tier allows requested tool. Compare tool name against schema's tier.tools field.

**File**: `pkg/routing/permissions.go`

**Imports**:
```go
package routing

import (
	"fmt"
	"strings"
)
```

**Implementation**:
```go
// ToolPermission represents result of permission check
type ToolPermission struct {
	Allowed         bool
	CurrentTier     string
	Tool            string
	AllowedTools    []string
	RecommendedTier string
}

// CheckToolPermission validates if tool is allowed for tier
func CheckToolPermission(schema *Schema, currentTier string, toolName string) *ToolPermission {
	result := &ToolPermission{
		CurrentTier: currentTier,
		Tool:        toolName,
	}

	// Get tier config
	tierConfig, exists := schema.Tiers[currentTier]
	if !exists {
		result.Allowed = false
		return result
	}

	// Check if tools is wildcard "*"
	if str, ok := tierConfig.Tools.(string); ok && str == "*" {
		result.Allowed = true
		result.AllowedTools = []string{"*"}
		return result
	}

	// Tools should be array of strings
	toolsArray, ok := tierConfig.Tools.([]interface{})
	if !ok {
		result.Allowed = false
		return result
	}

	// Convert to string array
	allowedTools := make([]string, 0, len(toolsArray))
	for _, t := range toolsArray {
		if toolStr, ok := t.(string); ok {
			allowedTools = append(allowedTools, toolStr)
		}
	}

	result.AllowedTools = allowedTools

	// Check if tool is in allowed list
	for _, allowed := range allowedTools {
		if allowed == toolName {
			result.Allowed = true
			return result
		}
	}

	// Tool not allowed - find which tier does allow it
	result.Allowed = false
	result.RecommendedTier = findTierForTool(schema, toolName)

	return result
}

// findTierForTool searches schema for lowest tier allowing tool
func findTierForTool(schema *Schema, toolName string) string {
	// Check tiers in order: haiku, haiku_thinking, sonnet, opus
	tierOrder := []string{"haiku", "haiku_thinking", "sonnet", "opus"}

	for _, tier := range tierOrder {
		tierConfig, exists := schema.Tiers[tier]
		if !exists {
			continue
		}

		// Check wildcard
		if str, ok := tierConfig.Tools.(string); ok && str == "*" {
			return tier
		}

		// Check array
		if toolsArray, ok := tierConfig.Tools.([]interface{}); ok {
			for _, t := range toolsArray {
				if toolStr, ok := t.(string); ok && toolStr == toolName {
					return tier
				}
			}
		}
	}

	return "unknown"
}

// FormatPermissionError creates formatted error message
func (p *ToolPermission) FormatPermissionError() string {
	allowedStr := strings.Join(p.AllowedTools, ", ")

	return fmt.Sprintf(
		"[routing] Tool '%s' not permitted at tier '%s'. Allowed tools for %s: %s. Tool '%s' requires tier: %s. Use --force-tier=%s to override.",
		p.Tool,
		p.CurrentTier,
		p.CurrentTier,
		allowedStr,
		p.Tool,
		p.RecommendedTier,
		p.RecommendedTier,
	)
}
```

**Tests**: `pkg/routing/permissions_test.go`

```go
package routing

import (
	"testing"
)

func TestCheckToolPermission_Allowed(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []interface{}{"Read", "Glob", "Grep"},
			},
		},
	}

	result := CheckToolPermission(schema, "haiku", "Read")

	if !result.Allowed {
		t.Error("Expected Read to be allowed for haiku tier")
	}
}

func TestCheckToolPermission_Denied(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {
				Tools: []interface{}{"Read", "Glob", "Grep"},
			},
			"sonnet": {
				Tools: []interface{}{"Read", "Write", "Edit", "Bash"},
			},
		},
	}

	result := CheckToolPermission(schema, "haiku", "Write")

	if result.Allowed {
		t.Error("Expected Write to be denied for haiku tier")
	}

	if result.RecommendedTier != "sonnet" {
		t.Errorf("Expected recommended tier 'sonnet', got: %s", result.RecommendedTier)
	}
}

func TestCheckToolPermission_Wildcard(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {
				Tools: "*",
			},
		},
	}

	result := CheckToolPermission(schema, "opus", "AnyTool")

	if !result.Allowed {
		t.Error("Expected wildcard to allow any tool")
	}

	if len(result.AllowedTools) != 1 || result.AllowedTools[0] != "*" {
		t.Errorf("Expected AllowedTools to be ['*'], got: %v", result.AllowedTools)
	}
}

func TestFormatPermissionError(t *testing.T) {
	result := &ToolPermission{
		Allowed:         false,
		CurrentTier:     "haiku",
		Tool:            "Write",
		AllowedTools:    []string{"Read", "Glob", "Grep"},
		RecommendedTier: "sonnet",
	}

	errMsg := result.FormatPermissionError()

	// Check error contains key information
	if !contains(errMsg, "Write") {
		t.Error("Error should mention tool name")
	}

	if !contains(errMsg, "haiku") {
		t.Error("Error should mention current tier")
	}

	if !contains(errMsg, "sonnet") {
		t.Error("Error should mention recommended tier")
	}

	if !contains(errMsg, "--force-tier=sonnet") {
		t.Error("Error should suggest override")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Acceptance Criteria**:
- [ ] `CheckToolPermission()` allows tool when in tier's tools array
- [ ] Denies tool when not in array
- [ ] Handles wildcard "*" correctly (allows all tools)
- [ ] Finds recommended tier for denied tools
- [ ] `FormatPermissionError()` includes tier, tool, allowed tools, override suggestion
- [ ] `go test ./pkg/routing` passes with ≥80% coverage

**Why This Matters**: Tool permission enforcement prevents tier violations. Clear error messages help users understand restrictions.

---

### GOgent-018: Implement Wildcard Tools Handling

**Time**: 1.5 hours
**Dependencies**: GOgent-017

**Task**:
Handle edge cases where tier.tools is "*" (all tools allowed) vs array of specific tools.

**File**: `pkg/routing/permissions.go` (extend existing)

**Add to existing file**:
```go
// IsWildcardTier returns true if tier allows all tools
func IsWildcardTier(schema *Schema, tier string) bool {
	tierConfig, exists := schema.Tiers[tier]
	if !exists {
		return false
	}

	str, ok := tierConfig.Tools.(string)
	return ok && str == "*"
}

// GetToolsList returns list of allowed tools for tier
// Returns ["*"] for wildcard tiers, actual tool list otherwise
func GetToolsList(schema *Schema, tier string) []string {
	tierConfig, exists := schema.Tiers[tier]
	if !exists {
		return []string{}
	}

	// Check wildcard
	if str, ok := tierConfig.Tools.(string); ok && str == "*" {
		return []string{"*"}
	}

	// Convert array
	toolsArray, ok := tierConfig.Tools.([]interface{})
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(toolsArray))
	for _, t := range toolsArray {
		if toolStr, ok := t.(string); ok {
			result = append(result, toolStr)
		}
	}

	return result
}
```

**Tests**: `pkg/routing/permissions_test.go` (extend existing)

**Add to existing test file**:
```go
func TestIsWildcardTier_True(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {Tools: "*"},
		},
	}

	if !IsWildcardTier(schema, "opus") {
		t.Error("Expected opus to be wildcard tier")
	}
}

func TestIsWildcardTier_False(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {Tools: []interface{}{"Read", "Glob"}},
		},
	}

	if IsWildcardTier(schema, "haiku") {
		t.Error("Expected haiku NOT to be wildcard tier")
	}
}

func TestGetToolsList_Wildcard(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"opus": {Tools: "*"},
		},
	}

	tools := GetToolsList(schema, "opus")

	if len(tools) != 1 || tools[0] != "*" {
		t.Errorf("Expected ['*'], got: %v", tools)
	}
}

func TestGetToolsList_Array(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{
			"haiku": {Tools: []interface{}{"Read", "Glob", "Grep"}},
		},
	}

	tools := GetToolsList(schema, "haiku")

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got: %d", len(tools))
	}

	// Check specific tools exist
	expected := map[string]bool{"Read": true, "Glob": true, "Grep": true}
	for _, tool := range tools {
		if !expected[tool] {
			t.Errorf("Unexpected tool in list: %s", tool)
		}
	}
}

func TestGetToolsList_InvalidTier(t *testing.T) {
	schema := &Schema{
		Tiers: map[string]TierConfig{},
	}

	tools := GetToolsList(schema, "nonexistent")

	if len(tools) != 0 {
		t.Errorf("Expected empty list for invalid tier, got: %v", tools)
	}
}
```

**Acceptance Criteria**:
- [ ] `IsWildcardTier()` returns true when tier.tools is "*"
- [ ] Returns false when tier.tools is array
- [ ] `GetToolsList()` returns ["*"] for wildcard tiers
- [ ] Returns actual tool list for array tiers
- [ ] Returns empty list for invalid tier
- [ ] Tests cover wildcard, array, and invalid cases
- [ ] `go test ./pkg/routing` passes

**Why This Matters**: Correct wildcard handling prevents false positives (blocking allowed tools) and false negatives (allowing restricted tools).

---

### GOgent-019: Tool Permission Tests

**Time**: 2 hours
**Dependencies**: GOgent-018

**Task**:
Comprehensive integration tests for tool permission workflow.

**File**: `test/integration/tool_permissions_test.go`

**Implementation**:
```go
package integration

import (
	"testing"

	"github.com/yourusername/gogent/pkg/routing"
)

func TestToolPermissions_HaikuTier(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"haiku": {
				Tools: []interface{}{"Read", "Glob", "Grep"},
			},
			"sonnet": {
				Tools: []interface{}{"Read", "Write", "Edit", "Bash", "Glob", "Grep"},
			},
		},
	}

	tests := []struct {
		tool     string
		allowed  bool
		recTier  string
	}{
		{"Read", true, ""},
		{"Glob", true, ""},
		{"Grep", true, ""},
		{"Write", false, "sonnet"},
		{"Edit", false, "sonnet"},
		{"Bash", false, "sonnet"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			result := routing.CheckToolPermission(schema, "haiku", tt.tool)

			if result.Allowed != tt.allowed {
				t.Errorf("Tool %s: expected allowed=%v, got %v", tt.tool, tt.allowed, result.Allowed)
			}

			if !tt.allowed && result.RecommendedTier != tt.recTier {
				t.Errorf("Tool %s: expected recommended tier %s, got %s",
					tt.tool, tt.recTier, result.RecommendedTier)
			}
		})
	}
}

func TestToolPermissions_WildcardTier(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"opus": {
				Tools: "*",
			},
		},
	}

	// All tools should be allowed
	tools := []string{"Read", "Write", "Edit", "Bash", "Task", "UnknownTool"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			result := routing.CheckToolPermission(schema, "opus", tool)

			if !result.Allowed {
				t.Errorf("Wildcard tier should allow %s", tool)
			}
		})
	}
}

func TestToolPermissions_TierNotExists(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"haiku": {
				Tools: []interface{}{"Read"},
			},
		},
	}

	result := routing.CheckToolPermission(schema, "nonexistent", "Read")

	if result.Allowed {
		t.Error("Nonexistent tier should deny all tools")
	}
}

func TestToolPermissions_ErrorMessages(t *testing.T) {
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"haiku": {
				Tools: []interface{}{"Read", "Glob"},
			},
			"sonnet": {
				Tools: []interface{}{"Read", "Write", "Edit"},
			},
		},
	}

	result := routing.CheckToolPermission(schema, "haiku", "Write")

	if result.Allowed {
		t.Fatal("Write should not be allowed for haiku")
	}

	errMsg := result.FormatPermissionError()

	// Verify error message components
	checks := []struct {
		name   string
		substr string
	}{
		{"mentions tool", "Write"},
		{"mentions current tier", "haiku"},
		{"mentions allowed tools", "Read"},
		{"mentions recommended tier", "sonnet"},
		{"suggests override", "--force-tier="},
	}

	for _, check := range checks {
		if !contains(errMsg, check.substr) {
			t.Errorf("Error message should %s (substring: %s). Got: %s",
				check.name, check.substr, errMsg)
		}
	}
}

func TestToolPermissions_RealWorldScenario(t *testing.T) {
	// Simulate real routing-schema.json structure
	schema := &routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"haiku": {
				Description: "Mechanical work tier",
				Model:       "claude-3-haiku",
				Tools:       []interface{}{"Read", "Glob", "Grep"},
			},
			"haiku_thinking": {
				Description: "Structured output tier",
				Model:       "claude-3-haiku",
				Thinking:    true,
				Tools:       []interface{}{"Read", "Write", "Edit", "Glob", "Grep"},
			},
			"sonnet": {
				Description: "Reasoning and implementation tier",
				Model:       "claude-3.5-sonnet",
				Thinking:    true,
				Tools:       []interface{}{"Read", "Write", "Edit", "Bash", "Glob", "Grep", "Task"},
			},
			"opus": {
				Description: "Complex reasoning tier",
				Model:       "claude-opus-4",
				Thinking:    true,
				Tools:       "*",
			},
		},
	}

	// Test haiku tier restrictions
	t.Run("Haiku restrictions", func(t *testing.T) {
		if r := routing.CheckToolPermission(schema, "haiku", "Write"); r.Allowed {
			t.Error("Haiku should not allow Write")
		}
	})

	// Test haiku_thinking additions
	t.Run("Haiku thinking additions", func(t *testing.T) {
		if r := routing.CheckToolPermission(schema, "haiku_thinking", "Write"); !r.Allowed {
			t.Error("Haiku thinking should allow Write")
		}

		if r := routing.CheckToolPermission(schema, "haiku_thinking", "Bash"); r.Allowed {
			t.Error("Haiku thinking should NOT allow Bash")
		}
	})

	// Test sonnet full permissions
	t.Run("Sonnet permissions", func(t *testing.T) {
		tools := []string{"Read", "Write", "Edit", "Bash", "Task"}
		for _, tool := range tools {
			if r := routing.CheckToolPermission(schema, "sonnet", tool); !r.Allowed {
				t.Errorf("Sonnet should allow %s", tool)
			}
		}
	})

	// Test opus wildcard
	t.Run("Opus wildcard", func(t *testing.T) {
		if !routing.IsWildcardTier(schema, "opus") {
			t.Error("Opus should be wildcard tier")
		}

		if r := routing.CheckToolPermission(schema, "opus", "AnythingGoes"); !r.Allowed {
			t.Error("Opus should allow any tool")
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
```

**Acceptance Criteria**:
- [ ] Tests verify haiku tier allows Read/Glob/Grep only
- [ ] Tests verify sonnet tier allows additional tools
- [ ] Tests verify opus wildcard allows everything
- [ ] Tests verify error messages contain required information
- [ ] Real-world scenario test covers all 4 tiers
- [ ] `go test ./test/integration` passes
- [ ] Coverage ≥80% for pkg/routing/permissions.go

**Why This Matters**: Comprehensive tests ensure permission system correctly enforces tier restrictions in all scenarios.

---

## Cross-File References

- **Depends on**: [01-week1-foundation-events.md](01-week1-foundation-events.md) - GOgent-007 (ParseToolEvent)
- **Used by**: [03-week1-validation-cli.md](03-week1-validation-cli.md) - GOgent-020 to 025 (validation orchestrator)
- **Standards**: [00-overview.md](00-overview.md) - Error format, testing strategy, XDG paths

---

## Quick Reference

**Key Functions Added**:
- `routing.ParseOverrides()` - Extract --force-* flags
- `config.GetGOgentDir()` - XDG-compliant path resolution
- `routing.LogViolation()` - JSONL audit logging
- `routing.LoadScoutMetrics()` - Read scout metrics with validation
- `routing.UpdateTierFromMetrics()` - Update current tier from complexity
- `routing.CheckToolPermission()` - Validate tool against tier permissions
- `routing.IsWildcardTier()` - Check if tier allows all tools

**Files Created**:
- `pkg/routing/overrides.go`
- `pkg/routing/violations.go`
- `pkg/config/paths.go`
- `pkg/routing/metrics.go`
- `pkg/routing/tier_update.go`
- `pkg/routing/permissions.go`
- `test/integration/overrides_test.go`
- `test/integration/complexity_routing_test.go`
- `test/integration/tool_permissions_test.go`

**Total Lines**: ~1400 lines of implementation + tests

---

## Completion Checklist

Before marking this file complete, verify:

- [ ] All 10 tickets (GOgent-010 to 019) have complete implementations
- [ ] All functions include complete imports
- [ ] All error messages follow `[component] What. Why. How.` format
- [ ] All paths use XDG compliance (no /tmp hardcoding)
- [ ] All tests include positive, negative, and edge cases
- [ ] Test coverage ≥80% for all packages
- [ ] All acceptance criteria checkboxes filled
- [ ] Cross-references to other files accurate
- [ ] No "omitted for brevity" or "implement logic here" placeholders

---

**Next**: [03-week1-validation-cli.md](03-week1-validation-cli.md) - GOgent-020 to 025 (Task validation, Einstein blocking, CLI build)
