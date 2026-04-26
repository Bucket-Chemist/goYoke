package claude_test

// HL-010: Tests for the three local harness slash commands.
//
// Each test verifies that the command is handled locally — it must NOT call
// the CLIDriver sender (which would forward the text to the remote Claude CLI).
// The distinguishing signal is:
//   - local harness commands → return a Harness*RequestMsg
//   - remote commands        → call stubSender.SendMessage
//
// Tests also confirm that DefaultCommands() and localCommandNames contain the
// new commands so they appear in help and cannot be shadowed by discovered skills.

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/components/slashcmd"
	"github.com/Bucket-Chemist/goYoke/internal/tui/model"
)

// ---------------------------------------------------------------------------
// /link-harness
// ---------------------------------------------------------------------------

func TestLinkHarness_IsLocal_NotForwardedToCLI(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/link-harness hermes"})

	// The sender must NOT be called — this is the core "not forwarded" assertion.
	assert.Empty(t, stub.sent, "/link-harness must not call the CLI sender")

	// The emitted command must carry the parsed HarnessLinkRequestMsg.
	msg := drainCmd(cmd)
	req, ok := msg.(model.HarnessLinkRequestMsg)
	require.True(t, ok, "expected HarnessLinkRequestMsg; got %T", msg)
	assert.Equal(t, "hermes", req.Provider)
}

func TestLinkHarness_NoArgs_ShowsUsage(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/link-harness"})

	assert.Empty(t, stub.sent, "/link-harness with no args must not call the CLI sender")

	msgs := m.Messages()
	require.NotEmpty(t, msgs, "should append a system message with usage")
	last := msgs[len(msgs)-1]
	assert.Equal(t, "system", last.Role)
	assert.Contains(t, last.Content, "/link-harness")
}

// ---------------------------------------------------------------------------
// /unlink-harness
// ---------------------------------------------------------------------------

func TestUnlinkHarness_IsLocal_NotForwardedToCLI(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/unlink-harness hermes"})

	assert.Empty(t, stub.sent, "/unlink-harness must not call the CLI sender")

	msg := drainCmd(cmd)
	req, ok := msg.(model.HarnessUnlinkRequestMsg)
	require.True(t, ok, "expected HarnessUnlinkRequestMsg; got %T", msg)
	assert.Equal(t, "hermes", req.Provider)
}

func TestUnlinkHarness_NoArgs_ShowsUsage(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	m, _ = m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/unlink-harness"})

	assert.Empty(t, stub.sent)
	msgs := m.Messages()
	require.NotEmpty(t, msgs)
	last := msgs[len(msgs)-1]
	assert.Equal(t, "system", last.Role)
	assert.Contains(t, last.Content, "/unlink-harness")
}

// ---------------------------------------------------------------------------
// /harness-status
// ---------------------------------------------------------------------------

func TestHarnessStatus_IsLocal_NotForwardedToCLI(t *testing.T) {
	stub := &stubSender{}
	m := newPanel()
	m.SetSender(stub)

	_, cmd := m.Update(slashcmd.SlashCmdSelectedMsg{Command: "/harness-status"})

	assert.Empty(t, stub.sent, "/harness-status must not call the CLI sender")

	msg := drainCmd(cmd)
	_, ok := msg.(model.HarnessStatusRequestMsg)
	require.True(t, ok, "expected HarnessStatusRequestMsg; got %T", msg)
}

// ---------------------------------------------------------------------------
// Command registration
// ---------------------------------------------------------------------------

func TestHarnessCommands_InDefaultCommands(t *testing.T) {
	cmds := slashcmd.DefaultCommands()
	names := make(map[string]bool, len(cmds))
	for _, c := range cmds {
		names[c.Name] = true
	}
	assert.True(t, names["link-harness"], "DefaultCommands must include link-harness")
	assert.True(t, names["unlink-harness"], "DefaultCommands must include unlink-harness")
	assert.True(t, names["harness-status"], "DefaultCommands must include harness-status")
}

func TestHarnessCommands_NotShadowedBySkills(t *testing.T) {
	// CommandsForNames filters local names so discovered skills can't shadow them.
	skillNames := []string{"link-harness", "unlink-harness", "harness-status"}
	filtered := slashcmd.CommandsForNames(skillNames)
	assert.Empty(t, filtered, "harness command names must be filtered from skill discovery")
}
