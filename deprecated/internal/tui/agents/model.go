package agents

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/pkg/telemetry"
)

// AgentStatus represents the lifecycle state of an agent
type AgentStatus string

const (
	StatusSpawning  AgentStatus = "spawning"
	StatusRunning   AgentStatus = "running"
	StatusCompleted AgentStatus = "completed"
	StatusError     AgentStatus = "error"
)

// AgentNode represents a single agent in the delegation tree
type AgentNode struct {
	// Identity
	AgentID   string
	ParentID  string
	SessionID string

	// Metadata
	Tier        string // "haiku", "sonnet", "opus"
	Description string
	DecisionID  string // Correlation with routing decisions

	// Lifecycle
	SpawnEvent    *telemetry.AgentLifecycleEvent
	CompleteEvent *telemetry.AgentLifecycleEvent
	Status        AgentStatus

	// Timing
	SpawnTime    time.Time
	CompleteTime *time.Time
	Duration     *time.Duration

	// Tree structure
	Children []*AgentNode
}

// IsActive returns true if agent is spawning or running
func (an *AgentNode) IsActive() bool {
	return an.Status == StatusSpawning || an.Status == StatusRunning
}

// GetDuration returns the duration of the agent (calculated or elapsed)
func (an *AgentNode) GetDuration() time.Duration {
	if an.Duration != nil {
		return *an.Duration
	}
	if an.CompleteTime != nil {
		return an.CompleteTime.Sub(an.SpawnTime)
	}
	return time.Since(an.SpawnTime)
}

// AddChild appends a child node to this node's children
func (an *AgentNode) AddChild(child *AgentNode) {
	an.Children = append(an.Children, child)
}

// AgentTree tracks all agents in the current session as a hierarchical tree
type AgentTree struct {
	Root      *AgentNode            // Root of tree (main session)
	nodes     map[string]*AgentNode // Index by agent ID for O(1) lookups
	mu        sync.RWMutex
	orphans   []*AgentNode // Nodes whose parent hasn't appeared yet
	sessionID string

	// Statistics
	TotalAgents     int
	ActiveAgents    int
	CompletedAgents int
	ErroredAgents   int
}

// NewAgentTree creates a new agent tree for the given session
func NewAgentTree(sessionID string) *AgentTree {
	return &AgentTree{
		Root:      nil,
		nodes:     make(map[string]*AgentNode),
		orphans:   make([]*AgentNode, 0),
		sessionID: sessionID,
	}
}

// ProcessSpawn handles a spawn lifecycle event
func (at *AgentTree) ProcessSpawn(event *telemetry.AgentLifecycleEvent) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	// Check if already exists (idempotent)
	if _, exists := at.nodes[event.AgentID]; exists {
		return nil // Already processed
	}

	// Create node
	spawnTime := time.Unix(event.Timestamp, 0)
	node := &AgentNode{
		AgentID:     event.AgentID,
		ParentID:    event.ParentAgent,
		SessionID:   event.SessionID,
		Tier:        event.Tier,
		Description: event.TaskDescription,
		DecisionID:  event.DecisionID,
		SpawnEvent:  event,
		Status:      StatusSpawning,
		SpawnTime:   spawnTime,
		Children:    make([]*AgentNode, 0),
	}

	// Index
	at.nodes[event.AgentID] = node

	// Attach to parent or root
	if event.ParentAgent == "" {
		// Root agent
		if at.Root == nil {
			at.Root = node
		} else {
			// Multiple root-level agents - attach to root as siblings
			at.Root.AddChild(node)
		}
	} else {
		parent, exists := at.nodes[event.ParentAgent]
		if exists {
			parent.AddChild(node)
			// Parent has spawned a child, transition to running if spawning
			if parent.Status == StatusSpawning {
				parent.Status = StatusRunning
			}
		} else {
			// Orphaned node - parent not found yet
			at.orphans = append(at.orphans, node)
		}
	}

	// Try to attach orphans if this node was their parent
	at.attachOrphans(event.AgentID)

	// Update stats
	at.TotalAgents++
	at.ActiveAgents++

	return nil
}

// attachOrphans tries to attach orphaned nodes to a newly added parent
func (at *AgentTree) attachOrphans(parentID string) {
	parent, exists := at.nodes[parentID]
	if !exists {
		return
	}

	remaining := make([]*AgentNode, 0)
	for _, orphan := range at.orphans {
		if orphan.ParentID == parentID {
			parent.AddChild(orphan)
			// Parent has children, ensure it's running
			if parent.Status == StatusSpawning {
				parent.Status = StatusRunning
			}
		} else {
			remaining = append(remaining, orphan)
		}
	}
	at.orphans = remaining
}

