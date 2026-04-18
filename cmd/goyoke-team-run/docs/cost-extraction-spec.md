# Cost Extraction Specification

**Based on**: `claude-cli-output-format.md` (verified 2026-02-07)
**Implements**: TC-005, consumed by TC-008

## Go Struct Definitions

```go
// CLIEvent represents a single event in the Claude CLI JSON array output.
type CLIEvent struct {
    Type    string `json:"type"`
    Subtype string `json:"subtype,omitempty"`
}

// CLIResultEvent is the final event in the CLI output array (type="result").
type CLIResultEvent struct {
    Type       string  `json:"type"`
    Subtype    string  `json:"subtype"`
    IsError    bool    `json:"is_error"`
    DurationMs int     `json:"duration_ms"`
    NumTurns   int     `json:"num_turns"`
    Result     *string `json:"result,omitempty"`   // Pointer: missing on budget exceeded
    StopReason *string `json:"stop_reason"`        // Pointer: null for normal completion

    SessionID    string  `json:"session_id"`
    TotalCostUSD float64 `json:"total_cost_usd"`

    Usage      CLIUsage                  `json:"usage"`
    ModelUsage map[string]CLIModelUsage  `json:"modelUsage"`

    PermissionDenials []interface{} `json:"permission_denials"`
    Errors            []interface{} `json:"errors,omitempty"`
}

// CLIUsage is the aggregate token usage.
type CLIUsage struct {
    InputTokens                int `json:"input_tokens"`
    CacheCreationInputTokens   int `json:"cache_creation_input_tokens"`
    CacheReadInputTokens       int `json:"cache_read_input_tokens"`
    OutputTokens               int `json:"output_tokens"`
}

// CLIModelUsage is per-model usage breakdown.
type CLIModelUsage struct {
    InputTokens                int     `json:"inputTokens"`
    OutputTokens               int     `json:"outputTokens"`
    CacheReadInputTokens       int     `json:"cacheReadInputTokens"`
    CacheCreationInputTokens   int     `json:"cacheCreationInputTokens"`
    WebSearchRequests          int     `json:"webSearchRequests"`
    CostUSD                    float64 `json:"costUSD"`
    ContextWindow              int     `json:"contextWindow"`
    MaxOutputTokens            int     `json:"maxOutputTokens"`
}
```

## Extraction Function

```go
// extractResult parses the CLI JSON array output and returns the result event.
func extractResult(rawOutput []byte) (*CLIResultEvent, error) {
    // CLI outputs a JSON array of events
    var events []json.RawMessage
    if err := json.Unmarshal(rawOutput, &events); err != nil {
        return nil, fmt.Errorf("parse CLI output as JSON array: %w", err)
    }

    if len(events) == 0 {
        return nil, fmt.Errorf("empty CLI output array")
    }

    // Find the result event (always last, but search backwards to be safe)
    for i := len(events) - 1; i >= 0; i-- {
        var peek CLIEvent
        if err := json.Unmarshal(events[i], &peek); err != nil {
            continue
        }
        if peek.Type == "result" {
            var result CLIResultEvent
            if err := json.Unmarshal(events[i], &result); err != nil {
                return nil, fmt.Errorf("parse result event: %w", err)
            }
            return &result, nil
        }
    }

    return nil, fmt.Errorf("no result event found in CLI output (%d events)", len(events))
}

// extractCost returns the cost in USD from CLI output.
// Returns 0 with no error if cost is legitimately zero (e.g., error before API call).
func extractCost(rawOutput []byte) (float64, error) {
    result, err := extractResult(rawOutput)
    if err != nil {
        return 0, err
    }
    return result.TotalCostUSD, nil
}

// extractAgentOutput returns the agent's text response.
// Returns empty string if result field is missing (e.g., budget exceeded).
func extractAgentOutput(rawOutput []byte) (string, error) {
    result, err := extractResult(rawOutput)
    if err != nil {
        return "", err
    }
    if result.Result == nil {
        return "", nil
    }
    return *result.Result, nil
}
```

## Error Detection

```go
// AgentOutcome represents the result of a spawned agent.
type AgentOutcome struct {
    Success        bool
    BudgetExceeded bool
    AgentError     bool
    Cost           float64
    Output         string
    DurationMs     int
    NumTurns       int
}

func classifyOutcome(result *CLIResultEvent) AgentOutcome {
    outcome := AgentOutcome{
        Cost:       result.TotalCostUSD,
        DurationMs: result.DurationMs,
        NumTurns:   result.NumTurns,
    }

    if result.Result != nil {
        outcome.Output = *result.Result
    }

    switch {
    case result.Subtype == "error_max_budget_usd":
        outcome.BudgetExceeded = true
    case result.IsError:
        outcome.AgentError = true
    default:
        outcome.Success = true
    }

    return outcome
}
```

## Fallback Strategy

If cost extraction fails entirely (malformed output, no result event):

1. Log raw output to `runner.log` with `[COST-PARSE-ERROR]` prefix
2. Set cost to 0
3. Mark `cost_estimated: true` on the member in config.json
4. **Continue execution** (don't block team on parse failure)

## Budget Enforcement

Three layers, in order of precedence:

1. **CLI-level**: `--max-budget-usd` flag on each spawned agent (hard cap by CLI itself)
2. **Team-level**: Check `budget_remaining_usd` before each spawn (Go binary enforces)
3. **Post-hoc**: Extract actual cost after completion, update `budget_remaining_usd`

The CLI-level cap means an agent can never exceed its per-agent budget even if our extraction fails.

## Test Cases for cost_test.go

```go
var extractCostTests = []struct {
    name     string
    input    string
    wantCost float64
    wantErr  bool
}{
    {
        name:     "valid success result",
        input:    `[{"type":"system"},{"type":"result","subtype":"success","is_error":false,"total_cost_usd":0.135,"usage":{},"modelUsage":{}}]`,
        wantCost: 0.135,
    },
    {
        name:     "budget exceeded",
        input:    `[{"type":"system"},{"type":"result","subtype":"error_max_budget_usd","is_error":false,"total_cost_usd":0.143,"usage":{},"modelUsage":{}}]`,
        wantCost: 0.143,
    },
    {
        name:     "error with zero cost",
        input:    `[{"type":"system"},{"type":"result","subtype":"success","is_error":true,"total_cost_usd":0,"usage":{},"modelUsage":{}}]`,
        wantCost: 0,
    },
    {
        name:    "empty array",
        input:   `[]`,
        wantErr: true,
    },
    {
        name:    "invalid json",
        input:   `not json`,
        wantErr: true,
    },
    {
        name:    "no result event",
        input:   `[{"type":"system"},{"type":"assistant"}]`,
        wantErr: true,
    },
    {
        name:     "result with no cost field defaults to zero",
        input:    `[{"type":"result","subtype":"success","is_error":false,"usage":{},"modelUsage":{}}]`,
        wantCost: 0,
    },
}
```
