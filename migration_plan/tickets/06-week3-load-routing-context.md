# Week 4: Session Initialization & Context Loading

**File**: `08-week4-load-routing-context.md`
**Tickets**: GOgent-056 to 062 (7 tickets)
**Total Time**: ~11 hours
**Phase**: Week 4

---

## Navigation

- **Previous**: [05-week2-sharp-edge-memory.md](05-week2-sharp-edge-memory.md) - GOgent-034 to 040
- **Next**: [07-week3-agent-workflow-hooks.md](07-week3-agent-workflow-hooks.md) - GOgent-063 to 074
- **Overview**: [00-overview.md](00-overview.md) - Testing strategy, rollback plan, standards
- **Template**: [TICKET-TEMPLATE.md](TICKET-TEMPLATE.md) - Required ticket structure
- **Untracked Hooks**: [UNTRACKED_HOOKS.md](UNTRACKED_HOOKS.md) - Hook inventory and planning

---

## Summary

This week translates the `load-routing-context.sh` hook from Bash to Go:

1. **SessionStart Event Parsing**: Handle session startup and resume events
2. **Routing Schema Loading**: Load and format routing tier summary
3. **Handoff Document Loading**: Load previous session context for resume
4. **Pending Learnings Detection**: Check for accumulated sharp edges
5. **Git Status Integration**: Include branch and uncommitted changes
6. **Project Type Detection**: Auto-detect Python/R/Shiny/JavaScript projects
7. **CLI Integration**: Build gogent-load-context binary

**Critical Context**:
- **First hook** that fires in every session (SessionStart event)
- Initializes tool counter for attention-gate hook (dependency)
- Loads routing schema that validate-routing hook depends on
- Project type detection drives convention loading in CLAUDE.md
- Handoff loading enables multi-session continuity

**Hook Trigger**: SessionStart (event fired on session startup or resume)

**Input JSON**:
```json
{
  "type": "startup",
  "session_id": "abc123",
  "hook_event_name": "SessionStart"
}
```

**Output JSON**:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "🚀 SESSION INITIALIZED (startup)\n\n<routing tiers>\n\n<git status>\n\n<project type>"
  }
}
```

---

## GOgent-056: Define SessionStart Event Structs

**Time**: 1.5 hours
**Dependencies**: GOgent-002 (STDIN timeout)

**Task**:
Define SessionStartEvent struct with session type detection (startup vs resume).

**File**: `pkg/session/events.go`

**Imports**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)
```

**Implementation**:
```go
// SessionStartEvent represents SessionStart hook event
type SessionStartEvent struct {
	Type          string `json:"type"`           // "startup" or "resume"
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"` // "SessionStart"
}

// ParseSessionStartEvent reads SessionStart event from STDIN
func ParseSessionStartEvent(r io.Reader, timeout time.Duration) (*SessionStartEvent, error) {
	type result struct {
		event *SessionStartEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			ch <- result{nil, fmt.Errorf("[load-context] Failed to read STDIN: %w", err)}
			return
		}

		var event SessionStartEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("[load-context] Failed to parse JSON: %w. Input: %s", err, string(data[:min(100, len(data))]))}
			return
		}

		// Default to "startup" if not specified
		if event.Type == "" {
			event.Type = "startup"
		}

		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("[load-context] STDIN read timeout after %v", timeout)
	}
}

