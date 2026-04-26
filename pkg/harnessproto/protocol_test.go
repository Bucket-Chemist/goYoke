package harnessproto_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// marshalThenUnmarshal round-trips v through JSON encoding and decoding,
// returning the decoded value and the raw JSON bytes for inspection.
func marshalThenUnmarshal[T any](t *testing.T, v T) (T, []byte) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out, b
}

// -----------------------------------------------------------------------
// Constants
// -----------------------------------------------------------------------

func TestProtocolConstants(t *testing.T) {
	if harnessproto.ProtocolName == "" {
		t.Error("ProtocolName must not be empty")
	}
	if harnessproto.ProtocolVersion == "" {
		t.Error("ProtocolVersion must not be empty")
	}
}

// -----------------------------------------------------------------------
// Request round-trips
// -----------------------------------------------------------------------

func TestRequest_PingRoundTrip(t *testing.T) {
	req := harnessproto.Request{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            harnessproto.KindPing,
	}

	got, raw := marshalThenUnmarshal(t, req)

	if got.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol: got %q want %q", got.Protocol, harnessproto.ProtocolName)
	}
	if got.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion: got %q want %q", got.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if got.Kind != harnessproto.KindPing {
		t.Errorf("Kind: got %q want %q", got.Kind, harnessproto.KindPing)
	}
	if got.Payload != nil {
		t.Errorf("Payload: expected nil for ping, got %s", got.Payload)
	}

	// omitempty: "payload" key must be absent from the JSON when empty.
	if strings.Contains(string(raw), `"payload"`) {
		t.Errorf("unexpected payload key in JSON: %s", raw)
	}
}

func TestRequest_SubmitPromptRoundTrip(t *testing.T) {
	payload := harnessproto.SubmitPromptRequest{Text: "hello world"}
	payloadBytes, _ := json.Marshal(payload)

	req := harnessproto.Request{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            harnessproto.KindSubmitPrompt,
		Payload:         payloadBytes,
	}

	got, _ := marshalThenUnmarshal(t, req)

	if got.Kind != harnessproto.KindSubmitPrompt {
		t.Errorf("Kind: got %q", got.Kind)
	}

	var decoded harnessproto.SubmitPromptRequest
	if err := json.Unmarshal(got.Payload, &decoded); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if decoded.Text != "hello world" {
		t.Errorf("Text: got %q", decoded.Text)
	}
}

func TestRequest_AllKindsRoundTrip(t *testing.T) {
	kinds := []string{
		harnessproto.KindPing,
		harnessproto.KindGetSnapshot,
		harnessproto.KindSubmitPrompt,
		harnessproto.KindInterrupt,
		harnessproto.KindRespondModal,
		harnessproto.KindRespondPermission,
		harnessproto.KindSetModel,
		harnessproto.KindSetEffort,
		harnessproto.KindSetCWD,
	}

	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			req := harnessproto.Request{
				Protocol:        harnessproto.ProtocolName,
				ProtocolVersion: harnessproto.ProtocolVersion,
				Kind:            kind,
			}
			got, _ := marshalThenUnmarshal(t, req)
			if got.Kind != kind {
				t.Errorf("Kind: got %q want %q", got.Kind, kind)
			}
			if got.ProtocolVersion != harnessproto.ProtocolVersion {
				t.Errorf("ProtocolVersion lost in round-trip")
			}
		})
	}
}

// -----------------------------------------------------------------------
// Response round-trips
// -----------------------------------------------------------------------

func TestResponse_SuccessNoPayload(t *testing.T) {
	resp := harnessproto.Response{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            harnessproto.KindPing,
		OK:              true,
	}

	got, raw := marshalThenUnmarshal(t, resp)

	if !got.OK {
		t.Error("OK should be true")
	}
	if got.Error != nil {
		t.Errorf("Error should be nil on success, got %+v", got.Error)
	}
	if strings.Contains(string(raw), `"error"`) {
		t.Errorf("error key must be absent in success response: %s", raw)
	}
	if strings.Contains(string(raw), `"payload"`) {
		t.Errorf("payload key must be absent when empty: %s", raw)
	}
}

func TestResponse_ErrorRoundTrip(t *testing.T) {
	resp := harnessproto.Response{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            harnessproto.KindSubmitPrompt,
		OK:              false,
		Error: &harnessproto.ErrorDetail{
			Code:    harnessproto.ErrUnavailableState,
			Message: "no active session",
		},
	}

	got, _ := marshalThenUnmarshal(t, resp)

	if got.OK {
		t.Error("OK must be false")
	}
	if got.Error == nil {
		t.Fatal("Error must not be nil")
	}
	if got.Error.Code != harnessproto.ErrUnavailableState {
		t.Errorf("Error.Code: got %q want %q", got.Error.Code, harnessproto.ErrUnavailableState)
	}
	if got.Error.Message != "no active session" {
		t.Errorf("Error.Message: got %q", got.Error.Message)
	}
}

