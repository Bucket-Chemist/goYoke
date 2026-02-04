package routing

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	// ConventionsMarker is used to detect if conventions were already injected
	ConventionsMarker = "[CONVENTIONS - AUTO-INJECTED BY gogent-validate]"
	ConventionsEndMarker = "[END CONVENTIONS]"
)

// BuildAugmentedPrompt creates a new prompt with conventions prepended.
// It loads the required rules and conventions based on the agent's context_requirements
// and prepends them to the original prompt in a clearly marked section.
//
// Parameters:
//   - originalPrompt: The original Task prompt from the router
//   - requirements: The agent's context requirements (rules and conventions)
//   - taskFiles: File paths mentioned in the prompt (for conditional convention matching)
//
// Returns the augmented prompt or the original if no requirements or already augmented.
func BuildAugmentedPrompt(originalPrompt string, requirements *ContextRequirements, taskFiles []string) (string, error) {
	// If no requirements, return original
	if requirements == nil || !requirements.HasContextRequirements() {
		return originalPrompt, nil
	}

	// Check if already augmented (prevent double-injection)
	if strings.Contains(originalPrompt, ConventionsMarker) {
		return originalPrompt, nil
	}

	var sections []string
	sections = append(sections, ConventionsMarker)
	sections = append(sections, "")

	// Load and add rules
	for _, rulesFile := range requirements.Rules {
		content, err := LoadRulesContent(rulesFile)
		if err != nil {
			// Log warning but continue - don't fail the whole injection
			fmt.Fprintf(os.Stderr, "[prompt-builder] Warning: Failed to load rules %s: %v\n", rulesFile, err)
			continue
		}
		sections = append(sections, fmt.Sprintf("--- %s ---", rulesFile))
		sections = append(sections, content)
		sections = append(sections, "")
	}

	// Get all applicable conventions
	conventions := requirements.GetAllConventions(taskFiles)

	// Load and add conventions
	for _, convFile := range conventions {
		content, err := LoadConventionContent(convFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[prompt-builder] Warning: Failed to load convention %s: %v\n", convFile, err)
			continue
		}
		sections = append(sections, fmt.Sprintf("--- %s ---", convFile))
		sections = append(sections, content)
		sections = append(sections, "")
	}

	sections = append(sections, ConventionsEndMarker)
	sections = append(sections, "")
	sections = append(sections, "---")
	sections = append(sections, "")
	sections = append(sections, originalPrompt)

	return strings.Join(sections, "\n"), nil
}

// ExtractFilesFromPrompt attempts to extract file paths mentioned in a prompt.
// It looks for common patterns like "/path/to/file.go" or "in src/main.go".
// Returns a slice of detected file paths (may be empty).
func ExtractFilesFromPrompt(prompt string) []string {
	var files []string

	// Pattern for Unix-style paths (including relative)
	pathPattern := regexp.MustCompile(`(?:^|[\s"'(])(/[^\s"'()]+\.[a-zA-Z]+|[a-zA-Z0-9_-]+/[^\s"'()]+\.[a-zA-Z]+)`)

	matches := pathPattern.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		if len(match) > 1 {
			path := strings.TrimSpace(match[1])
			// Filter out obvious non-file paths
			if !strings.HasPrefix(path, "http") && !strings.HasPrefix(path, "//") {
				files = append(files, path)
			}
		}
	}

	return files
}

// StripConventionsFromPrompt removes the auto-injected conventions section.
// Useful for logging or displaying the original prompt without injected content.
func StripConventionsFromPrompt(prompt string) string {
	startIdx := strings.Index(prompt, ConventionsMarker)
	if startIdx == -1 {
		return prompt
	}

	endIdx := strings.Index(prompt, ConventionsEndMarker)
	if endIdx == -1 {
		return prompt
	}

	// Find the separator after END CONVENTIONS
	afterEnd := endIdx + len(ConventionsEndMarker)
	sepIdx := strings.Index(prompt[afterEnd:], "---")
	if sepIdx == -1 {
		return prompt[afterEnd:]
	}

	// Return everything after the separator
	return strings.TrimSpace(prompt[afterEnd+sepIdx+3:])
}
