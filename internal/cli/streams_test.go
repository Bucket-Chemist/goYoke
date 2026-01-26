package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNDJSONReader_ValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []map[string]interface{}
	}{
		{
			name:  "single line",
			input: `{"type":"test","value":1}` + "\n",
			expected: []map[string]interface{}{
				{"type": "test", "value": float64(1)},
			},
		},
		{
			name: "multiple lines",
			input: `{"type":"first"}` + "\n" +
				`{"type":"second"}` + "\n" +
				`{"type":"third"}` + "\n",
			expected: []map[string]interface{}{
				{"type": "first"},
				{"type": "second"},
				{"type": "third"},
			},
		},
		{
			name:  "empty object",
			input: `{}` + "\n",
			expected: []map[string]interface{}{
				{},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reader := NewNDJSONReader(strings.NewReader(tc.input))

			for i, expected := range tc.expected {
				data, err := reader.Read()
				require.NoError(t, err, "Read failed on line %d", i)

				var result map[string]interface{}
				err = unmarshalJSON(data, &result)
				require.NoError(t, err, "Unmarshal failed on line %d", i)

				assert.Equal(t, expected, result, "Line %d mismatch", i)
			}

			// Verify EOF after all lines
			_, err := reader.Read()
			assert.Equal(t, io.EOF, err, "Expected EOF after last line")
		})
	}
}

func TestNDJSONReader_MalformedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "invalid json",
			input: `{invalid json}` + "\n",
		},
		{
			name:  "incomplete object",
			input: `{"key":"value"` + "\n",
		},
		{
			name:  "plain text",
			input: `not json at all` + "\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reader := NewNDJSONReader(strings.NewReader(tc.input))

			data, err := reader.Read()
			require.NoError(t, err, "Read should not fail on malformed JSON")

			// JSON unmarshaling should fail
			var result map[string]interface{}
			err = unmarshalJSON(data, &result)
			assert.Error(t, err, "Should fail to unmarshal malformed JSON")
		})
	}
}

func TestNDJSONReader_LongLines(t *testing.T) {
	// Create a JSON object with a very long string value (>64KB)
	longString := strings.Repeat("x", 100000)
	input := `{"type":"test","data":"` + longString + `"}` + "\n"

	reader := NewNDJSONReader(strings.NewReader(input))

	data, err := reader.Read()
	require.NoError(t, err, "Should handle long lines with custom buffer")

	var result map[string]interface{}
	err = unmarshalJSON(data, &result)
	require.NoError(t, err, "Should parse long line JSON")

	assert.Equal(t, "test", result["type"])
	assert.Equal(t, longString, result["data"])
}

func TestNDJSONReader_EOF(t *testing.T) {
	reader := NewNDJSONReader(strings.NewReader(""))

	_, err := reader.Read()
	assert.Equal(t, io.EOF, err, "Empty reader should return EOF")
}

func TestNDJSONWriter_ValidJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "simple object",
			input:    map[string]string{"type": "test"},
			expected: `{"type":"test"}` + "\n",
		},
		{
			name:     "nested object",
			input:    map[string]interface{}{"outer": map[string]int{"inner": 42}},
			expected: `{"outer":{"inner":42}}` + "\n",
		},
		{
			name:     "array",
			input:    []string{"a", "b", "c"},
			expected: `["a","b","c"]` + "\n",
		},
		{
			name:     "empty object",
			input:    map[string]interface{}{},
			expected: `{}` + "\n",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewNDJSONWriter(&buf)

			err := writer.Write(tc.input)
			require.NoError(t, err, "Write failed")

			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestNDJSONWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf)

	// Write multiple objects
	err := writer.Write(map[string]string{"first": "1"})
	require.NoError(t, err)

	err = writer.Write(map[string]string{"second": "2"})
	require.NoError(t, err)

	err = writer.Write(map[string]string{"third": "3"})
	require.NoError(t, err)

	expected := `{"first":"1"}` + "\n" +
		`{"second":"2"}` + "\n" +
		`{"third":"3"}` + "\n"

	assert.Equal(t, expected, buf.String())
}

func TestNDJSONWriter_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf)

	// Try to write something that can't be marshaled
	err := writer.Write(make(chan int))
	assert.Error(t, err, "Should fail on un-marshalable type")
}

