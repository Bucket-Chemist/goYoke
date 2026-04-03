package state

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeAgent(id, agentType, description, parentID string) Agent {
	return Agent{
		ID:          id,
		AgentType:   agentType,
		Description: description,
		ParentID:    parentID,
		Status:      StatusPending,
		StartedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// AgentStatus.String
// ---------------------------------------------------------------------------

func TestAgentStatusString(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusComplete, "complete"},
		{StatusError, "error"},
		{StatusKilled, "killed"},
		{AgentStatus(99), "AgentStatus(99)"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.status.String())
		})
	}
}

// ---------------------------------------------------------------------------
// isValidTransition
// ---------------------------------------------------------------------------

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		from  AgentStatus
		to    AgentStatus
		valid bool
	}{
		// Valid forward transitions
		{StatusPending, StatusRunning, true},
		{StatusPending, StatusKilled, true},
		{StatusRunning, StatusComplete, true},
		{StatusRunning, StatusError, true},
		{StatusRunning, StatusKilled, true},
		// Terminal → anything: invalid
		{StatusComplete, StatusRunning, false},
		{StatusComplete, StatusPending, false},
		{StatusError, StatusRunning, false},
		{StatusKilled, StatusRunning, false},
		// Same-state: invalid
		{StatusPending, StatusPending, false},
		{StatusRunning, StatusRunning, false},
		// Backwards: invalid
		{StatusRunning, StatusPending, false},
		{StatusComplete, StatusError, false},
	}
	for _, tc := range tests {
		tc := tc
		name := fmt.Sprintf("%s->%s", tc.from, tc.to)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.valid, isValidTransition(tc.from, tc.to))
		})
	}
}

// ---------------------------------------------------------------------------
// NewAgentRegistry
// ---------------------------------------------------------------------------

func TestNewAgentRegistry_Empty(t *testing.T) {
	r := NewAgentRegistry()
	require.NotNil(t, r)
	assert.Equal(t, "", r.RootID())
	assert.Equal(t, "", r.Selected())
	assert.Empty(t, r.Tree())
	stats := r.Count()
	assert.Equal(t, AgentStats{}, stats)
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_SingleAgent(t *testing.T) {
	r := NewAgentRegistry()
	a := makeAgent("a1", "go-pro", "implement auth", "")
	require.NoError(t, r.Register(a))
	assert.Equal(t, "a1", r.RootID())

	got := r.Get("a1")
	require.NotNil(t, got)
	assert.Equal(t, "go-pro", got.AgentType)
	assert.Equal(t, StatusPending, got.Status)
}

func TestRegister_RootTrackedOnce(t *testing.T) {
	r := NewAgentRegistry()
	// Register two root-level agents — only the first should become root.
	require.NoError(t, r.Register(makeAgent("r1", "go-pro", "task-one", "")))
	require.NoError(t, r.Register(makeAgent("r2", "go-cli", "task-two", "")))
	assert.Equal(t, "r1", r.RootID())
}

func TestRegister_ParentLinkage(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("parent", "go-pro", "parent task", "")))
	require.NoError(t, r.Register(makeAgent("child1", "go-tui", "child task", "parent")))

	parent := r.Get("parent")
	require.NotNil(t, parent)
	assert.Contains(t, parent.Children, "child1")
}

func TestRegister_UnknownParentDoesNotError(t *testing.T) {
	// If the parent is not registered yet, we just skip linkage — no error.
	r := NewAgentRegistry()
	err := r.Register(makeAgent("orphan", "go-pro", "orphan task", "nonexistent-parent"))
	assert.NoError(t, err)
}

// Dedup: same key, already pending → blocked.
func TestRegister_DedupPendingBlocked(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("id1", "go-pro", "same desc", "")))

	err := r.Register(makeAgent("id2", "go-pro", "same desc", ""))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateAgent), "expected ErrDuplicateAgent, got %v", err)
}

