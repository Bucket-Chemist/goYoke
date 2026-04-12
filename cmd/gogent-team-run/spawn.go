package main

import (
	"bufio"
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

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/session"
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
	waveIdx      int
	memIdx       int
	teamAgentID  string
}

// spawnResult holds the outputs from a completed CLI process.
type spawnResult struct {
	stdout      []byte
	exitCode    int
	pid         int
	teamAgentID string
}

// agentCLIConfig holds CLI flags and context requirements from agents-index.json.
type agentCLIConfig struct {
	AllowedTools        []string
	AdditionalFlags     []string
	Model               string
	ContextRequirements *routing.ContextRequirements
	FormalSchema        string // Minified JSON Schema for --json-schema constrained decoding
}

// Default fallback tools (W4: least-privilege READ-ONLY when agents-index.json unavailable)
var defaultFallbackTools = []string{"Read", "Glob", "Grep"}

// implementationTools are required for workers that create/modify files.
var implementationTools = []string{"Write", "Edit"}

// Health monitoring defaults.
// healthCheckInterval polls process health periodically to detect stalls.
// 30s balances responsiveness with avoiding excessive config writes.
var healthCheckInterval = 30 * time.Second

// stallWarningThreshold is the duration of no output before flagging stall.
// 90s allows for model thinking time while catching true stalls.
var stallWarningThreshold = 90 * time.Second

// progressTracker wraps a bytes.Buffer and timestamps every Write call.
// Used to detect agent activity — if lastActivity stops updating, agent may be stalled.
type progressTracker struct {
	buf           bytes.Buffer
	lastActivity  time.Time
	bytesReceived int64
	mu            sync.Mutex
}

func newProgressTracker() *progressTracker {
	return &progressTracker{
		lastActivity: time.Now(),
	}
}

func (pt *progressTracker) Write(p []byte) (int, error) {
	pt.mu.Lock()
	pt.lastActivity = time.Now()
	pt.bytesReceived += int64(len(p))
	pt.mu.Unlock()
	return pt.buf.Write(p)
}

func (pt *progressTracker) LastActivity() time.Time {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.lastActivity
}

func (pt *progressTracker) BytesReceived() int64 {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.bytesReceived
}

func (pt *progressTracker) Bytes() []byte {
	return pt.buf.Bytes()
}

// WatchFile polls a file and updates lastActivity/bytesReceived when it grows.
// Used when the child process writes directly to a file instead of through tracker.Write.
// Stops when ctx is cancelled.
func (pt *progressTracker) WatchFile(ctx context.Context, path string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stat, err := os.Stat(path)
			if err != nil {
				continue
			}
			size := stat.Size()
			pt.mu.Lock()
			if size > pt.bytesReceived {
				pt.lastActivity = time.Now()
				pt.bytesReceived = size
			}
			pt.mu.Unlock()
		}
	}
}