// IsResume returns true if this is a resume session
func (e *SessionStartEvent) IsResume() bool {
	return e.Type == "resume"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Tests**: `pkg/session/events_test.go`

```go
package session

import (
	"strings"
	"testing"
	"time"
)

func TestParseSessionStartEvent_Startup(t *testing.T) {
	jsonInput := `{
		"type": "startup",
		"session_id": "test-123",
		"hook_event_name": "SessionStart"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "startup" {
		t.Errorf("Expected type startup, got: %s", event.Type)
	}

	if event.IsResume() {
		t.Error("Startup session should not be resume")
	}
}

func TestParseSessionStartEvent_Resume(t *testing.T) {
	jsonInput := `{
		"type": "resume",
		"session_id": "test-456",
		"hook_event_name": "SessionStart"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "resume" {
		t.Errorf("Expected type resume, got: %s", event.Type)
	}

	if !event.IsResume() {
		t.Error("Resume session should return true for IsResume()")
	}
}

func TestParseSessionStartEvent_DefaultType(t *testing.T) {
	// Missing "type" field should default to "startup"
	jsonInput := `{
		"session_id": "test-789",
		"hook_event_name": "SessionStart"
	}`

	reader := strings.NewReader(jsonInput)
	event, err := ParseSessionStartEvent(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if event.Type != "startup" {
		t.Errorf("Expected default type startup, got: %s", event.Type)
	}
}

func TestParseSessionStartEvent_Timeout(t *testing.T) {
	// Create a reader that never returns
	reader := &blockingReader{}

	_, err := ParseSessionStartEvent(reader, 100*time.Millisecond)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

// blockingReader never returns data (for timeout tests)
type blockingReader struct{}

func (b *blockingReader) Read(p []byte) (n int, err error) {
	time.Sleep(10 * time.Second)
	return 0, nil
}
```

**Acceptance Criteria**:
- [ ] `ParseSessionStartEvent()` reads SessionStart events from STDIN
- [ ] Implements 5s timeout on STDIN read
- [ ] Defaults `type` to "startup" if missing
- [ ] `IsResume()` correctly identifies resume sessions
- [ ] Tests cover startup, resume, default, timeout cases
- [ ] `go test ./pkg/session` passes

**Why This Matters**: SessionStart is the first event in every Claude session. Correct parsing is critical for all downstream context injection.

---

## GOgent-057: Routing Schema Loading & Formatting

**Time**: 1.5 hours
**Dependencies**: GOgent-004a (routing schema structs)

**Task**:
Load routing schema from `~/.claude/routing-schema.json` and format tier summary for context injection.

**File**: `pkg/session/schema_loader.go`

**Imports**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yourusername/gogent-fortress/pkg/routing"
)
```

**Implementation**:
```go
// LoadRoutingSchemaSummary loads routing schema and formats tier summary
func LoadRoutingSchemaSummary() (string, error) {
	// Get schema path (XDG compliant)
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to get home dir: %w", err)
	}

	schemaPath := filepath.Join(home, ".claude", "routing-schema.json")

	// Check if schema exists
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return "Routing schema not found (expected at ~/.claude/routing-schema.json)", nil
	}

	// Read schema
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to read routing schema: %w", err)
	}

	// Parse schema
	var schema routing.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return "", fmt.Errorf("[load-context] Failed to parse routing schema: %w", err)
	}

	// Format tier summary
	var summary strings.Builder
	summary.WriteString("ROUTING TIERS ACTIVE:\n")

	for tierName, tierConfig := range schema.Tiers {
		// Get first 3 patterns
		patterns := tierConfig.Patterns
		if len(patterns) > 3 {
			patterns = patterns[:3]
		}

		// Get first 4 tools
		tools := tierConfig.Tools
		if len(tools) > 4 {
			tools = tools[:4]
		}

		summary.WriteString(fmt.Sprintf("  • %s: %s... → tools: %s\n",
			tierName,
			strings.Join(patterns, ", "),
			strings.Join(tools, ", "),
		))
	}

	return summary.String(), nil
}
```

**Tests**: `pkg/session/schema_loader_test.go`

```go
package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourusername/gogent-fortress/pkg/routing"
)

