package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ChaosConfig controls chaos testing parameters.
// The key insight: SharedKeyRatio determines how many agents share the same
// file:errorType composite key, which tests actual contention rather than
// isolated writes to unique keys.
type ChaosConfig struct {
	// NumAgents is the number of concurrent goroutines simulating agents
	NumAgents int

	// FailuresPerAgent is how many failures each agent generates
	FailuresPerAgent int

	// SharedKeyRatio is the fraction of agents that share the same key (0.0-1.0).
	// Default: 0.3 (30% of agents use the same file:errorType).
	// This is CRITICAL for testing actual contention scenarios.
	// Value 0.0 = all unique keys (no contention test)
	// Value 1.0 = all agents share one key (maximum contention)
	SharedKeyRatio float64

	// MaxFailures is the threshold that triggers blocking (default: 3)
	MaxFailures int

	// Duration is the maximum time to wait for all agents to complete
	Duration time.Duration

	// Seed enables reproducible chaos (same seed = same sequence of operations)
	Seed int64

	// SharpEdgePath is the path to gogent-sharp-edge binary
	SharpEdgePath string

	// SchemaPath is the path to routing-schema.json for test isolation
	SchemaPath string

	// AgentsPath is the path to agents-index.json for test isolation
	AgentsPath string

	// Verbose enables debug output
	Verbose bool
}

// DefaultChaosConfig returns sensible defaults for chaos testing.
func DefaultChaosConfig() ChaosConfig {
	return ChaosConfig{
		NumAgents:        10,
		FailuresPerAgent: 20,
		SharedKeyRatio:   0.3, // 30% of agents share keys (tests contention)
		MaxFailures:      3,
		Duration:         30 * time.Second,
		Seed:             time.Now().UnixNano(),
	}
}

// ChaosResult captures outcome from one agent.
type ChaosResult struct {
	AgentID     int    `json:"agent_id"`
	File        string `json:"file"`
	ErrorType   string `json:"error_type"`
	IsShared    bool   `json:"is_shared"`
	FinalCount  int    `json:"final_count"` // Last observed failure count
	BlockedAt   int    `json:"blocked_at"`  // Iteration when first blocked (0 = never)
	WriteErrors int    `json:"write_errors"`
	TotalCalls  int    `json:"total_calls"`
}

// ChaosReport summarizes chaos test outcome.
type ChaosReport struct {
	TotalAgents       int             `json:"total_agents"`
	TotalEvents       int             `json:"total_events"`
	SharedKeyAgents   int             `json:"shared_key_agents"`
	UniqueKeyAgents   int             `json:"unique_key_agents"`
	Passed            bool            `json:"passed"`
	Errors            []string        `json:"errors,omitempty"`
	Duration          time.Duration   `json:"duration"`
	AgentResults      []ChaosResult   `json:"agent_results,omitempty"`
	JSONLValidation   *JSONLCheckResult `json:"jsonl_validation,omitempty"`
}

// JSONLCheckResult captures JSONL integrity validation results.
type JSONLCheckResult struct {
	FilesChecked   int      `json:"files_checked"`
	TotalLines     int      `json:"total_lines"`
	InvalidLines   int      `json:"invalid_lines"`
	CorruptedFiles []string `json:"corrupted_files,omitempty"`
}

// KeyAssignment tracks what file/error an agent uses.
type KeyAssignment struct {
	File      string
	ErrorType string
	IsShared  bool // Whether this key is shared with other agents
}

// ChaosRunner simulates concurrent agents generating failures.
type ChaosRunner struct {
	config    ChaosConfig
	tempDir   string
	results   []ChaosResult
	resultsMu sync.Mutex
}

// NewChaosRunner creates a chaos runner with the given configuration.
// The tempDir should be dedicated to this chaos run (will be populated with files).
func NewChaosRunner(cfg ChaosConfig, tempDir string) *ChaosRunner {
	return &ChaosRunner{
		config:  cfg,
		tempDir: tempDir,
		results: make([]ChaosResult, 0, cfg.NumAgents),
	}
}