func TestResponse_AllErrorCodes(t *testing.T) {
	codes := []string{
		harnessproto.ErrUnsupportedOperation,
		harnessproto.ErrBadRequest,
		harnessproto.ErrUnavailableState,
		harnessproto.ErrVersionMismatch,
	}

	for _, code := range codes {
		t.Run(code, func(t *testing.T) {
			resp := harnessproto.Response{
				Protocol:        harnessproto.ProtocolName,
				ProtocolVersion: harnessproto.ProtocolVersion,
				Kind:            harnessproto.KindPing,
				OK:              false,
				Error:           &harnessproto.ErrorDetail{Code: code, Message: "test"},
			}
			got, _ := marshalThenUnmarshal(t, resp)
			if got.Error.Code != code {
				t.Errorf("Code: got %q want %q", got.Error.Code, code)
			}
		})
	}
}

func TestResponse_ProtocolVersionPreserved(t *testing.T) {
	resp := harnessproto.Response{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            harnessproto.KindGetSnapshot,
		OK:              true,
	}

	got, raw := marshalThenUnmarshal(t, resp)

	if got.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol: got %q want %q", got.Protocol, harnessproto.ProtocolName)
	}
	if got.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion: got %q want %q", got.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if !strings.Contains(string(raw), `"protocol_version"`) {
		t.Errorf("protocol_version must always be present in JSON: %s", raw)
	}
}

// -----------------------------------------------------------------------
// SessionSnapshot round-trips
// -----------------------------------------------------------------------

func TestSessionSnapshot_MinimalRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	snap := harnessproto.SessionSnapshot{
		Timestamp:       now,
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Status:          "idle",
		Streaming:       false,
		Agents:          []harnessproto.AgentSummary{},
		StateHash:       "abc123",
		PublishHash:     "def456",
	}

	got, raw := marshalThenUnmarshal(t, snap)

	if got.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol: got %q", got.Protocol)
	}
	if got.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion: got %q", got.ProtocolVersion)
	}
	if got.Status != "idle" {
		t.Errorf("Status: got %q", got.Status)
	}
	if got.StateHash != "abc123" {
		t.Errorf("StateHash: got %q", got.StateHash)
	}
	if got.PublishHash != "def456" {
		t.Errorf("PublishHash: got %q", got.PublishHash)
	}
	if got.Streaming {
		t.Error("Streaming should be false")
	}

	// Optional zero-value fields must not appear in JSON.
	for _, key := range []string{
		`"session_id"`, `"provider"`, `"model"`, `"effort"`, `"cwd"`,
		`"reconnecting"`, `"shutting_down"`,
		`"active_tab"`, `"focus"`,
		`"plan_active"`, `"plan_step"`, `"plan_total"`,
		`"team"`, `"pending"`,
		`"last_user"`, `"last_assistant"`, `"last_error"`, `"highlights"`,
	} {
		if strings.Contains(string(raw), key) {
			t.Errorf("optional field %s must be absent when zero-valued: %s", key, raw)
		}
	}

	// Required fields must always appear.
	for _, key := range []string{
		`"protocol"`, `"protocol_version"`, `"status"`, `"streaming"`,
		`"state_hash"`, `"publish_hash"`, `"timestamp"`,
	} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("required field %s must be present: %s", key, raw)
		}
	}
}

