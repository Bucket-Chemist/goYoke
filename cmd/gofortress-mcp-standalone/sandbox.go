package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// -----------------------------------------------------------------------------
// Tool input / output types
// -----------------------------------------------------------------------------

// SandboxWriteInput is the input for the sandbox_write tool.
type SandboxWriteInput struct {
	Content        string `json:"content"`
	DestPath       string `json:"dest_path"`
	MakeExecutable bool   `json:"make_executable,omitempty"`
}

// SandboxWriteOutput is the response from sandbox_write.
type SandboxWriteOutput struct {
	Success      bool   `json:"success"`
	Path         string `json:"path"`
	BytesWritten int    `json:"bytes_written"`
	Error        string `json:"error,omitempty"`
}

// SandboxStatusInput is the input for the sandbox_status tool.
// All fields are optional.
type SandboxStatusInput struct{}

// SandboxStatusOutput is the response from sandbox_status.
type SandboxStatusOutput struct {
	Allowlist    []string          `json:"allowlist"`
	WriteHistory []SandboxWriteLog `json:"write_history"`
}

// SandboxWriteLog records a successful write operation.
type SandboxWriteLog struct {
	Timestamp string `json:"timestamp"`
	Path      string `json:"path"`
	Bytes     int    `json:"bytes"`
}

// -----------------------------------------------------------------------------
// Global state
// -----------------------------------------------------------------------------

// sandboxState holds the session-scoped sandbox write history.
var sandboxState = &sandboxStateStore{}

// sandboxStateStore is the sandbox state container.
type sandboxStateStore struct {
	mu           sync.Mutex
	writeHistory []SandboxWriteLog
}

// addWriteLog appends a write log entry. Thread-safe.
func (s *sandboxStateStore) addWriteLog(path string, bytes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeHistory = append(s.writeHistory, SandboxWriteLog{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Path:      path,
		Bytes:     bytes,
	})
}

// getHistory returns a copy of the write history. Thread-safe.
func (s *sandboxStateStore) getHistory() []SandboxWriteLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]SandboxWriteLog, len(s.writeHistory))
	copy(result, s.writeHistory)
	return result
}

// -----------------------------------------------------------------------------
// Path helpers
// -----------------------------------------------------------------------------

const maxContentSize = 512 * 1024 // 512KB

// getProjectRoot returns the effective project root directory.
// Priority: GOFORTRESS_PROJECT_ROOT > GOGENT_PROJECT_DIR > CLAUDE_PROJECT_DIR > git root.
func getProjectRoot() string {
	if dir := os.Getenv("GOFORTRESS_PROJECT_ROOT"); dir != "" {
		return dir
	}
	if dir := os.Getenv("GOGENT_PROJECT_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("CLAUDE_PROJECT_DIR"); dir != "" {
		return dir
	}
	// Try git root.
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// getSandboxClaudeDir returns the ~/.claude directory path.
func getSandboxClaudeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = filepath.Join("/home", os.Getenv("USER"))
	}
	return filepath.Join(home, ".claude")
}

// resolveRoot resolves symlinks on a root directory, falling back to the
// original if resolution fails.
func resolveRoot(root string) string {
	if real, err := filepath.EvalSymlinks(root); err == nil {
		return real
	}
	return root
}

// validateSandboxPath checks whether destPath is an allowed write target.
// It returns nil if the path is valid, or an error describing the violation.
func validateSandboxPath(destPath string) error {
	// Check raw input for path traversal components.
	for _, part := range strings.Split(filepath.ToSlash(destPath), "/") {
		if part == ".." {
			return fmt.Errorf("path traversal not allowed: %q", destPath)
		}
	}

	cleaned := filepath.Clean(destPath)

	// Reject writes to .git/ directories.
	sep := string(filepath.Separator)
	if strings.Contains(cleaned, sep+".git"+sep) ||
		strings.HasSuffix(cleaned, sep+".git") {
		return fmt.Errorf("writes to .git directories are not allowed: %q", destPath)
	}

	// Resolve symlinks on the path itself for accurate root comparison.
	// If the path doesn't exist, use the cleaned version.
	resolved := cleaned
	if real, err := filepath.EvalSymlinks(cleaned); err == nil {
		resolved = real
	}

	claudeDir := getSandboxClaudeDir()
	projectRoot := getProjectRoot()

	// Check if under ~/.claude/
	if claudeDir != "" {
		rc := resolveRoot(claudeDir)
		if strings.HasPrefix(resolved, rc+sep) || resolved == rc {
			return nil
		}
	}

	// Check if under project root.
	if projectRoot != "" {
		rr := resolveRoot(projectRoot)
		if strings.HasPrefix(resolved, rr+sep) || resolved == rr {
			return nil
		}
	}

	return fmt.Errorf("path %q is not under allowed directories (project root or ~/.claude/)", destPath)
}

