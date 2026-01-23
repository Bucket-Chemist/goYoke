package session

import (
	"testing"
	"time"
)

func TestAggregateWeeklyIntents(t *testing.T) {
	weekStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	weekEnd := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)

	intents := []UserIntent{
		{
			Timestamp:  weekStart.Unix() + 3600, // Within week
			Category:   "routing",
			Response:   "Use sonnet for implementation",
			SessionID:  "session1",
			Keywords:   []string{"sonnet", "implementation"},
		},
		{
			Timestamp:  weekStart.Unix() + 7200, // Within week
			Category:   "routing",
			Response:   "use sonnet for implementation", // Same as above (normalized)
			SessionID:  "session2",
			Keywords:   []string{"sonnet"},
		},
		{
			Timestamp:  weekStart.Unix() + 10800, // Within week
			Category:   "tooling",
			Response:   "Prefer Edit over sed",
			SessionID:  "session1",
			Keywords:   []string{"edit", "sed"},
		},
		{
			Timestamp:  weekEnd.Unix() + 3600, // After week
			Category:   "routing",
			Response:   "Should not appear",
			SessionID:  "session3",
		},
		{
			Timestamp:  weekStart.Unix() - 3600, // Before week
			Category:   "routing",
			Response:   "Should not appear either",
			SessionID:  "session4",
		},
	}

	summary := AggregateWeeklyIntents(intents, weekStart, weekEnd)

	// Verify basic counts
	if summary.TotalIntents != 3 {
		t.Errorf("Expected 3 total intents, got: %d", summary.TotalIntents)
	}

	if summary.SessionCount != 2 {
		t.Errorf("Expected 2 sessions, got: %d", summary.SessionCount)
	}

	// Verify category distribution
	if summary.CategoryDistribution["routing"] != 2 {
		t.Errorf("Expected 2 routing intents, got: %d", summary.CategoryDistribution["routing"])
	}

	if summary.CategoryDistribution["tooling"] != 1 {
		t.Errorf("Expected 1 tooling intent, got: %d", summary.CategoryDistribution["tooling"])
	}

	// Verify percentages
	routingPct := summary.CategoryPercentages["routing"]
	if routingPct < 66.0 || routingPct > 67.0 {
		t.Errorf("Expected routing percentage ~66.67%%, got: %.2f", routingPct)
	}

	toolingPct := summary.CategoryPercentages["tooling"]
	if toolingPct < 33.0 || toolingPct > 34.0 {
		t.Errorf("Expected tooling percentage ~33.33%%, got: %.2f", toolingPct)
	}

	// Verify recurring preferences detected
	if len(summary.RecurringPreferences) != 1 {
		t.Errorf("Expected 1 recurring preference, got: %d", len(summary.RecurringPreferences))
	}

	if len(summary.RecurringPreferences) > 0 {
		pref := summary.RecurringPreferences[0]
		if pref.Category != "routing" {
			t.Errorf("Expected recurring preference category 'routing', got: %s", pref.Category)
		}
		if pref.Frequency != 2 {
			t.Errorf("Expected frequency 2, got: %d", pref.Frequency)
		}
	}
}

func TestAggregateWeeklyIntents_Empty(t *testing.T) {
	weekStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	weekEnd := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)

	summary := AggregateWeeklyIntents([]UserIntent{}, weekStart, weekEnd)

	if summary.TotalIntents != 0 {
		t.Errorf("Expected 0 intents, got: %d", summary.TotalIntents)
	}

	if len(summary.CategoryDistribution) != 0 {
		t.Errorf("Expected empty category distribution, got: %d entries", len(summary.CategoryDistribution))
	}

	if len(summary.RecurringPreferences) != 0 {
		t.Errorf("Expected no recurring preferences, got: %d", len(summary.RecurringPreferences))
	}
}

func TestDetectRecurringPreferences(t *testing.T) {
	intents := []UserIntent{
		{Category: "routing", Response: "Use sonnet for implementation", Keywords: []string{"sonnet"}, SessionID: "s1"},
		{Category: "routing", Response: "use sonnet for implementation", Keywords: []string{"sonnet"}, SessionID: "s2"}, // Normalized match
		{Category: "routing", Response: "Use haiku for search", Keywords: []string{"haiku"}, SessionID: "s1"},
		{Category: "tooling", Response: "Prefer Edit", Keywords: []string{"edit"}, SessionID: "s3"},
		{Category: "tooling", Response: "prefer edit", Keywords: []string{"edit"}, SessionID: "s4"}, // Normalized match
		{Category: "workflow", Response: "Run tests after edit", Keywords: []string{"test"}, SessionID: "s1"}, // Only once
	}

	recurring := DetectRecurringPreferences(intents)

	// Should find 2 recurring patterns (routing sonnet, tooling edit)
	if len(recurring) != 2 {
		t.Errorf("Expected 2 recurring preferences, got: %d", len(recurring))
	}

	// Verify sorted by frequency
	if len(recurring) >= 2 {
		if recurring[0].Frequency < recurring[1].Frequency {
			t.Error("Expected recurring preferences sorted by frequency descending")
		}
	}

	// Verify first preference
	if len(recurring) > 0 {
		pref := recurring[0]
		if pref.Category != "routing" && pref.Category != "tooling" {
			t.Errorf("Unexpected category for top preference: %s", pref.Category)
		}
		if pref.Frequency != 2 {
			t.Errorf("Expected frequency 2, got: %d", pref.Frequency)
		}
		if len(pref.SessionIDs) != 2 {
			t.Errorf("Expected 2 session IDs, got: %d", len(pref.SessionIDs))
		}
		if len(pref.Keywords) == 0 {
			t.Error("Expected keywords to be populated")
		}
	}
}

