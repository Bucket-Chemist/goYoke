package statusline_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/statusline"
)

func TestNewStatusLineModel(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	// Zero value fields should be rendered without panicking.
	_ = m.View()
}

func TestStatusLineInit(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil command")
	}
}

func TestStatusLineViewContainsCostField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.SessionCost = 1.2345
	view := m.View()
	if !strings.Contains(view, "cost:") {
		t.Errorf("View() missing 'cost:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "1.2345") {
		t.Errorf("View() missing cost value '1.2345'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsTokenField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.TokenCount = 42000
	view := m.View()
	if !strings.Contains(view, "tokens:") {
		t.Errorf("View() missing 'tokens:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "42000") {
		t.Errorf("View() missing token count '42000'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsContextPercent(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ContextPercent = 75.5
	view := m.View()
	if !strings.Contains(view, "ctx:") {
		t.Errorf("View() missing 'ctx:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "75.5%") {
		t.Errorf("View() missing context percent '75.5%%'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsPermissionMode(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.PermissionMode = "delegate"
	view := m.View()
	if !strings.Contains(view, "perm:") {
		t.Errorf("View() missing 'perm:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "delegate") {
		t.Errorf("View() missing permission mode 'delegate'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsModelField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.ActiveModel = "claude-sonnet-4-6"
	view := m.View()
	if !strings.Contains(view, "model:") {
		t.Errorf("View() missing 'model:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "claude-sonnet-4-6") {
		t.Errorf("View() missing model name; got:\n%s", view)
	}
}

func TestStatusLineViewContainsProviderField(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.Provider = "anthropic"
	view := m.View()
	if !strings.Contains(view, "provider:") {
		t.Errorf("View() missing 'provider:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "anthropic") {
		t.Errorf("View() missing provider 'anthropic'; got:\n%s", view)
	}
}

func TestStatusLineViewContainsGitBranch(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.GitBranch = "tui-migration"
	view := m.View()
	if !strings.Contains(view, "branch:") {
		t.Errorf("View() missing 'branch:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "tui-migration") {
		t.Errorf("View() missing branch name; got:\n%s", view)
	}
}

func TestStatusLineViewContainsAuthStatus(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	m.AuthStatus = "authenticated"
	view := m.View()
	if !strings.Contains(view, "auth:") {
		t.Errorf("View() missing 'auth:' label; got:\n%s", view)
	}
	if !strings.Contains(view, "authenticated") {
		t.Errorf("View() missing auth status; got:\n%s", view)
	}
}

func TestStatusLineViewTwoRows(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	view := m.View()
	view = strings.TrimRight(view, "\n")
	lines := strings.Split(view, "\n")
	if len(lines) != 2 {
		t.Errorf("View() should produce 2 rows; got %d:\n%s", len(lines), view)
	}
}

func TestStatusLineUpdateWindowSizeMsg(t *testing.T) {
	m := statusline.NewStatusLineModel(80)
	newModel, cmd := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}
	// Should not panic after resize.
	_ = newModel.(statusline.StatusLineModel).View()
}

func TestStatusLineUpdateUnknownMsg(t *testing.T) {
	m := statusline.NewStatusLineModel(120)
	newModel, cmd := m.Update("unknown")
	if cmd != nil {
		t.Error("unknown message should return nil command")
	}
	if newModel == nil {
		t.Error("Update must always return non-nil model")
	}
}

func TestStatusLineSetWidth(t *testing.T) {
	m := statusline.NewStatusLineModel(80)
	m.SetWidth(200)
	// Should not panic.
	_ = m.View()
}

func TestStatusLineAllFieldsPopulated(t *testing.T) {
	m := statusline.NewStatusLineModel(160)
	m.SessionCost = 0.0042
	m.TokenCount = 8500
	m.ContextPercent = 12.3
	m.PermissionMode = "bypass"
	m.ActiveModel = "claude-opus-4-6"
	m.Provider = "anthropic"
	m.GitBranch = "main"
	m.AuthStatus = "ok"

	view := m.View()

	expected := []string{
		"cost:", "$0.0042",
		"tokens:", "8500",
		"ctx:", "12.3%",
		"perm:", "bypass",
		"model:", "claude-opus-4-6",
		"provider:", "anthropic",
		"branch:", "main",
		"auth:", "ok",
	}
	for _, want := range expected {
		if !strings.Contains(view, want) {
			t.Errorf("View() missing %q; got:\n%s", want, view)
		}
	}
}
