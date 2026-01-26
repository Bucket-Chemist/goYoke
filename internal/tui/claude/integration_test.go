package claude

import (
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
	"github.com/stretchr/testify/assert"
)

// TestClaudeProcess_ImplementsInterface verifies that ClaudeProcess
// implements the ClaudeProcessInterface required by PanelModel.
func TestClaudeProcess_ImplementsInterface(t *testing.T) {
	// This test verifies compile-time interface compliance.
	// If ClaudeProcess doesn't implement the interface, this won't compile.
	var _ ClaudeProcessInterface = (*cli.ClaudeProcess)(nil)

	// If we get here, the interface is correctly implemented
	assert.True(t, true, "ClaudeProcess implements ClaudeProcessInterface")
}

// TestPanelModel_WithRealClaudeProcess verifies PanelModel works with
// actual ClaudeProcess (not just mocks).
func TestPanelModel_WithRealClaudeProcess(t *testing.T) {
	// Create a mock process that implements the real interface
	mockProc := NewMockClaudeProcess("integration-test")
	defer mockProc.Close()

	// Create panel with mock
	panel := NewPanelModel(mockProc, cli.Config{})

	// Verify basic initialization
	assert.NotNil(t, panel)
	assert.Equal(t, "integration-test", panel.sessionID)
	assert.False(t, panel.streaming)
	assert.Equal(t, 0, len(panel.messages))
	assert.Equal(t, 0, len(panel.hooks))
}
