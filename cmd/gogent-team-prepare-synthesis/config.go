package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TeamConfig is a minimal config for goyoke-team-prepare-synthesis.
// Only the fields needed for workflow dispatch and wave detection are included.
// Source of truth: cmd/goyoke-team-run/config.go (TeamConfig struct).
type TeamConfig struct {
	WorkflowType string     `json:"workflow_type"`
	Waves        []WaveInfo `json:"waves"`
}

// WaveInfo is a minimal wave representation for config reading.
type WaveInfo struct {
	Members []MemberInfo `json:"members"`
}

// MemberInfo is a minimal member representation for config reading.
type MemberInfo struct {
	Agent      string  `json:"agent"`
	StdinFile  string  `json:"stdin_file"`
	StdoutFile string  `json:"stdout_file"`
	Status     string  `json:"status"`
	CostUSD    float64 `json:"cost_usd"`
}

// loadConfig reads config.json from teamDir and returns the parsed config.
func loadConfig(teamDir string) (*TeamConfig, error) {
	configPath := filepath.Join(teamDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config.json: %w", err)
	}
	var cfg TeamConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	return &cfg, nil
}

// findCompletedWaveIdx returns the index of the first wave where:
//   - all members are in a terminal state (completed or failed), AND
//   - the next wave has at least one pending member.
//
// Returns -1 if no such wave is found.
func findCompletedWaveIdx(config *TeamConfig) int {
	for i, wave := range config.Waves {
		allTerminal := true
		for _, m := range wave.Members {
			if m.Status != "completed" && m.Status != "failed" {
				allTerminal = false
				break
			}
		}
		if allTerminal && i+1 < len(config.Waves) {
			for _, m := range config.Waves[i+1].Members {
				if m.Status == "pending" {
					return i
				}
			}
		}
	}
	return -1
}
