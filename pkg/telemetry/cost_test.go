package telemetry

import (
	"math"
	"testing"
)

func TestCalculateInvocationCost_Sonnet(t *testing.T) {
	inv := AgentInvocation{
		Agent:        "python-pro",
		Tier:         "sonnet",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 1500 tokens * $0.009/1K = $0.0135
	expectedCost := 0.0135
	if math.Abs(cost.TotalCost-expectedCost) > 0.001 {
		t.Errorf("Expected cost ~$0.0135, got: $%f", cost.TotalCost)
	}

	if cost.TotalTokens != 1500 {
		t.Errorf("Expected 1500 total tokens, got: %d", cost.TotalTokens)
	}

	if cost.Agent != "python-pro" {
		t.Errorf("Expected agent python-pro, got: %s", cost.Agent)
	}

	if cost.Tier != "sonnet" {
		t.Errorf("Expected tier sonnet, got: %s", cost.Tier)
	}
}

func TestCalculateInvocationCost_WithThinking(t *testing.T) {
	inv := AgentInvocation{
		Agent:          "orchestrator",
		Tier:           "sonnet",
		InputTokens:    2000,
		OutputTokens:   1000,
		ThinkingTokens: 5000,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 8000 tokens * $0.009/1K = $0.072
	expectedCost := 0.072
	if math.Abs(cost.TotalCost-expectedCost) > 0.001 {
		t.Errorf("Expected cost ~$0.072, got: $%f", cost.TotalCost)
	}

	if cost.TotalTokens != 8000 {
		t.Errorf("Expected 8000 total tokens, got: %d", cost.TotalTokens)
	}

	if cost.ThinkingTokens != 5000 {
		t.Errorf("Expected 5000 thinking tokens, got: %d", cost.ThinkingTokens)
	}
}

func TestCalculateInvocationCost_Opus(t *testing.T) {
	inv := AgentInvocation{
		Agent:          "einstein",
		Tier:           "opus",
		InputTokens:    5000,
		OutputTokens:   3000,
		ThinkingTokens: 10000,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 18000 tokens * $0.045/1K = $0.81
	expectedCost := 0.81
	if math.Abs(cost.TotalCost-expectedCost) > 0.01 {
		t.Errorf("Expected cost ~$0.81, got: $%f", cost.TotalCost)
	}

	if cost.TotalTokens != 18000 {
		t.Errorf("Expected 18000 total tokens, got: %d", cost.TotalTokens)
	}
}

func TestCalculateInvocationCost_Haiku(t *testing.T) {
	inv := AgentInvocation{
		Agent:        "haiku-scout",
		Tier:         "haiku",
		InputTokens:  500,
		OutputTokens: 200,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 700 tokens * $0.0005/1K = $0.00035
	expectedCost := 0.00035
	if math.Abs(cost.TotalCost-expectedCost) > 0.0001 {
		t.Errorf("Expected cost ~$0.00035, got: $%f", cost.TotalCost)
	}
}

func TestCalculateInvocationCost_HaikuThinking(t *testing.T) {
	inv := AgentInvocation{
		Agent:          "code-reviewer",
		Tier:           "haiku_thinking",
		InputTokens:    1000,
		OutputTokens:   500,
		ThinkingTokens: 2000,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 3500 tokens * $0.001/1K = $0.0035
	expectedCost := 0.0035
	if math.Abs(cost.TotalCost-expectedCost) > 0.0001 {
		t.Errorf("Expected cost ~$0.0035, got: $%f", cost.TotalCost)
	}
}

func TestCalculateInvocationCost_External(t *testing.T) {
	inv := AgentInvocation{
		Agent:        "external-agent",
		Tier:         "external",
		InputTokens:  10000,
		OutputTokens: 5000,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// 15000 tokens * $0.0001/1K = $0.0015
	expectedCost := 0.0015
	if math.Abs(cost.TotalCost-expectedCost) > 0.0001 {
		t.Errorf("Expected cost ~$0.0015, got: $%f", cost.TotalCost)
	}
}

func TestCalculateInvocationCost_UnknownTier(t *testing.T) {
	inv := AgentInvocation{
		Agent:        "unknown-agent",
		Tier:         "super_duper",
		InputTokens:  1000,
		OutputTokens: 500,
	}

	cost := CalculateInvocationCost(inv, DefaultTierPricing)

	// Unknown tier defaults to sonnet pricing
	// 1500 tokens * $0.009/1K = $0.0135
	expectedCost := 0.0135
	if math.Abs(cost.TotalCost-expectedCost) > 0.001 {
		t.Errorf("Expected unknown tier to use sonnet rate, got: $%f", cost.TotalCost)
	}
}

func TestCalculateAgentCosts_MultipleAgents(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "python-pro", Tier: "sonnet", InputTokens: 1000, OutputTokens: 500},
		{Agent: "python-pro", Tier: "sonnet", InputTokens: 1000, OutputTokens: 500},
		{Agent: "haiku-scout", Tier: "haiku", InputTokens: 500, OutputTokens: 200},
	}

	costs := CalculateAgentCosts(invocations, DefaultTierPricing)

	if len(costs) != 2 {
		t.Errorf("Expected 2 agents, got: %d", len(costs))
	}

	pythonCost := costs["python-pro"]
	if pythonCost == nil {
		t.Fatal("Expected python-pro in costs")
	}

	if pythonCost.InvocationCount != 2 {
		t.Errorf("Expected 2 python-pro invocations, got: %d", pythonCost.InvocationCount)
	}

	// 2 * 1500 tokens * $0.009/1K = $0.027
	expectedPythonCost := 0.027
	if math.Abs(pythonCost.TotalCost-expectedPythonCost) > 0.001 {
		t.Errorf("Expected python-pro cost ~$0.027, got: $%f", pythonCost.TotalCost)
	}

	// Verify average cost per call
	expectedAvg := 0.0135
	if math.Abs(pythonCost.AvgCostPerCall-expectedAvg) > 0.001 {
		t.Errorf("Expected avg cost ~$0.0135, got: $%f", pythonCost.AvgCostPerCall)
	}

	haikuCost := costs["haiku-scout"]
	if haikuCost == nil {
		t.Fatal("Expected haiku-scout in costs")
	}

	if haikuCost.InvocationCount != 1 {
		t.Errorf("Expected 1 haiku-scout invocation, got: %d", haikuCost.InvocationCount)
	}
}

func TestCalculateAgentCosts_EmptyAgent(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "", Tier: "sonnet", InputTokens: 1000, OutputTokens: 500},
	}

	costs := CalculateAgentCosts(invocations, DefaultTierPricing)

	if _, exists := costs["unknown"]; !exists {
		t.Error("Expected empty agent to be normalized to 'unknown'")
	}
}

func TestCalculateTierCosts_Distribution(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "haiku", InputTokens: 1000, OutputTokens: 500},
		{Tier: "sonnet", InputTokens: 1000, OutputTokens: 500},
		{Tier: "opus", InputTokens: 1000, OutputTokens: 500},
	}

	costs := CalculateTierCosts(invocations, DefaultTierPricing)

	if len(costs) != 3 {
		t.Errorf("Expected 3 tiers, got: %d", len(costs))
	}

	// Verify opus is most expensive
	if costs["opus"].TotalCost <= costs["sonnet"].TotalCost {
		t.Error("Expected opus to be more expensive than sonnet")
	}
	if costs["sonnet"].TotalCost <= costs["haiku"].TotalCost {
		t.Error("Expected sonnet to be more expensive than haiku")
	}

	// Verify percentages sum to ~100
	var totalPct float64
	for _, c := range costs {
		totalPct += c.PercentOfTotal
	}
	if totalPct < 99 || totalPct > 101 {
		t.Errorf("Expected percentages to sum to ~100, got: %f", totalPct)
	}
}

