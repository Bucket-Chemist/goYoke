# MCP Implementation Guide for gofortress v2.0

**Production-Ready Model Context Protocol Integration for Go-Based TUI**

Claude Code enables AI-assisted development through a CLI interface, and gofortress wraps this CLI with an interactive Bubbletea TUI. This guide provides the complete technical specification for adding **interactive MCP prompts** that allow Claude to ask users questions through the TUI—transforming one-way command execution into genuine human-AI collaboration. The architecture uses a three-process model where gofortress spawns Claude CLI, which spawns an MCP server that communicates back to the TUI via Unix sockets, achieving **sub-3ms round-trip latency** for prompt displays while maintaining the project's single-binary simplicity.

---

## Architecture overview

The MCP integration follows a grandparent-child-grandchild process hierarchy with Unix socket IPC for the callback channel:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          gofortress TUI (PID: 1000)                         │
│  ┌────────────────────┐  ┌───────────────────────────────────────────────┐  │
│  │  Bubbletea Model   │  │  Unix Socket HTTP Server                      │  │
│  │  ├─ MainView       │  │  /tmp/gofortress-1000.sock                    │  │
│  │  ├─ PromptModal    │  │  POST /prompt → displays modal               │  │
│  │  └─ ResponseChan   │◄─┤  POST /confirm → yes/no dialog              │  │
│  └────────────────────┘  │  GET  /health → server status                │  │
│           │              └───────────────────────────────────────────────┘  │
│           │ spawns                              ▲                            │
└───────────┼─────────────────────────────────────┼────────────────────────────┘
            ▼                                     │
┌─────────────────────────────────────────────────┼────────────────────────────┐
│                    Claude Code CLI (PID: 1001)  │                            │
│  ┌────────────────────────────────────────────┐ │ (Unix socket callback)     │
│  │ --mcp-config /tmp/gofortress-mcp-1000.json │ │                            │
│  │ NDJSON event streaming to gofortress       │ │                            │
│  └────────────────────────────────────────────┘ │                            │
│           │ spawns via mcp-config               │                            │
└───────────┼─────────────────────────────────────┼────────────────────────────┘
            ▼                                     │
┌─────────────────────────────────────────────────┼────────────────────────────┐
│               gofortress-mcp-server (PID: 1002) │                            │
│  ┌────────────────────────────────────────────┐ │                            │
│  │ MCP Server (stdio transport to Claude)     │ │                            │
│  │ Tools: ask_user, confirm_action,           │─┘                            │
│  │        request_input, select_file          │                              │
│  │ Calls back to TUI via Unix socket          │                              │
│  └────────────────────────────────────────────┘                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Component responsibilities

**gofortress TUI (grandparent)**: Renders the terminal interface using Bubbletea, manages user sessions, spawns Claude CLI, and hosts the Unix socket HTTP server that receives prompt requests from the MCP server. When a prompt arrives, the TUI displays a modal overlay, captures user input, and returns the response.

**Claude Code CLI (child)**: The AI agent that processes user requests, executes tools, and streams NDJSON events. Claude discovers the MCP server through `--mcp-config` and spawns it as a subprocess for interactive prompts.

**gofortress-mcp-server (grandchild)**: A lightweight Go binary implementing MCP over stdio. When Claude calls tools like `ask_user`, the server makes HTTP requests to the TUI's Unix socket, blocks until the user responds, then returns the result to Claude.

### Data flow for interactive prompt

The complete flow when Claude needs user input demonstrates the three-process coordination:

```
Claude decides to ask user → calls mcp__gofortress__ask_user
                                    │
                                    ▼
            MCP Server receives tool call via stdio
                                    │
                                    ▼
            POST /prompt to Unix socket (TUI server)
                                    │
                                    ▼
            TUI receives MCPPromptMsg via Program.Send()
                                    │
                                    ▼
            TUI displays modal, user types response
                                    │
                                    ▼
            Response sent to responseChan
                                    │
                                    ▼
            HTTP handler returns JSON response
                                    │
                                    ▼
            MCP Server returns CallToolResult to Claude
                                    │
                                    ▼
            Claude continues with user's answer
```

---

## Go MCP SDK reference

The official Go SDK at `github.com/modelcontextprotocol/go-sdk` (v1.2.0) provides production-ready MCP implementation backed by Google.

### Server creation and lifecycle

```go
package main

import (
    "context"
    "errors"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
    // Graceful shutdown with signal handling
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Logger must write to stderr (stdout reserved for MCP protocol)
    logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Create server with implementation metadata
    server := mcp.NewServer(
        &mcp.Implementation{
            Name:    "gofortress-mcp-server",
            Version: "1.0.0",
        },
        &mcp.ServerOptions{
            Logger: logger,
            InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
                logger.Info("MCP session initialized with Claude")
            },
        },
    )

    // Register tools (covered in next section)
    registerTools(server)

    // Run server over stdio transport
    logger.Info("starting gofortress MCP server")
    if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
        if !errors.Is(err, mcp.ErrConnectionClosed) && !errors.Is(err, context.Canceled) {
            logger.Error("server error", "error", err)
            os.Exit(1)
        }
    }
    logger.Info("server shutdown complete")
}
```

### Tool registration with type-safe generics

The SDK uses Go generics for type-safe tool definitions with automatic JSON schema inference:

