package routing

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

// TestParseToolEvent_RealCorpus (goYoke-009) validates event parsing against
// the captured real event corpus to verify it handles all production cases.
//
// This test ensures 100% parsing success rate across the entire corpus,
// validating that the parser handles all real-world edge cases.
func TestParseToolEvent_RealCorpus(t *testing.T) {
	// Load real event corpus
	corpusPath := "../../test/fixtures/event-corpus.json"
	data, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("Skipping corpus test: %v", err)
		return
	}

	var events []json.RawMessage
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("Failed to parse corpus: %v", err)
	}

	// Parse each event
	successCount := 0
	for i, rawEvent := range events {
		reader := strings.NewReader(string(rawEvent))
		event, err := ParseToolEvent(reader, 5*time.Second)

		if err != nil {
			t.Errorf("Event %d failed to parse: %v\nRaw event: %s", i, err, string(rawEvent))
			continue
		}

		// Validate required fields
		if event.ToolName == "" {
			t.Errorf("Event %d missing tool_name\nRaw event: %s", i, string(rawEvent))
		}
		if event.SessionID == "" {
			t.Errorf("Event %d missing session_id\nRaw event: %s", i, string(rawEvent))
		}
		if event.HookEventName == "" {
			t.Errorf("Event %d missing hook_event_name\nRaw event: %s", i, string(rawEvent))
		}

		successCount++
	}

	// Require 100% success rate
	if successCount != len(events) {
		t.Errorf("Only %d/%d events parsed successfully", successCount, len(events))
	} else {
		t.Logf("✓ Successfully parsed all %d real events", successCount)
	}
}

// TestParseTaskInput_RealCorpus (goYoke-009) validates Task event parsing
// against real corpus data to ensure TaskInput extraction works correctly.
//
// This test filters corpus events to Task tools and validates that
// ParseTaskInput correctly extracts the Task-specific fields.
func TestParseTaskInput_RealCorpus(t *testing.T) {
	// Load corpus and parse as ToolEvents
	corpusPath := "../../test/fixtures/event-corpus.json"
	data, err := os.ReadFile(corpusPath)
	if err != nil {
		t.Skipf("Skipping corpus test: %v", err)
		return
	}

	var events []ToolEvent
	if err := json.Unmarshal(data, &events); err != nil {
		t.Fatalf("Failed to parse corpus: %v", err)
	}

	// Parse Task events
	taskCount := 0
	taskSuccessCount := 0
	for i, event := range events {
		if event.ToolName != "Task" {
			continue
		}

		taskCount++
		taskInput, err := ParseTaskInput(event.ToolInput)
		if err != nil {
			t.Errorf("Event %d Task input parse failed: %v\nTool input: %+v", i, err, event.ToolInput)
			continue
		}

		// Validate Task input has prompt (required field)
		if taskInput.Prompt == "" {
			t.Errorf("Event %d Task missing prompt\nTask input: %+v", i, event.ToolInput)
		}

		taskSuccessCount++
	}

	if taskCount == 0 {
		t.Logf("⚠ No Task events found in corpus (this is OK if corpus doesn't include Task events)")
	} else {
		if taskSuccessCount != taskCount {
			t.Errorf("Only %d/%d Task events parsed successfully", taskSuccessCount, taskCount)
		} else {
			t.Logf("✓ Successfully parsed all %d Task events from corpus", taskSuccessCount)
		}
	}
}
