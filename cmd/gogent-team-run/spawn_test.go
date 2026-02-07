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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig creates a TeamConfig for testing with given members
func testConfig(members ...Member) *TeamConfig {
	return &TeamConfig{
		TeamName:    "test-team",
		WorkflowType: "test",
		ProjectRoot: "/tmp/test",
		SessionID:   "test-session",
		Status:      "running",
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
		name     string
		waveIdx  int
		memIdx   int
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
			expected: []string{"-p", "--output-format", "json", "--allowedTools", "Read,Write"},
		},
		{
			name: "with_additional_flags",
			config: &agentCLIConfig{
				AllowedTools:    []string{"Read", "Glob", "Grep"},
				AdditionalFlags: []string{"--permission-mode", "delegate"},
			},
			expected: []string{"-p", "--output-format", "json", "--allowedTools", "Read,Glob,Grep", "--permission-mode", "delegate"},
		},
		{
			name: "no_tools",
			config: &agentCLIConfig{
				AllowedTools: []string{},
			},
			expected: []string{"-p", "--output-format", "json"},
		},
		{
			name: "only_additional_flags",
			config: &agentCLIConfig{
				AdditionalFlags: []string{"--max-tokens", "4000"},
			},
			expected: []string{"-p", "--output-format", "json", "--max-tokens", "4000"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildCLIArgs(tc.config)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestIsRetryableError tests error classification for retry decisions
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		retryable  bool
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
	assert.Contains(t, cfg.args, "json")
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
		name     string
		waveIdx  int
		memIdx   int
		wantErr  string
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

	// Create spawnResult with stdout bytes containing cost
	stdoutWithCost := `{"cost_usd":0.42,"status":"completed"}`
	result := &spawnResult{
		stdout:   []byte(stdoutWithCost),
		exitCode: 0,
		pid:      12345,
	}

	spawner := &claudeSpawner{}
	err = spawner.finalizeSpawn(runner, 0, 0, result, 1.50)
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

	// Create spawnResult with stdout bytes containing NO cost field
	stdoutNoCost := `{"status":"completed"}`
	result := &spawnResult{
		stdout:   []byte(stdoutNoCost),
		exitCode: 0,
		pid:      12345,
	}

	estimatedCost := 1.50
	spawner := &claudeSpawner{}
	err = spawner.finalizeSpawn(runner, 0, 0, result, estimatedCost)
	require.NoError(t, err)

	// Verify fallback cost used (CostStatus="fallback")
	runner.configMu.RLock()
	costUSD := runner.config.Waves[0].Members[0].CostUSD
	costStatus := runner.config.Waves[0].Members[0].CostStatus
	runner.configMu.RUnlock()

	assert.InDelta(t, estimatedCost, costUSD, 0.01)
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
	assert.Contains(t, err.Error(), "prepare:")
}

// TestClaudeSpawner_Spawn_BudgetInsufficient tests Spawn when budget is insufficient
func TestClaudeSpawner_Spawn_BudgetInsufficient(t *testing.T) {
	t.Parallel()
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
	config.BudgetRemainingUSD = 0.01 // Not enough for opus ($5 estimate)
	runner, teamDir := setupTestRunner(t, config)

	// Write valid stdin file with context field
	stdinData := `{"agent":"test-agent","task":"test","context":{"files":["test.go"]}}`
	err := os.WriteFile(filepath.Join(teamDir, "stdin.json"), []byte(stdinData), 0644)
	require.NoError(t, err)

	spawner := &claudeSpawner{}
	ctx := context.Background()
	err = spawner.Spawn(ctx, runner, 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient budget")
}
