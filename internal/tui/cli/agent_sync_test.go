package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Bucket-Chemist/goYoke/internal/tui/state"
	"github.com/Bucket-Chemist/goYoke/internal/tui/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newRegistry() *state.AgentRegistry {
	return state.NewAgentRegistry()
}

func ptr(s string) *string { return &s }

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ---------------------------------------------------------------------------
// ParseTaskInput
// ---------------------------------------------------------------------------

func TestParseTaskInput(t *testing.T) {
	tests := []struct {
		name         string
		input        json.RawMessage
		wantOK       bool
		wantType     string
		wantModel    string
		wantTier     string
		wantDesc     string
	}{
		{
			name: "valid full input",
			input: mustJSON(map[string]any{
				"description":   "Brief description",
				"subagent_type": "GO Pro",
				"model":         "sonnet",
				"prompt":        "AGENT: go-pro\n\nTASK: ...",
			}),
			wantOK:    true,
			wantType:  "go-pro",
			wantModel: "sonnet",
			wantTier:  "sonnet",
			wantDesc:  "Brief description",
		},
		{
			name: "opus model maps to opus tier",
			input: mustJSON(map[string]any{
				"description":   "Deep analysis",
				"subagent_type": "Python ML Architect",
				"model":         "claude-opus-4",
				"prompt":        "...",
			}),
			wantOK:    true,
			wantType:  "python-ml-architect",
			wantModel: "claude-opus-4",
			wantTier:  "opus",
			wantDesc:  "Deep analysis",
		},
		{
			name: "haiku model maps to haiku tier",
			input: mustJSON(map[string]any{
				"description":   "Find files",
				"subagent_type": "Haiku Scout",
				"model":         "claude-haiku-3",
				"prompt":        "...",
			}),
			wantOK:    true,
			wantType:  "haiku-scout",
			wantModel: "claude-haiku-3",
			wantTier:  "haiku",
			wantDesc:  "Find files",
		},
		{
			name: "missing optional fields still succeeds",
			input: mustJSON(map[string]any{
				"description": "minimal",
			}),
			wantOK:    true,
			wantType:  "",
			wantModel: "",
			wantTier:  "",
			wantDesc:  "minimal",
		},
		{
			name:   "nil / empty input returns false",
			input:  nil,
			wantOK: false,
		},
		{
			name:   "invalid JSON returns false",
			input:  json.RawMessage(`{not valid json`),
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agent, ok := ParseTaskInput(tc.input)
			assert.Equal(t, tc.wantOK, ok)
			if !tc.wantOK {
				return
			}
			assert.Equal(t, tc.wantDesc, agent.Description)
			assert.Equal(t, tc.wantType, agent.AgentType)
			assert.Equal(t, tc.wantModel, agent.Model)
			assert.Equal(t, tc.wantTier, agent.Tier)
		})
	}
}

// ---------------------------------------------------------------------------
// extractToolTarget
// ---------------------------------------------------------------------------