// Run executes chaos testing with concurrent agents.
// Returns a report with validation results.
func (c *ChaosRunner) Run() (*ChaosReport, error) {
	start := time.Now()
	rng := rand.New(rand.NewSource(c.config.Seed))

	// Generate agent assignments with shared-key distribution
	assignments := c.generateAssignments(rng)

	// Setup temp directory structure
	if err := c.setupTempDir(); err != nil {
		return nil, fmt.Errorf("setup temp dir: %w", err)
	}

	// Channel for collecting errors during run
	errChan := make(chan error, c.config.NumAgents)

	// Run agents concurrently
	var wg sync.WaitGroup
	for i := 0; i < c.config.NumAgents; i++ {
		wg.Add(1)
		// Each agent gets its own seed derived from main seed + agent ID
		agentSeed := rng.Int63()
		go func(agentID int, assignment KeyAssignment, seed int64) {
			defer wg.Done()
			if err := c.runAgent(agentID, assignment, seed); err != nil {
				select {
				case errChan <- fmt.Errorf("agent %d: %w", agentID, err):
				default:
					// Error channel full, skip
				}
			}
		}(i, assignments[i], agentSeed)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All agents finished normally
	case <-time.After(c.config.Duration):
		return nil, fmt.Errorf("chaos test timed out after %v", c.config.Duration)
	}

	// Collect any errors
	close(errChan)
	var runErrors []string
	for err := range errChan {
		runErrors = append(runErrors, err.Error())
	}

	return c.buildReport(start, assignments, runErrors)
}

// generateAssignments creates key assignments with SharedKeyRatio sharing.
// Shared keys test actual contention; unique keys test isolated writes.
func (c *ChaosRunner) generateAssignments(rng *rand.Rand) []KeyAssignment {
	assignments := make([]KeyAssignment, c.config.NumAgents)

	// Calculate how many agents share keys
	sharedCount := int(float64(c.config.NumAgents) * c.config.SharedKeyRatio)
	if sharedCount < 2 && c.config.SharedKeyRatio > 0 && c.config.NumAgents >= 2 {
		sharedCount = 2 // Need at least 2 to test sharing
	}

	// Shared keys: all use the same file/errorType to create contention
	const sharedFile = "shared_contention.go"
	const sharedError = "ContentionError"

	for i := 0; i < c.config.NumAgents; i++ {
		if i < sharedCount {
			// Shared key assignment - these agents will race on the same key
			assignments[i] = KeyAssignment{
				File:      sharedFile,
				ErrorType: sharedError,
				IsShared:  true,
			}
		} else {
			// Unique key assignment - no contention with other agents
			assignments[i] = KeyAssignment{
				File:      fmt.Sprintf("agent_%d_file.go", i),
				ErrorType: fmt.Sprintf("UniqueError%d", i),
				IsShared:  false,
			}
		}
	}

	// Shuffle to distribute shared and unique agents randomly
	rng.Shuffle(len(assignments), func(i, j int) {
		assignments[i], assignments[j] = assignments[j], assignments[i]
	})

	return assignments
}

// runAgent simulates one agent generating failures.
func (c *ChaosRunner) runAgent(agentID int, assignment KeyAssignment, seed int64) error {
	rng := rand.New(rand.NewSource(seed))
	result := ChaosResult{
		AgentID:   agentID,
		File:      assignment.File,
		ErrorType: assignment.ErrorType,
		IsShared:  assignment.IsShared,
	}

	for i := 0; i < c.config.FailuresPerAgent; i++ {
		result.TotalCalls++

		// Simulate failure via sharp-edge CLI
		count, blocked, err := c.simulateFailure(assignment.File, assignment.ErrorType, agentID)
		if err != nil {
			result.WriteErrors++
			if c.config.Verbose {
				fmt.Printf("[CHAOS] Agent %d: error at iteration %d: %v\n", agentID, i, err)
			}
			continue
		}

		result.FinalCount = count
		if blocked && result.BlockedAt == 0 {
			result.BlockedAt = i + 1 // 1-indexed for human readability
		}

		// Small random delay to increase interleaving (10-110 microseconds)
		// Rationale: Real concurrent agents don't hit the CLI at exactly the same time.
		// This creates realistic interleaving patterns while keeping tests fast.
		delay := 10 + rng.Intn(100)
		time.Sleep(time.Microsecond * time.Duration(delay))
	}

	c.resultsMu.Lock()
	c.results = append(c.results, result)
	c.resultsMu.Unlock()

	return nil
}