func TestNDJSONWriter_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf)

	// Launch multiple goroutines writing concurrently
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				writer.Write(map[string]int{"id": id, "seq": j})
			}
			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all writes completed (100 lines total)
	lines := strings.Split(buf.String(), "\n")
	// Last line is empty after final newline
	assert.Equal(t, 101, len(lines), "Should have 100 JSON lines + empty final line")

	// Verify each line is valid JSON
	for i, line := range lines[:100] {
		var result map[string]int
		err := unmarshalJSON([]byte(line), &result)
		assert.NoError(t, err, "Line %d should be valid JSON", i)
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}
	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Should generate session ID
	assert.NotEmpty(t, proc.SessionID())

	// Should use "claude" as default path
	assert.Contains(t, proc.cmd.Path, "claude")
}

func TestConfig_CustomValues(t *testing.T) {
	cfg := Config{
		ClaudePath:     "/custom/path/claude",
		SessionID:      "custom-session-123",
		SettingsPath:   "/custom/settings.json",
		WorkingDir:     "/custom/workdir",
		Verbose:        true,
		IncludePartial: true,
	}

	proc, err := NewClaudeProcess(cfg)
	require.NoError(t, err)

	// Verify session ID
	assert.Equal(t, "custom-session-123", proc.SessionID())

	// Verify command construction
	args := proc.cmd.Args
	assert.Contains(t, args, "--verbose")
	assert.Contains(t, args, "--include-partial-messages")
	assert.Contains(t, args, "--settings")
	assert.Contains(t, args, "/custom/settings.json")
	assert.Contains(t, args, "--session-id")
	assert.Contains(t, args, "custom-session-123")

	// Verify working directory
	assert.Equal(t, "/custom/workdir", proc.cmd.Dir)
}

// Tests for Sync() method (GOgent-119)

func TestNDJSONWriter_Sync_FlushableWriter(t *testing.T) {
	// Create a bufio.Writer which implements Flush()
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)
	writer := NewNDJSONWriter(bufWriter)

	// Write some data
	err := writer.Write(map[string]string{"test": "data"})
	require.NoError(t, err)

	// Before Sync, buffer might not be flushed
	// (depends on buffer size, but often not flushed immediately)

	// Sync should flush the buffer
	err = writer.Sync()
	require.NoError(t, err)

	// Now the data should be in the underlying buffer
	assert.Contains(t, buf.String(), `{"test":"data"}`)
}

func TestNDJSONWriter_Sync_NonFlushableWriter(t *testing.T) {
	// bytes.Buffer doesn't implement Flush() or Sync()
	var buf bytes.Buffer
	writer := NewNDJSONWriter(&buf)

	// Write some data
	err := writer.Write(map[string]string{"test": "data"})
	require.NoError(t, err)

	// Sync should return nil (no-op) for non-flushable writer
	err = writer.Sync()
	assert.NoError(t, err)

	// Data should still be written (just not flushed)
	assert.Contains(t, buf.String(), `{"test":"data"}`)
}

func TestNDJSONWriter_Sync_ThreadSafe(t *testing.T) {
	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)
	writer := NewNDJSONWriter(bufWriter)

	// Launch multiple goroutines calling Sync concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				err := writer.Sync()
				assert.NoError(t, err)
			}
		}(i)
	}

	// Also write concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				writer.Write(map[string]int{"id": id, "seq": j})
			}
		}(i)
	}

	wg.Wait()
	// Test should not panic - verifies mutex protection
}

// mockFile implements io.Writer with Sync() method
type mockFile struct {
	buf       bytes.Buffer
	syncCount int
	mu        sync.Mutex
}

func (m *mockFile) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *mockFile) Sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncCount++
	return nil
}

func TestNDJSONWriter_Sync_FileLikeWriter(t *testing.T) {
	// Test with a writer that has Sync() method (like os.File)
	mock := &mockFile{}
	writer := NewNDJSONWriter(mock)

	// Write and sync
	err := writer.Write(map[string]string{"type": "test"})
	require.NoError(t, err)

	err = writer.Sync()
	require.NoError(t, err)

	// Verify Sync was called on the underlying writer
	mock.mu.Lock()
	syncCount := mock.syncCount
	mock.mu.Unlock()
	assert.Equal(t, 1, syncCount, "Sync should have been called once")

	// Verify data was written
	assert.Contains(t, mock.buf.String(), `{"type":"test"}`)
}

// Helper function to unmarshal JSON
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
