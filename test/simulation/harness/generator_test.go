package harness

import (
	"math/rand"
	"testing"
)

func TestRandomToolEvent_Reproducibility(t *testing.T) {
	gen := NewGenerator("./fixtures")

	event1 := gen.RandomToolEvent(12345)
	event2 := gen.RandomToolEvent(12345)

	if event1.ToolName != event2.ToolName {
		t.Errorf("Same seed should produce same tool name: got %s and %s", event1.ToolName, event2.ToolName)
	}
}

func TestRandomTaskInput_ContainsAgent(t *testing.T) {
	gen := NewGenerator("./fixtures")

	input := gen.RandomTaskInput(42)

	if input.Prompt == "" {
		t.Error("Expected non-empty prompt")
	}
	if input.SubagentType == "" {
		t.Error("Expected non-empty subagent_type")
	}
}

func TestWeightedChoice_Coverage(t *testing.T) {
	weights := map[string]float64{
		"A": 0.5,
		"B": 0.3,
		"C": 0.2,
	}

	counts := make(map[string]int)
	for seed := int64(0); seed < 1000; seed++ {
		rng := rand.New(rand.NewSource(seed))
		choice := weightedChoice(rng, weights)
		counts[choice]++
	}

	// Verify all options are selected at least once
	for k := range weights {
		if counts[k] == 0 {
			t.Errorf("Option %s was never selected", k)
		}
	}
}

func TestDefaultFuzzParams(t *testing.T) {
	params := DefaultFuzzParams()

	if len(params.ToolNameWeights) == 0 {
		t.Error("Expected non-empty tool name weights")
	}
	if len(params.AgentList) == 0 {
		t.Error("Expected non-empty agent list")
	}
	if params.PromptLengthMean <= 0 {
		t.Errorf("Expected positive prompt length mean, got: %d", params.PromptLengthMean)
	}
}

func TestGenerateToolEvent_DeterministicFixtures(t *testing.T) {
	gen := NewGenerator("../fixtures")

	tests := []struct {
		scenarioID   string
		wantToolName string
		wantTaskTool bool
	}{
		{"V001_passthrough", "Read", false},
		{"V002_valid_task", "Task", true},
		{"V003_einstein_block", "Task", true},
		{"V004_subagent_mismatch", "Task", true},
		{"V005_ceiling_violation", "Task", true},
		{"V006_model_warning", "Task", true},
		{"V007_unknown_agent", "Task", true},
		{"V008_empty_prompt", "Task", true},
	}

	for _, tt := range tests {
		t.Run(tt.scenarioID, func(t *testing.T) {
			event, err := gen.GenerateToolEvent(tt.scenarioID)
			if err != nil {
				t.Fatalf("Failed to load fixture %s: %v", tt.scenarioID, err)
			}

			if event.ToolName != tt.wantToolName {
				t.Errorf("ToolName = %q, want %q", event.ToolName, tt.wantToolName)
			}

			if event.HookEventName != "PreToolUse" {
				t.Errorf("HookEventName = %q, want %q", event.HookEventName, "PreToolUse")
			}

			if event.SessionID == "" {
				t.Error("SessionID should not be empty")
			}

			if tt.wantTaskTool {
				if _, ok := event.ToolInput["prompt"]; !ok {
					t.Error("Task fixture should have prompt in tool_input")
				}
				if _, ok := event.ToolInput["subagent_type"]; !ok {
					t.Error("Task fixture should have subagent_type in tool_input")
				}
			}
		})
	}
}
