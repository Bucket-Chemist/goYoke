package routing

import (
	"testing"
)

func TestParseOverrides_ForceTier(t *testing.T) {
	prompt := "--force-tier=opus\n\nAGENT: einstein\n\nAnalyze this problem"
	flags := ParseOverrides(prompt)

	if flags.ForceTier != "opus" {
		t.Errorf("Expected force-tier opus, got: %s", flags.ForceTier)
	}
	if flags.ForceDelegation != "" {
		t.Errorf("Expected no force-delegation, got: %s", flags.ForceDelegation)
	}
	if !flags.HasOverrides() {
		t.Error("Expected HasOverrides() to be true")
	}
}

func TestParseOverrides_ForceDelegation(t *testing.T) {
	prompt := "--force-delegation=sonnet\n\nTask requires reasoning"
	flags := ParseOverrides(prompt)

	if flags.ForceDelegation != "sonnet" {
		t.Errorf("Expected force-delegation sonnet, got: %s", flags.ForceDelegation)
	}
	if flags.ForceTier != "" {
		t.Errorf("Expected no force-tier, got: %s", flags.ForceTier)
	}
	if !flags.HasOverrides() {
		t.Error("Expected HasOverrides() to be true")
	}
}

func TestParseOverrides_Both(t *testing.T) {
	prompt := "--force-tier=haiku --force-delegation=sonnet\n\nSpecial case"
	flags := ParseOverrides(prompt)

	if flags.ForceTier != "haiku" || flags.ForceDelegation != "sonnet" {
		t.Errorf("Expected both flags, got: tier=%s delegation=%s",
			flags.ForceTier, flags.ForceDelegation)
	}
	if !flags.HasOverrides() {
		t.Error("Expected HasOverrides() to be true")
	}
}

func TestParseOverrides_None(t *testing.T) {
	prompt := "AGENT: python-pro\n\nImplement function"
	flags := ParseOverrides(prompt)

	if flags.HasOverrides() {
		t.Error("Expected no overrides")
	}
	if flags.ForceTier != "" {
		t.Errorf("Expected no force-tier, got: %s", flags.ForceTier)
	}
	if flags.ForceDelegation != "" {
		t.Errorf("Expected no force-delegation, got: %s", flags.ForceDelegation)
	}
}

// Test table-driven approach for comprehensive coverage
func TestParseOverrides_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		prompt          string
		expectedTier    string
		expectedDeleg   string
		expectedHas     bool
	}{
		{
			name:          "force-tier at start",
			prompt:        "--force-tier=opus\nSome task",
			expectedTier:  "opus",
			expectedDeleg: "",
			expectedHas:   true,
		},
		{
			name:          "force-tier in middle",
			prompt:        "AGENT: test\n--force-tier=sonnet\nImplement",
			expectedTier:  "sonnet",
			expectedDeleg: "",
			expectedHas:   true,
		},
		{
			name:          "force-tier at end",
			prompt:        "Do the thing\n--force-tier=haiku",
			expectedTier:  "haiku",
			expectedDeleg: "",
			expectedHas:   true,
		},
		{
			name:          "force-delegation at start",
			prompt:        "--force-delegation=haiku\nTask",
			expectedTier:  "",
			expectedDeleg: "haiku",
			expectedHas:   true,
		},
		{
			name:          "force-delegation in middle",
			prompt:        "AGENT: test\n--force-delegation=sonnet\nDo it",
			expectedTier:  "",
			expectedDeleg: "sonnet",
			expectedHas:   true,
		},
		{
			name:          "both flags same line",
			prompt:        "--force-tier=opus --force-delegation=sonnet",
			expectedTier:  "opus",
			expectedDeleg: "sonnet",
			expectedHas:   true,
		},
		{
			name:          "both flags different lines",
			prompt:        "--force-tier=haiku\n\nAGENT: test\n\n--force-delegation=sonnet",
			expectedTier:  "haiku",
			expectedDeleg: "sonnet",
			expectedHas:   true,
		},
		{
			name:          "no flags",
			prompt:        "AGENT: python-pro\n\nImplement feature X",
			expectedTier:  "",
			expectedDeleg: "",
			expectedHas:   false,
		},
		{
			name:          "invalid flag format (no equals)",
			prompt:        "--force-tier opus",
			expectedTier:  "",
			expectedDeleg: "",
			expectedHas:   false,
		},
		{
			name:          "valid and invalid flags",
			prompt:        "--force-tier=sonnet --invalid-flag=test",
			expectedTier:  "sonnet",
			expectedDeleg: "",
			expectedHas:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := ParseOverrides(tt.prompt)

			if flags.ForceTier != tt.expectedTier {
				t.Errorf("ForceTier: expected %q, got %q", tt.expectedTier, flags.ForceTier)
			}
			if flags.ForceDelegation != tt.expectedDeleg {
				t.Errorf("ForceDelegation: expected %q, got %q", tt.expectedDeleg, flags.ForceDelegation)
			}
			if flags.HasOverrides() != tt.expectedHas {
				t.Errorf("HasOverrides: expected %v, got %v", tt.expectedHas, flags.HasOverrides())
			}
		})
	}
}

// Test edge cases
func TestParseOverrides_EdgeCases(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		flags := ParseOverrides("")
		if flags.HasOverrides() {
			t.Error("Expected no overrides for empty string")
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		flags := ParseOverrides("   \n\n  \t  ")
		if flags.HasOverrides() {
			t.Error("Expected no overrides for whitespace")
		}
	})

	t.Run("duplicate force-tier", func(t *testing.T) {
		prompt := "--force-tier=haiku --force-tier=sonnet"
		flags := ParseOverrides(prompt)
		// Should match first occurrence
		if flags.ForceTier != "haiku" {
			t.Errorf("Expected first match 'haiku', got: %s", flags.ForceTier)
		}
	})

	t.Run("duplicate force-delegation", func(t *testing.T) {
		prompt := "--force-delegation=haiku --force-delegation=sonnet"
		flags := ParseOverrides(prompt)
		// Should match first occurrence
		if flags.ForceDelegation != "haiku" {
			t.Errorf("Expected first match 'haiku', got: %s", flags.ForceDelegation)
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		prompt := "--force-tier=OPUS --force-delegation=SONNET"
		flags := ParseOverrides(prompt)
		// Regex \w+ will match uppercase
		if flags.ForceTier != "OPUS" {
			t.Errorf("Expected 'OPUS', got: %s", flags.ForceTier)
		}
		if flags.ForceDelegation != "SONNET" {
			t.Errorf("Expected 'SONNET', got: %s", flags.ForceDelegation)
		}
	})
}