// simulateFailure calls gogent-sharp-edge with a failure event.
// Returns the failure count and whether blocking occurred.
func (c *ChaosRunner) simulateFailure(file, errorType string, agentID int) (count int, blocked bool, err error) {
	// Build PostToolUse event with failure
	event := map[string]interface{}{
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": fmt.Sprintf("test %s", file),
		},
		"tool_response": map[string]interface{}{
			"exit_code": 1,
			"output":    fmt.Sprintf("%s: test failed in %s", errorType, file),
		},
		"session_id":      fmt.Sprintf("chaos-agent-%d", agentID),
		"hook_event_name": "PostToolUse",
		"captured_at":     time.Now().Unix(),
	}

	inputBytes, err := json.Marshal(event)
	if err != nil {
		return 0, false, fmt.Errorf("marshal event: %w", err)
	}

	// Execute CLI with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.config.SharpEdgePath)
	cmd.Stdin = bytes.NewReader(inputBytes)
	cmd.Env = c.buildEnv()

	// Process group for clean cancellation
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmdErr := cmd.Run()

	// Check for timeout
	if ctx.Err() != nil {
		return 0, false, fmt.Errorf("timeout: %w", ctx.Err())
	}

	// Parse output
	output := stdout.String()
	if output == "" {
		// No output typically means pass-through (not a failure we track)
		return 0, false, nil
	}

	// Check for blocking decision
	blocked = strings.Contains(output, "block") || strings.Contains(output, "BLOCKED")

	// Try to extract count from output
	var resp map[string]interface{}
	if json.Unmarshal([]byte(output), &resp) == nil {
		if c, ok := resp["consecutive_failures"].(float64); ok {
			count = int(c)
		}
		if d, ok := resp["decision"].(string); ok && d == "block" {
			blocked = true
		}
	}

	// Non-zero exit code with blocking is expected
	if cmdErr != nil && !blocked {
		// Actual error (not just blocking)
		return count, blocked, fmt.Errorf("CLI error: %v (stderr: %s)", cmdErr, stderr.String())
	}

	return count, blocked, nil
}

// buildEnv creates environment for CLI execution with test isolation.
func (c *ChaosRunner) buildEnv() []string {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"NO_COLOR=1",
		"TERM=dumb",
	}

	// Test isolation
	env = append(env, "GOGENT_PROJECT_DIR="+c.tempDir)
	env = append(env, "GOGENT_STORAGE_PATH="+filepath.Join(c.tempDir, ".gogent", "failure-tracker.jsonl"))
	env = append(env, fmt.Sprintf("GOGENT_MAX_FAILURES=%d", c.config.MaxFailures))
	env = append(env, "GOGENT_FAILURE_WINDOW=999999999") // Very long window

	if c.config.SchemaPath != "" {
		env = append(env, "GOGENT_ROUTING_SCHEMA="+c.config.SchemaPath)
	}
	if c.config.AgentsPath != "" {
		env = append(env, "GOGENT_AGENTS_INDEX="+c.config.AgentsPath)
	}

	return env
}