func TestLoadRoutingSchemaSummary(t *testing.T) {
	// Create temporary home directory
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	// Create schema directory
	claudeDir := filepath.Join(tmpHome, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create mock schema
	schema := routing.Schema{
		Tiers: map[string]routing.TierConfig{
			"haiku": {
				Patterns: []string{"find files", "search codebase", "grep pattern"},
				Tools:    []string{"Glob", "Grep", "Read", "WebFetch"},
			},
			"sonnet": {
				Patterns: []string{"implement", "refactor", "debug"},
				Tools:    []string{"Read", "Write", "Edit", "Bash"},
			},
		},
	}

	schemaData, _ := json.Marshal(schema)
	schemaPath := filepath.Join(claudeDir, "routing-schema.json")
	os.WriteFile(schemaPath, schemaData, 0644)

	// Load summary
	summary, err := LoadRoutingSchemaSummary()

	if err != nil {
		t.Fatalf("LoadRoutingSchemaSummary failed: %v", err)
	}

	// Verify summary contains expected content
	if !strings.Contains(summary, "ROUTING TIERS ACTIVE") {
		t.Error("Summary should contain header")
	}

	if !strings.Contains(summary, "haiku") {
		t.Error("Summary should contain haiku tier")
	}

	if !strings.Contains(summary, "sonnet") {
		t.Error("Summary should contain sonnet tier")
	}

	if !strings.Contains(summary, "find files") {
		t.Error("Summary should contain pattern examples")
	}

	if !strings.Contains(summary, "Glob") {
		t.Error("Summary should contain tool examples")
	}
}

func TestLoadRoutingSchemaSummary_MissingFile(t *testing.T) {
	// Create temporary home directory without schema
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", originalHome)

	summary, err := LoadRoutingSchemaSummary()

	if err != nil {
		t.Fatalf("Should not error on missing schema, got: %v", err)
	}

	if !strings.Contains(summary, "not found") {
		t.Error("Should indicate schema not found")
	}
}
```

**Acceptance Criteria**:
- [ ] `LoadRoutingSchemaSummary()` reads from `~/.claude/routing-schema.json`
- [ ] Returns formatted tier summary with patterns and tools
- [ ] Limits output to first 3 patterns, first 4 tools per tier
- [ ] Handles missing schema gracefully (returns message, not error)
- [ ] Tests verify formatting and missing file handling
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Routing schema summary is injected in every session, keeping agent aware of tier capabilities and routing rules.

---

## GOgent-058: Handoff Document Loading

**Time**: 1 hour
**Dependencies**: GOgent-056

**Task**:
Load previous session handoff document for resume sessions.

**File**: `pkg/session/handoff_loader.go`

**Imports**:
```go
package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)
```

**Implementation**:
```go
// LoadHandoffDocument loads previous session handoff for resume sessions
func LoadHandoffDocument(projectDir string) (string, error) {
	handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")

	// Check if handoff exists
	if _, err := os.Stat(handoffPath); os.IsNotExist(err) {
		return "No handoff available", nil
	}

	// Read handoff file
	data, err := os.ReadFile(handoffPath)
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to read handoff: %w", err)
	}

	content := string(data)

	// Return first 30 lines (summary)
	lines := strings.Split(content, "\n")
	if len(lines) > 30 {
		lines = lines[:30]
		lines = append(lines, "\n(... truncated, full handoff in .claude/memory/last-handoff.md)")
	}

	return strings.Join(lines, "\n"), nil
}
```

**Tests**: `pkg/session/handoff_loader_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadHandoffDocument(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create mock handoff
	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	handoffContent := `# Session Handoff

## Summary
Last session implemented feature X.

## Pending Tasks
- Complete tests
- Update docs
`
	os.WriteFile(handoffPath, []byte(handoffContent), 0644)

	// Load handoff
	content, err := LoadHandoffDocument(tmpDir)

	if err != nil {
		t.Fatalf("LoadHandoffDocument failed: %v", err)
	}

	if !strings.Contains(content, "Session Handoff") {
		t.Error("Should contain handoff content")
	}

	if !strings.Contains(content, "feature X") {
		t.Error("Should contain session summary")
	}
}

func TestLoadHandoffDocument_Missing(t *testing.T) {
	tmpDir := t.TempDir()

	content, err := LoadHandoffDocument(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing handoff, got: %v", err)
	}

	if !strings.Contains(content, "No handoff available") {
		t.Error("Should indicate no handoff")
	}
}