func TestSessionSnapshot_FullRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	snap := harnessproto.SessionSnapshot{
		Timestamp:       now,
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		SessionID:       "sess-001",
		Provider:        "anthropic",
		Model:           "claude-sonnet-4-6",
		Effort:          "normal",
		CWD:             "/home/user/project",
		Status:          "streaming",
		Streaming:       true,
		Reconnecting:    false,
		ShuttingDown:    false,
		ActiveTab:       "chat",
		Focus:           "input",
		PlanActive:      true,
		PlanStep:        2,
		PlanTotal:       5,
		Agents: []harnessproto.AgentSummary{
			{ID: "a1", Name: "go-pro", Status: "running", Model: "sonnet"},
		},
		Team: &harnessproto.TeamSummary{
			ID:      "t1",
			Name:    "review-team",
			Status:  "active",
			Members: 3,
		},
		Pending: &harnessproto.PendingPrompt{
			Kind:    "modal",
			Message: "Confirm action?",
		},
		LastUser:      "implement the feature",
		LastAssistant: "Done.",
		LastError:     "",
		Highlights:    []string{"Build passed", "Tests green"},
		StateHash:     "state-xyz",
		PublishHash:   "pub-xyz",
	}

	got, _ := marshalThenUnmarshal(t, snap)

	if got.SessionID != "sess-001" {
		t.Errorf("SessionID: got %q", got.SessionID)
	}
	if got.Model != "claude-sonnet-4-6" {
		t.Errorf("Model: got %q", got.Model)
	}
	if !got.PlanActive {
		t.Error("PlanActive should be true")
	}
	if got.PlanStep != 2 || got.PlanTotal != 5 {
		t.Errorf("PlanStep/Total: got %d/%d", got.PlanStep, got.PlanTotal)
	}
	if len(got.Agents) != 1 || got.Agents[0].ID != "a1" {
		t.Errorf("Agents: got %+v", got.Agents)
	}
	if got.Team == nil || got.Team.ID != "t1" {
		t.Errorf("Team: got %+v", got.Team)
	}
	if got.Team.Members != 3 {
		t.Errorf("Team.Members: got %d", got.Team.Members)
	}
	if got.Pending == nil || got.Pending.Kind != "modal" {
		t.Errorf("Pending: got %+v", got.Pending)
	}
	if len(got.Highlights) != 2 {
		t.Errorf("Highlights: got %v", got.Highlights)
	}
	if got.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion lost in full round-trip")
	}
}

func TestSessionSnapshot_OptionalBoolsOmittedWhenFalse(t *testing.T) {
	snap := harnessproto.SessionSnapshot{
		Timestamp:       time.Now().UTC(),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Status:          "idle",
		Streaming:       false,
		Reconnecting:    false,
		ShuttingDown:    false,
		Agents:          []harnessproto.AgentSummary{},
		StateHash:       "h1",
		PublishHash:     "h2",
	}

	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(b)

	// Reconnecting and ShuttingDown are omitempty — must be absent when false.
	if strings.Contains(raw, `"reconnecting"`) {
		t.Errorf("reconnecting must be absent when false: %s", raw)
	}
	if strings.Contains(raw, `"shutting_down"`) {
		t.Errorf("shutting_down must be absent when false: %s", raw)
	}

	// Streaming has no omitempty — must be present even when false.
	if !strings.Contains(raw, `"streaming"`) {
		t.Errorf("streaming must always be present: %s", raw)
	}
}

func TestSessionSnapshot_StreamingTruePresent(t *testing.T) {
	snap := harnessproto.SessionSnapshot{
		Timestamp:       time.Now().UTC(),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Status:          "streaming",
		Streaming:       true,
		Agents:          []harnessproto.AgentSummary{},
		StateHash:       "h1",
		PublishHash:     "h2",
	}

	b, _ := json.Marshal(snap)
	if !strings.Contains(string(b), `"streaming":true`) {
		t.Errorf("streaming:true must be present: %s", b)
	}
}

// -----------------------------------------------------------------------
// Payload type round-trips
// -----------------------------------------------------------------------

func TestPayloadTypes_RoundTrip(t *testing.T) {
	t.Run("RespondModalRequest", func(t *testing.T) {
		v := harnessproto.RespondModalRequest{Selection: "yes"}
		got, _ := marshalThenUnmarshal(t, v)
		if got.Selection != "yes" {
			t.Errorf("Selection: got %q", got.Selection)
		}
	})

	t.Run("RespondPermissionRequest_allow", func(t *testing.T) {
		v := harnessproto.RespondPermissionRequest{Allow: true}
		got, _ := marshalThenUnmarshal(t, v)
		if !got.Allow {
			t.Error("Allow should be true")
		}
	})

	t.Run("RespondPermissionRequest_deny", func(t *testing.T) {
		v := harnessproto.RespondPermissionRequest{Allow: false}
		got, _ := marshalThenUnmarshal(t, v)
		if got.Allow {
			t.Error("Allow should be false")
		}
	})

	t.Run("SetModelRequest", func(t *testing.T) {
		v := harnessproto.SetModelRequest{Model: "claude-opus-4-7"}
		got, _ := marshalThenUnmarshal(t, v)
		if got.Model != "claude-opus-4-7" {
			t.Errorf("Model: got %q", got.Model)
		}
	})

	t.Run("SetEffortRequest", func(t *testing.T) {
		v := harnessproto.SetEffortRequest{Effort: "high"}
		got, _ := marshalThenUnmarshal(t, v)
		if got.Effort != "high" {
			t.Errorf("Effort: got %q", got.Effort)
		}
	})

	t.Run("SetCWDRequest", func(t *testing.T) {
		v := harnessproto.SetCWDRequest{CWD: "/tmp/work"}
		got, _ := marshalThenUnmarshal(t, v)
		if got.CWD != "/tmp/work" {
			t.Errorf("CWD: got %q", got.CWD)
		}
	})
}
