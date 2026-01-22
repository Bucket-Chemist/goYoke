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
