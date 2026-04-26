package harnesscontrol_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/harnesscontrol"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// -----------------------------------------------------------------------
// Test harness
// -----------------------------------------------------------------------

// testHarness wires up a real Server with a mock sendMsg and an in-process
// SnapshotStore.
type testHarness struct {
	store  *observability.SnapshotStore
	server *harnesscontrol.Server
	sock   string

	mu   sync.Mutex
	msgs []tea.Msg
}

func newHarness(t *testing.T) *testHarness {
	t.Helper()
	dir := t.TempDir()
	sock := filepath.Join(dir, "test.sock")

	h := &testHarness{
		store: observability.New(),
		sock:  sock,
	}
	h.server = harnesscontrol.NewServer(h.store, h.record, sock)
	if err := h.server.Start(); err != nil {
		t.Fatalf("Server.Start: %v", err)
	}
	t.Cleanup(func() { h.server.Stop() })
	return h
}

// record is the sendMsg callback injected into the server. It stores every
// received message and immediately acknowledges action messages via their
// ResponseCh so the server goroutine never blocks.
func (h *testHarness) record(msg tea.Msg) {
	h.mu.Lock()
	h.msgs = append(h.msgs, msg)
	h.mu.Unlock()

	switch m := msg.(type) {
	case model.RemoteSubmitPromptMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteInterruptMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteRespondModalMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteRespondPermissionMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteSetModelMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteSetEffortMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	case model.RemoteSetCWDMsg:
		if m.ResponseCh != nil {
			m.ResponseCh <- nil
		}
	}
}

// collectedMsgs returns a snapshot of received messages under the lock.
func (h *testHarness) collectedMsgs() []tea.Msg {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]tea.Msg, len(h.msgs))
	copy(out, h.msgs)
	return out
}

// roundtrip sends one request and returns the decoded response on a fresh
// connection. The connection is closed after the exchange.
func (h *testHarness) roundtrip(t *testing.T, req harnessproto.Request) harnessproto.Response {
	t.Helper()
	conn, err := net.Dial("unix", h.sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	return exchange(t, conn, req)
}

// exchange writes req as a JSON line and reads one JSON response line.
func exchange(t *testing.T, conn net.Conn, req harnessproto.Request) harnessproto.Response {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("write request: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatalf("no response line (scan returned false)")
	}
	var resp harnessproto.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

// makeReq constructs a minimal valid Request envelope.
func makeReq(kind string, payload any) harnessproto.Request {
	req := harnessproto.Request{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            kind,
	}
	if payload != nil {
		raw, _ := json.Marshal(payload)
		req.Payload = json.RawMessage(raw)
	}
	return req
}

// publishSnap stores a ready-to-use SessionSnapshot in the store.
func publishSnap(store *observability.SnapshotStore, status, stateHash string) {
	store.Update(harnessproto.SessionSnapshot{
		Timestamp:       time.Now(),
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Status:          status,
		StateHash:       stateHash,
		PublishHash:     stateHash,
		Agents:          []harnessproto.AgentSummary{},
	})
}

// -----------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------

func TestPing(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindPing, nil))

	if !resp.OK {
		t.Fatalf("ping: expected OK=true, got error %+v", resp.Error)
	}
	if resp.Kind != harnessproto.KindPing {
		t.Errorf("ping: Kind = %q, want %q", resp.Kind, harnessproto.KindPing)
	}
	if resp.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ping: ProtocolVersion = %q, want %q", resp.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if resp.Protocol != harnessproto.ProtocolName {
		t.Errorf("ping: Protocol = %q, want %q", resp.Protocol, harnessproto.ProtocolName)
	}
}

func TestGetSnapshot_Empty(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindGetSnapshot, nil))

	if resp.OK {
		t.Fatal("get_snapshot on empty store: expected OK=false")
	}
	if resp.Error == nil {
		t.Fatal("get_snapshot on empty store: expected non-nil Error")
	}
	if resp.Error.Code != harnessproto.ErrUnavailableState {
		t.Errorf("get_snapshot empty: Error.Code = %q, want %q",
			resp.Error.Code, harnessproto.ErrUnavailableState)
	}
}

func TestGetSnapshot_Populated(t *testing.T) {
	h := newHarness(t)
	publishSnap(h.store, "idle", "hash-abc")

	resp := h.roundtrip(t, makeReq(harnessproto.KindGetSnapshot, nil))

	if !resp.OK {
		t.Fatalf("get_snapshot: expected OK=true, got error %+v", resp.Error)
	}
	if resp.Kind != harnessproto.KindGetSnapshot {
		t.Errorf("get_snapshot: Kind = %q, want %q", resp.Kind, harnessproto.KindGetSnapshot)
	}

	var snap harnessproto.SessionSnapshot
	if err := json.Unmarshal(resp.Payload, &snap); err != nil {
		t.Fatalf("unmarshal snapshot payload: %v", err)
	}
	if snap.Status != "idle" {
		t.Errorf("snapshot Status = %q, want %q", snap.Status, "idle")
	}
	if snap.StateHash != "hash-abc" {
		t.Errorf("snapshot StateHash = %q, want %q", snap.StateHash, "hash-abc")
	}
}

