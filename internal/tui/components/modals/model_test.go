package modals

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// defaultKM returns the default keybinding map for tests.
func defaultKM() config.KeyMap {
	return config.DefaultKeyMap()
}

// newTestModal constructs a ModalModel for the given request.
func newTestModal(req ModalRequest) ModalModel {
	return newModalModel(req, defaultKM())
}

// pressKey simulates a key press and returns the updated model + any Cmd.
func pressKey(m ModalModel, key string) (ModalModel, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(ModalModel), cmd
}

// pressSpecialKey simulates a special key (up/down/enter/esc).
func pressSpecialKey(m ModalModel, kt tea.KeyType) (ModalModel, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: kt})
	return updated.(ModalModel), cmd
}

// extractResponse pulls a ModalResponseMsg from a tea.Cmd, failing the test
// if the cmd is nil or does not return a ModalResponseMsg.
func extractResponse(t *testing.T, cmd tea.Cmd) ModalResponseMsg {
	t.Helper()
	require.NotNil(t, cmd, "expected a non-nil tea.Cmd")
	msg := cmd()
	resp, ok := msg.(ModalResponseMsg)
	require.True(t, ok, "expected ModalResponseMsg, got %T", msg)
	return resp
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestModalModelInit(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm, Message: "Sure?"})
	cmd := m.Init()
	assert.Nil(t, cmd, "Init must return nil")
}

// ---------------------------------------------------------------------------
// Confirm modal
// ---------------------------------------------------------------------------

func TestConfirmModalDefaultSelection(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm, Message: "Proceed?"})
	assert.Equal(t, 0, m.selectedIdx, "default selection is first option (Yes)")
	assert.Equal(t, []string{"Yes", "No"}, m.effectiveOptions)
}

func TestConfirmModalNavigateDown(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm})
	m, _ = pressSpecialKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.selectedIdx)
}

func TestConfirmModalNavigateUpAtTop(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm})
	// Already at 0 — pressing Up should not go negative.
	m, _ = pressSpecialKey(m, tea.KeyUp)
	assert.Equal(t, 0, m.selectedIdx)
}

func TestConfirmModalNavigateDownAtBottom(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm})
	m, _ = pressSpecialKey(m, tea.KeyDown)
	m, _ = pressSpecialKey(m, tea.KeyDown) // try to go past "No"
	assert.Equal(t, 1, m.selectedIdx)
}

func TestConfirmModalSelectYes(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "c1", Type: Confirm, Message: "OK?"})
	// Enter on "Yes" (default selection index 0).
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "c1", resp.RequestID)
	assert.Equal(t, "Yes", resp.Response.Value)
	assert.False(t, resp.Response.Cancelled)
}

func TestConfirmModalSelectNo(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "c2", Type: Confirm})
	m, _ = pressSpecialKey(m, tea.KeyDown) // move to "No"
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "No", resp.Response.Value)
}

func TestConfirmModalEscapeCancel(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "c3", Type: Confirm})
	_, cmd := pressSpecialKey(m, tea.KeyEsc)
	resp := extractResponse(t, cmd)
	assert.True(t, resp.Response.Cancelled)
	assert.Empty(t, resp.Response.Value)
}

// ---------------------------------------------------------------------------
// Ask modal
// ---------------------------------------------------------------------------

func TestAskModalOptions(t *testing.T) {
	m := newTestModal(ModalRequest{
		Type:    Ask,
		Options: []string{"Alpha", "Beta"},
	})
	assert.Equal(t, []string{"Alpha", "Beta", "Other..."}, m.effectiveOptions)
}

func TestAskModalSelectNamedOption(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "a1", Type: Ask, Options: []string{"Option1"}})
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "Option1", resp.Response.Value)
}

func TestAskModalSelectOtherEntersInputMode(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Ask, Options: []string{"X"}})
	// Navigate to "Other..." (index 1).
	m, _ = pressSpecialKey(m, tea.KeyDown)
	assert.Equal(t, 1, m.selectedIdx)
	// Press Enter on "Other...".
	m, _ = pressSpecialKey(m, tea.KeyEnter)
	assert.True(t, m.inputMode, "should enter input mode after selecting Other...")
}

func TestAskModalTypingEntersInputMode(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Ask, Options: []string{"A", "B"}})
	// Press a printable character.
	m, _ = pressKey(m, "x")
	assert.True(t, m.inputMode, "typing should switch to input mode")
	// The cursor should jump to the "Other..." entry.
	assert.Equal(t, len(m.effectiveOptions)-1, m.selectedIdx)
}

func TestAskModalOtherSubmit(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "a2", Type: Ask, Options: []string{"A"}})
	// Navigate to "Other..." and enter input mode.
	m, _ = pressSpecialKey(m, tea.KeyDown)
	m, _ = pressSpecialKey(m, tea.KeyEnter)
	require.True(t, m.inputMode)

	// Type some text then submit.
	m.textInput.SetValue("custom response")
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "custom response", resp.Response.Value)
	assert.False(t, resp.Response.Cancelled)
}

// ---------------------------------------------------------------------------
// Input modal
// ---------------------------------------------------------------------------

func TestInputModalStartsInInputMode(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Input, Message: "Enter value:"})
	assert.True(t, m.inputMode, "Input modal must start in input mode")
	assert.Empty(t, m.effectiveOptions, "Input modal has no option list")
}

func TestInputModalSubmit(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "i1", Type: Input})
	m.textInput.SetValue("hello world")
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "hello world", resp.Response.Value)
	assert.False(t, resp.Response.Cancelled)
}