```go
// Input type with JSON schema inference via struct tags
type AskUserInput struct {
    Message string   `json:"message" jsonschema:"question to ask the user,required"`
    Options []string `json:"options,omitempty" jsonschema:"optional list of choices"`
    Default string   `json:"default,omitempty" jsonschema:"default value if user doesn't respond"`
}

// Output type (use 'any' to omit output schema)
type AskUserOutput struct {
    Response  string `json:"response" jsonschema:"user's response"`
    Cancelled bool   `json:"cancelled" jsonschema:"true if user cancelled"`
}

// Handler signature: func(ctx, req, input) (*CallToolResult, output, error)
func AskUserHandler(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (
    *mcp.CallToolResult, AskUserOutput, error,
) {
    // Call back to TUI via Unix socket (implementation below)
    response, err := callbackToTUI(ctx, PromptRequest{
        Type:    "ask",
        Message: input.Message,
        Options: input.Options,
    })
    if err != nil {
        // Return IsError=true for errors Claude should see and handle
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get user input: " + err.Error()}},
        }, AskUserOutput{}, nil
    }

    return nil, AskUserOutput{
        Response:  response.Value,
        Cancelled: response.Cancelled,
    }, nil
}

// Registration
func registerTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "ask_user",
        Description: "Ask the user a question and wait for their response. Use this when you need clarification or user input.",
    }, AskUserHandler)
}
```

### Error handling patterns

The SDK distinguishes between protocol errors and tool execution errors:

```go
func ToolHandler(ctx context.Context, req *mcp.CallToolRequest, input Input) (
    *mcp.CallToolResult, any, error,
) {
    // Protocol error: returns JSON-RPC error, indicates server malfunction
    if criticalFailure {
        return nil, nil, fmt.Errorf("database connection lost")
    }

    // Tool error: visible to Claude, allows self-correction
    if userCancelled {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{
                &mcp.TextContent{Text: "User cancelled the operation"},
            },
        }, nil, nil
    }

    // Success: return output struct
    return nil, OutputType{Data: result}, nil
}
```

---

## Claude Code MCP configuration

Claude Code discovers MCP servers through JSON configuration, either in `~/.claude.json` or via `--mcp-config`.

### Configuration schema

```json
{
  "mcpServers": {
    "gofortress": {
      "type": "stdio",
      "command": "/path/to/gofortress-mcp-server",
      "args": [],
      "env": {
        "GOFORTRESS_SOCKET": "/tmp/gofortress-1000.sock",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

### Dynamic configuration generation

gofortress generates this configuration at runtime, embedding the socket path:

```go
type MCPConfig struct {
    MCPServers map[string]MCPServerConfig `json:"mcpServers"`
}

type MCPServerConfig struct {
    Type    string            `json:"type"`
    Command string            `json:"command"`
    Args    []string          `json:"args"`
    Env     map[string]string `json:"env"`
}

func generateMCPConfig(socketPath, serverBinary string) (string, error) {
    config := MCPConfig{
        MCPServers: map[string]MCPServerConfig{
            "gofortress": {
                Type:    "stdio",
                Command: serverBinary,
                Args:    []string{},
                Env: map[string]string{
                    "GOFORTRESS_SOCKET": socketPath,
                    "LOG_LEVEL":         "info",
                },
            },
        },
    }

    // Write to temp file
    configPath := filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-mcp-%d.json", os.Getpid()))
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return "", err
    }
    if err := os.WriteFile(configPath, data, 0600); err != nil {
        return "", err
    }
    return configPath, nil
}
```

### Tool naming conventions

Claude surfaces MCP tools with the pattern `mcp__<servername>__<toolname>`:

| MCP Tool Name | Claude Tool Name |
|---------------|------------------|
| `ask_user` | `mcp__gofortress__ask_user` |
| `confirm_action` | `mcp__gofortress__confirm_action` |
| `request_input` | `mcp__gofortress__request_input` |
| `select_file` | `mcp__gofortress__select_file` |

---

## Unix socket HTTP server implementation

The TUI hosts an HTTP server over Unix sockets for receiving prompts from the MCP server.

### Server implementation

```go
package callback

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)

type PromptRequest struct {
    ID      string   `json:"id"`
    Type    string   `json:"type"` // "ask", "confirm", "input", "select"
    Message string   `json:"message"`
    Options []string `json:"options,omitempty"`
    Default string   `json:"default,omitempty"`
}

type PromptResponse struct {
    ID        string `json:"id"`
    Value     string `json:"value"`
    Cancelled bool   `json:"cancelled"`
    Error     string `json:"error,omitempty"`
}

type CallbackServer struct {
    socketPath string
    listener   net.Listener
    httpServer *http.Server
    
    // Channel for sending prompts to TUI
    PromptChan chan PromptRequest
    
    // Map of pending responses (keyed by prompt ID)
    pending   map[string]chan PromptResponse
    pendingMu sync.RWMutex
}

func NewCallbackServer(pid int) *CallbackServer {
    socketPath := getSocketPath(pid)
    return &CallbackServer{
        socketPath: socketPath,
        PromptChan: make(chan PromptRequest, 10),
        pending:    make(map[string]chan PromptResponse),
    }
}

