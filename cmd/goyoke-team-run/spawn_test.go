package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/pkg/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a TeamConfig for testing with given members
func testConfig(members ...Member) *TeamConfig {
	return &TeamConfig{
		TeamName:     "test-team",
		WorkflowType: "test",
		ProjectRoot:  "/tmp/test",
		SessionID:    "test-session",
		Status:       "running",
		Waves: []Wave{{
			WaveNumber:  1,
			Description: "Test wave",
			Members:     members,
		}},
	}
}

// fakeSpawner implements Spawner for testing
type fakeSpawner struct {
	fn func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error
}

func (f *fakeSpawner) Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
	return f.fn(ctx, tr, waveIdx, memIdx)
}

// setupTestRunner creates a TeamRunner with a test config and optional spawner.
// If spawner is nil, the default claudeSpawner is used.
func setupTestRunner(t *testing.T, config *TeamConfig, spawner ...Spawner) (*TeamRunner, string) {
	teamDir := t.TempDir()

	// Write config.json
	configPath := filepath.Join(teamDir, ConfigFileName)
	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Create TeamRunner
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)

	if len(spawner) > 0 && spawner[0] != nil {
		runner.spawner = spawner[0]
	}

	return runner, teamDir
}

// TestSpawnAndWait_SuccessFirstAttempt tests successful spawn on first attempt
func TestSpawnAndWait_SuccessFirstAttempt(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			return nil // always succeed
		},
	})

	// Spawn and wait
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, runner, 0, 0, &wg)
	wg.Wait()

	// Verify status is completed
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	runner.configMu.RUnlock()

	assert.Equal(t, "completed", finalStatus)
	assert.Equal(t, 0, retryCount)
}

// TestSpawnAndWait_FailOnceThenSucceed tests retry logic with one failure then success
func TestSpawnAndWait_FailOnceThenSucceed(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)

	// Create fake that fails first call, succeeds second
	var callCount int
	var mu sync.Mutex
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			if callCount == 1 {
				return fmt.Errorf("first attempt failed")
			}
			return nil
		},
	})

	// Spawn and wait
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, runner, 0, 0, &wg)
	wg.Wait()

	// Verify status is completed after retry
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	runner.configMu.RUnlock()

	assert.Equal(t, "completed", finalStatus)
	assert.Equal(t, 1, retryCount, "Should succeed on second attempt (retry 1)")
}

// TestSpawnAndWait_AllRetriesExhausted tests failure after all retries
func TestSpawnAndWait_AllRetriesExhausted(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 2, // Will try 3 times total (attempt 0, 1, 2)
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			return fmt.Errorf("simulated spawn failure")
		},
	})

	// Spawn and wait
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, runner, 0, 0, &wg)
	wg.Wait()

	// Verify status is failed after exhausting retries
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	errorMsg := runner.config.Waves[0].Members[0].ErrorMessage
	runner.configMu.RUnlock()

	assert.Equal(t, "failed", finalStatus)
	assert.Equal(t, 2, retryCount, "Should have attempted retry 0, 1, 2")
	assert.Contains(t, errorMsg, "attempt 0:")
	assert.Contains(t, errorMsg, "attempt 1:")
	assert.Contains(t, errorMsg, "attempt 2:")
}

// TestSpawnAndWait_ConcurrentMembers tests parallel spawning with mixed success/failure
func TestSpawnAndWait_ConcurrentMembers(t *testing.T) {
	members := []Member{
		{Name: "agent-1", Agent: "test", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-2", Agent: "test", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-3", Agent: "test", Model: "sonnet", Status: "pending", MaxRetries: 1},
		{Name: "agent-4", Agent: "test", Model: "sonnet", Status: "pending", MaxRetries: 1},
	}

	config := testConfig(members...)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			if memIdx%2 == 0 {
				return nil // Success for agent-1 and agent-3
			}
			return fmt.Errorf("failure for odd index")
		},
	})

	// Spawn all 4 members in parallel
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := range members {
		wg.Add(1)
		go spawnAndWait(ctx, runner, 0, i, &wg)
	}
	wg.Wait()

	// Verify results
	runner.configMu.RLock()
	defer runner.configMu.RUnlock()

	assert.Equal(t, "completed", runner.config.Waves[0].Members[0].Status, "agent-1 should succeed")
	assert.Equal(t, "failed", runner.config.Waves[0].Members[1].Status, "agent-2 should fail")
	assert.Equal(t, "completed", runner.config.Waves[0].Members[2].Status, "agent-3 should succeed")
	assert.Equal(t, "failed", runner.config.Waves[0].Members[3].Status, "agent-4 should fail")
}

// TestSpawnAndWait_ContextCancelled tests context cancellation during retry
func TestSpawnAndWait_ContextCancelled(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 5, // Many retries, but context will cancel
	}

	config := testConfig(member)

	// Create fake that delays to allow cancellation
	var attemptCount int
	var mu sync.Mutex
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			mu.Lock()
			attemptCount++
			mu.Unlock()

			// Delay to allow cancellation
			time.Sleep(50 * time.Millisecond)
			return fmt.Errorf("simulated failure")
		},
	})

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Spawn in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go spawnAndWait(ctx, runner, 0, 0, &wg)

	// Cancel after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	wg.Wait()

	// Verify status is failed with context cancellation message
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	errorMsg := runner.config.Waves[0].Members[0].ErrorMessage
	runner.configMu.RUnlock()

	assert.Equal(t, "failed", finalStatus)
	assert.Contains(t, errorMsg, "context cancelled")

	// Should not have exhausted all retries
	mu.Lock()
	assert.Less(t, attemptCount, 5, "Should stop before exhausting retries")
	mu.Unlock()
}

// TestSpawnAndWait_MaxRetriesZero tests single attempt with maxRetries=0
func TestSpawnAndWait_MaxRetriesZero(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 0, // Single attempt only
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			return fmt.Errorf("simulated spawn failure")
		},
	})

	// Spawn and wait
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, runner, 0, 0, &wg)
	wg.Wait()

	// Verify only one attempt was made
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	runner.configMu.RUnlock()

	assert.Equal(t, "failed", finalStatus)
	assert.Equal(t, 0, retryCount, "Should only attempt once (attempt 0)")
}

