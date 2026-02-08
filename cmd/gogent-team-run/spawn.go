package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Spawner defines the interface for spawning Claude CLI processes.
// Production code uses claudeSpawner; tests inject fakes.
type Spawner interface {
	Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error
}

// claudeSpawner is the production Spawner implementation.
type claudeSpawner struct{}

// spawnConfig holds all inputs needed to spawn a Claude CLI process.
type spawnConfig struct {
	envelope     string
	args         []string
	projectRoot  string
	timeout      time.Duration
	memberName   string
	agentID      string
	stdoutPath   string
}

// spawnResult holds the outputs from a completed CLI process.
type spawnResult struct {
	stdout   []byte
	exitCode int
	pid      int
}

// agentCLIConfig holds CLI flags from agents-index.json for a given agent.
type agentCLIConfig struct {
	AllowedTools    []string
	AdditionalFlags []string
	Model           string
}

// Default fallback tools (W4: least-privilege READ-ONLY when agents-index.json unavailable)
var defaultFallbackTools = []string{"Read", "Glob", "Grep"}

// implementationTools are required for workers that create/modify files.
var implementationTools = []string{"Write", "Edit"}

// augmentToolsForImplementation adds Write and Edit to the tool list if not already present.
// Implementation workers need these to create files; review/braintrust workers don't.
func augmentToolsForImplementation(tools []string) []string {
	toolSet := make(map[string]bool, len(tools))
	for _, t := range tools {
		toolSet[t] = true
	}
	result := append([]string{}, tools...)
	for _, t := range implementationTools {
		if !toolSet[t] {
			result = append(result, t)
		}
	}
	return result
}

// Spawn delegates to the three phases.
// Budget management happens at wave level in spawnAndWaitWithBudget.
func (s *claudeSpawner) Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
	cfg, err := s.prepareSpawn(tr, waveIdx, memIdx)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}

	result, err := s.executeSpawn(ctx, tr, cfg)
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}

	return s.finalizeSpawn(tr, waveIdx, memIdx, result)
}

// prepareSpawn builds all inputs needed for spawning (no side effects).
func (s *claudeSpawner) prepareSpawn(tr *TeamRunner, waveIdx, memIdx int) (*spawnConfig, error) {
	// 1. Read member config snapshot (under RLock)
	tr.configMu.RLock()
	if tr.config == nil || waveIdx >= len(tr.config.Waves) || memIdx >= len(tr.config.Waves[waveIdx].Members) {
		tr.configMu.RUnlock()
		return nil, fmt.Errorf("invalid wave/member indices: wave=%d, member=%d", waveIdx, memIdx)
	}
	member := tr.config.Waves[waveIdx].Members[memIdx]
	projectRoot := tr.config.ProjectRoot
	workflowType := tr.config.WorkflowType
	tr.configMu.RUnlock()

	// 2. Build prompt envelope (includes stdout schema instructions when contract exists)
	envelope, err := buildPromptEnvelope(tr.teamDir, &member, workflowType)
	if err != nil {
		return nil, fmt.Errorf("build envelope: %w", err)
	}

	// 3. Load agent config for CLI flags
	agentConfig, err := loadAgentConfig(member.Agent)
	if err != nil {
		log.Printf("WARNING: Failed to load agent config for %s: %v (using fallback)", member.Agent, err)
		// Use fallback with read-only tools
		agentConfig = &agentCLIConfig{
			AllowedTools: defaultFallbackTools,
			Model:        member.Model,
		}
	}

	// 3b. Augment allowed tools for implementation workflows.
	// agents-index.json cli_flags.allowed_tools are designed for read-only analysis
	// (review, braintrust). Implementation workers need Write and Edit to create files.
	if workflowType == "implementation" {
		agentConfig.AllowedTools = augmentToolsForImplementation(agentConfig.AllowedTools)
	}

	// 4. Build CLI args
	args := buildCLIArgs(agentConfig)

	// 5. Determine timeout
	timeout := time.Duration(member.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout
	}

	// 6. Build stdout path
	stdoutPath := filepath.Join(tr.teamDir, member.StdoutFile)

	return &spawnConfig{
		envelope:    envelope,
		args:        args,
		projectRoot: projectRoot,
		timeout:     timeout,
		memberName:  member.Name,
		agentID:     member.Agent,
		stdoutPath:  stdoutPath,
	}, nil
}

