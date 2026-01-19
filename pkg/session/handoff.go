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
const HandoffSchemaVersion = "1.0"

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
	SharpEdges         []SharpEdge       `json:"sharp_edges"`
	RoutingViolations  []RoutingViolation `json:"routing_violations"`
	ErrorPatterns      []ErrorPattern     `json:"error_patterns"`
}

// SharpEdge represents a debugging loop or gotcha discovered
type SharpEdge struct {
	File               string `json:"file"`
	ErrorType          string `json:"error_type"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	Context            string `json:"context,omitempty"`
	Timestamp          int64  `json:"timestamp"`
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
	}
}

// GenerateHandoff creates a JSONL handoff document with session context
func GenerateHandoff(cfg *HandoffConfig, metrics *SessionMetrics) error {
	if cfg == nil {
		return fmt.Errorf("[handoff] Config nil. Cannot generate handoff without configuration. Provide valid HandoffConfig.")
	}

	if metrics == nil {
		return fmt.Errorf("[handoff] Metrics nil. Cannot generate handoff without session metrics. Provide valid SessionMetrics.")
	}

	// Build session context
	context := buildSessionContext(cfg, metrics)

	// Load artifacts
	artifacts, err := LoadArtifacts(cfg)
	if err != nil {
		return fmt.Errorf("[handoff] Failed to load artifacts: %w", err)
	}

	// Generate actions
	actions := generateActions(artifacts)

	// Create handoff document
	handoff := Handoff{
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
		return fmt.Errorf("[handoff] Failed to create directory %s: %w. Check write permissions.", dir, err)
	}

	// Append to JSONL file
	f, err := os.OpenFile(cfg.HandoffPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("[handoff] Failed to open handoff file %s: %w. Check write permissions.", cfg.HandoffPath, err)
	}
	defer f.Close()

	// Serialize to JSON and append
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(handoff); err != nil {
		return fmt.Errorf("[handoff] Failed to write handoff: %w", err)
	}

	return nil
}

// LoadHandoff loads the most recent handoff from JSONL file
func LoadHandoff(handoffPath string) (*Handoff, error) {
	file, err := os.Open(handoffPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No handoff file is normal
		}
		return nil, fmt.Errorf("[handoff] Failed to open %s: %w", handoffPath, err)
	}
	defer file.Close()

	// Read all lines and return the last one
	var lastHandoff *Handoff
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
		lastHandoff = &h
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("[handoff] Error reading handoff file: %w", err)
	}

	return lastHandoff, nil
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
