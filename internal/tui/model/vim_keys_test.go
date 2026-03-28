package model

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/settingstree"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newVimModel returns an AppModel with vim mode enabled and the terminal
// flagged as ready (width/height set so View() doesn't return "Initializing").
func newVimModel() AppModel {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	m.keys.VimEnabled = true
	m.keys.VimMode = config.VimNormal
	m.statusLine.VimEnabled = true
	m.statusLine.VimMode = config.VimNormal.String()
	return m
}

// keyMsg builds a tea.KeyMsg for a printable rune key string (single char).
func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

// specialKeyMsg builds a tea.KeyMsg for a named key type.
func specialKeyMsg(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// ---------------------------------------------------------------------------
// VimEnabled toggle via SettingChangedMsg
// ---------------------------------------------------------------------------

func TestVimKeys_SettingChangedMsg_EnablesVim(t *testing.T) {
	m := NewAppModel()
	require.False(t, m.keys.VimEnabled, "vim must be off by default")

	result, _ := m.Update(settingstree.SettingChangedMsg{Key: "vim_keys", Value: "on"})
	updated := result.(AppModel)

	assert.True(t, updated.keys.VimEnabled)
	assert.True(t, updated.statusLine.VimEnabled)
	assert.Equal(t, "NORMAL", updated.statusLine.VimMode)
}

func TestVimKeys_SettingChangedMsg_DisablesVim(t *testing.T) {
	m := newVimModel()

	result, _ := m.Update(settingstree.SettingChangedMsg{Key: "vim_keys", Value: "off"})
	updated := result.(AppModel)

	assert.False(t, updated.keys.VimEnabled)
	assert.False(t, updated.statusLine.VimEnabled)
	assert.Empty(t, updated.statusLine.VimMode)
}

func TestVimKeys_SettingChangedMsg_ResetsVimModeOnDisable(t *testing.T) {
	m := newVimModel()
	m.keys.VimMode = config.VimInsert

	result, _ := m.Update(settingstree.SettingChangedMsg{Key: "vim_keys", Value: "off"})
	updated := result.(AppModel)

	// VimMode on KeyMap should be reset to Normal.
	assert.Equal(t, config.VimNormal, updated.keys.VimMode)
}

// ---------------------------------------------------------------------------
// Normal → Insert → Normal mode transitions
// ---------------------------------------------------------------------------

func TestVimKeys_NormalToInsert_ViaI(t *testing.T) {
	m := newVimModel()
	require.Equal(t, config.VimNormal, m.keys.VimMode)

	result, _ := m.Update(keyMsg("i"))
	updated := result.(AppModel)

	assert.Equal(t, config.VimInsert, updated.keys.VimMode)
	assert.Equal(t, "INSERT", updated.statusLine.VimMode)
}

func TestVimKeys_InsertToNormal_ViaEsc(t *testing.T) {
	m := newVimModel()
	m.keys.VimMode = config.VimInsert
	m.statusLine.VimMode = config.VimInsert.String()

	result, _ := m.Update(specialKeyMsg(tea.KeyEsc))
	updated := result.(AppModel)

	assert.Equal(t, config.VimNormal, updated.keys.VimMode)
	assert.Equal(t, "NORMAL", updated.statusLine.VimMode)
}

// ---------------------------------------------------------------------------
// Mode transitions: table-driven
// ---------------------------------------------------------------------------

func TestVimKeys_ModeTransitionTable(t *testing.T) {
	tests := []struct {
		name         string
		startMode    config.VimMode
		key          tea.KeyMsg
		wantMode     config.VimMode
		wantConsumed bool
	}{
		{
			name:         "Normal+i→Insert",
			startMode:    config.VimNormal,
			key:          keyMsg("i"),
			wantMode:     config.VimInsert,
			wantConsumed: true,
		},
		{
			name:         "Insert+Esc→Normal",
			startMode:    config.VimInsert,
			key:          specialKeyMsg(tea.KeyEsc),
			wantMode:     config.VimNormal,
			wantConsumed: true,
		},
		{
			name:         "Normal+j→Normal(stays)",
			startMode:    config.VimNormal,
			key:          keyMsg("j"),
			wantMode:     config.VimNormal,
			wantConsumed: true,
		},
		{
			name:         "Normal+k→Normal(stays)",
			startMode:    config.VimNormal,
			key:          keyMsg("k"),
			wantMode:     config.VimNormal,
			wantConsumed: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newVimModel()
			m.keys.VimMode = tc.startMode

			result, _ := m.Update(tc.key)
			updated := result.(AppModel)

			assert.Equal(t, tc.wantMode, updated.keys.VimMode)
		})
	}
}

// ---------------------------------------------------------------------------
// Normal mode: j/k/h/l are consumed (not passed to text input)
// ---------------------------------------------------------------------------

