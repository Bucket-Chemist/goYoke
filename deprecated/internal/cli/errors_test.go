package cli

import (
	"errors"
	"testing"
	"time"
)

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		exitCode int
		wantType ErrorType
		wantMsg  string
	}{
		// Rate limit errors
		{
			name:     "rate limit 429",
			stderr:   "Error: 429 Too Many Requests",
			exitCode: 1,
			wantType: ErrorRateLimit,
			wantMsg:  "API rate limit exceeded",
		},
		{
			name:     "rate limit text",
			stderr:   "Error: Rate limit exceeded. Please try again later.",
			exitCode: 1,
			wantType: ErrorRateLimit,
			wantMsg:  "API rate limit exceeded",
		},
		{
			name:     "quota exceeded",
			stderr:   "Error: Quota exceeded for this API key",
			exitCode: 1,
			wantType: ErrorRateLimit,
			wantMsg:  "API rate limit exceeded",
		},
		{
			name:     "request limit",
			stderr:   "Error: Request limit reached",
			exitCode: 1,
			wantType: ErrorRateLimit,
			wantMsg:  "API rate limit exceeded",
		},
		{
			name:     "usage limit",
			stderr:   "Error: Usage limit exceeded",
			exitCode: 1,
			wantType: ErrorRateLimit,
			wantMsg:  "API rate limit exceeded",
		},

		// Authentication errors
		{
			name:     "invalid api key",
			stderr:   "Error: Invalid API key provided",
			exitCode: 1,
			wantType: ErrorAuthentication,
			wantMsg:  "Authentication failed",
		},
		{
			name:     "authentication failed",
			stderr:   "Error: Authentication failed",
			exitCode: 1,
			wantType: ErrorAuthentication,
			wantMsg:  "Authentication failed",
		},
		{
			name:     "unauthorized 401",
			stderr:   "Error: 401 Unauthorized",
			exitCode: 1,
			wantType: ErrorAuthentication,
			wantMsg:  "Authentication failed",
		},

		// Permission errors
		{
			name:     "permission denied",
			stderr:   "Error: Permission denied",
			exitCode: 1,
			wantType: ErrorPermission,
			wantMsg:  "Permission denied",
		},
		{
			name:     "forbidden 403",
			stderr:   "Error: 403 Forbidden",
			exitCode: 1,
			wantType: ErrorPermission,
			wantMsg:  "Permission denied",
		},
		{
			name:     "not allowed",
			stderr:   "Error: Operation not allowed",
			exitCode: 1,
			wantType: ErrorPermission,
			wantMsg:  "Permission denied",
		},

		// Network errors
		{
			name:     "network error",
			stderr:   "Error: Network error occurred",
			exitCode: 1,
			wantType: ErrorNetwork,
			wantMsg:  "Network error",
		},
		{
			name:     "connection refused",
			stderr:   "Error: Connection refused",
			exitCode: 1,
			wantType: ErrorNetwork,
			wantMsg:  "Network error",
		},
		{
			name:     "dns failure",
			stderr:   "Error: DNS lookup failed",
			exitCode: 1,
			wantType: ErrorNetwork,
			wantMsg:  "Network error",
		},
		{
			name:     "unreachable",
			stderr:   "Error: Host unreachable",
			exitCode: 1,
			wantType: ErrorNetwork,
			wantMsg:  "Network error",
		},

		// Timeout errors
		{
			name:     "timeout",
			stderr:   "Error: Operation timeout",
			exitCode: 1,
			wantType: ErrorTimeout,
			wantMsg:  "Operation timed out",
		},
		{
			name:     "timed out",
			stderr:   "Error: Request timed out",
			exitCode: 1,
			wantType: ErrorTimeout,
			wantMsg:  "Operation timed out",
		},

		// Session errors
		{
			name:     "session error",
			stderr:   "Error: Session expired",
			exitCode: 1,
			wantType: ErrorSession,
			wantMsg:  "Session error",
		},
		{
			name:     "conversation error",
			stderr:   "Error: Conversation not found",
			exitCode: 1,
			wantType: ErrorSession,
			wantMsg:  "Session error",
		},

		// MCP errors
		{
			name:     "mcp error",
			stderr:   "Error: MCP server failed to start",
			exitCode: 1,
			wantType: ErrorMCP,
			wantMsg:  "MCP server error",
		},
		{
			name:     "server error",
			stderr:   "Error: Server error occurred",
			exitCode: 1,
			wantType: ErrorMCP,
			wantMsg:  "MCP server error",
		},

		// Unknown errors
		{
			name:     "unknown error",
			stderr:   "Error: Something went wrong",
			exitCode: 1,
			wantType: ErrorUnknown,
			wantMsg:  "Error: Something went wrong",
		},
		{
			name:     "empty stderr",
			stderr:   "",
			exitCode: 1,
			wantType: ErrorUnknown,
			wantMsg:  "",
		},
		{
			name:     "multiline error",
			stderr:   "Error: First line\nSecond line\nThird line",
			exitCode: 1,
			wantType: ErrorUnknown,
			wantMsg:  "Error: First line",
		},
		{
			name:     "very long error",
			stderr:   "Error: " + string(make([]byte, 200)),
			exitCode: 1,
			wantType: ErrorUnknown,
			wantMsg:  "Error: " + string(make([]byte, 93)) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseError(tt.stderr, tt.exitCode)
			if got.Type != tt.wantType {
				t.Errorf("ParseError().Type = %v, want %v", got.Type, tt.wantType)
			}
			if got.Message != tt.wantMsg {
				t.Errorf("ParseError().Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Code != tt.exitCode {
				t.Errorf("ParseError().Code = %d, want %d", got.Code, tt.exitCode)
			}
			if got.Stderr != tt.stderr {
				t.Errorf("ParseError().Stderr = %q, want %q", got.Stderr, tt.stderr)
			}
		})
	}
}