func TestLoadHandoffDocument_Truncation(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create large handoff (40 lines)
	var lines []string
	for i := 1; i <= 40; i++ {
		lines = append(lines, fmt.Sprintf("Line %d content", i))
	}
	handoffContent := strings.Join(lines, "\n")

	handoffPath := filepath.Join(memoryDir, "last-handoff.md")
	os.WriteFile(handoffPath, []byte(handoffContent), 0644)

	content, err := LoadHandoffDocument(tmpDir)

	if err != nil {
		t.Fatalf("LoadHandoffDocument failed: %v", err)
	}

	// Should be truncated
	if !strings.Contains(content, "truncated") {
		t.Error("Should indicate truncation for large handoff")
	}

	// Should not contain line 35
	if strings.Contains(content, "Line 35") {
		t.Error("Should truncate after 30 lines")
	}
}
```

**Acceptance Criteria**:
- [ ] `LoadHandoffDocument()` reads from `.claude/memory/last-handoff.md`
- [ ] Returns first 30 lines with truncation indicator
- [ ] Handles missing handoff gracefully
- [ ] Tests verify content loading, missing file, truncation
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Handoff loading enables multi-session continuity. Resume sessions need context from previous work.

---

## GOgent-059: Pending Learnings Detection & Git Integration

**Time**: 1.5 hours
**Dependencies**: None

**Task**:
Check for pending learnings and get git status for context.

**File**: `pkg/session/context_collectors.go`

**Imports**:
```go
package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)
```

**Implementation**:
```go
// CheckPendingLearnings checks for accumulated sharp edges
func CheckPendingLearnings(projectDir string) (string, error) {
	pendingPath := filepath.Join(projectDir, ".claude", "memory", "pending-learnings.jsonl")

	// Check if file exists and has content
	info, err := os.Stat(pendingPath)
	if os.IsNotExist(err) {
		return "", nil // No pending learnings
	}
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to stat pending learnings: %w", err)
	}

	if info.Size() == 0 {
		return "", nil // Empty file
	}

	// Count lines
	data, err := os.ReadFile(pendingPath)
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to read pending learnings: %w", err)
	}

	lineCount := strings.Count(string(data), "\n")

	return fmt.Sprintf("⚠️ PENDING LEARNINGS: %d sharp edges from previous sessions need review. Check .claude/memory/pending-learnings.jsonl", lineCount), nil
}

// GetGitStatus returns git branch and uncommitted change count
func GetGitStatus(projectDir string) string {
	// Check if git is available
	if _, err := exec.LookPath("git"); err != nil {
		return "" // Git not available
	}

	// Check if in git repo
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		return "" // Not a git repo
	}

	// Get current branch
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = projectDir
	branchOutput, err := cmd.Output()
	if err != nil {
		return "" // Can't determine branch
	}
	branch := strings.TrimSpace(string(branchOutput))
	if branch == "" {
		branch = "unknown"
	}

	// Get uncommitted changes count
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	statusOutput, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("GIT: Branch '%s'", branch)
	}

	dirtyCount := strings.Count(string(statusOutput), "\n")

	return fmt.Sprintf("GIT: Branch '%s' with %d uncommitted changes", branch, dirtyCount)
}
```

**Tests**: `pkg/session/context_collectors_test.go`

```go
package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPendingLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	memoryDir := filepath.Join(tmpDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	// Create pending learnings
	pendingPath := filepath.Join(memoryDir, "pending-learnings.jsonl")
	content := `{"ts":123,"file":"test.go"}
{"ts":456,"file":"main.go"}
{"ts":789,"file":"utils.go"}
`
	os.WriteFile(pendingPath, []byte(content), 0644)

	// Check pending learnings
	message, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("CheckPendingLearnings failed: %v", err)
	}

	if !strings.Contains(message, "3 sharp edges") {
		t.Errorf("Expected 3 sharp edges, got: %s", message)
	}

	if !strings.Contains(message, "PENDING LEARNINGS") {
		t.Error("Message should indicate pending learnings")
	}
}

func TestCheckPendingLearnings_Missing(t *testing.T) {
	tmpDir := t.TempDir()

	message, err := CheckPendingLearnings(tmpDir)

	if err != nil {
		t.Fatalf("Should not error on missing file, got: %v", err)
	}

	if message != "" {
		t.Errorf("Expected empty message for missing file, got: %s", message)
	}
}

func TestGetGitStatus(t *testing.T) {
	// Skip if git not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git not available")
	}

	// Create temporary git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skip("Cannot initialize git repo")
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	cmd.Run()

	// Create uncommitted change
	os.WriteFile(testFile, []byte("modified"), 0644)

	// Get git status
	status := GetGitStatus(tmpDir)

	if !strings.Contains(status, "GIT:") {
		t.Error("Should contain GIT prefix")
	}

	if !strings.Contains(status, "1 uncommitted") {
		t.Errorf("Should detect 1 uncommitted change, got: %s", status)
	}
}