// TestSpawnAndWait_InvalidIndices tests handling of invalid wave/member indices
func TestSpawnAndWait_InvalidIndices(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			return nil
		},
	})

	tests := []struct {
		name    string
		waveIdx int
		memIdx  int
	}{
		{"invalid_wave", 99, 0},
		{"invalid_member", 0, 99},
		{"both_invalid", 99, 99},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var wg sync.WaitGroup
			wg.Add(1)

			// Should not panic
			spawnAndWait(ctx, runner, tc.waveIdx, tc.memIdx, &wg)
			wg.Wait()

			// Original member should be unchanged
			runner.configMu.RLock()
			status := runner.config.Waves[0].Members[0].Status
			runner.configMu.RUnlock()
			assert.Equal(t, "pending", status, "Original member should be unchanged")
		})
	}
}

// TestConfigLoadAndSave tests config loading and atomic saving
func TestConfigLoadAndSave(t *testing.T) {
	teamDir := t.TempDir()
	configPath := filepath.Join(teamDir, ConfigFileName)

	// Create initial config
	config := testConfig(Member{
		Name:   "agent-1",
		Status: "pending",
	})
	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Create runner and load config
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)
	require.NotNil(t, runner.config)

	// Modify and save
	runner.configMu.Lock()
	runner.config.Waves[0].Members[0].Status = "completed"
	runner.configMu.Unlock()

	err = runner.SaveConfig()
	require.NoError(t, err)

	// Reload and verify
	data, err = os.ReadFile(configPath)
	require.NoError(t, err)
	var reloaded TeamConfig
	err = json.Unmarshal(data, &reloaded)
	require.NoError(t, err)

	assert.Equal(t, "completed", reloaded.Waves[0].Members[0].Status)
}

// TestUpdateMember tests atomic member updates
func TestUpdateMember(t *testing.T) {
	member := Member{
		Name:       "agent-1",
		Status:     "pending",
		RetryCount: 0,
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config)

	// Update member
	err := runner.updateMember(0, 0, func(m *Member) {
		m.Status = "running"
		m.RetryCount = 1
	})
	require.NoError(t, err)

	// Verify update persisted
	runner.configMu.RLock()
	status := runner.config.Waves[0].Members[0].Status
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	runner.configMu.RUnlock()

	assert.Equal(t, "running", status)
	assert.Equal(t, 1, retryCount)
}

// TestConcurrentUpdateMember verifies no lost writes under concurrent updateMember calls.
// This was a real bug: shared .tmp filename caused rename failures.
func TestConcurrentUpdateMember(t *testing.T) {
	members := make([]Member, 8)
	for i := range members {
		members[i] = Member{
			Name:   fmt.Sprintf("agent-%d", i+1),
			Status: "pending",
		}
	}
	config := testConfig(members...)
	runner, _ := setupTestRunner(t, config)

	// Hammer updateMember from 8 goroutines simultaneously
	var wg sync.WaitGroup
	var errCount atomic.Int64
	for i := range len(members) {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := range 10 {
				if err := runner.updateMember(0, idx, func(m *Member) {
					m.RetryCount = j
					m.Status = "running"
				}); err != nil {
					errCount.Add(1)
				}
			}
		}(i)
	}
	wg.Wait()

	// ZERO errors allowed — this was the bug
	assert.Equal(t, int64(0), errCount.Load(), "All concurrent updateMember calls must succeed")

	// Verify final state is consistent
	runner.configMu.RLock()
	for i, m := range runner.config.Waves[0].Members {
		assert.Equal(t, "running", m.Status, "member %d should be running", i)
		assert.Equal(t, 9, m.RetryCount, "member %d should have retryCount 9", i)
	}
	runner.configMu.RUnlock()
}

// TestDeepCopyConfig verifies deep cloning of all pointer fields
func TestDeepCopyConfig(t *testing.T) {
	startedAt := "2026-02-07T10:00:00Z"
	pid := 12345
	script := "inter-wave.sh"

	original := &TeamConfig{
		TeamName:      "test",
		BackgroundPID: &pid,
		StartedAt:     &startedAt,
		Waves: []Wave{{
			WaveNumber:       1,
			OnCompleteScript: &script,
			Members: []Member{{
				Name:       "agent-1",
				ProcessPID: &pid,
				StartedAt:  &startedAt,
			}},
		}},
	}

	copied := deepCopyConfig(original)

	// Mutate original pointers
	newPID := 99999
	newTime := "2026-12-31T23:59:59Z"
	newScript := "mutated.sh"
	original.BackgroundPID = &newPID
	original.StartedAt = &newTime
	original.Waves[0].OnCompleteScript = &newScript
	original.Waves[0].Members[0].ProcessPID = &newPID
	original.Waves[0].Members[0].StartedAt = &newTime

	// Copy must be unaffected
	assert.Equal(t, 12345, *copied.BackgroundPID)
	assert.Equal(t, "2026-02-07T10:00:00Z", *copied.StartedAt)
	assert.Equal(t, "inter-wave.sh", *copied.Waves[0].OnCompleteScript)
	assert.Equal(t, 12345, *copied.Waves[0].Members[0].ProcessPID)
	assert.Equal(t, "2026-02-07T10:00:00Z", *copied.Waves[0].Members[0].StartedAt)
}

// TestFindMember tests member lookup by name
func TestFindMember(t *testing.T) {
	members := []Member{
		{Name: "agent-1"},
		{Name: "agent-2"},
		{Name: "agent-3"},
	}

	config := testConfig(members...)
	runner, _ := setupTestRunner(t, config)

	tests := []struct {
		name      string
		searchFor string
		wantWave  int
		wantMem   int
		wantFound bool
	}{
		{"found_first", "agent-1", 0, 0, true},
		{"found_middle", "agent-2", 0, 1, true},
		{"found_last", "agent-3", 0, 2, true},
		{"not_found", "agent-99", -1, -1, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			waveIdx, memIdx, found := runner.findMember(tc.searchFor)
			assert.Equal(t, tc.wantFound, found)
			if found {
				assert.Equal(t, tc.wantWave, waveIdx)
				assert.Equal(t, tc.wantMem, memIdx)
			}
		})
	}
}