// setupTempDir creates required directories for the chaos test.
func (c *ChaosRunner) setupTempDir() error {
	dirs := []string{
		filepath.Join(c.tempDir, ".gogent", "memory"),
		filepath.Join(c.tempDir, ".gogent"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return nil
}

// buildReport analyzes results for correctness and builds the final report.
func (c *ChaosRunner) buildReport(start time.Time, assignments []KeyAssignment, runErrors []string) (*ChaosReport, error) {
	report := &ChaosReport{
		TotalAgents:  c.config.NumAgents,
		Duration:     time.Since(start),
		AgentResults: c.results,
		Errors:       runErrors,
	}

	// Count shared vs unique key agents
	for _, a := range assignments {
		if a.IsShared {
			report.SharedKeyAgents++
		} else {
			report.UniqueKeyAgents++
		}
	}

	// Count total events
	for _, r := range c.results {
		report.TotalEvents += r.TotalCalls - r.WriteErrors
	}

	// Validate: Check for JSONL corruption
	jsonlResult, jsonlErr := c.validateJSONL()
	report.JSONLValidation = jsonlResult
	if jsonlErr != nil {
		report.Errors = append(report.Errors, jsonlErr.Error())
	}

	// Validate: Shared-key agents should see blocking after combined failures exceed threshold
	if report.SharedKeyAgents > 0 {
		totalSharedFailures := 0
		sharedBlocked := false
		for _, r := range c.results {
			if r.IsShared {
				totalSharedFailures += r.TotalCalls - r.WriteErrors
				if r.BlockedAt > 0 {
					sharedBlocked = true
				}
			}
		}

		// If total shared failures exceed threshold, at least one should have been blocked
		if totalSharedFailures >= c.config.MaxFailures && !sharedBlocked {
			report.Errors = append(report.Errors,
				fmt.Sprintf("shared keys had %d total failures but no blocking occurred (threshold: %d)",
					totalSharedFailures, c.config.MaxFailures))
		}
	}

	// Validate: Check for count consistency among shared-key agents
	// Due to race conditions, counts may differ but should be within reasonable bounds
	sharedCounts := make(map[string][]int)
	for _, r := range c.results {
		if r.IsShared {
			key := r.File + ":" + r.ErrorType
			sharedCounts[key] = append(sharedCounts[key], r.FinalCount)
		}
	}

	for key, counts := range sharedCounts {
		if len(counts) < 2 {
			continue
		}
		// Check that counts are reasonably consistent (within 2x of max failures)
		// Exact consistency is not expected due to race conditions
		minCount, maxCount := counts[0], counts[0]
		for _, c := range counts {
			if c < minCount {
				minCount = c
			}
			if c > maxCount {
				maxCount = c
			}
		}
		// If variance is huge, something went wrong
		if maxCount > 0 && minCount < maxCount/2 && maxCount > c.config.MaxFailures*2 {
			report.Errors = append(report.Errors,
				fmt.Sprintf("key %s: highly inconsistent counts (%d to %d)", key, minCount, maxCount))
		}
	}

	report.Passed = len(report.Errors) == 0

	return report, nil
}

// validateJSONL checks all JSONL files for corruption.
func (c *ChaosRunner) validateJSONL() (*JSONLCheckResult, error) {
	result := &JSONLCheckResult{}

	jsonlFiles := []string{
		filepath.Join(c.tempDir, ".gogent", "memory", "pending-learnings.jsonl"),
		filepath.Join(c.tempDir, ".gogent", "failure-tracker.jsonl"),
	}

	for _, path := range jsonlFiles {
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return result, fmt.Errorf("read %s: %w", path, err)
		}

		result.FilesChecked++

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			result.TotalLines++

			var obj interface{}
			if err := json.Unmarshal([]byte(line), &obj); err != nil {
				result.InvalidLines++
				if !containsString(result.CorruptedFiles, path) {
					result.CorruptedFiles = append(result.CorruptedFiles, path)
				}
			}
		}
	}

	if len(result.CorruptedFiles) > 0 {
		return result, fmt.Errorf("JSONL corruption detected in %d files", len(result.CorruptedFiles))
	}

	return result, nil
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