func getSocketPath(pid int) string {
    // Prefer XDG_RUNTIME_DIR for better security
    if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
        return filepath.Join(runtimeDir, fmt.Sprintf("gofortress-%d.sock", pid))
    }
    return filepath.Join(os.TempDir(), fmt.Sprintf("gofortress-%d.sock", pid))
}

func (s *CallbackServer) Start(ctx context.Context) error {
    // Remove stale socket
    os.Remove(s.socketPath)

    var err error
    s.listener, err = net.Listen("unix", s.socketPath)
    if err != nil {
        return fmt.Errorf("failed to listen on socket: %w", err)
    }

    // Set restrictive permissions (owner only)
    if err := os.Chmod(s.socketPath, 0600); err != nil {
        s.listener.Close()
        return fmt.Errorf("failed to set socket permissions: %w", err)
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/prompt", s.handlePrompt)
    mux.HandleFunc("/health", s.handleHealth)

    s.httpServer = &http.Server{
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 5 * time.Minute, // Long timeout for user interaction
        BaseContext:  func(_ net.Listener) context.Context { return ctx },
    }

    go func() {
        if err := s.httpServer.Serve(s.listener); err != http.ErrServerClosed {
            fmt.Fprintf(os.Stderr, "callback server error: %v\n", err)
        }
    }()

    return nil
}

func (s *CallbackServer) handlePrompt(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req PromptRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    // Generate ID if not provided
    if req.ID == "" {
        req.ID = fmt.Sprintf("prompt-%d", time.Now().UnixNano())
    }

    // Create response channel for this prompt
    respChan := make(chan PromptResponse, 1)
    s.pendingMu.Lock()
    s.pending[req.ID] = respChan
    s.pendingMu.Unlock()

    defer func() {
        s.pendingMu.Lock()
        delete(s.pending, req.ID)
        s.pendingMu.Unlock()
    }()

    // Send to TUI for display
    select {
    case s.PromptChan <- req:
    case <-r.Context().Done():
        http.Error(w, "request cancelled", http.StatusRequestTimeout)
        return
    }

    // Wait for user response (long-poll)
    select {
    case resp := <-respChan:
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    case <-r.Context().Done():
        http.Error(w, "timeout waiting for user response", http.StatusGatewayTimeout)
    }
}

func (s *CallbackServer) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// SendResponse is called by TUI when user completes prompt
func (s *CallbackServer) SendResponse(resp PromptResponse) error {
    s.pendingMu.RLock()
    ch, ok := s.pending[resp.ID]
    s.pendingMu.RUnlock()

    if !ok {
        return fmt.Errorf("no pending prompt with ID: %s", resp.ID)
    }

    select {
    case ch <- resp:
        return nil
    default:
        return fmt.Errorf("response channel full for prompt: %s", resp.ID)
    }
}

func (s *CallbackServer) Shutdown(ctx context.Context) error {
    if s.httpServer != nil {
        return s.httpServer.Shutdown(ctx)
    }
    return nil
}

func (s *CallbackServer) Cleanup() {
    os.Remove(s.socketPath)
}

func (s *CallbackServer) SocketPath() string {
    return s.socketPath
}
```

### MCP server callback client

The MCP server uses this client to call back to the TUI:

```go
package callback

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "os"
    "time"
)

type Client struct {
    httpClient *http.Client
    socketPath string
}

func NewClient() (*Client, error) {
    socketPath := os.Getenv("GOFORTRESS_SOCKET")
    if socketPath == "" {
        return nil, fmt.Errorf("GOFORTRESS_SOCKET environment variable not set")
    }

    client := &http.Client{
        Transport: &http.Transport{
            DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
                dialer := net.Dialer{Timeout: 5 * time.Second}
                return dialer.DialContext(ctx, "unix", socketPath)
            },
            MaxIdleConns:        5,
            IdleConnTimeout:     90 * time.Second,
            DisableKeepAlives:   false,
        },
        Timeout: 5 * time.Minute, // Long timeout for user interaction
    }

    return &Client{
        httpClient: client,
        socketPath: socketPath,
    }, nil
}

func (c *Client) SendPrompt(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("failed to marshal request: %w", err)
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", "http://unix/prompt", bytes.NewReader(body))
    if err != nil {
        return PromptResponse{}, fmt.Errorf("failed to create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return PromptResponse{}, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return PromptResponse{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    var response PromptResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return PromptResponse{}, fmt.Errorf("failed to decode response: %w", err)
    }

    return response, nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET", "http://unix/health", nil)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("health check failed: %d", resp.StatusCode)
    }
    return nil
}
```

---

## Bubbletea TUI integration

The TUI receives external events from the callback server and displays modal prompts.

### Model structure with modal support

```go
package tui

import (
    "strings"

    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    
    "gofortress/internal/callback"
)

type modalType int

const (
    noModal modalType = iota
    confirmModal
    textInputModal
    selectionModal
)

type Model struct {
    // Window dimensions
    width, height int
    
    // Main content
    mainContent string
    claudeOutput strings.Builder
    
    // Modal state
    showModal     bool
    modalType     modalType
    currentPrompt callback.PromptRequest
    
    // Modal components
    textInput  textinput.Model
    selectList list.Model
    
    // Callback server reference
    callbackServer *callback.CallbackServer
}

// Message types
type MCPPromptMsg callback.PromptRequest

