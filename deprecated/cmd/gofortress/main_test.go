package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/callback"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/lifecycle"
	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStartupSequence verifies the critical startup sequence:
// 1. Stale socket cleanup
// 2. Callback server creation
// 3. Signal handler setup
// 4. MCP config generation
func TestStartupSequence(t *testing.T) {
	// Clean up any existing sockets
	err := lifecycle.CleanupStaleSockets()
	require.NoError(t, err, "Initial cleanup should succeed")

	// Create callback server
	pid := os.Getpid()
	server := callback.NewServer(pid)
	require.NotNil(t, server, "Server should be created")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server
	err = server.Start(ctx)
	require.NoError(t, err, "Server should start successfully")
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Verify socket exists
	_, err = os.Stat(server.SocketPath())
	assert.NoError(t, err, "Socket file should exist")

	// Set up lifecycle manager
	processManager := lifecycle.NewProcessManager(server.SocketPath())
	require.NotNil(t, processManager, "Process manager should be created")

	// Start signal handler
	processManager.StartSignalHandler(ctx, func() {
		cancel()
		server.Shutdown(context.Background())
	})

	// Find MCP server binary (may not exist in test env)
	serverBinary, err := mcp.FindServerBinary()
	if err != nil {
		t.Skipf("Skipping MCP config test: %v", err)
		return
	}

	// Generate MCP config
	configPath, err := mcp.GenerateConfig(pid, server.SocketPath(), serverBinary)
	require.NoError(t, err, "Config generation should succeed")
	require.NotEmpty(t, configPath, "Config path should not be empty")
	defer mcp.Cleanup(configPath)

	// Verify config exists
	_, err = os.Stat(configPath)
	assert.NoError(t, err, "Config file should exist")

	// Read and verify config content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should read config file")
	assert.Contains(t, string(data), "gofortress", "Config should contain server name")
	assert.Contains(t, string(data), server.SocketPath(), "Config should contain socket path")
}

// TestCleanupOnShutdown verifies that all resources are cleaned up on shutdown
func TestCleanupOnShutdown(t *testing.T) {
	pid := os.Getpid()
	server := callback.NewServer(pid)

	ctx, cancel := context.WithCancel(context.Background())

	err := server.Start(ctx)
	require.NoError(t, err)

	socketPath := server.SocketPath()
	_, err = os.Stat(socketPath)
	require.NoError(t, err, "Socket should exist after start")

	// Generate config
	serverBinary, err := mcp.FindServerBinary()
	if err != nil {
		t.Skipf("Skipping cleanup test: %v", err)
		return
	}

	configPath, err := mcp.GenerateConfig(pid, socketPath, serverBinary)
	require.NoError(t, err)

	// Simulate shutdown
	cancel()
	server.Shutdown(ctx)
	server.Cleanup()
	mcp.Cleanup(configPath)

	// Verify socket is removed
	_, err = os.Stat(socketPath)
	assert.True(t, os.IsNotExist(err), "Socket should be removed after cleanup")

	// Verify config is removed
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err), "Config should be removed after cleanup")
}

// TestGracefulDegradation verifies the system continues if MCP fails
func TestGracefulDegradation(t *testing.T) {
	// Verify FindServerBinary handles missing binary gracefully
	// (This may succeed if binary exists, which is fine - we're testing the error path when it doesn't)
	_, err := mcp.FindServerBinary()
	if err != nil {
		assert.Contains(t, err.Error(), "not found", "Error should indicate binary not found")
	}

	// Test that config generation succeeds even with unusual paths
	// (We can't easily test failure without breaking permissions)
	pid := os.Getpid()
	socketPath := os.TempDir() + "/test-socket.sock"
	configPath, err := mcp.GenerateConfig(pid, socketPath, "/usr/bin/nonexistent")
	if err == nil {
		// Cleanup if it succeeded
		defer mcp.Cleanup(configPath)
		assert.NotEmpty(t, configPath, "Config path should be returned")
	}
}

// TestStaleSocketCleanup verifies cleanup of orphaned sockets
func TestStaleSocketCleanup(t *testing.T) {
	// Create a fake stale socket
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = os.TempDir()
	}

	// Use a very high PID that won't exist
	stalePath := runtimeDir + "/gofortress-999999999.sock"
	err := os.WriteFile(stalePath, []byte{}, 0600)
	require.NoError(t, err, "Should create fake stale socket")
	defer os.Remove(stalePath)

	// Wait a bit to ensure file is written
	time.Sleep(10 * time.Millisecond)

	// Verify file exists
	_, err = os.Stat(stalePath)
	require.NoError(t, err, "Stale socket should exist before cleanup")

	// Run cleanup
	err = lifecycle.CleanupStaleSockets()
	require.NoError(t, err, "Cleanup should succeed")

	// Wait a bit for cleanup to complete
	time.Sleep(50 * time.Millisecond)

	// Verify stale socket is removed
	_, err = os.Stat(stalePath)
	assert.True(t, os.IsNotExist(err), "Stale socket should be removed")
}

// TestContextCancellation verifies context cancellation unblocks listeners
func TestContextCancellation(t *testing.T) {
	pid := os.Getpid()
	server := callback.NewServer(pid)

	ctx, cancel := context.WithCancel(context.Background())

	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Cleanup()
	defer server.Shutdown(ctx)

	// Cancel context
	cancel()

	// Wait briefly
	time.Sleep(50 * time.Millisecond)

	// Server should still be cleanable
	server.Shutdown(context.Background())
	server.Cleanup()

	// Socket should be removed
	_, err = os.Stat(server.SocketPath())
	assert.True(t, os.IsNotExist(err), "Socket should be cleaned up")
}
