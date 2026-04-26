// Package harnesscontrol implements a Unix domain socket server that exposes
// the public harnessproto contract for external automation and integration.
//
// The server is transport-agnostic at the protocol layer: it speaks newline-
// delimited JSON (NDJSON) over a single persistent UDS connection. It is
// entirely separate from the internal MCP bridge and shares no wire types or
// dispatch tables with internal/tui/bridge.
//
// Architecture:
//
//	harness client ──NDJSON──► UDS listener
//	                             │
//	                             ├─ get_snapshot ──► SnapshotStore.Latest()
//	                             │
//	                             └─ action ──► sendMsg(Remote*Msg) ──► Bubbletea loop
package harnesscontrol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/internal/tui/observability"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// actionTimeout is the maximum time the server waits for the Bubbletea event
// loop to acknowledge an action before returning ErrUnavailableState.
const actionTimeout = 30 * time.Second

// Server accepts harnessproto requests over a Unix domain socket and
// dispatches them into the Bubbletea event loop via a sendMsg callback.
//
// Its zero value is not usable; use NewServer.
type Server struct {
	store      *observability.SnapshotStore
	sendMsg    func(tea.Msg)
	socketPath string

	mu       sync.Mutex
	listener net.Listener
	stopped  bool
	wg       sync.WaitGroup
}

// NewServer constructs a Server. The server is not started until Start is
// called.
//
//   - store: the observability snapshot store (HL-004)
//   - sendMsg: program.Send callback — injects messages into the Bubbletea loop
//   - socketPath: UDS path (e.g. from config.GetHarnessSocketPath)
func NewServer(store *observability.SnapshotStore, sendMsg func(tea.Msg), socketPath string) *Server {
	return &Server{
		store:      store,
		sendMsg:    sendMsg,
		socketPath: socketPath,
	}
}

// Start binds the socket and launches the accept loop in a goroutine.
// Any stale socket file at socketPath is removed before binding.
// The socket is created with mode 0600 so only the owning user can connect.
func (s *Server) Start() error {
	_ = os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("harnesscontrol: listen %s: %w", s.socketPath, err)
	}

	if err := os.Chmod(s.socketPath, 0600); err != nil {
		_ = ln.Close()
		_ = os.Remove(s.socketPath)
		return fmt.Errorf("harnesscontrol: chmod socket: %w", err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	s.wg.Go(func() { s.acceptLoop(ln) })
	return nil
}

// Stop closes the listener and waits for all active connection handlers to
// drain (no timeout — callers should cancel outstanding operations before
// calling Stop). The socket file is removed on return. Stop is idempotent.
func (s *Server) Stop() error {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return nil
	}
	s.stopped = true
	ln := s.listener
	s.mu.Unlock()

	var closeErr error
	if ln != nil {
		closeErr = ln.Close()
	}
	s.wg.Wait()
	_ = os.Remove(s.socketPath)
	return closeErr
}

// -----------------------------------------------------------------------
// Internal: accept loop and connection handler
// -----------------------------------------------------------------------

func (s *Server) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// Listener closed — normal shutdown path.
			return
		}
		s.wg.Go(func() { s.handleConn(conn) })
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	enc := json.NewEncoder(conn)
	for scanner.Scan() {
		resp := s.dispatch(scanner.Bytes())
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

// -----------------------------------------------------------------------
// Request dispatch
// -----------------------------------------------------------------------

func (s *Server) dispatch(line []byte) harnessproto.Response {
	var req harnessproto.Request
	if err := json.Unmarshal(line, &req); err != nil {
		return errResp("", harnessproto.ErrBadRequest,
			fmt.Sprintf("malformed JSON: %v", err))
	}

	switch req.Kind {
	case harnessproto.KindPing:
		return okResp(harnessproto.KindPing, nil)

	case harnessproto.KindGetSnapshot:
		return s.handleGetSnapshot()

	case harnessproto.KindSubmitPrompt:
		return s.handleSubmitPrompt(req.Payload)

	case harnessproto.KindInterrupt:
		return s.handleInterrupt()

	case harnessproto.KindRespondModal:
		return s.handleRespondModal(req.Payload)

	case harnessproto.KindRespondPermission:
		return s.handleRespondPermission(req.Payload)

	case harnessproto.KindSetModel:
		return s.handleSetModel(req.Payload)

	case harnessproto.KindSetEffort:
		return s.handleSetEffort(req.Payload)

	case harnessproto.KindSetCWD:
		return s.handleSetCWD(req.Payload)

	default:
		return errResp(req.Kind, harnessproto.ErrUnsupportedOperation,
			fmt.Sprintf("unknown kind: %q", req.Kind))
	}
}

// -----------------------------------------------------------------------
// Operation handlers
// -----------------------------------------------------------------------

func (s *Server) handleGetSnapshot() harnessproto.Response {
	snap := s.store.Latest()
	if snap.Timestamp.IsZero() {
		return errResp(harnessproto.KindGetSnapshot, harnessproto.ErrUnavailableState,
			"no snapshot available: TUI has not published state yet")
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return errResp(harnessproto.KindGetSnapshot, harnessproto.ErrUnavailableState,
			fmt.Sprintf("snapshot marshal error: %v", err))
	}
	return okResp(harnessproto.KindGetSnapshot, json.RawMessage(payload))
}

func (s *Server) handleSubmitPrompt(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.SubmitPromptRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindSubmitPrompt, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	if p.Text == "" {
		return errResp(harnessproto.KindSubmitPrompt, harnessproto.ErrBadRequest,
			"text must not be empty")
	}
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteSubmitPromptMsg{Prompt: p.Text, ResponseCh: ch})
	return awaitAction(harnessproto.KindSubmitPrompt, ch)
}

