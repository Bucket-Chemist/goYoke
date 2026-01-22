package harness

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Generator creates test inputs for simulation.
type Generator interface {
	GenerateToolEvent(scenarioID string) (*ToolEvent, error)
	GenerateSessionEvent(scenarioID string) (*SessionEvent, error)
	RandomToolEvent(seed int64) *ToolEvent
	RandomTaskInput(seed int64) *TaskInput
	RandomSessionEvent(seed int64) *SessionEvent
	RandomSessionMetrics(seed int64) *SessionMetrics
	GenerateWithParams(params FuzzParams) interface{}
}

// ToolEvent represents a PreToolUse hook input.
type ToolEvent struct {
	ToolName      string                 `json:"tool_name"`
	ToolInput     map[string]interface{} `json:"tool_input"`
	SessionID     string                 `json:"session_id"`
	HookEventName string                 `json:"hook_event_name"`
	CapturedAt    int64                  `json:"captured_at"`
}

// TaskInput represents the input for a Task tool call.
type TaskInput struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
	Model        string `json:"model,omitempty"`
}

// SessionEvent represents a SessionEnd hook input.
type SessionEvent struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`
	CapturedAt    int64  `json:"captured_at"`
}

// SessionMetrics represents session statistics.
type SessionMetrics struct {
	ToolCalls      int           `json:"tool_calls"`
	Duration       time.Duration `json:"duration"`
	TokensUsed     int           `json:"tokens_used"`
	ViolationCount int           `json:"violation_count"`
}

// FuzzParams controls random generation distributions.
type FuzzParams struct {
	ToolNameWeights     map[string]float64 `json:"tool_name_weights"`
	ModelWeights        map[string]float64 `json:"model_weights"`
	SubagentTypeWeights map[string]float64 `json:"subagent_type_weights"`
	AgentList           []string           `json:"agent_list"`
	PromptLengthMean    int                `json:"prompt_length_mean"`
	PromptLengthMax     int                `json:"prompt_length_max"`
	ErrorRate           float64            `json:"error_rate"`
	ViolationRate       float64            `json:"violation_rate"`
}

// DefaultFuzzParams returns sensible defaults for fuzz testing.
func DefaultFuzzParams() FuzzParams {
	return FuzzParams{
		ToolNameWeights: map[string]float64{
			"Task": 0.6,
			"Read": 0.2,
			"Bash": 0.1,
			"Glob": 0.1,
		},
		ModelWeights: map[string]float64{
			"haiku":  0.5,
			"sonnet": 0.4,
			"opus":   0.1,
		},
		SubagentTypeWeights: map[string]float64{
			"Explore":         0.3,
			"general-purpose": 0.5,
			"Plan":            0.2,
		},
		AgentList: []string{
			"codebase-search", "haiku-scout", "tech-docs-writer",
			"python-pro", "orchestrator", "architect",
		},
		PromptLengthMean: 200,
		PromptLengthMax:  1000,
		ErrorRate:        0.05,
		ViolationRate:    0.1,
	}
}

// DefaultGenerator implements Generator with fixture loading and random generation.
type DefaultGenerator struct {
	fixturesDir string
	params      FuzzParams
}

// NewGenerator creates a generator with the given fixtures directory.
func NewGenerator(fixturesDir string) *DefaultGenerator {
	return &DefaultGenerator{
		fixturesDir: fixturesDir,
		params:      DefaultFuzzParams(),
	}
}

// GenerateToolEvent loads a deterministic fixture by scenario ID.
func (g *DefaultGenerator) GenerateToolEvent(scenarioID string) (*ToolEvent, error) {
	path := filepath.Join(g.fixturesDir, "deterministic", "pretooluse", scenarioID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load fixture %s: %w", scenarioID, err)
	}

	var fixture struct {
		Input ToolEvent `json:"input"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse fixture %s: %w", scenarioID, err)
	}

	return &fixture.Input, nil
}