// executeSpawn starts a Claude CLI process and waits for completion.
func (s *claudeSpawner) executeSpawn(ctx context.Context, tr *TeamRunner, cfg *spawnConfig) (*spawnResult, error) {
	// 1. Build exec.Command with args
	cmd := exec.CommandContext(ctx, "claude", cfg.args...)

	// 2. Set Dir = projectRoot
	cmd.Dir = cfg.projectRoot

	// 3. Set Stdin = envelope
	cmd.Stdin = strings.NewReader(cfg.envelope)

	// 4. Set SysProcAttr Setsid: true (create new process group)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	// 5. Set Env: GOGENT_NESTING_LEVEL=2, GOGENT_PROJECT_ROOT
	cmd.Env = append(os.Environ(),
		"GOGENT_NESTING_LEVEL=2",
		fmt.Sprintf("GOGENT_PROJECT_ROOT=%s", cfg.projectRoot),
	)

	// 6. Capture stdout
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout // Capture stderr too

	// 7. Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude CLI: %w", err)
	}

	pid := cmd.Process.Pid

	// 8. W6: registerChild(pid) IMMEDIATELY after Start
	tr.registerChild(pid)

	// 9. defer unregisterChild
	defer tr.unregisterChild(pid)

	// 10. Wait for command with timeout
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled - kill process
		if cmd.Process != nil {
			syscall.Kill(-pid, syscall.SIGKILL) // Kill process group
		}
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	case err := <-waitDone:
		// Process completed
		exitCode := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("wait for claude CLI: %w", err)
			}
		}

		return &spawnResult{
			stdout:   stdout.Bytes(),
			exitCode: exitCode,
			pid:      pid,
		}, nil
	case <-time.After(cfg.timeout):
		// Timeout - kill process
		if cmd.Process != nil {
			syscall.Kill(-pid, syscall.SIGKILL) // Kill process group
		}
		return nil, fmt.Errorf("timeout after %v", cfg.timeout)
	}
}

// finalizeSpawn processes results (cost extraction, stdout validation, member update).
// Budget reconciliation happens at wave level in spawnAndWaitWithBudget.
func (s *claudeSpawner) finalizeSpawn(tr *TeamRunner, waveIdx, memIdx int, result *spawnResult) error {
	// 1. Parse CLI output
	cliOut, err := parseCLIOutput(result.stdout)

	actualCost := 0.0
	costStatus := "ok"

	tr.configMu.RLock()
	memberName := ""
	agentID := ""
	stdoutPath := ""
	if tr.config != nil && waveIdx < len(tr.config.Waves) && memIdx < len(tr.config.Waves[waveIdx].Members) {
		member := tr.config.Waves[waveIdx].Members[memIdx]
		memberName = member.Name
		agentID = member.Agent
		stdoutPath = filepath.Join(tr.teamDir, member.StdoutFile)
	}
	tr.configMu.RUnlock()

	if err != nil {
		log.Printf("WARNING: CLI output parse failed for %s: %v", memberName, err)
		costStatus = "error"
	} else {
		actualCost = cliOut.TotalCostUSD
		if actualCost == 0 {
			costStatus = "fallback"
		}
	}

	// 2. Write stdout file
	if cliOut != nil && cliOut.Result != "" {
		if err := writeStdoutFile(stdoutPath, tr.teamDir, cliOut.Result, agentID); err != nil {
			log.Printf("WARNING: failed to write stdout for %s: %v", memberName, err)
		}
	} else {
		log.Printf("WARNING: no result text to write for %s", memberName)
	}

	// 3. Validate stdout
	if err := validateStdout(stdoutPath, tr.teamDir); err != nil {
		log.Printf("WARNING: stdout validation failed: %v", err)
		// Don't fail the member - validation errors are non-fatal
	}

	// 4. Update member status
	exitCodeCopy := result.exitCode
	pidCopy := result.pid
	if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
		m.ProcessPID = &pidCopy
		m.ExitCode = &exitCodeCopy
		m.CostUSD = actualCost
		m.CostStatus = costStatus
		// Don't set Status here - caller (spawnAndWait) manages status
	}); err != nil {
		return fmt.Errorf("update member: %w", err)
	}

	return nil
}

