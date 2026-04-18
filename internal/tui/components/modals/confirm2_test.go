package modals

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/goYoke/internal/tui/config"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestConfirm2 builds a TwoStepConfirmModal with a fixed test phrase.
func newTestConfirm2(title, description, phrase string) TwoStepConfirmModal {
	return NewTwoStepConfirmModal(title, description, phrase)
}

// typePhrase simulates typing each character of s into the modal by forwarding
// individual KeyRunes messages and returns the final model.
func typePhrase(m TwoStepConfirmModal, s string) TwoStepConfirmModal {
	for _, r := range s {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated
	}
	return m
}

// pressSpecialKey2 sends a special key to a TwoStepConfirmModal and returns
// the updated model plus any command.
func pressSpecialKey2(m TwoStepConfirmModal, kt tea.KeyType) (TwoStepConfirmModal, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: kt})
	return updated, cmd
}

// extractResponse2 pulls a ModalResponseMsg from a tea.Cmd, failing the test
// if cmd is nil or does not return a ModalResponseMsg.
func extractResponse2(t *testing.T, cmd tea.Cmd) ModalResponseMsg {
	t.Helper()
	require.NotNil(t, cmd, "expected a non-nil tea.Cmd carrying ModalResponseMsg")
	msg := cmd()
	resp, ok := msg.(ModalResponseMsg)
	require.True(t, ok, "expected ModalResponseMsg, got %T", msg)
	return resp
}

// ---------------------------------------------------------------------------
// Construction
// ---------------------------------------------------------------------------

func TestNewTwoStepConfirmModal_Fields(t *testing.T) {
	m := NewTwoStepConfirmModal("Delete Session", "This will erase all data.", "delete session")
	assert.Equal(t, "Delete Session", m.Title)
	assert.Equal(t, "This will erase all data.", m.Description)
	assert.Equal(t, "delete session", m.RequiredPhrase)
	assert.False(t, m.IsConfirmed(), "must not start confirmed")
}

func TestNewTwoStepConfirmModal_DefaultTermSize(t *testing.T) {
	m := NewTwoStepConfirmModal("T", "D", "phrase")
	assert.Equal(t, 80, m.termWidth)
	assert.Equal(t, 24, m.termHeight)
}

func TestTwoStepConfirmModal_WithRequestID(t *testing.T) {
	m := NewTwoStepConfirmModal("T", "D", "phrase").WithRequestID("req-42")
	assert.Equal(t, "req-42", m.requestID)
}

func TestTwoStepConfirmModal_WithResponseCh(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	m := NewTwoStepConfirmModal("T", "D", "phrase").WithResponseCh(ch)
	assert.Equal(t, ch, m.responseCh)
}

func TestTwoStepConfirmModal_SetTermSize(t *testing.T) {
	m := NewTwoStepConfirmModal("T", "D", "phrase")
	m.SetTermSize(160, 50)
	assert.Equal(t, 160, m.termWidth)
	assert.Equal(t, 50, m.termHeight)
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func TestTwoStepConfirmModal_Init_NotNil(t *testing.T) {
	m := NewTwoStepConfirmModal("T", "D", "phrase")
	cmd := m.Init()
	// Init returns textinput.Blink which is a non-nil Cmd.
	assert.NotNil(t, cmd, "Init must return the textinput blink command")
}

// ---------------------------------------------------------------------------
// phraseMatches — case sensitivity
// ---------------------------------------------------------------------------

func TestPhraseMatches_ExactMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete session")
	assert.True(t, m.phraseMatches(), "exact match (same case) must return true")
}

func TestPhraseMatches_CaseInsensitiveMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "Delete Session")
	assert.True(t, m.phraseMatches(), "uppercase input must still match lowercase phrase")
}

func TestPhraseMatches_UppercasePhrase_LowercaseInput(t *testing.T) {
	m := newTestConfirm2("T", "D", "DELETE SESSION")
	m = typePhrase(m, "delete session")
	assert.True(t, m.phraseMatches())
}

func TestPhraseMatches_PartialMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete")
	assert.False(t, m.phraseMatches(), "partial match must return false")
}

func TestPhraseMatches_NoMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "xyz")
	assert.False(t, m.phraseMatches())
}

func TestPhraseMatches_Empty(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	assert.False(t, m.phraseMatches(), "empty input must not match")
}

// ---------------------------------------------------------------------------
// IsConfirmed
// ---------------------------------------------------------------------------

func TestIsConfirmed_FalseBeforeEnter(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete session")
	// Phrase matches but Enter not pressed yet.
	assert.False(t, m.IsConfirmed())
}

func TestIsConfirmed_TrueAfterEnter(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete session")
	m, _ = pressSpecialKey2(m, tea.KeyEnter)
	assert.True(t, m.IsConfirmed())
}

// ---------------------------------------------------------------------------
// Enter key behaviour
// ---------------------------------------------------------------------------

func TestEnterBeforeConfirmed_NoResponseEmitted(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete")  // partial — not confirmed
	_, cmd := pressSpecialKey2(m, tea.KeyEnter)
	assert.Nil(t, cmd, "Enter before phrase matches must not emit a response")
}

