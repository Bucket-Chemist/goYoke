package layout

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBannerModel verifies banner initialization
func TestNewBannerModel(t *testing.T) {
	sessionID := "test-session-123"
	banner := NewBannerModel(sessionID)

	assert.Equal(t, ViewClaude, banner.activeView, "Initial active view should be Claude")
	assert.Equal(t, sessionID, banner.sessionID, "Session ID should be set")
	assert.Equal(t, 0.0, banner.cost, "Initial cost should be zero")
	assert.Equal(t, 0, banner.width, "Initial width should be zero")
}

// TestSetActiveView verifies view switching
func TestSetActiveView(t *testing.T) {
	banner := NewBannerModel("test-session")

	tests := []struct {
		name string
		view View
	}{
		{"claude", ViewClaude},
		{"agents", ViewAgents},
		{"stats", ViewStats},
		{"query", ViewQuery},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner.SetActiveView(tc.view)
			assert.Equal(t, tc.view, banner.activeView, "Active view should be updated")
		})
	}
}

// TestSetCost verifies cost updating
func TestSetCost(t *testing.T) {
	banner := NewBannerModel("test-session")

	tests := []struct {
		name string
		cost float64
	}{
		{"zero", 0.0},
		{"small", 0.05},
		{"medium", 1.23},
		{"large", 42.99},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner.SetCost(tc.cost)
			assert.Equal(t, tc.cost, banner.cost, "Cost should be updated")
		})
	}
}

// TestSetWidth verifies width updating
func TestSetWidth(t *testing.T) {
	banner := NewBannerModel("test-session")

	tests := []struct {
		name  string
		width int
	}{
		{"narrow", 80},
		{"medium", 120},
		{"wide", 200},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner.SetWidth(tc.width)
			assert.Equal(t, tc.width, banner.width, "Width should be updated")
		})
	}
}

// TestTruncateSessionID verifies session ID truncation
func TestTruncateSessionID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short_id",
			input:    "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly_8_chars",
			input:    "12345678",
			expected: "12345678",
		},
		{
			name:     "long_id",
			input:    "very-long-session-id-that-needs-truncation",
			expected: "very-lon",
		},
		{
			name:     "uuid",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400",
		},
		{
			name:     "empty",
			input:    "",
			expected: "--------", // Placeholder for empty ID
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateSessionID(tc.input)
			assert.Equal(t, tc.expected, result, "Truncation should match expected")
			assert.LessOrEqual(t, len(result), 8, "Result should be 8 chars or less")
		})
	}
}

// TestBannerViewActiveHighlight verifies active tab highlighting
func TestBannerViewActiveHighlight(t *testing.T) {
	tests := []struct {
		name       string
		activeView View
		shouldFind []string
	}{
		{
			name:       "claude_active",
			activeView: ViewClaude,
			shouldFind: []string{"[1] Claude", "[2] Agents", "[3] Stats", "[4] Query"},
		},
		{
			name:       "agents_active",
			activeView: ViewAgents,
			shouldFind: []string{"[1] Claude", "[2] Agents", "[3] Stats", "[4] Query"},
		},
		{
			name:       "stats_active",
			activeView: ViewStats,
			shouldFind: []string{"[1] Claude", "[2] Agents", "[3] Stats", "[4] Query"},
		},
		{
			name:       "query_active",
			activeView: ViewQuery,
			shouldFind: []string{"[1] Claude", "[2] Agents", "[3] Stats", "[4] Query"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner := NewBannerModel("test123")
			banner.SetActiveView(tc.activeView)
			banner.SetWidth(120)
			banner.SetCost(0.42)

			view := banner.View()

			// All tabs should be present (ANSI codes may be present)
			for _, tab := range tc.shouldFind {
				assert.Contains(t, view, tab, "View should contain tab: %s", tab)
			}

			// Session info should be present
			assert.Contains(t, view, "Session: test123", "View should contain session ID")
			assert.Contains(t, view, "Cost: $0.42", "View should contain cost")
		})
	}
}