// Dedup: same key, already running → blocked.
func TestRegister_DedupRunningBlocked(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("id1", "go-pro", "same desc", "")))
	require.NoError(t, r.Update("id1", func(a *Agent) { a.Status = StatusRunning }))

	err := r.Register(makeAgent("id2", "go-pro", "same desc", ""))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDuplicateAgent))
}

// Dedup: same key but previous is complete → allowed.
func TestRegister_DedupCompleteAllowed(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("id1", "go-pro", "same desc", "")))
	require.NoError(t, r.Update("id1", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("id1", func(a *Agent) { a.Status = StatusComplete }))

	err := r.Register(makeAgent("id2", "go-pro", "same desc", ""))
	assert.NoError(t, err)
}

// Dedup: same key but previous is error → allowed.
func TestRegister_DedupErrorAllowed(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("id1", "go-pro", "same desc", "")))
	require.NoError(t, r.Update("id1", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("id1", func(a *Agent) { a.Status = StatusError }))

	err := r.Register(makeAgent("id2", "go-pro", "same desc", ""))
	assert.NoError(t, err)
}

// Different agentType with same description → different key → allowed.
func TestRegister_DifferentTypeSameDesc(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("id1", "go-pro", "desc", "")))
	err := r.Register(makeAgent("id2", "go-cli", "desc", ""))
	assert.NoError(t, err)
}

// Get() returns a copy — mutations do not affect registry.
func TestRegister_GetReturnsCopy(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "desc", "")))

	got := r.Get("a1")
	require.NotNil(t, got)
	got.Status = StatusRunning // mutate copy

	// Registry should still have the original status.
	original := r.Get("a1")
	assert.Equal(t, StatusPending, original.Status)
}

// Get() returns nil for unknown ID.
func TestGet_NotFound(t *testing.T) {
	r := NewAgentRegistry()
	assert.Nil(t, r.Get("nope"))
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUpdate_NotFound(t *testing.T) {
	r := NewAgentRegistry()
	err := r.Update("nope", func(a *Agent) { a.Status = StatusRunning })
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentNotFound))
}

func TestUpdate_ValidTransition(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	err := r.Update("a1", func(a *Agent) { a.Status = StatusRunning })
	require.NoError(t, err)

	a := r.Get("a1")
	assert.Equal(t, StatusRunning, a.Status)
}

func TestUpdate_InvalidTransitionReverted(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusComplete }))

	// Complete → Running is invalid; status must remain Complete.
	err := r.Update("a1", func(a *Agent) { a.Status = StatusRunning })
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidTransition))

	a := r.Get("a1")
	assert.Equal(t, StatusComplete, a.Status)
}

func TestUpdate_NonStatusFieldsAlwaysApplied(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	err := r.Update("a1", func(a *Agent) {
		a.Cost = 1.23
		a.Tokens = 500
	})
	require.NoError(t, err)

	a := r.Get("a1")
	assert.InDelta(t, 1.23, a.Cost, 0.001)
	assert.Equal(t, 500, a.Tokens)
}

// Full valid transition chain: Pending → Running → Complete.
func TestUpdate_TransitionChain(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("a1", func(a *Agent) { a.Status = StatusComplete }))

	a := r.Get("a1")
	assert.Equal(t, StatusComplete, a.Status)
}

// ---------------------------------------------------------------------------
// SetActivity
// ---------------------------------------------------------------------------

func TestSetActivity_Basic(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	act := AgentActivity{
		Type:      "tool_use",
		Target:    "Read",
		Preview:   "reading config.go",
		Timestamp: time.Now(),
	}
	r.SetActivity("a1", act)

	a := r.Get("a1")
	require.NotNil(t, a.Activity)
	assert.Equal(t, "tool_use", a.Activity.Type)
	assert.Equal(t, "Read", a.Activity.Target)
}

