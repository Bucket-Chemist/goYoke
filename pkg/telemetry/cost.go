package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TierPricing captures cost per 1000 tokens for each model tier.
// Extracted from routing-schema.json tiers section.
type TierPricing struct {
	Haiku         float64 `json:"haiku"`
	HaikuThinking float64 `json:"haiku_thinking"`
	Sonnet        float64 `json:"sonnet"`
	Opus          float64 `json:"opus"`
	External      float64 `json:"external"`
}

// DefaultTierPricing provides fallback pricing when schema unavailable.
// Based on Anthropic API pricing as of 2026-01.
var DefaultTierPricing = TierPricing{
	Haiku:         0.0005,  // $0.0005 per 1K tokens
	HaikuThinking: 0.001,   // $0.001 per 1K tokens (with thinking)
	Sonnet:        0.009,   // $0.009 per 1K tokens
	Opus:          0.045,   // $0.045 per 1K tokens
	External:      0.0001,  // $0.0001 per 1K tokens (Gemini)
}

// LoadTierPricing extracts tier pricing from routing-schema.json.
// Falls back to DefaultTierPricing if schema unavailable or parsing fails.
func LoadTierPricing() TierPricing {
	schemaPath := getRoutingSchemaPath()

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return DefaultTierPricing
	}

	var schema struct {
		Tiers map[string]struct {
			CostPer1kTokens float64 `json:"cost_per_1k_tokens"`
		} `json:"tiers"`
	}

	if err := json.Unmarshal(data, &schema); err != nil {
		return DefaultTierPricing
	}

	pricing := DefaultTierPricing // Start with defaults

	if tier, ok := schema.Tiers["haiku"]; ok {
		pricing.Haiku = tier.CostPer1kTokens
	}
	if tier, ok := schema.Tiers["haiku_thinking"]; ok {
		pricing.HaikuThinking = tier.CostPer1kTokens
	}
	if tier, ok := schema.Tiers["sonnet"]; ok {
		pricing.Sonnet = tier.CostPer1kTokens
	}
	if tier, ok := schema.Tiers["opus"]; ok {
		pricing.Opus = tier.CostPer1kTokens
	}
	if tier, ok := schema.Tiers["external"]; ok {
		pricing.External = tier.CostPer1kTokens
	}

	return pricing
}

// getRoutingSchemaPath returns path to routing-schema.json
func getRoutingSchemaPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "routing-schema.json")
}

// GetTierCostRate returns the cost per 1K tokens for a given tier.
func (p TierPricing) GetTierCostRate(tier string) float64 {
	switch tier {
	case "haiku":
		return p.Haiku
	case "haiku_thinking":
		return p.HaikuThinking
	case "sonnet":
		return p.Sonnet
	case "opus":
		return p.Opus
	case "external":
		return p.External
	default:
		// Default to sonnet pricing for unknown tiers
		return p.Sonnet
	}
}

// InvocationCost represents the cost breakdown for a single invocation.
type InvocationCost struct {
	Agent          string  `json:"agent"`
	Tier           string  `json:"tier"`
	InputTokens    int     `json:"input_tokens"`
	OutputTokens   int     `json:"output_tokens"`
	ThinkingTokens int     `json:"thinking_tokens"`
	TotalTokens    int     `json:"total_tokens"`
	CostRate       float64 `json:"cost_rate"`  // Per 1K tokens
	TotalCost      float64 `json:"total_cost"` // In dollars
}

// CalculateInvocationCost computes the dollar cost for a single invocation.
// Formula: (input + output + thinking) * tier_rate / 1000
func CalculateInvocationCost(inv AgentInvocation, pricing TierPricing) InvocationCost {
	totalTokens := inv.InputTokens + inv.OutputTokens + inv.ThinkingTokens
	rate := pricing.GetTierCostRate(inv.Tier)
	cost := float64(totalTokens) * rate / 1000.0

	return InvocationCost{
		Agent:          inv.Agent,
		Tier:           inv.Tier,
		InputTokens:    inv.InputTokens,
		OutputTokens:   inv.OutputTokens,
		ThinkingTokens: inv.ThinkingTokens,
		TotalTokens:    totalTokens,
		CostRate:       rate,
		TotalCost:      cost,
	}
}

