package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCaptureInput_EmptyArgs tests empty input handling.
func TestCaptureInput_EmptyArgs(t *testing.T) {
	input, err := CaptureInput([]string{})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if input != "" {
		t.Errorf("Expected empty string, got: %q", input)
	}
}

// TestCaptureInput_FileArgs tests reading from file arguments.
func TestCaptureInput_FileArgs(t *testing.T) {
	// Create temp files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.txt")
	file2 := filepath.Join(tmpDir, "test2.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test single file
	input, err := CaptureInput([]string{file1})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if input != "content1" {
		t.Errorf("Expected 'content1', got: %q", input)
	}

	// Test multiple files
	input, err = CaptureInput([]string{file1, file2})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := "content1\ncontent2"
	if input != expected {
		t.Errorf("Expected %q, got: %q", expected, input)
	}
}

// TestCaptureInput_NonexistentFile tests error handling for missing files.
func TestCaptureInput_NonexistentFile(t *testing.T) {
	_, err := CaptureInput([]string{"/nonexistent/file.txt"})
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "reading file") {
		t.Errorf("Expected 'reading file' in error, got: %v", err)
	}
}

// TestLoadProtocol tests protocol loading.
func TestLoadProtocol(t *testing.T) {
	// Test valid protocol
	config, content, err := LoadProtocol("mapper")
	if err != nil {
		t.Fatalf("Expected no error for mapper protocol, got: %v", err)
	}

	if config.Name != "mapper" {
		t.Errorf("Expected name 'mapper', got: %s", config.Name)
	}
	if config.Model != "gemini-3-flash-preview" {
		t.Errorf("Expected model 'gemini-3-flash-preview', got: %s", config.Model)
	}
	if config.Timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got: %v", config.Timeout)
	}
	if !strings.Contains(content, "PROTOCOL") {
		t.Errorf("Expected protocol content to contain 'PROTOCOL'")
	}

	// Test Pro protocol
	config, _, err = LoadProtocol("architect")
	if err != nil {
		t.Fatalf("Expected no error for architect protocol, got: %v", err)
	}
	if config.Model != "gemini-3-pro-preview" {
		t.Errorf("Expected Pro model for architect, got: %s", config.Model)
	}
	if config.Timeout != 180*time.Second {
		t.Errorf("Expected timeout 180s, got: %v", config.Timeout)
	}

	// Test invalid protocol
	_, _, err = LoadProtocol("nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent protocol, got nil")
	}
}

// TestProtocolModelMapping tests all 9 protocol configurations.
func TestProtocolModelMapping(t *testing.T) {
	tests := []struct {
		protocol      string
		expectedModel string
		expectedTier  string
	}{
		{"mapper", "gemini-3-flash-preview", "Flash"},
		{"memory-drift", "gemini-3-flash-preview", "Flash"},
		{"benchmark-score", "gemini-3-flash-preview", "Flash"},
		{"deps", "gemini-3-flash-preview", "Flash"},
		{"api-surface", "gemini-3-flash-preview", "Flash"},
		{"architect", "gemini-3-pro-preview", "Pro"},
		{"debugger", "gemini-3-pro-preview", "Pro"},
		{"memory-audit", "gemini-3-pro-preview", "Pro"},
		{"benchmark-audit", "gemini-3-pro-preview", "Pro"},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			config, exists := protocolConfig[tt.protocol]
			if !exists {
				t.Fatalf("Protocol %s not found in config", tt.protocol)
			}
			if config.Model != tt.expectedModel {
				t.Errorf("Expected model %s, got: %s", tt.expectedModel, config.Model)
			}
		})
	}
}

// TestListProtocols tests protocol directory listing.
func TestListProtocols(t *testing.T) {
	protocols, err := ListProtocols()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(protocols) == 0 {
		t.Fatal("Expected at least one protocol, got none")
	}

	// Check that key protocols exist
	expectedProtocols := []string{"mapper", "architect", "debugger", "deps"}
	for _, expected := range expectedProtocols {
		found := false
		for _, p := range protocols {
			if p == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected protocol %s not found in list", expected)
		}
	}
}

// TestBuildPrompt tests prompt assembly.
func TestBuildPrompt(t *testing.T) {
	protocolContent := "PROTOCOL: TEST"
	instruction := "Do something"
	inputContext := "Input data"

	prompt := buildPrompt(protocolContent, instruction, inputContext)

	if !strings.Contains(prompt, protocolContent) {
		t.Error("Prompt missing protocol content")
	}
	if !strings.Contains(prompt, instruction) {
		t.Error("Prompt missing instruction")
	}
	if !strings.Contains(prompt, inputContext) {
		t.Error("Prompt missing input context")
	}
	if !strings.Contains(prompt, "SPECIFIC INSTRUCTION") {
		t.Error("Prompt missing section header")
	}
	if !strings.Contains(prompt, "INPUT CONTEXT") {
		t.Error("Prompt missing input context header")
	}
}

// TestNewGeminiExecutor tests executor creation.
func TestNewGeminiExecutor(t *testing.T) {
	model := "gemini-3-flash-preview"
	timeout := 60 * time.Second

	executor := NewGeminiExecutor(model, timeout)

	if executor.Model != model {
		t.Errorf("Expected model %s, got: %s", model, executor.Model)
	}
	if executor.Timeout != timeout {
		t.Errorf("Expected timeout %v, got: %v", timeout, executor.Timeout)
	}
	if !strings.Contains(executor.HomeOverride, ".gemini-slave") {
		t.Errorf("Expected HomeOverride to contain .gemini-slave, got: %s", executor.HomeOverride)
	}
}
