package routing

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// slowReader simulates a reader that takes longer than timeout
type slowReader struct {
	delay time.Duration
	data  string
}

func (sr *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(sr.delay)
	return copy(p, sr.data), io.EOF
}

// errorReader simulates a reader that returns an error
type errorReader struct {
	err error
}

func (er *errorReader) Read(p []byte) (n int, err error) {
	return 0, er.err
}

// immediateEOFReader simulates a reader that returns EOF immediately
type immediateEOFReader struct{}

func (ir *immediateEOFReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func TestReadStdin_Success(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		timeout  time.Duration
		expected string
	}{
		{
			name:     "small input",
			input:    `{"event":"test"}`,
			timeout:  1 * time.Second,
			expected: `{"event":"test"}`,
		},
		{
			name:     "multiline input",
			input:    "line1\nline2\nline3",
			timeout:  1 * time.Second,
			expected: "line1\nline2\nline3",
		},
		{
			name:     "input with special characters",
			input:    "data with\ttabs\nand\nnewlines",
			timeout:  1 * time.Second,
			expected: "data with\ttabs\nand\nnewlines",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := strings.NewReader(tc.input)
			result, err := ReadStdin(reader, tc.timeout)

			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			if string(result) != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, string(result))
			}
		})
	}
}

func TestReadStdin_Timeout(t *testing.T) {
	// Reader that takes 2 seconds, timeout of 100ms
	reader := &slowReader{
		delay: 2 * time.Second,
		data:  "too slow",
	}

	start := time.Now()
	result, err := ReadStdin(reader, 100*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result, got: %v", result)
	}

	if !strings.Contains(err.Error(), "[stdin-reader]") {
		t.Errorf("error should have [stdin-reader] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("error should mention timeout, got: %v", err)
	}

	// Verify timeout actually happened around 100ms, not 2s
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestReadStdin_EmptyInput(t *testing.T) {
	reader := &immediateEOFReader{}

	result, err := ReadStdin(reader, 1*time.Second)

	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result, got: %v", result)
	}

	if !strings.Contains(err.Error(), "[stdin-reader]") {
		t.Errorf("error should have [stdin-reader] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "No data received") {
		t.Errorf("error should mention no data received, got: %v", err)
	}
}

func TestReadStdin_LargeInput(t *testing.T) {
	// Create 1MB of data
	largeData := strings.Repeat("x", 1024*1024)
	reader := strings.NewReader(largeData)

	result, err := ReadStdin(reader, 5*time.Second)

	if err != nil {
		t.Fatalf("expected no error for large input, got: %v", err)
	}

	if len(result) != len(largeData) {
		t.Errorf("expected %d bytes, got %d bytes", len(largeData), len(result))
	}

	if string(result) != largeData {
		t.Error("large input data mismatch")
	}
}

func TestReadStdin_ImmediateEOF(t *testing.T) {
	reader := &immediateEOFReader{}

	result, err := ReadStdin(reader, 1*time.Second)

	if err == nil {
		t.Fatal("expected error for immediate EOF, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result, got: %v", result)
	}

	// Should detect as empty input, not a read error
	if !strings.Contains(err.Error(), "No data received") {
		t.Errorf("expected 'No data received' error, got: %v", err)
	}
}

func TestReadStdin_ReadError(t *testing.T) {
	expectedErr := errors.New("simulated read failure")
	reader := &errorReader{err: expectedErr}

	result, err := ReadStdin(reader, 1*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if result != nil {
		t.Errorf("expected nil result, got: %v", result)
	}

	if !strings.Contains(err.Error(), "[stdin-reader]") {
		t.Errorf("error should have [stdin-reader] prefix, got: %v", err)
	}

	if !strings.Contains(err.Error(), "Failed to read input") {
		t.Errorf("error should mention read failure, got: %v", err)
	}

	// Verify original error is wrapped
	if !strings.Contains(err.Error(), expectedErr.Error()) {
		t.Errorf("error should wrap original error, got: %v", err)
	}
}

func TestReadStdin_MultipleTimeouts(t *testing.T) {
	// Test that different timeout values work correctly
	tests := []struct {
		name    string
		delay   time.Duration
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "fast read within timeout",
			delay:   10 * time.Millisecond,
			timeout: 100 * time.Millisecond,
			wantErr: false,
		},
		{
			name:    "slow read exceeds timeout",
			delay:   200 * time.Millisecond,
			timeout: 50 * time.Millisecond,
			wantErr: true,
		},
		{
			name:    "borderline timing",
			delay:   50 * time.Millisecond,
			timeout: 500 * time.Millisecond,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reader := &slowReader{
				delay: tc.delay,
				data:  "test data",
			}

			_, err := ReadStdin(reader, tc.timeout)

			if tc.wantErr && err == nil {
				t.Error("expected timeout error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestReadStdin_ErrorMessageFormat(t *testing.T) {
	tests := []struct {
		name          string
		reader        io.Reader
		expectedParts []string
	}{
		{
			name:   "timeout error format",
			reader: &slowReader{delay: 2 * time.Second, data: "data"},
			expectedParts: []string{
				"[stdin-reader]",
				"timeout",
				"Check hook configuration",
			},
		},
		{
			name:   "empty input error format",
			reader: &immediateEOFReader{},
			expectedParts: []string{
				"[stdin-reader]",
				"No data received",
				"Hook may not be receiving STDIN input",
			},
		},
		{
			name:   "read error format",
			reader: &errorReader{err: errors.New("disk failure")},
			expectedParts: []string{
				"[stdin-reader]",
				"Failed to read input",
				"Ensure hook receives valid data",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ReadStdin(tc.reader, 100*time.Millisecond)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			for _, part := range tc.expectedParts {
				if !strings.Contains(errMsg, part) {
					t.Errorf("error message should contain %q, got: %v", part, errMsg)
				}
			}
		})
	}
}