// -----------------------------------------------------------------------------
// Tool registration
// -----------------------------------------------------------------------------

// registerSandboxWrite registers the sandbox_write tool.
func registerSandboxWrite(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_write",
		Description: "Write a file to a protected path (e.g. .claude/ dirs) that Claude Code's sandbox blocks. Validates path against allowlist before writing.",
	}, handleSandboxWrite)
}

// registerSandboxStatus registers the sandbox_status tool.
func registerSandboxStatus(server *mcpsdk.Server) {
	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "sandbox_status",
		Description: "Return the sandbox allowlist and this session's write history.",
	}, handleSandboxStatus)
}

// -----------------------------------------------------------------------------
// Tool handlers
// -----------------------------------------------------------------------------

// handleSandboxWrite handles the sandbox_write tool call.
func handleSandboxWrite(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	input SandboxWriteInput,
) (*mcpsdk.CallToolResult, SandboxWriteOutput, error) {
	_ = ctx

	if input.DestPath == "" {
		return nil, SandboxWriteOutput{}, fmt.Errorf("sandbox_write: dest_path is required")
	}

	// Content size check.
	if len(input.Content) > maxContentSize {
		return nil, SandboxWriteOutput{
			Success: false,
			Error:   fmt.Sprintf("content size %d exceeds maximum of %d bytes", len(input.Content), maxContentSize),
		}, nil
	}

	// Path validation.
	if err := validateSandboxPath(input.DestPath); err != nil {
		return nil, SandboxWriteOutput{
			Success: false,
			Path:    input.DestPath,
			Error:   err.Error(),
		}, nil
	}

	cleaned := filepath.Clean(input.DestPath)

	// Create parent directories.
	if err := os.MkdirAll(filepath.Dir(cleaned), 0o755); err != nil {
		return nil, SandboxWriteOutput{
			Success: false,
			Path:    cleaned,
			Error:   fmt.Sprintf("create parent directories: %v", err),
		}, nil
	}

	// Determine file mode.
	mode := os.FileMode(0o644)
	if input.MakeExecutable {
		mode = 0o755
	}

	// Write file.
	if err := os.WriteFile(cleaned, []byte(input.Content), mode); err != nil {
		return nil, SandboxWriteOutput{
			Success: false,
			Path:    cleaned,
			Error:   fmt.Sprintf("write file: %v", err),
		}, nil
	}

	bytes := len(input.Content)
	sandboxState.addWriteLog(cleaned, bytes)

	return nil, SandboxWriteOutput{
		Success:      true,
		Path:         cleaned,
		BytesWritten: bytes,
	}, nil
}

// handleSandboxStatus handles the sandbox_status tool call.
func handleSandboxStatus(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	input SandboxStatusInput,
) (*mcpsdk.CallToolResult, SandboxStatusOutput, error) {
	_ = input

	claudeDir := getSandboxClaudeDir()
	projectRoot := getProjectRoot()

	allowlist := []string{}
	if projectRoot != "" {
		allowlist = append(allowlist, projectRoot)
	}
	if claudeDir != "" {
		allowlist = append(allowlist, claudeDir)
	}

	history := sandboxState.getHistory()
	if history == nil {
		history = []SandboxWriteLog{}
	}

	return nil, SandboxStatusOutput{
		Allowlist:    allowlist,
		WriteHistory: history,
	}, nil
}