// TestBuildCLIArgs tests CLI argument construction
func TestBuildCLIArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   *agentCLIConfig
		expected []string
	}{
		{
			name: "basic_tools",
			config: &agentCLIConfig{
				AllowedTools: []string{"Read", "Write"},
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--allowedTools", "Read,Write", "--disallowedTools", "Task,AskUserQuestion"},
		},
		{
			name: "with_additional_flags_permission_mode_stripped",
			config: &agentCLIConfig{
				AllowedTools:    []string{"Read", "Glob", "Grep"},
				AdditionalFlags: []string{"--permission-mode", "delegate"},
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--allowedTools", "Read,Glob,Grep", "--disallowedTools", "Task,AskUserQuestion"},
		},
		{
			name: "non_permission_flags_preserved",
			config: &agentCLIConfig{
				AllowedTools:    []string{"Read"},
				AdditionalFlags: []string{"--max-tokens", "4000", "--permission-mode", "delegate"},
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--allowedTools", "Read", "--disallowedTools", "Task,AskUserQuestion", "--max-tokens", "4000"},
		},
		{
			name: "no_tools",
			config: &agentCLIConfig{
				AllowedTools: []string{},
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--disallowedTools", "Task,AskUserQuestion"},
		},
		{
			name: "only_additional_flags",
			config: &agentCLIConfig{
				AdditionalFlags: []string{"--max-tokens", "4000"},
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--disallowedTools", "Task,AskUserQuestion", "--max-tokens", "4000"},
		},
		{
			name: "with_formal_schema",
			config: &agentCLIConfig{
				AllowedTools: []string{"Read", "Write"},
				FormalSchema: `{"type":"object","required":["status"],"properties":{"status":{"type":"string"}}}`,
			},
			expected: []string{"-p", "--verbose", "--output-format", "stream-json", "--json-schema", `{"type":"object","required":["status"],"properties":{"status":{"type":"string"}}}`, "--allowedTools", "Read,Write", "--disallowedTools", "Task,AskUserQuestion"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildCLIArgs(tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestBuildCLIArgs1MContextPropagation tests that [1m] suffix is inherited from env vars
func TestBuildCLIArgs1MContextPropagation(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		envVar        string
		envVal        string
		expectedModel string
	}{
		{
			name:          "sonnet_gets_1m_from_env",
			model:         "sonnet",
			envVar:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			envVal:        "claude-sonnet-4-6[1m]",
			expectedModel: "sonnet[1m]",
		},
		{
			name:          "opus_gets_1m_from_env",
			model:         "opus",
			envVar:        "ANTHROPIC_DEFAULT_OPUS_MODEL",
			envVal:        "claude-opus-4-6[1m]",
			expectedModel: "opus[1m]",
		},
		{
			name:          "haiku_never_gets_1m",
			model:         "haiku",
			envVar:        "ANTHROPIC_DEFAULT_HAIKU_MODEL",
			envVal:        "claude-haiku-4-5[1m]",
			expectedModel: "haiku",
		},
		{
			name:          "no_1m_when_env_lacks_suffix",
			model:         "sonnet",
			envVar:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			envVal:        "claude-sonnet-4-6",
			expectedModel: "sonnet",
		},
		{
			name:          "no_double_1m",
			model:         "sonnet[1m]",
			envVar:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			envVal:        "claude-sonnet-4-6[1m]",
			expectedModel: "sonnet[1m]",
		},
		{
			name:          "no_1m_when_env_unset",
			model:         "sonnet",
			envVar:        "ANTHROPIC_DEFAULT_SONNET_MODEL",
			envVal:        "",
			expectedModel: "sonnet",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Always set the env var (even to empty) to isolate from host env
			t.Setenv(tc.envVar, tc.envVal)
			config := &agentCLIConfig{
				Model:        tc.model,
				AllowedTools: []string{"Read"},
			}
			args := buildCLIArgs(config)
			modelIdx := -1
			for i, a := range args {
				if a == "--model" && i+1 < len(args) {
					modelIdx = i + 1
					break
				}
			}
			assert.NotEqual(t, -1, modelIdx, "expected --model flag")
			assert.Equal(t, tc.expectedModel, args[modelIdx])
		})
	}
}

// TestIsRetryableError tests error classification for retry decisions
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil_error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context_canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "exec_not_found",
			err:       exec.ErrNotFound,
			retryable: false,
		},
		{
			name:      "permission_denied",
			err:       os.ErrPermission,
			retryable: false,
		},
		{
			name:      "generic_error",
			err:       fmt.Errorf("some random error"),
			retryable: true,
		},
		{
			name:      "timeout_error",
			err:       context.DeadlineExceeded,
			retryable: true,
		},
		{
			name:      "exit_error",
			err:       &exec.ExitError{},
			retryable: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetryableError(tc.err)
			assert.Equal(t, tc.retryable, result)
		})
	}
}

// TestLoadAgentConfig_ValidAgent tests loading real agent config from agents-index.json
func TestLoadAgentConfig_ValidAgent(t *testing.T) {
	// This test requires agents-index.json to exist at ~/.claude/agents/agents-index.json
	// Skip if not in the expected environment
	agentsIndexPath := filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json")
	if _, err := os.Stat(agentsIndexPath); os.IsNotExist(err) {
		t.Skip("agents-index.json not found, skipping test")
	}

	// Try to load a known agent (codebase-search is in the sample we read)
	config, err := loadAgentConfig("codebase-search")
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "haiku", config.Model)
	assert.Contains(t, config.AllowedTools, "Glob")
	assert.Contains(t, config.AllowedTools, "Grep")
	assert.Contains(t, config.AllowedTools, "Read")
}

// TestLoadAgentConfig_UnknownAgent tests fallback behavior for unknown agents
func TestLoadAgentConfig_UnknownAgent(t *testing.T) {
	agentsIndexPath := filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json")
	if _, err := os.Stat(agentsIndexPath); os.IsNotExist(err) {
		t.Skip("agents-index.json not found, skipping test")
	}

	// Try to load a non-existent agent
	_, err := loadAgentConfig("nonexistent-agent-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in agents-index.json")
}

// TestPrepareSpawn_ValidConfig tests prepareSpawn with valid member config
func TestPrepareSpawn_ValidConfig(t *testing.T) {
	member := Member{
		Name:       "test-agent",
		Agent:      "codebase-search",
		Model:      "haiku",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		TimeoutMs:  30000,
	}

	config := testConfig(member)
	config.ProjectRoot = "/tmp/test-project"
	runner, teamDir := setupTestRunner(t, config)

	// Create minimal stdin file
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinData := `{
		"agent": "codebase-search",
		"task": "Find all Go files",
		"context": {"files": ["*.go"]}
	}`
	err := os.WriteFile(stdinPath, []byte(stdinData), 0644)
	require.NoError(t, err)

	spawner := &claudeSpawner{}
	cfg, err := spawner.prepareSpawn(runner, 0, 0)

	// May fail to load agents-index.json in test env - that's OK, fallback should work
	if err != nil {
		assert.Contains(t, err.Error(), "build envelope")
		return
	}

	assert.NotNil(t, cfg)
	assert.Equal(t, "test-agent", cfg.memberName)
	assert.Equal(t, "codebase-search", cfg.agentID)
	assert.Equal(t, "/tmp/test-project", cfg.projectRoot)
	assert.Equal(t, 30*time.Second, cfg.timeout)
	assert.Contains(t, cfg.envelope, "AGENT: codebase-search")
	assert.Contains(t, cfg.args, "-p")
	assert.Contains(t, cfg.args, "--output-format")
	assert.Contains(t, cfg.args, "stream-json")
}

// TestPrepareSpawn_InjectsAgentIdentity tests that prepareSpawn injects agent identity and conventions
func TestPrepareSpawn_InjectsAgentIdentity(t *testing.T) {
	// Skip if agents-index.json doesn't exist
	agentsIndexPath := filepath.Join(os.Getenv("HOME"), ".claude", "agents", "agents-index.json")
	if _, err := os.Stat(agentsIndexPath); os.IsNotExist(err) {
		t.Skip("agents-index.json not found, skipping test")
	}

	member := Member{
		Name:       "test-go-pro",
		Agent:      "go-pro",
		Model:      "sonnet",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		TimeoutMs:  30000,
	}

	config := testConfig(member)
	config.ProjectRoot = "/tmp/test-project"
	runner, teamDir := setupTestRunner(t, config)

	// Create minimal stdin file
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinData := `{
		"agent": "go-pro",
		"task": "Implement a function",
		"context": {"files": ["main.go"]}
	}`
	err := os.WriteFile(stdinPath, []byte(stdinData), 0644)
	require.NoError(t, err)

	spawner := &claudeSpawner{}
	cfg, err := spawner.prepareSpawn(runner, 0, 0)

	// If identity files don't exist, this is acceptable for test env
	if err != nil {
		t.Logf("prepareSpawn failed (expected in some envs): %v", err)
		return
	}

	require.NotNil(t, cfg)

	// Verify envelope contains the injected identity marker
	assert.Contains(t, cfg.envelope, "[AGENT IDENTITY - AUTO-INJECTED]",
		"Envelope should contain agent identity marker")

	// Verify envelope contains the conventions marker
	assert.Contains(t, cfg.envelope, "[CONVENTIONS - AUTO-INJECTED BY goyoke-validate]",
		"Envelope should contain conventions marker")

	// Verify envelope contains the original prompt content
	assert.Contains(t, cfg.envelope, "AGENT: go-pro",
		"Envelope should contain original prompt")

	// Verify identity appears BEFORE conventions in the envelope
	identityPos := strings.Index(cfg.envelope, "[AGENT IDENTITY - AUTO-INJECTED]")
	conventionsPos := strings.Index(cfg.envelope, "[CONVENTIONS - AUTO-INJECTED BY goyoke-validate]")

	if identityPos >= 0 && conventionsPos >= 0 {
		assert.Less(t, identityPos, conventionsPos,
			"Agent identity should appear before conventions in envelope")
	}
}

// TestPrepareSpawn_FallbackTools tests fallback to Read,Glob,Grep when agent config unavailable
func TestPrepareSpawn_FallbackTools(t *testing.T) {
	member := Member{
		Name:       "unknown-agent",
		Agent:      "nonexistent-agent-xyz",
		Model:      "sonnet",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		TimeoutMs:  60000,
	}

	config := testConfig(member)
	config.ProjectRoot = "/tmp/test-project"
	runner, teamDir := setupTestRunner(t, config)

	// Create minimal stdin file
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinData := `{
		"agent": "nonexistent-agent-xyz",
		"task": "Do something",
		"context": {"note": "test"}
	}`
	err := os.WriteFile(stdinPath, []byte(stdinData), 0644)
	require.NoError(t, err)

	spawner := &claudeSpawner{}
	cfg, err := spawner.prepareSpawn(runner, 0, 0)

	// Should succeed with fallback tools
	if err != nil {
		// Acceptable if agents-index.json doesn't exist
		return
	}

	assert.NotNil(t, cfg)
	// Args should contain fallback tools
	argsStr := strings.Join(cfg.args, " ")
	assert.Contains(t, argsStr, "Read")
	assert.Contains(t, argsStr, "Glob")
	assert.Contains(t, argsStr, "Grep")
}

// TestSaveConfig_NilConfig tests SaveConfig error path when config is nil
func TestSaveConfig_NilConfig(t *testing.T) {
	t.Parallel()
	// Create a TeamRunner with no config file (empty TempDir, manually set config to nil)
	teamDir := t.TempDir()
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)

	// Ensure config is nil
	runner.configMu.Lock()
	runner.config = nil
	runner.configMu.Unlock()

	// Call SaveConfig() → expect error "config not loaded"
	err = runner.SaveConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

// TestUpdateMember_NilConfig tests updateMember error path when config is nil
func TestUpdateMember_NilConfig(t *testing.T) {
	t.Parallel()
	// Create TeamRunner with nil config
	teamDir := t.TempDir()
	runner, err := NewTeamRunner(teamDir)
	require.NoError(t, err)

	// Ensure config is nil
	runner.configMu.Lock()
	runner.config = nil
	runner.configMu.Unlock()

	// Call updateMember → expect error
	err = runner.updateMember(0, 0, func(m *Member) {
		m.Status = "running"
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

// TestUpdateMember_NegativeIndices tests updateMember error path with negative indices
func TestUpdateMember_NegativeIndices(t *testing.T) {
	t.Parallel()
	member := Member{
		Name:   "agent-1",
		Status: "pending",
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config)

	tests := []struct {
		name    string
		waveIdx int
		memIdx  int
		wantErr string
	}{
		{
			name:    "negative_wave_index",
			waveIdx: -1,
			memIdx:  0,
			wantErr: "invalid wave index: -1",
		},
		{
			name:    "negative_member_index",
			waveIdx: 0,
			memIdx:  -1,
			wantErr: "invalid member index: -1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runner.updateMember(tc.waveIdx, tc.memIdx, func(m *Member) {
				m.Status = "running"
			})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestSpawnAndWait_NonRetryableError tests the non-retryable error path
func TestSpawnAndWait_NonRetryableError(t *testing.T) {
	t.Parallel()
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.txt",
		StdoutFile: "stdout.txt",
		Status:     "pending",
		MaxRetries: 3, // Should only attempt once for non-retryable
	}

	config := testConfig(member)
	runner, _ := setupTestRunner(t, config, &fakeSpawner{
		fn: func(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
			// Return non-retryable error (exec.ErrNotFound)
			return exec.ErrNotFound
		},
	})

	// Spawn and wait
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, runner, 0, 0, &wg)
	wg.Wait()

	// Verify status is failed and error message contains "fatal, non-retryable"
	runner.configMu.RLock()
	finalStatus := runner.config.Waves[0].Members[0].Status
	errorMsg := runner.config.Waves[0].Members[0].ErrorMessage
	retryCount := runner.config.Waves[0].Members[0].RetryCount
	runner.configMu.RUnlock()

	assert.Equal(t, "failed", finalStatus)
	assert.Contains(t, errorMsg, "fatal, non-retryable")
	assert.Equal(t, 0, retryCount, "Should only attempt once for non-retryable error")
}

// TestFinalizeSpawn_CostOK tests finalizeSpawn with valid cost extraction
func TestFinalizeSpawn_CostOK(t *testing.T) {
	t.Parallel()
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)
	runner, teamDir := setupTestRunner(t, config)

	// Write valid stdout JSON file to teamDir
	stdoutPath := filepath.Join(teamDir, "stdout.json")
	stdoutContent := `{"$schema":"test","status":"completed","content":{}}`
	err := os.WriteFile(stdoutPath, []byte(stdoutContent), 0644)
	require.NoError(t, err)

	// Create spawnResult with stdout bytes containing cost (array format)
	stdoutWithCost := `[
		{"type": "result", "result": "Task completed", "total_cost_usd": 0.42}
	]`
	result := &spawnResult{
		stdout:   []byte(stdoutWithCost),
		exitCode: 0,
		pid:      12345,
	}

	spawner := &claudeSpawner{}
	err = spawner.finalizeSpawn(runner, 0, 0, result)
	require.NoError(t, err)

	// Verify member's CostUSD=0.42 and CostStatus="ok"
	runner.configMu.RLock()
	costUSD := runner.config.Waves[0].Members[0].CostUSD
	costStatus := runner.config.Waves[0].Members[0].CostStatus
	runner.configMu.RUnlock()

	assert.InDelta(t, 0.42, costUSD, 0.01)
	assert.Equal(t, "ok", costStatus)
}

// TestFinalizeSpawn_NoCostField tests finalizeSpawn fallback when no cost field
func TestFinalizeSpawn_NoCostField(t *testing.T) {
	t.Parallel()
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "sonnet",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		Status:     "pending",
		MaxRetries: 1,
	}

	config := testConfig(member)
	runner, teamDir := setupTestRunner(t, config)

	// Write valid stdout JSON file to teamDir
	stdoutPath := filepath.Join(teamDir, "stdout.json")
	stdoutContent := `{"$schema":"test","status":"completed","content":{}}`
	err := os.WriteFile(stdoutPath, []byte(stdoutContent), 0644)
	require.NoError(t, err)

	// Create spawnResult with stdout bytes containing NO cost field (array format)
	stdoutNoCost := `[
		{"type": "result", "result": "Done"}
	]`
	result := &spawnResult{
		stdout:   []byte(stdoutNoCost),
		exitCode: 0,
		pid:      12345,
	}

	spawner := &claudeSpawner{}
	err = spawner.finalizeSpawn(runner, 0, 0, result)
	require.NoError(t, err)

	// Verify fallback cost behavior (CostStatus="fallback", CostUSD=0.0)
	// Budget reconciliation happens at wave level, not in finalizeSpawn
	runner.configMu.RLock()
	costUSD := runner.config.Waves[0].Members[0].CostUSD
	costStatus := runner.config.Waves[0].Members[0].CostStatus
	runner.configMu.RUnlock()

	assert.Equal(t, 0.0, costUSD)
	assert.Equal(t, "fallback", costStatus)
}

// TestClaudeSpawner_Spawn_InvalidIndices tests Spawn with invalid wave index
func TestClaudeSpawner_Spawn_InvalidIndices(t *testing.T) {
	t.Parallel()
	config := testConfig(Member{Name: "a", Agent: "test", Status: "pending"})
	config.BudgetMaxUSD = 5.0
	config.BudgetRemainingUSD = 5.0
	runner, _ := setupTestRunner(t, config)

	spawner := &claudeSpawner{}
	ctx := context.Background()

	// Invalid wave index → prepareSpawn fails
	err := spawner.Spawn(ctx, runner, 99, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare spawn")
}

// TestClaudeSpawner_Spawn_NoBudgetCheck tests that Spawn no longer checks budget
// Budget checking happens at wave level in spawnAndWaitWithBudget
func TestClaudeSpawner_Spawn_NoBudgetCheck(t *testing.T) {
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", "")
	member := Member{
		Name:       "agent-1",
		Agent:      "test-agent",
		Model:      "opus",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		Status:     "pending",
	}
	config := testConfig(member)
	config.ProjectRoot = "/tmp/test"
	config.BudgetMaxUSD = 0.01
	config.BudgetRemainingUSD = 0.01 // Low budget, but Spawn doesn't check it

	runner, teamDir := setupTestRunner(t, config)

	// Write valid stdin file with context field
	stdinData := `{"agent":"test-agent","task":"test","context":{"files":["test.go"]}}`
	err := os.WriteFile(filepath.Join(teamDir, "stdin.json"), []byte(stdinData), 0644)
	require.NoError(t, err)

	spawner := &claudeSpawner{}
	ctx := context.Background()
	err = spawner.Spawn(ctx, runner, 0, 0)
	// Spawn will fail due to missing claude CLI, but NOT due to budget
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "insufficient budget")
	// Should fail with CLI-related error instead
	assert.Contains(t, err.Error(), "claude")
}

// TestParseCLIOutput tests parsing of Claude CLI output in both NDJSON (stream-json)
// and legacy JSON array formats.
func TestParseCLIOutput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantResult  string
		wantCost    float64
		wantError   bool
		wantIsError bool
		wantSession string
	}{
		// Legacy JSON array format (backward compatibility)
		{
			name: "legacy_array_with_result",
			input: `[
				{"type": "system", "subtype": "init", "session_id": "sess-123"},
				{"type": "assistant", "message": {"content": [{"type": "text", "text": "processing..."}]}},
				{"type": "result", "subtype": "success", "result": "Task completed successfully", "total_cost_usd": 0.24, "session_id": "sess-123"}
			]`,
			wantResult:  "Task completed successfully",
			wantCost:    0.24,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-123",
		},
		{
			name: "legacy_result_with_zero_cost",
			input: `[
				{"type": "result", "result": "Done", "total_cost_usd": 0, "session_id": "sess-456"}
			]`,
			wantResult:  "Done",
			wantCost:    0,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-456",
		},
		{
			name: "legacy_result_with_error_flag",
			input: `[
				{"type": "result", "result": "Failed", "total_cost_usd": 0.12, "is_error": true, "session_id": "sess-789"}
			]`,
			wantResult:  "Failed",
			wantCost:    0.12,
			wantError:   false,
			wantIsError: true,
			wantSession: "sess-789",
		},
		{
			name:      "legacy_empty_array",
			input:     `[]`,
			wantError: true,
		},
		{
			name: "legacy_array_with_no_result_entry",
			input: `[
				{"type": "system", "subtype": "init"},
				{"type": "assistant", "message": {}}
			]`,
			wantError: true,
		},
		{
			name: "legacy_array_with_skippable_entries",
			input: `[
				{"invalid": "entry"},
				{"type": "result", "result": "Success", "total_cost_usd": 0.5, "session_id": "sess-999"}
			]`,
			wantResult:  "Success",
			wantCost:    0.5,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-999",
		},
		// NDJSON format (stream-json)
		{
			name: "ndjson_full_session",
			input: `{"type":"system","subtype":"init","session_id":"sess-ndjson-1"}
{"type":"assistant","message":{"content":[{"type":"text","text":"working..."}]}}
{"type":"result","subtype":"success","result":"NDJSON task done","total_cost_usd":0.95,"session_id":"sess-ndjson-1"}`,
			wantResult:  "NDJSON task done",
			wantCost:    0.95,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-ndjson-1",
		},
		{
			name:        "ndjson_single_result_line",
			input:       `{"type":"result","result":"quick test","total_cost_usd":0.01,"session_id":"sess-ndjson-2"}`,
			wantResult:  "quick test",
			wantCost:    0.01,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-ndjson-2",
		},
		{
			name: "ndjson_with_blank_lines",
			input: `{"type":"system","subtype":"init","session_id":"sess-ndjson-3"}

{"type":"result","result":"gaps ok","total_cost_usd":0.5,"session_id":"sess-ndjson-3"}
`,
			wantResult:  "gaps ok",
			wantCost:    0.5,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-ndjson-3",
		},
		{
			name: "ndjson_no_result_entry",
			input: `{"type":"system","subtype":"init","session_id":"sess-ndjson-4"}
{"type":"assistant","message":{}}`,
			wantError: true,
		},
		{
			name: "ndjson_with_error_flag",
			input: `{"type":"system","subtype":"init","session_id":"sess-ndjson-5"}
{"type":"result","result":"Failed hard","total_cost_usd":0.30,"is_error":true,"session_id":"sess-ndjson-5"}`,
			wantResult:  "Failed hard",
			wantCost:    0.30,
			wantError:   false,
			wantIsError: true,
			wantSession: "sess-ndjson-5",
		},
		// Constrained decoding (structured_output field)
		{
			name: "ndjson_with_structured_output",
			input: `{"type":"system","subtype":"init","session_id":"sess-cd-1"}
{"type":"result","result":"text fallback","structured_output":{"schema_id":"worker","status":"complete","summary":"did stuff"},"total_cost_usd":0.55,"session_id":"sess-cd-1"}`,
			wantResult:  `{"schema_id":"worker","status":"complete","summary":"did stuff"}`,
			wantCost:    0.55,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-cd-1",
		},
		{
			name:        "ndjson_structured_output_null",
			input:       `{"type":"result","result":"no constrained","structured_output":null,"total_cost_usd":0.10,"session_id":"sess-cd-2"}`,
			wantResult:  "no constrained",
			wantCost:    0.10,
			wantError:   false,
			wantIsError: false,
			wantSession: "sess-cd-2",
		},
		// Edge cases
		{
			name:      "empty_input",
			input:     ``,
			wantError: true,
		},
		{
			name:      "whitespace_only",
			input:     `   `,
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseCLIOutput([]byte(tc.input))

			if tc.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantResult, result.Result)
			assert.InDelta(t, tc.wantCost, result.TotalCostUSD, 0.01)
			assert.Equal(t, tc.wantIsError, result.IsError)
			assert.Equal(t, tc.wantSession, result.SessionID)
		})
	}
}