func TestExtractToolTarget(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    json.RawMessage
		want     string
	}{
		{
			name:     "Read uses file_path",
			toolName: "Read",
			input:    mustJSON(map[string]any{"file_path": "/src/main.go"}),
			want:     "/src/main.go",
		},
		{
			name:     "Write uses file_path",
			toolName: "Write",
			input:    mustJSON(map[string]any{"file_path": "/src/output.go", "content": "..."}),
			want:     "/src/output.go",
		},
		{
			name:     "Edit uses file_path",
			toolName: "Edit",
			input:    mustJSON(map[string]any{"file_path": "/src/edit.go", "old_string": "x", "new_string": "y"}),
			want:     "/src/edit.go",
		},
		{
			name:     "Bash uses command truncated to 80",
			toolName: "Bash",
			input:    mustJSON(map[string]any{"command": "go test ./..."}),
			want:     "go test ./...",
		},
		{
			name:     "Bash truncates long command",
			toolName: "Bash",
			input: mustJSON(map[string]any{
				// 93 rune command — Truncate(s, 80) returns first 79 runes + "…" = 80 runes
				"command": "GOOS=linux GOARCH=amd64 go build -ldflags '-X main.version=1.0.0' -o dist/binary ./cmd/binary",
			}),
			want: "GOOS=linux GOARCH=amd64 go build -ldflags '-X main.version=1.0.0' -o dist/binar…",
		},
		{
			name:     "Grep uses pattern",
			toolName: "Grep",
			input:    mustJSON(map[string]any{"pattern": "func.*Error", "path": "./..."}),
			want:     "func.*Error",
		},
		{
			name:     "Glob uses pattern",
			toolName: "Glob",
			input:    mustJSON(map[string]any{"pattern": "**/*.go"}),
			want:     "**/*.go",
		},
		{
			name:     "WebFetch uses url",
			toolName: "WebFetch",
			input:    mustJSON(map[string]any{"url": "https://pkg.go.dev/sync"}),
			want:     "https://pkg.go.dev/sync",
		},
		{
			name:     "WebSearch uses query",
			toolName: "WebSearch",
			input:    mustJSON(map[string]any{"query": "golang errgroup pattern"}),
			want:     "golang errgroup pattern",
		},
		{
			name:     "unknown tool falls back to tool name",
			toolName: "TodoWrite",
			input:    mustJSON(map[string]any{"todos": []string{"fix it"}}),
			want:     "TodoWrite",
		},
		{
			name:     "nil input falls back to tool name",
			toolName: "Read",
			input:    nil,
			want:     "Read",
		},
		{
			name:     "invalid JSON falls back to tool name",
			toolName: "Bash",
			input:    json.RawMessage(`{bad json`),
			want:     "Bash",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractToolTarget(tc.toolName, tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// ExtractToolActivities
// ---------------------------------------------------------------------------

func TestExtractToolActivities_SingleActivity(t *testing.T) {
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_abc",
		Name:  "Read",
		Input: mustJSON(map[string]any{"file_path": "/src/config.go"}),
	}

	before := time.Now()
	activities := ExtractToolActivities(block, "")
	after := time.Now()

	require.Len(t, activities, 1)
	activity := activities[0]
	assert.Equal(t, "tool_use", activity.Type)
	assert.Equal(t, "/src/config.go", activity.Target)
	assert.Equal(t, "Read: /src/config.go", activity.Preview)
	assert.True(t, !activity.Timestamp.Before(before))
	assert.True(t, !activity.Timestamp.After(after))
}

func TestExtractToolActivities_EmptyTarget(t *testing.T) {
	// A tool with no recognised input fields should still produce a valid activity.
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_xyz",
		Name:  "UnknownTool",
		Input: mustJSON(map[string]any{"someField": "value"}),
	}

	activities := ExtractToolActivities(block, "")
	require.Len(t, activities, 1)
	activity := activities[0]
	assert.Equal(t, "tool_use", activity.Type)
	// extractToolTarget falls back to tool name
	assert.Equal(t, "UnknownTool", activity.Target)
	assert.Equal(t, "UnknownTool: UnknownTool", activity.Preview)
}

// ---------------------------------------------------------------------------
// stripProjectRoot
// ---------------------------------------------------------------------------

func TestStripProjectRoot(t *testing.T) {
	tests := []struct {
		name string
		path string
		root string
		want string
	}{
		{
			name: "absolute path under root strips to relative",
			path: "/home/user/project/internal/pkg/file.go",
			root: "/home/user/project",
			want: "internal/pkg/file.go",
		},
		{
			name: "path at root level",
			path: "/home/user/project/main.go",
			root: "/home/user/project",
			want: "main.go",
		},
		{
			name: "path outside root returned unchanged",
			path: "/other/place/file.go",
			root: "/home/user/project",
			want: "/other/place/file.go",
		},
		{
			name: "empty root disables stripping",
			path: "/home/user/project/file.go",
			root: "",
			want: "/home/user/project/file.go",
		},
		{
			name: "empty path returned unchanged",
			path: "",
			root: "/home/user/project",
			want: "",
		},
		{
			name: "already relative path (no leading slash) returned unchanged",
			path: "internal/pkg/file.go",
			root: "/home/user/project",
			want: "internal/pkg/file.go",
		},
		{
			name: "sibling directory returns unchanged (starts with ..)",
			path: "/home/user/other/file.go",
			root: "/home/user/project",
			want: "/home/user/other/file.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripProjectRoot(tc.path, tc.root)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// splitCompoundCommand
// ---------------------------------------------------------------------------

func TestSplitCompoundCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "simple compound command",
			input: "go build ./... && go test ./...",
			want:  []string{"go build ./...", "go test ./..."},
		},
		{
			name:  "three parts",
			input: "make clean && make build && make test",
			want:  []string{"make clean", "make build", "make test"},
		},
		{
			name:  "single command (no &&) returned as one element",
			input: "go test ./...",
			want:  []string{"go test ./..."},
		},
		{
			name:  "extra whitespace trimmed",
			input: "cmd1  &&   cmd2",
			want:  []string{"cmd1", "cmd2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCompoundCommand(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// ExtractToolActivities — path stripping and compound split
// ---------------------------------------------------------------------------

func TestExtractToolActivities_StripsProjectRoot(t *testing.T) {
	root := "/home/user/project"
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_read",
		Name:  "Read",
		Input: mustJSON(map[string]any{"file_path": "/home/user/project/internal/tui/app.go"}),
	}

	activities := ExtractToolActivities(block, root)

	require.Len(t, activities, 1)
	assert.Equal(t, "internal/tui/app.go", activities[0].Target)
	assert.Equal(t, "Read: internal/tui/app.go", activities[0].Preview)
}

func TestExtractToolActivities_PathOutsideRootUnchanged(t *testing.T) {
	root := "/home/user/project"
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_read",
		Name:  "Read",
		Input: mustJSON(map[string]any{"file_path": "/etc/hosts"}),
	}

	activities := ExtractToolActivities(block, root)

	require.Len(t, activities, 1)
	assert.Equal(t, "/etc/hosts", activities[0].Target)
}

func TestExtractToolActivities_CompoundBashSplit(t *testing.T) {
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_bash",
		Name:  "Bash",
		Input: mustJSON(map[string]any{"command": "go build ./... && go test ./..."}),
	}

	activities := ExtractToolActivities(block, "")

	require.Len(t, activities, 2)
	assert.Equal(t, "go build ./...", activities[0].Target)
	assert.Equal(t, "Bash: go build ./...", activities[0].Preview)
	assert.Equal(t, "go test ./...", activities[1].Target)
	assert.Equal(t, "Bash: go test ./...", activities[1].Preview)
	// Both share the same ToolID
	assert.Equal(t, "toolu_bash", activities[0].ToolID)
	assert.Equal(t, "toolu_bash", activities[1].ToolID)
}

func TestExtractToolActivities_SimpleBashNoSplit(t *testing.T) {
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_bash",
		Name:  "Bash",
		Input: mustJSON(map[string]any{"command": "go test ./..."}),
	}

	activities := ExtractToolActivities(block, "")

	require.Len(t, activities, 1)
	assert.Equal(t, "go test ./...", activities[0].Target)
}

func TestExtractToolActivities_NonFileToolNotStripped(t *testing.T) {
	root := "/home/user/project"
	block := ContentBlock{
		Type:  "tool_use",
		ID:    "toolu_grep",
		Name:  "Grep",
		Input: mustJSON(map[string]any{"pattern": "func.*Error"}),
	}

	activities := ExtractToolActivities(block, root)

	require.Len(t, activities, 1)
	// Pattern should not be modified by path stripping
	assert.Equal(t, "func.*Error", activities[0].Target)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — Task tool_use registers agent
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_TaskRegistersAgent(t *testing.T) {
	reg := newRegistry()

	ev := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_001",
					Name: "Task",
					Input: mustJSON(map[string]any{
						"description":   "Go implementation",
						"subagent_type": "GO Pro",
						"model":         "sonnet",
						"prompt":        "AGENT: go-pro\n\nTASK: ...",
					}),
				},
			},
		},
		ParentToolUseID: nil,
	}

	result := SyncAssistantEvent(ev, reg)

	require.Len(t, result.Registered, 1)
	assert.Equal(t, "toolu_001", result.Registered[0])
	assert.Empty(t, result.Activity)

	agent := reg.Get("toolu_001")
	require.NotNil(t, agent)
	assert.Equal(t, "go-pro", agent.AgentType)
	assert.Equal(t, "Go implementation", agent.Description)
	assert.Equal(t, "sonnet", agent.Model)
	assert.Equal(t, "sonnet", agent.Tier)
	assert.Equal(t, state.StatusRunning, agent.Status)
	assert.Empty(t, agent.ParentID) // root-level spawn
}

