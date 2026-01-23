package session

import (
	"testing"
	"time"
)

// TestClassifyIntent_Performance verifies classification takes <10ms
func TestClassifyIntent_Performance(t *testing.T) {
	// Test cases covering all categories
	testCases := []struct {
		question string
		response string
	}{
		{"Which model?", "Use sonnet for this"},
		{"Which tool?", "Use Edit not sed"},
		{"Format?", "Be more concise"},
		{"Sequence?", "Always check first"},
		{"Framework?", "We use React"},
		{"Confirm?", "No, I meant the other one"},
		{"Ready?", "Yes"},
		{"Proceed?", "Stop"},
		{"Random?", "Some random text"},
	}

	// Warmup (compile patterns if needed)
	for _, tc := range testCases {
		_ = ClassifyIntent(tc.question, tc.response)
	}

	// Measure performance
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		for _, tc := range testCases {
			_ = ClassifyIntent(tc.question, tc.response)
		}
	}
	elapsed := time.Since(start)

	avgPerCall := elapsed / time.Duration(iterations*len(testCases))

	// Requirement: <10ms per classification
	if avgPerCall > 10*time.Millisecond {
		t.Errorf("Classification too slow: %v per call (requirement: <10ms)", avgPerCall)
	}

	// Log actual performance for reference
	t.Logf("Classification performance: %v per call (%d iterations)", avgPerCall, iterations*len(testCases))
}

// TestExtractKeywords_Performance verifies keyword extraction is fast
func TestExtractKeywords_Performance(t *testing.T) {
	// Test cases with varying complexity
	testCases := []string{
		"Use Edit not sed, and run bash after",
		"Use sonnet not haiku, opus is too expensive",
		"Run pytest and go test, skip jest",
		"Check pkg/session/query.go and main.go and test.py",
		"After git commit, do git push using make build",
	}

	// Warmup
	for _, tc := range testCases {
		_ = ExtractKeywords(tc)
	}

	// Measure performance
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		for _, tc := range testCases {
			_ = ExtractKeywords(tc)
		}
	}
	elapsed := time.Since(start)

	avgPerCall := elapsed / time.Duration(iterations*len(testCases))

	// Requirement: <10ms per extraction (same as classification)
	if avgPerCall > 10*time.Millisecond {
		t.Errorf("Keyword extraction too slow: %v per call (requirement: <10ms)", avgPerCall)
	}

	t.Logf("Keyword extraction performance: %v per call (%d iterations)", avgPerCall, iterations*len(testCases))
}

// BenchmarkClassifyIntent benchmarks classification performance
func BenchmarkClassifyIntent(b *testing.B) {
	question := "Which model should I use?"
	response := "Use sonnet for this task"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyIntent(question, response)
	}
}

// BenchmarkExtractKeywords benchmarks keyword extraction performance
func BenchmarkExtractKeywords(b *testing.B) {
	response := "Use Edit not sed, check pkg/session/query.go, run go test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ExtractKeywords(response)
	}
}
