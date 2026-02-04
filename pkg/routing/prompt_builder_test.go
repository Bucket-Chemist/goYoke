package routing

import (
	"strings"
	"testing"
)

func TestBuildAugmentedPrompt_NoRequirements(t *testing.T) {
	original := "AGENT: go-pro\n\nTASK: Do something"

	result, err := BuildAugmentedPrompt(original, nil, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != original {
		t.Error("Expected original prompt when no requirements")
	}
}

func TestBuildAugmentedPrompt_AlreadyAugmented(t *testing.T) {
	original := ConventionsMarker + "\nsome content\n" + ConventionsEndMarker + "\n---\noriginal prompt"

	result, err := BuildAugmentedPrompt(original, &ContextRequirements{
		Rules: []string{"agent-guidelines.md"},
	}, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != original {
		t.Error("Expected no double-injection")
	}
}

func TestExtractFilesFromPrompt(t *testing.T) {
	prompt := `AGENT: go-pro

TASK: Implement feature in /home/user/project/cmd/main.go
Also update src/utils/helper.go

CONTEXT: See pkg/routing/types.go for reference`

	files := ExtractFilesFromPrompt(prompt)

	if len(files) == 0 {
		t.Error("Expected to find file paths")
	}

	// Should find at least some of these paths
	found := false
	for _, f := range files {
		if strings.Contains(f, "main.go") || strings.Contains(f, "helper.go") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find .go files, got: %v", files)
	}
}

func TestStripConventionsFromPrompt(t *testing.T) {
	augmented := ConventionsMarker + "\n\nconvention content here\n\n" + ConventionsEndMarker + "\n\n---\n\nAGENT: go-pro\n\nTASK: Do thing"

	stripped := StripConventionsFromPrompt(augmented)

	if strings.Contains(stripped, ConventionsMarker) {
		t.Error("Conventions marker still present")
	}

	if !strings.Contains(stripped, "AGENT: go-pro") {
		t.Error("Original prompt content missing")
	}
}