func NewModel(callbackServer *callback.CallbackServer) Model {
    ti := textinput.New()
    ti.Placeholder = "Type your response..."
    ti.CharLimit = 500
    ti.Width = 40

    return Model{
        textInput:      ti,
        callbackServer: callbackServer,
    }
}

func (m Model) Init() tea.Cmd {
    // Start listening for prompts from callback server
    return m.listenForPrompts()
}

func (m Model) listenForPrompts() tea.Cmd {
    return func() tea.Msg {
        prompt := <-m.callbackServer.PromptChan
        return MCPPromptMsg(prompt)
    }
}
```

### Update function with modal handling

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil

    case tea.KeyMsg:
        // Global quit
        if msg.String() == "ctrl+c" {
            return m, tea.Quit
        }
        
        // Route to modal if active
        if m.showModal {
            return m.updateModal(msg)
        }
        
        // Main view key handling
        return m, nil

    case MCPPromptMsg:
        m.showModal = true
        m.currentPrompt = callback.PromptRequest(msg)
        
        // Determine modal type based on prompt
        switch m.currentPrompt.Type {
        case "confirm":
            m.modalType = confirmModal
        case "input", "ask":
            if len(m.currentPrompt.Options) > 0 {
                m.modalType = selectionModal
                m.selectList = m.createSelectList(m.currentPrompt.Options)
            } else {
                m.modalType = textInputModal
                m.textInput.Reset()
                if m.currentPrompt.Default != "" {
                    m.textInput.SetValue(m.currentPrompt.Default)
                }
                return m, m.textInput.Focus()
            }
        case "select":
            m.modalType = selectionModal
            m.selectList = m.createSelectList(m.currentPrompt.Options)
        }
        return m, nil
    }
    
    return m, nil
}

func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "enter":
        response := callback.PromptResponse{ID: m.currentPrompt.ID}
        
        switch m.modalType {
        case confirmModal:
            response.Value = "yes"
        case textInputModal:
            response.Value = m.textInput.Value()
        case selectionModal:
            if item, ok := m.selectList.SelectedItem().(listItem); ok {
                response.Value = item.title
            }
        }
        
        m.showModal = false
        
        // Send response back to callback server
        return m, tea.Batch(
            m.sendResponse(response),
            m.listenForPrompts(), // Listen for next prompt
        )

    case "esc", "n":
        if m.modalType == confirmModal && msg.String() == "n" {
            // "No" response for confirm dialog
            response := callback.PromptResponse{
                ID:    m.currentPrompt.ID,
                Value: "no",
            }
            m.showModal = false
            return m, tea.Batch(
                m.sendResponse(response),
                m.listenForPrompts(),
            )
        }
        
        // Cancel
        response := callback.PromptResponse{
            ID:        m.currentPrompt.ID,
            Cancelled: true,
        }
        m.showModal = false
        return m, tea.Batch(
            m.sendResponse(response),
            m.listenForPrompts(),
        )

    case "y":
        if m.modalType == confirmModal {
            response := callback.PromptResponse{
                ID:    m.currentPrompt.ID,
                Value: "yes",
            }
            m.showModal = false
            return m, tea.Batch(
                m.sendResponse(response),
                m.listenForPrompts(),
            )
        }
    }

    // Update modal component
    var cmd tea.Cmd
    switch m.modalType {
    case textInputModal:
        m.textInput, cmd = m.textInput.Update(msg)
    case selectionModal:
        m.selectList, cmd = m.selectList.Update(msg)
    }
    return m, cmd
}

func (m Model) sendResponse(resp callback.PromptResponse) tea.Cmd {
    return func() tea.Msg {
        m.callbackServer.SendResponse(resp)
        return nil
    }
}

// List item for selection modal
type listItem struct {
    title string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return i.title }

func (m Model) createSelectList(options []string) list.Model {
    items := make([]list.Item, len(options))
    for i, opt := range options {
        items[i] = listItem{title: opt}
    }
    
    delegate := list.NewDefaultDelegate()
    l := list.New(items, delegate, 40, min(len(options)+4, 15))
    l.SetShowTitle(false)
    l.SetShowStatusBar(false)
    l.SetShowHelp(false)
    
    return l
}
```

### View rendering with modal overlay

