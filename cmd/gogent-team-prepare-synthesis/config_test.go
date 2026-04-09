package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigJSON(t *testing.T, dir string, cfg *TeamConfig) {
	t.Helper()
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
	require.NoError(t, err)
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	cfg := &TeamConfig{
		WorkflowType: "braintrust",
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Agent: "einstein", Status: "completed"},
			}},
		},
	}
	writeConfigJSON(t, dir, cfg)

	result, err := loadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "braintrust", result.WorkflowType)
	require.Len(t, result.Waves, 1)
	require.Len(t, result.Waves[0].Members, 1)
	assert.Equal(t, "einstein", result.Waves[0].Members[0].Agent)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := loadConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "config.json")
}

func TestLoadConfig_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{invalid"), 0644)
	require.NoError(t, err)

	_, err = loadConfig(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config.json")
}

func TestFindCompletedWaveIdx_Wave0CompletedWave1Pending(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
				{Status: "failed"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
		},
	}
	assert.Equal(t, 0, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_BothWavesCompleted(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
			}},
			{Members: []MemberInfo{
				{Status: "completed"},
			}},
		},
	}
	assert.Equal(t, -1, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_Wave0StillRunning(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
				{Status: "running"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
		},
	}
	assert.Equal(t, -1, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_MixedTerminalWave0PendingWave1(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
				{Status: "failed"},
				{Status: "completed"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
		},
	}
	assert.Equal(t, 0, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_SingleWave(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
			}},
		},
	}
	assert.Equal(t, -1, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_AllFailedWave0PendingWave1(t *testing.T) {
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "failed"},
				{Status: "failed"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
		},
	}
	assert.Equal(t, 0, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_EmptyWaves(t *testing.T) {
	cfg := &TeamConfig{Waves: []WaveInfo{}}
	assert.Equal(t, -1, findCompletedWaveIdx(cfg))
}

func TestFindCompletedWaveIdx_Wave1Pending(t *testing.T) {
	// Three waves: wave0 completed, wave1 pending, wave2 pending
	// Should return 0 (wave0 is completed, wave1 has pending members)
	cfg := &TeamConfig{
		Waves: []WaveInfo{
			{Members: []MemberInfo{
				{Status: "completed"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
			{Members: []MemberInfo{
				{Status: "pending"},
			}},
		},
	}
	assert.Equal(t, 0, findCompletedWaveIdx(cfg))
}