func TestGetGitStatus_NotGitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	status := GetGitStatus(tmpDir)

	if status != "" {
		t.Errorf("Expected empty string for non-git repo, got: %s", status)
	}
}
```

**Acceptance Criteria**:
- [ ] `CheckPendingLearnings()` counts lines in pending-learnings.jsonl
- [ ] Returns empty string if file missing or empty
- [ ] `GetGitStatus()` returns branch and uncommitted change count
- [ ] Handles non-git directories gracefully
- [ ] Tests verify file counting, missing files, git detection
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Pending learnings alert prevents loss of debugging insights. Git status helps agent understand working state.

---

## GOgent-060: Project Type Detection

**Time**: 1.5 hours
**Dependencies**: None

**Task**:
Auto-detect project type (Python, R, R+Shiny, JavaScript, Go) for convention loading.

**File**: `pkg/session/project_detection.go`

**Imports**:
```go
package session

import (
	"os"
	"path/filepath"
	"strings"
)
```

**Implementation**:
```go
// ProjectType represents detected project language/framework
type ProjectType string

const (
	ProjectGeneric    ProjectType = "generic"
	ProjectPython     ProjectType = "python"
	ProjectR          ProjectType = "r"
	ProjectRShiny     ProjectType = "r-shiny"
	ProjectJavaScript ProjectType = "javascript"
	ProjectGo         ProjectType = "go"
)

// DetectProjectType auto-detects project type from indicator files
func DetectProjectType(projectDir string) ProjectType {
	// Check Python indicators
	pythonFiles := []string{"pyproject.toml", "setup.py", "requirements.txt"}
	for _, file := range pythonFiles {
		if fileExists(filepath.Join(projectDir, file)) {
			return ProjectPython
		}
	}

	// Check R indicators
	if fileExists(filepath.Join(projectDir, "DESCRIPTION")) || fileExists(filepath.Join(projectDir, "renv.lock")) {
		// Check if Shiny app
		descPath := filepath.Join(projectDir, "DESCRIPTION")
		if descContent, err := os.ReadFile(descPath); err == nil {
			if strings.Contains(string(descContent), "shiny") {
				return ProjectRShiny
			}
		}

		// Check for app.R or ui.R
		if fileExists(filepath.Join(projectDir, "app.R")) || fileExists(filepath.Join(projectDir, "ui.R")) {
			return ProjectRShiny
		}

		return ProjectR
	}

	// Check Go indicators
	if fileExists(filepath.Join(projectDir, "go.mod")) {
		return ProjectGo
	}

	// Check JavaScript/TypeScript indicators
	if fileExists(filepath.Join(projectDir, "package.json")) {
		return ProjectJavaScript
	}

	return ProjectGeneric
}

// fileExists checks if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
```

**Tests**: `pkg/session/project_detection_test.go`

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProjectType_Python(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(""), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectPython {
		t.Errorf("Expected Python, got: %s", projectType)
	}
}

func TestDetectProjectType_R(t *testing.T) {
	tmpDir := t.TempDir()

	// Create DESCRIPTION without shiny
	descContent := `Package: mypackage
Title: Test Package
Version: 1.0.0
`
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte(descContent), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectR {
		t.Errorf("Expected R, got: %s", projectType)
	}
}

func TestDetectProjectType_RShiny_Description(t *testing.T) {
	tmpDir := t.TempDir()

	// Create DESCRIPTION with shiny
	descContent := `Package: myapp
Title: Shiny App
Version: 1.0.0
Depends: shiny
`
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte(descContent), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectRShiny {
		t.Errorf("Expected R+Shiny, got: %s", projectType)
	}
}

func TestDetectProjectType_RShiny_AppFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create app.R
	os.WriteFile(filepath.Join(tmpDir, "app.R"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "DESCRIPTION"), []byte(""), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectRShiny {
		t.Errorf("Expected R+Shiny, got: %s", projectType)
	}
}

func TestDetectProjectType_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectGo {
		t.Errorf("Expected Go, got: %s", projectType)
	}
}

func TestDetectProjectType_JavaScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create package.json
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectJavaScript {
		t.Errorf("Expected JavaScript, got: %s", projectType)
	}
}

func TestDetectProjectType_Generic(t *testing.T) {
	tmpDir := t.TempDir()

	projectType := DetectProjectType(tmpDir)

	if projectType != ProjectGeneric {
		t.Errorf("Expected Generic, got: %s", projectType)
	}
}
```

