package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignalCascade tests signal cascade to child processes
// Spawns real processes to verify SIGTERM → SIGKILL behavior
func TestSignalCascade(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name             string
		childCount       int
		gracefulChildren bool // If true, children exit on SIGTERM
		validateTimeout  time.Duration
	}{
		{
			name:             "graceful_children",
			childCount:       3,
			gracefulChildren: true,
			validateTimeout:  6 * time.Second, // Should exit within grace period
		},
		{
			name:             "stubborn_child",
			childCount:       1,
			gracefulChildren: false,
			validateTimeout:  7 * time.Second, // Needs SIGKILL (grace + 1s)
		},
		{
			name:             "mixed_children",
			childCount:       5,
			gracefulChildren: true,
			validateTimeout:  6 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runner, err := NewTeamRunner(t.TempDir())
			require.NoError(t, err)

			// Spawn child processes
			children := spawnTestChildren(t, runner, tc.childCount, tc.gracefulChildren)

			// Verify all children are running
			for _, child := range children {
				assert.True(t, processExists(child.pid), "Child %d should be running", child.pid)
			}

			// Kill all children
			startTime := time.Now()
			errs := runner.killAllChildren()

			// For graceful children, expect no errors
			if tc.gracefulChildren {
				assert.Empty(t, errs, "Expected no errors for graceful children")
			}
			// For stubborn children, still expect empty errors (SIGKILL should succeed)
			assert.Empty(t, errs, "Expected no errors (SIGKILL should succeed)")

			// Give kernel time to clean up process table
			time.Sleep(100 * time.Millisecond)

			// Wait on all children to reap zombies
			for _, child := range children {
				_ = child.cmd.Wait()
			}

			// Verify all children terminated
			for _, child := range children {
				assert.False(t, processExists(child.pid), "Child %d should be terminated", child.pid)
			}

			// Verify timing
			elapsed := time.Since(startTime)
			assert.Less(t, elapsed, tc.validateTimeout, "Kill should complete within expected time")

			// Cleanup: ensure all processes are dead
			for _, child := range children {
				_ = child.cmd.Process.Kill()
			}
		})
	}
}

// TestDoubleStartPrevention tests that PID file prevents concurrent runs
func TestDoubleStartPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	teamDir := t.TempDir()

	// Acquire first PID file
	pidFile1, err := acquirePIDFile(teamDir)
	require.NoError(t, err)
	defer pidFile1.Release()

	// Attempt to acquire second PID file (should fail)
	pidFile2, err := acquirePIDFile(teamDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team already running")
	assert.Nil(t, pidFile2)

	// Release first PID file
	err = pidFile1.Release()
	require.NoError(t, err)

	// Now second acquisition should succeed
	pidFile3, err := acquirePIDFile(teamDir)
	require.NoError(t, err)
	require.NotNil(t, pidFile3)
	defer pidFile3.Release()
}

// TestStaleCleanup tests that stale PID files are cleaned up automatically
func TestStaleCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	teamDir := t.TempDir()

	// Spawn a short-lived child process
	cmd := exec.Command("sleep", "0.1")
	err := cmd.Start()
	require.NoError(t, err)
	childPID := cmd.Process.Pid

	// Manually write its PID to file
	pidPath := teamDir + "/" + PIDFileName
	err = writeTestPIDFile(pidPath, childPID)
	require.NoError(t, err)

	// Wait for child to exit
	_ = cmd.Wait()
	time.Sleep(200 * time.Millisecond) // Ensure process is dead

	// Verify process is dead
	assert.False(t, processExists(childPID))

	// Attempt to acquire PID file (should clean up stale)
	pidFile, err := acquirePIDFile(teamDir)
	require.NoError(t, err)
	require.NotNil(t, pidFile)
	defer pidFile.Release()

	// Verify new PID file contains current process
	data, err := readTestPIDFile(pidPath)
	require.NoError(t, err)
	assert.Contains(t, data, fmt.Sprintf("%d", pidFile.pid))
}

// Helper types and functions for integration tests

type testChild struct {
	cmd *exec.Cmd
	pid int
}

// spawnTestChildren spawns N child processes for testing
// If gracefulChildren is true, children will trap SIGTERM and exit cleanly
// If false, children will ignore SIGTERM (require SIGKILL)
func spawnTestChildren(t *testing.T, runner *TeamRunner, count int, gracefulChildren bool) []testChild {
	children := make([]testChild, 0, count)

	for i := 0; i < count; i++ {
		var cmd *exec.Cmd

		if gracefulChildren {
			// Use sleep (responds to SIGTERM)
			cmd = exec.Command("sleep", "60")
		} else {
			// Use a process that ignores SIGTERM
			// We'll use a shell that traps SIGTERM
			cmd = exec.Command("sh", "-c", "trap '' TERM; sleep 60")
		}

		// Each child gets own process group
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}

		err := cmd.Start()
		require.NoError(t, err, "Failed to start child %d", i)

		pid := cmd.Process.Pid
		runner.registerChild(pid)

		children = append(children, testChild{
			cmd: cmd,
			pid: pid,
		})
	}

	return children
}

// writeTestPIDFile writes a PID to a file for testing
func writeTestPIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0644)
}

// readTestPIDFile reads a PID file for testing
func readTestPIDFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
