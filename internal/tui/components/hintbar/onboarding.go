// Package hintbar — onboarding.go provides persistence for the first-run
// orientation hint state. It reads/writes a JSON file at
// $XDG_DATA_HOME/goyoke/onboarding.json (fallback: ~/.local/share/goyoke/onboarding.json).
package hintbar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// OnboardingState holds the persisted onboarding state.
type OnboardingState struct {
	SessionCount int      `json:"session_count"`
	Dismissed    []string `json:"dismissed"`
}

// onboardingPath returns the path to the onboarding state file.
// It respects $XDG_DATA_HOME with ~/.local/share as fallback.
func onboardingPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "goyoke", "onboarding.json")
}

// LoadOnboarding reads the onboarding state from the default path.
// If the file does not exist or cannot be read, it returns the zero state.
func LoadOnboarding() OnboardingState {
	path := onboardingPath()
	if path == "" {
		return OnboardingState{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return OnboardingState{}
	}
	var state OnboardingState
	if err := json.Unmarshal(data, &state); err != nil {
		return OnboardingState{}
	}
	return state
}

// SaveOnboarding writes the onboarding state to the default path.
// It creates the parent directory if it does not exist.
func SaveOnboarding(state OnboardingState) error {
	path := onboardingPath()
	if path == "" {
		return fmt.Errorf("cannot determine onboarding file path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create onboarding dir: %w", err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal onboarding state: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// IncrementSession loads the current onboarding state, increments the session
// count by one, saves the updated state, and returns the new state.
// Save errors are silently ignored so a non-writable filesystem does not crash
// the TUI startup path.
func IncrementSession() OnboardingState {
	state := LoadOnboarding()
	state.SessionCount++
	_ = SaveOnboarding(state)
	return state
}