**Acceptance Criteria**:
- [ ] `DetectProjectType()` detects Python, R, R+Shiny, Go, JavaScript
- [ ] Returns generic for unrecognized projects
- [ ] Shiny detection checks DESCRIPTION content and app.R presence
- [ ] Tests verify all project types
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Project type detection drives convention loading (python.md, R.md, etc.) in CLAUDE.md Gate 1.

---

## GOgent-061: Session Context Response Generation

**Time**: 1.5 hours
**Dependencies**: GOgent-056 to 060

**Task**:
Combine all context sources and generate SessionStart hook response.

**File**: `pkg/session/response.go`

**Imports**:
```go
package session

import (
	"encoding/json"
	"fmt"
	"strings"
)
```

**Implementation**:
```go
// SessionContext aggregates all context sources
type SessionContext struct {
	SessionType      string
	RoutingSummary   string
	HandoffContent   string
	PendingLearnings string
	GitStatus        string
	ProjectType      ProjectType
}

// GenerateSessionStartResponse creates context injection response
func GenerateSessionStartResponse(ctx *SessionContext) (string, error) {
	var contextParts []string

	// Add routing summary
	if ctx.RoutingSummary != "" {
		contextParts = append(contextParts, ctx.RoutingSummary)
	}

	// Add handoff for resume sessions
	if ctx.SessionType == "resume" && ctx.HandoffContent != "" {
		contextParts = append(contextParts, "PREVIOUS SESSION HANDOFF:\n"+ctx.HandoffContent)
	}

	// Add pending learnings warning
	if ctx.PendingLearnings != "" {
		contextParts = append(contextParts, ctx.PendingLearnings)
	}

	// Add git status
	if ctx.GitStatus != "" {
		contextParts = append(contextParts, ctx.GitStatus)
	}

	// Add project type
	contextParts = append(contextParts, fmt.Sprintf("PROJECT TYPE DETECTED: %s", ctx.ProjectType))

	// Combine all context
	fullContext := strings.Join(contextParts, "\n\n")

	// Build response
	response := map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName": "SessionStart",
			"additionalContext": fmt.Sprintf(
				"🚀 SESSION INITIALIZED (%s)\n\n%s\n\nRouting hooks are ACTIVE. Tool usage will be validated against routing-schema.json.",
				ctx.SessionType,
				fullContext,
			),
		},
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("[load-context] Failed to marshal response: %w", err)
	}

	return string(data), nil
}

// InitializeToolCounter creates tool counter file for attention-gate
func InitializeToolCounter() error {
	// NOTE: Using fixed filename, not PID-based
	// This allows attention-gate to track across multiple hook invocations
	counterPath := "/tmp/claude-tool-counter"

	if err := os.WriteFile(counterPath, []byte("0"), 0644); err != nil {
		return fmt.Errorf("[load-context] Failed to initialize tool counter: %w", err)
	}

	return nil
}
```

**Tests**: `pkg/session/response_test.go`

