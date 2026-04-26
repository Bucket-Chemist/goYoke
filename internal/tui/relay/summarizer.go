// Package relay provides optional downstream notification sinks that consume
// the observability SnapshotStore's publication stream.
//
// Each sink is independent and optional: if the configuration for a sink (e.g.
// a webhook URL) is absent, the sink does nothing and no internal TUI packages
// or protocol types are modified.
package relay

import (
	"fmt"
	"strings"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// Summarize produces a Discord-friendly message describing what changed
// between old and new snapshots.
//
// Returns ("", false) when PublishHash is unchanged — meaning no human
// notification is warranted. Returns a non-empty message and true when a
// notification should be sent.
//
// The first call (old is zero value with empty PublishHash) always produces a
// summary when new.PublishHash is non-empty.
func Summarize(old, new harnessproto.SessionSnapshot) (string, bool) {
	if old.PublishHash != "" && old.PublishHash == new.PublishHash {
		return "", false
	}

	var parts []string

	// Status transitions
	if old.Status != new.Status {
		parts = append(parts, fmt.Sprintf("Status: %s → %s", statusLabel(old.Status), statusLabel(new.Status)))
	}

	// Model/effort changes
	if old.Model != new.Model && new.Model != "" {
		parts = append(parts, fmt.Sprintf("Model: %s", new.Model))
	}
	if old.Effort != new.Effort && new.Effort != "" {
		parts = append(parts, fmt.Sprintf("Effort: %s", new.Effort))
	}

	// New error state
	if new.LastError != "" && old.LastError != new.LastError {
		parts = append(parts, fmt.Sprintf("Error: %s", truncate(new.LastError, 120)))
	}

	// Modal / permission requests
	if new.Pending != nil {
		if old.Pending == nil || old.Pending.Kind != new.Pending.Kind || old.Pending.Message != new.Pending.Message {
			switch new.Pending.Kind {
			case "permission":
				parts = append(parts, fmt.Sprintf("Permission request: %s", truncate(new.Pending.Message, 120)))
			case "modal":
				parts = append(parts, fmt.Sprintf("Modal: %s", truncate(new.Pending.Message, 120)))
			default:
				parts = append(parts, fmt.Sprintf("Prompt (%s): %s", new.Pending.Kind, truncate(new.Pending.Message, 120)))
			}
		}
	}

	// Agent lifecycle
	if summary := agentDiff(old.Agents, new.Agents); summary != "" {
		parts = append(parts, summary)
	}

	// Team lifecycle
	switch {
	case new.Team != nil && old.Team == nil:
		parts = append(parts, fmt.Sprintf("Team started: %s (%d members)", new.Team.Name, new.Team.Members))
	case new.Team != nil && old.Team != nil && old.Team.Status != new.Team.Status:
		parts = append(parts, fmt.Sprintf("Team %s: %s", new.Team.Name, new.Team.Status))
	case new.Team == nil && old.Team != nil:
		parts = append(parts, "Team completed")
	}

	// Response completion: streaming ended and an assistant reply is present
	if old.Streaming && !new.Streaming && new.LastAssistant != "" {
		parts = append(parts, fmt.Sprintf("Response: %s", truncate(new.LastAssistant, 160)))
	}

	if len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("Session update (status: %s)", new.Status))
	}

	return strings.Join(parts, "\n"), true
}

// statusLabel returns a human-readable label for a harnessproto status string.
func statusLabel(s string) string {
	switch s {
	case "idle":
		return "idle"
	case "streaming":
		return "streaming"
	case "waiting_modal":
		return "waiting (modal)"
	case "waiting_permission":
		return "waiting (permission)"
	case "shutting_down":
		return "shutting down"
	case "":
		return "unknown"
	default:
		return s
	}
}

// agentDiff summarises agents that appeared, disappeared, or changed status.
func agentDiff(old, new []harnessproto.AgentSummary) string {
	oldByID := make(map[string]harnessproto.AgentSummary, len(old))
	for _, a := range old {
		oldByID[a.ID] = a
	}

	newByID := make(map[string]struct{}, len(new))
	for _, a := range new {
		newByID[a.ID] = struct{}{}
	}

	var notes []string
	for _, a := range new {
		prev, known := oldByID[a.ID]
		if !known {
			notes = append(notes, fmt.Sprintf("Agent started: %s", agentLabel(a)))
		} else if prev.Status != a.Status && a.Status != "" {
			notes = append(notes, fmt.Sprintf("Agent %s: %s", agentLabel(a), a.Status))
		}
	}
	for _, a := range old {
		if _, still := newByID[a.ID]; !still {
			notes = append(notes, fmt.Sprintf("Agent done: %s", agentLabel(a)))
		}
	}

	return strings.Join(notes, ", ")
}

func agentLabel(a harnessproto.AgentSummary) string {
	if a.Name != "" {
		return a.Name
	}
	return a.ID
}

// truncate shortens s to at most maxLen bytes, appending "…" when truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…"
}
