package mcp

// spawner.go implements the subprocess management for spawn_agent
// within the TUI's MCP server context (adds UDS notifications for agent tracking).

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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/cli"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
	routing "github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

// acWrittenToRe matches "<filename> written to <path>" in criterion text.
var acWrittenToRe = regexp.MustCompile(`(?i)(\S+)\s+written\s+to\s+(\S+)`)

// acWrittenToDirRe matches "written to <dir/>" (trailing slash) without a
// leading filename token — used as a fallback for directory-listing checks.
var acWrittenToDirRe = regexp.MustCompile(`(?i)written\s+to\s+(\S+/)`)

const (
	defaultTimeoutMS = 900_000          // 15 minutes
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

type cliOutputCollector struct {
	stdoutBuf   bytes.Buffer
	maxBytes    int
	truncated   bool
	resultEntry *cliResult
}

func newCLIOutputCollector(maxBytes int) *cliOutputCollector {
	return &cliOutputCollector{maxBytes: maxBytes}
}

func (c *cliOutputCollector) appendLine(line []byte) {
	if c.resultEntry == nil {
		if result, ok := parseCLIResultEntry(line); ok {
			c.resultEntry = result
		}
	}

	if c.truncated || c.stdoutBuf.Len() >= c.maxBytes {
		return
	}

	c.stdoutBuf.Write(line)
	c.stdoutBuf.WriteByte('\n')
	if c.stdoutBuf.Len() >= c.maxBytes {
		c.truncated = true
		c.stdoutBuf.WriteString("[OUTPUT TRUNCATED]")
	}
}

func (c *cliOutputCollector) bytes() []byte {
	return c.stdoutBuf.Bytes()
}

func (c *cliOutputCollector) fallbackResult() *cliResult {
	if c.resultEntry != nil {
		cloned := *c.resultEntry
		return &cloned
	}
	return nil
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

func parseCLIResultEntry(entry []byte) (*cliResult, bool) {
	var partial struct {
		Type     string  `json:"type"`
		Result   string  `json:"result"`
		Cost     float64 `json:"total_cost_usd"`
		IsError  bool    `json:"is_error"`
		Session  string  `json:"session_id"`
		NumTurns int     `json:"num_turns"`
	}

	if err := json.Unmarshal(entry, &partial); err != nil || partial.Type != "result" {
		return nil, false
	}

	return &cliResult{
		Result:       partial.Result,
		TotalCostUSD: partial.Cost,
		NumTurns:     partial.NumTurns,
		IsError:      partial.IsError,
		SessionID:    partial.Session,
	}, true
}

// runSubprocess starts a claude CLI subprocess, applies a SIGTERM/SIGKILL
// timeout lifecycle, and returns the parsed result. When uds is non-nil,
// live tool_use events are parsed from the NDJSON stream and forwarded as
// AgentActivity IPC notifications for real-time TUI display.
func runSubprocess(ctx context.Context, agent *routing.Agent, input SpawnAgentInput, augmentedPrompt string, agentID string, uds *UDSClient) (*cliResult, error) {
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

	// Notify TUI with the subprocess PID for interrupt support.
	// This is a second "running" notification (the first is sent by
	// handleSpawnAgent before runSubprocess is called) but this one
	// carries the PID which is only available after cmd.Start().
	if uds != nil {
		uds.notify(TypeAgentUpdate, AgentUpdatePayload{
			AgentID: agentID,
			Status:  "running",
			PID:     pid,
		})
	}

	var (
		outputCollector = newCLIOutputCollector(maxBufferBytes)
		stderrBuf       bytes.Buffer
		wg              sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		// acState tracks the goroutine-local AC state updated on each TodoWrite.
		// Initialised from input.AcceptanceCriteria so that MatchTodosToAC can
		// compare against the injected criteria text from the first TodoWrite call.
		acState := make([]state.AcceptanceCriterion, len(input.AcceptanceCriteria))
		for idx, text := range input.AcceptanceCriteria {
			acState[idx] = state.AcceptanceCriterion{Text: text}
		}
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 512*1024), maxBufferBytes)
		for scanner.Scan() {
			line := scanner.Bytes()
			outputCollector.appendLine(line)
			// Live NDJSON parsing: extract tool_use events and forward
			// as AgentActivity IPC notifications for the TUI detail panel.
			if uds != nil {
				event, parseErr := cli.ParseCLIEvent(line)
				if parseErr != nil {
					continue
				}
				if ae, ok := event.(cli.AssistantEvent); ok {
					for _, block := range ae.Message.Content {
						if block.Type == "tool_use" && block.Name != "" {
							activities := cli.ExtractToolActivities(block, "")
							for _, act := range activities {
								uds.notify(TypeAgentActivity, AgentActivityPayload{
									AgentID: agentID,
									Tool:    block.Name,
									Target:  act.Target,
									Preview: act.Preview,
								})
							}
							// Detect TodoWrite and forward full todo state to TUI.
							// Also write the AC sidecar file from this goroutine to
							// avoid a race with the endstate hook (M-3).
							if block.Name == "TodoWrite" {
								todos := parseTodoWriteInput(block.Input)
								if len(todos) > 0 {
									uds.notify(TypeAgentTodoUpdate, AgentTodoUpdatePayload{
										AgentID: agentID,
										Todos:   todos,
									})
									acState = writeACSidecar(agentID, todos, acState)
								}
							}
						}
					}
				}
			}
		}
		// Post-completion AC verification: programmatic filesystem checks for
		// any criteria that the agent's TodoWrite output did not mark completed.
		// Runs after the subprocess has exited (scanner returns EOF), before
		// wg.Done() fires, so no extra synchronisation is needed.
		if len(acState) > 0 {
			acState = verifyACDeliverables(agentID, acState, uds)
			finalTodos := make([]TodoItem, len(acState))
			for i, ac := range acState {
				status := "pending"
				if ac.Completed {
					status = "completed"
				}
				finalTodos[i] = TodoItem{Content: ac.Text, Status: status}
			}
			writeACSidecar(agentID, finalTodos, acState)
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

	result, parseErr := parseCLIOutput(outputCollector.bytes())
	if parseErr != nil {
		if fallback := outputCollector.fallbackResult(); fallback != nil {
			result = fallback
		} else {
			slog.Warn("spawn_agent CLI output parse failed", "agent", input.Agent, "err", parseErr)
			result = &cliResult{Result: string(outputCollector.bytes())}
		}
	}
	result.Truncated = outputCollector.truncated

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

// parseTodoWriteInput parses the JSON input of a TodoWrite tool_use block and
// returns the list of todo items. Returns nil for empty or malformed input
// without panicking (M-2).
func parseTodoWriteInput(input json.RawMessage) []TodoItem {
	if len(input) == 0 {
		return nil
	}
	var payload struct {
		Todos []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"todos"`
	}
	if err := json.Unmarshal(input, &payload); err != nil {
		slog.Warn("parseTodoWriteInput: malformed input", "err", err)
		return nil
	}
	if len(payload.Todos) == 0 {
		return nil
	}
	items := make([]TodoItem, len(payload.Todos))
	for i, t := range payload.Todos {
		items[i] = TodoItem{Content: t.Content, Status: t.Status}
	}
	return items
}

// writeACSidecar matches todos against the current AC state using
// state.MatchTodosToAC, writes the updated state to
// SESSION_DIR/ac/{agentID}.json, and returns the updated AC slice.
//
// Called from the NDJSON scanner goroutine — before wg.Wait() and before
// the endstate hook fires — which eliminates the race condition (M-3).
func writeACSidecar(agentID string, todos []TodoItem, acState []state.AcceptanceCriterion) []state.AcceptanceCriterion {
	updates := make([]state.TodoUpdate, len(todos))
	for i, t := range todos {
		updates[i] = state.TodoUpdate{Content: t.Content, Status: t.Status}
	}
	updated := state.MatchTodosToAC(acState, updates)

	sessionDir := os.Getenv("GOGENT_SESSION_DIR")
	if sessionDir == "" {
		slog.Warn("writeACSidecar: GOGENT_SESSION_DIR not set, skipping sidecar write",
			"agentID", agentID)
		return updated
	}

	acDir := filepath.Join(sessionDir, "ac")
	if err := os.MkdirAll(acDir, 0o755); err != nil {
		slog.Warn("writeACSidecar: failed to create ac dir", "err", err, "dir", acDir)
		return updated
	}

	data, err := json.Marshal(updated)
	if err != nil {
		slog.Warn("writeACSidecar: failed to marshal AC state", "err", err, "agentID", agentID)
		return updated
	}

	sidecarPath := filepath.Join(acDir, agentID+".json")
	if err := os.WriteFile(sidecarPath, data, 0o644); err != nil {
		slog.Warn("writeACSidecar: failed to write sidecar", "err", err, "path", sidecarPath)
	}

	return updated
}

// verifyACDeliverables performs post-completion programmatic verification of
// unmet acceptance criteria by checking for deliverable files on disk. It does
// NOT spawn any LLM calls — purely filesystem checks.
//
// For each criterion that is not yet completed, it looks for "X written to Y"
// patterns and calls os.Stat to confirm the deliverable exists. When at least
// one unmet criterion is newly satisfied, a final AgentTodoUpdate UDS
// notification is sent so the TUI reflects the updated state.
//
// This is called at the end of the stdout scanner goroutine (after the
// subprocess exits) so that acState is in-scope without extra synchronisation.
func verifyACDeliverables(agentID string, acState []state.AcceptanceCriterion, uds *UDSClient) []state.AcceptanceCriterion {
	if len(acState) == 0 {
		return acState
	}

	updated := make([]state.AcceptanceCriterion, len(acState))
	copy(updated, acState)
	changed := false

	for i, ac := range updated {
		if ac.Completed {
			continue
		}
		if acFileExists(ac.Text) {
			updated[i].Completed = true
			changed = true
			slog.Info("verifyACDeliverables: criterion satisfied by file check",
				"agentID", agentID, "criterion", ac.Text)
		}
	}

	if !changed || uds == nil {
		return updated
	}

	// Send final UDS notification so the TUI reflects the verified state.
	todos := make([]TodoItem, len(updated))
	for i, ac := range updated {
		status := "pending"
		if ac.Completed {
			status = "completed"
		}
		todos[i] = TodoItem{Content: ac.Text, Status: status}
	}
	uds.notify(TypeAgentTodoUpdate, AgentTodoUpdatePayload{
		AgentID: agentID,
		Todos:   todos,
	})

	return updated
}

// acFileExists checks whether a criterion's mentioned deliverable exists on
// disk. It parses patterns such as:
//   - "foo.json written to dir/"        → checks dir/foo.json
//   - "foo.json written to dir/foo.json"→ checks dir/foo.json as a file
//   - "written to dir/"                 → checks that dir/ has at least one file
func acFileExists(criterionText string) bool {
	if m := acWrittenToRe.FindStringSubmatch(criterionText); len(m) == 3 {
		filename := strings.TrimRight(m[1], "/,.")
		target := strings.TrimRight(m[2], "/,.")

		// Attempt <target>/<filename>.
		if _, err := os.Stat(filepath.Join(target, filename)); err == nil {
			return true
		}
		// Attempt target as the full file path itself (no trailing dir component).
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			return true
		}
		// If target is a directory, any non-directory entry satisfies the check.
		if entries, err := os.ReadDir(target); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					return true
				}
			}
		}
	}

	// Fallback: "written to <dir/>" without a leading filename token.
	if m := acWrittenToDirRe.FindStringSubmatch(criterionText); len(m) == 2 {
		dirPath := strings.TrimRight(m[1], "/,.")
		if entries, err := os.ReadDir(dirPath); err == nil {
			for _, e := range entries {
				if !e.IsDir() {
					return true
				}
			}
		}
	}

	return false
}
