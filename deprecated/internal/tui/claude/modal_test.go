package claude

import (
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/deprecated/internal/callback"
)

// TestNewModalState verifies the initialization of a new ModalState
func TestNewModalState(t *testing.T) {
	t.Helper()

	m := NewModalState()

	if m.Active {
		t.Error("NewModalState() Active should be false, got true")
	}

	if m.Type != NoModal {
		t.Errorf("NewModalState() Type = %v, want %v", m.Type, NoModal)
	}

	if m.Server != nil {
		t.Error("NewModalState() Server should be nil")
	}

	// Verify TextInput is properly initialized
	if m.TextInput.Placeholder != "Type your response..." {
		t.Errorf("NewModalState() TextInput.Placeholder = %q, want %q",
			m.TextInput.Placeholder, "Type your response...")
	}

	if m.TextInput.CharLimit != 500 {
		t.Errorf("NewModalState() TextInput.CharLimit = %d, want %d",
			m.TextInput.CharLimit, 500)
	}

	if m.TextInput.Width != 40 {
		t.Errorf("NewModalState() TextInput.Width = %d, want %d",
			m.TextInput.Width, 40)
	}
}

// TestHandlePrompt verifies modal activation and type selection for different prompt types
func TestHandlePrompt(t *testing.T) {
	t.Helper()

	tests := []struct {
		name       string
		prompt     callback.PromptRequest
		wantType   ModalType
		wantActive bool
		wantCmd    bool // whether we expect a non-nil tea.Cmd
		checkInput func(t *testing.T, m *ModalState)
	}{
		{
			name: "confirm type activates ConfirmModal",
			prompt: callback.PromptRequest{
				ID:      "confirm-1",
				Type:    "confirm",
				Message: "Are you sure?",
			},
			wantType:   ConfirmModal,
			wantActive: true,
			wantCmd:    false,
		},
		{
			name: "input type without options activates TextInputModal",
			prompt: callback.PromptRequest{
				ID:      "input-1",
				Type:    "input",
				Message: "Enter your name:",
			},
			wantType:   TextInputModal,
			wantActive: true,
			wantCmd:    true, // Focus() returns a cmd
			checkInput: func(t *testing.T, m *ModalState) {
				t.Helper()
				if m.TextInput.Value() != "" {
					t.Errorf("TextInput should be empty, got %q", m.TextInput.Value())
				}
			},
		},
		{
			name: "input type with default value sets TextInput value",
			prompt: callback.PromptRequest{
				ID:      "input-2",
				Type:    "input",
				Message: "Enter port:",
				Default: "8080",
			},
			wantType:   TextInputModal,
			wantActive: true,
			wantCmd:    true,
			checkInput: func(t *testing.T, m *ModalState) {
				t.Helper()
				if m.TextInput.Value() != "8080" {
					t.Errorf("TextInput.Value() = %q, want %q", m.TextInput.Value(), "8080")
				}
			},
		},
		{
			name: "ask type without options activates TextInputModal",
			prompt: callback.PromptRequest{
				ID:      "ask-1",
				Type:    "ask",
				Message: "What is your quest?",
			},
			wantType:   TextInputModal,
			wantActive: true,
			wantCmd:    true,
		},
		{
			name: "input type with options activates SelectionModal",
			prompt: callback.PromptRequest{
				ID:      "input-select-1",
				Type:    "input",
				Message: "Choose environment:",
				Options: []string{"dev", "staging", "prod"},
			},
			wantType:   SelectionModal,
			wantActive: true,
			wantCmd:    false,
			checkInput: func(t *testing.T, m *ModalState) {
				t.Helper()
				items := m.SelectList.Items()
				if len(items) != 3 {
					t.Errorf("SelectList items length = %d, want 3", len(items))
				}
			},
		},
		{
			name: "select type activates SelectionModal",
			prompt: callback.PromptRequest{
				ID:      "select-1",
				Type:    "select",
				Message: "Pick a color:",
				Options: []string{"red", "green", "blue"},
			},
			wantType:   SelectionModal,
			wantActive: true,
			wantCmd:    false,
			checkInput: func(t *testing.T, m *ModalState) {
				t.Helper()
				items := m.SelectList.Items()
				if len(items) != 3 {
					t.Errorf("SelectList items length = %d, want 3", len(items))
				}
				// Verify first item
				if item, ok := items[0].(listItem); ok {
					if item.title != "red" {
						t.Errorf("First item title = %q, want %q", item.title, "red")
					}
				} else {
					t.Error("SelectList item is not a listItem")
				}
			},
		},
		{
			name: "unknown type falls back to TextInputModal",
			prompt: callback.PromptRequest{
				ID:      "unknown-1",
				Type:    "unknown-type",
				Message: "Mystery prompt",
			},
			wantType:   TextInputModal,
			wantActive: true,
			wantCmd:    true,
		},
		{
			name: "empty type falls back to TextInputModal",
			prompt: callback.PromptRequest{
				ID:      "empty-1",
				Type:    "",
				Message: "No type specified",
			},
			wantType:   TextInputModal,
			wantActive: true,
			wantCmd:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			m := NewModalState()
			server, _ := setupModalTest(tt.prompt.ID)

			cmd := m.HandlePrompt(tt.prompt, server)

			// Verify Active state
			if m.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", m.Active, tt.wantActive)
			}

			// Verify ModalType
			if m.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", m.Type, tt.wantType)
			}

			// Verify prompt was stored
			if m.Prompt.ID != tt.prompt.ID {
				t.Errorf("Prompt.ID = %q, want %q", m.Prompt.ID, tt.prompt.ID)
			}

			// Verify server was stored
			if m.Server == nil {
				t.Error("Server not properly assigned")
			}

			// Verify command return
			if tt.wantCmd {
				if cmd == nil {
					t.Error("Expected non-nil tea.Cmd, got nil")
				}
			} else {
				if cmd != nil {
					t.Error("Expected nil tea.Cmd, got non-nil")
				}
			}

			// Run additional checks if provided
			if tt.checkInput != nil {
				tt.checkInput(t, &m)
			}
		})
	}
}