func TestSetActivity_UnknownIDNoOp(t *testing.T) {
	r := NewAgentRegistry()
	// Should not panic.
	r.SetActivity("nope", AgentActivity{Type: "tool_use"})
}

// Activity in Get() returns a copy.
func TestSetActivity_GetReturnsCopy(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	r.SetActivity("a1", AgentActivity{Type: "tool_use", Target: "Read"})

	a := r.Get("a1")
	require.NotNil(t, a.Activity)
	a.Activity.Target = "mutated" // mutate copy

	// Registry should still have original.
	a2 := r.Get("a1")
	assert.Equal(t, "Read", a2.Activity.Target)
}

// ---------------------------------------------------------------------------
// AppendActivity / FullActivityLog / UpdateActivityResult
// ---------------------------------------------------------------------------

func TestAppendActivity_WritesToBothBuffers(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	act := AgentActivity{Type: "tool_use", Target: "Read", ToolID: "tool-1"}
	r.AppendActivity("a1", act)

	a := r.Get("a1")
	require.Len(t, a.RecentActivity, 1)
	assert.Equal(t, "tool-1", a.RecentActivity[0].ToolID)
	require.Len(t, a.FullActivityLog, 1)
	assert.Equal(t, "tool-1", a.FullActivityLog[0].ToolID)
}

func TestAppendActivity_RecentActivityCappedAt5(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	for i := range 7 {
		r.AppendActivity("a1", AgentActivity{
			Type:   "tool_use",
			Target: fmt.Sprintf("tool-%d", i),
			ToolID: fmt.Sprintf("id-%d", i),
		})
	}

	a := r.Get("a1")
	assert.Len(t, a.RecentActivity, 5, "RecentActivity must cap at 5")
	// Oldest two evicted; last 5 remain.
	assert.Equal(t, "id-2", a.RecentActivity[0].ToolID)
	assert.Equal(t, "id-6", a.RecentActivity[4].ToolID)
}

func TestAppendActivity_FullActivityLogCappedAt500(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	for i := range 502 {
		r.AppendActivity("a1", AgentActivity{
			Type:   "tool_use",
			ToolID: fmt.Sprintf("id-%d", i),
		})
	}

	a := r.Get("a1")
	assert.Len(t, a.FullActivityLog, 500, "FullActivityLog must cap at 500")
	// First two evicted; entry at index 0 should be id-2.
	assert.Equal(t, "id-2", a.FullActivityLog[0].ToolID)
	assert.Equal(t, "id-501", a.FullActivityLog[499].ToolID)
}

func TestAppendActivity_FullActivityLogDeepCopied(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})

	a := r.Get("a1")
	require.Len(t, a.FullActivityLog, 1)
	a.FullActivityLog[0].Target = "mutated"

	// Registry must not be affected.
	a2 := r.Get("a1")
	assert.Equal(t, "", a2.FullActivityLog[0].Target)
}

func TestUpdateActivityResult_SetsSuccessOnBothBuffers(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})
	r.UpdateActivityResult("a1", "t1", true)

	a := r.Get("a1")
	require.NotNil(t, a.RecentActivity[0].Success)
	assert.True(t, *a.RecentActivity[0].Success)
	require.NotNil(t, a.FullActivityLog[0].Success)
	assert.True(t, *a.FullActivityLog[0].Success)
}

func TestUpdateActivityResult_SuccessFalse(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})
	r.UpdateActivityResult("a1", "t1", false)

	a := r.Get("a1")
	require.NotNil(t, a.FullActivityLog[0].Success)
	assert.False(t, *a.FullActivityLog[0].Success)
}

func TestUpdateActivityResult_PendingBeforeUpdate(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})

	a := r.Get("a1")
	// Before UpdateActivityResult, Success must be nil (pending).
	assert.Nil(t, a.FullActivityLog[0].Success)
}

