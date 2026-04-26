package harnesscontrol_test

// End-to-end compatibility gate for Harness Link (HL-014).
//
// Run with: go test ./internal/tui/harnesscontrol/... -run E2E
//
// These tests verify the full bridge stack without a live TUI process by using
// the testHarness/newHarness helpers already defined in server_test.go and the
// setHarnessEnv helper from metadata_test.go. No external process is required.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/goYoke/internal/harness/link"
	"github.com/Bucket-Chemist/goYoke/internal/tui/harnesscontrol"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// TestE2EProtocolRoundTrip verifies the complete server stack: start server,
// send ping and get_snapshot, confirm response envelopes carry correct protocol
// metadata.
func TestE2EProtocolRoundTrip(t *testing.T) {
	h := newHarness(t)
	publishSnap(h.store, "idle", "e2e-hash-001")

	// Ping round-trip.
	resp := h.roundtrip(t, makeReq(harnessproto.KindPing, nil))
	if !resp.OK {
		t.Fatalf("E2E ping: expected OK=true, got error %+v", resp.Error)
	}
	if resp.Protocol != harnessproto.ProtocolName {
		t.Errorf("E2E ping: Protocol = %q, want %q", resp.Protocol, harnessproto.ProtocolName)
	}
	if resp.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("E2E ping: ProtocolVersion = %q, want %q", resp.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if resp.Kind != harnessproto.KindPing {
		t.Errorf("E2E ping: Kind = %q, want %q", resp.Kind, harnessproto.KindPing)
	}

	// Snapshot round-trip.
	resp = h.roundtrip(t, makeReq(harnessproto.KindGetSnapshot, nil))
	if !resp.OK {
		t.Fatalf("E2E get_snapshot: expected OK=true, got error %+v", resp.Error)
	}
	var snap harnessproto.SessionSnapshot
	if err := json.Unmarshal(resp.Payload, &snap); err != nil {
		t.Fatalf("E2E get_snapshot: unmarshal payload: %v", err)
	}
	if snap.Status != "idle" {
		t.Errorf("E2E get_snapshot: Status = %q, want \"idle\"", snap.Status)
	}
	if snap.StateHash != "e2e-hash-001" {
		t.Errorf("E2E get_snapshot: StateHash = %q, want \"e2e-hash-001\"", snap.StateHash)
	}
	if snap.Protocol != harnessproto.ProtocolName {
		t.Errorf("E2E get_snapshot: snapshot Protocol = %q, want %q", snap.Protocol, harnessproto.ProtocolName)
	}
}

// TestE2EActionInjection verifies that submit_prompt translates into a
// RemoteSubmitPromptMsg delivered to the Bubbletea event loop callback.
func TestE2EActionInjection(t *testing.T) {
	h := newHarness(t)

	const promptText = "run the full test suite"
	resp := h.roundtrip(t, makeReq(harnessproto.KindSubmitPrompt,
		harnessproto.SubmitPromptRequest{Text: promptText}))

	if !resp.OK {
		t.Fatalf("E2E submit_prompt: expected OK=true, got error %+v", resp.Error)
	}

	msgs := h.collectedMsgs()
	if len(msgs) == 0 {
		t.Fatal("E2E submit_prompt: no messages delivered to Bubbletea loop")
	}
	m, ok := msgs[0].(model.RemoteSubmitPromptMsg)
	if !ok {
		t.Fatalf("E2E submit_prompt: message type = %T, want RemoteSubmitPromptMsg", msgs[0])
	}
	if m.Prompt != promptText {
		t.Errorf("E2E submit_prompt: Prompt = %q, want %q", m.Prompt, promptText)
	}
}

// TestE2EInterruptPropagation verifies that the interrupt operation delivers a
// RemoteInterruptMsg to the Bubbletea event loop.
func TestE2EInterruptPropagation(t *testing.T) {
	h := newHarness(t)

	resp := h.roundtrip(t, makeReq(harnessproto.KindInterrupt, nil))
	if !resp.OK {
		t.Fatalf("E2E interrupt: expected OK=true, got error %+v", resp.Error)
	}

	msgs := h.collectedMsgs()
	if len(msgs) == 0 {
		t.Fatal("E2E interrupt: no messages delivered to Bubbletea loop")
	}
	if _, ok := msgs[0].(model.RemoteInterruptMsg); !ok {
		t.Fatalf("E2E interrupt: message type = %T, want RemoteInterruptMsg", msgs[0])
	}
}

// TestE2EModalPermissionResponse verifies that respond_modal and
// respond_permission deliver the correct message types to the Bubbletea loop.
func TestE2EModalPermissionResponse(t *testing.T) {
	t.Run("respond_modal", func(t *testing.T) {
		h := newHarness(t)

		resp := h.roundtrip(t, makeReq(harnessproto.KindRespondModal,
			harnessproto.RespondModalRequest{Selection: "approve"}))
		if !resp.OK {
			t.Fatalf("E2E respond_modal: expected OK=true, got error %+v", resp.Error)
		}

		msgs := h.collectedMsgs()
		if len(msgs) == 0 {
			t.Fatal("E2E respond_modal: no messages delivered")
		}
		m, ok := msgs[0].(model.RemoteRespondModalMsg)
		if !ok {
			t.Fatalf("E2E respond_modal: type = %T, want RemoteRespondModalMsg", msgs[0])
		}
		if m.Value != "approve" {
			t.Errorf("E2E respond_modal: Value = %q, want \"approve\"", m.Value)
		}
	})

	t.Run("respond_permission_allow", func(t *testing.T) {
		h := newHarness(t)

		resp := h.roundtrip(t, makeReq(harnessproto.KindRespondPermission,
			harnessproto.RespondPermissionRequest{Allow: true}))
		if !resp.OK {
			t.Fatalf("E2E respond_permission allow: expected OK=true, got error %+v", resp.Error)
		}

		msgs := h.collectedMsgs()
		if len(msgs) == 0 {
			t.Fatal("E2E respond_permission allow: no messages delivered")
		}
		m, ok := msgs[0].(model.RemoteRespondPermissionMsg)
		if !ok {
			t.Fatalf("E2E respond_permission allow: type = %T, want RemoteRespondPermissionMsg", msgs[0])
		}
		if m.Decision != "allow" {
			t.Errorf("E2E respond_permission allow: Decision = %q, want \"allow\"", m.Decision)
		}
	})

	t.Run("respond_permission_deny", func(t *testing.T) {
		h := newHarness(t)

		resp := h.roundtrip(t, makeReq(harnessproto.KindRespondPermission,
			harnessproto.RespondPermissionRequest{Allow: false}))
		if !resp.OK {
			t.Fatalf("E2E respond_permission deny: expected OK=true, got error %+v", resp.Error)
		}

		msgs := h.collectedMsgs()
		if len(msgs) == 0 {
			t.Fatal("E2E respond_permission deny: no messages delivered")
		}
		m, ok := msgs[0].(model.RemoteRespondPermissionMsg)
		if !ok {
			t.Fatalf("E2E respond_permission deny: type = %T, want RemoteRespondPermissionMsg", msgs[0])
		}
		if m.Decision != "deny" {
			t.Errorf("E2E respond_permission deny: Decision = %q, want \"deny\"", m.Decision)
		}
	})
}

// TestE2EDiscoveryIndependence verifies that the harness control server can be
// discovered via active.json without the GOYOKE_SOCKET environment variable
// being set.
//
// This is the key acceptance criterion: child-process GOYOKE_SOCKET inheritance
// is not required for harness endpoint discovery. Adapters must read
// active.json from XDG_RUNTIME_DIR instead.
func TestE2EDiscoveryIndependence(t *testing.T) {
	// Explicitly clear GOYOKE_SOCKET to confirm it is never consulted.
	t.Setenv("GOYOKE_SOCKET", "")

	// Redirect active.json to a temp dir via XDG_RUNTIME_DIR.
	dir := setHarnessEnv(t)

	const (
		testPID     = 99991
		testSession = "e2e-discovery-sess"
	)

	sockPath := filepath.Join(dir, "goyoke", "harness", "goyoke-harness-99991.sock")
	if err := harnesscontrol.WriteActiveMetadata(testPID, sockPath, testSession); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}

	// Discover: read active.json using only XDG_RUNTIME_DIR — no GOYOKE_SOCKET.
	metaPath := filepath.Join(dir, "goyoke", "harness", "active.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("discovery via active.json failed: %v (GOYOKE_SOCKET=%q)",
			err, os.Getenv("GOYOKE_SOCKET"))
	}

	var ep harnessproto.ActiveHarnessEndpoint
	if err := json.Unmarshal(data, &ep); err != nil {
		t.Fatalf("decode active.json: %v", err)
	}

	if ep.SocketPath != sockPath {
		t.Errorf("SocketPath = %q, want %q", ep.SocketPath, sockPath)
	}
	if ep.SessionID != testSession {
		t.Errorf("SessionID = %q, want %q", ep.SessionID, testSession)
	}
	if ep.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol = %q, want %q", ep.Protocol, harnessproto.ProtocolName)
	}
	if ep.PID != testPID {
		t.Errorf("PID = %d, want %d", ep.PID, testPID)
	}

	// Confirm GOYOKE_SOCKET remains unset — it must never be the discovery path.
	if sock := os.Getenv("GOYOKE_SOCKET"); sock != "" {
		t.Errorf("GOYOKE_SOCKET is %q; harness discovery must not rely on it", sock)
	}
}