func TestErrorType_String(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		want      string
	}{
		{ErrorUnknown, "unknown"},
		{ErrorAuthentication, "authentication"},
		{ErrorRateLimit, "rate_limit"},
		{ErrorPermission, "permission"},
		{ErrorNetwork, "network"},
		{ErrorTimeout, "timeout"},
		{ErrorSession, "session"},
		{ErrorMCP, "mcp"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.errorType.String(); got != tt.want {
				t.Errorf("ErrorType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeError_Error(t *testing.T) {
	err := &ClaudeError{
		Type:    ErrorRateLimit,
		Code:    1,
		Message: "API rate limit exceeded",
		Stderr:  "Error: 429 Too Many Requests",
	}

	want := "claude error (type=rate_limit, code=1): API rate limit exceeded"
	if got := err.Error(); got != want {
		t.Errorf("ClaudeError.Error() = %q, want %q", got, want)
	}
}

func TestClaudeError_Unwrap(t *testing.T) {
	original := errors.New("original error")
	err := &ClaudeError{
		Type:     ErrorNetwork,
		Code:     1,
		Message:  "Network error",
		Original: original,
		Stderr:   "Error: connection refused",
	}

	if got := err.Unwrap(); got != original {
		t.Errorf("ClaudeError.Unwrap() = %v, want %v", got, original)
	}

	// Test with nil original
	err2 := &ClaudeError{
		Type:    ErrorNetwork,
		Code:    1,
		Message: "Network error",
		Stderr:  "Error: connection refused",
	}

	if got := err2.Unwrap(); got != nil {
		t.Errorf("ClaudeError.Unwrap() = %v, want nil", got)
	}
}

func TestClaudeError_IsRetryable(t *testing.T) {
	tests := []struct {
		name       string
		errorType  ErrorType
		stderr     string
		message    string
		wantRetry  bool
	}{
		// Always retryable
		{"rate limit", ErrorRateLimit, "", "", true},
		{"network error", ErrorNetwork, "", "", true},
		{"timeout", ErrorTimeout, "", "", true},

		// Never retryable
		{"authentication", ErrorAuthentication, "", "", false},
		{"permission", ErrorPermission, "", "", false},
		{"session", ErrorSession, "", "", false},
		{"unknown", ErrorUnknown, "", "", false},

		// MCP - retryable if connection-related
		{"mcp connection in message", ErrorMCP, "", "connection failed", true},
		{"mcp connection in stderr", ErrorMCP, "connection error", "", true},
		{"mcp config error", ErrorMCP, "invalid config", "config error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ClaudeError{
				Type:    tt.errorType,
				Message: tt.message,
				Stderr:  tt.stderr,
			}
			if got := err.IsRetryable(); got != tt.wantRetry {
				t.Errorf("ClaudeError.IsRetryable() = %v, want %v", got, tt.wantRetry)
			}
		})
	}
}

func TestClaudeError_RetryDelay(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		stderr    string
		want      time.Duration
	}{
		// Rate limit - default
		{"rate limit default", ErrorRateLimit, "Error: 429", 60 * time.Second},
		// Rate limit - with retry-after header
		{"rate limit with retry-after colon", ErrorRateLimit, "Retry-After: 30", 30 * time.Second},
		{"rate limit with retry after space", ErrorRateLimit, "Retry After 45 seconds", 45 * time.Second},
		{"rate limit with retry-after hyphen", ErrorRateLimit, "retry-after: 120", 120 * time.Second},

		// Network and timeout
		{"network error", ErrorNetwork, "", 5 * time.Second},
		{"timeout error", ErrorTimeout, "", 5 * time.Second},

		// MCP
		{"mcp error", ErrorMCP, "", 3 * time.Second},

		// Non-retryable - zero delay
		{"authentication", ErrorAuthentication, "", 0},
		{"permission", ErrorPermission, "", 0},
		{"session", ErrorSession, "", 0},
		{"unknown", ErrorUnknown, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ClaudeError{
				Type:   tt.errorType,
				Stderr: tt.stderr,
			}
			if got := err.RetryDelay(); got != tt.want {
				t.Errorf("ClaudeError.RetryDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractRetryAfter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Duration
	}{
		{"colon separator", "retry-after: 30", 30 * time.Second},
		{"space separator", "Retry After 60 seconds", 60 * time.Second},
		{"hyphen format", "retry-after: 120", 120 * time.Second},
		{"no match", "some random text", 0},
		{"zero value", "retry-after: 0", 0},
		{"mixed case", "Retry-After: 45", 45 * time.Second},
		{"embedded in text", "Error: Rate limited. Retry-after: 90. Please wait.", 90 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractRetryAfter(tt.input); got != tt.want {
				t.Errorf("extractRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		want       bool
	}{
		{"single match", "error: rate limit", []string{"rate limit"}, true},
		{"multiple match first", "error: timeout", []string{"timeout", "network"}, true},
		{"multiple match second", "error: network", []string{"timeout", "network"}, true},
		{"no match", "error: unknown", []string{"timeout", "network"}, false},
		{"empty substrings", "error: something", []string{}, false},
		{"empty string", "", []string{"error"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsAny(tt.s, tt.substrings...); got != tt.want {
				t.Errorf("containsAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single line", "error message", "error message"},
		{"multiline", "first line\nsecond line\nthird line", "first line"},
		{"long line", string(make([]byte, 150)), string(make([]byte, 100)) + "..."},
		{"empty", "", ""},
		{"whitespace", "  \n  message  \n  ", "message"},
		{"exactly 100 chars", string(make([]byte, 100)), string(make([]byte, 100))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstLine(tt.input); got != tt.want {
				t.Errorf("firstLine() = %q, want %q", got, tt.want)
			}
		})
	}
}