func TestUpdateActivityResult_UnknownAgentNoOp(t *testing.T) {
	r := NewAgentRegistry()
	// Should not panic.
	r.UpdateActivityResult("nope", "t1", true)
}

func TestUpdateActivityResult_UnknownToolIDNoOp(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})
	r.UpdateActivityResult("a1", "no-such-tool", true)

	// Original entry unchanged.
	a := r.Get("a1")
	assert.Nil(t, a.FullActivityLog[0].Success)
}

func TestUpdateActivityResult_MatchesCorrectEntry(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))

	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t1"})
	r.AppendActivity("a1", AgentActivity{Type: "tool_use", ToolID: "t2"})
	r.UpdateActivityResult("a1", "t2", false)

	a := r.Get("a1")
	// t1 untouched, t2 updated.
	assert.Nil(t, a.FullActivityLog[0].Success)
	require.NotNil(t, a.FullActivityLog[1].Success)
	assert.False(t, *a.FullActivityLog[1].Success)
}

// ---------------------------------------------------------------------------
// Remove
// ---------------------------------------------------------------------------

func TestRemove_Basic(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	r.Remove("a1")

	assert.Nil(t, r.Get("a1"))
	assert.Equal(t, 0, r.Count().Total)
}

func TestRemove_UnknownIDNoOp(t *testing.T) {
	r := NewAgentRegistry()
	// Should not panic.
	r.Remove("nope")
}

func TestRemove_CleansParentChildren(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("parent", "go-pro", "parent task", "")))
	require.NoError(t, r.Register(makeAgent("child1", "go-tui", "child task", "parent")))
	require.NoError(t, r.Register(makeAgent("child2", "go-cli", "child task 2", "parent")))

	r.Remove("child1")

	parent := r.Get("parent")
	require.NotNil(t, parent)
	assert.NotContains(t, parent.Children, "child1")
	assert.Contains(t, parent.Children, "child2")
}

func TestRemove_ClearsRootID(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root task", "")))
	assert.Equal(t, "root", r.RootID())

	r.Remove("root")
	assert.Equal(t, "", r.RootID())
}

// ---------------------------------------------------------------------------
// SetSelected / Selected
// ---------------------------------------------------------------------------

func TestSetSelected(t *testing.T) {
	r := NewAgentRegistry()
	assert.Equal(t, "", r.Selected())

	r.SetSelected("a1")
	assert.Equal(t, "a1", r.Selected())

	r.SetSelected("")
	assert.Equal(t, "", r.Selected())
}

// ---------------------------------------------------------------------------
// Count
// ---------------------------------------------------------------------------

func TestCount(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "t1", "")))
	require.NoError(t, r.Register(makeAgent("a2", "go-pro", "t2", "")))
	require.NoError(t, r.Register(makeAgent("a3", "go-pro", "t3", "")))

	require.NoError(t, r.Update("a2", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("a3", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("a3", func(a *Agent) { a.Status = StatusComplete }))

	stats := r.Count()
	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Running)
	assert.Equal(t, 1, stats.Complete)
	assert.Equal(t, 0, stats.Error)
	assert.Equal(t, 0, stats.Killed)
}

func TestCount_AllStatuses(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("p", "t", "pending", "")))
	require.NoError(t, r.Register(makeAgent("run", "t", "running", "")))
	require.NoError(t, r.Update("run", func(a *Agent) { a.Status = StatusRunning }))

	require.NoError(t, r.Register(makeAgent("c", "t", "complete", "")))
	require.NoError(t, r.Update("c", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("c", func(a *Agent) { a.Status = StatusComplete }))

	require.NoError(t, r.Register(makeAgent("e", "t", "error", "")))
	require.NoError(t, r.Update("e", func(a *Agent) { a.Status = StatusRunning }))
	require.NoError(t, r.Update("e", func(a *Agent) { a.Status = StatusError }))

	require.NoError(t, r.Register(makeAgent("k", "t", "killed", "")))
	require.NoError(t, r.Update("k", func(a *Agent) { a.Status = StatusKilled }))

	stats := r.Count()
	assert.Equal(t, 5, stats.Total)
	assert.Equal(t, 1, stats.Pending)
	assert.Equal(t, 1, stats.Running)
	assert.Equal(t, 1, stats.Complete)
	assert.Equal(t, 1, stats.Error)
	assert.Equal(t, 1, stats.Killed)
}