// ProcessComplete handles a complete lifecycle event
func (at *AgentTree) ProcessComplete(event *telemetry.AgentLifecycleEvent) error {
	at.mu.Lock()
	defer at.mu.Unlock()

	node, exists := at.nodes[event.AgentID]
	if !exists {
		// Complete event before spawn (shouldn't happen but handle gracefully)
		return fmt.Errorf("complete event for unknown agent: %s", event.AgentID)
	}

	// Update node
	node.CompleteEvent = event
	completeTime := time.Unix(event.Timestamp, 0)
	node.CompleteTime = &completeTime

	// Determine status
	if event.Success != nil && !*event.Success {
		node.Status = StatusError
		at.ErroredAgents++
	} else {
		node.Status = StatusCompleted
		at.CompletedAgents++
	}

	// Calculate duration
	if event.DurationMs != nil {
		duration := time.Duration(*event.DurationMs) * time.Millisecond
		node.Duration = &duration
	} else {
		duration := completeTime.Sub(node.SpawnTime)
		node.Duration = &duration
	}

	// Update stats
	at.ActiveAgents--

	return nil
}

// GetNode returns the node for the given agent ID
func (at *AgentTree) GetNode(agentID string) (*AgentNode, bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	node, exists := at.nodes[agentID]
	return node, exists
}

// GetChildren returns all child nodes of the given agent
func (at *AgentTree) GetChildren(agentID string) []*AgentNode {
	at.mu.RLock()
	defer at.mu.RUnlock()

	node, exists := at.nodes[agentID]
	if !exists {
		return nil
	}

	// Return copy to prevent external modification
	children := make([]*AgentNode, len(node.Children))
	copy(children, node.Children)
	return children
}

// GetActiveAgents returns all agents with active status
func (at *AgentTree) GetActiveAgents() []*AgentNode {
	at.mu.RLock()
	defer at.mu.RUnlock()

	active := make([]*AgentNode, 0)
	for _, node := range at.nodes {
		if node.IsActive() {
			active = append(active, node)
		}
	}
	return active
}

// WalkTree performs a depth-first traversal of the tree
// The function fn is called for each node. If fn returns false, traversal stops.
func (at *AgentTree) WalkTree(fn func(*AgentNode) bool) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	if at.Root == nil {
		return
	}

	var walk func(*AgentNode) bool
	walk = func(node *AgentNode) bool {
		// Visit node
		if !fn(node) {
			return false // Stop traversal
		}

		// Visit children
		for _, child := range node.Children {
			if !walk(child) {
				return false
			}
		}

		return true
	}

	walk(at.Root)
}

// TreeNode represents a serializable tree node for JSON export
type TreeNode struct {
	AgentID     string      `json:"agent_id"`
	ParentID    string      `json:"parent_id"`
	Tier        string      `json:"tier"`
	Description string      `json:"description"`
	Status      AgentStatus `json:"status"`
	SpawnTime   int64       `json:"spawn_time"`
	Duration    *int64      `json:"duration_ms,omitempty"`
	Children    []TreeNode  `json:"children"`
}

// ToJSON serializes the tree to JSON
func (at *AgentTree) ToJSON() ([]byte, error) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	if at.Root == nil {
		return json.Marshal(map[string]interface{}{
			"session_id": at.sessionID,
			"tree":       nil,
			"stats": map[string]int{
				"total":     at.TotalAgents,
				"active":    at.ActiveAgents,
				"completed": at.CompletedAgents,
				"errored":   at.ErroredAgents,
			},
		})
	}

	var buildTreeNode func(*AgentNode) TreeNode
	buildTreeNode = func(node *AgentNode) TreeNode {
		tn := TreeNode{
			AgentID:     node.AgentID,
			ParentID:    node.ParentID,
			Tier:        node.Tier,
			Description: node.Description,
			Status:      node.Status,
			SpawnTime:   node.SpawnTime.Unix(),
			Children:    make([]TreeNode, 0, len(node.Children)),
		}

		if node.Duration != nil {
			durationMs := node.Duration.Milliseconds()
			tn.Duration = &durationMs
		}

		for _, child := range node.Children {
			tn.Children = append(tn.Children, buildTreeNode(child))
		}

		return tn
	}

	result := map[string]interface{}{
		"session_id": at.sessionID,
		"tree":       buildTreeNode(at.Root),
		"stats": map[string]int{
			"total":     at.TotalAgents,
			"active":    at.ActiveAgents,
			"completed": at.CompletedAgents,
			"errored":   at.ErroredAgents,
		},
	}

	return json.Marshal(result)
}

// GetStats returns current tree statistics
func (at *AgentTree) GetStats() TreeStats {
	at.mu.RLock()
	defer at.mu.RUnlock()

	return TreeStats{
		TotalAgents:     at.TotalAgents,
		ActiveAgents:    at.ActiveAgents,
		CompletedAgents: at.CompletedAgents,
		ErroredAgents:   at.ErroredAgents,
		OrphanedNodes:   len(at.orphans),
	}
}

// TreeStats provides statistics about the agent tree
type TreeStats struct {
	TotalAgents     int
	ActiveAgents    int
	CompletedAgents int
	ErroredAgents   int
	OrphanedNodes   int
}