func TestInputModalEscapeCancel(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "i2", Type: Input})
	_, cmd := pressSpecialKey(m, tea.KeyEsc)
	resp := extractResponse(t, cmd)
	assert.True(t, resp.Response.Cancelled)
}

// ---------------------------------------------------------------------------
// Select modal
// ---------------------------------------------------------------------------

func TestSelectModalOptions(t *testing.T) {
	m := newTestModal(ModalRequest{
		Type:    Select,
		Options: []string{"One", "Two", "Three"},
	})
	assert.Equal(t, []string{"One", "Two", "Three"}, m.effectiveOptions)
}

func TestSelectModalNavigation(t *testing.T) {
	m := newTestModal(ModalRequest{
		Type:    Select,
		Options: []string{"P", "Q", "R"},
	})
	m, _ = pressSpecialKey(m, tea.KeyDown)
	m, _ = pressSpecialKey(m, tea.KeyDown)
	assert.Equal(t, 2, m.selectedIdx)
	m, _ = pressSpecialKey(m, tea.KeyUp)
	assert.Equal(t, 1, m.selectedIdx)
}

func TestSelectModalSelect(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "s1", Type: Select, Options: []string{"Alpha", "Beta"}})
	m, _ = pressSpecialKey(m, tea.KeyDown)
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "Beta", resp.Response.Value)
}

// ---------------------------------------------------------------------------
// Permission modal
// ---------------------------------------------------------------------------

func TestPermissionModalOptions(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Permission, Message: "Allow tool?"})
	assert.Equal(t, []string{"Allow", "Deny"}, m.effectiveOptions)
}

func TestPermissionModalAllowDefault(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "p1", Type: Permission, Message: "Allow?"})
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "Allow", resp.Response.Value)
}

func TestPermissionModalDeny(t *testing.T) {
	m := newTestModal(ModalRequest{ID: "p2", Type: Permission})
	m, _ = pressSpecialKey(m, tea.KeyDown)
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	resp := extractResponse(t, cmd)
	assert.Equal(t, "Deny", resp.Response.Value)
}

// ---------------------------------------------------------------------------
// ResponseCh delivery
// ---------------------------------------------------------------------------

func TestResponseChReceivesResponse(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	m := newTestModal(ModalRequest{
		ID:         "ch1",
		Type:       Confirm,
		ResponseCh: ch,
	})
	_, cmd := pressSpecialKey(m, tea.KeyEnter)
	// The Cmd should also trigger the message.
	require.NotNil(t, cmd)
	// Channel must have received the response.
	select {
	case resp := <-ch:
		assert.Equal(t, "Yes", resp.Value)
	default:
		t.Fatal("expected response on ResponseCh")
	}
}

func TestResponseChCancelledOnEscape(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	m := newTestModal(ModalRequest{
		ID:         "ch2",
		Type:       Select,
		Options:    []string{"X"},
		ResponseCh: ch,
	})
	_, _ = pressSpecialKey(m, tea.KeyEsc)
	select {
	case resp := <-ch:
		assert.True(t, resp.Cancelled)
	default:
		t.Fatal("expected response on ResponseCh")
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestModalModelViewContainsHeader(t *testing.T) {
	m := newTestModal(ModalRequest{
		Type:    Confirm,
		Header:  "Confirm Action",
		Message: "Are you sure?",
	})
	m.SetTermSize(120, 40)
	view := m.View()
	assert.Contains(t, view, "Confirm Action")
	assert.Contains(t, view, "Are you sure?")
}

func TestModalModelViewContainsOptions(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm, Message: "?"})
	m.SetTermSize(100, 30)
	view := m.View()
	assert.Contains(t, view, "Yes")
	assert.Contains(t, view, "No")
}

func TestModalModelViewFallbackHeader(t *testing.T) {
	// When Header is empty, the modal type name should be used.
	m := newTestModal(ModalRequest{Type: Permission, Message: "ok?"})
	m.SetTermSize(100, 30)
	view := m.View()
	assert.Contains(t, view, "Permission")
}

func TestModalModelViewContainsHints(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm})
	m.SetTermSize(100, 30)
	view := m.View()
	assert.Contains(t, view, "enter")
	assert.Contains(t, view, "esc")
}

func TestModalModelViewInputType(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Input, Message: "Type something:"})
	m.SetTermSize(100, 30)
	view := m.View()
	assert.Contains(t, view, "Type something:")
}

func TestModalModelViewSelectCursor(t *testing.T) {
	m := newTestModal(ModalRequest{
		Type:    Select,
		Options: []string{"Foo", "Bar"},
	})
	m.SetTermSize(100, 30)
	view := m.View()
	// The cursor indicator for the first (selected) option.
	assert.Contains(t, view, "> Foo")
}

// ---------------------------------------------------------------------------
// SetTermSize
// ---------------------------------------------------------------------------

func TestSetTermSize(t *testing.T) {
	m := newTestModal(ModalRequest{Type: Confirm})
	m.SetTermSize(200, 60)
	assert.Equal(t, 200, m.termWidth)
	assert.Equal(t, 60, m.termHeight)
}

// ---------------------------------------------------------------------------
// isPrintable
// ---------------------------------------------------------------------------

func TestIsPrintable(t *testing.T) {
	assert.True(t, isPrintable(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}))
	assert.False(t, isPrintable(tea.KeyMsg{Type: tea.KeyEnter}))
	assert.False(t, isPrintable(tea.KeyMsg{Type: tea.KeyRunes, Runes: nil}))
}