func TestSubmitPrompt_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindSubmitPrompt,
		harnessproto.SubmitPromptRequest{Text: "hello world"}))

	if !resp.OK {
		t.Fatalf("submit_prompt: expected OK=true, got error %+v", resp.Error)
	}
	if resp.Kind != harnessproto.KindSubmitPrompt {
		t.Errorf("submit_prompt: Kind = %q, want %q", resp.Kind, harnessproto.KindSubmitPrompt)
	}

	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteSubmitPromptMsg)
	if !ok {
		t.Fatalf("injected message type = %T, want RemoteSubmitPromptMsg", msgs[0])
	}
	if m.Prompt != "hello world" {
		t.Errorf("Prompt = %q, want %q", m.Prompt, "hello world")
	}
}

func TestSubmitPrompt_EmptyText(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindSubmitPrompt,
		harnessproto.SubmitPromptRequest{Text: ""}))

	if resp.OK {
		t.Fatal("submit_prompt with empty text: expected OK=false")
	}
	if resp.Error == nil || resp.Error.Code != harnessproto.ErrBadRequest {
		t.Errorf("submit_prompt empty text: Error.Code = %+v, want %q", resp.Error, harnessproto.ErrBadRequest)
	}
}

func TestInterrupt_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindInterrupt, nil))

	if !resp.OK {
		t.Fatalf("interrupt: expected OK=true, got error %+v", resp.Error)
	}

	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	if _, ok := msgs[0].(model.RemoteInterruptMsg); !ok {
		t.Fatalf("injected message type = %T, want RemoteInterruptMsg", msgs[0])
	}
}

func TestInvalidJSON(t *testing.T) {
	h := newHarness(t)

	conn, err := net.Dial("unix", h.sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("{not valid json}\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		t.Fatal("no response line")
	}
	var resp harnessproto.Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.OK {
		t.Fatal("malformed JSON: expected OK=false")
	}
	if resp.Error == nil || resp.Error.Code != harnessproto.ErrBadRequest {
		t.Errorf("malformed JSON: Error.Code = %+v, want %q", resp.Error, harnessproto.ErrBadRequest)
	}
}

func TestUnknownKind(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq("does_not_exist", nil))

	if resp.OK {
		t.Fatal("unknown kind: expected OK=false")
	}
	if resp.Error == nil || resp.Error.Code != harnessproto.ErrUnsupportedOperation {
		t.Errorf("unknown kind: Error.Code = %+v, want %q", resp.Error, harnessproto.ErrUnsupportedOperation)
	}
}

func TestSetModel_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindSetModel,
		harnessproto.SetModelRequest{Model: "sonnet"}))

	if !resp.OK {
		t.Fatalf("set_model: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteSetModelMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteSetModelMsg", msgs[0])
	}
	if m.ModelID != "sonnet" {
		t.Errorf("ModelID = %q, want %q", m.ModelID, "sonnet")
	}
}

func TestSetEffort_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindSetEffort,
		harnessproto.SetEffortRequest{Effort: "high"}))

	if !resp.OK {
		t.Fatalf("set_effort: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteSetEffortMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteSetEffortMsg", msgs[0])
	}
	if m.Level != "high" {
		t.Errorf("Level = %q, want %q", m.Level, "high")
	}
}

func TestSetCWD_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindSetCWD,
		harnessproto.SetCWDRequest{CWD: "/tmp/workspace"}))

	if !resp.OK {
		t.Fatalf("set_cwd: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteSetCWDMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteSetCWDMsg", msgs[0])
	}
	if m.Path != "/tmp/workspace" {
		t.Errorf("Path = %q, want %q", m.Path, "/tmp/workspace")
	}
}

func TestRespondModal_Success(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindRespondModal,
		harnessproto.RespondModalRequest{Selection: "yes"}))

	if !resp.OK {
		t.Fatalf("respond_modal: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteRespondModalMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteRespondModalMsg", msgs[0])
	}
	if m.Value != "yes" {
		t.Errorf("Value = %q, want %q", m.Value, "yes")
	}
}

func TestRespondPermission_Allow(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindRespondPermission,
		harnessproto.RespondPermissionRequest{Allow: true}))

	if !resp.OK {
		t.Fatalf("respond_permission allow: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 injected message, got %d", len(msgs))
	}
	m, ok := msgs[0].(model.RemoteRespondPermissionMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteRespondPermissionMsg", msgs[0])
	}
	if m.Decision != "allow" {
		t.Errorf("Decision = %q, want %q", m.Decision, "allow")
	}
}

