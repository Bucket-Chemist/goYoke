package teamrun

import (
	"encoding/json"
	"fmt"
	"log"
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
	// DO NOT mutate directly. Use tryReserveBudget()/reconcileCost() ONLY.
	// Field must remain exported for JSON marshaling, but access via BudgetRemaining() reader.
	BudgetRemainingUSD  float64 `json:"budget_remaining_usd"`
	WarningThresholdUSD float64 `json:"warning_threshold_usd"`
	Status              string  `json:"status"`
	BackgroundPID       *int    `json:"background_pid"`
	StartedAt           *string `json:"started_at"`
	CompletedAt         *string `json:"completed_at"`
	StallDetectionMode  string  `json:"stall_detection_mode,omitempty"` // ""=shadow|"active"=enforce
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
	Name             string  `json:"name"`
	Agent            string  `json:"agent"`
	Model            string  `json:"model"`
	StdinFile        string  `json:"stdin_file"`
	StdoutFile       string  `json:"stdout_file"`
	Status           string  `json:"status"`                      // pending|running|completed|failed
	ProcessPID       *int    `json:"process_pid"`                 // PID of spawned Claude CLI process
	ExitCode         *int    `json:"exit_code"`                   // Exit code after completion
	CostUSD          float64 `json:"cost_usd"`                    // Extracted cost from CLI output
	CostStatus       string  `json:"cost_status"`                 // ok|warning|exceeded
	ErrorMessage     string  `json:"error_message"`               // Error details if failed
	KillReason       string  `json:"kill_reason,omitempty"`       // Reason for termination (e.g., "timeout")
	RetryCount       int     `json:"retry_count"`                 // Current retry attempt
	MaxRetries       int     `json:"max_retries"`                 // Maximum retry attempts
	TimeoutMs        int     `json:"timeout_ms"`                  // Process timeout in milliseconds
	StartedAt        *string `json:"started_at"`                  // ISO 8601 timestamp
	CompletedAt      *string `json:"completed_at"`                // ISO 8601 timestamp
	HealthStatus     string  `json:"health_status,omitempty"`     // healthy|stall_warning|stalled
	LastActivityTime *string `json:"last_activity_time,omitempty"` // ISO 8601 timestamp of last output
	StallCount       int     `json:"stall_count,omitempty"`       // Number of consecutive stall warnings
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
			m.LastActivityTime = cloneStringPtr(m.LastActivityTime)
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
	// C4: Acquire write mutex to serialize disk writes
	tr.writeMu.Lock()
	defer tr.writeMu.Unlock()

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
		if removeErr := os.Remove(tmpPath); removeErr != nil {
			log.Printf("WARNING: Failed to clean up temp file %s: %v", tmpPath, removeErr)
		}
		return fmt.Errorf("rename tmp config: %w", err)
	}

	return nil
}

// updateMember atomically updates a member by applying a function to it
// Locks configMu, calls fn, saves config
func (tr *TeamRunner) updateMember(waveIdx, memIdx int, fn func(*Member)) error {
	// C4: Acquire write mutex FIRST to serialize disk writes
	tr.writeMu.Lock()
	defer tr.writeMu.Unlock()

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

	// Marshal and write (writeMu still held - no concurrent writes)
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

// BudgetRemaining returns the current budget remaining with thread-safe read access.
// This is the ONLY safe way to read budget without holding both locks.
func (tr *TeamRunner) BudgetRemaining() float64 {
	tr.configMu.RLock()
	defer tr.configMu.RUnlock()

	if tr.config == nil {
		return 0
	}

	return tr.config.BudgetRemainingUSD
}

// tryReserveBudget atomically checks if sufficient budget exists and reserves it.
// Returns true if reservation succeeded, false if insufficient budget.
// This is the ONLY safe way to reserve budget before spawning an agent.
func (tr *TeamRunner) tryReserveBudget(estimatedCost float64) bool {
	tr.writeMu.Lock()
	defer tr.writeMu.Unlock()

	tr.configMu.Lock()
	defer tr.configMu.Unlock()

	if tr.config == nil {
		return false
	}

	if tr.config.BudgetRemainingUSD < estimatedCost {
		return false
	}

	tr.config.BudgetRemainingUSD -= estimatedCost
	return true
}

// reconcileCost returns the estimated cost reservation and deducts the actual cost.
// Ensures budget never goes negative (C1: floor enforcement).
// Logs at CRITICAL level when clamping occurs.
func (tr *TeamRunner) reconcileCost(estimated, actual float64) error {
	tr.writeMu.Lock()
	defer tr.writeMu.Unlock()

	tr.configMu.Lock()

	if tr.config == nil {
		tr.configMu.Unlock()
		return fmt.Errorf("config not loaded")
	}

	// Return the reservation
	tr.config.BudgetRemainingUSD += estimated

	// Deduct actual cost
	tr.config.BudgetRemainingUSD -= actual

	// C1: Floor enforcement - budget must never go negative
	if tr.config.BudgetRemainingUSD < 0 {
		log.Printf("[CRITICAL] Budget went negative ($%.4f), clamping to $0.00", tr.config.BudgetRemainingUSD)
		tr.config.BudgetRemainingUSD = 0
	}

	// Make a deep copy for serialization
	configCopy := deepCopyConfig(tr.config)
	configPath := tr.configPath

	tr.configMu.Unlock()

	// Marshal and write (writeMu still held)
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

// estimateCost returns a conservative cost estimate for an agent based on model.
// Uses model-based heuristics with fallback for unknown agents.
// Default fallback: $1.50 (conservative Sonnet estimate).
func (tr *TeamRunner) estimateCost(agentID string) float64 {
	// TODO(TC-008): Load from agents-index.json when available
	// For now, use model-based heuristics

	tr.configMu.RLock()
	model := ""
	if tr.config != nil {
		// Find agent's model in config
		for _, wave := range tr.config.Waves {
			for _, member := range wave.Members {
				if member.Agent == agentID {
					model = member.Model
					break
				}
			}
			if model != "" {
				break
			}
		}
	}
	tr.configMu.RUnlock()

	// Model-based cost estimates (conservative)
	switch model {
	case "haiku":
		return 0.10 // ~$0.10 for typical haiku task
	case "sonnet":
		return 1.50 // ~$1.50 for typical sonnet task
	case "opus":
		return 5.00 // ~$5.00 for typical opus task
	default:
		// Unknown agent or model - use conservative fallback
		return 1.50
	}
}