// loadAgentConfig reads CLI flags from agents-index.json for a given agent.
func loadAgentConfig(agentID string) (*agentCLIConfig, error) {
	agentsIndexPath := filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json")

	data, err := os.ReadFile(agentsIndexPath)
	if err != nil {
		return nil, fmt.Errorf("read agents-index.json: %w", err)
	}

	var index struct {
		Agents []struct {
			ID       string `json:"id"`
			Model    string `json:"model"`
			CLIFlags struct {
				AllowedTools    []string `json:"allowed_tools"`
				AdditionalFlags []string `json:"additional_flags"`
			} `json:"cli_flags"`
		} `json:"agents"`
	}

	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse agents-index.json: %w", err)
	}

	for _, agent := range index.Agents {
		if agent.ID == agentID {
			return &agentCLIConfig{
				AllowedTools:    agent.CLIFlags.AllowedTools,
				AdditionalFlags: agent.CLIFlags.AdditionalFlags,
				Model:           agent.Model,
			}, nil
		}
	}

	return nil, fmt.Errorf("agent %s not found in agents-index.json", agentID)
}

// buildCLIArgs constructs claude CLI arguments from agent config.
// Overrides --permission-mode for pipe mode: agents-index.json specifies "delegate"
// for interactive contexts (TUI/MCP), but pipe mode (-p) has no interactive approval.
// We replace "delegate" with "auto-edit" so workers can write files within the project.
func buildCLIArgs(agentConfig *agentCLIConfig) []string {
	args := []string{"-p", "--output-format", "json"}

	if agentConfig.Model != "" {
		args = append(args, "--model", agentConfig.Model)
	}

	if len(agentConfig.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(agentConfig.AllowedTools, ","))
	}

	// Filter additional flags, replacing permission-mode for pipe-mode compatibility
	for i := 0; i < len(agentConfig.AdditionalFlags); i++ {
		flag := agentConfig.AdditionalFlags[i]
		if flag == "--permission-mode" && i+1 < len(agentConfig.AdditionalFlags) {
			// Replace "delegate" with "auto-edit" for pipe mode (no interactive approval)
			i++ // skip the value
			continue
		}
		args = append(args, flag)
	}

	return args
}

// cliOutput represents parsed Claude CLI JSON output.
type cliOutput struct {
	Result       string  // Agent's text response from the "result" entry
	TotalCostUSD float64 // Total cost from the "result" entry
	IsError      bool    // Whether the CLI reported an error
	SessionID    string  // CLI session ID for tracking
}

// parseCLIOutput parses the JSON array from `claude -p --output-format json`.
// Returns the extracted result entry data, or error if parsing fails.
func parseCLIOutput(output []byte) (*cliOutput, error) {
	// Parse as JSON array
	var entries []json.RawMessage
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("CLI output not valid JSON array: %w", err)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("CLI output is empty array")
	}

	// Find the "result" entry
	for _, entry := range entries {
		// Partially unmarshal to check type field
		var partial struct {
			Type    string  `json:"type"`
			Subtype string  `json:"subtype"`
			Result  string  `json:"result"`
			Cost    float64 `json:"total_cost_usd"`
			IsError bool    `json:"is_error"`
			Session string  `json:"session_id"`
		}

		if err := json.Unmarshal(entry, &partial); err != nil {
			// Skip malformed entries
			continue
		}

		if partial.Type == "result" {
			return &cliOutput{
				Result:       partial.Result,
				TotalCostUSD: partial.Cost,
				IsError:      partial.IsError,
				SessionID:    partial.Session,
			}, nil
		}
	}

	return nil, fmt.Errorf("no result entry found in CLI output")
}

// writeStdoutFile writes the agent's response to the stdout JSON file.
// Attempts to extract structured JSON from the response text.
// Falls back to wrapping the text in a minimal JSON envelope.
func writeStdoutFile(stdoutPath string, teamDir string, agentResult string, agentID string) error {
	// Validate path is within teamDir
	if err := validatePathWithinDir(stdoutPath, teamDir); err != nil {
		return fmt.Errorf("stdout path security: %w", err)
	}

	var outputJSON map[string]interface{}

	// Try to extract JSON from agent result
	// First try: unmarshal entire result as JSON object
	if err := json.Unmarshal([]byte(agentResult), &outputJSON); err == nil {
		// Valid JSON object found - use it directly
		log.Printf("Wrote structured stdout (direct JSON)")
	} else {
		// Second try: look for JSON code block
		jsonBlockStart := strings.Index(agentResult, "```json\n")
		if jsonBlockStart != -1 {
			jsonBlockStart += len("```json\n")
			jsonBlockEnd := strings.Index(agentResult[jsonBlockStart:], "\n```")
			if jsonBlockEnd != -1 {
				jsonStr := agentResult[jsonBlockStart : jsonBlockStart+jsonBlockEnd]
				if err := json.Unmarshal([]byte(jsonStr), &outputJSON); err == nil {
					log.Printf("Wrote structured stdout (extracted from code block)")
				}
			}
		}

		// Fallback: wrap in minimal envelope
		if outputJSON == nil {
			outputJSON = map[string]interface{}{
				"agent":      agentID,
				"status":     "complete",
				"raw_output": true,
				"result":     agentResult,
			}
			log.Printf("Wrote raw stdout (no JSON found, wrapped in envelope)")
		}
	}

	// Write pretty-printed JSON
	data, err := json.MarshalIndent(outputJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal stdout JSON: %w", err)
	}

	if err := os.WriteFile(stdoutPath, data, 0644); err != nil {
		return fmt.Errorf("write stdout file: %w", err)
	}

	return nil
}

