package main

import (
	"testing"
)

func boolPtr(v bool) *bool { return &v }

func TestExtractThinkingValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *bool
	}{
		// Nil
		{name: "nil", input: nil, expected: nil},

		// Bool cases
		{name: "bool true", input: true, expected: boolPtr(true)},
		{name: "bool false", input: false, expected: boolPtr(false)},

		// Legacy {enabled: bool}
		{name: "enabled true", input: map[string]interface{}{"enabled": true}, expected: boolPtr(true)},
		{name: "enabled false", input: map[string]interface{}{"enabled": false}, expected: boolPtr(false)},

		// Opus 4.6 {type: ...}
		{name: "type adaptive", input: map[string]interface{}{"type": "adaptive"}, expected: boolPtr(true)},
		{name: "type enabled with budget", input: map[string]interface{}{"type": "enabled", "budget_tokens": 8000}, expected: boolPtr(true)},
		{name: "type disabled", input: map[string]interface{}{"type": "disabled"}, expected: boolPtr(false)},

		// Unknown type string → nil
		{name: "unknown type", input: map[string]interface{}{"type": "unknown"}, expected: nil},

		// Empty map → nil
		{name: "empty map", input: map[string]interface{}{}, expected: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractThinkingValue(tc.input)
			if tc.expected == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Errorf("expected %v, got nil", *tc.expected)
				return
			}
			if *got != *tc.expected {
				t.Errorf("expected %v, got %v", *tc.expected, *got)
			}
		})
	}
}

func TestNormalizeModel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"opus API string", "claude-opus-4-6", "opus"},
		{"sonnet API string", "claude-sonnet-4-6", "sonnet"},
		{"haiku API string", "claude-haiku-4-5-20251001", "haiku"},
		{"opus tier name passthrough", "opus", "opus"},
		{"sonnet tier name passthrough", "sonnet", "sonnet"},
		{"haiku tier name passthrough", "haiku", "haiku"},
		{"unknown string passthrough", "claude-3-opus", "claude-3-opus"},
		{"empty string passthrough", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeModel(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeModel(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
