package util

import "testing"

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxRunes int
		want     string
	}{
		{
			name:     "ASCII string under limit unchanged",
			input:    "hello",
			maxRunes: 10,
			want:     "hello",
		},
		{
			name:     "ASCII string at exact boundary unchanged",
			input:    "hello",
			maxRunes: 5,
			want:     "hello",
		},
		{
			name:     "ASCII string over limit truncated with ellipsis",
			input:    "hello world",
			maxRunes: 5,
			want:     "hello…",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			maxRunes: 10,
			want:     "",
		},
		{
			name:     "empty string with zero limit",
			input:    "",
			maxRunes: 0,
			want:     "",
		},
		{
			name:     "zero maxRunes returns empty",
			input:    "hello",
			maxRunes: 0,
			want:     "",
		},
		{
			name:     "CJK characters truncated at rune boundary",
			input:    "日本語テスト",
			maxRunes: 4,
			want:     "日本語テ…",
		},
		{
			name:     "emoji multi-byte truncated at rune boundary",
			input:    "😀😁😂😃😄",
			maxRunes: 3,
			want:     "😀😁😂…",
		},
		{
			name:     "mixed ASCII and CJK over limit",
			input:    "abc日本語",
			maxRunes: 4,
			want:     "abc日…",
		},
		{
			name:     "single character over limit returns ellipsis only",
			input:    "hello",
			maxRunes: 1,
			want:     "h…",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Truncate(tc.input, tc.maxRunes)
			if got != tc.want {
				t.Errorf("Truncate(%q, %d) = %q; want %q", tc.input, tc.maxRunes, got, tc.want)
			}
		})
	}
}
