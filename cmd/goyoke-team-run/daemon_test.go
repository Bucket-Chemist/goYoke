package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcquirePIDFile tests PID file acquisition in various scenarios
func TestAcquirePIDFile(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(t *testing.T, teamDir string) // Prepare test state
		wantErr       bool
		errContains   string
		validateAfter func(t *testing.T, teamDir string, pidFile *PIDFile) // Validate post-conditions
	}{
		{
			name: "new_team_directory",
			setup: func(t *testing.T, teamDir string) {
				// No setup - clean directory
			},
			wantErr: false,
			validateAfter: func(t *testing.T, teamDir string, pidFile *PIDFile) {
				// Verify PID file exists and contains current PID
				pidPath := filepath.Join(teamDir, PIDFileName)
				data, err := os.ReadFile(pidPath)
				require.NoError(t, err)
				writtenPID, err := strconv.Atoi(strings.TrimSpace(string(data)))
				require.NoError(t, err)
				assert.Equal(t, os.Getpid(), writtenPID)
			},
		},
		{
			name: "double_start",
			setup: func(t *testing.T, teamDir string) {
				// Write PID file with current process (simulates running instance)
				pidPath := filepath.Join(teamDir, PIDFileName)
				err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644)
				require.NoError(t, err)
			},
			wantErr:     true,
			errContains: "team already running",
			validateAfter: func(t *testing.T, teamDir string, pidFile *PIDFile) {
				// PID file should still exist (not modified)
				pidPath := filepath.Join(teamDir, PIDFileName)
				_, err := os.Stat(pidPath)
				require.NoError(t, err)
			},
		},
		{
			name: "stale_pid_file",
			setup: func(t *testing.T, teamDir string) {
				// Write PID file with non-existent PID
				pidPath := filepath.Join(teamDir, PIDFileName)
				err := os.WriteFile(pidPath, []byte("999999\n"), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validateAfter: func(t *testing.T, teamDir string, pidFile *PIDFile) {
				// Stale PID file should be replaced with current PID
				pidPath := filepath.Join(teamDir, PIDFileName)
				data, err := os.ReadFile(pidPath)
				require.NoError(t, err)
				writtenPID, err := strconv.Atoi(strings.TrimSpace(string(data)))
				require.NoError(t, err)
				assert.Equal(t, os.Getpid(), writtenPID)
			},
		},
		{
			name: "malformed_pid_file",
			setup: func(t *testing.T, teamDir string) {
				// Write PID file with invalid content
				pidPath := filepath.Join(teamDir, PIDFileName)
				err := os.WriteFile(pidPath, []byte("not-a-number\n"), 0644)
				require.NoError(t, err)
			},
			wantErr: false,
			validateAfter: func(t *testing.T, teamDir string, pidFile *PIDFile) {
				// Malformed file should be replaced
				pidPath := filepath.Join(teamDir, PIDFileName)
				data, err := os.ReadFile(pidPath)
				require.NoError(t, err)
				writtenPID, err := strconv.Atoi(strings.TrimSpace(string(data)))
				require.NoError(t, err)
				assert.Equal(t, os.Getpid(), writtenPID)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Create temporary team directory
			teamDir := t.TempDir()

			// Run setup
			tc.setup(t, teamDir)

			// Acquire PID file
			pidFile, err := acquirePIDFile(teamDir)

			// Check error expectation
			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, pidFile)
				defer pidFile.Release() // Cleanup
			}

			// Run post-validation
			if tc.validateAfter != nil {
				tc.validateAfter(t, teamDir, pidFile)
			}
		})
	}
}

// TestPIDFileRelease tests PID file cleanup
func TestPIDFileRelease(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T, teamDir string) *PIDFile
		wantErr     bool
		errContains string
	}{
		{
			name: "normal_release",
			setup: func(t *testing.T, teamDir string) *PIDFile {
				pidFile, err := acquirePIDFile(teamDir)
				require.NoError(t, err)
				return pidFile
			},
			wantErr: false,
		},
		{
			name: "double_release",
			setup: func(t *testing.T, teamDir string) *PIDFile {
				pidFile, err := acquirePIDFile(teamDir)
				require.NoError(t, err)
				err = pidFile.Release()
				require.NoError(t, err)
				return pidFile // Return already-released PID file
			},
			wantErr: false, // Second release should be idempotent
		},
		{
			name: "release_after_manual_delete",
			setup: func(t *testing.T, teamDir string) *PIDFile {
				pidFile, err := acquirePIDFile(teamDir)
				require.NoError(t, err)
				// Manually delete the file
				err = os.Remove(pidFile.path)
				require.NoError(t, err)
				return pidFile
			},
			wantErr: false, // Should not error on missing file
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			teamDir := t.TempDir()
			pidFile := tc.setup(t, teamDir)

			err := pidFile.Release()

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				require.NoError(t, err)

				// Verify PID file no longer exists
				pidPath := filepath.Join(teamDir, PIDFileName)
				_, err := os.Stat(pidPath)
				assert.True(t, os.IsNotExist(err), "PID file should not exist after release")
			}
		})
	}
}