// workflowTimeout returns the default timeout for a workflow type.
// Per-member TimeoutMs override takes precedence when set in config.json.
func workflowTimeout(workflowType string) time.Duration {
	switch workflowType {
	case "braintrust":
		return 30 * time.Minute
	case "implementation":
		return 10 * time.Minute
	case "review":
		return 5 * time.Minute
	default:
		return 15 * time.Minute // Safety net — catches Mozart bugs that leave workflow_type empty
	}
}

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
		return fmt.Errorf("prepare spawn (wave=%d, member=%d): %w", waveIdx, memIdx, err)
	}

	result, err := s.executeSpawn(ctx, tr, cfg)
	if err != nil {
		return fmt.Errorf("execute %s (agent=%s): %w", cfg.memberName, cfg.agentID, err)
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

	// 1b. Validate required member fields
	if member.Agent == "" {
		return nil, fmt.Errorf("member %q has empty agent ID", member.Name)
	}
	if member.Model != "" {
		switch member.Model {
		case "haiku", "sonnet", "opus":
			// Valid model
		default:
			return nil, fmt.Errorf("member %q has invalid model %q (expected haiku|sonnet|opus)", member.Name, member.Model)
		}
	}

	// 2. Load agent config for CLI flags (moved up — needed before envelope)
	agentConfig, err := loadAgentConfig(member.Agent)
	if err != nil {
		log.Printf("WARNING: Failed to load agent config for %s: %v (using fallback)", member.Agent, err)
		agentConfig = &agentCLIConfig{
			AllowedTools: defaultFallbackTools,
			Model:        member.Model,
		}
	}

	// 2b. Resolve formal schema for constrained decoding
	formalSchema, schemaFound := resolveFormalSchema(member.Agent)
	if schemaFound {
		agentConfig.FormalSchema = formalSchema
	}

	// 3. Build prompt envelope (schema_id vs $schema instruction depends on constrained decoding)
	envelope, err := buildPromptEnvelope(tr.teamDir, &member, workflowType, schemaFound)
	if err != nil {
		return nil, fmt.Errorf("build envelope: %w", err)
	}

	// 3b. Inject agent identity + rules + conventions into envelope.
	// This gives team-run agents the same context Task()-spawned agents get via gogent-validate.
	if agentConfig.ContextRequirements != nil || member.Agent != "" {
		augmented, err := routing.BuildFullAgentContext(
			member.Agent,
			agentConfig.ContextRequirements,
			nil, // no taskFiles — file context is in stdin JSON
			envelope,
		)
		if err != nil {
			log.Printf("WARNING: Failed to inject agent context for %s: %v", member.Agent, err)
		} else {
			envelope = augmented
		}
	}

	// 3c. Augment allowed tools for implementation workflows.
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
		timeout = workflowTimeout(workflowType)
	}

	// 6. Build stdout path
	stdoutPath := filepath.Join(tr.teamDir, member.StdoutFile)

	teamName := filepath.Base(tr.teamDir)
	return &spawnConfig{
		envelope:    envelope,
		args:        args,
		projectRoot: projectRoot,
		timeout:     timeout,
		memberName:  member.Name,
		agentID:     member.Agent,
		stdoutPath:  stdoutPath,
		waveIdx:     waveIdx,
		memIdx:      memIdx,
		teamAgentID: fmt.Sprintf("team:%s:%s", teamName, member.Name),
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

	// 5. Set Env: GOGENT_NESTING_LEVEL=2, GOGENT_PROJECT_ROOT, GOGENT_SESSION_DIR
	// Filter out Claude Code session env vars that block nested CLI invocations.
	// team-run spawns independent CLI processes that must not be treated as nested sessions.
	cmd.Env = append(filterEnv(os.Environ(), "CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT"),
		"GOGENT_NESTING_LEVEL=2",
		fmt.Sprintf("GOGENT_PROJECT_ROOT=%s", cfg.projectRoot),
	)
	// Add session dir if available (read from current-session marker file)
	if sessionDir, err := session.ReadCurrentSession(cfg.projectRoot); err == nil && sessionDir != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOGENT_SESSION_DIR=%s", sessionDir))
	} else if err != nil {
		log.Printf("INFO: Could not read current-session marker: %v (child will inherit parent session)", err)
	}

	// 6. Capture output by writing child stdout directly to the stream file.
	// Direct-to-file avoids routing bytes through the runner process: if the runner
	// dies (e.g. SIGKILL from an MCP timeout), the child continues writing and the
	// data survives on disk. O_SYNC ensures kernel flushes to disk on every write.
	tracker := newProgressTracker()
	streamPath := filepath.Join(tr.teamDir, fmt.Sprintf("stream_%s.ndjson", cfg.agentID))
	streamFile, err := os.OpenFile(streamPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_SYNC, 0644)
	if err != nil {
		log.Printf("WARNING: Could not create stream file %s: %v (falling back to in-memory capture)", streamPath, err)
		// Fallback: route through tracker so parseCLIOutput still has bytes
		cmd.Stdout = tracker
		cmd.Stderr = tracker
	} else {
		// Child writes directly to file; tracker monitors file growth for health checks
		cmd.Stdout = streamFile
		cmd.Stderr = streamFile
	}
	defer func() {
		if streamFile != nil {
			streamFile.Close()
		}
	}()

	// 7. Send agent_register before starting so the TUI knows about this agent
	// before it produces any output.
	if tr.uds != nil && !tr.uds.isNoop() {
		teamName := filepath.Base(tr.teamDir)
		tr.uds.notify(typeAgentRegister, agentRegisterPayload{
			AgentID:     cfg.teamAgentID,
			AgentType:   cfg.agentID,
			ParentID:    "team:" + teamName,
			Description: cfg.memberName,
		})
	}

	// 8. Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude CLI: %w", err)
	}

	pid := cmd.Process.Pid

	// 9. Send agent_update(running, PID) now that we have the PID.
	if tr.uds != nil && !tr.uds.isNoop() {
		tr.uds.notify(typeAgentUpdate, agentUpdatePayload{
			AgentID: cfg.teamAgentID,
			Status:  "running",
			PID:     pid,
		})
	}

	// 11. W6: registerChild(pid) IMMEDIATELY after Start
	tr.registerChild(pid)

	// 12. defer unregisterChild
	defer tr.unregisterChild(pid)

	// Start health monitor (shadow mode — observe only, no kills).
	// When child writes directly to streamFile, watch the file's size for activity.
	// Fallback (streamFile == nil): tracker.Write receives bytes directly.
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()
	if streamFile != nil {
		go tracker.WatchFile(monitorCtx, streamPath, 500*time.Millisecond)
	}
	// Tail-follow the stream file for UDS notifications (tool_use events → TUI).
	// Runs alongside WatchFile with its own independent FD — no conflict.
	if streamFile != nil && tr.uds != nil && !tr.uds.isNoop() {
		go tailStreamForUDS(monitorCtx, streamPath, tr.uds, cfg.teamAgentID)
	}
	go startHealthMonitor(monitorCtx, tr, cfg.waveIdx, cfg.memIdx, tracker, healthCheckInterval)

	// 10. Wait for command with timeout
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	// Guard against double-kill when context cancellation and timeout race
	var killOnce sync.Once
	doKill := func(sig syscall.Signal, reason string) {
		killOnce.Do(func() {
			if cmd.Process != nil {
				log.Printf("[%s] Sending %v to process group %d", reason, sig, -pid)
				if err := syscall.Kill(-pid, sig); err != nil {
					log.Printf("WARNING: Failed to send %v to process group %d: %v", sig, -pid, err)
				}
			}
		})
	}

	select {
	case <-ctx.Done():
		// Context cancelled - kill process group
		doKill(syscall.SIGKILL, "CANCELLED")
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

		// Read stdout from the stream file (child wrote directly to it).
		// Fallback to tracker bytes if streamFile was never opened.
		var stdoutBytes []byte
		if streamFile != nil {
			stdoutBytes, _ = os.ReadFile(streamPath)
		} else {
			stdoutBytes = tracker.Bytes()
		}

		return &spawnResult{
			stdout:      stdoutBytes,
			exitCode:    exitCode,
			pid:         pid,
			teamAgentID: cfg.teamAgentID,
		}, nil
	case <-time.After(cfg.timeout):
		// Timeout - attempt graceful shutdown with SIGTERM first
		log.Printf("[TIMEOUT] Sending SIGTERM to process group %d after %v", -pid, cfg.timeout)
		if cmd.Process != nil {
			if err := syscall.Kill(-pid, syscall.SIGTERM); err != nil {
				log.Printf("WARNING: Failed to send SIGTERM to process group %d: %v", -pid, err)
			}
		}
		select {
		case <-waitDone:
			// Process exited gracefully after SIGTERM
			log.Printf("[TIMEOUT] Process group %d exited gracefully after SIGTERM", -pid)
		case <-time.After(sigTermGracePeriod):
			// Grace period expired, escalate to SIGKILL (guarded by killOnce)
			doKill(syscall.SIGKILL, "TIMEOUT")
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

	// 5. Send completion notification via UDS.
	if tr.uds != nil && !tr.uds.isNoop() && result.teamAgentID != "" {
		status := "complete"
		if result.exitCode != 0 {
			status = "error"
		}
		tr.uds.notify(typeAgentUpdate, agentUpdatePayload{
			AgentID: result.teamAgentID,
			Status:  status,
		})
	}

	return nil
}

// loadAgentConfig reads CLI flags from agents-index.json for a given agent.
func loadAgentConfig(agentID string) (*agentCLIConfig, error) {
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		return nil, fmt.Errorf("resolve config dir: %w", err)
	}
	agentsIndexPath := filepath.Join(configDir, "agents", "agents-index.json")

	data, err := os.ReadFile(agentsIndexPath)
	if err != nil {
		return nil, fmt.Errorf("read agents-index.json: %w", err)
	}

	var index struct {
		Agents []struct {
			ID                  string                       `json:"id"`
			Model               string                       `json:"model"`
			CLIFlags            struct {
				AllowedTools    []string `json:"allowed_tools"`
				AdditionalFlags []string `json:"additional_flags"`
			} `json:"cli_flags"`
			ContextRequirements *routing.ContextRequirements `json:"context_requirements,omitempty"`
		} `json:"agents"`
	}

	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse agents-index.json: %w", err)
	}

	for _, agent := range index.Agents {
		if agent.ID == agentID {
			return &agentCLIConfig{
				AllowedTools:        agent.CLIFlags.AllowedTools,
				AdditionalFlags:     agent.CLIFlags.AdditionalFlags,
				Model:               agent.Model,
				ContextRequirements: agent.ContextRequirements,
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
	args := []string{"-p", "--verbose", "--output-format", "stream-json"}

	if agentConfig.Model != "" {
		model := agentConfig.Model
		// Inherit 1M context from env vars — if the user's default model
		// includes [1m], propagate it to agents (haiku doesn't support 1M).
		if !strings.Contains(model, "[1m]") && !strings.Contains(model, "haiku") {
			envVar := "ANTHROPIC_DEFAULT_" + strings.ToUpper(model) + "_MODEL"
			if val := os.Getenv(envVar); strings.Contains(val, "[1m]") {
				model += "[1m]"
			}
		}
		args = append(args, "--model", model)
	}

	if agentConfig.FormalSchema != "" {
		args = append(args, "--json-schema", agentConfig.FormalSchema)
	}

	if len(agentConfig.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(agentConfig.AllowedTools, ","))
	}

	// Block built-in tools that cannot work in team-run's pipe mode.
	// Task: spawns unconstrained subprocesses bypassing the fortress architecture.
	// AskUserQuestion: requires an interactive terminal — hangs or fails in -p mode.
	// A future "convoy listener" daemon could bridge team-run to the TUI's UDS
	// socket and provide MCP equivalents (ask_user, spawn_agent) instead.
	args = append(args, "--disallowedTools", "Task,AskUserQuestion")

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

// parseCLIOutput parses Claude CLI output in either NDJSON (stream-json) or
// JSON array (legacy json) format. Returns the extracted result entry data.
func parseCLIOutput(output []byte) (*cliOutput, error) {
	var entries []json.RawMessage

	// Try NDJSON first (stream-json format: one JSON object per line)
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("CLI output is empty")
	}

	if trimmed[0] == '[' {
		// Legacy JSON array format
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil, fmt.Errorf("CLI output not valid JSON array: %w", err)
		}
	} else {
		// NDJSON format (stream-json): one JSON object per line
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

	// Find the "result" entry
	for _, entry := range entries {
		var partial struct {
			Type             string          `json:"type"`
			Subtype          string          `json:"subtype"`
			Result           string          `json:"result"`
			StructuredOutput json.RawMessage `json:"structured_output"`
			Cost             float64         `json:"total_cost_usd"`
			IsError          bool            `json:"is_error"`
			Session          string          `json:"session_id"`
		}

		if err := json.Unmarshal(entry, &partial); err != nil {
			continue
		}

		if partial.Type == "result" {
			result := partial.Result
			// When constrained decoding is active, structured_output contains
			// the schema-enforced JSON. Use it instead of the text result.
			if len(partial.StructuredOutput) > 0 && string(partial.StructuredOutput) != "null" {
				result = string(partial.StructuredOutput)
			}
			return &cliOutput{
				Result:       result,
				TotalCostUSD: partial.Cost,
				IsError:      partial.IsError,
				SessionID:    partial.Session,
			}, nil
		}
	}

	return nil, fmt.Errorf("no result entry found in CLI output")
}

// filterEnv returns a copy of environ with entries matching any of the given
// key names removed. Comparison is prefix-based ("KEY=").
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

// startHealthMonitor watches a progressTracker and updates member health in config.json.
// Shadow mode only: logs warnings, never kills. Runs until ctx is cancelled.
func startHealthMonitor(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int, tracker *progressTracker, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lastActivity := tracker.LastActivity()
			sinceActivity := time.Since(lastActivity)
			now := time.Now().UTC().Format(time.RFC3339)

			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.LastActivityTime = &now
				if sinceActivity > stallWarningThreshold {
					m.StallCount++
					if m.StallCount >= 3 {
						m.HealthStatus = "stalled"
					} else {
						m.HealthStatus = "stall_warning"
					}
					log.Printf("[SHADOW] health: member %s %s (no output for %v, count=%d)",
						m.Name, m.HealthStatus, sinceActivity.Round(time.Second), m.StallCount)
				} else {
					m.StallCount = 0
					m.HealthStatus = "healthy"
				}
			}); err != nil {
				log.Printf("ERROR: health monitor update failed for wave %d member %d: %v", waveIdx, memIdx, err)
			}
		}
	}
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

		// Set kill_reason if this was a timeout
		if strings.Contains(err.Error(), "timeout") {
			if updateErr := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.KillReason = "timeout"
			}); updateErr != nil {
				log.Printf("ERROR: Failed to set kill_reason for %s: %v", member.Name, updateErr)
			}
		}

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
			if tr.uds != nil && !tr.uds.isNoop() {
				tr.uds.notify(typeToast, toastPayload{
					Message: fmt.Sprintf("%s failed — /team-status for details", member.Name),
					Level:   "warn",
				})
			}
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
	if tr.uds != nil && !tr.uds.isNoop() {
		tr.uds.notify(typeToast, toastPayload{
			Message: fmt.Sprintf("%s failed after %d retries — /team-status for details",
				member.Name, member.MaxRetries+1),
			Level: "warn",
		})
	}
}

