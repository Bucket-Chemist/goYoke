// Package state provides shared, thread-safe state containers for the
// GOgent-Fortress TUI. It has no dependency on the model, cli, bridge, or any
// Bubbletea packages, keeping the import graph acyclic.
package state

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// AgentStatus
// ---------------------------------------------------------------------------

// AgentStatus is the lifecycle state of a single agent process.
type AgentStatus int

const (
	// StatusPending means the agent has been registered but has not started.
	StatusPending AgentStatus = iota
	// StatusRunning means the agent is actively executing.
	StatusRunning
	// StatusComplete means the agent finished successfully.
	StatusComplete
	// StatusError means the agent finished with an error.
	StatusError
	// StatusKilled means the agent was forcibly terminated.
	StatusKilled
)

// String returns a human-readable name for the status.
func (s AgentStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusComplete:
		return "complete"
	case StatusError:
		return "error"
	case StatusKilled:
		return "killed"
	default:
		return fmt.Sprintf("AgentStatus(%d)", int(s))
	}
}

// validTransitions maps each non-terminal status to the set of statuses it
// may transition into. Terminal statuses (Complete, Error, Killed) are absent,
// meaning no further transitions are allowed from them.
var validTransitions = map[AgentStatus][]AgentStatus{
	StatusPending: {StatusRunning, StatusKilled},
	StatusRunning: {StatusComplete, StatusError, StatusKilled},
}