func TestEnterAfterConfirmed_ResponseEmitted(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session").WithRequestID("req-1")
	m = typePhrase(m, "delete session")
	_, cmd := pressSpecialKey2(m, tea.KeyEnter)
	resp := extractResponse2(t, cmd)
	assert.Equal(t, "req-1", resp.RequestID)
	assert.False(t, resp.Response.Cancelled)
	assert.Equal(t, TwoStepConfirm, resp.Response.Type)
}

func TestEnterAfterCaseInsensitiveMatch_ResponseEmitted(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "DELETE SESSION")
	_, cmd := pressSpecialKey2(m, tea.KeyEnter)
	resp := extractResponse2(t, cmd)
	assert.False(t, resp.Response.Cancelled)
}

func TestEnterOnEmptyInput_NoResponseEmitted(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	// No typing at all.
	_, cmd := pressSpecialKey2(m, tea.KeyEnter)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// Escape cancels
// ---------------------------------------------------------------------------

func TestEscape_Cancels(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session").WithRequestID("req-esc")
	_, cmd := pressSpecialKey2(m, tea.KeyEsc)
	resp := extractResponse2(t, cmd)
	assert.Equal(t, "req-esc", resp.RequestID)
	assert.True(t, resp.Response.Cancelled)
	assert.Equal(t, TwoStepConfirm, resp.Response.Type)
}

func TestEscape_CancelsEvenAfterTyping(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete session")
	_, cmd := pressSpecialKey2(m, tea.KeyEsc)
	resp := extractResponse2(t, cmd)
	assert.True(t, resp.Response.Cancelled)
}

// ---------------------------------------------------------------------------
// ResponseCh delivery
// ---------------------------------------------------------------------------

func TestResponseCh_ReceivesConfirmation(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	m := newTestConfirm2("T", "D", "delete session").WithResponseCh(ch)
	m = typePhrase(m, "delete session")
	_, _ = pressSpecialKey2(m, tea.KeyEnter)
	select {
	case resp := <-ch:
		assert.False(t, resp.Cancelled)
		assert.Equal(t, TwoStepConfirm, resp.Type)
	default:
		t.Fatal("expected response on ResponseCh but channel was empty")
	}
}

func TestResponseCh_ReceivesCancellation(t *testing.T) {
	ch := make(chan ModalResponse, 1)
	m := newTestConfirm2("T", "D", "delete session").WithResponseCh(ch)
	_, _ = pressSpecialKey2(m, tea.KeyEsc)
	select {
	case resp := <-ch:
		assert.True(t, resp.Cancelled)
	default:
		t.Fatal("expected cancellation on ResponseCh but channel was empty")
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func TestView_ContainsTitle(t *testing.T) {
	m := newTestConfirm2("Delete Session", "This will erase all data.", "delete session")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, "Delete Session")
}

func TestView_ContainsDescription(t *testing.T) {
	m := newTestConfirm2("T", "This will erase all data.", "delete session")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, "This will erase all data.")
}

func TestView_ContainsRequiredPhrase(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, "delete session")
}

func TestView_ContainsHint(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, "esc")
}

func TestView_HintChangesWhenPhraseMatched(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m.SetTermSize(120, 40)

	vBefore := m.View()
	m = typePhrase(m, "delete session")
	vAfter := m.View()

	// Before match: hint says "type phrase to enable enter"
	assert.Contains(t, vBefore, "type phrase to enable enter")
	// After match: hint says "enter: confirm"
	assert.Contains(t, vAfter, "enter: confirm")
}

func TestView_ContainsShadow(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, shadowChar, "View must render a drop shadow")
}

func TestView_FallbackTitleUsesTypeName(t *testing.T) {
	// Empty title falls back to "TwoStepConfirm".
	m := NewTwoStepConfirmModal("", "", "phrase")
	m.SetTermSize(120, 40)
	v := m.View()
	assert.Contains(t, v, "TwoStepConfirm")
}

// ---------------------------------------------------------------------------
// renderProgress
// ---------------------------------------------------------------------------

func TestRenderProgress_NoInput(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	progress := m.renderProgress()
	// No match — entire phrase should be present in the dim style rendering.
	assert.Contains(t, progress, "delete session")
}

func TestRenderProgress_PartialMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "del")
	progress := m.renderProgress()
	// Both portions must appear somewhere in the rendered string.
	assert.Contains(t, progress, "del")
	assert.Contains(t, progress, "ete session")
}

func TestRenderProgress_FullMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "delete session")
	progress := m.renderProgress()
	assert.Contains(t, progress, "delete session")
}

func TestRenderProgress_WrongInput_ShowsFullPhraseUnmatched(t *testing.T) {
	m := newTestConfirm2("T", "D", "delete session")
	m = typePhrase(m, "xyz")
	progress := m.renderProgress()
	// Mismatch at first character: all dim.
	assert.Contains(t, progress, "delete session")
}

// ---------------------------------------------------------------------------
// ModalType enum additions
// ---------------------------------------------------------------------------

