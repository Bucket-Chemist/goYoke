package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

// Schema version for handoff format evolution
const HandoffSchemaVersion = "1.2"

// Handoff represents a complete session handoff document in JSONL format
type Handoff struct {
	SchemaVersion string            `json:"schema_version"`
	Timestamp     int64             `json:"timestamp"`
	SessionID     string            `json:"session_id"`
	Context       SessionContext    `json:"context"`
	Artifacts     HandoffArtifacts  `json:"artifacts"`
	Actions       []Action          `json:"actions"`
}

// SessionContext captures the session's execution context
type SessionContext struct {
	ProjectDir    string            `json:"project_dir"`
	Metrics       SessionMetrics    `json:"metrics"`
	ActiveTicket  string            `json:"active_ticket,omitempty"`
	Phase         string            `json:"phase,omitempty"`
	GitInfo       GitInfo           `json:"git_info,omitempty"`
}

// HandoffArtifacts contains references to session artifacts
type HandoffArtifacts struct {
	SharpEdges        []SharpEdge        `json:"sharp_edges"`
	RoutingViolations []RoutingViolation `json:"routing_violations"`
	ErrorPatterns     []ErrorPattern     `json:"error_patterns"`
	UserIntents       []UserIntent       `json:"user_intents"`
	// Extended fields (v1.1 - backward compatible via omitempty)
	Decisions           []Decision           `json:"decisions,omitempty"`
	PreferenceOverrides []PreferenceOverride `json:"preference_overrides,omitempty"`
	PerformanceMetrics  []PerformanceMetric  `json:"performance_metrics,omitempty"`
}

