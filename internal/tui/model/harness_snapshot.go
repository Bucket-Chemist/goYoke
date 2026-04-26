// Package model defines shared state types for the goYoke TUI.
// This file implements the harness snapshot builder that translates live
// AppModel state into the public harnessproto.SessionSnapshot DTO.
// It accesses sharedState fields directly (same package) without adding
// getter-sprawl to unrelated packages, per the HL-003 architectural decision.
package model

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// BuildHarnessSnapshot translates the live AppModel state into a
// harnessproto.SessionSnapshot.
//
// The builder reads directly from AppModel fields, m.shared.* (same package),
// and widget interface methods. It never reads viewport text, ANSI output, or
// rendered strings, and does not start servers or write files.
func (m AppModel) BuildHarnessSnapshot() harnessproto.SessionSnapshot {
	snap := harnessproto.SessionSnapshot{
		Timestamp:       time.Now(),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		SessionID:       m.sessionID,
		Model:           m.activeModel,
		Effort:          m.activeEffort,
		ActiveTab:       m.activeTab.String(),
		Focus:           m.focus.String(),
		ShuttingDown:    m.shutdownInProgress,
		Reconnecting:    m.reconnectCount > 0,
		CWD:             m.statusLine.CWD,
		PlanActive:      m.statusLine.PlanActive,
		PlanStep:        m.statusLine.PlanStep,
		PlanTotal:       m.statusLine.PlanTotalSteps,
		Agents:          m.buildAgentSummaries(),
	}

	if m.shared != nil && m.shared.providerState != nil {
		snap.Provider = string(m.shared.providerState.GetActiveProvider())
	}

	// Streaming: authoritative from claudePanel when wired, falls back to statusLine.
	snap.Streaming = m.statusLine.Streaming
	if m.shared != nil && m.shared.claudePanel != nil {
		snap.Streaming = snap.Streaming || m.shared.claudePanel.IsStreaming()
	}

	snap.Status = m.deriveSnapshotStatus(snap.Streaming)
	snap.Team = m.buildTeamSummary()
	snap.Pending = m.buildPendingPrompt()
	m.populateLastMessages(&snap)

	snap.StateHash = computeShortHash(stateHashInput(snap))
	snap.PublishHash = computeShortHash(publishHashInput(snap))

	return snap
}

// deriveSnapshotStatus returns the coarse session-state label for Status.
// Priority (highest first): shutting_down > waiting_permission > waiting_modal > streaming > idle.
func (m AppModel) deriveSnapshotStatus(streaming bool) string {
	if m.shutdownInProgress {
		return "shutting_down"
	}
	if m.shared != nil {
		if m.shared.permHandler != nil && m.shared.permHandler.HasActivePermissionGate() {
			return "waiting_permission"
		}
		if m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
			return "waiting_modal"
		}
		if m.shared.drawerStack != nil && m.shared.drawerStack.HasActiveModal() {
			return "waiting_modal"
		}
	}
	if streaming {
		return "streaming"
	}
	return "idle"
}

// buildAgentSummaries converts the registry tree into lightweight AgentSummary
// DTOs. Returns an empty (non-nil) slice when no agents are registered.
func (m AppModel) buildAgentSummaries() []harnessproto.AgentSummary {
	if m.shared == nil || m.shared.agentRegistry == nil {
		return []harnessproto.AgentSummary{}
	}
	nodes := m.shared.agentRegistry.Tree()
	summaries := make([]harnessproto.AgentSummary, 0, len(nodes))
	for _, node := range nodes {
		if node == nil || node.Agent == nil {
			continue
		}
		a := node.Agent
		name := a.Description
		if name == "" {
			name = a.AgentType
		}
		summaries = append(summaries, harnessproto.AgentSummary{
			ID:     a.ID,
			Name:   name,
			Status: a.Status.String(),
			Model:  a.Model,
		})
	}
	return summaries
}

