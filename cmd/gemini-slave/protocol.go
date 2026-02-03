package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Protocol defines configuration for a Gemini protocol.
type Protocol struct {
	Name    string
	Model   string        // gemini-3-flash-preview or gemini-3-pro-preview
	Timeout time.Duration // Execution timeout
}

// protocolConfig maps protocol names to their model and timeout settings.
// Fast protocols use Flash, complex reasoning uses Pro.
var protocolConfig = map[string]Protocol{
	// Fast protocols (Flash)
	"mapper": {
		Name:    "mapper",
		Model:   "gemini-3-flash-preview",
		Timeout: 60 * time.Second,
	},
	"memory-drift": {
		Name:    "memory-drift",
		Model:   "gemini-3-flash-preview",
		Timeout: 60 * time.Second,
	},
	"benchmark-score": {
		Name:    "benchmark-score",
		Model:   "gemini-3-flash-preview",
		Timeout: 60 * time.Second,
	},
	"deps": {
		Name:    "deps",
		Model:   "gemini-3-flash-preview",
		Timeout: 90 * time.Second,
	},
	"api-surface": {
		Name:    "api-surface",
		Model:   "gemini-3-flash-preview",
		Timeout: 90 * time.Second,
	},

	// Complex protocols (Pro)
	"architect": {
		Name:    "architect",
		Model:   "gemini-3-pro-preview",
		Timeout: 180 * time.Second,
	},
	"debugger": {
		Name:    "debugger",
		Model:   "gemini-3-pro-preview",
		Timeout: 180 * time.Second,
	},
	"memory-audit": {
		Name:    "memory-audit",
		Model:   "gemini-3-pro-preview",
		Timeout: 300 * time.Second,
	},
	"benchmark-audit": {
		Name:    "benchmark-audit",
		Model:   "gemini-3-pro-preview",
		Timeout: 300 * time.Second,
	},
}

// LoadProtocol loads a protocol file and returns its configuration.
// Protocol files are stored in ~/.gemini-slave/protocols/{name}.md
func LoadProtocol(name string) (Protocol, string, error) {
	// Get protocol config (for model/timeout)
	config, exists := protocolConfig[name]
	if !exists {
		return Protocol{}, "", fmt.Errorf("unknown protocol: %s", name)
	}

	// Load protocol content from file
	home := os.Getenv("HOME")
	if home == "" {
		return Protocol{}, "", fmt.Errorf("HOME environment variable not set")
	}

	protocolPath := filepath.Join(home, ".gemini-slave", "protocols", name+".md")
	content, err := os.ReadFile(protocolPath)
	if err != nil {
		return Protocol{}, "", fmt.Errorf("reading protocol file %s: %w", protocolPath, err)
	}

	return config, string(content), nil
}

// ListProtocols returns a list of available protocol names.
func ListProtocols() ([]string, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return nil, fmt.Errorf("HOME environment variable not set")
	}

	protocolDir := filepath.Join(home, ".gemini-slave", "protocols")
	entries, err := os.ReadDir(protocolDir)
	if err != nil {
		return nil, fmt.Errorf("reading protocol directory: %w", err)
	}

	var protocols []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			name := entry.Name()[:len(entry.Name())-3] // Remove .md extension
			protocols = append(protocols, name)
		}
	}

	return protocols, nil
}
