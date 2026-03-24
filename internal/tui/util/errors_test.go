package util

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func allThemes() []struct {
	name  string
	theme config.Theme
} {
	return []struct {
		name  string
		theme config.Theme
	}{
		{"Dark", config.NewTheme(config.ThemeDark)},
		{"Light", config.NewTheme(config.ThemeLight)},
		{"HighContrast", config.NewTheme(config.ThemeHighContrast)},
	}
}

// ---------------------------------------------------------------------------
// FormatError
// ---------------------------------------------------------------------------

func TestFormatError_AllFields(t *testing.T) {
	t.Parallel()

	theme := config.DefaultTheme()
	ed := ErrorDisplay{
		Category:   "Connection",
		Message:    "dial tcp: connection refused",
		Suggestion: "Check that the server is running",
		Timestamp:  time.Now(),
	}

	got := FormatError(ed, theme)

	if !strings.Contains(got, "Connection") {
		t.Errorf("output missing category; got: %q", got)
	}
	if !strings.Contains(got, ed.Message) {
		t.Errorf("output missing message; got: %q", got)
	}
	if !strings.Contains(got, "Suggestion") {
		t.Errorf("output missing suggestion label; got: %q", got)
	}
	if !strings.Contains(got, ed.Suggestion) {
		t.Errorf("output missing suggestion text; got: %q", got)
	}

	// Icon must appear somewhere in the raw string (may be ANSI-wrapped).
	icon := theme.Icons().Error
	if !strings.Contains(got, icon) {
		t.Errorf("output missing error icon %q; got: %q", icon, got)
	}
}

func TestFormatError_NoSuggestion(t *testing.T) {
	t.Parallel()

	theme := config.DefaultTheme()
	ed := ErrorDisplay{
		Category:  "Timeout",
		Message:   "deadline exceeded after 30s",
		Timestamp: time.Now(),
	}

	got := FormatError(ed, theme)

	if strings.Contains(got, "Suggestion") {
		t.Errorf("output should not contain 'Suggestion' when field is empty; got: %q", got)
	}
	// Must still contain category and message.
	if !strings.Contains(got, "Timeout") {
		t.Errorf("output missing category; got: %q", got)
	}
	if !strings.Contains(got, ed.Message) {
		t.Errorf("output missing message; got: %q", got)
	}
}

func TestFormatError_AllThemeVariants(t *testing.T) {
	t.Parallel()

	ed := ErrorDisplay{
		Category:  "Permission",
		Message:   "access denied",
		Timestamp: time.Now(),
	}

	for _, tc := range allThemes() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := FormatError(ed, tc.theme)
			if got == "" {
				t.Errorf("FormatError returned empty string for theme %s", tc.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatWarning
// ---------------------------------------------------------------------------

func TestFormatWarning_RendersNonEmpty(t *testing.T) {
	t.Parallel()

	theme := config.DefaultTheme()
	got := FormatWarning("RateLimit", "approaching quota", theme)

	if got == "" {
		t.Fatal("FormatWarning returned empty string")
	}
	if !strings.Contains(got, "RateLimit") {
		t.Errorf("output missing category; got: %q", got)
	}

	icon := theme.Icons().Warning
	if !strings.Contains(got, icon) {
		t.Errorf("output missing warning icon %q; got: %q", icon, got)
	}
}

func TestFormatWarning_AllThemeVariants(t *testing.T) {
	t.Parallel()

	for _, tc := range allThemes() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := FormatWarning("Memory", "heap usage high", tc.theme)
			if got == "" {
				t.Errorf("FormatWarning returned empty string for theme %s", tc.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ClassifyError
// ---------------------------------------------------------------------------

func TestClassifyError_KnownPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"connection refused", "dial tcp 127.0.0.1:8080: connect: connection refused", "Connection"},
		{"no such host", "lookup example.invalid: no such host", "Connection"},
		{"dial tcp", "dial tcp: lookup foo.bar", "Connection"},
		{"permission denied", "open /etc/shadow: permission denied", "Permission"},
		{"access denied", "access denied to resource", "Permission"},
		{"timeout", "request timed out: timeout waiting for response", "Timeout"},
		{"deadline exceeded", "context deadline exceeded", "Timeout"},
		{"rate limit", "rate limit exceeded", "Rate Limit"},
		{"429", "HTTP 429: too many calls", "Rate Limit"},
		{"too many requests", "too many requests, slow down", "Rate Limit"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyError(errors.New(tc.errMsg))
			if got != tc.expected {
				t.Errorf("ClassifyError(%q) = %q; want %q", tc.errMsg, got, tc.expected)
			}
		})
	}
}

func TestClassifyError_UnknownPattern(t *testing.T) {
	t.Parallel()

	got := ClassifyError(errors.New("some completely unknown situation occurred"))
	if got != "Error" {
		t.Errorf("expected 'Error' for unknown pattern; got %q", got)
	}
}

func TestClassifyError_NilError(t *testing.T) {
	t.Parallel()

	got := ClassifyError(nil)
	if got != "Error" {
		t.Errorf("expected 'Error' for nil; got %q", got)
	}
}

func TestClassifyError_CaseInsensitive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errMsg   string
		expected string
	}{
		{"upper CONNECTION REFUSED", "CONNECTION REFUSED", "Connection"},
		{"upper PERMISSION DENIED", "PERMISSION DENIED", "Permission"},
		{"upper TIMEOUT", "TIMEOUT", "Timeout"},
		{"upper DEADLINE EXCEEDED", "DEADLINE EXCEEDED", "Timeout"},
		{"upper RATE LIMIT", "RATE LIMIT EXCEEDED", "Rate Limit"},
		{"upper TOO MANY REQUESTS", "TOO MANY REQUESTS", "Rate Limit"},
		{"mixed Dial", "Dial tcp failed", "Connection"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ClassifyError(errors.New(tc.errMsg))
			if got != tc.expected {
				t.Errorf("ClassifyError(%q) = %q; want %q", tc.errMsg, got, tc.expected)
			}
		})
	}
}
