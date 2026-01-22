package routing

import (
	"testing"
	"time"
)

func TestDetectFailure_ExplicitFailure(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Edit",
		ToolInput:  map[string]interface{}{"file_path": "/test/file.py"},
		ToolResponse: map[string]interface{}{
			"success": false,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "explicit_failure" {
		t.Errorf("Expected error_type 'explicit_failure', got '%s'", info.ErrorType)
	}

	if info.File != "/test/file.py" {
		t.Errorf("Expected file '/test/file.py', got '%s'", info.File)
	}
}

func TestDetectFailure_ExitCode1(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "go test ./..."},
		ToolResponse: map[string]interface{}{
			"exit_code": 1,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "general_error" {
		t.Errorf("Expected error_type 'general_error', got '%s'", info.ErrorType)
	}

	if info.ExitCode != 1 {
		t.Errorf("Expected exit_code 1, got %d", info.ExitCode)
	}
}

func TestDetectFailure_ExitCode127(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "nonexistent-cmd"},
		ToolResponse: map[string]interface{}{
			"exit_code": 127,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "command_not_found" {
		t.Errorf("Expected error_type 'command_not_found', got '%s'", info.ErrorType)
	}

	if info.ExitCode != 127 {
		t.Errorf("Expected exit_code 127, got %d", info.ExitCode)
	}
}

func TestDetectFailure_TypeError(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python script.py"},
		ToolResponse: map[string]interface{}{
			"output": "Traceback (most recent call last):\nTypeError: cannot concatenate 'str' and 'int'",
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "typeerror" {
		t.Errorf("Expected error_type 'typeerror', got '%s'", info.ErrorType)
	}

	if info.ErrorMatch != "TypeError" {
		t.Errorf("Expected error_match 'TypeError', got '%s'", info.ErrorMatch)
	}
}

func TestDetectFailure_ValueError(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "python script.py"},
		ToolResponse: map[string]interface{}{
			"error": "ValueError: invalid literal for int() with base 10",
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "valueerror" {
		t.Errorf("Expected error_type 'valueerror', got '%s'", info.ErrorType)
	}

	if info.ErrorMatch != "ValueError" {
		t.Errorf("Expected error_match 'ValueError', got '%s'", info.ErrorMatch)
	}
}

func TestDetectFailure_GenericError(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "make build"},
		ToolResponse: map[string]interface{}{
			"output": "Build error: compilation failed",
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "generic_error" {
		t.Errorf("Expected error_type 'generic_error', got '%s'", info.ErrorType)
	}

	if info.ErrorMatch != "error" {
		t.Errorf("Expected error_match 'error', got '%s'", info.ErrorMatch)
	}
}

func TestDetectFailure_NoFailure(t *testing.T) {
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "echo success"},
		ToolResponse: map[string]interface{}{
			"output":    "success",
			"exit_code": 0,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info != nil {
		t.Errorf("Expected nil for successful output, got %+v", info)
	}
}

func TestDetectFailure_NilEvent(t *testing.T) {
	info := DetectFailure(nil)

	if info != nil {
		t.Errorf("Expected nil for nil event, got %+v", info)
	}
}

func TestDetectFailure_NilResponse(t *testing.T) {
	event := &PostToolEvent{
		ToolName:     "Bash",
		ToolInput:    map[string]interface{}{"command": "test"},
		ToolResponse: nil,
		CapturedAt:   time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info != nil {
		t.Errorf("Expected nil for nil response, got %+v", info)
	}
}

func TestExtractFilePath_WithFilePath(t *testing.T) {
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"file_path": "/home/user/test.go",
		},
	}

	path := ExtractFilePath(event)

	if path != "/home/user/test.go" {
		t.Errorf("Expected '/home/user/test.go', got '%s'", path)
	}
}

func TestExtractFilePath_WithCommand(t *testing.T) {
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"command": "go test ./pkg/routing",
		},
	}

	path := ExtractFilePath(event)

	if path != "./pkg/routing" {
		t.Errorf("Expected './pkg/routing', got '%s'", path)
	}
}

func TestExtractFilePath_Unknown(t *testing.T) {
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"some_other_field": "value",
		},
	}

	path := ExtractFilePath(event)

	if path != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", path)
	}
}

func TestExtractFilePath_NilEvent(t *testing.T) {
	path := ExtractFilePath(nil)

	if path != "unknown" {
		t.Errorf("Expected 'unknown' for nil event, got '%s'", path)
	}
}

func TestExtractFilePath_NilInput(t *testing.T) {
	event := &PostToolEvent{
		ToolInput: nil,
	}

	path := ExtractFilePath(event)

	if path != "unknown" {
		t.Errorf("Expected 'unknown' for nil input, got '%s'", path)
	}
}