func TestSyncAssistantEvent_TaskWithParentID(t *testing.T) {
	// Register parent first so link works.
	reg := newRegistry()
	require.NoError(t, reg.Register(state.Agent{
		ID:        "toolu_parent",
		AgentType: "orchestrator",
		Status:    state.StatusRunning,
	}))

	parentID := "toolu_parent"
	ev := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_child",
					Name: "Task",
					Input: mustJSON(map[string]any{
						"description":   "Child task",
						"subagent_type": "GO Pro",
						"model":         "sonnet",
						"prompt":        "...",
					}),
				},
			},
		},
		ParentToolUseID: &parentID,
	}

	result := SyncAssistantEvent(ev, reg)

	require.Len(t, result.Registered, 1)
	agent := reg.Get("toolu_child")
	require.NotNil(t, agent)
	assert.Equal(t, "toolu_parent", agent.ParentID)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — duplicate registration is skipped
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_DuplicateSkipped(t *testing.T) {
	reg := newRegistry()

	taskBlock := ContentBlock{
		Type: "tool_use",
		ID:   "toolu_dup",
		Name: "Task",
		Input: mustJSON(map[string]any{
			"description":   "Duplicate agent",
			"subagent_type": "GO Pro",
			"model":         "sonnet",
			"prompt":        "...",
		}),
	}

	ev := AssistantEvent{
		Type:    "assistant",
		Message: AssistantMessage{Content: []ContentBlock{taskBlock}},
	}

	result1 := SyncAssistantEvent(ev, reg)
	require.Len(t, result1.Registered, 1)

	// Second event with same agentType+description — should be silently skipped.
	ev2 := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type: "tool_use",
					ID:   "toolu_dup2", // different tool_use ID
					Name: "Task",
					Input: mustJSON(map[string]any{
						"description":   "Duplicate agent",
						"subagent_type": "GO Pro",
						"model":         "sonnet",
						"prompt":        "...",
					}),
				},
			},
		},
	}

	result2 := SyncAssistantEvent(ev2, reg)
	assert.Empty(t, result2.Registered)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — non-Task tool_use with ParentToolUseID sets activity
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_NonTaskToolUseSetsActivity(t *testing.T) {
	reg := newRegistry()
	parentID := "toolu_parent"
	require.NoError(t, reg.Register(state.Agent{
		ID:        parentID,
		AgentType: "go-pro",
		Status:    state.StatusRunning,
	}))

	ev := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_read",
					Name:  "Read",
					Input: mustJSON(map[string]any{"file_path": "/src/main.go"}),
				},
			},
		},
		ParentToolUseID: &parentID,
	}

	result := SyncAssistantEvent(ev, reg)

	assert.Empty(t, result.Registered)
	require.Len(t, result.Activity, 1)
	assert.Equal(t, parentID, result.Activity[0])

	agent := reg.Get(parentID)
	require.NotNil(t, agent)
	require.NotNil(t, agent.Activity)
	assert.Equal(t, "tool_use", agent.Activity.Type)
	assert.Equal(t, "/src/main.go", agent.Activity.Target)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — root-level non-Task tool_use is skipped
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_RootNonTaskSkipped(t *testing.T) {
	reg := newRegistry()

	ev := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_bash",
					Name:  "Bash",
					Input: mustJSON(map[string]any{"command": "go build ./..."}),
				},
			},
		},
		ParentToolUseID: nil, // root-level
	}

	result := SyncAssistantEvent(ev, reg)

	assert.Empty(t, result.Registered)
	assert.Empty(t, result.Activity)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — invalid Task input is silently skipped
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_InvalidTaskInputSkipped(t *testing.T) {
	reg := newRegistry()

	ev := AssistantEvent{
		Type: "assistant",
		Message: AssistantMessage{
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_bad",
					Name:  "Task",
					Input: json.RawMessage(`{bad json`),
				},
			},
		},
	}

	result := SyncAssistantEvent(ev, reg)
	assert.Empty(t, result.Registered)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — tool_result completes agent