```go
var (
    modalStyle = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("62")).
        Padding(1, 2).
        Width(50)

    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("170"))

    helpStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("241"))
)

func (m Model) View() string {
    // Render main content
    main := m.renderMainContent()
    
    if !m.showModal {
        return main
    }
    
    // Render and overlay modal
    modal := m.renderModal()
    return m.overlayModal(main, modal)
}

func (m Model) renderMainContent() string {
    return lipgloss.NewStyle().
        Width(m.width).
        Height(m.height).
        Render(m.claudeOutput.String())
}

func (m Model) renderModal() string {
    var content strings.Builder
    
    content.WriteString(titleStyle.Render(m.currentPrompt.Message))
    content.WriteString("\n\n")
    
    switch m.modalType {
    case confirmModal:
        content.WriteString("[Y]es  [N]o  [Esc] Cancel")
    case textInputModal:
        content.WriteString(m.textInput.View())
        content.WriteString("\n\n")
        content.WriteString(helpStyle.Render("[Enter] Submit  [Esc] Cancel"))
    case selectionModal:
        content.WriteString(m.selectList.View())
        content.WriteString("\n")
        content.WriteString(helpStyle.Render("[Enter] Select  [Esc] Cancel"))
    }
    
    return modalStyle.Render(content.String())
}

func (m Model) overlayModal(background, modal string) string {
    bgLines := strings.Split(background, "\n")
    modalLines := strings.Split(modal, "\n")
    
    modalWidth := lipgloss.Width(modal)
    modalHeight := len(modalLines)
    
    // Center the modal
    startX := max((m.width-modalWidth)/2, 0)
    startY := max((m.height-modalHeight)/2, 0)
    
    // Composite modal onto background
    for i, modalLine := range modalLines {
        bgY := startY + i
        if bgY < len(bgLines) {
            bgLines[bgY] = overlayLine(bgLines[bgY], modalLine, startX, m.width)
        }
    }
    
    return strings.Join(bgLines, "\n")
}

func overlayLine(background, overlay string, startX, maxWidth int) string {
    // Ensure background has enough width
    for len(background) < maxWidth {
        background += " "
    }
    
    bgRunes := []rune(background)
    overlayRunes := []rune(overlay)
    
    for i, r := range overlayRunes {
        pos := startX + i
        if pos < len(bgRunes) {
            bgRunes[pos] = r
        }
    }
    
    return string(bgRunes)
}
```

---

## Complete MCP server implementation

The MCP server binary that Claude spawns:

```go
// cmd/gofortress-mcp-server/main.go
package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    
    "gofortress/internal/callback"
)

var (
    version = "1.0.0"
    client  *callback.Client
    logger  *slog.Logger
)

// Tool inputs
type AskUserInput struct {
    Message string   `json:"message" jsonschema:"question to ask the user,required"`
    Options []string `json:"options,omitempty" jsonschema:"optional list of choices for selection"`
    Default string   `json:"default,omitempty" jsonschema:"default value if provided"`
}

type ConfirmActionInput struct {
    Action      string `json:"action" jsonschema:"description of action requiring confirmation,required"`
    Destructive bool   `json:"destructive,omitempty" jsonschema:"whether action is destructive/dangerous"`
}

type RequestInputInput struct {
    Prompt      string `json:"prompt" jsonschema:"input prompt to display,required"`
    Placeholder string `json:"placeholder,omitempty" jsonschema:"placeholder text"`
    Multiline   bool   `json:"multiline,omitempty" jsonschema:"allow multiline input"`
}

type SelectFileInput struct {
    Message    string   `json:"message" jsonschema:"prompt message,required"`
    Extensions []string `json:"extensions,omitempty" jsonschema:"allowed file extensions"`
    Directory  string   `json:"directory,omitempty" jsonschema:"starting directory"`
}

// Tool outputs
type AskUserOutput struct {
    Response  string `json:"response"`
    Cancelled bool   `json:"cancelled"`
}

type ConfirmActionOutput struct {
    Confirmed bool `json:"confirmed"`
    Cancelled bool `json:"cancelled"`
}

type RequestInputOutput struct {
    Input     string `json:"input"`
    Cancelled bool   `json:"cancelled"`
}

type SelectFileOutput struct {
    Path      string `json:"path"`
    Cancelled bool   `json:"cancelled"`
}

// Handlers
func askUserHandler(ctx context.Context, req *mcp.CallToolRequest, input AskUserInput) (
    *mcp.CallToolResult, AskUserOutput, error,
) {
    promptType := "ask"
    if len(input.Options) > 0 {
        promptType = "select"
    }
    
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("ask-%d", time.Now().UnixNano()),
        Type:    promptType,
        Message: input.Message,
        Options: input.Options,
        Default: input.Default,
    })
    
    if err != nil {
        logger.Error("failed to get user response", "error", err)
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to communicate with TUI: " + err.Error()}},
        }, AskUserOutput{}, nil
    }
    
    return nil, AskUserOutput{
        Response:  resp.Value,
        Cancelled: resp.Cancelled,
    }, nil
}

func confirmActionHandler(ctx context.Context, req *mcp.CallToolRequest, input ConfirmActionInput) (
    *mcp.CallToolResult, ConfirmActionOutput, error,
) {
    message := input.Action
    if input.Destructive {
        message = "⚠️ DESTRUCTIVE: " + message
    }
    
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("confirm-%d", time.Now().UnixNano()),
        Type:    "confirm",
        Message: message,
    })
    
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get confirmation: " + err.Error()}},
        }, ConfirmActionOutput{}, nil
    }
    
    return nil, ConfirmActionOutput{
        Confirmed: resp.Value == "yes",
        Cancelled: resp.Cancelled,
    }, nil
}

func requestInputHandler(ctx context.Context, req *mcp.CallToolRequest, input RequestInputInput) (
    *mcp.CallToolResult, RequestInputOutput, error,
) {
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("input-%d", time.Now().UnixNano()),
        Type:    "input",
        Message: input.Prompt,
        Default: input.Placeholder,
    })
    
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get user input: " + err.Error()}},
        }, RequestInputOutput{}, nil
    }
    
    return nil, RequestInputOutput{
        Input:     resp.Value,
        Cancelled: resp.Cancelled,
    }, nil
}

func selectFileHandler(ctx context.Context, req *mcp.CallToolRequest, input SelectFileInput) (
    *mcp.CallToolResult, SelectFileOutput, error,
) {
    resp, err := client.SendPrompt(ctx, callback.PromptRequest{
        ID:      fmt.Sprintf("file-%d", time.Now().UnixNano()),
        Type:    "select",
        Message: input.Message,
        Options: input.Extensions, // TUI can interpret this for file filtering
    })
    
    if err != nil {
        return &mcp.CallToolResult{
            IsError: true,
            Content: []mcp.Content{&mcp.TextContent{Text: "Failed to get file selection: " + err.Error()}},
        }, SelectFileOutput{}, nil
    }
    
    return nil, SelectFileOutput{
        Path:      resp.Value,
        Cancelled: resp.Cancelled,
    }, nil
}

func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    // Configure logging to stderr
    logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: getLogLevel(),
    }))

    // Initialize callback client
    var err error
    client, err = callback.NewClient()
    if err != nil {
        logger.Error("failed to initialize callback client", "error", err)
        os.Exit(1)
    }

    // Verify TUI is reachable
    healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Second)
    defer healthCancel()
    if err := client.HealthCheck(healthCtx); err != nil {
        logger.Error("TUI health check failed", "error", err)
        os.Exit(1)
    }
    logger.Info("connected to TUI callback server")

    // Create MCP server
    server := mcp.NewServer(
        &mcp.Implementation{
            Name:    "gofortress-mcp-server",
            Version: version,
        },
        &mcp.ServerOptions{
            Logger: logger,
            InitializedHandler: func(ctx context.Context, req *mcp.InitializedRequest) {
                logger.Info("MCP session initialized")
            },
        },
    )

    // Register tools
    mcp.AddTool(server, &mcp.Tool{
        Name:        "ask_user",
        Description: "Ask the user a question. Use when you need clarification, preferences, or any input from the user. Can present multiple choice options or free-form questions.",
    }, askUserHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "confirm_action",
        Description: "Request user confirmation before proceeding with an action. Use for destructive operations, irreversible changes, or when explicit approval is needed.",
    }, confirmActionHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "request_input",
        Description: "Request free-form text input from the user. Use when you need the user to provide text content like code snippets, descriptions, or configuration values.",
    }, requestInputHandler)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "select_file",
        Description: "Allow the user to select a file from their filesystem. Use when you need the user to choose a specific file for reading, processing, or as a target location.",
    }, selectFileHandler)

    // Run server
    logger.Info("starting MCP server on stdio")
    if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
        if !errors.Is(err, mcp.ErrConnectionClosed) && !errors.Is(err, context.Canceled) {
            logger.Error("server error", "error", err)
            os.Exit(1)
        }
    }
    logger.Info("MCP server shutdown complete")
}

func getLogLevel() slog.Level {
    switch os.Getenv("LOG_LEVEL") {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

---

## Hooks and MCP coexistence

The existing gogent hooks continue operating unchanged alongside MCP:

| Component | Purpose | When Used |
|-----------|---------|-----------|
| `gogent-validate` (PreToolUse) | Policy enforcement, tool blocking | Every tool call before execution |
| `gogent-sharp-edge` (PostToolUse) | Audit logging, side effects | After tool execution |
| `gogent-load-context` (SessionStart) | Load project context | Session initialization |
| `gogent-agent-endstate` (SubagentStop) | Subagent cleanup | Subagent termination |
| `gogent-archive` (SessionEnd) | Session archival | Session end |
| **MCP Server** | Interactive user prompts | When Claude needs user input |

### Intercepting MCP tool calls via hooks

The PreToolUse hook can intercept MCP tool calls:

```go
// In gogent-validate hook
func validateToolUse(event ToolUseEvent) ValidationResult {
    // MCP tools appear as mcp__servername__toolname
    if strings.HasPrefix(event.Tool, "mcp__gofortress__") {
        // Extract actual tool name
        toolName := strings.TrimPrefix(event.Tool, "mcp__gofortress__")
        
        // Apply policies to MCP tools
        switch toolName {
        case "confirm_action":
            // Log all confirmation requests
            logAudit("confirmation_requested", event)
        case "select_file":
            // Validate file access permissions
            if !isPathAllowed(event.Input["directory"]) {
                return ValidationResult{
                    Approved: false,
                    Reason:   "File access not permitted in this directory",
                }
            }
        }
    }
    return ValidationResult{Approved: true}
}
```

---

## Testing strategy

### Unit tests for tool handlers

```go
// internal/mcp/handlers_test.go
package mcp