func TestVimKeys_Normal_JKHLConsumedNotPassedThrough(t *testing.T) {
	// We verify this indirectly: if the key is consumed, the vim handleKey
	// returns before reaching focus-specific routing.  We use a mock Claude
	// panel and check that HandleMsg is NOT called for j/k/h/l in normal mode.
	m := newVimModel()
	panel := &mockClaudePanel{}
	m.shared.claudePanel = panel
	m.focus = FocusClaude

	for _, k := range []string{"j", "k", "h", "l"} {
		panel.handleMsgCalled = false
		m.Update(keyMsg(k))
		// j and k are routed to the panel as synthetic arrow keys, not the
		// literal "j"/"k" runes — so handleMsgCalled may be true.
		// The important assertion is that the original rune is NOT forwarded.
		// We test this by checking that the last message received is NOT "j"/"k".
		if panel.handleMsgCalled {
			// If the panel was called, it must have been with a synthetic
			// arrow key (KeyUp / KeyDown / KeyHome / KeyEnd), not the rune.
			msg, ok := panel.lastMsg.(tea.KeyMsg)
			if ok {
				assert.NotEqual(t, k, msg.String(),
					"vim normal key %q must not be forwarded as-is to the panel", k)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Insert mode: j/k/h/l pass through to standard handlers
// ---------------------------------------------------------------------------

func TestVimKeys_Insert_JKPassThroughToPanel(t *testing.T) {
	m := newVimModel()
	m.keys.VimMode = config.VimInsert
	panel := &mockClaudePanel{}
	m.shared.claudePanel = panel
	m.focus = FocusClaude

	// In insert mode, "j" must reach the Claude panel unchanged.
	m.Update(keyMsg("j"))
	assert.True(t, panel.handleMsgCalled, "panel must receive key in insert mode")
	if panel.handleMsgCalled {
		msg, ok := panel.lastMsg.(tea.KeyMsg)
		require.True(t, ok)
		assert.Equal(t, "j", msg.String(), "j must reach panel as 'j' in insert mode")
	}
}

// ---------------------------------------------------------------------------
// Vim disabled: j/k/h/l pass through to text input
// ---------------------------------------------------------------------------

func TestVimKeys_Disabled_JKPassThrough(t *testing.T) {
	m := NewAppModel()
	m.width = 120
	m.height = 40
	m.ready = true
	// vim is off by default.
	require.False(t, m.keys.VimEnabled)

	panel := &mockClaudePanel{}
	m.shared.claudePanel = panel
	m.focus = FocusClaude

	m.Update(keyMsg("j"))
	assert.True(t, panel.handleMsgCalled, "j must reach panel when vim is disabled")
	if panel.handleMsgCalled {
		msg, ok := panel.lastMsg.(tea.KeyMsg)
		require.True(t, ok)
		assert.Equal(t, "j", msg.String())
	}
}

// ---------------------------------------------------------------------------
// h/l focus switching in normal mode
// ---------------------------------------------------------------------------

func TestVimKeys_Normal_LAdvancesFocus(t *testing.T) {
	m := newVimModel()
	m.focus = FocusClaude

	result, _ := m.Update(keyMsg("l"))
	updated := result.(AppModel)

	// "l" should advance focus (same as tab).
	assert.NotEqual(t, FocusClaude, updated.focus,
		"l in vim normal mode must advance focus")
}

func TestVimKeys_Normal_HReversesFocus(t *testing.T) {
	m := newVimModel()
	m.focus = FocusAgents // start on agents so reversing gives a different focus.

	result, _ := m.Update(keyMsg("h"))
	updated := result.(AppModel)

	assert.NotEqual(t, FocusAgents, updated.focus,
		"h in vim normal mode must reverse focus")
}

// ---------------------------------------------------------------------------
// Standard global keys still work in VimNormal and VimInsert modes
// ---------------------------------------------------------------------------

func TestVimKeys_GlobalCtrlCWorksInNormalMode(t *testing.T) {
	m := newVimModel()
	m.keys.VimMode = config.VimNormal
	// Ctrl+C is the ForceQuit binding.  It must still trigger shutdown.
	// We verify the shutdownInProgress flag transitions.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := result.(AppModel)
	// First Ctrl+C sets shutdownInProgress.
	assert.True(t, updated.shutdownInProgress,
		"ctrl+c must still work in vim normal mode")
}

func TestVimKeys_GlobalCtrlCWorksInInsertMode(t *testing.T) {
	m := newVimModel()
	m.keys.VimMode = config.VimInsert
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := result.(AppModel)
	assert.True(t, updated.shutdownInProgress,
		"ctrl+c must still work in vim insert mode")
}
