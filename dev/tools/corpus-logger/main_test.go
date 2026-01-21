package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessEvent(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name: "valid PreToolUse event",
			input: `{
				"tool_name": "Task",
				"tool_input": {
					"model": "sonnet",
					"prompt": "AGENT: python-pro\n\nImplement function",
					"subagent_type": "general-purpose"
				},
				"session_id": "abc-123",
				"hook_event_name": "PreToolUse"
			}`,
			wantError: false,
		},
		{
			name: "valid PostToolUse event",
			input: `{
				"tool_name": "Bash",
				"tool_response": {
					"exit_code": 1,
					"stderr": "Error"
				},
				"session_id": "abc-123",
				"hook_event_name": "PostToolUse"
			}`,
			wantError: false,
		},
		{
			name:      "empty input",
			input:     "",
			wantError: false, // Should skip gracefully
		},
		{
			name:      "invalid json",
			input:     `{invalid json}`,
			wantError: true,
		},
		{
			name: "minimal valid event",
			input: `{
				"tool_name": "Read",
				"session_id": "test-123",
				"hook_event_name": "PreToolUse"
			}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary output directory
			tmpDir := t.TempDir()

			// Override environment to use test directory
			// Must unset XDG_RUNTIME_DIR to force use of XDG_CACHE_HOME
			oldRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
			os.Unsetenv("XDG_RUNTIME_DIR")
			defer func() {
				if oldRuntimeDir != "" {
					os.Setenv("XDG_RUNTIME_DIR", oldRuntimeDir)
				}
			}()

			os.Setenv("XDG_CACHE_HOME", tmpDir)
			defer os.Unsetenv("XDG_CACHE_HOME")

			// Process event
			err := processEvent([]byte(tt.input))

			if (err != nil) != tt.wantError {
				t.Errorf("processEvent() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// For successful cases with non-empty input, verify output
			if !tt.wantError && tt.input != "" {
				// Check file was created
				expectedPath := filepath.Join(tmpDir, "gogent", "event-corpus-raw.jsonl")
				if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
					t.Errorf("output file not created at %s", expectedPath)
					return
				}

				// Read and parse output
				data, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatalf("reading output file: %v", err)
				}

				var event HookEvent
				if err := json.Unmarshal(data, &event); err != nil {
					t.Fatalf("parsing output json: %v", err)
				}

				// Verify timestamp was added
				if event.CapturedAt == 0 {
					t.Error("captured_at timestamp not set")
				}

				// Verify required fields from input
				var inputEvent HookEvent
				if err := json.Unmarshal([]byte(tt.input), &inputEvent); err == nil {
					if event.ToolName != inputEvent.ToolName {
						t.Errorf("tool_name mismatch: got %s, want %s", event.ToolName, inputEvent.ToolName)
					}
					if event.SessionID != inputEvent.SessionID {
						t.Errorf("session_id mismatch: got %s, want %s", event.SessionID, inputEvent.SessionID)
					}
					if event.HookEventName != inputEvent.HookEventName {
						t.Errorf("hook_event_name mismatch: got %s, want %s", event.HookEventName, inputEvent.HookEventName)
					}
				}
			}
		})
	}
}

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		wantContain string
	}{
		{
			name: "uses XDG_RUNTIME_DIR when set",
			setupEnv: func() {
				os.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
			},
			cleanupEnv: func() {
				os.Unsetenv("XDG_RUNTIME_DIR")
			},
			wantContain: "/run/user/1000/gogent/event-corpus-raw.jsonl",
		},
		{
			name: "uses XDG_CACHE_HOME when XDG_RUNTIME_DIR not set",
			setupEnv: func() {
				os.Unsetenv("XDG_RUNTIME_DIR")
				os.Setenv("XDG_CACHE_HOME", "/custom/cache")
			},
			cleanupEnv: func() {
				os.Unsetenv("XDG_CACHE_HOME")
			},
			wantContain: "/custom/cache/gogent/event-corpus-raw.jsonl",
		},
		{
			name: "falls back to ~/.cache when no XDG vars",
			setupEnv: func() {
				os.Unsetenv("XDG_RUNTIME_DIR")
				os.Unsetenv("XDG_CACHE_HOME")
			},
			cleanupEnv:  func() {},
			wantContain: ".cache/gogent/event-corpus-raw.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			path, err := resolveOutputPath()
			if err != nil {
				t.Fatalf("resolveOutputPath() error = %v", err)
			}

			if tt.wantContain != "" && !contains(path, tt.wantContain) {
				t.Errorf("resolveOutputPath() = %v, want to contain %v", path, tt.wantContain)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || filepath.Clean(s) == filepath.Clean(substr) || len(filepath.Dir(s)) > 0 && filepath.Base(filepath.Dir(s))+"/"+filepath.Base(s) == substr || s[len(s)-len(substr):] == substr)
}
