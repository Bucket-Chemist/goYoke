// Package agents exports internal symbols for use in external tests (_test
// package). This file is compiled only during testing.
package agents

// NodeHeight exposes the unexported nodeHeight method for external tests.
func (m AgentTreeModel) NodeHeight(idx int) int {
	return m.nodeHeight(idx)
}
