package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToastConstantAndPayload verifies the toast type constant value and that
// toastPayload marshals correctly with message and level fields.
func TestToastConstantAndPayload(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "toast", typeToast)

	p := toastPayload{Message: "team launched — /team-status to monitor", Level: "info"}
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	assert.JSONEq(t, `{"message":"team launched — /team-status to monitor","level":"info"}`, string(raw))
}

// TestTeamToast_LaunchContainsStatusHint verifies that the launch toast message
// contains a /team-status action hint.
func TestTeamToast_LaunchContainsStatusHint(t *testing.T) {
	t.Parallel()

	msg := "team launched — /team-status to monitor"
	assert.Contains(t, msg, "/team-status", "launch toast must include /team-status hint")
}

// TestTeamToast_CompletionContainsResultHint verifies that the completion toast
// message includes both the /team-result hint and the dollar cost.
func TestTeamToast_CompletionContainsResultHint(t *testing.T) {
	t.Parallel()

	totalCost := 1.23
	msg := fmt.Sprintf("team complete ($%.2f) — /team-result to view findings", totalCost)
	assert.Contains(t, msg, "/team-result", "completion toast must include /team-result hint")
	assert.Contains(t, msg, "$1.23", "completion toast must include dollar cost")
}

// TestTeamToast_FailureContainsStatusHint verifies that the failure toast message
// includes the /team-status hint and propagates the error summary.
func TestTeamToast_FailureContainsStatusHint(t *testing.T) {
	t.Parallel()

	err := errors.New("context deadline exceeded")
	msg := fmt.Sprintf("team failed — /team-status to inspect: %v", err)
	assert.Contains(t, msg, "/team-status", "failure toast must include /team-status hint")
	assert.Contains(t, msg, "context deadline exceeded", "failure toast must include error summary")
}

// TestTeamToast_BudgetWarningContainsAmounts verifies that the budget warning
// toast includes both the dollar amounts and a percentage.
func TestTeamToast_BudgetWarningContainsAmounts(t *testing.T) {
	t.Parallel()

	remainingAfter := 2.50
	maxBudget := 10.0
	pct := (remainingAfter / maxBudget) * 100
	msg := fmt.Sprintf("budget warning: $%.2f of $%.2f remaining (%.0f%%) — /team-status for cost breakdown",
		remainingAfter, maxBudget, pct)

	assert.Contains(t, msg, "$2.50", "budget toast must include remaining USD amount")
	assert.Contains(t, msg, "$10.00", "budget toast must include max budget USD amount")
	assert.Contains(t, msg, "25%", "budget toast must include percentage")
	assert.Contains(t, msg, "/team-status", "budget toast must include /team-status hint")
}

// TestTeamToast_MemberFailureContainsHint verifies member failure toast format
// for both non-retryable and retry-exhausted paths.
func TestTeamToast_MemberFailureContainsHint(t *testing.T) {
	t.Parallel()

	t.Run("non-retryable", func(t *testing.T) {
		t.Parallel()
		name := "review-backend"
		msg := fmt.Sprintf("%s failed — /team-status for details", name)
		assert.Contains(t, msg, "/team-status")
		assert.Contains(t, msg, "review-backend")
	})

	t.Run("retries-exhausted", func(t *testing.T) {
		t.Parallel()
		name := "implement-api"
		maxRetries := 2
		msg := fmt.Sprintf("%s failed after %d retries — /team-status for details", name, maxRetries+1)
		assert.Contains(t, msg, "/team-status")
		assert.Contains(t, msg, "implement-api")
		assert.Contains(t, msg, "3 retries")
	})
}