// ---------------------------------------------------------------------------

func TestSyncUserEvent_CompletesAgent(t *testing.T) {
	reg := newRegistry()
	agentID := "toolu_agent"
	require.NoError(t, reg.Register(state.Agent{
		ID:        agentID,
		AgentType: "go-pro",
		Status:    state.StatusRunning,
	}))

	ev := UserEvent{
		Type: "user",
		Message: UserMessage{
			Content: []ContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: agentID,
					IsError:   false,
				},
			},
		},
		ParentToolUseID: nil,
	}

	result := SyncUserEvent(ev, reg)

	require.Len(t, result.Updated, 1)
	assert.Equal(t, agentID, result.Updated[0])

	agent := reg.Get(agentID)
	require.NotNil(t, agent)
	assert.Equal(t, state.StatusComplete, agent.Status)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — tool_result with is_error sets StatusError
// ---------------------------------------------------------------------------

func TestSyncUserEvent_ErrorsAgent(t *testing.T) {
	reg := newRegistry()
	agentID := "toolu_err"
	require.NoError(t, reg.Register(state.Agent{
		ID:        agentID,
		AgentType: "go-pro",
		Status:    state.StatusRunning,
	}))

	ev := UserEvent{
		Type: "user",
		Message: UserMessage{
			Content: []ContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: agentID,
					IsError:   true,
				},
			},
		},
	}

	result := SyncUserEvent(ev, reg)

	require.Len(t, result.Updated, 1)

	agent := reg.Get(agentID)
	require.NotNil(t, agent)
	assert.Equal(t, state.StatusError, agent.Status)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — orphaned tool_result is gracefully ignored