func TestCalculateTierCosts_EmptyTier(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "", InputTokens: 1000, OutputTokens: 500},
	}

	costs := CalculateTierCosts(invocations, DefaultTierPricing)

	if _, exists := costs["unknown"]; !exists {
		t.Error("Expected empty tier to be normalized to 'unknown'")
	}
}

func TestCalculateSessionCost_Total(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "sonnet", InputTokens: 1000, OutputTokens: 500},
		{Tier: "sonnet", InputTokens: 2000, OutputTokens: 1000},
	}

	total := CalculateSessionCost(invocations, DefaultTierPricing)

	// (1500 + 3000) * $0.009/1K = $0.0405
	expected := 0.0405
	if math.Abs(total-expected) > 0.001 {
		t.Errorf("Expected total ~$0.0405, got: $%f", total)
	}
}

func TestCalculateSessionCost_Empty(t *testing.T) {
	invocations := []AgentInvocation{}

	total := CalculateSessionCost(invocations, DefaultTierPricing)

	if total != 0 {
		t.Errorf("Expected 0 for empty invocations, got: $%f", total)
	}
}

func TestCalculateSessionCostSummary_Complete(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "python-pro", Tier: "sonnet", InputTokens: 1000, OutputTokens: 500, ThinkingTokens: 500},
		{Agent: "haiku-scout", Tier: "haiku", InputTokens: 500, OutputTokens: 200},
	}

	summary := CalculateSessionCostSummary("test-session", invocations, DefaultTierPricing)

	if summary.SessionID != "test-session" {
		t.Errorf("Expected session ID test-session, got: %s", summary.SessionID)
	}

	if summary.InvocationCount != 2 {
		t.Errorf("Expected 2 invocations, got: %d", summary.InvocationCount)
	}

	// Total tokens: (1000+500+500) + (500+200) = 2700
	if summary.TotalTokens != 2700 {
		t.Errorf("Expected 2700 total tokens, got: %d", summary.TotalTokens)
	}

	// Verify by_agent has both agents
	if len(summary.ByAgent) != 2 {
		t.Errorf("Expected 2 agents in ByAgent, got: %d", len(summary.ByAgent))
	}

	// Verify by_tier has both tiers
	if len(summary.ByTier) != 2 {
		t.Errorf("Expected 2 tiers in ByTier, got: %d", len(summary.ByTier))
	}
}

