package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		jsonOutput bool
		verify     func(t *testing.T, buf *bytes.Buffer, err error)
	}{
		{
			name:       "plain text output",
			jsonOutput: false,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				assert.Equal(t, "GOgent-Fortress v0.1.0 (built: development)\n", buf.String())
			},
		},
		{
			name:       "json output",
			jsonOutput: true,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				var info versionInfo
				err = json.Unmarshal(buf.Bytes(), &info)
				require.NoError(t, err)
				assert.Equal(t, "0.1.0", info.Version)
				assert.Equal(t, "development", info.BuildTime)
			},
		},
		{
			name:       "json output is valid JSON",
			jsonOutput: true,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				assert.True(t, json.Valid(buf.Bytes()))
			},
		},
		{
			name:       "plain text contains version",
			jsonOutput: false,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				assert.Contains(t, buf.String(), "v0.1.0")
			},
		},
		{
			name:       "plain text contains build time",
			jsonOutput: false,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				assert.Contains(t, buf.String(), "development")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := run(buf, tt.jsonOutput)
			tt.verify(t, buf, err)
		})
	}
}

func TestBuildTimeOverride(t *testing.T) {
	// Save original and restore after test
	orig := BuildTime
	defer func() { BuildTime = orig }()

	// Override BuildTime
	BuildTime = "2026-01-15T10:00:00Z"

	tests := []struct {
		name       string
		jsonOutput bool
		verify     func(t *testing.T, buf *bytes.Buffer, err error)
	}{
		{
			name:       "plain with custom build time",
			jsonOutput: false,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				assert.Contains(t, buf.String(), "2026-01-15T10:00:00Z")
			},
		},
		{
			name:       "json with custom build time",
			jsonOutput: true,
			verify: func(t *testing.T, buf *bytes.Buffer, err error) {
				require.NoError(t, err)
				var info versionInfo
				err = json.Unmarshal(buf.Bytes(), &info)
				require.NoError(t, err)
				assert.Equal(t, "2026-01-15T10:00:00Z", info.BuildTime)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := run(buf, tt.jsonOutput)
			tt.verify(t, buf, err)
		})
	}
}
