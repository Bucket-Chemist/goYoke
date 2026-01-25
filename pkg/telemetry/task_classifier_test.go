package telemetry

import "testing"

func TestClassifyTask_Implementation(t *testing.T) {
	tests := []struct {
		desc     string
		wantType string
	}{
		{"implement user authentication", "implementation"},
		{"create a new API endpoint", "implementation"},
		{"add logging to the service", "implementation"},
		{"build the CLI binary", "implementation"},
	}
	for _, tc := range tests {
		taskType, _ := ClassifyTask(tc.desc)
		if taskType != tc.wantType {
			t.Errorf("ClassifyTask(%q) type = %q, want %q", tc.desc, taskType, tc.wantType)
		}
	}
}

func TestClassifyTask_Search(t *testing.T) {
	taskType, _ := ClassifyTask("find all files that handle authentication")
	if taskType != "search" {
		t.Errorf("Expected 'search', got %q", taskType)
	}
}

func TestClassifyTask_Domain(t *testing.T) {
	_, domain := ClassifyTask("implement python logging")
	if domain != "python" {
		t.Errorf("Expected 'python', got %q", domain)
	}

	_, domain = ClassifyTask("add go test coverage")
	if domain != "go" {
		t.Errorf("Expected 'go', got %q", domain)
	}
}

func TestClassifyTask_Understanding(t *testing.T) {
	taskType, _ := ClassifyTask("summarize the key points from this document")
	if taskType != "document_understanding" {
		t.Errorf("Expected 'document_understanding', got %q", taskType)
	}

	taskType, _ = ClassifyTask("how does the authentication system work")
	if taskType != "codebase_understanding" {
		t.Errorf("Expected 'codebase_understanding', got %q", taskType)
	}
}

func TestClassifyTask_Accuracy(t *testing.T) {
	// Sample of 20 descriptions with known correct labels
	samples := []struct {
		desc       string
		expectType string
	}{
		{"implement feature X", "implementation"},
		{"find where errors are handled", "search"},
		{"fix the login bug", "debug"},
		{"refactor the auth module", "refactor"},
		{"review this pull request", "review"},
		{"write unit tests for service", "test"},
		{"summarize this architectural doc", "document_understanding"},
		{"explain how routing works", "codebase_understanding"},
		{"combine all the findings", "synthesis"},
		{"create API documentation", "documentation"},
		{"build the docker image", "implementation"},
		{"locate config files", "search"},
		{"debug memory leak", "debug"},
		{"clean up unused code", "refactor"},
		{"audit the security implementation", "review"},
		{"add test coverage", "test"},
		{"extract key points from spec", "document_understanding"},
		{"trace through the system", "codebase_understanding"},
		{"consolidate the results", "synthesis"},
		{"update the readme", "documentation"},
	}

	correct := 0
	for _, s := range samples {
		taskType, _ := ClassifyTask(s.desc)
		if taskType == s.expectType {
			correct++
		} else {
			t.Logf("MISS: %q -> got %q, expected %q", s.desc, taskType, s.expectType)
		}
	}

	accuracy := float64(correct) / float64(len(samples))
	if accuracy < 0.85 {
		t.Errorf("Classification accuracy %.2f < 0.85 threshold (correct: %d/%d)", accuracy, correct, len(samples))
	}
}

func TestTaskTypeLabels(t *testing.T) {
	labels := TaskTypeLabels()
	if len(labels) != 10 {
		t.Errorf("Expected 10 task type labels, got %d", len(labels))
	}

	expectedLabels := map[string]bool{
		"implementation":         true,
		"search":                 true,
		"documentation":          true,
		"debug":                  true,
		"refactor":               true,
		"review":                 true,
		"test":                   true,
		"document_understanding": true,
		"codebase_understanding": true,
		"synthesis":              true,
	}

	for _, label := range labels {
		if !expectedLabels[label] {
			t.Errorf("Unexpected label in TaskTypeLabels: %s", label)
		}
	}
}

func TestTaskDomainLabels(t *testing.T) {
	labels := TaskDomainLabels()
	if len(labels) != 6 {
		t.Errorf("Expected 6 domain labels, got %d", len(labels))
	}

	expectedLabels := map[string]bool{
		"python":         true,
		"go":             true,
		"r":              true,
		"javascript":     true,
		"infrastructure": true,
		"documentation":  true,
	}

	for _, label := range labels {
		if !expectedLabels[label] {
			t.Errorf("Unexpected label in TaskDomainLabels: %s", label)
		}
	}
}

func TestClassifyTask_UnknownTypeAndDomain(t *testing.T) {
	taskType, taskDomain := ClassifyTask("do something vague and unclear")
	if taskType != "unknown" {
		t.Errorf("Expected 'unknown' task type, got %q", taskType)
	}
	if taskDomain != "unknown" {
		t.Errorf("Expected 'unknown' task domain, got %q", taskDomain)
	}
}