// TestBannerViewSessionInfo verifies session info rendering
func TestBannerViewSessionInfo(t *testing.T) {
	tests := []struct {
		name            string
		sessionID       string
		cost            float64
		expectedSession string
		expectedCost    string
	}{
		{
			name:            "short_session",
			sessionID:       "abc123",
			cost:            0.05,
			expectedSession: "Session: abc123",
			expectedCost:    "Cost: $0.05",
		},
		{
			name:            "long_session_truncated",
			sessionID:       "very-long-session-id",
			cost:            12.34,
			expectedSession: "Session: very-lon",
			expectedCost:    "Cost: $12.34",
		},
		{
			name:            "zero_cost",
			sessionID:       "test",
			cost:            0.0,
			expectedSession: "Session: test",
			expectedCost:    "Cost: $0.00",
		},
		{
			name:            "large_cost",
			sessionID:       "prod",
			cost:            999.99,
			expectedSession: "Session: prod",
			expectedCost:    "Cost: $999.99",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner := NewBannerModel(tc.sessionID)
			banner.SetCost(tc.cost)
			banner.SetWidth(120)

			view := banner.View()

			assert.Contains(t, view, tc.expectedSession, "View should contain expected session")
			assert.Contains(t, view, tc.expectedCost, "View should contain expected cost")
		})
	}
}

// TestBannerViewWidth verifies width handling
func TestBannerViewWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{"narrow", 80},
		{"medium", 120},
		{"wide", 200},
		{"very_wide", 300},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			banner := NewBannerModel("test-session")
			banner.SetActiveView(ViewClaude)
			banner.SetCost(0.42)
			banner.SetWidth(tc.width)

			view := banner.View()

			// View should not be empty
			assert.NotEmpty(t, view, "View should render content")

			// All tabs should be present
			assert.Contains(t, view, "[1] Claude", "View should contain Claude tab")
			assert.Contains(t, view, "[2] Agents", "View should contain Agents tab")
			assert.Contains(t, view, "[3] Stats", "View should contain Stats tab")
			assert.Contains(t, view, "[4] Query", "View should contain Query tab")

			// Session info should be present
			assert.Contains(t, view, "Session:", "View should contain session label")
			assert.Contains(t, view, "Cost:", "View should contain cost label")
		})
	}
}

// TestBannerViewPadding verifies padding calculation
func TestBannerViewPadding(t *testing.T) {
	banner := NewBannerModel("test")
	banner.SetActiveView(ViewClaude)
	banner.SetCost(1.23)
	banner.SetWidth(120)

	view := banner.View()

	// Should contain spacing between tabs and session info
	// The exact amount of spaces depends on ANSI codes, but there should be multiple
	spaceCount := strings.Count(view, "  ")
	assert.Greater(t, spaceCount, 1, "View should contain padding between sections")
}

// TestBannerViewRendering verifies basic rendering smoke test
func TestBannerViewRendering(t *testing.T) {
	banner := NewBannerModel("abc123def")
	banner.SetActiveView(ViewAgents)
	banner.SetCost(5.67)
	banner.SetWidth(150)

	view := banner.View()

	// Should not panic and should produce output
	require.NotEmpty(t, view, "View should render content")

	// All expected components should be present
	expectedComponents := []string{
		"[1] Claude",
		"[2] Agents",
		"[3] Stats",
		"[4] Query",
		"Session: abc123de", // Truncated
		"Cost: $5.67",
	}

	for _, component := range expectedComponents {
		assert.Contains(t, view, component, "View should contain: %s", component)
	}
}

// TestBannerViewEnumValues verifies all View enum values work
func TestBannerViewEnumValues(t *testing.T) {
	views := []View{ViewClaude, ViewAgents, ViewStats, ViewQuery}

	for _, view := range views {
		t.Run(view.String(), func(t *testing.T) {
			banner := NewBannerModel("test")
			banner.SetActiveView(view)
			banner.SetWidth(100)

			rendered := banner.View()
			assert.NotEmpty(t, rendered, "View should render for %v", view)
		})
	}
}

// String returns string representation of View (for testing)
func (v View) String() string {
	switch v {
	case ViewClaude:
		return "Claude"
	case ViewAgents:
		return "Agents"
	case ViewStats:
		return "Stats"
	case ViewQuery:
		return "Query"
	default:
		return "Unknown"
	}
}

// BenchmarkBannerView benchmarks banner rendering performance
func BenchmarkBannerView(b *testing.B) {
	banner := NewBannerModel("test-session-123")
	banner.SetActiveView(ViewClaude)
	banner.SetCost(1.23)
	banner.SetWidth(120)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = banner.View()
	}
}

// BenchmarkTruncateSessionID benchmarks session ID truncation
func BenchmarkTruncateSessionID(b *testing.B) {
	longID := "very-long-session-id-that-needs-truncation-for-display"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = truncateSessionID(longID)
	}
}