// isValidTransition returns true when transitioning from "from" to "to" is
// allowed. Terminal statuses always return false (no outgoing transitions).
func isValidTransition(from, to AgentStatus) bool {
	if from == to {
		return false
	}
	allowed, ok := validTransitions[from]
	if !ok {
		// Terminal status — no transitions permitted.
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// AgentActivity
// ---------------------------------------------------------------------------

// AgentActivity describes what an agent is currently doing at a fine-grained
// level (e.g. which tool it is invoking).
type AgentActivity struct {
	// Type classifies the activity, e.g. "tool_use" or "thinking".
	Type string
	// Target is the subject of the activity, e.g. the tool name.
	Target string
	// Preview is a short human-readable summary suitable for one-line display.
	Preview string
	// Timestamp records when this activity started.
	Timestamp time.Time
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent is the canonical representation of a single spawned agent process
// tracked by the TUI.
type Agent struct {
	// ID is the unique identifier for this agent instance (UUID or canonical ID).
	ID string
	// ParentID is the ID of the parent agent. Empty for root agents.
	ParentID string
	// AgentType is the agent kind, e.g. "go-pro" or "einstein".
	AgentType string
	// Description is a short human-readable label for this agent invocation.
	Description string
	// Model is the LLM model name, e.g. "sonnet" or "opus".
	Model string
	// Tier classifies the cost tier, e.g. "haiku", "sonnet", "opus".
	Tier string
	// Status is the current lifecycle state.
	Status AgentStatus
	// Activity is the most recent fine-grained activity, or nil when idle.
	Activity *AgentActivity
	// StartedAt is the wall-clock time when the agent began executing.
	StartedAt time.Time
	// Duration is the elapsed time for a completed agent; zero for in-progress.
	Duration time.Duration
	// Cost is the estimated USD cost of this agent's execution so far.
	Cost float64
	// Tokens is the total token count consumed by this agent so far.
	Tokens int
	// ErrorOutput holds the captured stderr/error text for StatusError agents.
	ErrorOutput string
	// Children lists the IDs of direct child agents spawned by this agent.
	Children []string
	// Conventions lists the convention files loaded for this agent (e.g. "go.md").
	Conventions []string
	// Prompt is the augmented prompt sent to the agent (may be truncated).
	Prompt string
}

// dedupKey returns the deduplication key for this agent: agentType + ":" + description.
func (a *Agent) dedupKey() string {
	return a.AgentType + ":" + a.Description
}

// copyOf returns a shallow copy of the Agent value. The Children slice is
// duplicated so callers cannot mutate the registry's internal slice.
func (a *Agent) copyOf() Agent {
	cp := *a
	if a.Children != nil {
		cp.Children = make([]string, len(a.Children))
		copy(cp.Children, a.Children)
	}
	if a.Conventions != nil {
		cp.Conventions = make([]string, len(a.Conventions))
		copy(cp.Conventions, a.Conventions)
	}
	if a.Activity != nil {
		act := *a.Activity
		cp.Activity = &act
	}
	return cp
}

// ---------------------------------------------------------------------------
// AgentTreeNode
// ---------------------------------------------------------------------------

// AgentTreeNode is a single entry in the flat DFS-ordered tree projection used
// by View rendering. It carries a copy of the Agent and the information needed
// to draw tree-connector glyphs.
type AgentTreeNode struct {
	// Agent is a copy of the agent at this position in the tree.
	Agent *Agent
	// Depth is the zero-based nesting level (0 = root).
	Depth int
	// IsLast is true when this node is the last child at its depth level.
	IsLast bool
}

// ---------------------------------------------------------------------------
// AgentStats
// ---------------------------------------------------------------------------

// AgentStats is a snapshot of agent counts grouped by status.
type AgentStats struct {
	Total    int
	Running  int
	Complete int
	Error    int
	Pending  int
	Killed   int
}

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	// ErrAgentNotFound is returned when an operation targets an unknown agent ID.
	ErrAgentNotFound = fmt.Errorf("agent not found")
	// ErrDuplicateAgent is returned by Register when the dedup check fires.
	ErrDuplicateAgent = fmt.Errorf("duplicate agent: same agentType+description is already pending or running")
	// ErrInvalidTransition is returned by Update when a status transition is disallowed.
	ErrInvalidTransition = fmt.Errorf("invalid status transition")
)

// ---------------------------------------------------------------------------
// AgentRegistry
// ---------------------------------------------------------------------------

// AgentRegistry is a thread-safe store for all Agent records tracked by the
// TUI in a single session.
//
// The zero value is not usable; use NewAgentRegistry instead.
//
// Concurrency model:
//   - Write methods (Register, Update, Remove, SetActivity, SetSelected,
//     InvalidateTreeCache) acquire a full write lock (mu.Lock).
//   - Read methods (Get, GetChildren, Tree, Count, Selected, RootID) acquire
//     a shared read lock (mu.RLock).
//
// Tree cache discipline (Review M-3):
//   - InvalidateTreeCache recomputes the treeCache field and MUST be called
//     only from the Bubbletea Update() goroutine (i.e. after receiving an
//     AgentRegisteredMsg / AgentUpdatedMsg). Register() does NOT call it
//     directly, ensuring tree mutations are serialised through Bubbletea's
//     single-goroutine Update loop.
type AgentRegistry struct {
	agents      map[string]*Agent
	rootAgentID string
	selectedID  string
	treeCache   []*AgentTreeNode
	mu          sync.RWMutex
}

// NewAgentRegistry allocates and returns an empty AgentRegistry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*Agent),
	}
}

// ---------------------------------------------------------------------------
// Write methods
// ---------------------------------------------------------------------------