// ---------------------------------------------------------------------------

func TestSyncUserEvent_OrphanedToolResultIgnored(t *testing.T) {
	reg := newRegistry()

	ev := UserEvent{
		Type: "user",
		Message: UserMessage{
			Content: []ContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: "toolu_unknown",
				},
			},
		},
	}

	// Must not panic.
	result := SyncUserEvent(ev, reg)
	assert.Empty(t, result.Updated)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — subagent tool_result clears parent activity
// ---------------------------------------------------------------------------

func TestSyncUserEvent_ClearsParentActivity(t *testing.T) {
	reg := newRegistry()
	parentID := "toolu_parent"
	require.NoError(t, reg.Register(state.Agent{
		ID:        parentID,
		AgentType: "orchestrator",
		Status:    state.StatusRunning,
	}))
	// Set some existing activity on parent.
	reg.SetActivity(parentID, state.AgentActivity{
		Type:    "tool_use",
		Target:  "/src/file.go",
		Preview: "Read: /src/file.go",
	})

	ev := UserEvent{
		Type: "user",
		Message: UserMessage{
			// No tool_result blocks for agents here — just testing parent activity clear.
			Content: []ContentBlock{},
		},
		ParentToolUseID: &parentID,
	}

	result := SyncUserEvent(ev, reg)

	// parentID should appear in Activity list.
	require.Contains(t, result.Activity, parentID)

	agent := reg.Get(parentID)
	require.NotNil(t, agent)
	require.NotNil(t, agent.Activity)
	assert.Equal(t, "idle", agent.Activity.Type)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — non-running agent status not changed
// ---------------------------------------------------------------------------

func TestSyncUserEvent_NonRunningAgentNotChanged(t *testing.T) {
	reg := newRegistry()
	agentID := "toolu_done"
	require.NoError(t, reg.Register(state.Agent{
		ID:        agentID,
		AgentType: "go-pro",
		Status:    state.StatusRunning,
	}))
	// Transition to Complete first.
	require.NoError(t, reg.Update(agentID, func(a *state.Agent) {
		a.Status = state.StatusComplete
	}))

	ev := UserEvent{
		Type: "user",
		Message: UserMessage{
			Content: []ContentBlock{
				{
					Type:      "tool_result",
					ToolUseID: agentID,
					IsError:   false,
				},
			},
		},
	}

	result := SyncUserEvent(ev, reg)
	// Status was already Complete; should not re-update.
	assert.Empty(t, result.Updated)

	agent := reg.Get(agentID)
	assert.Equal(t, state.StatusComplete, agent.Status)
}

// ---------------------------------------------------------------------------
// SyncAssistantEvent — no panic on empty content
// ---------------------------------------------------------------------------

func TestSyncAssistantEvent_EmptyContent(t *testing.T) {
	reg := newRegistry()

	ev := AssistantEvent{
		Type:    "assistant",
		Message: AssistantMessage{Content: nil},
	}

	result := SyncAssistantEvent(ev, reg)
	assert.Empty(t, result.Registered)
	assert.Empty(t, result.Activity)
}

// ---------------------------------------------------------------------------
// SyncUserEvent — no panic on empty content
// ---------------------------------------------------------------------------

func TestSyncUserEvent_EmptyContent(t *testing.T) {
	reg := newRegistry()

	ev := UserEvent{
		Type:    "user",
		Message: UserMessage{Content: nil},
	}

	result := SyncUserEvent(ev, reg)
	assert.Empty(t, result.Updated)
	assert.Empty(t, result.Activity)
}

// ---------------------------------------------------------------------------
// normaliseAgentType
// ---------------------------------------------------------------------------

func TestNormaliseAgentType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"GO Pro", "go-pro"},
		{"Python ML Architect", "python-ml-architect"},
		{"Staff Architect Critical Review", "staff-architect-critical-review"},
		{"", ""},
		{"haiku-scout", "haiku-scout"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, normaliseAgentType(tc.input))
		})
	}
}

// ---------------------------------------------------------------------------
// modelToTier
// ---------------------------------------------------------------------------

func TestModelToTier(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"claude-haiku-3-5", "haiku"},
		{"claude-opus-4", "opus"},
		{"claude-sonnet-4-5", "sonnet"},
		{"sonnet", "sonnet"},
		{"opus", "opus"},
		{"haiku", "haiku"},
		{"", ""},
		{"claude-unknown-2", "claude-unknown-2"},
	}

	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			assert.Equal(t, tc.want, modelToTier(tc.model))
		})
	}
}

// ---------------------------------------------------------------------------
// truncate
// ---------------------------------------------------------------------------

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exact length unchanged", "hello", 5, "hello"},
		{"truncates with ellipsis", "hello world", 5, "hell…"},
		{"empty string", "", 10, ""},
		{"unicode aware", "日本語テスト", 4, "日本語…"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, util.Truncate(tc.input, tc.max))
		})
	}
}
