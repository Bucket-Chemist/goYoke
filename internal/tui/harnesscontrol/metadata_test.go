package harnesscontrol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/harnesscontrol"
	"github.com/Bucket-Chemist/goYoke/pkg/harnessproto"
)

// setHarnessEnv points XDG_RUNTIME_DIR at a temp dir so config.GetHarness*
// path helpers resolve inside t.TempDir().
func setHarnessEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)
	t.Setenv("XDG_CACHE_HOME", "")
	return dir
}

// activeJSONPath returns the expected path of active.json given the current
// XDG_RUNTIME_DIR (set by setHarnessEnv).
func activeJSONPath(t *testing.T) string {
	t.Helper()
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		t.Fatal("XDG_RUNTIME_DIR not set; call setHarnessEnv first")
	}
	return filepath.Join(base, "goyoke", "harness", "active.json")
}

// readActiveJSON reads and JSON-decodes active.json into a raw map.
func readActiveJSON(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile(activeJSONPath(t))
	if err != nil {
		t.Fatalf("read active.json: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal active.json: %v", err)
	}
	return m
}

func TestWriteActiveMetadata_CreatesFile(t *testing.T) {
	base := setHarnessEnv(t)

	const pid = 12345
	const sockPath = "/tmp/test.sock"
	const sessionID = "sess-abc"

	before := time.Now().UTC().Truncate(time.Second)
	if err := harnesscontrol.WriteActiveMetadata(pid, sockPath, sessionID); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}

	metaPath := filepath.Join(base, "goyoke", "harness", "active.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("active.json not created: %v", err)
	}

	var ep harnessproto.ActiveHarnessEndpoint
	if err := json.Unmarshal(data, &ep); err != nil {
		t.Fatalf("unmarshal active.json: %v", err)
	}

	if ep.PID != pid {
		t.Errorf("PID: got %d, want %d", ep.PID, pid)
	}
	if ep.SocketPath != sockPath {
		t.Errorf("SocketPath: got %q, want %q", ep.SocketPath, sockPath)
	}
	if ep.SessionID != sessionID {
		t.Errorf("SessionID: got %q, want %q", ep.SessionID, sessionID)
	}
	if ep.Protocol != harnessproto.ProtocolName {
		t.Errorf("Protocol: got %q, want %q", ep.Protocol, harnessproto.ProtocolName)
	}
	if ep.ProtocolVersion != harnessproto.ProtocolVersion {
		t.Errorf("ProtocolVersion: got %q, want %q", ep.ProtocolVersion, harnessproto.ProtocolVersion)
	}
	if ep.StartedAt.Before(before) {
		t.Errorf("StartedAt %v is before test start %v", ep.StartedAt, before)
	}
}

func TestWriteActiveMetadata_EmptySessionID(t *testing.T) {
	setHarnessEnv(t)

	if err := harnesscontrol.WriteActiveMetadata(1, "/tmp/s.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata with empty session_id: %v", err)
	}

	raw := readActiveJSON(t)
	// session_id is omitempty — it should be absent or an empty string.
	if sid, ok := raw["session_id"]; ok && sid != "" {
		t.Errorf("session_id should be absent or empty; got %v", sid)
	}
}

func TestUpdateSessionID_UpdatesExistingFile(t *testing.T) {
	setHarnessEnv(t)

	if err := harnesscontrol.WriteActiveMetadata(999, "/tmp/upd.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}

	const newSID = "sess-updated"
	if err := harnesscontrol.UpdateSessionID(newSID); err != nil {
		t.Fatalf("UpdateSessionID: %v", err)
	}

	raw := readActiveJSON(t)
	if got, _ := raw["session_id"].(string); got != newSID {
		t.Errorf("session_id after update: got %q, want %q", got, newSID)
	}
	// Other fields must be preserved.
	if pid, _ := raw["pid"].(float64); int(pid) != 999 {
		t.Errorf("pid after update: got %v, want 999", pid)
	}
	if sp, _ := raw["socket_path"].(string); sp != "/tmp/upd.sock" {
		t.Errorf("socket_path after update: got %q, want /tmp/upd.sock", sp)
	}
}

func TestUpdateSessionID_ErrorWhenFileAbsent(t *testing.T) {
	setHarnessEnv(t)

	err := harnesscontrol.UpdateSessionID("irrelevant")
	if err == nil {
		t.Fatal("expected error when metadata file does not exist, got nil")
	}
}

func TestRemoveActiveMetadata_RemovesFile(t *testing.T) {
	setHarnessEnv(t)

	if err := harnesscontrol.WriteActiveMetadata(1, "/tmp/r.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}

	if err := harnesscontrol.RemoveActiveMetadata(); err != nil {
		t.Fatalf("RemoveActiveMetadata: %v", err)
	}

	path := activeJSONPath(t)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected active.json to be removed; stat err: %v", err)
	}
}

func TestRemoveActiveMetadata_IdempotentWhenAbsent(t *testing.T) {
	setHarnessEnv(t)

	// First call without prior write.
	if err := harnesscontrol.RemoveActiveMetadata(); err != nil {
		t.Fatalf("RemoveActiveMetadata when absent: %v", err)
	}

	// Write then remove twice.
	if err := harnesscontrol.WriteActiveMetadata(2, "/tmp/r2.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}
	if err := harnesscontrol.RemoveActiveMetadata(); err != nil {
		t.Fatalf("RemoveActiveMetadata (first): %v", err)
	}
	if err := harnesscontrol.RemoveActiveMetadata(); err != nil {
		t.Fatalf("RemoveActiveMetadata (second, idempotent): %v", err)
	}
}

func TestWriteActiveMetadata_FilePermissions(t *testing.T) {
	setHarnessEnv(t)

	if err := harnesscontrol.WriteActiveMetadata(1, "/tmp/perm.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}

	info, err := os.Stat(activeJSONPath(t))
	if err != nil {
		t.Fatalf("stat active.json: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("active.json permissions: got %o, want 0600", info.Mode().Perm())
	}
}

func TestWriteUpdateRemove_FullLifecycle(t *testing.T) {
	setHarnessEnv(t)

	// Write with empty session ID.
	if err := harnesscontrol.WriteActiveMetadata(777, "/tmp/lifecycle.sock", ""); err != nil {
		t.Fatalf("WriteActiveMetadata: %v", err)
	}
	if _, err := os.Stat(activeJSONPath(t)); err != nil {
		t.Fatalf("active.json should exist after write: %v", err)
	}

	// Update session ID.
	const sid = "session-lifecycle"
	if err := harnesscontrol.UpdateSessionID(sid); err != nil {
		t.Fatalf("UpdateSessionID: %v", err)
	}
	raw := readActiveJSON(t)
	if got, _ := raw["session_id"].(string); got != sid {
		t.Errorf("session_id: got %q, want %q", got, sid)
	}

	// Remove.
	if err := harnesscontrol.RemoveActiveMetadata(); err != nil {
		t.Fatalf("RemoveActiveMetadata: %v", err)
	}
	if _, err := os.Stat(activeJSONPath(t)); !os.IsNotExist(err) {
		t.Error("active.json should not exist after remove")
	}
}