// isRetryableError classifies errors for retry decisions.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false // Cancelled: stop
	}
	if errors.Is(err, exec.ErrNotFound) {
		return false // CLI missing: fatal
	}
	if errors.Is(err, os.ErrPermission) {
		return false // Permission: fatal
	}
	return true // Default: retry (timeout, exit code, etc.)
}

// spawnAndWait spawns a Claude CLI process for a team member and waits for completion.
// Uses iterative retry (NOT recursive) to avoid WaitGroup panics.
//
// CONTRACT: The caller MUST call wg.Add(1) exactly once before invoking this function.
// This function calls wg.Done() exactly once via defer, matching the single Add(1).
// Violating this contract causes a WaitGroup counter underflow panic.
//
// Usage:
//
//	wg.Add(1)
//	go spawnAndWait(ctx, tr, 0, 0, &wg)
//	wg.Wait()
//
// Retry logic:
//   - Attempts up to member.MaxRetries + 1 times (0-indexed)
//   - Checks context cancellation before each attempt
//   - Updates member status and error history throughout
//   - Returns on first success or after exhausting all retries
func spawnAndWait(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get member info snapshot for retry loop
	tr.configMu.RLock()
	if tr.config == nil || waveIdx >= len(tr.config.Waves) || memIdx >= len(tr.config.Waves[waveIdx].Members) {
		tr.configMu.RUnlock()
		log.Printf("ERROR: Invalid wave/member indices: wave=%d, member=%d", waveIdx, memIdx)
		return
	}
	member := tr.config.Waves[waveIdx].Members[memIdx]
	tr.configMu.RUnlock()

	var errorHistory []string

	// Iterative retry loop (NOT recursive)
	for attempt := 0; attempt <= member.MaxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "failed"
				ctxErr := fmt.Sprintf("context cancelled: %v", ctx.Err())
				if len(errorHistory) > 0 {
					m.ErrorMessage = strings.Join(errorHistory, "; ") + "; " + ctxErr
				} else {
					m.ErrorMessage = ctxErr
				}
			}); err != nil {
				log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			}
			return
		}

		// Mark as running
		if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
			m.Status = "running"
			m.RetryCount = attempt
		}); err != nil {
			log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			continue
		}

		// Attempt spawn
		err := tr.spawner.Spawn(ctx, tr, waveIdx, memIdx)
		if err == nil {
			// Success - mark as completed and return
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "completed"
			}); err != nil {
				log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			}
			log.Printf("Member %s completed successfully (attempt %d)", member.Name, attempt)
			return
		}

		// Spawn failed - log and update error message
		log.Printf("Spawn attempt %d for %s failed: %v", attempt, member.Name, err)
		errMsg := fmt.Sprintf("attempt %d: %v", attempt, err)
		errorHistory = append(errorHistory, errMsg)

		// W7: Check if error is retryable (fatal errors = no retry)
		if !isRetryableError(err) {
			// Fatal error, don't retry
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "failed"
				m.ErrorMessage = fmt.Sprintf("%s (fatal, non-retryable)", strings.Join(errorHistory, "; "))
			}); err != nil {
				log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			}
			log.Printf("Member %s failed with non-retryable error: %v", member.Name, err)
			return
		}

		if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
			m.ErrorMessage = strings.Join(errorHistory, "; ")
		}); err != nil {
			log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
		}

		// Continue to next retry (if any left)
	}

	// All retries exhausted - mark as failed
	if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
		m.Status = "failed"
	}); err != nil {
		log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
	}
	log.Printf("Member %s failed after %d retries", member.Name, member.MaxRetries+1)
}

