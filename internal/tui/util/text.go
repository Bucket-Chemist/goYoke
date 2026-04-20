// Package util provides shared utility functions for the goYoke TUI.
// It has no dependency on any TUI, model, or Bubbletea packages so it can be
// imported from any layer of the package hierarchy without creating cycles.
package util

// Truncate returns s truncated to maxRunes runes with "…" appended if
// truncated. It operates on runes, not bytes, so multi-byte UTF-8 characters
// are handled correctly.
//
// If len(s) <= maxRunes the original string is returned unchanged.
// If maxRunes <= 0 an empty string is returned.
func Truncate(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 1 {
		return "…"
	}
	return string(runes[:maxRunes-1]) + "…"
}