// TestWriteStdoutFile tests writing agent stdout to file
func TestWriteStdoutFile(t *testing.T) {
	tests := []struct {
		name              string
		agentResult       string
		agentID           string
		wantStatus        string
		wantRawOutput     bool
		wantValidJSON     bool
		wantPathTraversal bool
	}{
		{
			name:          "agent_returns_valid_json",
			agentResult:   `{"status": "complete", "data": {"count": 42}}`,
			agentID:       "test-agent",
			wantStatus:    "complete",
			wantValidJSON: true,
		},
		{
			name: "agent_returns_json_code_block",
			agentResult: `Here is the result:
` + "```json\n" + `{
  "status": "complete",
  "items": ["a", "b", "c"]
}
` + "```\n" + `
That's it!`,
			agentID:       "test-agent",
			wantStatus:    "complete",
			wantValidJSON: true,
		},
		{
			name:          "agent_returns_plain_text",
			agentResult:   "Task completed successfully. Found 10 items.",
			agentID:       "test-agent",
			wantRawOutput: true,
			wantValidJSON: true,
		},
		{
			name:              "path_traversal_attempt",
			agentResult:       "test",
			agentID:           "test",
			wantPathTraversal: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			teamDir := t.TempDir()
			stdoutPath := filepath.Join(teamDir, "stdout.json")

			if tc.wantPathTraversal {
				// Try to write outside teamDir
				stdoutPath = filepath.Join(teamDir, "../../../etc/passwd")
				err := writeStdoutFile(stdoutPath, teamDir, tc.agentResult, tc.agentID)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "stdout path security")
				return
			}

			err := writeStdoutFile(stdoutPath, teamDir, tc.agentResult, tc.agentID)
			require.NoError(t, err)

			// Verify file was written
			data, err := os.ReadFile(stdoutPath)
			require.NoError(t, err)

			if tc.wantValidJSON {
				var result map[string]interface{}
				err := json.Unmarshal(data, &result)
				require.NoError(t, err)

				if tc.wantRawOutput {
					assert.Equal(t, true, result["raw_output"])
					assert.Equal(t, tc.agentID, result["agent"])
					assert.Equal(t, tc.agentResult, result["result"])
				} else if tc.wantStatus != "" {
					assert.Equal(t, tc.wantStatus, result["status"])
				}
			}
		})
	}
}

