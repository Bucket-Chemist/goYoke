package cli

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrorType categorizes Claude CLI errors for handling decisions
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorAuthentication // API key, auth failures
	ErrorRateLimit      // 429, quota exceeded
	ErrorPermission     // Tool access denied
	ErrorNetwork        // Connection, DNS issues
	ErrorTimeout        // Operation timed out
	ErrorSession        // Session management errors
	ErrorMCP            // MCP server errors
)

// String returns the string representation of ErrorType
func (et ErrorType) String() string {
	switch et {
	case ErrorAuthentication:
		return "authentication"
	case ErrorRateLimit:
		return "rate_limit"
	case ErrorPermission:
		return "permission"
	case ErrorNetwork:
		return "network"
	case ErrorTimeout:
		return "timeout"
	case ErrorSession:
		return "session"
	case ErrorMCP:
		return "mcp"
	default:
		return "unknown"
	}
}

// ClaudeError provides structured error information
type ClaudeError struct {
	Type     ErrorType
	Code     int    // Exit code
	Message  string // Human-readable message
	Original error  // Wrapped original error
	Stderr   string // Raw stderr output for debugging
}

// Error implements the error interface
func (e *ClaudeError) Error() string {
	return fmt.Sprintf("claude error (type=%s, code=%d): %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the original error for errors.Is/As support
func (e *ClaudeError) Unwrap() error {
	return e.Original
}

// IsRetryable returns true if this error type warrants retry
func (e *ClaudeError) IsRetryable() bool {
	switch e.Type {
	case ErrorRateLimit, ErrorNetwork, ErrorTimeout:
		return true
	case ErrorMCP:
		// MCP connection errors are retryable, config errors are not
		return strings.Contains(strings.ToLower(e.Message), "connection") ||
			strings.Contains(strings.ToLower(e.Stderr), "connection")
	}
	return false
}

// RetryDelay returns the recommended delay before retry
func (e *ClaudeError) RetryDelay() time.Duration {
	switch e.Type {
	case ErrorRateLimit:
		// Try to extract retry-after header
		if delay := extractRetryAfter(e.Stderr); delay > 0 {
			return delay
		}
		return 60 * time.Second // Default for rate limits
	case ErrorNetwork, ErrorTimeout:
		return 5 * time.Second
	case ErrorMCP:
		return 3 * time.Second
	}
	return 0
}

// ParseError classifies stderr output into a structured ClaudeError
func ParseError(stderr string, exitCode int) *ClaudeError {
	lower := strings.ToLower(stderr)

	err := &ClaudeError{
		Code:   exitCode,
		Stderr: stderr,
	}

	switch {
	case containsAny(lower, "rate limit", "too many requests", "429", "quota exceeded", "request limit", "usage limit"):
		err.Type = ErrorRateLimit
		err.Message = "API rate limit exceeded"
	case containsAny(lower, "authentication", "api key", "unauthorized", "401", "invalid key"):
		err.Type = ErrorAuthentication
		err.Message = "Authentication failed"
	case containsAny(lower, "permission", "denied", "forbidden", "403", "not allowed"):
		err.Type = ErrorPermission
		err.Message = "Permission denied"
	case containsAny(lower, "network", "connection", "dns", "unreachable", "refused"):
		err.Type = ErrorNetwork
		err.Message = "Network error"
	case containsAny(lower, "timeout", "timed out"):
		err.Type = ErrorTimeout
		err.Message = "Operation timed out"
	case containsAny(lower, "session", "conversation"):
		err.Type = ErrorSession
		err.Message = "Session error"
	case containsAny(lower, "mcp", "server error"):
		err.Type = ErrorMCP
		err.Message = "MCP server error"
	default:
		err.Type = ErrorUnknown
		err.Message = firstLine(stderr)
	}

	return err
}

// Helper functions

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

var retryAfterRe = regexp.MustCompile(`retry[- ]?after[:\s]+(\d+)`)

func extractRetryAfter(s string) time.Duration {
	matches := retryAfterRe.FindStringSubmatch(strings.ToLower(s))
	if len(matches) >= 2 {
		var seconds int
		fmt.Sscanf(matches[1], "%d", &seconds)
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return 0
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	if len(s) > 100 {
		return s[:100] + "..."
	}
	return s
}
