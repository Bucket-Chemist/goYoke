package claude_test

// timestamps_test.go covers UX-024: the optional 5-char relative-time gutter
// in the conversation panel.

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/claude"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/model"
)

// ---------------------------------------------------------------------------
// fmtRelativeTime unit tests
// ---------------------------------------------------------------------------

func TestFmtRelativeTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"sub-minute", now.Add(-30 * time.Second), "now  "},
		{"1 min", now.Add(-1 * time.Minute), "1m   "},
		{"5 mins", now.Add(-5 * time.Minute), "5m   "},
		{"59 mins", now.Add(-59 * time.Minute), "59m  "},
		{"1 hour", now.Add(-1 * time.Hour), "1h   "},
		{"2 hours", now.Add(-2 * time.Hour), "2h   "},
		{"23 hours", now.Add(-23 * time.Hour), "23h  "},
		{"1 day", now.Add(-24 * time.Hour), "1d   "},
		{"3 days", now.Add(-72 * time.Hour), "3d   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := claude.FmtRelativeTime(tc.t)
			assert.Equal(t, tc.want, got)
			assert.Len(t, got, 5, "gutter must always be exactly 5 chars")
		})
	}
}

// ---------------------------------------------------------------------------
// Gutter visibility: enabled vs disabled
// ---------------------------------------------------------------------------

// TestTimestampGutter_DefaultOff verifies no gutter is rendered by default.
func TestTimestampGutter_DefaultOff(t *testing.T) {
	m := newPanel()
	m2, _ := m.Update(model.AssistantMsg{Text: "hello"})
	view := stripANSI(m2.View())
	assert.NotContains(t, view, "now  Claude:",
		"timestamp gutter must not appear when timestamps are disabled (default)")
}

// TestTimestampGutter_EnabledShowsGutter verifies the gutter appears at the
// first turn boundary after SetShowTimestamps(true).
func TestTimestampGutter_EnabledShowsGutter(t *testing.T) {
	m := newPanel()
	cmd := m.SetShowTimestamps(true)
	assert.NotNil(t, cmd, "SetShowTimestamps(true) must return a tick Cmd")

	m2, _ := m.Update(model.AssistantMsg{Text: "hello"})
	view := stripANSI(m2.View())
	// Message was just created so fmtRelativeTime returns "now  ".
	assert.Contains(t, view, "now  Claude:",
		"gutter 'now  ' must precede role label when timestamps are enabled")
}

// TestTimestampGutter_DisabledHidesGutter verifies the gutter is absent after
// SetShowTimestamps(false).
func TestTimestampGutter_DisabledHidesGutter(t *testing.T) {
	m := newPanel()
	m.SetShowTimestamps(false) // explicitly off
	m2, _ := m.Update(model.AssistantMsg{Text: "hello"})
	view := stripANSI(m2.View())
	assert.NotContains(t, view, "now  Claude:",
		"gutter must not appear when timestamps are disabled")
}

// TestTimestampGutter_TurnBoundaryOnly verifies the gutter appears on the
// role-label line only, not on every content line.
func TestTimestampGutter_TurnBoundaryOnly(t *testing.T) {
	m := newPanel()
	m.SetShowTimestamps(true)
	// Send a multi-line assistant message.
	m2, _ := m.Update(model.AssistantMsg{Text: "line one\nline two\nline three"})
	view := stripANSI(m2.View())

	lines := strings.Split(view, "\n")
	gutterCount := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "now  ") {
			gutterCount++
		}
	}
	assert.Equal(t, 1, gutterCount,
		"gutter must appear on exactly one line (the role label), not content lines")
}

// ---------------------------------------------------------------------------
// SetShowTimestamps cmd return contract
// ---------------------------------------------------------------------------

// TestSetShowTimestamps_TickCmdContract verifies enabling returns a Cmd and
// disabling returns nil.
func TestSetShowTimestamps_TickCmdContract(t *testing.T) {
	m := newPanel()

	cmd := m.SetShowTimestamps(true)
	assert.NotNil(t, cmd, "enabling timestamps must return a tick Cmd")

	cmd = m.SetShowTimestamps(false)
	assert.Nil(t, cmd, "disabling timestamps must return nil")
}