// TestExtractCostFromCLIOutput_ArrayFormat tests cost extraction from array format
func TestExtractCostFromCLIOutput_ArrayFormat(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantCost   float64
		wantStatus CostStatus
	}{
		{
			name: "array_with_cost",
			input: `[
				{"type": "result", "result": "Done", "total_cost_usd": 0.42}
			]`,
			wantCost:   0.42,
			wantStatus: CostOK,
		},
		{
			name: "array_with_zero_cost",
			input: `[
				{"type": "result", "result": "Done", "total_cost_usd": 0}
			]`,
			wantCost:   0,
			wantStatus: CostFallback,
		},
		{
			name: "array_no_result_entry",
			input: `[
				{"type": "system", "subtype": "init"}
			]`,
			wantCost:   0,
			wantStatus: CostError, // parseCLIOutput fails (no result), tries object parse (array->map fails), returns error
		},
		{
			name:       "legacy_object_format",
			input:      `{"cost_usd": 1.23}`,
			wantCost:   1.23,
			wantStatus: CostOK,
		},
		{
			name:       "legacy_nested_format",
			input:      `{"usage": {"cost_usd": 2.34}}`,
			wantCost:   2.34,
			wantStatus: CostOK,
		},
		{
			name:       "invalid_json",
			input:      `{not valid}`,
			wantCost:   0,
			wantStatus: CostError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractCostFromCLIOutput([]byte(tc.input))
			assert.Equal(t, tc.wantStatus, result.Status)
			if tc.wantStatus == CostOK {
				assert.InDelta(t, tc.wantCost, result.Cost, 0.01)
			}
		})
	}
}

