// Package agents_test provides tests for AgentTreeModel.Search (TUI-059).
package agents_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/components/agents"
	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/state"
)

// buildTree builds an AgentTreeModel pre-loaded with the given nodes.
func buildTree(nodes []*state.AgentTreeNode) *agents.AgentTreeModel {
	m := agents.NewAgentTreeModel()
	m.SetNodes(nodes)
	return &m
}

// ---------------------------------------------------------------------------
// AgentTreeModel.Search tests
// ---------------------------------------------------------------------------

func TestAgentTreeSearch_EmptyQueryReturnsNil(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "go-pro", "implement feature", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("")
	assert.Nil(t, results)
}

func TestAgentTreeSearch_NoMatchReturnsEmpty(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "go-pro", "implement feature", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("zzz_no_match")
	assert.Empty(t, results)
}

func TestAgentTreeSearch_AgentTypeMatch(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "go-pro", "implement feature", state.StatusRunning), 0, false),
		makeNode(makeAgent("a2", "", "python-pro", "data analysis", state.StatusComplete), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("go-pro")
	require.Len(t, results, 1)
	assert.Equal(t, "go-pro", results[0].Label)
	assert.Equal(t, "agents", results[0].Source)
}

func TestAgentTreeSearch_DescriptionMatch(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "go-pro", "implement the feature", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("feature")
	require.Len(t, results, 1)
	assert.Equal(t, "agents", results[0].Source)
	assert.Equal(t, "go-pro", results[0].Label)
	assert.Equal(t, "implement the feature", results[0].Detail)
}

func TestAgentTreeSearch_CaseInsensitive(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "GO-PRO", "IMPLEMENT", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("go-pro")
	require.NotEmpty(t, results)
}

func TestAgentTreeSearch_NameMatchScoresHigherThanDescMatch(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		// Name contains query.
		makeNode(makeAgent("a1", "", "go-tui", "build interface", state.StatusRunning), 0, false),
		// Only description contains query.
		makeNode(makeAgent("a2", "", "python-pro", "build go-style code", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("go")
	require.Len(t, results, 2)
	// go-tui matches by name → higher score.
	nameMatchResult := findAgentResult(results, "go-tui")
	descMatchResult := findAgentResult(results, "python-pro")
	require.NotNil(t, nameMatchResult, "name match must appear")
	require.NotNil(t, descMatchResult, "description match must appear")
	assert.Greater(t, nameMatchResult.Score, descMatchResult.Score,
		"name match must score higher than desc-only match")
}

func TestAgentTreeSearch_EmptyTreeReturnsNil(t *testing.T) {
	m := agents.NewAgentTreeModel()
	results := m.Search("anything")
	assert.Nil(t, results)
}

func TestAgentTreeSearch_MultipleMatches(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		makeNode(makeAgent("a1", "", "go-pro", "implement feature", state.StatusRunning), 0, false),
		makeNode(makeAgent("a2", "", "go-tui", "build go tui", state.StatusRunning), 0, false),
		makeNode(makeAgent("a3", "", "python-pro", "python analysis", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("go")
	// go-pro, go-tui match by name; "build go tui" also matches by desc.
	// go-tui matches BOTH name and description.
	assert.GreaterOrEqual(t, len(results), 2)
}

func TestAgentTreeSearch_BothNameAndDescMatch_OnlyOneResultPerNode(t *testing.T) {
	nodes := []*state.AgentTreeNode{
		// Both agentType and description contain "go".
		makeNode(makeAgent("a1", "", "go-pro", "build go application", state.StatusRunning), 0, true),
	}
	m := buildTree(nodes)
	results := m.Search("go")
	// Should produce exactly one result per matching node.
	assert.Len(t, results, 1)
}

// findAgentResult returns the first SearchResult whose Label equals label.
func findAgentResult(results []state.SearchResult, label string) *state.SearchResult {
	for i := range results {
		if results[i].Label == label {
			return &results[i]
		}
	}
	return nil
}
