package session

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// WeeklyIntentSummary captures intent patterns for a week
type WeeklyIntentSummary struct {
	WeekStart            time.Time                 `json:"week_start"`
	WeekEnd              time.Time                 `json:"week_end"`
	TotalIntents         int                       `json:"total_intents"`
	SessionCount         int                       `json:"session_count"`
	CategoryDistribution map[string]int            `json:"category_distribution"`
	CategoryPercentages  map[string]float64        `json:"category_percentages"`
	RecurringPreferences []RecurringPreference     `json:"recurring_preferences"`
	DriftAlerts          []PreferenceDriftAlert    `json:"drift_alerts,omitempty"`
	// GOgent-041c: Honor rate tracking
	TotalAnalyzed        int                `json:"total_analyzed"`         // Intents with Honored != nil
	TotalHonored         int                `json:"total_honored"`          // Intents with Honored == true
	HonorRatePercent     float64            `json:"honor_rate_percent"`     // Overall honor rate
	HonorRateByCategory  map[string]float64 `json:"honor_rate_by_category"` // Per-category honor rates
}

// RecurringPreference represents a response pattern that appears multiple times
type RecurringPreference struct {
	Category   string   `json:"category"`
	Pattern    string   `json:"pattern"`     // Common response pattern
	Frequency  int      `json:"frequency"`   // Times seen this week
	Keywords   []string `json:"keywords"`    // Common keywords
	SessionIDs []string `json:"session_ids"` // Which sessions
}

// PreferenceDriftAlert represents a change in user preferences week-over-week
type PreferenceDriftAlert struct {
	Type       string `json:"type"`                   // "new", "dropped", "changed"
	Category   string `json:"category"`
	OldPattern string `json:"old_pattern,omitempty"`
	NewPattern string `json:"new_pattern,omitempty"`
	Message    string `json:"message"`
}

// AggregateWeeklyIntents creates a summary of intents for the given week
func AggregateWeeklyIntents(intents []UserIntent, weekStart, weekEnd time.Time) WeeklyIntentSummary {
	summary := WeeklyIntentSummary{
		WeekStart:            weekStart,
		WeekEnd:              weekEnd,
		CategoryDistribution: make(map[string]int),
		CategoryPercentages:  make(map[string]float64),
	}

	// Filter to week and count
	sessionSet := make(map[string]bool)
	var weekIntents []UserIntent

	for _, intent := range intents {
		intentTime := time.Unix(intent.Timestamp, 0)
		if (intentTime.After(weekStart) || intentTime.Equal(weekStart)) &&
		   intentTime.Before(weekEnd) {
			weekIntents = append(weekIntents, intent)
			summary.CategoryDistribution[intent.Category]++
			if intent.SessionID != "" {
				sessionSet[intent.SessionID] = true
			}
		}
	}

	summary.TotalIntents = len(weekIntents)
	summary.SessionCount = len(sessionSet)

	// Calculate percentages
	if summary.TotalIntents > 0 {
		for cat, count := range summary.CategoryDistribution {
			summary.CategoryPercentages[cat] = float64(count) / float64(summary.TotalIntents) * 100
		}
	}

	// GOgent-041c: Calculate honor rates
	summary.HonorRateByCategory = make(map[string]float64)
	categoryHonored := make(map[string]int)
	categoryAnalyzed := make(map[string]int)

	for _, intent := range weekIntents {
		if intent.Honored != nil {
			summary.TotalAnalyzed++
			categoryAnalyzed[intent.Category]++
			if *intent.Honored {
				summary.TotalHonored++
				categoryHonored[intent.Category]++
			}
		}
	}

	// Calculate overall honor rate
	if summary.TotalAnalyzed > 0 {
		summary.HonorRatePercent = float64(summary.TotalHonored) / float64(summary.TotalAnalyzed) * 100
	}

	// Calculate per-category honor rates
	for cat, analyzed := range categoryAnalyzed {
		if analyzed > 0 {
			honored := categoryHonored[cat]
			summary.HonorRateByCategory[cat] = float64(honored) / float64(analyzed) * 100
		}
	}

	// Detect recurring preferences
	summary.RecurringPreferences = DetectRecurringPreferences(weekIntents)

	return summary
}

