package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeat_FileCreated(t *testing.T) {
	teamDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Start heartbeat with short interval for fast test
	startHeartbeatWithInterval(ctx, teamDir, 50*time.Millisecond)

	// Wait for file to be created (retry loop)
	var content []byte
	var err error
	for i := 0; i < 20; i++ {
		time.Sleep(10 * time.Millisecond)
		content, err = os.ReadFile(heartbeatPath)
		if err == nil && len(content) > 0 {
			break
		}
	}

	// Verify file exists and has content
	require.NoError(t, err, "heartbeat file should exist")
	assert.NotEmpty(t, content, "heartbeat file should contain timestamp")
}

func TestHeartbeat_PeriodicTouch(t *testing.T) {
	teamDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Start heartbeat with 50ms interval
	startHeartbeatWithInterval(ctx, teamDir, 50*time.Millisecond)

	// Wait for initial write
	time.Sleep(20 * time.Millisecond)

	// Collect mtimes to verify updates
	var mtimes []time.Time
	for i := 0; i < 4; i++ {
		info, err := os.Stat(heartbeatPath)
		require.NoError(t, err)
		mtimes = append(mtimes, info.ModTime())
		time.Sleep(60 * time.Millisecond) // Slightly longer than interval
	}

	// Verify mtime updated at least 3 times
	// (Some updates might be too close together to distinguish)
	uniqueCount := 0
	for i := 1; i < len(mtimes); i++ {
		if !mtimes[i].Equal(mtimes[i-1]) {
			uniqueCount++
		}
	}

	assert.GreaterOrEqual(t, uniqueCount, 2, "heartbeat should update multiple times")
}

func TestHeartbeat_ContextCancellation(t *testing.T) {
	teamDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Start heartbeat
	startHeartbeatWithInterval(ctx, teamDir, 50*time.Millisecond)

	// Wait for initial writes
	time.Sleep(100 * time.Millisecond)

	// Get mtime before cancellation
	info1, err := os.Stat(heartbeatPath)
	require.NoError(t, err)
	mtime1 := info1.ModTime()

	// Cancel context
	cancel()

	// Wait longer than interval
	time.Sleep(150 * time.Millisecond)

	// Get mtime after cancellation
	info2, err := os.Stat(heartbeatPath)
	require.NoError(t, err)
	mtime2 := info2.ModTime()

	// Mtime should not have changed (goroutine stopped)
	// Allow small tolerance for filesystem timestamp resolution
	timeDiff := mtime2.Sub(mtime1)
	assert.Less(t, timeDiff, 100*time.Millisecond, "heartbeat should stop after context cancellation")
}

func TestHeartbeat_Interval(t *testing.T) {
	// Verify the constant matches TC-012 requirement
	assert.Equal(t, 10*time.Second, HeartbeatInterval, "HeartbeatInterval must be 10 seconds per TC-012")
}

func TestWriteHeartbeat(t *testing.T) {
	teamDir := t.TempDir()
	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Write heartbeat
	writeHeartbeat(heartbeatPath)

	// Verify file exists
	content, err := os.ReadFile(heartbeatPath)
	require.NoError(t, err)

	// Verify content is a timestamp (non-empty numeric string)
	assert.NotEmpty(t, content)
	assert.Regexp(t, `^\d+\n$`, string(content), "heartbeat should contain Unix timestamp")
}

func TestWriteHeartbeat_ErrorHandling(t *testing.T) {
	// Write to invalid path (should not panic)
	// This tests that writeHeartbeat handles errors gracefully
	invalidPath := "/invalid/path/that/does/not/exist/heartbeat"

	// Should not panic
	require.NotPanics(t, func() {
		writeHeartbeat(invalidPath)
	})
}

// TestStartHeartbeat tests the public startHeartbeat wrapper
func TestStartHeartbeat(t *testing.T) {
	teamDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Start heartbeat using public wrapper
	startHeartbeat(ctx, teamDir)

	// Wait for initial heartbeat write
	time.Sleep(100 * time.Millisecond)

	// Verify heartbeat file was created
	_, err := os.Stat(heartbeatPath)
	assert.NoError(t, err, "heartbeat file should exist")

	// Cancel context to stop heartbeat
	cancel()

	// Wait for goroutine to stop
	time.Sleep(50 * time.Millisecond)
}