// TestProcessExists tests process existence checking
func TestProcessExists(t *testing.T) {
	tests := []struct {
		name     string
		pid      int
		expected bool
	}{
		{
			name:     "running_process",
			pid:      os.Getpid(), // Current process always exists
			expected: true,
		},
		{
			name:     "dead_process",
			pid:      999999, // Non-existent PID
			expected: false,
		},
		{
			name:     "pid_zero",
			pid:      0,
			expected: false, // PID 0 is invalid
		},
		{
			name:     "negative_pid",
			pid:      -1,
			expected: false, // Negative PID is invalid
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := processExists(tc.pid)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestSentinelErrors tests that sentinel errors work with errors.Is()
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		setup  func(t *testing.T, teamDir string)
		target error
	}{
		{
			name: "team_already_running",
			setup: func(t *testing.T, teamDir string) {
				// Write PID file with current process PID (simulates running instance)
				pidPath := filepath.Join(teamDir, PIDFileName)
				err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0644)
				require.NoError(t, err)
			},
			target: ErrTeamAlreadyRunning,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			teamDir := t.TempDir()
			tc.setup(t, teamDir)
			_, err := acquirePIDFile(teamDir)
			require.Error(t, err)
			assert.True(t, errors.Is(err, tc.target), "expected errors.Is(%v, %v) to be true", err, tc.target)
		})
	}
}

// TestTeamRunnerChildManagement tests child process registration and tracking
func TestTeamRunnerChildManagement(t *testing.T) {
	runner, err := NewTeamRunner("/tmp/test-team")
	require.NoError(t, err)

	// Test registration
	runner.registerChild(1234)
	runner.registerChild(5678)

	assert.Equal(t, 2, runner.childCount())

	// Verify specific PIDs are tracked (need lock for map access)
	runner.childrenMu.Lock()
	assert.Contains(t, runner.childPIDs, 1234)
	assert.Contains(t, runner.childPIDs, 5678)
	runner.childrenMu.Unlock()

	// Test unregistration
	runner.unregisterChild(1234)

	assert.Equal(t, 1, runner.childCount())

	runner.childrenMu.Lock()
	assert.NotContains(t, runner.childPIDs, 1234)
	assert.Contains(t, runner.childPIDs, 5678)
	runner.childrenMu.Unlock()

	// Test unregister non-existent
	runner.unregisterChild(9999) // Should not panic
	assert.Equal(t, 1, runner.childCount())
}

// TestKillAllChildrenErrorCollection tests error collection when killing non-existent PIDs
func TestKillAllChildrenErrorCollection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	runner, err := NewTeamRunner(t.TempDir())
	require.NoError(t, err)

	// Register a PID that doesn't exist
	runner.registerChild(999999)

	errs := runner.killAllChildren()
	// Should have errors for the non-existent process
	assert.NotEmpty(t, errs, "Expected errors when killing non-existent PIDs")

	// Child should be cleared from tracking
	assert.Equal(t, 0, runner.childCount())
}

// TestRedirectOutput tests redirectOutput function
func TestRedirectOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fd-manipulation test in short mode")
	}

	teamDir := t.TempDir()

	// Save original stdout/stderr file descriptors (NOT just the File pointers)
	oldStdoutFd, err := syscall.Dup(int(os.Stdout.Fd()))
	require.NoError(t, err)
	oldStderrFd, err := syscall.Dup(int(os.Stderr.Fd()))
	require.NoError(t, err)

	defer func() {
		// Restore original fds using Dup2
		syscall.Dup2(oldStdoutFd, int(os.Stdout.Fd()))
		syscall.Dup2(oldStderrFd, int(os.Stderr.Fd()))
		syscall.Close(oldStdoutFd)
		syscall.Close(oldStderrFd)
	}()

	// Call redirectOutput
	logFile, err := redirectOutput(teamDir)
	require.NoError(t, err)
	require.NotNil(t, logFile)
	defer logFile.Close()

	// Verify runner.log was created
	logPath := filepath.Join(teamDir, RunnerLogFile)
	_, err = os.Stat(logPath)
	require.NoError(t, err, "runner.log should be created")

	// Write something via log.Printf and verify it appears in runner.log
	log.Printf("TEST_MESSAGE_12345")

	// Flush and read back
	logFile.Sync()
	content, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "TEST_MESSAGE_12345", "Log message should appear in runner.log")
}

// TestDaemonizeStdin tests daemonizeStdin function
func TestDaemonizeStdin(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping fd-manipulation test in short mode")
	}

	// Save original stdin file descriptor
	oldStdinFd, err := syscall.Dup(int(os.Stdin.Fd()))
	require.NoError(t, err)
	defer func() {
		// Restore original stdin
		syscall.Dup2(oldStdinFd, int(os.Stdin.Fd()))
		syscall.Close(oldStdinFd)
	}()

	// Call daemonizeStdin - this should succeed
	err = daemonizeStdin()
	require.NoError(t, err)

	// Verify stdin is now connected to /dev/null by checking stat
	stat, err := os.Stdin.Stat()
	require.NoError(t, err)
	// /dev/null is a character device
	assert.NotEqual(t, 0, stat.Mode()&os.ModeCharDevice, "stdin should be a character device (/dev/null)")
}

// TestNewTeamRunner_InvalidConfig tests NewTeamRunner with malformed JSON
func TestNewTeamRunner_InvalidConfig(t *testing.T) {
	t.Parallel()
	teamDir := t.TempDir()

	// Write malformed JSON to config.json
	configPath := filepath.Join(teamDir, ConfigFileName)
	malformedJSON := `{"team_name": "test", "waves": [invalid json}`
	err := os.WriteFile(configPath, []byte(malformedJSON), 0644)
	require.NoError(t, err)

	// Call NewTeamRunner → expect error containing "unmarshal"
	_, err = NewTeamRunner(teamDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

// TestNewTeamRunner_NoConfig tests NewTeamRunner with empty dir (no config.json)
func TestNewTeamRunner_NoConfig(t *testing.T) {
	t.Parallel()
	teamDir := t.TempDir()

	// Call NewTeamRunner with empty dir → expect success with nil config
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)
	require.NotNil(t, runner)

	// Verify config is nil
	runner.configMu.RLock()
	isNil := runner.config == nil
	runner.configMu.RUnlock()
	assert.True(t, isNil, "Config should be nil when config.json doesn't exist")
}