// SharpEdge represents a debugging loop or gotcha discovered
// NOTE: Stays on schema v1.0 - all new fields are optional (omitempty)
// Optional fields with omitempty are backward-compatible within v1.0
// Old readers simply ignore new fields, new readers handle missing fields as zero values
type SharpEdge struct {
	// Existing fields (DO NOT CHANGE order or tags)
	File                string `json:"file"`
	ErrorType           string `json:"error_type"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Context             string `json:"context,omitempty"`
	Timestamp           int64  `json:"timestamp"`

	// Extended fields (v1.0 compatible - all omitempty)
	ErrorMessage string `json:"error_message,omitempty"` // Full error text
	Severity     string `json:"severity,omitempty"`      // "high", "medium", "low"
	Resolution   string `json:"resolution,omitempty"`    // What fixed it
	ResolvedAt   int64  `json:"resolved_at,omitempty"`   // When resolved (0 = unresolved)

	// NEW FIELDS (v1.2 - GOgent-037b/c, 038c/d series)
	Type            string `json:"type,omitempty"`             // "sharp_edge"
	Tool            string `json:"tool,omitempty"`             // "Edit", "Write", "Bash"
	CodeSnippet     string `json:"code_snippet,omitempty"`     // Code context around error (037b)
	Status          string `json:"status,omitempty"`           // "pending_review", "resolved"
	AttemptedChange string `json:"attempted_change,omitempty"` // What was attempted (037c)
}

// RoutingViolation represents a tier/agent routing issue
type RoutingViolation struct {
	Agent          string `json:"agent"`
	ViolationType  string `json:"violation_type"`
	ExpectedTier   string `json:"expected_tier,omitempty"`
	ActualTier     string `json:"actual_tier,omitempty"`
	Timestamp      int64  `json:"timestamp"`
}

// ErrorPattern represents a recurring error pattern
type ErrorPattern struct {
	ErrorType  string `json:"error_type"`
	Count      int    `json:"count"`
	LastSeen   int64  `json:"last_seen"`
	Context    string `json:"context,omitempty"`
}

// Action represents an actionable next step
type Action struct {
	Priority    int    `json:"priority"`
	Description string `json:"description"`
	Context     string `json:"context,omitempty"`
}

// GitInfo captures git repository state
type GitInfo struct {
	Branch      string   `json:"branch"`
	IsDirty     bool     `json:"is_dirty"`
	Uncommitted []string `json:"uncommitted,omitempty"` // Files with uncommitted changes
}

// HandoffConfig contains paths for handoff generation
type HandoffConfig struct {
	ProjectDir        string
	HandoffPath       string // .claude/memory/handoffs.jsonl
	PendingPath       string // .claude/memory/pending-learnings.jsonl
	ViolationsPath    string // .claude/memory/routing-violations.jsonl
	ErrorPatternsPath string // /tmp/claude-error-patterns.jsonl
	TranscriptPath    string // Optional: session transcript for archival
	UserIntentsPath   string // .claude/memory/user-intents.jsonl
	DecisionsPath     string // .claude/memory/decisions.jsonl
	PreferencesPath   string // .claude/memory/preferences.jsonl
	PerformancePath   string // .claude/memory/performance.jsonl
}

// HandoffMetrics captures timing and artifact counts from handoff generation
type HandoffMetrics struct {
	GenerationTimeMs int64 // Time to generate handoff in milliseconds
	SharpEdgeCount   int   // Number of sharp edges captured
	ViolationCount   int   // Number of routing violations
	PatternCount     int   // Number of unique violation patterns
}

// countPatterns counts unique ViolationType values in routing violations
func countPatterns(violations []RoutingViolation) int {
	seen := make(map[string]struct{})
	for _, v := range violations {
		if v.ViolationType != "" {
			seen[v.ViolationType] = struct{}{}
		}
	}
	return len(seen)
}

// DefaultHandoffConfig creates default paths for handoff generation
func DefaultHandoffConfig(projectDir string) *HandoffConfig {
	claudeDir := filepath.Join(projectDir, ".claude", "memory")

	return &HandoffConfig{
		ProjectDir:        projectDir,
		HandoffPath:       filepath.Join(claudeDir, "handoffs.jsonl"),
		PendingPath:       filepath.Join(claudeDir, "pending-learnings.jsonl"),
		ViolationsPath:    config.GetViolationsLogPath(),
		ErrorPatternsPath: "/tmp/claude-error-patterns.jsonl",
		UserIntentsPath:   filepath.Join(claudeDir, "user-intents.jsonl"),
		DecisionsPath:     filepath.Join(claudeDir, "decisions.jsonl"),
		PreferencesPath:   filepath.Join(claudeDir, "preferences.jsonl"),
		PerformancePath:   filepath.Join(claudeDir, "performance.jsonl"),
	}
}

// GenerateHandoff creates a JSONL handoff document with session context.
// Returns the generated Handoff, metrics about the generation, and any error.
func GenerateHandoff(cfg *HandoffConfig, metrics *SessionMetrics) (*Handoff, *HandoffMetrics, error) {
	startTime := time.Now()

	if cfg == nil {
		return nil, nil, fmt.Errorf("[handoff] Config nil. Cannot generate handoff without configuration. Provide valid HandoffConfig.")
	}

	if metrics == nil {
		return nil, nil, fmt.Errorf("[handoff] Metrics nil. Cannot generate handoff without session metrics. Provide valid SessionMetrics.")
	}

	// Build session context
	context := buildSessionContext(cfg, metrics)

	// Load artifacts
	artifacts, err := LoadArtifacts(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("[handoff] Failed to load artifacts: %w", err)
	}

	// Generate actions
	actions := generateActions(artifacts)

	// Create handoff document
	handoff := &Handoff{
		SchemaVersion: HandoffSchemaVersion,
		Timestamp:     time.Now().Unix(),
		SessionID:     metrics.SessionID,
		Context:       context,
		Artifacts:     artifacts,
		Actions:       actions,
	}

	// Ensure directory exists
	dir := filepath.Dir(cfg.HandoffPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, fmt.Errorf("[handoff] Failed to create directory %s: %w. Check write permissions.", dir, err)
	}

	// Append to JSONL file
	f, err := os.OpenFile(cfg.HandoffPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("[handoff] Failed to open handoff file %s: %w. Check write permissions.", cfg.HandoffPath, err)
	}
	defer f.Close()

	// Serialize to JSON and append
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(handoff); err != nil {
		return nil, nil, fmt.Errorf("[handoff] Failed to write handoff: %w", err)
	}

	// Collect metrics after successful JSONL write
	handoffMetrics := &HandoffMetrics{
		GenerationTimeMs: time.Since(startTime).Milliseconds(),
		SharpEdgeCount:   len(artifacts.SharpEdges),
		ViolationCount:   len(artifacts.RoutingViolations),
		PatternCount:     countPatterns(artifacts.RoutingViolations),
	}

	return handoff, handoffMetrics, nil
}

// LoadHandoff loads the most recent handoff from JSONL file.
// It reads the schema_version field and migrates if needed.
func LoadHandoff(handoffPath string) (*Handoff, error) {
	data, err := os.ReadFile(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No handoff file is normal
		}
		return nil, fmt.Errorf("[handoff] Failed to read JSONL from %s: %w. Check file exists and is readable.", handoffPath, err)
	}

	// Split into lines and find last non-empty line
	lines := strings.Split(string(data), "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" {
			lastLine = trimmed
			break
		}
	}

	if lastLine == "" {
		return nil, nil // Empty file is acceptable
	}

	// Check schema version first
	var versionCheck struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal([]byte(lastLine), &versionCheck); err != nil {
		return nil, fmt.Errorf("[handoff] Failed to parse JSONL schema version from %s: %w. File may be corrupted.", handoffPath, err)
	}

	// Migrate if needed
	if versionCheck.SchemaVersion != HandoffSchemaVersion {
		return migrateHandoff(versionCheck.SchemaVersion, []byte(lastLine))
	}

	// Current version - parse directly
	var handoff Handoff
	if err := json.Unmarshal([]byte(lastLine), &handoff); err != nil {
		return nil, fmt.Errorf("[handoff] Failed to parse JSONL from %s: %w. Schema may have changed.", handoffPath, err)
	}

	return &handoff, nil
}

// migrateHandoff handles conversion from older schema versions to current.
func migrateHandoff(oldVersion string, data []byte) (*Handoff, error) {
	switch oldVersion {
	case "1.0":
		// v1.0 -> v1.1 migration: new fields have omitempty, so just parse directly
		// Missing fields will be zero values (empty slices after initialization)
		var handoff Handoff
		if err := json.Unmarshal(data, &handoff); err != nil {
			return nil, fmt.Errorf("[handoff] Failed to parse v1.0 handoff: %w", err)
		}
		// Initialize new slices if nil (v1.0 handoffs won't have these fields)
		if handoff.Artifacts.Decisions == nil {
			handoff.Artifacts.Decisions = []Decision{}
		}
		if handoff.Artifacts.PreferenceOverrides == nil {
			handoff.Artifacts.PreferenceOverrides = []PreferenceOverride{}
		}
		if handoff.Artifacts.PerformanceMetrics == nil {
			handoff.Artifacts.PerformanceMetrics = []PerformanceMetric{}
		}
		// Update schema version to current
		handoff.SchemaVersion = HandoffSchemaVersion
		return &handoff, nil

	case "1.1":
		// v1.1 -> v1.2 migration: new SharpEdge fields have omitempty, parse directly
		var handoff Handoff
		if err := json.Unmarshal(data, &handoff); err != nil {
			return nil, fmt.Errorf("[handoff] Failed to parse v1.1 handoff: %w", err)
		}
		// Update schema version to current
		handoff.SchemaVersion = HandoffSchemaVersion
		return &handoff, nil

	case "1.2":
		// Current version - parse directly
		var handoff Handoff
		if err := json.Unmarshal(data, &handoff); err != nil {
			return nil, fmt.Errorf("[handoff] Failed to parse v1.2 handoff: %w", err)
		}
		return &handoff, nil

	case "2.0":
		// Future version - migration stub
		// When v2.0 is defined, implement conversion logic here
		return nil, fmt.Errorf("[handoff] Migration from v2.0 to v%s not yet implemented. This is a bug.", HandoffSchemaVersion)

	case "":
		// Empty version - treat as unknown
		return nil, fmt.Errorf("[handoff] Unsupported schema version (empty). Expected v%s or older. Upgrade gogent-archive binary.", HandoffSchemaVersion)

	default:
		return nil, fmt.Errorf("[handoff] Unsupported schema version %s. Expected v%s or older. Upgrade gogent-archive binary.", oldVersion, HandoffSchemaVersion)
	}
}

// LoadAllHandoffs loads all handoffs from JSONL file
func LoadAllHandoffs(handoffPath string) ([]Handoff, error) {
	file, err := os.Open(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Handoff{}, nil // No handoff file is normal
		}
		return nil, fmt.Errorf("[handoff] Failed to open %s: %w", handoffPath, err)
	}
	defer file.Close()

	var handoffs []Handoff
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var h Handoff
		if err := json.Unmarshal([]byte(line), &h); err != nil {
			// Skip malformed lines but continue
			continue
		}
		handoffs = append(handoffs, h)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[handoff] Error reading handoff file: %w", err)
	}

	return handoffs, nil
}

// buildSessionContext creates session context from config and metrics
func buildSessionContext(cfg *HandoffConfig, metrics *SessionMetrics) SessionContext {
	context := SessionContext{
		ProjectDir: cfg.ProjectDir,
		Metrics:    *metrics,
	}

	// Get active ticket if available
	context.ActiveTicket = getActiveTicket(cfg.ProjectDir)

	// Get git info
	context.GitInfo = collectGitInfo(cfg.ProjectDir)

	return context
}

// getActiveTicket attempts to load current ticket from .ticket-current
func getActiveTicket(projectDir string) string {
	ticketPath := filepath.Join(projectDir, ".ticket-current")
	data, err := os.ReadFile(ticketPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// collectGitInfo gathers git repository state using git commands.
// Returns empty GitInfo{} silently for non-git directories or on command failures.
// Errors are logged to stderr with [git-info] prefix but do not propagate.
func collectGitInfo(projectDir string) GitInfo {
	info := GitInfo{}

	// Check if this is a git repository
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = projectDir
	if err := cmd.Run(); err != nil {
		// Not a git repo - return empty info silently
		return info
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectDir
	if output, err := cmd.Output(); err == nil {
		info.Branch = strings.TrimSpace(string(output))
	} else {
		// Log warning but continue - partial git info is acceptable
		fmt.Fprintf(os.Stderr, "[git-info] Failed to get branch name. Command failed with exit code %v. Working directory: %s. Continuing with partial git info.\n", err, projectDir)
	}

	// Get dirty status - check if there are uncommitted changes
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectDir
	if output, err := cmd.Output(); err == nil {
		statusOutput := strings.TrimSpace(string(output))
		info.IsDirty = len(statusOutput) > 0

		// Parse uncommitted files if dirty
		if info.IsDirty {
			lines := strings.Split(statusOutput, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				// Status format: "XY filename" where XY are status codes
				// Extract filename (skip first 3 characters: status codes + space)
				if len(line) > 3 {
					filename := strings.TrimSpace(line[3:])
					info.Uncommitted = append(info.Uncommitted, filename)
				}
			}
		}
	} else {
		// Log warning but continue
		fmt.Fprintf(os.Stderr, "[git-info] Failed to get working tree status. Command 'git status --porcelain' failed with exit code %v. Working directory: %s. Continuing with partial git info.\n", err, projectDir)
	}

	// Optional: Get ahead/behind count (upstream tracking)
	// This command may fail if no upstream is configured - that's acceptable
	cmd = exec.Command("git", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	cmd.Dir = projectDir
	if output, err := cmd.Output(); err == nil {
		// Output format: "N\tM" where N=behind, M=ahead
		// We don't store this yet in GitInfo struct, but logging for future enhancement
		countStr := strings.TrimSpace(string(output))
		if countStr != "" {
			// Future: Parse and store in GitInfo if struct is extended
			_ = countStr
		}
	}
	// Silently ignore upstream count errors - not all repos have upstreams

	return info
}

// generateActions creates prioritized action list from artifacts
func generateActions(artifacts HandoffArtifacts) []Action {
	var actions []Action
	priority := 1

	// Priority 1: Sharp edges
	if len(artifacts.SharpEdges) > 0 {
		actions = append(actions, Action{
			Priority:    priority,
			Description: fmt.Sprintf("Review %d sharp edge(s) before continuing work", len(artifacts.SharpEdges)),
			Context:     "Debugging loops captured - may indicate missing patterns or documentation",
		})
		priority++
	}

	// Priority 2: Routing violations
	if len(artifacts.RoutingViolations) > 0 {
		actions = append(actions, Action{
			Priority:    priority,
			Description: fmt.Sprintf("Review %d routing violation(s) for pattern issues", len(artifacts.RoutingViolations)),
			Context:     "May indicate incorrect tier selection or agent usage",
		})
		priority++
	}

	// Priority 3: Error patterns
	if len(artifacts.ErrorPatterns) > 0 {
		actions = append(actions, Action{
			Priority:    priority,
			Description: fmt.Sprintf("Investigate %d error pattern(s)", len(artifacts.ErrorPatterns)),
			Context:     "Recurring errors may need systematic fixes",
		})
		priority++
	}

	return actions
}
