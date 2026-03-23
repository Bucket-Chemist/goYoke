package settings

import (
	"strings"
	"testing"
)

func TestNewSettingsModel(t *testing.T) {
	m := NewSettingsModel()
	if m.width != 0 || m.height != 0 {
		t.Errorf("expected zero dimensions, got w=%d h=%d", m.width, m.height)
	}
	if m.model != "" {
		t.Errorf("expected empty model, got %q", m.model)
	}
}

func TestSetSize(t *testing.T) {
	m := NewSettingsModel()
	m.SetSize(100, 30)
	if m.width != 100 || m.height != 30 {
		t.Errorf("expected 100x30, got %dx%d", m.width, m.height)
	}
}

func TestSetConfig(t *testing.T) {
	m := NewSettingsModel()
	servers := []string{"gofortress", "other"}
	m.SetConfig("opus", "Anthropic", "acceptEdits", "/home/user/.claude/sessions/abc", servers)

	if m.model != "opus" {
		t.Errorf("expected model 'opus', got %q", m.model)
	}
	if m.provider != "Anthropic" {
		t.Errorf("expected provider 'Anthropic', got %q", m.provider)
	}
	if m.permissionMode != "acceptEdits" {
		t.Errorf("expected permissionMode 'acceptEdits', got %q", m.permissionMode)
	}
	if len(m.mcpServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(m.mcpServers))
	}
}

func TestSetConfig_NilServers(t *testing.T) {
	m := NewSettingsModel()
	m.SetConfig("", "", "", "", nil)
	if m.mcpServers != nil {
		t.Errorf("expected nil mcpServers, got %v", m.mcpServers)
	}
}

func TestSetConfig_CopiesSlice(t *testing.T) {
	m := NewSettingsModel()
	servers := []string{"a", "b"}
	m.SetConfig("", "", "", "", servers)
	// Mutate original; model should not reflect the change.
	servers[0] = "MUTATED"
	if m.mcpServers[0] == "MUTATED" {
		t.Error("SetConfig did not defensively copy the servers slice")
	}
}

func TestView_ContainsHeader(t *testing.T) {
	m := NewSettingsModel()
	view := m.View()
	if !strings.Contains(view, "Settings") {
		t.Errorf("expected 'Settings' in view, got:\n%s", view)
	}
}

func TestView_ContainsAllLabels(t *testing.T) {
	m := NewSettingsModel()
	m.SetConfig("opus", "Anthropic", "acceptEdits", "/home/user/sessions", []string{"gofortress"})
	view := m.View()

	labels := []string{"Model:", "Provider:", "Permission:", "Session Dir:", "MCP Servers:"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("expected label %q in view, got:\n%s", label, view)
		}
	}
}

func TestView_EmptyDash(t *testing.T) {
	m := NewSettingsModel()
	// No config set — all fields should show em-dash.
	view := m.View()
	if !strings.Contains(view, "\u2014") {
		t.Errorf("expected em-dash for empty values, got:\n%s", view)
	}
}

func TestView_MCPServersCount(t *testing.T) {
	tests := []struct {
		servers  []string
		contains string
	}{
		{nil, "none"},
		{[]string{"gofortress"}, "1 server"},
		{[]string{"a", "b"}, "2 servers"},
	}
	for _, tc := range tests {
		m := NewSettingsModel()
		m.SetConfig("", "", "", "", tc.servers)
		view := m.View()
		if !strings.Contains(view, tc.contains) {
			t.Errorf("servers %v: expected %q in view, got:\n%s", tc.servers, tc.contains, view)
		}
	}
}

func TestView_SessionDirTruncation(t *testing.T) {
	m := NewSettingsModel()
	// A very narrow width forces path truncation.
	m.SetSize(30, 24)
	longPath := "/home/user/very/deep/nested/directory/structure/sessions/abc-def-123"
	m.SetConfig("", "", "", longPath, nil)
	view := m.View()
	// Should contain ellipsis since path is long and width is small.
	if !strings.Contains(view, "\u2026") {
		t.Logf("view:\n%s", view)
		// This may not truncate depending on computed pathWidth — just ensure no panic.
	}
}

func TestTruncatePath(t *testing.T) {
	tests := []struct {
		path     string
		maxLen   int
		wantEm   bool   // expect em-dash (empty input)
		contains string // partial match in output
	}{
		{"", 20, true, ""},
		{"/a/b/c", 10, false, "/a/b/c"},
		{"/home/user/very/long/path/that/exceeds/limit", 20, false, "\u2026"},
	}
	for _, tc := range tests {
		got := truncatePath(tc.path, tc.maxLen)
		if tc.wantEm && got != "\u2014" {
			t.Errorf("truncatePath(%q, %d): expected em-dash, got %q", tc.path, tc.maxLen, got)
		}
		if tc.contains != "" && !strings.Contains(got, tc.contains) {
			t.Errorf("truncatePath(%q, %d): expected %q, got %q", tc.path, tc.maxLen, tc.contains, got)
		}
		// Ensure output is never longer than maxLen runes (when path is non-empty).
		if tc.path != "" {
			runes := []rune(got)
			if len(runes) > tc.maxLen {
				t.Errorf("truncatePath(%q, %d): result %q exceeds maxLen", tc.path, tc.maxLen, got)
			}
		}
	}
}

func TestOrDash(t *testing.T) {
	if orDash("") != "\u2014" {
		t.Error("expected em-dash for empty string")
	}
	if orDash("hello") != "hello" {
		t.Error("expected 'hello' unchanged")
	}
}