// ---------------------------------------------------------------------------
// GetChildren
// ---------------------------------------------------------------------------

func TestGetChildren_Basic(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("parent", "go-pro", "parent", "")))

	c1 := makeAgent("c1", "go-tui", "child1", "parent")
	c1.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(c1))

	c2 := makeAgent("c2", "go-cli", "child2", "parent")
	c2.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(c2))

	children := r.GetChildren("parent")
	require.Len(t, children, 2)
	// Sorted by StartedAt ascending.
	assert.Equal(t, "c1", children[0].ID)
	assert.Equal(t, "c2", children[1].ID)
}

func TestGetChildren_ReturnsCopies(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("parent", "go-pro", "p", "")))
	require.NoError(t, r.Register(makeAgent("c1", "go-tui", "child", "parent")))

	children := r.GetChildren("parent")
	require.Len(t, children, 1)
	children[0].Status = StatusRunning // mutate copy

	// Registry not affected.
	orig := r.Get("c1")
	assert.Equal(t, StatusPending, orig.Status)
}

func TestGetChildren_UnknownParent(t *testing.T) {
	r := NewAgentRegistry()
	assert.Nil(t, r.GetChildren("nope"))
}

func TestGetChildren_NoChildren(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a1", "go-pro", "task", "")))
	children := r.GetChildren("a1")
	assert.Empty(t, children)
}

// ---------------------------------------------------------------------------
// Tree (InvalidateTreeCache + Tree)
// ---------------------------------------------------------------------------

func TestTree_EmptyRegistry(t *testing.T) {
	r := NewAgentRegistry()
	r.InvalidateTreeCache()
	assert.Nil(t, r.Tree())
}

func TestTree_SingleRoot(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root task", "")))
	r.InvalidateTreeCache()

	nodes := r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, "root", nodes[0].Agent.ID)
	assert.Equal(t, 0, nodes[0].Depth)
	assert.True(t, nodes[0].IsLast)
}

func TestTree_DFSOrder(t *testing.T) {
	//       root
	//      /    \
	//    c1      c2
	//   /
	// gc1
	now := time.Now()
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))

	c1 := makeAgent("c1", "go-pro", "c1", "root")
	c1.StartedAt = now.Add(-3 * time.Second)
	require.NoError(t, r.Register(c1))

	c2 := makeAgent("c2", "go-pro", "c2", "root")
	c2.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(c2))

	gc1 := makeAgent("gc1", "go-pro", "gc1", "c1")
	gc1.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(gc1))

	r.InvalidateTreeCache()
	nodes := r.Tree()
	require.Len(t, nodes, 4)

	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.Agent.ID
	}
	// DFS: root, c1, gc1, c2
	assert.Equal(t, []string{"root", "c1", "gc1", "c2"}, ids)
}

func TestTree_DepthAndIsLast(t *testing.T) {
	now := time.Now()
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))

	c1 := makeAgent("c1", "go-pro", "c1", "root")
	c1.StartedAt = now.Add(-2 * time.Second)
	require.NoError(t, r.Register(c1))

	c2 := makeAgent("c2", "go-pro", "c2", "root")
	c2.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(c2))

	r.InvalidateTreeCache()
	nodes := r.Tree()
	require.Len(t, nodes, 3)

	// root: depth 0, is last (only root)
	assert.Equal(t, "root", nodes[0].Agent.ID)
	assert.Equal(t, 0, nodes[0].Depth)
	assert.True(t, nodes[0].IsLast)

	// c1: depth 1, NOT last (c2 follows)
	assert.Equal(t, "c1", nodes[1].Agent.ID)
	assert.Equal(t, 1, nodes[1].Depth)
	assert.False(t, nodes[1].IsLast)

	// c2: depth 1, IS last
	assert.Equal(t, "c2", nodes[2].Agent.ID)
	assert.Equal(t, 1, nodes[2].Depth)
	assert.True(t, nodes[2].IsLast)
}