// Register adds a new agent to the registry.
//
// Deduplication: if an agent with the same (agentType + ":" + description) key
// already exists and is in StatusPending or StatusRunning, Register returns
// ErrDuplicateAgent and does not modify the registry.
//
// Parent linkage: if agent.ParentID is non-empty and the parent exists, the
// new agent's ID is appended to the parent's Children slice.
//
// Root tracking: the first agent with an empty ParentID is recorded as the
// root agent ID. Subsequent root-level agents do not overwrite the first root.
//
// NOTE (Review M-3): Register does NOT call InvalidateTreeCache. The caller
// must send an AgentRegisteredMsg through the Bubbletea event loop; the
// Update() handler is responsible for calling InvalidateTreeCache after
// processing that message.
func (r *AgentRegistry) Register(agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Deduplication check.
	key := agent.dedupKey()
	for _, existing := range r.agents {
		if existing.dedupKey() == key {
			if existing.Status == StatusPending || existing.Status == StatusRunning {
				return fmt.Errorf("register agent %q: %w", agent.ID, ErrDuplicateAgent)
			}
		}
	}

	// Store a pointer to a copy of the caller's value.
	cp := agent.copyOf()
	r.agents[cp.ID] = &cp

	// Record root agent (first agent without a parent).
	if cp.ParentID == "" && r.rootAgentID == "" {
		r.rootAgentID = cp.ID
	}

	// Append to parent's Children slice.
	if cp.ParentID != "" {
		if parent, ok := r.agents[cp.ParentID]; ok {
			parent.Children = append(parent.Children, cp.ID)
		}
	}

	return nil
}

// Update applies fn to the agent identified by id under a write lock.
//
// Status transition validation: if fn changes the agent's Status field,
// Update verifies the transition is allowed according to validTransitions. If
// the transition is invalid, the change is reverted and ErrInvalidTransition
// is returned (wrapped with context).
//
// NOTE (Review M-3): Update may be called from any goroutine that holds a
// reference to the registry, but callers should prefer driving mutations
// through Bubbletea messages so that InvalidateTreeCache is called in the
// correct goroutine.
func (r *AgentRegistry) Update(id string, fn func(*Agent)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.agents[id]
	if !ok {
		return fmt.Errorf("update agent %q: %w", id, ErrAgentNotFound)
	}

	prevStatus := a.Status
	fn(a)
	newStatus := a.Status

	if newStatus != prevStatus {
		if !isValidTransition(prevStatus, newStatus) {
			// Revert the status change.
			a.Status = prevStatus
			return fmt.Errorf("update agent %q: %w: %s → %s",
				id, ErrInvalidTransition, prevStatus, newStatus)
		}
	}

	return nil
}

// SetActivity sets the current AgentActivity for the agent identified by id.
// If the agent does not exist, SetActivity is a no-op.
func (r *AgentRegistry) SetActivity(id string, activity AgentActivity) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if a, ok := r.agents[id]; ok {
		act := activity
		a.Activity = &act
	}
}

// Remove deletes the agent identified by id from the registry and removes its
// ID from its parent's Children slice. If the agent does not exist, Remove is
// a no-op.
func (r *AgentRegistry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.agents[id]
	if !ok {
		return
	}

	// Remove from parent's Children list.
	if a.ParentID != "" {
		if parent, exists := r.agents[a.ParentID]; exists {
			var filtered []string
			for _, childID := range parent.Children {
				if childID != id {
					filtered = append(filtered, childID)
				}
			}
			parent.Children = filtered
		}
	}

	// Clear root tracking if the root agent is being removed.
	if r.rootAgentID == id {
		r.rootAgentID = ""
	}

	delete(r.agents, id)
}

// SetSelected records the ID of the currently selected agent for UI
// highlighting. Passing an empty string deselects.
func (r *AgentRegistry) SetSelected(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.selectedID = id
}

// InvalidateTreeCache recomputes the treeCache from the current agents map.
//
// This method MUST be called only from the Bubbletea Update() goroutine.
// Calling it from other goroutines introduces a data race between the
// Bubbletea render goroutine (which reads treeCache via Tree()) and the
// mutation.
//
// The DFS traversal visits the root agent first, then recursively visits each
// agent's children sorted by StartedAt (ascending).
func (r *AgentRegistry) InvalidateTreeCache() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.treeCache = r.buildTree()
}

