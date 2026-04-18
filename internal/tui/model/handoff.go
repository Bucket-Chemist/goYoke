// Package model defines shared state types for the goYoke TUI.
// This file implements context handoff generation for provider switches.
package model

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/Bucket-Chemist/goYoke/internal/tui/util"
)

// maxHandoffMessages is the number of recent messages to scan for handoff context.
const maxHandoffMessages = 10

// maxContentLen truncates individual message content in the handoff summary.
const maxContentLen = 200

// buildHandoffSummary creates a compact context summary from the most recent
// messages in a conversation. The summary is injected as a system message
// into the target provider's conversation so it has context about what was
// being discussed.
//
// Returns "" if there are fewer than 2 messages (nothing meaningful to transfer).
func buildHandoffSummary(
	msgs []state.DisplayMessage,
	fromProvider, toProvider state.ProviderID,
) string {
	if len(msgs) < 2 {
		return ""
	}

	// Take last N messages.
	start := 0
	if len(msgs) > maxHandoffMessages {
		start = len(msgs) - maxHandoffMessages
	}
	recent := msgs[start:]

	// Find last user and assistant message; count roles and tool blocks.
	var lastUser, lastAssistant string
	var userCount, assistantCount, toolCount int

	for i := len(recent) - 1; i >= 0; i-- {
		msg := recent[i]
		switch msg.Role {
		case "user":
			userCount++
			if lastUser == "" {
				lastUser = util.Truncate(msg.Content, maxContentLen)
			}
		case "assistant":
			assistantCount++
			if lastAssistant == "" {
				lastAssistant = util.Truncate(msg.Content, maxContentLen)
			}
			toolCount += len(msg.ToolBlocks)
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "[Context transferred from %s → %s]\n", fromProvider, toProvider)
	fmt.Fprintf(&sb, "Conversation: %d messages (%d user, %d assistant)",
		len(msgs), userCount, assistantCount)
	if toolCount > 0 {
		fmt.Fprintf(&sb, ", %d tool calls", toolCount)
	}
	sb.WriteString("\n")

	if lastUser != "" {
		fmt.Fprintf(&sb, "Last request: %s\n", lastUser)
	}
	if lastAssistant != "" {
		fmt.Fprintf(&sb, "Last response: %s\n", lastAssistant)
	}

	return sb.String()
}