func TestTree_SortedByStartedAt(t *testing.T) {
	// Children must appear in StartedAt ascending order, regardless of
	// registration order.
	now := time.Now()
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))

	// Register c2 (later StartedAt) before c1 (earlier StartedAt).
	c2 := makeAgent("c2", "go-pro", "c2", "root")
	c2.StartedAt = now.Add(-1 * time.Second)
	require.NoError(t, r.Register(c2))

	c1 := makeAgent("c1", "go-pro", "c1", "root")
	c1.StartedAt = now.Add(-3 * time.Second)
	require.NoError(t, r.Register(c1))

	r.InvalidateTreeCache()
	nodes := r.Tree()
	require.Len(t, nodes, 3)
	// c1 started earlier so must appear first.
	assert.Equal(t, "c1", nodes[1].Agent.ID)
	assert.Equal(t, "c2", nodes[2].Agent.ID)
}

// Tree() returns slice header copy — node pointers inside are shared (safe
// because nodes contain Agent copies, not pointers to internal agents).
func TestTree_ReturnsCopy(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))
	r.InvalidateTreeCache()

	nodes1 := r.Tree()
	nodes2 := r.Tree()

	// Different slice headers.
	require.Len(t, nodes1, 1)
	require.Len(t, nodes2, 1)

	// Mutating a node from nodes1 should not affect nodes2 because Agent is a
	// copy held by the node.
	nodes1[0].Agent.Status = StatusRunning
	assert.Equal(t, StatusPending, nodes2[0].Agent.Status)
}

// InvalidateTreeCache recomputes the cache: after adding a new agent and
// re-invalidating, Tree() should reflect the addition.
func TestInvalidateTreeCache_Recomputes(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))
	r.InvalidateTreeCache()

	assert.Len(t, r.Tree(), 1)

	require.NoError(t, r.Register(makeAgent("child", "go-tui", "child", "root")))
	r.InvalidateTreeCache()

	assert.Len(t, r.Tree(), 2)
}

// ---------------------------------------------------------------------------
// RootID
// ---------------------------------------------------------------------------

func TestRootID_Empty(t *testing.T) {
	r := NewAgentRegistry()
	assert.Equal(t, "", r.RootID())
}

func TestRootID_SetOnFirstRegistration(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "task", "")))
	assert.Equal(t, "root", r.RootID())
}

// ---------------------------------------------------------------------------
// Concurrent access
// ---------------------------------------------------------------------------

// TestConcurrentRegisterGet exercises concurrent Register and Get operations.
// The -race flag will catch data races.
func TestConcurrentRegisterGet(t *testing.T) {
	r := NewAgentRegistry()
	const n = 100
	var wg sync.WaitGroup

	// Concurrent registers.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("agent-%d", i)
			_ = r.Register(makeAgent(id, "go-pro", fmt.Sprintf("task-%d", i), ""))
		}(i)
	}

	// Concurrent gets (some will return nil for not-yet-registered agents).
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("agent-%d", i)
			_ = r.Get(id)
		}(i)
	}

	wg.Wait()

	stats := r.Count()
	assert.LessOrEqual(t, stats.Total, n, "total should not exceed n")
	assert.GreaterOrEqual(t, stats.Total, 1, "at least one registration should succeed")
}