// tailStreamForUDS opens an independent read-only FD on the stream file and
// follows it as the child appends NDJSON lines. It parses tool_use events
// and forwards them as UDS agent_activity notifications.
//
// The stream file is written by the child with O_SYNC, guaranteeing each
// write() is flushed to disk before returning. POSIX guarantees atomic writes
// for sizes <= PIPE_BUF (4096 on Linux); NDJSON tool_use events are typically
// <1KB, so partial lines are not possible at the OS level.
//
// This goroutine runs alongside WatchFile (health monitoring) without conflict:
// each holds its own *os.File with independent file offset.
func tailStreamForUDS(ctx context.Context, path string, uds *TeamRunUDSClient, agentID string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("WARNING: tailStreamForUDS: open %s: %v", path, err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 64KB initial, 10MB max

	for {
		for scanner.Scan() {
			line := scanner.Bytes()
			ev := parseStreamEvent(line)
			if ev == nil || ev.Type != "assistant" {
				continue
			}
			ae := parseAssistantEvent(line)
			if ae == nil {
				continue
			}
			for _, block := range ae.Message.Content {
				if block.Type != "tool_use" {
					continue
				}
				act := extractToolActivity(block)
				uds.notify(typeAgentActivity, agentActivityPayload{
					AgentID: agentID,
					Tool:    act.Tool,
					Target:  act.Target,
					Preview: act.Preview,
				})
				// Forward TodoWrite events for acceptance criteria matching.
				if block.Name == "TodoWrite" {
					todos := parseTodoItems(block.Input)
					if len(todos) > 0 {
						uds.notify(typeAgentTodoUpdate, agentTodoUpdatePayload{
							AgentID: agentID,
							Todos:   todos,
						})
					}
				}
			}
		}
		// EOF reached — child may still be writing. Poll every 200ms.
		select {
		case <-ctx.Done():
			return
		case <-time.After(200 * time.Millisecond):
			// Continue scanning — new data may have been appended.
		}
	}
}

