package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	defaultTimeoutMS = 300_000          // 5 minutes
	sigkillGraceMS   = 5_000            // 5s between SIGTERM and SIGKILL
	maxBufferBytes   = 10 * 1024 * 1024 // 10MB stdout buffer limit
	maxNestingDepth  = 10
)

// cliResult holds parsed output from a claude CLI subprocess.
type cliResult struct {
	Result       string
	TotalCostUSD float64
	NumTurns     int
	IsError      bool
	SessionID    string
	Truncated    bool
}

// validateNestingDepth checks GOGENT_NESTING_LEVEL and returns an error if the
// maximum nesting depth has been reached. Absent or non-numeric values are
// treated as level 0 (no error).
func validateNestingDepth() error {
	level := getCurrentNestingLevel()
	if level >= maxNestingDepth {
		return fmt.Errorf("maximum nesting depth (%d) exceeded: current level %d", maxNestingDepth, level)
	}
	return nil
}

// getCurrentNestingLevel reads GOGENT_NESTING_LEVEL and returns it as an int.
// Returns 0 if the variable is absent or cannot be parsed.
func getCurrentNestingLevel() int {
	val := os.Getenv("GOGENT_NESTING_LEVEL")
	if val == "" {
		return 0
	}
	level, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return level
}

// buildSpawnArgs constructs the claude CLI argument slice for spawning an agent.
// It always includes: -p --output-format json --model {m} --allowedTools {tools}
// and conditionally --max-budget-usd when input.MaxBudget > 0.
//
// NOTE: --timeout is NOT passed as a CLI flag (not supported by claude CLI).
// Timeout is managed by runSubprocess() via time.AfterFunc.
//
// Uses --output-format json (not stream-json) for one-shot -p invocations:
// json produces a single result object, simpler to parse, and doesn't require
// --verbose (which stream-json needs since claude CLI 2.1.81+).
// --permission-mode bypassPermissions is required because -p (print mode) has
// no interactive terminal to approve permissions.
func buildSpawnArgs(agent *routing.Agent, input SpawnAgentInput) []string {
	args := []string{"-p", "--output-format", "json", "--permission-mode", "bypassPermissions"}

	// Model: prefer explicit override, fall back to agent config.
	model := input.Model
	if model == "" {
		model = agent.Model
	}
	args = append(args, "--model", model)

	// MCP config: only for interactive agents when the config path is available.
	mcpConfigPath := os.Getenv("GOFORTRESS_MCP_CONFIG")
	hasMCP := agent.Interactive && mcpConfigPath != ""
	if hasMCP {
		args = append(args, "--mcp-config", mcpConfigPath)
	}

	// Allowed tools: prefer explicit override, fall back to agent config.
	tools := input.AllowedTools
	if len(tools) == 0 {
		tools = agent.GetAllowedTools()
	}
	// Merge MCP tool glob for interactive agents so spawned Claude can call
	// ask_user, confirm_action, spawn_agent, etc.
	if hasMCP {
		tools = append(tools, "mcp__gofortress-interactive__*")
	}
	if len(tools) > 0 {
		args = append(args, "--allowedTools", strings.Join(tools, ","))
	}

	// Optional cost ceiling.
	if input.MaxBudget > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.4f", input.MaxBudget))
	}

	return args
}

// buildSpawnEnv constructs the environment for the claude subprocess.
// It filters Claude Code session variables that would interfere with nested
// CLI invocations, then injects GOgent nesting metadata.
func buildSpawnEnv(nestingLevel int, agentID string) []string {
	env := filterEnv(os.Environ(),
		"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT",
		"GOGENT_NESTING_LEVEL", "GOGENT_PARENT_AGENT", "GOGENT_SPAWN_METHOD",
	)
	env = append(env,
		"GOGENT_NESTING_LEVEL="+strconv.Itoa(nestingLevel+1),
		"GOGENT_PARENT_AGENT="+agentID,
		"GOGENT_SPAWN_METHOD=mcp-cli",
	)
	return env
}