func TestDetectFailure_ExitCodeAsFloat(t *testing.T) {
	// JSON unmarshaling often produces float64 for numbers
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"exit_code": 127.0, // float64 instead of int
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "command_not_found" {
		t.Errorf("Expected error_type 'command_not_found', got '%s'", info.ErrorType)
	}

	if info.ExitCode != 127 {
		t.Errorf("Expected exit_code 127, got %d", info.ExitCode)
	}
}

func TestDetectFailure_PriorityExplicitOverExitCode(t *testing.T) {
	// If both success=false and exit_code are present, success should take priority
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"success":   false,
			"exit_code": 1,
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "explicit_failure" {
		t.Errorf("Expected 'explicit_failure' (priority 1), got '%s'", info.ErrorType)
	}
}

func TestDetectFailure_PriorityExitCodeOverKeywords(t *testing.T) {
	// If both exit_code and error keywords are present, exit_code should take priority
	event := &PostToolEvent{
		ToolName:   "Bash",
		ToolInput:  map[string]interface{}{"command": "test"},
		ToolResponse: map[string]interface{}{
			"exit_code": 127,
			"output":    "TypeError: something went wrong",
		},
		CapturedAt: time.Now().Unix(),
	}

	info := DetectFailure(event)

	if info == nil {
		t.Fatal("Expected FailureInfo, got nil")
	}

	if info.ErrorType != "command_not_found" {
		t.Errorf("Expected 'command_not_found' (priority 2), got '%s'", info.ErrorType)
	}
}

func TestExtractPathFromCommand_MultipleArgs(t *testing.T) {
	// Test command with multiple arguments, first path-like arg should be extracted
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"command": "python src/main.py --config config.yaml",
		},
	}

	path := ExtractFilePath(event)

	// Should extract first path-like argument
	if path != "src/main.py" {
		t.Errorf("Expected 'src/main.py', got '%s'", path)
	}
}

func TestExtractPathFromCommand_NoPathLikeArgs(t *testing.T) {
	// Test command with no path-like arguments
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"command": "echo hello",
		},
	}

	path := ExtractFilePath(event)

	// Should fall back to first argument
	if path != "echo" {
		t.Errorf("Expected 'echo', got '%s'", path)
	}
}

func TestDetectErrorKeywords_CaseInsensitive(t *testing.T) {
	// Test that error detection is case-insensitive
	tests := []struct {
		text          string
		expectedType  string
		expectedMatch string
	}{
		{"TYPEERROR: something", "typeerror", "TYPEERROR"},
		{"ValueError: bad value", "valueerror", "ValueError"},
		{"attributeerror occurred", "attributeerror", "attributeerror"},
		{"Build ERROR detected", "generic_error", "ERROR"},
		{"FAILED to compile", "generic_failed", "FAILED"},
	}

	for _, tt := range tests {
		errorType, match := detectErrorKeywords(tt.text)
		if errorType != tt.expectedType {
			t.Errorf("For text '%s': expected type '%s', got '%s'", tt.text, tt.expectedType, errorType)
		}
		if match != tt.expectedMatch {
			t.Errorf("For text '%s': expected match '%s', got '%s'", tt.text, tt.expectedMatch, match)
		}
	}
}

func TestFormatExitCode_CommonCodes(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{0, "exit_code_0"},    // Zero edge case
		{1, "general_error"},
		{2, "misuse"},
		{126, "not_executable"},
		{127, "command_not_found"},
		{128, "invalid_exit"},
		{130, "interrupted"},
		{42, "exit_code_42"},   // Arbitrary code
		{255, "exit_code_255"}, // Unknown code formats as is
	}

	for _, tt := range tests {
		result := formatExitCode(tt.code)
		if result != tt.expected {
			t.Errorf("formatExitCode(%d) = '%s', expected '%s'", tt.code, result, tt.expected)
		}
	}
}

func TestExtractOutput_PreferOutput(t *testing.T) {
	// When both output and error fields exist, output should be preferred
	response := map[string]interface{}{
		"output": "stdout content",
		"error":  "stderr content",
	}

	result := extractOutput(response)

	if result != "stdout content" {
		t.Errorf("Expected 'stdout content', got '%s'", result)
	}
}

func TestExtractOutput_FallbackToError(t *testing.T) {
	// When only error field exists, use it
	response := map[string]interface{}{
		"error": "stderr content",
	}

	result := extractOutput(response)

	if result != "stderr content" {
		t.Errorf("Expected 'stderr content', got '%s'", result)
	}
}

func TestExtractOutput_EmptyResponse(t *testing.T) {
	// When neither output nor error exists, return empty string
	response := map[string]interface{}{
		"some_other_field": "value",
	}

	result := extractOutput(response)

	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestExtractFilePath_EmptyCommand(t *testing.T) {
	// Test command with empty string
	event := &PostToolEvent{
		ToolInput: map[string]interface{}{
			"command": "",
		},
	}

	path := ExtractFilePath(event)

	if path != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", path)
	}
}