func (s *Server) handleInterrupt() harnessproto.Response {
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteInterruptMsg{ResponseCh: ch})
	return awaitAction(harnessproto.KindInterrupt, ch)
}

func (s *Server) handleRespondModal(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.RespondModalRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindRespondModal, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	ch := make(chan error, 1)
	// RequestID is not part of the public harnessproto contract; pass empty to
	// let the bridge resolve the current pending modal.
	s.sendMsg(model.RemoteRespondModalMsg{RequestID: "", Value: p.Selection, ResponseCh: ch})
	return awaitAction(harnessproto.KindRespondModal, ch)
}

func (s *Server) handleRespondPermission(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.RespondPermissionRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindRespondPermission, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	decision := "deny"
	if p.Allow {
		decision = "allow"
	}
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteRespondPermissionMsg{RequestID: "", Decision: decision, ResponseCh: ch})
	return awaitAction(harnessproto.KindRespondPermission, ch)
}

func (s *Server) handleSetModel(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.SetModelRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindSetModel, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	if p.Model == "" {
		return errResp(harnessproto.KindSetModel, harnessproto.ErrBadRequest,
			"model must not be empty")
	}
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteSetModelMsg{ModelID: p.Model, ResponseCh: ch})
	return awaitAction(harnessproto.KindSetModel, ch)
}

func (s *Server) handleSetEffort(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.SetEffortRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindSetEffort, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteSetEffortMsg{Level: p.Effort, ResponseCh: ch})
	return awaitAction(harnessproto.KindSetEffort, ch)
}

func (s *Server) handleSetCWD(raw json.RawMessage) harnessproto.Response {
	var p harnessproto.SetCWDRequest
	if err := json.Unmarshal(raw, &p); err != nil {
		return errResp(harnessproto.KindSetCWD, harnessproto.ErrBadRequest,
			fmt.Sprintf("invalid payload: %v", err))
	}
	if p.CWD == "" {
		return errResp(harnessproto.KindSetCWD, harnessproto.ErrBadRequest,
			"cwd must not be empty")
	}
	ch := make(chan error, 1)
	s.sendMsg(model.RemoteSetCWDMsg{Path: p.CWD, ResponseCh: ch})
	return awaitAction(harnessproto.KindSetCWD, ch)
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

// awaitAction blocks until ch delivers a result or actionTimeout elapses.
func awaitAction(kind string, ch <-chan error) harnessproto.Response {
	select {
	case err := <-ch:
		if err != nil {
			return errResp(kind, harnessproto.ErrUnavailableState, err.Error())
		}
		return okResp(kind, nil)
	case <-time.After(actionTimeout):
		return errResp(kind, harnessproto.ErrUnavailableState,
			"action timed out: Bubbletea event loop did not respond within 30 s")
	}
}

func okResp(kind string, payload json.RawMessage) harnessproto.Response {
	return harnessproto.Response{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            kind,
		OK:              true,
		Payload:         payload,
	}
}

func errResp(kind, code, message string) harnessproto.Response {
	return harnessproto.Response{
		Protocol:        harnessproto.ProtocolName,
		ProtocolVersion: harnessproto.ProtocolVersion,
		Kind:            kind,
		OK:              false,
		Error:           &harnessproto.ErrorDetail{Code: code, Message: message},
	}
}