// GenerateSessionEvent loads a deterministic SessionEnd fixture.
func (g *DefaultGenerator) GenerateSessionEvent(scenarioID string) (*SessionEvent, error) {
	path := filepath.Join(g.fixturesDir, "deterministic", "sessionend", scenarioID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load fixture %s: %w", scenarioID, err)
	}

	var fixture struct {
		Input SessionEvent `json:"input"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		return nil, fmt.Errorf("parse fixture %s: %w", scenarioID, err)
	}

	return &fixture.Input, nil
}

// RandomToolEvent generates a random tool event with the given seed.
func (g *DefaultGenerator) RandomToolEvent(seed int64) *ToolEvent {
	rng := rand.New(rand.NewSource(seed))

	toolName := weightedChoice(rng, g.params.ToolNameWeights)

	event := &ToolEvent{
		ToolName:      toolName,
		ToolInput:     make(map[string]interface{}),
		SessionID:     fmt.Sprintf("sim-session-%d", rng.Int63()),
		HookEventName: "PreToolUse",
		CapturedAt:    1700000000 + rng.Int63n(1000000), // Deterministic timestamp from seed
	}

	if toolName == "Task" {
		taskInput := g.RandomTaskInput(rng.Int63())
		event.ToolInput["description"] = taskInput.Description
		event.ToolInput["prompt"] = taskInput.Prompt
		event.ToolInput["subagent_type"] = taskInput.SubagentType
		if taskInput.Model != "" {
			event.ToolInput["model"] = taskInput.Model
		}
	}

	return event
}

// RandomTaskInput generates a random Task tool input.
func (g *DefaultGenerator) RandomTaskInput(seed int64) *TaskInput {
	rng := rand.New(rand.NewSource(seed))

	model := weightedChoice(rng, g.params.ModelWeights)
	subagentType := weightedChoice(rng, g.params.SubagentTypeWeights)
	agent := g.params.AgentList[rng.Intn(len(g.params.AgentList))]

	promptLen := rng.Intn(g.params.PromptLengthMax-g.params.PromptLengthMean) + g.params.PromptLengthMean
	prompt := randomString(rng, promptLen)

	return &TaskInput{
		Description:  fmt.Sprintf("Test task for %s", agent),
		Prompt:       fmt.Sprintf("AGENT: %s\n\n%s", agent, prompt),
		SubagentType: subagentType,
		Model:        model,
	}
}

// RandomSessionEvent generates a random SessionEnd event.
func (g *DefaultGenerator) RandomSessionEvent(seed int64) *SessionEvent {
	rng := rand.New(rand.NewSource(seed))

	return &SessionEvent{
		SessionID:     fmt.Sprintf("sim-session-%d", rng.Int63()),
		HookEventName: "SessionEnd",
		CapturedAt:    1700000000 + rng.Int63n(1000000), // Deterministic timestamp from seed
	}
}

// RandomSessionMetrics generates random session metrics.
func (g *DefaultGenerator) RandomSessionMetrics(seed int64) *SessionMetrics {
	rng := rand.New(rand.NewSource(seed))

	violationCount := 0
	if rng.Float64() < g.params.ViolationRate {
		violationCount = rng.Intn(5) + 1
	}

	return &SessionMetrics{
		ToolCalls:      rng.Intn(100) + 1,
		Duration:       time.Duration(rng.Intn(3600)) * time.Second,
		TokensUsed:     rng.Intn(100000) + 1000,
		ViolationCount: violationCount,
	}
}

// GenerateWithParams generates input using custom parameters.
func (g *DefaultGenerator) GenerateWithParams(params FuzzParams) interface{} {
	g.params = params
	return g.RandomToolEvent(time.Now().UnixNano())
}

// weightedChoice selects a key based on weights using integer arithmetic.
// Keys are sorted for deterministic results with the same seed.
// Uses Int63 instead of Float64 to avoid platform-dependent floating-point precision.
func weightedChoice(rng *rand.Rand, weights map[string]float64) string {
	// Sort keys for deterministic iteration order
	keys := make([]string, 0, len(weights))
	for k := range weights {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Convert float weights to integer milliweights (1000x) to avoid float precision issues
	const precision = 1000
	milliWeights := make([]int64, len(keys))
	var total int64
	for i, k := range keys {
		milliWeights[i] = int64(weights[k] * precision)
		total += milliWeights[i]
	}

	if total == 0 {
		if len(keys) > 0 {
			return keys[0]
		}
		return ""
	}

	// Use Int63n for deterministic cross-platform selection
	r := rng.Int63n(total)
	var cumulative int64
	for i, k := range keys {
		cumulative += milliWeights[i]
		if r < cumulative {
			return k
		}
	}

	// Fallback to last key (edge case when r == total-1 and rounding)
	if len(keys) > 0 {
		return keys[len(keys)-1]
	}
	return ""
}

// randomString generates a random string of given length.
func randomString(rng *rand.Rand, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rng.Intn(len(charset))]
	}
	return string(result)
}
