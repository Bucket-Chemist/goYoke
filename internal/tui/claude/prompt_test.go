package claude

import "testing"

func TestSanitizePrompt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Normal text", "Normal text"},
		{"\x1b[31mRed text\x1b[0m", "Red text"},
		{"\x1b[2J\x1b[HClear and home", "Clear and home"},
		{"Text with\x1b]0;fake title\x07OSC", "Text withake title\x07OSC"}, // stripansi removes CSI/OSC sequences
	}

	for _, tc := range tests {
		got := sanitizePrompt(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizePrompt(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}
