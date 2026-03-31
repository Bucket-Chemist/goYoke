package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
)

const (
	// IdentityMarker prevents double-injection of agent identity
	IdentityMarker    = "[AGENT IDENTITY - AUTO-INJECTED]"
	IdentityEndMarker = "[END AGENT IDENTITY]"

	// SessionMarker prevents double-injection of session context
	SessionMarker    = "[SESSION CONTEXT]"
	SessionEndMarker = "[END SESSION CONTEXT]"
)

// LoadAgentIdentity loads the markdown body (post-frontmatter) from
// ~/.claude/agents/{agentID}/{agentID}.md
//
// Returns empty string without error if file doesn't exist.
// Results are cached in conventionCache with "identity:" prefix.
func LoadAgentIdentity(agentID string) (string, error) {
	if agentID == "" {
		return "", nil
	}

	cacheKey := "identity:" + agentID

	// Check cache
	cacheMutex.RLock()
	if content, ok := conventionCache[cacheKey]; ok {
		cacheMutex.RUnlock()
		return content, nil
	}
	cacheMutex.RUnlock()

	// Load from disk
	configDir, err := GetClaudeConfigDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(configDir, "agents", agentID, agentID+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache empty result to avoid repeated disk checks
			cacheMutex.Lock()
			conventionCache[cacheKey] = ""
			cacheMutex.Unlock()
			return "", nil
		}
		return "", fmt.Errorf("read agent identity %s: %w", agentID, err)
	}

	body := StripYAMLFrontmatter(string(data))

	// Cache the result
	cacheMutex.Lock()
	conventionCache[cacheKey] = body
	cacheMutex.Unlock()

	return body, nil
}

// StripYAMLFrontmatter removes YAML frontmatter (between --- delimiters)
// from a markdown file, returning only the body content.
// No YAML parsing — pure string processing.
func StripYAMLFrontmatter(content string) string {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return content // No frontmatter
	}

	// Find the opening ---
	openIdx := strings.Index(content, "---")
	rest := content[openIdx+3:]

	// Find closing ---
	closeIdx := strings.Index(rest, "\n---")
	if closeIdx == -1 {
		return content // Malformed, return as-is
	}

	// Return everything after closing --- line
	afterClose := rest[closeIdx+4:]

	// Skip the rest of the --- line (in case of "---\n" vs "---extra")
	if nlIdx := strings.Index(afterClose, "\n"); nlIdx >= 0 {
		afterClose = afterClose[nlIdx+1:]
	}

	return strings.TrimLeft(afterClose, "\n")
}

// GetSessionDir reads the current session directory path.
// First checks GOGENT_SESSION_DIR (set directly by team-run for spawned agents),
// then falls back to reading the current-session marker file using project dir resolution:
// GOGENT_PROJECT_ROOT → GOGENT_PROJECT_DIR → CLAUDE_PROJECT_DIR → os.Getwd().
// Returns empty string if unavailable.
//
// Note: Cannot import pkg/session (circular dependency via sharp_edge_utils.go),
// so env var resolution is inlined here.
func GetSessionDir() string {
	// Direct path: team-run sets this for spawned CLI processes
	if dir := os.Getenv("GOGENT_SESSION_DIR"); dir != "" {
		return dir
	}

	// File-based path: read from current-session marker
	projectDir := os.Getenv("GOGENT_PROJECT_ROOT")
	if projectDir == "" {
		projectDir = os.Getenv("GOGENT_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	if projectDir == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(config.RuntimeDir(projectDir), "current-session"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[identity-loader] current-session not found at %s/.gogent/current-session\n", projectDir)
		return ""
	}
	return strings.TrimSpace(string(data))
}

// BuildFullAgentContext builds complete agent context: identity + rules + conventions.
// Unified entry point for both Task() (gogent-validate) and team-run (envelope.go) paths.
//
// Injection order:
//  1. Agent identity (from ~/.claude/agents/{agentID}/{agentID}.md body)
//  2. Rules (from context_requirements.rules → ~/.claude/rules/)
//  3. Conventions (from context_requirements.conventions → ~/.claude/conventions/)
//  4. Original prompt
//
// Returns the augmented prompt with all context prepended.
// If no context is available, returns originalPrompt unchanged.
func BuildFullAgentContext(agentID string, requirements *ContextRequirements, taskFiles []string, originalPrompt string) (string, error) {
	// Prevent double-injection
	if strings.Contains(originalPrompt, IdentityMarker) {
		// Identity already present — still try conventions via existing function
		return BuildAugmentedPrompt(originalPrompt, requirements, taskFiles)
	}

	var sections []string
	injected := false

	// 0. Inject session directory context
	if !strings.Contains(originalPrompt, SessionMarker) {
		if sessionDir := GetSessionDir(); sessionDir != "" {
			sections = append(sections, SessionMarker)
			sections = append(sections, fmt.Sprintf("SESSION_DIR: %s", sessionDir))
			sections = append(sections, "Write output artifacts (plans, reviews, analysis) to SESSION_DIR/.")
			sections = append(sections, SessionEndMarker)
			sections = append(sections, "")
			injected = true
		}
	}

	// 1. Load agent identity
	identity, err := LoadAgentIdentity(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[identity-loader] Warning: Failed to load identity for %s: %v\n", agentID, err)
	}
	if identity != "" {
		sections = append(sections, IdentityMarker)
		sections = append(sections, fmt.Sprintf("--- %s identity ---", agentID))
		sections = append(sections, identity)
		sections = append(sections, IdentityEndMarker)
		sections = append(sections, "")
		injected = true
	}

	// 2. Load rules and conventions via existing BuildAugmentedPrompt
	if requirements != nil && requirements.HasContextRequirements() {
		// Check if conventions already present (defensive)
		if !strings.Contains(originalPrompt, ConventionsMarker) {
			var convSections []string
			convSections = append(convSections, ConventionsMarker)
			convSections = append(convSections, "")

			// Load rules
			for _, rulesFile := range requirements.Rules {
				content, err := LoadRulesContent(rulesFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[identity-loader] Warning: Failed to load rules %s: %v\n", rulesFile, err)
					continue
				}
				convSections = append(convSections, fmt.Sprintf("--- %s ---", rulesFile))
				convSections = append(convSections, content)
				convSections = append(convSections, "")
			}

			// Load conventions (base + conditional)
			conventions := requirements.GetAllConventions(taskFiles)
			for _, convFile := range conventions {
				content, err := LoadConventionContent(convFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[identity-loader] Warning: Failed to load convention %s: %v\n", convFile, err)
					continue
				}
				convSections = append(convSections, fmt.Sprintf("--- %s ---", convFile))
				convSections = append(convSections, content)
				convSections = append(convSections, "")
			}

			convSections = append(convSections, ConventionsEndMarker)
			convSections = append(convSections, "")

			// Only add if we actually loaded something
			if len(convSections) > 4 { // marker + empty + end marker + empty = 4 (nothing loaded)
				sections = append(sections, strings.Join(convSections, "\n"))
				injected = true
			}
		}
	}

	if !injected {
		return originalPrompt, nil
	}

	// Add separator and original prompt
	sections = append(sections, "---")
	sections = append(sections, "")
	sections = append(sections, originalPrompt)

	return strings.Join(sections, "\n"), nil
}