func TestTwoStepConfirmModalType_Value(t *testing.T) {
	// TwoStepConfirm must be 6 (one past PlanView=5).
	assert.Equal(t, ModalType(6), TwoStepConfirm)
}

func TestTwoStepConfirmModalType_String(t *testing.T) {
	assert.Equal(t, "TwoStepConfirm", TwoStepConfirm.String())
}

// ---------------------------------------------------------------------------
// modalBorderColor for TwoStepConfirm
// ---------------------------------------------------------------------------

func TestModalBorderColor_TwoStepConfirm_IsError(t *testing.T) {
	color := modalBorderColor(TwoStepConfirm)
	assert.Equal(t, config.ColorError, color, "TwoStepConfirm modals must use ColorError (red)")
}

// ---------------------------------------------------------------------------
// Non-key messages forwarded to textinput
// ---------------------------------------------------------------------------

func TestUpdate_NonKeyMsg_ForwardedToInput(t *testing.T) {
	m := newTestConfirm2("T", "D", "phrase")
	// A non-key, non-modal message should be forwarded to the text input
	// and return a (possibly nil) Cmd without panicking.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.NotNil(t, updated, "Update must always return a model")
}

// ---------------------------------------------------------------------------
// Table-driven phrase-matching tests
// ---------------------------------------------------------------------------

func TestPhraseMatches_Table(t *testing.T) {
	tests := []struct {
		name     string
		phrase   string
		input    string
		wantTrue bool
	}{
		{
			name:     "exact lowercase",
			phrase:   "delete session",
			input:    "delete session",
			wantTrue: true,
		},
		{
			name:     "input uppercase",
			phrase:   "delete session",
			input:    "Delete Session",
			wantTrue: true,
		},
		{
			name:     "phrase uppercase",
			phrase:   "DELETE SESSION",
			input:    "delete session",
			wantTrue: true,
		},
		{
			name:     "mixed case both sides",
			phrase:   "Delete Session",
			input:    "DELETE SESSION",
			wantTrue: true,
		},
		{
			name:     "partial match",
			phrase:   "delete session",
			input:    "delete",
			wantTrue: false,
		},
		{
			name:     "no match at all",
			phrase:   "delete session",
			input:    "xyz abc",
			wantTrue: false,
		},
		{
			name:     "empty input",
			phrase:   "delete session",
			input:    "",
			wantTrue: false,
		},
		{
			name:     "extra character",
			phrase:   "delete session",
			input:    "delete session!",
			wantTrue: false,
		},
		{
			name:     "trailing whitespace trimmed",
			phrase:   "delete session",
			input:    "delete session   ",
			wantTrue: true,
		},
		{
			name:     "leading whitespace trimmed",
			phrase:   "delete session",
			input:    "   delete session",
			wantTrue: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestConfirm2("T", "D", tc.phrase)
			// Directly set the value to avoid character-by-character simulation
			// for tests that just verify phraseMatches.
			m.input.SetValue(tc.input)
			got := m.phraseMatches()
			if tc.wantTrue {
				assert.True(t, got)
			} else {
				assert.False(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Table-driven enter key tests
// ---------------------------------------------------------------------------

func TestEnterKey_Table(t *testing.T) {
	tests := []struct {
		name         string
		phrase       string
		typedValue   string
		wantResponse bool
		wantCancel   bool
	}{
		{
			name:         "enter after exact match emits confirmed",
			phrase:       "delete session",
			typedValue:   "delete session",
			wantResponse: true,
			wantCancel:   false,
		},
		{
			name:         "enter after case-insensitive match emits confirmed",
			phrase:       "delete session",
			typedValue:   "DELETE SESSION",
			wantResponse: true,
			wantCancel:   false,
		},
		{
			name:         "enter on partial match does not emit",
			phrase:       "delete session",
			typedValue:   "delete",
			wantResponse: false,
		},
		{
			name:         "enter on empty input does not emit",
			phrase:       "delete session",
			typedValue:   "",
			wantResponse: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestConfirm2("T", "D", tc.phrase)
			m.input.SetValue(tc.typedValue)
			_, cmd := pressSpecialKey2(m, tea.KeyEnter)
			if tc.wantResponse {
				require.NotNil(t, cmd, "expected a Cmd but got nil")
				msg := cmd()
				resp, ok := msg.(ModalResponseMsg)
				require.True(t, ok, "expected ModalResponseMsg, got %T", msg)
				assert.Equal(t, tc.wantCancel, resp.Response.Cancelled)
			} else {
				assert.Nil(t, cmd, "expected no Cmd for unmatched phrase")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// renderHint
// ---------------------------------------------------------------------------

func TestRenderHint_BeforeMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "phrase")
	hint := m.renderHint()
	assert.Contains(t, hint, "type phrase to enable enter")
	assert.Contains(t, hint, "esc")
}

func TestRenderHint_AfterMatch(t *testing.T) {
	m := newTestConfirm2("T", "D", "phrase")
	m.input.SetValue("phrase")
	hint := m.renderHint()
	assert.Contains(t, hint, "enter: confirm")
	assert.Contains(t, strings.ToLower(hint), "esc")
}