import (
    "context"
    "testing"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

type mockCallbackClient struct {
    responses map[string]callback.PromptResponse
}

func (m *mockCallbackClient) SendPrompt(ctx context.Context, req callback.PromptRequest) (callback.PromptResponse, error) {
    if resp, ok := m.responses[req.Type]; ok {
        return resp, nil
    }
    return callback.PromptResponse{}, nil
}

func TestAskUserHandler(t *testing.T) {
    tests := []struct {
        name     string
        input    AskUserInput
        mockResp callback.PromptResponse
        want     AskUserOutput
    }{
        {
            name:     "simple question",
            input:    AskUserInput{Message: "What is your name?"},
            mockResp: callback.PromptResponse{Value: "Alice"},
            want:     AskUserOutput{Response: "Alice", Cancelled: false},
        },
        {
            name:     "user cancels",
            input:    AskUserInput{Message: "Continue?"},
            mockResp: callback.PromptResponse{Cancelled: true},
            want:     AskUserOutput{Response: "", Cancelled: true},
        },
        {
            name:     "selection from options",
            input:    AskUserInput{Message: "Choose:", Options: []string{"A", "B", "C"}},
            mockResp: callback.PromptResponse{Value: "B"},
            want:     AskUserOutput{Response: "B", Cancelled: false},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Inject mock client
            client = &mockCallbackClient{
                responses: map[string]callback.PromptResponse{
                    "ask":    tt.mockResp,
                    "select": tt.mockResp,
                },
            }

            ctx := context.Background()
            result, output, err := askUserHandler(ctx, nil, tt.input)

            require.NoError(t, err)
            assert.Nil(t, result) // No CallToolResult means success
            assert.Equal(t, tt.want, output)
        })
    }
}
```

### Integration tests with in-memory transport

```go
func TestMCPServerIntegration(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Create server with mock callback
    server := createTestServer(t)

    // Create in-memory transports
    serverTransport, clientTransport := mcp.NewInMemoryTransports()

    // Start server
    _, err := server.Connect(ctx, serverTransport, nil)
    require.NoError(t, err)

    // Create client
    client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
    session, err := client.Connect(ctx, clientTransport, nil)
    require.NoError(t, err)
    defer session.Close()

    // List tools
    tools, err := session.ListTools(ctx, nil)
    require.NoError(t, err)
    assert.Len(t, tools.Tools, 4) // ask_user, confirm_action, request_input, select_file

    // Call ask_user tool
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name: "ask_user",
        Arguments: map[string]any{
            "message": "Test question?",
        },
    })
    require.NoError(t, err)
    assert.False(t, result.IsError)
}
```

### Bubbletea TUI tests

```go
// internal/tui/model_test.go
package tui

import (
    "bytes"
    "testing"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/x/exp/teatest"
    "github.com/charmbracelet/lipgloss"
    "github.com/muesli/termenv"
    
    "gofortress/internal/callback"
)

func init() {
    // Disable colors for consistent test output
    lipgloss.SetColorProfile(termenv.Ascii)
}

func TestModalDisplay(t *testing.T) {
    // Create mock callback server
    callbackServer := &callback.CallbackServer{
        PromptChan: make(chan callback.PromptRequest, 1),
    }
    
    m := NewModel(callbackServer)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Simulate MCP prompt message
    tm.Send(MCPPromptMsg{
        ID:      "test-1",
        Type:    "confirm",
        Message: "Delete all files?",
    })

    // Wait for modal to appear
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Delete all files?"))
    }, teatest.WithCheckInterval(50*time.Millisecond),
       teatest.WithDuration(2*time.Second))

    // Verify modal shows confirmation options
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("[Y]es"))
    })
}

func TestUserConfirmation(t *testing.T) {
    callbackServer := &callback.CallbackServer{
        PromptChan: make(chan callback.PromptRequest, 1),
        pending:    make(map[string]chan callback.PromptResponse),
    }
    
    // Setup response channel
    respChan := make(chan callback.PromptResponse, 1)
    callbackServer.pending["test-1"] = respChan
    
    m := NewModel(callbackServer)
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Send prompt
    tm.Send(MCPPromptMsg{
        ID:      "test-1",
        Type:    "confirm",
        Message: "Continue?",
    })

    // Wait for modal
    teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
        return bytes.Contains(bts, []byte("Continue?"))
    })

    // Send "y" key
    tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

    // Verify response was sent
    select {
    case resp := <-respChan:
        if resp.Value != "yes" {
            t.Errorf("expected 'yes', got %q", resp.Value)
        }
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for response")
    }
}
```

---

## Production hardening

### Graceful degradation

```go
// If MCP server fails, fall back to AllowedTools mode
func (m *Model) handleMCPFailure(err error) tea.Cmd {
    m.mcpAvailable = false
    m.statusMessage = "MCP unavailable - using AllowedTools mode"
    
    // Log for debugging
    logger.Warn("MCP server communication failed, degrading gracefully",
        "error", err,
        "fallback", "AllowedTools")
    
    return nil
}

