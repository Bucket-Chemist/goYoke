package relay

import (
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// helpers

func snapStatus(publishHash, status string) harnessproto.SessionSnapshot {
	return harnessproto.SessionSnapshot{
		PublishHash: publishHash,
		Status:      status,
		Agents:      []harnessproto.AgentSummary{},
	}
}

// TestSummarize_UnchangedPublishHash_Suppressed verifies the deduplication
// path: identical PublishHash → no notification.
func TestSummarize_UnchangedPublishHash_Suppressed(t *testing.T) {
	old := snapStatus("hash-abc", "idle")
	new := snapStatus("hash-abc", "streaming") // state changed but publish hash unchanged

	msg, ok := Summarize(old, new)
	if ok {
		t.Errorf("expected ok=false when PublishHash unchanged, got ok=true msg=%q", msg)
	}
	if msg != "" {
		t.Errorf("expected empty message, got %q", msg)
	}
}

// TestSummarize_EmptyOldPublishHash_AlwaysSends verifies that the initial
// transition (old hash is zero value) is not suppressed.
func TestSummarize_EmptyOldPublishHash_AlwaysSends(t *testing.T) {
	old := harnessproto.SessionSnapshot{Status: "idle", Agents: []harnessproto.AgentSummary{}}
	new := snapStatus("hash-1", "streaming")

	_, ok := Summarize(old, new)
	if !ok {
		t.Error("expected ok=true when old has empty PublishHash and new has non-empty")
	}
}

func TestSummarize_StatusTransition(t *testing.T) {
	old := snapStatus("hash-1", "idle")
	new := snapStatus("hash-2", "streaming")

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true when PublishHash changed")
	}
	if !strings.Contains(msg, "Status:") {
		t.Errorf("expected status transition in message, got %q", msg)
	}
	if !strings.Contains(msg, "idle") || !strings.Contains(msg, "streaming") {
		t.Errorf("expected both status values in message, got %q", msg)
	}
}

func TestSummarize_ErrorAppears(t *testing.T) {
	old := snapStatus("hash-1", "streaming")
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "idle",
		LastError:   "claude: connection reset by peer",
		Agents:      []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true when PublishHash changed")
	}
	if !strings.Contains(msg, "Error:") {
		t.Errorf("expected Error: prefix in message, got %q", msg)
	}
	if !strings.Contains(msg, "connection reset") {
		t.Errorf("expected error text in message, got %q", msg)
	}
}

func TestSummarize_PendingPermissionRequest(t *testing.T) {
	old := snapStatus("hash-1", "streaming")
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "waiting_permission",
		Pending: &harnessproto.PendingPrompt{
			Kind:    "permission",
			Message: "allow tool: bash",
		},
		Agents: []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Permission request") {
		t.Errorf("expected Permission request in message, got %q", msg)
	}
	if !strings.Contains(msg, "bash") {
		t.Errorf("expected tool name in message, got %q", msg)
	}
}

func TestSummarize_PendingModal(t *testing.T) {
	old := snapStatus("hash-1", "streaming")
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "waiting_modal",
		Pending: &harnessproto.PendingPrompt{
			Kind:    "modal",
			Message: "Choose an option",
		},
		Agents: []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Modal:") {
		t.Errorf("expected Modal: in message, got %q", msg)
	}
}

func TestSummarize_AgentStarted(t *testing.T) {
	old := harnessproto.SessionSnapshot{
		PublishHash: "hash-1",
		Status:      "idle",
		Agents:      []harnessproto.AgentSummary{},
	}
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "streaming",
		Agents: []harnessproto.AgentSummary{
			{ID: "agent-1", Name: "go-pro", Status: "running"},
		},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Agent started") {
		t.Errorf("expected 'Agent started' in message, got %q", msg)
	}
	if !strings.Contains(msg, "go-pro") {
		t.Errorf("expected agent name in message, got %q", msg)
	}
}

func TestSummarize_AgentCompleted(t *testing.T) {
	old := harnessproto.SessionSnapshot{
		PublishHash: "hash-1",
		Status:      "streaming",
		Agents: []harnessproto.AgentSummary{
			{ID: "agent-1", Name: "go-pro", Status: "running"},
		},
	}
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "idle",
		Agents:      []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Agent done") {
		t.Errorf("expected 'Agent done' in message, got %q", msg)
	}
}

func TestSummarize_ResponseCompletion(t *testing.T) {
	old := harnessproto.SessionSnapshot{
		PublishHash: "hash-1",
		Status:      "streaming",
		Streaming:   true,
		Agents:      []harnessproto.AgentSummary{},
	}
	new := harnessproto.SessionSnapshot{
		PublishHash:   "hash-2",
		Status:        "idle",
		Streaming:     false,
		LastAssistant: "Here is the implementation you requested.",
		Agents:        []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Response:") {
		t.Errorf("expected Response: in message, got %q", msg)
	}
	if !strings.Contains(msg, "implementation") {
		t.Errorf("expected response text in message, got %q", msg)
	}
}

func TestSummarize_TeamStarted(t *testing.T) {
	old := snapStatus("hash-1", "idle")
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "streaming",
		Team: &harnessproto.TeamSummary{
			ID:      "team-1",
			Name:    "review-team",
			Status:  "running",
			Members: 3,
		},
		Agents: []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Team started") {
		t.Errorf("expected 'Team started' in message, got %q", msg)
	}
}

func TestSummarize_ModelChanged(t *testing.T) {
	old := harnessproto.SessionSnapshot{
		PublishHash: "hash-1",
		Status:      "idle",
		Model:       "claude-sonnet-4-6",
		Agents:      []harnessproto.AgentSummary{},
	}
	new := harnessproto.SessionSnapshot{
		PublishHash: "hash-2",
		Status:      "idle",
		Model:       "claude-opus-4-7",
		Agents:      []harnessproto.AgentSummary{},
	}

	msg, ok := Summarize(old, new)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(msg, "Model:") {
		t.Errorf("expected Model: in message, got %q", msg)
	}
	if !strings.Contains(msg, "claude-opus-4-7") {
		t.Errorf("expected new model name in message, got %q", msg)
	}
}

func TestTruncate_ShortString_Unchanged(t *testing.T) {
	s := "hello"
	if got := truncate(s, 100); got != s {
		t.Errorf("truncate changed short string: got %q want %q", got, s)
	}
}

func TestTruncate_LongString_Ellipsis(t *testing.T) {
	long := strings.Repeat("a", 200)
	got := truncate(long, 100)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncate did not append ellipsis: %q", got)
	}
	if len(got) > 110 { // 100 bytes + multibyte ellipsis is fine
		t.Errorf("truncate result too long: len=%d", len(got))
	}
}