```go
package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSessionStartResponse_Startup(t *testing.T) {
	ctx := &SessionContext{
		SessionType:    "startup",
		RoutingSummary: "ROUTING TIERS ACTIVE:\n  • haiku: find, search...",
		GitStatus:      "GIT: Branch 'main' with 2 uncommitted changes",
		ProjectType:    ProjectPython,
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	// Verify valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// Verify structure
	output, ok := parsed["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing hookSpecificOutput")
	}

	context, ok := output["additionalContext"].(string)
	if !ok {
		t.Fatal("Missing additionalContext")
	}

	// Verify content
	if !strings.Contains(context, "SESSION INITIALIZED (startup)") {
		t.Error("Should indicate startup session")
	}

	if !strings.Contains(context, "ROUTING TIERS") {
		t.Error("Should include routing summary")
	}

	if !strings.Contains(context, "GIT: Branch 'main'") {
		t.Error("Should include git status")
	}

	if !strings.Contains(context, "python") {
		t.Error("Should include project type")
	}
}

func TestGenerateSessionStartResponse_Resume(t *testing.T) {
	ctx := &SessionContext{
		SessionType:      "resume",
		RoutingSummary:   "ROUTING TIERS ACTIVE:\n  • haiku: find...",
		HandoffContent:   "# Session Handoff\n\nLast session completed feature X.",
		PendingLearnings: "⚠️ PENDING LEARNINGS: 3 sharp edges",
		ProjectType:      ProjectGo,
	}

	response, err := GenerateSessionStartResponse(ctx)

	if err != nil {
		t.Fatalf("GenerateSessionStartResponse failed: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal([]byte(response), &parsed)
	output := parsed["hookSpecificOutput"].(map[string]interface{})
	context := output["additionalContext"].(string)

	// Verify resume-specific content
	if !strings.Contains(context, "resume") {
		t.Error("Should indicate resume session")
	}

	if !strings.Contains(context, "PREVIOUS SESSION HANDOFF") {
		t.Error("Should include handoff for resume session")
	}

	if !strings.Contains(context, "feature X") {
		t.Error("Should include handoff content")
	}

	if !strings.Contains(context, "PENDING LEARNINGS") {
		t.Error("Should include pending learnings warning")
	}
}

func TestInitializeToolCounter(t *testing.T) {
	// Clean up any existing counter
	os.Remove("/tmp/claude-tool-counter")

	err := InitializeToolCounter()

	if err != nil {
		t.Fatalf("InitializeToolCounter failed: %v", err)
	}

	// Verify file created
	content, err := os.ReadFile("/tmp/claude-tool-counter")
	if err != nil {
		t.Fatal("Counter file not created")
	}

	if string(content) != "0" {
		t.Errorf("Expected counter initialized to 0, got: %s", string(content))
	}
}
```

**Acceptance Criteria**:
- [ ] `GenerateSessionStartResponse()` combines all context sources
- [ ] Includes routing summary, git status, project type for all sessions
- [ ] Includes handoff content only for resume sessions
- [ ] Includes pending learnings warning if present
- [ ] Outputs valid JSON with hookSpecificOutput structure
- [ ] `InitializeToolCounter()` creates /tmp/claude-tool-counter with "0"
- [ ] Tests verify startup vs resume, content inclusion
- [ ] `go test ./pkg/session` passes

**Why This Matters**: Response generation is final step in context injection. Must be correct and complete for agent awareness.

---

## GOgent-062: Build gogent-load-context CLI

**Time**: 1.5 hours
**Dependencies**: GOgent-061

**Task**:
Build CLI binary that reads SessionStart events and generates context injection.

**File**: `cmd/gogent-load-context/main.go`

**Imports**:
```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/yourusername/gogent-fortress/pkg/session"
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

	// Parse SessionStart event
	event, err := session.ParseSessionStartEvent(os.Stdin, DEFAULT_TIMEOUT)
	if err != nil {
		outputError(fmt.Sprintf("Failed to parse event: %v", err))
		os.Exit(1)
	}

	// Initialize tool counter for attention-gate
	if err := session.InitializeToolCounter(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize tool counter: %v\n", err)
		// Don't exit - non-fatal
	}

	// Collect all context
	ctx := &session.SessionContext{
		SessionType: event.Type,
	}

	// Load routing schema summary
	if summary, err := session.LoadRoutingSchemaSummary(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load routing schema: %v\n", err)
	} else {
		ctx.RoutingSummary = summary
	}

	// Load handoff for resume sessions
	if event.IsResume() {
		if handoff, err := session.LoadHandoffDocument(projectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load handoff: %v\n", err)
		} else {
			ctx.HandoffContent = handoff
		}
	}

	// Check pending learnings
	if pending, err := session.CheckPendingLearnings(projectDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to check pending learnings: %v\n", err)
	} else {
		ctx.PendingLearnings = pending
	}

	// Get git status
	ctx.GitStatus = session.GetGitStatus(projectDir)

	// Detect project type
	ctx.ProjectType = session.DetectProjectType(projectDir)

	// Generate response
	response, err := session.GenerateSessionStartResponse(ctx)
	if err != nil {
		outputError(fmt.Sprintf("Failed to generate response: %v", err))
		os.Exit(1)
	}

	// Output response
	fmt.Println(response)
}

// outputError writes error in hook format
func outputError(message string) {
	fmt.Printf(`{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "🔴 %s"
  }
}`, message)
}
```

