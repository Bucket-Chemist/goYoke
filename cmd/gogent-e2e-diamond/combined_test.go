package main

import "testing"

func TestNewPair(t *testing.T) {
	tests := []struct {
		name  string
		left  string
		right string
		want  Pair
	}{
		{
			name:  "basic pair",
			left:  "a",
			right: "b",
			want:  Pair{Left: "a", Right: "b"},
		},
		{
			name:  "empty strings",
			left:  "",
			right: "",
			want:  Pair{Left: "", Right: ""},
		},
		{
			name:  "mixed empty and non-empty",
			left:  "hello",
			right: "",
			want:  Pair{Left: "hello", Right: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPair(tt.left, tt.right)
			if got != tt.want {
				t.Errorf("NewPair(%q, %q) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basic string",
			input: "hello",
			want:  "olleh",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single character",
			input: "a",
			want:  "a",
		},
		{
			name:  "palindrome",
			input: "racecar",
			want:  "racecar",
		},
		{
			name:  "unicode characters",
			input: "hello世界",
			want:  "界世olleh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reverse(tt.input)
			if got != tt.want {
				t.Errorf("Reverse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