// buildTree performs a DFS traversal of the agent hierarchy and returns the
// ordered flat list of AgentTreeNode values. It must be called with r.mu held
// for writing (or during construction when no concurrent access exists).
func (r *AgentRegistry) buildTree() []*AgentTreeNode {
	if len(r.agents) == 0 {
		return nil
	}

	// Find the root agent.
	rootID := r.rootAgentID
	if rootID == "" {
		// Fall back: pick the agent whose ParentID is empty.
		for _, a := range r.agents {
			if a.ParentID == "" {
				rootID = a.ID
				break
			}
		}
	}

	if rootID == "" {
		return nil
	}

	root, ok := r.agents[rootID]
	if !ok {
		return nil
	}

	var result []*AgentTreeNode
	r.dfsAppend(root, 0, true, &result)
	return result
}

// dfsAppend recursively appends AgentTreeNode entries for agent and its
// descendants. isLast indicates whether agent is the last child in its
// parent's sorted child list.
func (r *AgentRegistry) dfsAppend(a *Agent, depth int, isLast bool, out *[]*AgentTreeNode) {
	cp := a.copyOf()
	*out = append(*out, &AgentTreeNode{
		Agent:  &cp,
		Depth:  depth,
		IsLast: isLast,
	})

	// Collect and sort children by StartedAt ascending.
	children := make([]*Agent, 0, len(a.Children))
	for _, childID := range a.Children {
		if child, ok := r.agents[childID]; ok {
			children = append(children, child)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartedAt.Before(children[j].StartedAt)
	})

	for i, child := range children {
		r.dfsAppend(child, depth+1, i == len(children)-1, out)
	}
}

// ---------------------------------------------------------------------------
// Read methods
// ---------------------------------------------------------------------------

// Get returns a copy of the agent with the given id, or the zero value of
// Agent if the id is not found. Callers receive a copy and cannot mutate
// internal state through the returned value.
func (r *AgentRegistry) Get(id string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, ok := r.agents[id]
	if !ok {
		return nil
	}
	cp := a.copyOf()
	return &cp
}

// GetChildren returns copies of all direct children of the agent identified by
// parentID. The returned slice is sorted by StartedAt ascending. An empty
// slice is returned if the parent has no children or does not exist.
func (r *AgentRegistry) GetChildren(parentID string) []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parent, ok := r.agents[parentID]
	if !ok {
		return nil
	}

	result := make([]*Agent, 0, len(parent.Children))
	for _, childID := range parent.Children {
		if child, exists := r.agents[childID]; exists {
			cp := child.copyOf()
			result = append(result, &cp)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})

	return result
}

// Tree returns the cached flat DFS-ordered tree projection. Each call returns
// a fresh slice whose AgentTreeNode entries contain independent Agent copies,
// so callers may safely read (but must not rely on writing to) the returned
// values without coordination.
//
// The cache is stale until InvalidateTreeCache is called. Tree() is safe to
// call from any goroutine.
func (r *AgentRegistry) Tree() []*AgentTreeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.treeCache == nil {
		return nil
	}

	// Deep-copy each node so the caller receives fully independent values.
	result := make([]*AgentTreeNode, len(r.treeCache))
	for i, cached := range r.treeCache {
		agentCopy := cached.Agent.copyOf()
		result[i] = &AgentTreeNode{
			Agent:  &agentCopy,
			Depth:  cached.Depth,
			IsLast: cached.IsLast,
		}
	}
	return result
}

// Count returns a snapshot of agent counts grouped by status.
func (r *AgentRegistry) Count() AgentStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var stats AgentStats
	stats.Total = len(r.agents)
	for _, a := range r.agents {
		switch a.Status {
		case StatusPending:
			stats.Pending++
		case StatusRunning:
			stats.Running++
		case StatusComplete:
			stats.Complete++
		case StatusError:
			stats.Error++
		case StatusKilled:
			stats.Killed++
		}
	}
	return stats
}

// Selected returns the currently selected agent ID, or an empty string when
// nothing is selected.
func (r *AgentRegistry) Selected() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.selectedID
}

// RootID returns the ID of the root agent (the first agent registered with an
// empty ParentID), or an empty string if no root has been registered yet.
func (r *AgentRegistry) RootID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rootAgentID
}
