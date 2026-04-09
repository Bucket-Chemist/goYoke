package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// wave0Output represents a single reviewer's result for Pasteur's stdin.
type wave0Output struct {
	ReviewerID     string  `json:"reviewer_id"`
	StdoutFilePath string  `json:"stdout_file_path"`
	Status         string  `json:"status"`
	CostUSD        float64 `json:"cost_usd"`
}

// prepareBioinformaticsReview updates stdin_pasteur.json with wave_0_outputs
// from the completed reviewer wave. This allows Pasteur to receive partial
// results even when some reviewers fail.
//
// Graceful degradation: if Pasteur is not found, or if stdin_pasteur.json
// is missing or unparseable, logs a warning and returns nil.
func prepareBioinformaticsReview(teamDir string, config *TeamConfig, completedWaveIdx int) error {
	if completedWaveIdx+1 >= len(config.Waves) {
		log.Printf("[WARN] prepareBioinformaticsReview: completedWaveIdx %d has no next wave", completedWaveIdx)
		return nil
	}

	// Find Pasteur's stdin file in the next wave.
	pasteurStdinFile := ""
	for _, m := range config.Waves[completedWaveIdx+1].Members {
		if m.Agent == "pasteur" {
			pasteurStdinFile = m.StdinFile
			break
		}
	}
	if pasteurStdinFile == "" {
		log.Printf("[WARN] prepareBioinformaticsReview: pasteur member not found in wave %d, skipping", completedWaveIdx+1)
		return nil
	}

	stdinPath := filepath.Join(teamDir, pasteurStdinFile)
	data, err := os.ReadFile(stdinPath)
	if err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: cannot read %s: %v, skipping", stdinPath, err)
		return nil
	}

	var stdinData map[string]interface{}
	if err := json.Unmarshal(data, &stdinData); err != nil {
		log.Printf("[WARN] prepareBioinformaticsReview: cannot parse %s: %v, skipping", stdinPath, err)
		return nil
	}

	// Build wave_0_outputs from completed wave members.
	outputs := make([]wave0Output, 0, len(config.Waves[completedWaveIdx].Members))
	for _, m := range config.Waves[completedWaveIdx].Members {
		outputs = append(outputs, wave0Output{
			ReviewerID:     m.Agent,
			StdoutFilePath: filepath.Join(teamDir, m.StdoutFile),
			Status:         memberStatusToOutputStatus(m.Status),
			CostUSD:        m.CostUSD,
		})
	}

	stdinData["wave_0_outputs"] = outputs

	updated, err := json.MarshalIndent(stdinData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal updated stdin: %w", err)
	}

	// Atomic write: write to .tmp then rename.
	tmpPath := stdinPath + ".tmp"
	if err := os.WriteFile(tmpPath, updated, 0644); err != nil {
		return fmt.Errorf("write tmp stdin file: %w", err)
	}
	if err := os.Rename(tmpPath, stdinPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename tmp to stdin file: %w", err)
	}

	log.Printf("[INFO] prepareBioinformaticsReview: updated %s with %d wave_0_outputs", stdinPath, len(outputs))
	return nil
}

// memberStatusToOutputStatus maps member.Status to the output status values
// expected by Pasteur's stdin schema.
func memberStatusToOutputStatus(status string) string {
	switch status {
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	default:
		return "timeout"
	}
}