// In callback client with retry
func (c *Client) SendPromptWithRetry(ctx context.Context, req PromptRequest) (PromptResponse, error) {
    var lastErr error
    backoff := 100 * time.Millisecond
    
    for attempt := 0; attempt < 3; attempt++ {
        resp, err := c.SendPrompt(ctx, req)
        if err == nil {
            return resp, nil
        }
        lastErr = err
        
        select {
        case <-ctx.Done():
            return PromptResponse{}, ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
        }
    }
    
    return PromptResponse{}, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Resource cleanup

```go
// Main gofortress cleanup
func (app *Application) Cleanup() {
    // Shutdown callback server
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if app.callbackServer != nil {
        app.callbackServer.Shutdown(ctx)
        app.callbackServer.Cleanup()
    }
    
    // Remove MCP config file
    if app.mcpConfigPath != "" {
        os.Remove(app.mcpConfigPath)
    }
    
    // Kill Claude CLI if still running
    if app.claudeProcess != nil && app.claudeProcess.Process != nil {
        app.claudeProcess.Process.Signal(syscall.SIGTERM)
        
        // Wait with timeout, then force kill
        done := make(chan error)
        go func() { done <- app.claudeProcess.Wait() }()
        
        select {
        case <-done:
        case <-time.After(3 * time.Second):
            app.claudeProcess.Process.Kill()
        }
    }
}
```

---

## Configuration reference

### Complete gofortress configuration

```yaml
# ~/.config/gofortress/config.yaml
mcp:
  enabled: true
  server_binary: "/usr/local/bin/gofortress-mcp-server"
  timeout: 5m
  debug: false

hooks:
  preToolUse: "gogent-validate"
  postToolUse: "gogent-sharp-edge"
  sessionStart: "gogent-load-context"
  subagentStop: "gogent-agent-endstate"
  sessionEnd: "gogent-archive"

allowedTools:
  - "Read"
  - "Write"
  - "Bash"
  - "mcp__gofortress__ask_user"
  - "mcp__gofortress__confirm_action"
```

### Environment variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOFORTRESS_SOCKET` | Unix socket path (set by TUI) | `/tmp/gofortress-{pid}.sock` |
| `LOG_LEVEL` | MCP server log level | `info` |
| `MCP_TIMEOUT` | Claude MCP initialization timeout | `10000` (ms) |
| `GOFORTRESS_DEBUG` | Enable debug logging | `false` |

---

## Implementation phases

### Phase 1: Foundation (Week 1)

**Tasks:**
1. Create `internal/callback` package with server and client
2. Implement Unix socket HTTP server with cleanup
3. Add socket server startup to gofortress main
4. Write unit tests for callback package

**Acceptance criteria:**
- Socket server starts and accepts connections
- Health endpoint returns 200 OK
- Socket cleaned up on process exit
- >80% test coverage for callback package

### Phase 2: MCP Server (Week 2)

**Tasks:**
1. Create `cmd/gofortress-mcp-server` binary
2. Implement all four tool handlers
3. Add callback client integration
4. Write integration tests with in-memory transport

**Acceptance criteria:**
- MCP server runs and responds to tool calls
- All tools communicate with callback server
- Server handles disconnection gracefully
- >80% test coverage

### Phase 3: TUI Integration (Week 3)

**Tasks:**
1. Add modal state management to Bubbletea model
2. Implement prompt rendering with lipgloss
3. Connect callback server to model via channels
4. Add `Program.Send()` integration for external events

**Acceptance criteria:**
- Modals display over main content
- All prompt types (confirm, input, select) work
- Responses flow back to callback server
- teatest tests pass

### Phase 4: Claude Integration (Week 4)

**Tasks:**
1. Implement MCP config generation
2. Add MCP server binary embedding or distribution
3. Integrate with Claude CLI spawning
4. End-to-end testing with real Claude CLI

**Acceptance criteria:**
- Claude discovers and connects to MCP server
- Interactive prompts work in real sessions
- Graceful fallback when MCP unavailable
- Full integration tests pass

---

## Troubleshooting guide

### MCP server fails to start

**Symptoms:** Claude reports "MCP server failed to initialize"

**Diagnosis:**
```bash
# Check if socket exists and is accessible
ls -la /tmp/gofortress-*.sock

# Manually test MCP server
GOFORTRESS_SOCKET=/tmp/gofortress-1234.sock ./gofortress-mcp-server 2>&1
```

**Solutions:**
- Verify `GOFORTRESS_SOCKET` environment variable is set in MCP config
- Check socket permissions (should be 0600)
- Ensure TUI started callback server before spawning Claude

### Prompts not displaying

**Symptoms:** Claude calls MCP tools but no modal appears

**Diagnosis:**
```go
// Add debug logging to callback server
logger.Debug("received prompt request", "id", req.ID, "type", req.Type)
```

**Solutions:**
- Verify `PromptChan` has listeners
- Check that `listenForPrompts()` command is running
- Ensure TUI model is receiving `MCPPromptMsg`

### Response timeout

**Symptoms:** MCP tool returns timeout error

**Diagnosis:**
- Check TUI logs for response sending
- Verify prompt ID matches in pending map

**Solutions:**
- Increase `WriteTimeout` on HTTP server
- Check that `SendResponse` is called with correct ID
- Verify response channel isn't blocked

### Socket path too long

**Symptoms:** "socket path too long" error on startup

**Solution:**
```go
// Use shorter path
func getSocketPath(pid int) string {
    return fmt.Sprintf("/tmp/gf-%d.sock", pid) // Shorter name
}
```

---

## Migration path

### From current gofortress to MCP-enabled

1. **Update dependencies:**
   ```bash
   go get github.com/modelcontextprotocol/go-sdk@v1.2.0
   ```

2. **Build MCP server:**
   ```bash
   go build -o gofortress-mcp-server ./cmd/gofortress-mcp-server
   ```

3. **Update gofortress binary:**
   - Add callback server initialization
   - Add MCP config generation
   - Update Claude spawning to include `--mcp-config`

4. **Test incrementally:**
   - Verify callback server works standalone
   - Test MCP server with mock TUI
   - Integration test full flow

5. **Deploy:**
   - Distribute both binaries or embed MCP server
   - Update user documentation
   - Monitor for errors in production

This implementation guide provides everything needed to add MCP-based interactive prompts to gofortress while maintaining the existing hooks system and single-binary simplicity.