// DetectRecurringPreferences finds response patterns that appear multiple times
func DetectRecurringPreferences(intents []UserIntent) []RecurringPreference {
	// Group by category + normalized response
	type prefKey struct {
		category string
		pattern  string
	}

	groups := make(map[prefKey][]UserIntent)

	for _, intent := range intents {
		// Only group intents with category set
		if intent.Category == "" {
			continue
		}

		// Normalize response for grouping
		normalized := normalizeResponse(intent.Response)
		if normalized == "" {
			continue
		}

		key := prefKey{intent.Category, normalized}
		groups[key] = append(groups[key], intent)
	}

	// Find recurring (2+ occurrences)
	var recurring []RecurringPreference
	for key, group := range groups {
		if len(group) >= 2 {
			// Collect session IDs
			sessionSet := make(map[string]bool)
			var allKeywords []string
			for _, intent := range group {
				if intent.SessionID != "" {
					sessionSet[intent.SessionID] = true
				}
				allKeywords = append(allKeywords, intent.Keywords...)
			}

			sessions := make([]string, 0, len(sessionSet))
			for sid := range sessionSet {
				sessions = append(sessions, sid)
			}

			recurring = append(recurring, RecurringPreference{
				Category:   key.category,
				Pattern:    group[0].Response, // Use first as representative
				Frequency:  len(group),
				Keywords:   dedupeKeywords(allKeywords),
				SessionIDs: sessions,
			})
		}
	}

	// Sort by frequency descending
	sort.Slice(recurring, func(i, j int) bool {
		return recurring[i].Frequency > recurring[j].Frequency
	})

	// Limit to top 10
	if len(recurring) > 10 {
		recurring = recurring[:10]
	}

	return recurring
}

// DetectPreferenceDrift compares two weekly summaries for changes
func DetectPreferenceDrift(current, previous WeeklyIntentSummary) []PreferenceDriftAlert {
	var alerts []PreferenceDriftAlert

	// Build maps for comparison
	currentPrefs := make(map[string]RecurringPreference)
	for _, pref := range current.RecurringPreferences {
		key := pref.Category + ":" + normalizeResponse(pref.Pattern)
		currentPrefs[key] = pref
	}

	previousPrefs := make(map[string]RecurringPreference)
	for _, pref := range previous.RecurringPreferences {
		key := pref.Category + ":" + normalizeResponse(pref.Pattern)
		previousPrefs[key] = pref
	}

	// Detect new preferences
	for key, pref := range currentPrefs {
		if _, existed := previousPrefs[key]; !existed {
			alerts = append(alerts, PreferenceDriftAlert{
				Type:       "new",
				Category:   pref.Category,
				NewPattern: pref.Pattern,
				Message:    fmt.Sprintf("New %s preference: %q", pref.Category, truncate(pref.Pattern, 50)),
			})
		}
	}

	// Detect dropped preferences
	for key, pref := range previousPrefs {
		if _, exists := currentPrefs[key]; !exists {
			alerts = append(alerts, PreferenceDriftAlert{
				Type:       "dropped",
				Category:   pref.Category,
				OldPattern: pref.Pattern,
				Message:    fmt.Sprintf("Dropped %s preference: %q", pref.Category, truncate(pref.Pattern, 50)),
			})
		}
	}

	return alerts
}

// normalizeResponse creates a canonical form for comparison
func normalizeResponse(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)

	// Remove common punctuation
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "\"", "")
	s = strings.ReplaceAll(s, "'", "")

	// Collapse multiple spaces into single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")

	return strings.TrimSpace(s)
}

// dedupeKeywords removes duplicate keywords while preserving order
func dedupeKeywords(keywords []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if !seen[kw] {
			seen[kw] = true
			result = append(result, kw)
		}
	}
	return result
}

// truncate shortens a string to maxLen characters with ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
