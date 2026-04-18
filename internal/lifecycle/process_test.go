package lifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupStaleSockets(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create a stale socket (non-existent PID)
	stalePath := filepath.Join(tmpDir, "goyoke-99999999.sock")
	if err := os.WriteFile(stalePath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create stale socket: %v", err)
	}

	// Create a valid socket (current process)
	validPath := filepath.Join(tmpDir, fmt.Sprintf("goyoke-%d.sock", os.Getpid()))
	if err := os.WriteFile(validPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create valid socket: %v", err)
	}

	// Run cleanup
	if err := CleanupStaleSockets(); err != nil {
		t.Fatalf("CleanupStaleSockets failed: %v", err)
	}

	// Stale should be removed
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("Stale socket was not removed")
	}

	// Valid should remain
	if _, err := os.Stat(validPath); err != nil {
		t.Error("Valid socket was incorrectly removed")
	}
}

func TestProcessManager_SignalPropagation(t *testing.T) {
	pm := NewProcessManager("/tmp/test.sock")
	ctx, cancel := context.WithCancel(context.Background())

	pm.StartSignalHandler(ctx, func() {
		// Shutdown callback executed
	})

	// Cancel context to trigger shutdown path
	cancel()

	select {
	case <-pm.done:
		// Good - shutdown completed
	case <-time.After(time.Second):
		t.Error("Signal handler did not complete")
	}
}

func TestExtractPIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected int
	}{
		{"/run/user/1000/goyoke-12345.sock", 12345},
		{"/tmp/goyoke-1.sock", 1},
		{"/tmp/goyoke-notapid.sock", 0},
		{"/tmp/other-file.sock", 0},
	}

	for _, tc := range tests {
		got := extractPIDFromPath(tc.path)
		if got != tc.expected {
			t.Errorf("extractPIDFromPath(%q) = %d, want %d", tc.path, got, tc.expected)
		}
	}
}

func TestProcessManager_NewProcessManager(t *testing.T) {
	socketPath := "/tmp/test.sock"
	pm := NewProcessManager(socketPath)

	if pm == nil {
		t.Fatal("NewProcessManager returned nil")
	}

	if pm.socketPath != socketPath {
		t.Errorf("socketPath = %q, want %q", pm.socketPath, socketPath)
	}

	if pm.sigChan == nil {
		t.Error("sigChan was not initialized")
	}

	if pm.done == nil {
		t.Error("done channel was not initialized")
	}
}

func TestProcessManager_SetChildProcess(t *testing.T) {
	pm := NewProcessManager("/tmp/test.sock")

	// Get current process for testing
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}

	pm.SetChildProcess(process)

	if pm.childProcess == nil {
		t.Error("childProcess was not set")
	}

	if pm.childProcess.Pid != os.Getpid() {
		t.Errorf("childProcess PID = %d, want %d", pm.childProcess.Pid, os.Getpid())
	}
}

func TestProcessExists(t *testing.T) {
	tests := []struct {
		name     string
		pid      int
		expected bool
	}{
		{"current process", os.Getpid(), true},
		{"non-existent process", 99999999, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := processExists(tc.pid)
			if got != tc.expected {
				t.Errorf("processExists(%d) = %v, want %v", tc.pid, got, tc.expected)
			}
		})
	}
}

func TestCleanupStaleSockets_EmptyDir(t *testing.T) {
	// Create temp dir with no socket files
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Run cleanup - should not error
	if err := CleanupStaleSockets(); err != nil {
		t.Errorf("CleanupStaleSockets failed on empty directory: %v", err)
	}
}