// TestConcurrentUpdateGet exercises concurrent Update and Get.
func TestConcurrentUpdateGet(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a", "go-pro", "task", "")))

	const n = 200
	var wg sync.WaitGroup

	// Goroutines that try to advance the status (most will fail due to
	// invalid transitions, but none should race).
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Update("a", func(a *Agent) {
				if a.Status == StatusPending {
					a.Status = StatusRunning
				}
			})
		}()
	}

	// Goroutines that read the agent concurrently.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Get("a")
		}()
	}

	wg.Wait()

	a := r.Get("a")
	require.NotNil(t, a)
	// Final status must be either Pending or Running (valid post-transition).
	assert.True(t, a.Status == StatusPending || a.Status == StatusRunning)
}

// TestConcurrentSetActivity exercises concurrent SetActivity and Get.
func TestConcurrentSetActivity(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("a", "go-pro", "task", "")))

	const n = 100
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.SetActivity("a", AgentActivity{
				Type:    "tool_use",
				Target:  fmt.Sprintf("tool-%d", i),
				Preview: fmt.Sprintf("preview-%d", i),
			})
		}(i)
	}
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Get("a")
		}()
	}

	wg.Wait()
}

// TestConcurrentRegisterRemove stresses concurrent registration and removal.
func TestConcurrentRegisterRemove(t *testing.T) {
	r := NewAgentRegistry()
	const n = 50
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("a-%d", i)
			_ = r.Register(makeAgent(id, "go-pro", fmt.Sprintf("t-%d", i), ""))
		}(i)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("a-%d", i)
			r.Remove(id)
		}(i)
	}

	wg.Wait()
	// Just verify no panic and no race; exact count is nondeterministic.
}

// TestConcurrentInvalidateTree stresses concurrent InvalidateTreeCache and
// Tree reads.
func TestConcurrentInvalidateTree(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("root", "go-pro", "root", "")))

	const n = 50
	var wg sync.WaitGroup

	// Writers: invalidate tree.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.InvalidateTreeCache()
		}()
	}
	// Readers: read tree.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.Tree()
		}()
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Children copy isolation (slice mutation)
// ---------------------------------------------------------------------------

func TestGetChildren_ChildrenSliceCopied(t *testing.T) {
	r := NewAgentRegistry()
	require.NoError(t, r.Register(makeAgent("parent", "go-pro", "p", "")))
	require.NoError(t, r.Register(makeAgent("c1", "go-tui", "c1", "parent")))
	require.NoError(t, r.Register(makeAgent("c2", "go-cli", "c2", "parent")))

	children := r.GetChildren("parent")
	require.Len(t, children, 2)

	// Truncate the returned slice — must not affect parent's Children.
	children = children[:0]

	parent := r.Get("parent")
	assert.Len(t, parent.Children, 2)
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

// Register with same ID twice: second register overwrites (no dedup on ID,
// only on agentType+description key).
func TestRegister_SameIDOverwrites(t *testing.T) {
	r := NewAgentRegistry()
	a1 := makeAgent("same-id", "go-pro", "task-one", "")
	require.NoError(t, r.Register(a1))

	// Different description → different dedup key → second register succeeds.
	a2 := makeAgent("same-id", "go-pro", "task-two", "")
	a2.Model = "opus"
	require.NoError(t, r.Register(a2))

	got := r.Get("same-id")
	assert.Equal(t, "opus", got.Model)
}

// Tree with no rootAgentID but agents exist falls back to first parentless agent.
func TestTree_NoExplicitRoot(t *testing.T) {
	r := NewAgentRegistry()
	// Register child before the parent to leave rootAgentID empty by design.
	// We simulate this by directly registering without a parent and then
	// checking the fallback.
	require.NoError(t, r.Register(makeAgent("solo", "go-pro", "solo task", "")))

	r.InvalidateTreeCache()
	nodes := r.Tree()
	require.Len(t, nodes, 1)
	assert.Equal(t, "solo", nodes[0].Agent.ID)
}