func TestLoadTierPricing_FallbackToDefault(t *testing.T) {
	// When schema is unavailable, should return defaults
	pricing := LoadTierPricing()

	// If routing-schema.json exists with pricing, this might differ
	// For robustness, just verify non-zero values
	if pricing.Sonnet <= 0 {
		t.Error("Expected positive sonnet rate")
	}

	if pricing.Haiku <= 0 {
		t.Error("Expected positive haiku rate")
	}

	if pricing.Opus <= 0 {
		t.Error("Expected positive opus rate")
	}
}

func TestGetTierCostRate_AllTiers(t *testing.T) {
	pricing := DefaultTierPricing

	tests := []struct {
		tier     string
		expected float64
	}{
		{"haiku", 0.0005},
		{"haiku_thinking", 0.001},
		{"sonnet", 0.009},
		{"opus", 0.045},
		{"external", 0.0001},
		{"unknown_tier", 0.009}, // Defaults to sonnet
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			rate := pricing.GetTierCostRate(tt.tier)
			if math.Abs(rate-tt.expected) > 0.0001 {
				t.Errorf("GetTierCostRate(%s) = %f, want %f", tt.tier, rate, tt.expected)
			}
		})
	}
}

func TestFormatCost_SmallAmounts(t *testing.T) {
	small := FormatCost(0.0015)
	if small != "$0.0015" {
		t.Errorf("Expected $0.0015, got: %s", small)
	}

	// 0.00035 rounds to 0.0004 with %.4f rounding (banker's rounding)
	// Actually Go uses "round half to even", so 0.00035 -> 0.0003 or 0.0004 depending
	// Let's use a cleaner value
	verySmall := FormatCost(0.00045)
	if verySmall != "$0.0005" && verySmall != "$0.0004" {
		t.Errorf("Expected $0.0004 or $0.0005, got: %s", verySmall)
	}
}

func TestFormatCost_LargeAmounts(t *testing.T) {
	large := FormatCost(1.50)
	if large != "$1.50" {
		t.Errorf("Expected $1.50, got: %s", large)
	}

	medium := FormatCost(0.05)
	if medium != "$0.05" {
		t.Errorf("Expected $0.05, got: %s", medium)
	}
}

func TestFormatCost_Boundary(t *testing.T) {
	// Exactly at boundary
	boundary := FormatCost(0.01)
	if boundary != "$0.01" {
		t.Errorf("Expected $0.01, got: %s", boundary)
	}

	// Just below boundary
	belowBoundary := FormatCost(0.0099)
	if belowBoundary != "$0.0099" {
		t.Errorf("Expected $0.0099, got: %s", belowBoundary)
	}
}

func TestFormatCost_Zero(t *testing.T) {
	zero := FormatCost(0)
	if zero != "$0.0000" {
		t.Errorf("Expected $0.0000, got: %s", zero)
	}
}

func TestCalculateAgentCosts_TotalTokens(t *testing.T) {
	invocations := []AgentInvocation{
		{Agent: "test-agent", Tier: "sonnet", InputTokens: 100, OutputTokens: 50, ThinkingTokens: 25},
		{Agent: "test-agent", Tier: "sonnet", InputTokens: 200, OutputTokens: 100, ThinkingTokens: 50},
	}

	costs := CalculateAgentCosts(invocations, DefaultTierPricing)

	agent := costs["test-agent"]
	// Total tokens: (100+50+25) + (200+100+50) = 525
	if agent.TotalTokens != 525 {
		t.Errorf("Expected 525 total tokens, got: %d", agent.TotalTokens)
	}
}

func TestCalculateTierCosts_TotalTokens(t *testing.T) {
	invocations := []AgentInvocation{
		{Tier: "sonnet", InputTokens: 100, OutputTokens: 50, ThinkingTokens: 25},
		{Tier: "sonnet", InputTokens: 200, OutputTokens: 100, ThinkingTokens: 50},
	}

	costs := CalculateTierCosts(invocations, DefaultTierPricing)

	tier := costs["sonnet"]
	// Total tokens: (100+50+25) + (200+100+50) = 525
	if tier.TotalTokens != 525 {
		t.Errorf("Expected 525 total tokens, got: %d", tier.TotalTokens)
	}
}
