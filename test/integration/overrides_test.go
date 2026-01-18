package integration

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/config"
	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/routing"
)

func TestOverrideWorkflow(t *testing.T) {
	// Create temp violations log
	tmpLog := "/tmp/test-overrides.jsonl"
	defer os.Remove(tmpLog)

	// Mock XDG environment to use temp log
	origRuntime := os.Getenv("XDG_RUNTIME_DIR")
	defer os.Setenv("XDG_RUNTIME_DIR", origRuntime)

	tmpDir := t.TempDir()
	os.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Parse event with override
	eventJSON := `{
		"tool_name": "Task",
		"tool_input": {
			"model": "sonnet",
			"prompt": "--force-delegation=sonnet\n\nAGENT: architect\n\nCreate plan"
		},
		"session_id": "test-override",
		"hook_event_name": "PreToolUse"
	}`

	reader := strings.NewReader(eventJSON)
	event, err := routing.ParseToolEvent(reader, 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to parse event: %v", err)
	}

	// Parse Task input
	taskInput, err := routing.ParseTaskInput(event.ToolInput)
	if err != nil {
		t.Fatalf("Failed to parse task input: %v", err)
	}

	// Parse overrides
	overrides := routing.ParseOverrides(taskInput.Prompt)
	if overrides.ForceDelegation != "sonnet" {
		t.Errorf("Expected force-delegation sonnet, got: %s", overrides.ForceDelegation)
	}

	// Log a violation (simulated ceiling check)
	violation := &routing.Violation{
		SessionID:     event.SessionID,
		ViolationType: "delegation_ceiling",
		Agent:         "architect",
		Model:         "sonnet",
		Reason:        "Ceiling is haiku, agent requires sonnet",
		Override:      "force-delegation=sonnet",
	}

	if err := routing.LogViolation(violation, ""); err != nil {
		t.Fatalf("Failed to log violation: %v", err)
	}

	// Verify log
	logPath := config.GetViolationsLogPath()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log: %v", err)
	}

	var logged routing.Violation
	if err := json.Unmarshal(data, &logged); err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}

	if logged.Override != "force-delegation=sonnet" {
		t.Errorf("Expected override logged, got: %s", logged.Override)
	}

	t.Logf("✓ Override workflow complete: parsed, logged, verified")
}
