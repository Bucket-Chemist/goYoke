package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// writeCounter provides unique suffixes for atomic config writes.
// Prevents concurrent writes from clobbering each other's .tmp files.
var writeCounter atomic.Int64

// TeamConfig represents the team configuration as defined in .claude/schemas/teams/team-config.json
type TeamConfig struct {
	TeamName            string  `json:"team_name"`
	WorkflowType        string  `json:"workflow_type"`
	ProjectRoot         string  `json:"project_root"`
	SessionID           string  `json:"session_id"`
	CreatedAt           string  `json:"created_at"`
	BudgetMaxUSD        float64 `json:"budget_max_usd"`
	BudgetRemainingUSD  float64 `json:"budget_remaining_usd"`
	WarningThresholdUSD float64 `json:"warning_threshold_usd"`
	Status              string  `json:"status"`
	BackgroundPID       *int    `json:"background_pid"`
	StartedAt           *string `json:"started_at"`
	CompletedAt         *string `json:"completed_at"`
	Waves               []Wave  `json:"waves"`
}

// Wave represents a wave of parallel agent executions
type Wave struct {
	WaveNumber       int      `json:"wave_number"`
	Description      string   `json:"description"`
	Members          []Member `json:"members"`
	OnCompleteScript *string  `json:"on_complete_script"`
}

// Member represents a single team member (agent) within a wave
type Member struct {
	Name         string  `json:"name"`
	Agent        string  `json:"agent"`
	Model        string  `json:"model"`
	StdinFile    string  `json:"stdin_file"`
	StdoutFile   string  `json:"stdout_file"`
	Status       string  `json:"status"`        // pending|running|completed|failed
	ProcessPID   *int    `json:"process_pid"`   // PID of spawned Claude CLI process
	ExitCode     *int    `json:"exit_code"`     // Exit code after completion
	CostUSD      float64 `json:"cost_usd"`      // Extracted cost from CLI output
	CostStatus   string  `json:"cost_status"`   // ok|warning|exceeded
	ErrorMessage string  `json:"error_message"` // Error details if failed
	RetryCount   int     `json:"retry_count"`   // Current retry attempt
	MaxRetries   int     `json:"max_retries"`   // Maximum retry attempts
	TimeoutMs    int     `json:"timeout_ms"`    // Process timeout in milliseconds
	StartedAt    *string `json:"started_at"`    // ISO 8601 timestamp
	CompletedAt  *string `json:"completed_at"`  // ISO 8601 timestamp
}

// configManager provides thread-safe access to TeamConfig
// Embedded into TeamRunner in daemon.go
type configManager struct {
	config     *TeamConfig
	configPath string
	mu         sync.RWMutex
}

// LoadConfig reads and unmarshals the team config.json
// If config.json doesn't exist, returns nil error (for tests that don't need config)
func (tr *TeamRunner) LoadConfig() error {
	configPath := filepath.Join(tr.teamDir, ConfigFileName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist - this is OK for tests that don't need it
			return nil
		}
		return fmt.Errorf("read config.json: %w", err)
	}

	var config TeamConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("unmarshal config.json: %w", err)
	}

	tr.configMu.Lock()
	tr.config = &config
	tr.configPath = configPath
	tr.configMu.Unlock()

	return nil
}

// deepCopyConfig creates a deep copy of a TeamConfig, cloning all pointer fields.
// Used by updateMember to serialize config without holding the lock.
func deepCopyConfig(src *TeamConfig) TeamConfig {
	dst := *src

	// Clone top-level pointer fields
	dst.BackgroundPID = cloneIntPtr(src.BackgroundPID)
	dst.StartedAt = cloneStringPtr(src.StartedAt)
	dst.CompletedAt = cloneStringPtr(src.CompletedAt)

	// Deep copy waves and members
	dst.Waves = make([]Wave, len(src.Waves))
	for i := range src.Waves {
		dst.Waves[i] = src.Waves[i]
		dst.Waves[i].OnCompleteScript = cloneStringPtr(src.Waves[i].OnCompleteScript)
		dst.Waves[i].Members = make([]Member, len(src.Waves[i].Members))
		for j := range src.Waves[i].Members {
			m := src.Waves[i].Members[j]
			m.ProcessPID = cloneIntPtr(m.ProcessPID)
			m.ExitCode = cloneIntPtr(m.ExitCode)
			m.StartedAt = cloneStringPtr(m.StartedAt)
			m.CompletedAt = cloneStringPtr(m.CompletedAt)
			dst.Waves[i].Members[j] = m
		}
	}

	return dst
}

func cloneStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func cloneIntPtr(p *int) *int {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}

// uniqueTmpPath returns a unique temporary file path for atomic writes.
// Format: config.json.<pid>.<counter>.tmp
func uniqueTmpPath(configPath string) string {
	count := writeCounter.Add(1)
	return fmt.Sprintf("%s.%d.%d.tmp", configPath, os.Getpid(), count)
}

// SaveConfig marshals and atomically writes the team config.json
// Uses atomic write pattern: write to .tmp, rename
func (tr *TeamRunner) SaveConfig() error {
	tr.configMu.RLock()
	if tr.config == nil {
		tr.configMu.RUnlock()
		return fmt.Errorf("config not loaded")
	}
	config := tr.config
	configPath := tr.configPath
	tr.configMu.RUnlock()

	// Marshal with indentation for human readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Atomic write: write to .tmp, rename
	tmpPath := uniqueTmpPath(configPath)
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write tmp config: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		// Cleanup tmp file on failure
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp config: %w", err)
	}

	return nil
}

// updateMember atomically updates a member by applying a function to it
// Locks configMu, calls fn, saves config
func (tr *TeamRunner) updateMember(waveIdx, memIdx int, fn func(*Member)) error {
	tr.configMu.Lock()

	if tr.config == nil {
		tr.configMu.Unlock()
		return fmt.Errorf("config not loaded")
	}

	if waveIdx < 0 || waveIdx >= len(tr.config.Waves) {
		tr.configMu.Unlock()
		return fmt.Errorf("invalid wave index: %d", waveIdx)
	}

	if memIdx < 0 || memIdx >= len(tr.config.Waves[waveIdx].Members) {
		tr.configMu.Unlock()
		return fmt.Errorf("invalid member index: %d", memIdx)
	}

	// Apply mutation
	fn(&tr.config.Waves[waveIdx].Members[memIdx])

	// Make a deep copy for serialization (avoid holding lock during I/O)
	configCopy := deepCopyConfig(tr.config)
	configPath := tr.configPath

	tr.configMu.Unlock()

	// Marshal and write (no lock held)
	data, err := json.MarshalIndent(&configCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	tmpPath := uniqueTmpPath(configPath)
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write tmp config: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp config: %w", err)
	}

	return nil
}

// findMember finds a member by name across all waves.
// Returns (waveIdx, memIdx, true) if found, (-1, -1, false) otherwise.
func (tr *TeamRunner) findMember(name string) (waveIdx, memIdx int, found bool) {
	tr.configMu.RLock()
	defer tr.configMu.RUnlock()

	if tr.config == nil {
		return -1, -1, false
	}

	for wIdx, wave := range tr.config.Waves {
		for mIdx, member := range wave.Members {
			if member.Name == name {
				return wIdx, mIdx, true
			}
		}
	}

	return -1, -1, false
}