func TestCleanupStaleSockets_NonMatchingFiles(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create files that don't match the pattern
	otherFile := filepath.Join(tmpDir, "other-file.sock")
	if err := os.WriteFile(otherFile, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Run cleanup
	if err := CleanupStaleSockets(); err != nil {
		t.Fatalf("CleanupStaleSockets failed: %v", err)
	}

	// Non-matching file should remain
	if _, err := os.Stat(otherFile); err != nil {
		t.Error("Non-matching file was incorrectly removed")
	}
}

func TestProcessManager_Wait(t *testing.T) {
	pm := NewProcessManager("/tmp/test.sock")

	// Close done channel to simulate completion
	close(pm.done)

	// Wait should return immediately
	done := make(chan struct{})
	go func() {
		pm.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good - Wait returned
	case <-time.After(100 * time.Millisecond):
		t.Error("Wait did not return after done channel closed")
	}
}

func TestProcessManager_SignalPropagationWithProcess(t *testing.T) {
	// Create a temp socket file for testing
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	pm := NewProcessManager(socketPath)
	ctx := context.Background()

	// Set the current process as child (safe for testing)
	process, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find current process: %v", err)
	}
	pm.SetChildProcess(process)

	shutdownExecuted := false
	pm.StartSignalHandler(ctx, func() {
		shutdownExecuted = true
	})

	// Create socket file to test cleanup
	if err := os.WriteFile(socketPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create socket file: %v", err)
	}

	// Manually trigger signal by closing sigChan path via cancel
	// We test context cancellation path which is safer than sending actual signals
	pm.sigChan <- os.Interrupt

	// Wait for shutdown to complete
	select {
	case <-pm.done:
		if !shutdownExecuted {
			t.Error("Shutdown callback was not executed")
		}
		// Socket should be removed
		if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
			t.Error("Socket file was not removed during shutdown")
		}
	case <-time.After(time.Second):
		t.Error("Signal handler did not complete shutdown")
	}
}

func TestCleanupStaleSockets_WithoutXDGRuntimeDir(t *testing.T) {
	// Save original XDG_RUNTIME_DIR
	originalXDG := os.Getenv("XDG_RUNTIME_DIR")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_RUNTIME_DIR", originalXDG)
		}
	}()

	// Unset XDG_RUNTIME_DIR to test fallback to TempDir
	os.Unsetenv("XDG_RUNTIME_DIR")

	// This should not error and should use os.TempDir()
	if err := CleanupStaleSockets(); err != nil {
		t.Errorf("CleanupStaleSockets failed without XDG_RUNTIME_DIR: %v", err)
	}
}

func TestCleanupStaleSockets_MultipleStale(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create multiple stale sockets
	stale1 := filepath.Join(tmpDir, "goyoke-99999998.sock")
	stale2 := filepath.Join(tmpDir, "goyoke-99999997.sock")
	stale3 := filepath.Join(tmpDir, "goyoke-99999996.sock")

	for _, path := range []string{stale1, stale2, stale3} {
		if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
			t.Fatalf("Failed to create stale socket: %v", err)
		}
	}

	// Run cleanup
	if err := CleanupStaleSockets(); err != nil {
		t.Fatalf("CleanupStaleSockets failed: %v", err)
	}

	// All stale sockets should be removed
	for _, path := range []string{stale1, stale2, stale3} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Stale socket %s was not removed", path)
		}
	}
}

func TestProcessExists_FindProcessError(t *testing.T) {
	// Test with PID 0, which should fail on FindProcess on some systems
	// or at least not exist as a normal user process
	exists := processExists(0)
	// We can't guarantee the result, but the function should not panic
	_ = exists
}

func TestCleanupStaleSockets_RemoveError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmpDir)

	// Create a stale socket in a directory we'll make read-only
	subDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	stalePath := filepath.Join(subDir, "goyoke-99999995.sock")
	if err := os.WriteFile(stalePath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create stale socket: %v", err)
	}

	// Make directory read-only to cause remove to fail
	if err := os.Chmod(subDir, 0500); err != nil {
		t.Fatalf("Failed to change directory permissions: %v", err)
	}
	defer os.Chmod(subDir, 0755) // Restore permissions for cleanup

	// Need to update XDG_RUNTIME_DIR to point to subdir for this test
	t.Setenv("XDG_RUNTIME_DIR", subDir)

	// Cleanup should not return error even if removal fails
	// It logs to stderr but continues
	if err := CleanupStaleSockets(); err != nil {
		t.Errorf("CleanupStaleSockets should not error on remove failure: %v", err)
	}
}