// filterEnv returns a copy of environ with entries whose key matches any of the
// given keys removed. Matching is prefix-based ("KEY=").
func filterEnv(environ []string, keys ...string) []string {
	filtered := make([]string, 0, len(environ))
	for _, e := range environ {
		skip := false
		for _, k := range keys {
			if strings.HasPrefix(e, k+"=") {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// parseCLIOutput parses Claude CLI output in either NDJSON (stream-json) or
// JSON array (legacy) format. Returns the extracted result entry, or an error
// if no result entry is found. Malformed JSON lines are skipped silently.
func parseCLIOutput(output []byte) (*cliResult, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("CLI output is empty")
	}

	var entries []json.RawMessage

	if trimmed[0] == '[' {
		// Legacy JSON array format.
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("CLI output not valid JSON array: %w", err)
		}
	} else {
		// NDJSON format (stream-json): one JSON object per line.
		scanner := bufio.NewScanner(bytes.NewReader(trimmed))
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB max line
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}
			entries = append(entries, json.RawMessage(append([]byte{}, line...)))
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning NDJSON output: %w", err)
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("CLI output has no entries")
	}

	// Find the "result" entry.
	for _, entry := range entries {
		var partial struct {
			Type     string  `json:"type"`
			Result   string  `json:"result"`
			Cost     float64 `json:"total_cost_usd"` // CRITICAL: field is total_cost_usd, NOT cost_usd
			IsError  bool    `json:"is_error"`
			Session  string  `json:"session_id"`
			NumTurns int     `json:"num_turns"`
		}

		if err := json.Unmarshal(entry, &partial); err != nil {
			// Skip malformed lines.
			continue
		}

		if partial.Type == "result" {
			return &cliResult{
				Result:       partial.Result,
				TotalCostUSD: partial.Cost,
				NumTurns:     partial.NumTurns,
				IsError:      partial.IsError,
				SessionID:    partial.Session,
			}, nil
		}
	}

	return nil, fmt.Errorf("no result entry found in CLI output")
}

// runSubprocess starts a claude CLI subprocess, applies a SIGTERM/SIGKILL
// timeout lifecycle, and returns the parsed result. The augmentedPrompt is
// written to the subprocess stdin.
func runSubprocess(ctx context.Context, agent *routing.Agent, input SpawnAgentInput, augmentedPrompt string, agentID string) (*cliResult, error) {
	args := buildSpawnArgs(agent, input)
	nestingLevel := getCurrentNestingLevel()
	env := buildSpawnEnv(nestingLevel, agentID)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(augmentedPrompt)

	// Inherit CWD from the GOGENT_CWD env var if set. This controls the scope
	// of CC's hardcoded write restrictions (CC can write to CWD and subdirs).
	if cwdOverride := os.Getenv("GOGENT_CWD"); cwdOverride != "" {
		cmd.Dir = cwdOverride
	}

	// Create a new process group so that SIGTERM/SIGKILL targets the entire
	// tree (claude parent + any MCP server children), not just the direct child.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	pid := cmd.Process.Pid

	// Read stdout and stderr concurrently, respecting the buffer cap.
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
		truncated bool
		wg        sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stdoutPipe.Read(buf)
			if n > 0 {
				if !truncated && stdoutBuf.Len() < maxBufferBytes {
					stdoutBuf.Write(buf[:n])
					if stdoutBuf.Len() >= maxBufferBytes {
						truncated = true
						stdoutBuf.WriteString("\n[OUTPUT TRUNCATED]")
					}
				}
			}
			if readErr != nil {
				break
			}
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(&stderrBuf, stderrPipe) //nolint:errcheck
	}()

	// killOnce guards against double-kill races when context cancellation and
	// the timeout timer fire simultaneously.
	var killOnce sync.Once
	killProcess := func(sig syscall.Signal) {
		killOnce.Do(func() {
			if cmd.Process != nil {
				slog.Warn("spawn_agent: sending signal to process group",
					"agent", input.Agent, "signal", sig, "pid", pid)
				if err := syscall.Kill(-pid, sig); err != nil {
					slog.Warn("spawn_agent: failed to send signal",
						"agent", input.Agent, "signal", sig, "pid", pid, "err", err)
				}
			}
		})
	}

	// Timeout: SIGTERM first, then SIGKILL after the grace period.
	timeoutMS := defaultTimeoutMS
	if input.Timeout > 0 {
		timeoutMS = input.Timeout
	}
	timer := time.AfterFunc(time.Duration(timeoutMS)*time.Millisecond, func() {
		killProcess(syscall.SIGTERM)
		time.AfterFunc(time.Duration(sigkillGraceMS)*time.Millisecond, func() {
			if cmd.Process != nil {
				slog.Warn("spawn_agent: kill grace expired, sending SIGKILL", "agent", input.Agent)
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
			}
		})
	})

	// done is closed when the subprocess I/O goroutines have drained and
	// wg.Wait() returns. It lets the ctx-watcher below exit cleanly.
	done := make(chan struct{})

	// Context-cancellation watcher: exec.CommandContext kills only the direct
	// process (SIGKILL to the PID) when ctx expires. Because Setsid is set,
	// claude's child processes (MCP servers, etc.) are in the same process
	// group but are NOT killed by the direct-PID SIGKILL. Those children hold
	// the stdout/stderr pipes open, blocking wg.Wait() until the 5-minute
	// timer fires. Sending SIGTERM to the process group (-pid) closes all
	// descendants promptly.
	go func() {
		select {
		case <-ctx.Done():
			killProcess(syscall.SIGTERM)
			time.AfterFunc(time.Duration(sigkillGraceMS)*time.Millisecond, func() {
				select {
				case <-done:
					// process group already exited
				default:
					if cmd.Process != nil {
						syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
					}
				}
			})
		case <-done:
		}
	}()

	wg.Wait()
	close(done)
	timer.Stop()

	waitErr := cmd.Wait()

	// Parse the NDJSON output. Fall back to raw stdout on parse failure.
	result, parseErr := parseCLIOutput(stdoutBuf.Bytes())
	if parseErr != nil {
		slog.Warn("spawn_agent CLI output parse failed", "agent", input.Agent, "err", parseErr)
		result = &cliResult{Result: stdoutBuf.String()}
	}
	result.Truncated = truncated

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			slog.Warn("spawn_agent subprocess non-zero exit",
				"agent", input.Agent,
				"code", exitErr.ExitCode(),
				"stderr", stderrBuf.String())
			if result.Result == "" {
				result.Result = stderrBuf.String()
			}
			return result, fmt.Errorf("claude exited %d: %s", exitErr.ExitCode(), stderrBuf.String())
		}
		return result, fmt.Errorf("wait claude: %w", waitErr)
	}

	return result, nil
}