// AgentCostSummary aggregates costs for a single agent.
type AgentCostSummary struct {
	Agent           string  `json:"agent"`
	InvocationCount int     `json:"invocation_count"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	AvgCostPerCall  float64 `json:"avg_cost_per_call"`
}

// TierCostSummary aggregates costs for a model tier.
type TierCostSummary struct {
	Tier            string  `json:"tier"`
	InvocationCount int     `json:"invocation_count"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
	PercentOfTotal  float64 `json:"percent_of_total"`
}

// CalculateAgentCosts aggregates costs by agent.
func CalculateAgentCosts(invocations []AgentInvocation, pricing TierPricing) map[string]*AgentCostSummary {
	costs := make(map[string]*AgentCostSummary)

	for _, inv := range invocations {
		agent := inv.Agent
		if agent == "" {
			agent = "unknown"
		}

		summary, exists := costs[agent]
		if !exists {
			summary = &AgentCostSummary{Agent: agent}
			costs[agent] = summary
		}

		invCost := CalculateInvocationCost(inv, pricing)
		summary.InvocationCount++
		summary.TotalTokens += invCost.TotalTokens
		summary.TotalCost += invCost.TotalCost
	}

	// Calculate averages
	for _, summary := range costs {
		if summary.InvocationCount > 0 {
			summary.AvgCostPerCall = summary.TotalCost / float64(summary.InvocationCount)
		}
	}

	return costs
}

// CalculateTierCosts aggregates costs by tier.
func CalculateTierCosts(invocations []AgentInvocation, pricing TierPricing) map[string]*TierCostSummary {
	costs := make(map[string]*TierCostSummary)
	var grandTotal float64

	for _, inv := range invocations {
		tier := inv.Tier
		if tier == "" {
			tier = "unknown"
		}

		summary, exists := costs[tier]
		if !exists {
			summary = &TierCostSummary{Tier: tier}
			costs[tier] = summary
		}

		invCost := CalculateInvocationCost(inv, pricing)
		summary.InvocationCount++
		summary.TotalTokens += invCost.TotalTokens
		summary.TotalCost += invCost.TotalCost
		grandTotal += invCost.TotalCost
	}

	// Calculate percentages
	for _, summary := range costs {
		if grandTotal > 0 {
			summary.PercentOfTotal = (summary.TotalCost / grandTotal) * 100
		}
	}

	return costs
}

// CalculateSessionCost computes total cost for all invocations.
func CalculateSessionCost(invocations []AgentInvocation, pricing TierPricing) float64 {
	var total float64
	for _, inv := range invocations {
		invCost := CalculateInvocationCost(inv, pricing)
		total += invCost.TotalCost
	}
	return total
}

// SessionCostSummary provides a complete cost breakdown for a session.
type SessionCostSummary struct {
	SessionID       string                      `json:"session_id"`
	TotalCost       float64                     `json:"total_cost"`
	TotalTokens     int                         `json:"total_tokens"`
	InvocationCount int                         `json:"invocation_count"`
	ByAgent         map[string]*AgentCostSummary `json:"by_agent"`
	ByTier          map[string]*TierCostSummary  `json:"by_tier"`
}

// CalculateSessionCostSummary provides complete cost breakdown.
func CalculateSessionCostSummary(sessionID string, invocations []AgentInvocation, pricing TierPricing) *SessionCostSummary {
	var totalTokens int
	for _, inv := range invocations {
		totalTokens += inv.InputTokens + inv.OutputTokens + inv.ThinkingTokens
	}

	return &SessionCostSummary{
		SessionID:       sessionID,
		TotalCost:       CalculateSessionCost(invocations, pricing),
		TotalTokens:     totalTokens,
		InvocationCount: len(invocations),
		ByAgent:         CalculateAgentCosts(invocations, pricing),
		ByTier:          CalculateTierCosts(invocations, pricing),
	}
}

// FormatCost formats a dollar amount for display.
func FormatCost(cost float64) string {
	if cost < 0.01 {
		return fmt.Sprintf("$%.4f", cost)
	}
	return fmt.Sprintf("$%.2f", cost)
}