func TestRespondPermission_Deny(t *testing.T) {
	h := newHarness(t)
	resp := h.roundtrip(t, makeReq(harnessproto.KindRespondPermission,
		harnessproto.RespondPermissionRequest{Allow: false}))

	if !resp.OK {
		t.Fatalf("respond_permission deny: expected OK=true, got %+v", resp.Error)
	}
	msgs := h.collectedMsgs()
	m, ok := msgs[0].(model.RemoteRespondPermissionMsg)
	if !ok {
		t.Fatalf("injected type = %T, want RemoteRespondPermissionMsg", msgs[0])
	}
	if m.Decision != "deny" {
		t.Errorf("Decision = %q, want %q", m.Decision, "deny")
	}
}

// TestMultiRequestSingleConn verifies that the server handles multiple
// sequential requests on the same persistent connection.
func TestMultiRequestSingleConn(t *testing.T) {
	h := newHarness(t)
	publishSnap(h.store, "idle", "multi-hash")

	conn, err := net.Dial("unix", h.sock)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for i, kind := range []string{harnessproto.KindPing, harnessproto.KindGetSnapshot, harnessproto.KindPing} {
		req := makeReq(kind, nil)
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			t.Fatalf("request %d: write: %v", i, err)
		}
		if !scanner.Scan() {
			t.Fatalf("request %d: no response", i)
		}
		var resp harnessproto.Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			t.Fatalf("request %d: unmarshal: %v", i, err)
		}
		if !resp.OK {
			t.Errorf("request %d (%s): expected OK=true, got %+v", i, kind, resp.Error)
		}
	}
}

// TestConcurrentGetSnapshot fires N concurrent get_snapshot requests across
// separate connections to ensure there are no races or deadlocks.
func TestConcurrentGetSnapshot(t *testing.T) {
	h := newHarness(t)
	publishSnap(h.store, "idle", "concurrent-hash")

	const n = 20
	type result struct {
		idx int
		err error
	}
	results := make(chan result, n)

	for i := range n {
		go func(idx int) {
			conn, err := net.Dial("unix", h.sock)
			if err != nil {
				results <- result{idx, fmt.Errorf("dial: %w", err)}
				return
			}
			defer conn.Close()

			req := makeReq(harnessproto.KindGetSnapshot, nil)
			data, _ := json.Marshal(req)
			data = append(data, '\n')
			if _, err := conn.Write(data); err != nil {
				results <- result{idx, fmt.Errorf("write: %w", err)}
				return
			}

			scanner := bufio.NewScanner(conn)
			if !scanner.Scan() {
				results <- result{idx, fmt.Errorf("no response")}
				return
			}
			var resp harnessproto.Response
			if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
				results <- result{idx, fmt.Errorf("unmarshal: %w", err)}
				return
			}
			if !resp.OK {
				results <- result{idx, fmt.Errorf("OK=false: %+v", resp.Error)}
				return
			}
			results <- result{idx, nil}
		}(i)
	}

	for range n {
		r := <-results
		if r.err != nil {
			t.Errorf("goroutine %d: %v", r.idx, r.err)
		}
	}
}

// TestStop_Idempotent verifies that calling Stop multiple times does not panic.
func TestStop_Idempotent(t *testing.T) {
	store := observability.New()
	dir := t.TempDir()
	srv := harnesscontrol.NewServer(store, func(tea.Msg) {}, filepath.Join(dir, "s.sock"))
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := srv.Stop(); err != nil {
		t.Errorf("first Stop: %v", err)
	}
	if err := srv.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
}

// TestActionHandler_PropagatesError verifies that when sendMsg delivers an
// error through ResponseCh the server returns ErrUnavailableState.
func TestActionHandler_PropagatesError(t *testing.T) {
	store := observability.New()
	dir := t.TempDir()

	sendMsg := func(msg tea.Msg) {
		if m, ok := msg.(model.RemoteInterruptMsg); ok && m.ResponseCh != nil {
			m.ResponseCh <- fmt.Errorf("cli driver not available")
		}
	}

	srv := harnesscontrol.NewServer(store, sendMsg, filepath.Join(dir, "err.sock"))
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { srv.Stop() })

	conn, err := net.Dial("unix", filepath.Join(dir, "err.sock"))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	resp := exchange(t, conn, makeReq(harnessproto.KindInterrupt, nil))
	if resp.OK {
		t.Fatal("expected OK=false when handler returns error")
	}
	if resp.Error == nil || resp.Error.Code != harnessproto.ErrUnavailableState {
		t.Errorf("Error.Code = %+v, want %q", resp.Error, harnessproto.ErrUnavailableState)
	}
}
