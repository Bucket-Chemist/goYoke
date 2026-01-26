# TUI-CLI-05: Session Management

> **Estimated Hours:** 1.5
> **Priority:** P2 - Enhancement
> **Dependencies:** TUI-CLI-01a
> **Phase:** 5 - Session

---

## Description

Session picker, history, and continuation support.

---

## Features

- List recent sessions
- Resume previous session (with `--resume` flag)
- Fork session (new ID, same context)
- Session naming/labeling
- Delete session option

---

## Tasks

### 1. Create Session Data Types

**File:** `internal/cli/session.go`

```go
package cli

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "time"
)

type Session struct {
    ID        string    `json:"id"`
    Name      string    `json:"name,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    LastUsed  time.Time `json:"last_used"`
    Cost      float64   `json:"cost"`
    ToolCalls int       `json:"tool_calls"`
}

// SessionManager handles session persistence and retrieval
type SessionManager struct {
    sessionsDir string
}

func NewSessionManager() (*SessionManager, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    sessionsDir := filepath.Join(homeDir, ".claude", "sessions")
    if err := os.MkdirAll(sessionsDir, 0755); err != nil {
        return nil, err
    }

    return &SessionManager{sessionsDir: sessionsDir}, nil
}
```

### 2. Implement Session Listing

```go
func (sm *SessionManager) ListSessions() ([]Session, error) {
    entries, err := os.ReadDir(sm.sessionsDir)
    if err != nil {
        return nil, err
    }

    var sessions []Session
    for _, entry := range entries {
        if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
            session, err := sm.loadSession(entry.Name())
            if err != nil {
                continue // Skip corrupt files
            }
            sessions = append(sessions, session)
        }
    }

    // Sort by LastUsed descending
    sort.Slice(sessions, func(i, j int) bool {
        return sessions[i].LastUsed.After(sessions[j].LastUsed)
    })

    return sessions, nil
}

func (sm *SessionManager) loadSession(filename string) (Session, error) {
    path := filepath.Join(sm.sessionsDir, filename)
    data, err := os.ReadFile(path)
    if err != nil {
        return Session{}, err
    }

    var session Session
    if err := json.Unmarshal(data, &session); err != nil {
        return Session{}, err
    }

    return session, nil
}
```

### 3. Implement Session Operations

```go
func (sm *SessionManager) ResumeSession(id string) (*ClaudeProcess, error) {
    cfg := Config{
        SessionID: id,
        // Resume flag handled internally by Claude CLI
    }

    process, err := NewClaudeProcess(cfg)
    if err != nil {
        return nil, err
    }

    // Update last used time
    sm.updateLastUsed(id)

    return process, process.Start()
}

func (sm *SessionManager) ForkSession(id string) (*ClaudeProcess, error) {
    // Create new session with new ID but inherit context
    // Claude CLI handles this via session context
    newID := generateSessionID()

    cfg := Config{
        SessionID: newID,
        // Fork would need special handling - may need to copy session data
    }

    process, err := NewClaudeProcess(cfg)
    if err != nil {
        return nil, err
    }

    return process, process.Start()
}

func (sm *SessionManager) DeleteSession(id string) error {
    path := filepath.Join(sm.sessionsDir, id+".json")
    return os.Remove(path)
}

func (sm *SessionManager) updateLastUsed(id string) error {
    session, err := sm.loadSession(id + ".json")
    if err != nil {
        return err
    }

    session.LastUsed = time.Now()
    return sm.saveSession(session)
}

func (sm *SessionManager) saveSession(session Session) error {
    path := filepath.Join(sm.sessionsDir, session.ID+".json")
    data, err := json.MarshalIndent(session, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

func generateSessionID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

### 4. Create Session Picker Modal

**File:** `internal/tui/session/picker.go`

```go
package session

import (
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "github.com/Bucket-Chemist/GOgent-Fortress/internal/cli"
)

type PickerModel struct {
    sessions []cli.Session
    cursor   int
    width    int
    height   int
    selected *cli.Session
}

func NewPickerModel(sessions []cli.Session) PickerModel {
    return PickerModel{
        sessions: sessions,
    }
}

func (m PickerModel) Init() tea.Cmd {
    return nil
}

func (m PickerModel) Update(msg tea.Msg) (PickerModel, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.sessions)-1 {
                m.cursor++
            }
        case "enter":
            if m.cursor < len(m.sessions) {
                m.selected = &m.sessions[m.cursor]
            }
        case "d":
            // Delete selected session
            if m.cursor < len(m.sessions) {
                return m, m.deleteSession(m.sessions[m.cursor].ID)
            }
        case "esc", "q":
            m.selected = nil
            return m, nil
        }
    }
    return m, nil
}

func (m PickerModel) deleteSession(id string) tea.Cmd {
    return func() tea.Msg {
        // Emit delete message
        return sessionDeletedMsg{id}
    }
}

type sessionDeletedMsg struct{ id string }

func (m PickerModel) View() string {
    var b strings.Builder

    b.WriteString(headerStyle.Render("Select Session"))
    b.WriteString("\n")
    b.WriteString(strings.Repeat("-", m.width-4))
    b.WriteString("\n\n")

    for i, session := range m.sessions {
        line := m.renderSession(session, i == m.cursor)
        b.WriteString(line)
        b.WriteString("\n")
    }

    b.WriteString("\n")
    b.WriteString(hintStyle.Render("[Enter] Resume  [d] Delete  [Esc] Cancel"))

    return modalStyle.Width(m.width).Height(m.height).Render(b.String())
}

func (m PickerModel) renderSession(session cli.Session, selected bool) string {
    // Format: ID (name) - Cost: $0.23 - 15 tools - 2h ago
    name := session.ID[:8]
    if session.Name != "" {
        name = session.Name
    }

    age := formatAge(session.LastUsed)

    line := fmt.Sprintf("%s - Cost: $%.2f - %d tools - %s",
        name,
        session.Cost,
        session.ToolCalls,
        age,
    )

    if selected {
        return selectedStyle.Render("> " + line)
    }
    return "  " + line
}

func (m PickerModel) Selected() *cli.Session {
    return m.selected
}

// Styles
var (
    modalStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2)

    headerStyle   = lipgloss.NewStyle().Bold(true)
    selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan"))
    hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

func formatAge(t time.Time) string {
    age := time.Since(t)
    if age < time.Hour {
        return fmt.Sprintf("%dm ago", int(age.Minutes()))
    }
    if age < 24*time.Hour {
        return fmt.Sprintf("%dh ago", int(age.Hours()))
    }
    return fmt.Sprintf("%dd ago", int(age.Hours()/24))
}
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `internal/cli/session.go` | Session management logic |
| `internal/cli/session_test.go` | Session operations tests |
| `internal/tui/session/picker.go` | Session picker modal |
| `internal/tui/session/picker_test.go` | Picker tests |

---

## Acceptance Criteria

- [ ] Sessions listed by recency (most recent first)
- [ ] Resume continues with full context
- [ ] Fork creates new session ID
- [ ] Delete removes session data
- [ ] Session info displayed (cost, tool calls, age)
- [ ] Keyboard navigation works
- [ ] Modal dismisses on Esc

---

## Test Strategy

### Unit Tests
- Session listing and sorting
- Resume/fork/delete operations
- Age formatting

### Integration Tests
- Session persistence across restarts
- Resume with actual Claude process