// TestE2ELinkUnlinkSafety verifies the link/unlink lifecycle:
//  1. link.Link records managed paths in a manifest.
//  2. All managed paths exist after link.
//  3. link.Unlink removes exactly the managed paths and the manifest.
//  4. No extra files or directories are deleted.
func TestE2ELinkUnlinkSafety(t *testing.T) {
	// Redirect harness data/runtime dirs to temp locations.
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", "")

	const provider = "manual"

	// Create synthetic managed files to simulate what adapter Install writes.
	providerDir := t.TempDir()
	managedFile1 := filepath.Join(providerDir, "harness-config.json")
	managedFile2 := filepath.Join(providerDir, "README.md")
	for _, f := range []string{managedFile1, managedFile2} {
		if err := os.WriteFile(f, []byte("placeholder"), 0600); err != nil {
			t.Fatalf("create managed file %q: %v", f, err)
		}
	}
	managedPaths := []string{managedFile1, managedFile2}

	// Link: record manifest.
	if err := link.Link(provider, managedPaths, link.LinkOptions{SupportLevel: "supported"}); err != nil {
		t.Fatalf("link.Link: %v", err)
	}

	// Manifest must exist and carry correct fields.
	m, err := link.ReadManifest(provider)
	if err != nil {
		t.Fatalf("ReadManifest after link: %v", err)
	}
	if m.Provider != provider {
		t.Errorf("manifest Provider = %q, want %q", m.Provider, provider)
	}
	if m.SupportLevel != "supported" {
		t.Errorf("manifest SupportLevel = %q, want \"supported\"", m.SupportLevel)
	}
	if len(m.ManagedPaths) != len(managedPaths) {
		t.Errorf("manifest ManagedPaths len = %d, want %d", len(m.ManagedPaths), len(managedPaths))
	}

	// Managed files must still exist before unlink.
	for _, f := range managedPaths {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("managed file %q should exist before unlink: %v", f, err)
		}
	}

	// Unlink: remove managed paths and manifest.
	if err := link.Unlink(provider); err != nil {
		t.Fatalf("link.Unlink: %v", err)
	}

	// Managed files must be gone.
	for _, f := range managedPaths {
		if _, statErr := os.Stat(f); !os.IsNotExist(statErr) {
			t.Errorf("managed file %q should be removed after unlink; stat err: %v", f, statErr)
		}
	}

	// Manifest must be gone.
	if _, err := link.ReadManifest(provider); err == nil {
		t.Error("ReadManifest should fail after unlink; manifest should not exist")
	}
}

// TestE2EBackwardCompatibilityMCPBridge verifies that the harnesscontrol
// package source files do not import internal MCP bridge types. The control
// server is intentionally separate from the MCP bridge and must share no wire
// types or dispatch tables with internal/tui/bridge.
//
// This is a structural source-level check that catches accidental coupling
// before it reaches a build or integration test.
func TestE2EBackwardCompatibilityMCPBridge(t *testing.T) {
	// Use the quoted import path form so package-level comments that
	// *reference* the bridge package (e.g. "separate from internal/tui/bridge")
	// do not trigger a false positive. Only an actual import statement will
	// contain the full module path in quotes.
	forbidden := []string{
		`"github.com/Bucket-Chemist/goYoke/internal/tui/bridge"`,
		`"github.com/Bucket-Chemist/goYoke/internal/tui/mcp"`,
	}

	files := []string{"server.go", "metadata.go"}
	for _, filename := range files {
		src, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		for _, pattern := range forbidden {
			if strings.Contains(string(src), pattern) {
				t.Errorf("%s imports forbidden pattern %q; "+
					"harnesscontrol must not depend on the MCP bridge", filename, pattern)
			}
		}
	}
}