// buildTeamSummary returns a TeamSummary from the teamsHealth widget, or nil
// when no team is active.
func (m AppModel) buildTeamSummary() *harnessproto.TeamSummary {
	if m.shared == nil || m.shared.teamsHealth == nil {
		return nil
	}
	ind := m.shared.teamsHealth.TeamIndicator()
	if !ind.Active {
		return nil
	}
	return &harnessproto.TeamSummary{
		ID:      ind.Name,
		Name:    ind.Name,
		Status:  "running",
		Members: len(ind.MemberStatuses),
	}
}

// buildPendingPrompt returns a PendingPrompt descriptor when a modal or
// permission gate is waiting for user input, or nil otherwise.
//
// Detection priority: drawer modal → permission gate → queued modal overlay.
func (m AppModel) buildPendingPrompt() *harnessproto.PendingPrompt {
	if m.shared == nil {
		return nil
	}

	// Options-drawer modal (non-compact tier).
	if m.shared.drawerStack != nil && m.shared.drawerStack.HasActiveModal() {
		return &harnessproto.PendingPrompt{Kind: "modal"}
	}

	// Tool-permission gate (FlowToolPermission via PermissionHandler).
	if m.shared.permHandler != nil && m.shared.permHandler.HasActivePermissionGate() {
		msg := ""
		if m.shared.modalQueue != nil {
			msg = m.shared.modalQueue.ActiveRequestMessage()
		}
		return &harnessproto.PendingPrompt{Kind: "permission", Message: msg}
	}

	// Regular ask/confirm/input modal overlay.
	if m.shared.modalQueue != nil && m.shared.modalQueue.IsActive() {
		return &harnessproto.PendingPrompt{
			Kind:    "modal",
			Message: m.shared.modalQueue.ActiveRequestMessage(),
		}
	}

	return nil
}

// populateLastMessages scans the conversation history from newest to oldest and
// fills snap.LastUser, snap.LastAssistant, and snap.LastError.
// Stops scanning as soon as all three fields are populated.
func (m AppModel) populateLastMessages(snap *harnessproto.SessionSnapshot) {
	if m.shared == nil || m.shared.claudePanel == nil {
		return
	}
	msgs := m.shared.claudePanel.SaveMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		switch msg.Role {
		case "user":
			if snap.LastUser == "" {
				snap.LastUser = truncateRunes(msg.Content, 200)
			}
		case "assistant":
			if snap.LastAssistant == "" {
				snap.LastAssistant = truncateRunes(msg.Content, 200)
			}
		case "system":
			if snap.LastError == "" && looksLikeError(msg.Content) {
				snap.LastError = truncateRunes(msg.Content, 200)
			}
		}
		if snap.LastUser != "" && snap.LastAssistant != "" && snap.LastError != "" {
			break
		}
	}
}

// truncateRunes limits s to at most n runes, appending "…" when truncated.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// looksLikeError returns true for system messages that represent error conditions.
func looksLikeError(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "disconnected")
}

// stateHashInput constructs the canonical string hashed for StateHash.
// Changes on every operator-visible state mutation.
func stateHashInput(s harnessproto.SessionSnapshot) string {
	pending := ""
	if s.Pending != nil {
		pending = s.Pending.Kind
	}
	team := ""
	if s.Team != nil {
		team = s.Team.ID
	}
	return fmt.Sprintf("%s|%s|%v|%s|%s|%s|%v|%v|%d|%d|%s|%s",
		s.Status, s.Model, s.Streaming,
		s.LastUser, s.LastAssistant, s.LastError,
		s.PlanActive, s.ShuttingDown,
		len(s.Agents), s.PlanStep,
		pending, team,
	)
}

// publishHashInput constructs the canonical string hashed for PublishHash.
// Changes only when a human notification is warranted: streaming completes,
// a new assistant message arrives, an error surfaces, or a prompt appears.
func publishHashInput(s harnessproto.SessionSnapshot) string {
	pending := ""
	if s.Pending != nil {
		pending = s.Pending.Kind
	}
	return fmt.Sprintf("%s|%s|%s|%s|%v",
		s.Status, s.LastAssistant, s.LastError, pending, s.ShuttingDown,
	)
}

// computeShortHash returns the first 16 hex characters of the SHA-256 of input.
func computeShortHash(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:8])
}
