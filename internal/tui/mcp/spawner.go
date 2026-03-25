package mcp

// spawner.go implements the subprocess management for spawn_agent.
// Ported from cmd/gofortress-mcp-standalone/spawner.go to work within the
// TUI's MCP server context (adds UDS notifications for agent tracking).

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
// maximum nesting depth has been reached.
func validateNestingDepth() error {
	level := getCurrentNestingLevel()
	if level >= maxNestingDepth {
		return fmt.Errorf("maximum nesting depth (%d) exceeded: current level %d", maxNestingDepth, level)
	}
	return nil
}

// getCurrentNestingLevel reads GOGENT_NESTING_LEVEL and returns it as an int.
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

// buildSpawnEnv constructs the environment for the claude subprocess.
func buildSpawnEnv(nestingLevel int, agentID string) []string {
	env := filterEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT")
	env = append(env,
		"GOGENT_NESTING_LEVEL="+strconv.Itoa(nestingLevel+1),
		"GOGENT_PARENT_AGENT="+agentID,
		"GOGENT_SPAWN_METHOD=mcp-cli",
	)
	return env
}

// filterEnv removes env entries whose key matches any of the given prefixes.
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

// parseCLIOutput parses Claude CLI NDJSON output and extracts the result entry.
func parseCLIOutput(output []byte) (*cliResult, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("CLI output is empty")
	}

	var entries []json.RawMessage

	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("CLI output not valid JSON array: %w", err)
		}
	} else {
		scanner := bufio.NewScanner(bytes.NewReader(trimmed))
		scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
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

	for _, entry := range entries {
		var partial struct {
			Type     string  `json:"type"`
			Result   string  `json:"result"`
			Cost     float64 `json:"total_cost_usd"`
			IsError  bool    `json:"is_error"`
			Session  string  `json:"session_id"`
			NumTurns int     `json:"num_turns"`
		}

		if err := json.Unmarshal(entry, &partial); err != nil {
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
// timeout lifecycle, and returns the parsed result.
func runSubprocess(ctx context.Context, agent *routing.Agent, input SpawnAgentInput, augmentedPrompt string, agentID string) (*cliResult, error) {
	args := buildSpawnArgs(agent, input)
	nestingLevel := getCurrentNestingLevel()
	env := buildSpawnEnv(nestingLevel, agentID)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(augmentedPrompt)

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

	wg.Wait()
	timer.Stop()

	waitErr := cmd.Wait()

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