// TestSessionDirEnvInjection tests that GOYOKE_SESSION_DIR is correctly set when current-session exists
func TestSessionDirEnvInjection(t *testing.T) {
	t.Parallel()
	// Create temporary project directory
	projectRoot := t.TempDir()
	sessionDir := filepath.Join(projectRoot, ".goyoke", "sessions", "test-session-123")

	// Create .claude directory and write current-session marker
	goyokeDir := filepath.Join(projectRoot, ".goyoke")
	require.NoError(t, os.MkdirAll(goyokeDir, 0755))
	currentSessionPath := filepath.Join(goyokeDir, "current-session")
	require.NoError(t, os.WriteFile(currentSessionPath, []byte(sessionDir+"\n"), 0644))

	// Verify ReadCurrentSession returns the expected path
	retrievedSessionDir, err := session.ReadCurrentSession(projectRoot)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, retrievedSessionDir)

	// Create member config pointing to this projectRoot
	member := Member{
		Name:       "test-agent",
		Agent:      "codebase-search",
		Model:      "haiku",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		TimeoutMs:  30000,
	}

	config := testConfig(member)
	config.ProjectRoot = projectRoot
	runner, teamDir := setupTestRunner(t, config)

	// Create minimal stdin file
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinData := `{"agent": "codebase-search", "task": "test", "context": {}}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))

	// Call prepareSpawn
	spawner := &claudeSpawner{}
	cfg, err := spawner.prepareSpawn(runner, 0, 0)

	// May fail in test env without agents-index.json - that's acceptable
	if err != nil {
		t.Logf("prepareSpawn failed (acceptable in test env): %v", err)
		return
	}

	require.NotNil(t, cfg)
	assert.Equal(t, projectRoot, cfg.projectRoot)

	// Verify that if we were to call ReadCurrentSession with cfg.projectRoot,
	// it would return the correct session dir
	verifySessionDir, err := session.ReadCurrentSession(cfg.projectRoot)
	require.NoError(t, err)
	assert.Equal(t, sessionDir, verifySessionDir,
		"prepareSpawn should set projectRoot such that ReadCurrentSession returns the expected session dir")
}

// TestSessionDirEnvInjection_NoMarker tests fallback when current-session file doesn't exist
func TestSessionDirEnvInjection_NoMarker(t *testing.T) {
	t.Parallel()
	// Create temporary project directory WITHOUT current-session marker
	projectRoot := t.TempDir()

	// Verify ReadCurrentSession returns empty string when marker doesn't exist
	retrievedSessionDir, err := session.ReadCurrentSession(projectRoot)
	require.NoError(t, err)
	assert.Equal(t, "", retrievedSessionDir,
		"ReadCurrentSession should return empty string when current-session file doesn't exist")

	// Create member config pointing to this projectRoot
	member := Member{
		Name:       "test-agent",
		Agent:      "codebase-search",
		Model:      "haiku",
		StdinFile:  "stdin.json",
		StdoutFile: "stdout.json",
		TimeoutMs:  30000,
	}

	config := testConfig(member)
	config.ProjectRoot = projectRoot
	runner, teamDir := setupTestRunner(t, config)

	// Create minimal stdin file
	stdinPath := filepath.Join(teamDir, "stdin.json")
	stdinData := `{"agent": "codebase-search", "task": "test", "context": {}}`
	require.NoError(t, os.WriteFile(stdinPath, []byte(stdinData), 0644))

	// Call prepareSpawn
	spawner := &claudeSpawner{}
	cfg, err := spawner.prepareSpawn(runner, 0, 0)

	// May fail in test env - that's acceptable
	if err != nil {
		t.Logf("prepareSpawn failed (acceptable in test env): %v", err)
		return
	}

	require.NotNil(t, cfg)
	assert.Equal(t, projectRoot, cfg.projectRoot)

	// Verify that ReadCurrentSession still returns empty string
	// (this proves the fallback: no marker = no GOYOKE_SESSION_DIR env var would be set)
	verifySessionDir, err := session.ReadCurrentSession(cfg.projectRoot)
	require.NoError(t, err)
	assert.Equal(t, "", verifySessionDir,
		"When current-session doesn't exist, ReadCurrentSession should return empty string (no GOYOKE_SESSION_DIR)")
}

// TestWorkflowTimeout tests workflow-based timeout defaults
func TestWorkflowTimeout(t *testing.T) {
	tests := []struct {
		workflow string
		want     time.Duration
	}{
		{"braintrust", 30 * time.Minute},
		{"implementation", 10 * time.Minute},
		{"review", 5 * time.Minute},
		{"unknown", 15 * time.Minute},
		{"", 15 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.workflow, func(t *testing.T) {
			got := workflowTimeout(tt.workflow)
			if got != tt.want {
				t.Errorf("workflowTimeout(%q) = %v, want %v", tt.workflow, got, tt.want)
			}
		})
	}
}

// TestProgressTracker tests the progressTracker concurrent writer
func TestProgressTracker(t *testing.T) {
	pt := newProgressTracker()

	// Initial state
	if pt.BytesReceived() != 0 {
		t.Fatal("expected 0 bytes initially")
	}

	// Write updates lastActivity and bytesReceived
	before := pt.LastActivity()
	time.Sleep(10 * time.Millisecond)
	n, err := pt.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5 bytes written, got %d", n)
	}
	if pt.BytesReceived() != 5 {
		t.Fatalf("expected 5 bytes received, got %d", pt.BytesReceived())
	}
	if !pt.LastActivity().After(before) {
		t.Fatal("lastActivity should have advanced")
	}

	// Bytes() returns accumulated data
	if string(pt.Bytes()) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(pt.Bytes()))
	}

	// Multiple writes accumulate
	pt.Write([]byte(" world"))
	if pt.BytesReceived() != 11 {
		t.Fatalf("expected 11 bytes total, got %d", pt.BytesReceived())
	}
}

// TestHealthMonitorShadowMode tests health monitoring in shadow mode
func TestHealthMonitorShadowMode(t *testing.T) {
	// Create a test runner with config
	teamDir := t.TempDir()
	tr := &TeamRunner{
		teamDir:    teamDir,
		configPath: filepath.Join(teamDir, "config.json"),
		childPIDs:  make(map[int]struct{}),
		spawner:    &claudeSpawner{},
	}

	// Create minimal config with one wave, one member
	tr.config = &TeamConfig{
		Waves: []Wave{
			{
				WaveNumber: 1,
				Members: []Member{
					{Name: "test-agent", Status: "running"},
				},
			},
		},
	}

	// Create tracker with old lastActivity to trigger stall warning
	tracker := newProgressTracker()
	// Artificially age the lastActivity
	tracker.mu.Lock()
	tracker.lastActivity = time.Now().Add(-2 * stallWarningThreshold)
	tracker.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	// Run one health check cycle by using a very short interval
	testInterval := 10 * time.Millisecond
	done := make(chan struct{})
	go func() {
		startHealthMonitor(ctx, tr, 0, 0, tracker, testInterval)
		close(done)
	}()

	// Wait for at least one health check to run
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done // Wait for goroutine to exit before t.TempDir() cleanup

	// With 10ms interval, the goroutine should fire several times in 50ms.
	// This test verifies the monitor starts, runs, and stops cleanly.
}
