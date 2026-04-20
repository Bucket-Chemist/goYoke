package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigGuard_BlocksSettingsChange(t *testing.T) {
	event := &configChangeEvent{
		SessionID:     "test-session",
		HookEventName: "ConfigChange",
		Source:        "project_settings",
		FilePath:      "/home/user/.claude/settings.json",
	}

	block, reason := decision(event)

	assert.True(t, block, "expected settings.json change to be blocked")
	assert.Contains(t, reason, "Mid-session settings changes blocked")
}

func TestConfigGuard_AllowsPolicySettings(t *testing.T) {
	event := &configChangeEvent{
		SessionID:     "test-session",
		HookEventName: "ConfigChange",
		Source:        "policy_settings",
		FilePath:      "/path/to/settings.json",
	}

	block, reason := decision(event)

	assert.False(t, block, "expected policy_settings to always be allowed")
	assert.Empty(t, reason)
}

func TestConfigGuard_AllowsSkillChanges(t *testing.T) {
	event := &configChangeEvent{
		SessionID:     "test-session",
		HookEventName: "ConfigChange",
		Source:        "skills",
		FilePath:      "/path/to/skill-config.json",
	}

	block, reason := decision(event)

	assert.False(t, block, "expected skills source to always be allowed")
	assert.Empty(t, reason)
}

func TestConfigGuard_AllowsNonSettingsFile(t *testing.T) {
	event := &configChangeEvent{
		SessionID:     "test-session",
		HookEventName: "ConfigChange",
		Source:        "project_settings",
		FilePath:      "/path/to/agents-index.json",
	}

	block, reason := decision(event)

	assert.False(t, block, "expected non-settings.json file to be allowed")
	assert.Empty(t, reason)
}

func TestConfigGuard_ReadEvent_ValidJSON(t *testing.T) {
	input := `{
		"session_id": "abc123",
		"hook_event_name": "ConfigChange",
		"source": "project_settings",
		"file_path": "/path/to/settings.json",
		"cwd": "/home/user/project",
		"permission_mode": "default"
	}`

	event, err := readEvent(strings.NewReader(input), 5*time.Second)

	require.NoError(t, err)
	assert.Equal(t, "abc123", event.SessionID)
	assert.Equal(t, "project_settings", event.Source)
	assert.Equal(t, "/path/to/settings.json", event.FilePath)
}

func TestConfigGuard_ReadEvent_InvalidJSON(t *testing.T) {
	input := `not valid json`

	_, err := readEvent(strings.NewReader(input), 5*time.Second)

	assert.Error(t, err)
}

func TestConfigGuard_BlocksLocalSettingsChange(t *testing.T) {
	event := &configChangeEvent{
		SessionID: "test-session",
		Source:    "local_settings",
		FilePath:  "/project/.claude/settings.json",
	}

	block, _ := decision(event)

	assert.True(t, block, "expected local_settings settings.json to be blocked")
}
