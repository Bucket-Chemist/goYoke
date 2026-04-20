//go:build integration

package subcmd_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// binary is the path to the compiled goyoke binary, set by TestMain.
var binary string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "goyoke-integ-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}

	binary = filepath.Join(tmpDir, "goyoke")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", binary, "./cmd/goyoke")
	cmd.Dir = projectRoot()
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build goyoke: " + err.Error() + "\n" + string(out))
	}

	code := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

func projectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// minimalHookEvent is a minimal Claude Code hook event payload.
// Hooks must not panic when receiving any well-formed JSON event.
const minimalHookEvent = `{"hook_event_name":"test","tool_name":"Bash","session_id":"test-session"}`

func TestHookDispatch(t *testing.T) {
	hooks := []string{
		"load-context",
		"validate",
		"skill-guard",
		"direct-impl-check",
		"permission-gate",
		"sharp-edge",
		"agent-endstate",
		"orchestrator-guard",
		"archive",
		"config-guard",
		"instructions-audit",
	}

	for _, hook := range hooks {
		hook := hook
		t.Run(hook, func(t *testing.T) {
			t.Parallel()

			cmd := exec.Command(binary, "hook", hook)
			cmd.Stdin = strings.NewReader(minimalHookEvent)
			out, _ := cmd.CombinedOutput()

			if strings.Contains(string(out), "panic:") {
				t.Fatalf("hook %s panicked:\n%s", hook, out)
			}
		})
	}
}

func TestUtilityDispatch(t *testing.T) {
	cmd := exec.Command(binary, "version")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("goyoke version failed: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "version") && !strings.Contains(got, "dev") {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestMCPDispatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "mcp")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}

	initMsg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	if _, err := stdinPipe.Write([]byte(initMsg)); err != nil {
		t.Fatalf("write init: %v", err)
	}

	// Read the initialize response before closing stdin.
	buf := make([]byte, 4096)
	n, _ := stdoutPipe.Read(buf)
	stdinPipe.Close()
	cmd.Wait()

	got := string(buf[:n])
	if !strings.Contains(got, "jsonrpc") {
		t.Fatalf("MCP did not respond with JSON-RPC, got: %q", got)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	cmd := exec.Command(binary, "nonexistent-command-xyz")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit for unknown command, got exit 0")
	}
}

func TestNoArgsTUIDoesntPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary)
	cmd.Env = append(os.Environ(), "TERM=dumb")
	out, _ := cmd.CombinedOutput()

	if strings.Contains(string(out), "panic:") {
		t.Fatalf("TUI panicked:\n%s", out)
	}
}
