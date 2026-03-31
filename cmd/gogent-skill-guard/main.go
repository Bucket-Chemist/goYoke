package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

const (
	defaultTimeout = 5 * time.Second
	staleTTL       = 30 * time.Minute
	guardFileName  = "active-skill.json"
)

// ActiveSkill represents the guard file written during skill execution.
type ActiveSkill struct {
	Skill              string   `json:"skill"`
	TeamDir            string   `json:"team_dir"`
	RouterAllowedTools []string `json:"router_allowed_tools"`
	CreatedAt          string   `json:"created_at"`
}

// SkillGuardConfig from agents-index.json skill_guards section.
type SkillGuardConfig struct {
	RouterAllowedTools []string `json:"router_allowed_tools"`
	TeamDirSuffix      string   `json:"team_dir_suffix"`
}

func main() {
	event, err := parseEvent(os.Stdin, defaultTimeout)
	if err != nil {
		// Parse failure: pass through (don't block)
		fmt.Fprintln(os.Stderr, "[gogent-skill-guard] Warning: parse error:", err)
		fmt.Println("{}")
		return
	}

	sessionDir := resolveSessionDir()
	guardPath := filepath.Join(sessionDir, guardFileName)

	// ── SETUP MODE: Skill tool invocation ──
	if event.ToolName == "Skill" {
		handleSetupMode(event, sessionDir, guardPath)
		return
	}

	// ── GUARD MODE: All other tools ──
	output := handleGuardMode(event.ToolName, guardPath)
	fmt.Println(output)
}

// handleSetupMode processes Skill tool invocations.
func handleSetupMode(event *routing.ToolEvent, sessionDir, guardPath string) {
	skillName := extractSkillName(event.ToolInput)
	if skillName == "" {
		fmt.Println("{}")
		return
	}
	guardConfig := loadSkillGuardConfig(skillName)
	handleSetupModeWithConfig(skillName, guardConfig, sessionDir, guardPath)
}

// handleSetupModeWithConfig is the testable core of handleSetupMode.
func handleSetupModeWithConfig(skillName string, guardConfig *SkillGuardConfig, sessionDir, guardPath string) {
	if guardConfig == nil {
		// Non-team skill (e.g., /dummies-guide), no guard needed
		fmt.Println("{}")
		return
	}

	// Create team directory
	timestamp := time.Now().Unix()
	teamDir := filepath.Join(sessionDir, "teams",
		fmt.Sprintf("%d.%s", timestamp, guardConfig.TeamDirSuffix))
	if err := os.MkdirAll(teamDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-skill-guard] Warning: failed to create team dir: %v\n", err)
		fmt.Println("{}")
		return
	}

	// Write guard file
	guard := ActiveSkill{
		Skill:              skillName,
		TeamDir:            teamDir,
		RouterAllowedTools: guardConfig.RouterAllowedTools,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(guard, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-skill-guard] Warning: failed to marshal guard file: %v\n", err)
		fmt.Println("{}")
		return
	}
	if err := os.WriteFile(guardPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-skill-guard] Warning: failed to write guard file: %v\n", err)
	}

	fmt.Println("{}")
}

// handleGuardMode checks the guard file and returns JSON output string.
func handleGuardMode(toolName string, guardPath string) string {
	data, err := os.ReadFile(guardPath)
	if err != nil {
		// No guard active — fast path
		return "{}"
	}

	var guard ActiveSkill
	if err := json.Unmarshal(data, &guard); err != nil {
		// Malformed guard file — delete and pass through
		os.Remove(guardPath)
		return "{}"
	}

	// Staleness check (C-1 fix)
	if createdAt, err := time.Parse(time.RFC3339, guard.CreatedAt); err == nil {
		if time.Since(createdAt) > staleTTL {
			os.Remove(guardPath)
			return "{}"
		}
	}

	// Check if tool is allowed
	for _, allowed := range guard.RouterAllowedTools {
		if toolName == allowed {
			return "{}"
		}
	}

	// Block — tool not in allowed list
	resp := routing.NewBlockResponse("PreToolUse",
		fmt.Sprintf("[skill-guard] %s blocked during /%s skill. Router should dispatch via Task(), not use %s directly.",
			toolName, guard.Skill, toolName))

	var buf strings.Builder
	if err := resp.Marshal(&buf); err != nil {
		fmt.Fprintf(os.Stderr, "[gogent-skill-guard] Error: marshal failed: %v\n", err)
		return "{}"
	}
	return buf.String()
}

func resolveSessionDir() string {
	if dir := os.Getenv("GOGENT_SESSION_DIR"); dir != "" {
		return dir
	}

	// Try project dir resolution chain
	projectDir := os.Getenv("GOGENT_PROJECT_DIR")
	if projectDir == "" {
		projectDir = os.Getenv("CLAUDE_PROJECT_DIR")
	}
	if projectDir == "" {
		projectDir, _ = os.Getwd()
	}
	if projectDir != "" {
		data, err := os.ReadFile(filepath.Join(config.RuntimeDir(projectDir), "current-session"))
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return filepath.Join(".gogent", "sessions", "unknown")
}

func extractSkillName(toolInput map[string]interface{}) string {
	if skill, ok := toolInput["skill"]; ok {
		if s, ok := skill.(string); ok {
			return s
		}
	}
	return ""
}

func loadSkillGuardConfig(skillName string) *SkillGuardConfig {
	configDir, err := routing.GetClaudeConfigDir()
	if err != nil {
		return nil
	}
	indexPath := filepath.Join(configDir, "agents", "agents-index.json")
	return loadSkillGuardConfigFrom(skillName, indexPath)
}

// loadSkillGuardConfigFrom loads the skill guard config from a specific index file path.
// This is the testable core function that allows tests to provide fixture paths.
func loadSkillGuardConfigFrom(skillName, indexPath string) *SkillGuardConfig {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}

	var index struct {
		SkillGuards map[string]*SkillGuardConfig `json:"skill_guards"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil
	}

	if index.SkillGuards == nil {
		return nil
	}

	return index.SkillGuards[skillName]
}

func parseEvent(r io.Reader, timeout time.Duration) (*routing.ToolEvent, error) {
	type result struct {
		event *routing.ToolEvent
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		reader := bufio.NewReader(r)
		data, err := io.ReadAll(reader)
		if err != nil {
			ch <- result{nil, err}
			return
		}
		var event routing.ToolEvent
		if err := json.Unmarshal(data, &event); err != nil {
			ch <- result{nil, fmt.Errorf("invalid JSON: %w", err)}
			return
		}
		ch <- result{&event, nil}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("STDIN read timeout after %v", timeout)
	}
}