// TestHandlePrompt_TextInputReset verifies TextInput is reset between prompts
func TestHandlePrompt_TextInputReset(t *testing.T) {
	t.Helper()

	m := NewModalState()

	// First prompt with value
	m.TextInput.SetValue("old value")

	// Second prompt should reset
	prompt := callback.PromptRequest{
		ID:      "reset-test",
		Type:    "input",
		Message: "New prompt",
	}
	server, _ := setupModalTest(prompt.ID)

	m.HandlePrompt(prompt, server)

	if m.TextInput.Value() != "" {
		t.Errorf("TextInput not reset, got value: %q", m.TextInput.Value())
	}
}

// TestSendResponse verifies response delivery and state cleanup
func TestSendResponse(t *testing.T) {
	t.Helper()

	tests := []struct {
		name      string
		value     string
		cancelled bool
		promptID  string
	}{
		{
			name:      "successful response",
			value:     "user input",
			cancelled: false,
			promptID:  "prompt-1",
		},
		{
			name:      "cancelled response",
			value:     "",
			cancelled: true,
			promptID:  "prompt-2",
		},
		{
			name:      "response with value but cancelled",
			value:     "partial input",
			cancelled: true,
			promptID:  "prompt-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			m := NewModalState()
			server, respChan := setupModalTest(tt.promptID)

			// Setup modal state
			m.Active = true
			m.Prompt = callback.PromptRequest{ID: tt.promptID}
			m.Server = server

			// Send response
			m.SendResponse(tt.value, tt.cancelled)

			// Verify response was sent
			select {
			case resp := <-respChan:
				if resp.ID != tt.promptID {
					t.Errorf("Response.ID = %q, want %q", resp.ID, tt.promptID)
				}
				if resp.Value != tt.value {
					t.Errorf("Response.Value = %q, want %q", resp.Value, tt.value)
				}
				if resp.Cancelled != tt.cancelled {
					t.Errorf("Response.Cancelled = %v, want %v", resp.Cancelled, tt.cancelled)
				}
			default:
				t.Error("No response received on channel")
			}

			// Verify state cleanup
			if m.Active {
				t.Error("Active should be false after SendResponse")
			}

			if m.Server != nil {
				t.Error("Server should be nil after SendResponse")
			}
		})
	}
}

// TestSendResponse_NilChannel verifies safety when ResponseChan is nil
func TestSendResponse_NilChannel(t *testing.T) {
	t.Helper()

	m := NewModalState()

	// Ensure Server is nil
	m.Server = nil
	m.Active = true

	// Should not panic
	m.SendResponse("test", false)

	// State should remain unchanged
	if !m.Active {
		t.Error("Active should remain true when Server is nil")
	}
}

// TestListItemImplementation verifies listItem satisfies list.Item interface
func TestListItemImplementation(t *testing.T) {
	t.Helper()

	item := listItem{title: "test item"}

	if got := item.Title(); got != "test item" {
		t.Errorf("Title() = %q, want %q", got, "test item")
	}

	if got := item.Description(); got != "" {
		t.Errorf("Description() = %q, want empty string", got)
	}

	if got := item.FilterValue(); got != "test item" {
		t.Errorf("FilterValue() = %q, want %q", got, "test item")
	}
}

// TestCreateSelectList verifies selection list creation
func TestCreateSelectList(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		options []string
	}{
		{
			name:    "empty options",
			options: []string{},
		},
		{
			name:    "single option",
			options: []string{"only"},
		},
		{
			name:    "multiple options",
			options: []string{"one", "two", "three"},
		},
		{
			name:    "many options exceeds max height",
			options: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			list := createSelectList(tt.options)

			// Verify items count
			items := list.Items()
			if len(items) != len(tt.options) {
				t.Errorf("Items count = %d, want %d", len(items), len(tt.options))
			}

			// Verify items match options
			for i, opt := range tt.options {
				if item, ok := items[i].(listItem); ok {
					if item.title != opt {
						t.Errorf("Item[%d].title = %q, want %q", i, item.title, opt)
					}
				} else {
					t.Errorf("Item[%d] is not a listItem", i)
				}
			}

			// Verify list properties
			if list.ShowTitle() {
				t.Error("ShowTitle should be false")
			}
			if list.ShowStatusBar() {
				t.Error("ShowStatusBar should be false")
			}
			if list.ShowHelp() {
				t.Error("ShowHelp should be false")
			}
			if list.FilteringEnabled() {
				t.Error("FilteringEnabled should be false")
			}
		})
	}
}

// TestMinFunction verifies the min helper function
func TestMinFunction(t *testing.T) {
	t.Helper()

	tests := []struct {
		name string
		a    int
		b    int
		want int
	}{
		{
			name: "a less than b",
			a:    5,
			b:    10,
			want: 5,
		},
		{
			name: "b less than a",
			a:    20,
			b:    15,
			want: 15,
		},
		{
			name: "a equals b",
			a:    7,
			b:    7,
			want: 7,
		},
		{
			name: "negative numbers",
			a:    -5,
			b:    -10,
			want: -10,
		},
		{
			name: "zero and positive",
			a:    0,
			b:    5,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