**Build Script**: `scripts/build-load-context.sh`

```bash
#!/bin/bash
set -euo pipefail

echo "Building gogent-load-context..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-load-context ./cmd/gogent-load-context

echo "✓ Built: bin/gogent-load-context"
```

**Install Script**: `scripts/install-load-context.sh`

```bash
#!/bin/bash
set -euo pipefail

BIN_DIR="${HOME}/.local/bin"
mkdir -p "$BIN_DIR"

cp bin/gogent-load-context "$BIN_DIR/"
chmod +x "$BIN_DIR/gogent-load-context"

echo "✓ Installed to $BIN_DIR/gogent-load-context"
```

**Manual Test**:

```bash
# Build
./scripts/build-load-context.sh

# Test startup session
echo '{"type":"startup","session_id":"test-1","hook_event_name":"SessionStart"}' | \
  ./bin/gogent-load-context

# Test resume session
echo '{"type":"resume","session_id":"test-2","hook_event_name":"SessionStart"}' | \
  CLAUDE_PROJECT_DIR="/path/to/project" ./bin/gogent-load-context
```

**Acceptance Criteria**:
- [ ] CLI reads SessionStart events from STDIN
- [ ] Initializes tool counter file
- [ ] Loads routing schema summary
- [ ] Loads handoff for resume sessions (skips for startup)
- [ ] Checks pending learnings
- [ ] Gets git status
- [ ] Detects project type
- [ ] Outputs complete context injection JSON
- [ ] Build script creates bin/gogent-load-context
- [ ] Install script copies to ~/.local/bin
- [ ] Manual tests successful for startup and resume
- [ ] Warnings printed to stderr, not stdout

**Why This Matters**: CLI is SessionStart hook implementation. Must aggregate all context correctly for first-hook-in-session responsibility.

---

## Cross-File References

- **Depends on**:
  - GOgent-002 (STDIN timeout pattern)
  - GOgent-004a (routing schema structs)
  - GOgent-008a (hook response format)
- **Used by**:
  - Week 5: attention-gate depends on tool counter initialization
  - CLAUDE.md Gate 1: Session initialization protocol references this hook
- **Standards**: [00-overview.md](00-overview.md) - STDIN timeout, error format, XDG paths

---

## Quick Reference

**Key Functions Added**:
- `session.ParseSessionStartEvent()` - Parse SessionStart events
- `session.LoadRoutingSchemaSummary()` - Load routing schema
- `session.LoadHandoffDocument()` - Load session handoff
- `session.CheckPendingLearnings()` - Check for sharp edges
- `session.GetGitStatus()` - Get git branch and changes
- `session.DetectProjectType()` - Auto-detect project type
- `session.GenerateSessionStartResponse()` - Create context injection
- `session.InitializeToolCounter()` - Create counter for attention-gate
- `gogent-load-context` CLI - SessionStart → context workflow

**Files Created**:
- `pkg/session/events.go`
- `pkg/session/schema_loader.go`
- `pkg/session/handoff_loader.go`
- `pkg/session/context_collectors.go`
- `pkg/session/project_detection.go`
- `pkg/session/response.go`
- `cmd/gogent-load-context/main.go`
- `scripts/build-load-context.sh`
- `scripts/install-load-context.sh`

**Total Lines**: ~800 lines of implementation + ~600 lines of tests = ~1400 lines

---

## Completion Checklist

- [ ] All 7 tickets (GOgent-056 to 062) complete
- [ ] All functions have complete imports
- [ ] Error messages use `[component] What. Why. How.` format
- [ ] STDIN timeout implemented (5s)
- [ ] XDG-compliant paths (no /tmp hardcoding except tool counter)
- [ ] Tests cover positive, negative, edge cases
- [ ] Test coverage ≥80%
- [ ] All acceptance criteria filled
- [ ] CLI binary buildable
- [ ] Manual tests successful
- [ ] No placeholders

---

**Next**: [09-week4-agent-workflow-hooks.md](09-week4-agent-workflow-hooks.md) - GOgent-063 to 074 (agent-endstate + attention-gate)
