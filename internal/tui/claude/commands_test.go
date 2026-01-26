package claude

import (
	"strings"
	"testing"
)

func TestIsNativeCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "model command",
			input: "/model",
			want:  true,
		},
		{
			name:  "context command",
			input: "/context",
			want:  true,
		},
		{
			name:  "model with args",
			input: "/model opus",
			want:  true,
		},
		{
			name:  "unknown command",
			input: "/unknown",
			want:  false,
		},
		{
			name:  "not a command",
			input: "hello world",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "slash only",
			input: "/",
			want:  false,
		},
		{
			name:  "case insensitive",
			input: "/MODEL",
			want:  true,
		},
		{
			name:  "whitespace before",
			input: "  /model",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNativeCommand(tt.input)
			if got != tt.want {
				t.Errorf("IsNativeCommand(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeModelName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Opus variations
		{name: "opus", input: "opus", want: "opus"},
		{name: "opus uppercase", input: "OPUS", want: "opus"},
		{name: "claude-3-opus", input: "claude-3-opus", want: "opus"},
		{name: "claude-opus-4", input: "claude-opus-4", want: "opus"},
		{name: "claude-opus-4-5", input: "claude-opus-4-5", want: "opus"},
		{name: "opus-4", input: "opus-4", want: "opus"},

		// Sonnet variations
		{name: "sonnet", input: "sonnet", want: "sonnet"},
		{name: "sonnet uppercase", input: "SONNET", want: "sonnet"},
		{name: "claude-3-sonnet", input: "claude-3-sonnet", want: "sonnet"},
		{name: "claude-sonnet-4", input: "claude-sonnet-4", want: "sonnet"},
		{name: "claude-sonnet-4-5", input: "claude-sonnet-4-5", want: "sonnet"},
		{name: "sonnet-4", input: "sonnet-4", want: "sonnet"},

		// Haiku variations
		{name: "haiku", input: "haiku", want: "haiku"},
		{name: "haiku uppercase", input: "HAIKU", want: "haiku"},
		{name: "claude-3-haiku", input: "claude-3-haiku", want: "haiku"},
		{name: "claude-haiku-3.5", input: "claude-haiku-3.5", want: "haiku"},
		{name: "haiku-3.5", input: "haiku-3.5", want: "haiku"},

		// Invalid cases
		{name: "unknown", input: "gpt-4", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "random", input: "random", want: ""},

		// Whitespace handling
		{name: "with whitespace", input: "  opus  ", want: "opus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeModelName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeModelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExecuteCommand_Model(t *testing.T) {
	ctx := CommandContext{
		SessionID:    "test-session",
		CurrentModel: "sonnet",
		MessageCount: 5,
		TotalCost:    0.042,
	}

	tests := []struct {
		name                string
		input               string
		wantError           bool
		wantRequiresRestart bool
		wantNewModel        string
		wantMessageContains string
	}{
		{
			name:                "no args shows current",
			input:               "/model",
			wantError:           false,
			wantRequiresRestart: false,
			wantMessageContains: "Current model: sonnet",
		},
		{
			name:                "change to opus",
			input:               "/model opus",
			wantError:           false,
			wantRequiresRestart: true,
			wantNewModel:        "opus",
			wantMessageContains: "sonnet → opus",
		},
		{
			name:                "change to haiku",
			input:               "/model haiku",
			wantError:           false,
			wantRequiresRestart: true,
			wantNewModel:        "haiku",
			wantMessageContains: "sonnet → haiku",
		},
		{
			name:                "invalid model",
			input:               "/model gpt-4",
			wantError:           true,
			wantRequiresRestart: false,
			wantMessageContains: "Unknown model",
		},
		{
			name:                "already using model",
			input:               "/model sonnet",
			wantError:           false,
			wantRequiresRestart: false,
			wantMessageContains: "Already using",
		},
		{
			name:                "case insensitive model",
			input:               "/model OPUS",
			wantError:           false,
			wantRequiresRestart: true,
			wantNewModel:        "opus",
		},
		{
			name:                "full model name",
			input:               "/model claude-opus-4-5",
			wantError:           false,
			wantRequiresRestart: true,
			wantNewModel:        "opus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExecuteCommand(tt.input, ctx)

			if (result.Error != nil) != tt.wantError {
				t.Errorf("ExecuteCommand() error = %v, wantError %v", result.Error, tt.wantError)
			}

			if result.RequiresRestart != tt.wantRequiresRestart {
				t.Errorf("ExecuteCommand() RequiresRestart = %v, want %v", result.RequiresRestart, tt.wantRequiresRestart)
			}

			if result.NewModel != tt.wantNewModel {
				t.Errorf("ExecuteCommand() NewModel = %v, want %v", result.NewModel, tt.wantNewModel)
			}

			if tt.wantMessageContains != "" && !strings.Contains(result.Message, tt.wantMessageContains) {
				t.Errorf("ExecuteCommand() Message = %q, want to contain %q", result.Message, tt.wantMessageContains)
			}
		})
	}
}

func TestExecuteCommand_Context(t *testing.T) {
	ctx := CommandContext{
		SessionID:    "abc123xyz",
		CurrentModel: "opus",
		MessageCount: 42,
		TotalCost:    1.234,
	}

	result := ExecuteCommand("/context", ctx)

	if result.Error != nil {
		t.Errorf("ExecuteCommand(/context) unexpected error: %v", result.Error)
	}

	if result.RequiresRestart {
		t.Errorf("ExecuteCommand(/context) should not require restart")
	}

	// Check that message contains key information
	expectedContains := []string{
		"abc123xyz",
		"opus",
		"42",
		"1.234",
	}

	for _, expected := range expectedContains {
		if !strings.Contains(result.Message, expected) {
			t.Errorf("ExecuteCommand(/context) Message missing %q. Got: %q", expected, result.Message)
		}
	}
}

func TestExecuteCommand_Unknown(t *testing.T) {
	ctx := CommandContext{
		SessionID:    "test",
		CurrentModel: "sonnet",
	}

	result := ExecuteCommand("/unknown", ctx)

	if result.Error == nil {
		t.Errorf("ExecuteCommand(/unknown) expected error, got nil")
	}

	if !strings.Contains(result.Message, "Unknown command") {
		t.Errorf("ExecuteCommand(/unknown) Message should mention unknown command. Got: %q", result.Message)
	}
}

func TestExecuteCommand_Empty(t *testing.T) {
	ctx := CommandContext{
		SessionID:    "test",
		CurrentModel: "sonnet",
	}

	result := ExecuteCommand("", ctx)

	if result.Error == nil {
		t.Errorf("ExecuteCommand(\"\") expected error, got nil")
	}
}