func TestDetectRecurringPreferences_LimitTop10(t *testing.T) {
	// Create 15 recurring patterns
	var intents []UserIntent
	for i := 0; i < 15; i++ {
		// Each pattern appears twice
		response := "Pattern " + string(rune('A'+i))
		intents = append(intents,
			UserIntent{Category: "test", Response: response, SessionID: "s1"},
			UserIntent{Category: "test", Response: response, SessionID: "s2"},
		)
	}

	recurring := DetectRecurringPreferences(intents)

	// Should limit to top 10
	if len(recurring) != 10 {
		t.Errorf("Expected 10 recurring preferences (limit), got: %d", len(recurring))
	}
}

func TestDetectRecurringPreferences_IgnoreNoCategoryOrResponse(t *testing.T) {
	intents := []UserIntent{
		{Category: "", Response: "No category", SessionID: "s1"},
		{Category: "", Response: "No category", SessionID: "s2"},
		{Category: "test", Response: "", SessionID: "s3"},
		{Category: "test", Response: "", SessionID: "s4"},
		{Category: "valid", Response: "Valid response", SessionID: "s1"},
		{Category: "valid", Response: "Valid response", SessionID: "s2"},
	}

	recurring := DetectRecurringPreferences(intents)

	// Should only find the valid pattern
	if len(recurring) != 1 {
		t.Errorf("Expected 1 recurring preference, got: %d", len(recurring))
	}

	if len(recurring) > 0 && recurring[0].Category != "valid" {
		t.Errorf("Expected category 'valid', got: %s", recurring[0].Category)
	}
}

func TestDetectPreferenceDrift(t *testing.T) {
	current := WeeklyIntentSummary{
		RecurringPreferences: []RecurringPreference{
			{Category: "routing", Pattern: "Use sonnet", Frequency: 3},
			{Category: "tooling", Pattern: "Prefer Edit", Frequency: 2},
		},
	}

	previous := WeeklyIntentSummary{
		RecurringPreferences: []RecurringPreference{
			{Category: "routing", Pattern: "Use haiku", Frequency: 4}, // Dropped
			{Category: "tooling", Pattern: "Prefer Edit", Frequency: 2}, // Same (no drift)
		},
	}

	alerts := DetectPreferenceDrift(current, previous)

	// Should detect: 1 new (sonnet), 1 dropped (haiku)
	if len(alerts) != 2 {
		t.Fatalf("Expected 2 drift alerts, got: %d", len(alerts))
	}

	// Find new and dropped alerts
	var newAlert, droppedAlert *PreferenceDriftAlert
	for i := range alerts {
		if alerts[i].Type == "new" {
			newAlert = &alerts[i]
		} else if alerts[i].Type == "dropped" {
			droppedAlert = &alerts[i]
		}
	}

	if newAlert == nil {
		t.Error("Expected to find 'new' drift alert")
	} else {
		if newAlert.Category != "routing" {
			t.Errorf("Expected new alert for routing, got: %s", newAlert.Category)
		}
		if newAlert.NewPattern != "Use sonnet" {
			t.Errorf("Expected new pattern 'Use sonnet', got: %s", newAlert.NewPattern)
		}
	}

	if droppedAlert == nil {
		t.Error("Expected to find 'dropped' drift alert")
	} else {
		if droppedAlert.Category != "routing" {
			t.Errorf("Expected dropped alert for routing, got: %s", droppedAlert.Category)
		}
		if droppedAlert.OldPattern != "Use haiku" {
			t.Errorf("Expected old pattern 'Use haiku', got: %s", droppedAlert.OldPattern)
		}
	}
}

func TestDetectPreferenceDrift_NoDrift(t *testing.T) {
	summary := WeeklyIntentSummary{
		RecurringPreferences: []RecurringPreference{
			{Category: "routing", Pattern: "Use sonnet", Frequency: 3},
		},
	}

	// Same as current
	previous := WeeklyIntentSummary{
		RecurringPreferences: []RecurringPreference{
			{Category: "routing", Pattern: "Use sonnet", Frequency: 2},
		},
	}

	alerts := DetectPreferenceDrift(summary, previous)

	// No drift expected
	if len(alerts) != 0 {
		t.Errorf("Expected no drift alerts, got: %d", len(alerts))
	}
}

func TestNormalizeResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Use sonnet for implementation", "use sonnet for implementation"},
		{"Use Sonnet For Implementation", "use sonnet for implementation"},
		{"  Use   sonnet   ", "use sonnet"},
		{"Use sonnet, always!", "use sonnet always"},
		{"Use sonnet.", "use sonnet"},
		{"Use \"sonnet\" please?", "use sonnet please"},
		{"Multiple   spaces   collapsed", "multiple spaces collapsed"},
		{"", ""},
	}

	for _, test := range tests {
		result := normalizeResponse(test.input)
		if result != test.expected {
			t.Errorf("normalizeResponse(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestDedupeKeywords(t *testing.T) {
	keywords := []string{"sonnet", "implementation", "sonnet", "test", "implementation", ""}
	result := dedupeKeywords(keywords)

	// Should keep unique keywords, remove empty
	expected := 3 // sonnet, implementation, test
	if len(result) != expected {
		t.Errorf("Expected %d unique keywords, got: %d", expected, len(result))
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, kw := range result {
		if kw == "" {
			t.Error("Empty keyword should be removed")
		}
		if seen[kw] {
			t.Errorf("Duplicate keyword found: %s", kw)
		}
		seen[kw] = true
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input   string
		maxLen  int
		expected string
	}{
		{"Short text", 20, "Short text"},
		{"This is a longer text", 10, "This is..."},
		{"Exact", 5, "Exact"},
		{"TooLong", 5, "To..."},
		{"", 10, ""},
		{"Tiny", 3, "Tin"},
	}

	for _, test := range tests {
		result := truncate(test.input, test.maxLen)
		if result != test.expected {
			t.Errorf("truncate(%q, %d) = %q, expected %q", test.input, test.maxLen, result, test.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{7, 7, 7},
		{0, 10, 0},
		{-5, 5, -5},
	}

	for _, test := range tests {
		result := min(test.a, test.b)
		if result != test.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

// Integration test with mock weekly data
func TestWeeklyIntentSummary_Integration(t *testing.T) {
	weekStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	weekEnd := time.Date(2026, 1, 22, 0, 0, 0, 0, time.UTC)

	// Create realistic mock data
	intents := []UserIntent{
		{Timestamp: weekStart.Unix() + 1000, Category: "approval", Response: "yes", SessionID: "s1", Keywords: []string{"yes"}},
		{Timestamp: weekStart.Unix() + 2000, Category: "approval", Response: "yes", SessionID: "s2", Keywords: []string{"yes"}},
		{Timestamp: weekStart.Unix() + 3000, Category: "approval", Response: "yes", SessionID: "s3", Keywords: []string{"yes"}},
		{Timestamp: weekStart.Unix() + 4000, Category: "domain", Response: "Use pytest not unittest", SessionID: "s1", Keywords: []string{"pytest", "unittest"}},
		{Timestamp: weekStart.Unix() + 5000, Category: "domain", Response: "use pytest not unittest", SessionID: "s2", Keywords: []string{"pytest"}},
		{Timestamp: weekStart.Unix() + 6000, Category: "workflow", Response: "Run tests after edit", SessionID: "s1", Keywords: []string{"test", "edit"}},
		{Timestamp: weekStart.Unix() + 7000, Category: "workflow", Response: "run tests after edit", SessionID: "s3", Keywords: []string{"test"}},
		{Timestamp: weekStart.Unix() + 8000, Category: "tooling", Response: "Prefer Edit over sed", SessionID: "s2", Keywords: []string{"edit", "sed"}},
	}

	summary := AggregateWeeklyIntents(intents, weekStart, weekEnd)

	// Verify statistics
	if summary.TotalIntents != 8 {
		t.Errorf("Expected 8 total intents, got: %d", summary.TotalIntents)
	}

	if summary.SessionCount != 3 {
		t.Errorf("Expected 3 unique sessions, got: %d", summary.SessionCount)
	}

	// Verify category distribution
	if summary.CategoryDistribution["approval"] != 3 {
		t.Errorf("Expected 3 approval intents, got: %d", summary.CategoryDistribution["approval"])
	}

	// Verify recurring preferences
	if len(summary.RecurringPreferences) < 3 {
		t.Errorf("Expected at least 3 recurring preferences, got: %d", len(summary.RecurringPreferences))
	}

	// Most frequent should be "yes" (3 occurrences)
	topPref := summary.RecurringPreferences[0]
	if topPref.Frequency != 3 {
		t.Errorf("Expected top preference frequency 3, got: %d", topPref.Frequency)
	}

	// Verify keywords are deduplicated
	if topPref.Category == "approval" {
		if len(topPref.Keywords) != 1 || topPref.Keywords[0] != "yes" {
			t.Errorf("Expected single keyword 'yes', got: %v", topPref.Keywords)
		}
	}
}
