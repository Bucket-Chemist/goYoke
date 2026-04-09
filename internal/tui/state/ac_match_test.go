package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// MatchTodosToAC
// ---------------------------------------------------------------------------

func TestMatchTodosToAC_PositionalExactMatch(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
		{Text: "write tests"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
		{Content: "write tests", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 2)
	assert.True(t, result[0].Completed)
	assert.True(t, result[1].Completed)
}

func TestMatchTodosToAC_PositionalNotCompleted(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "in-progress"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.False(t, result[0].Completed)
}

func TestMatchTodosToAC_FuzzyFallback(t *testing.T) {
	// positional text differs → fuzzy match by substring
	criteria := []AcceptanceCriterion{
		{Text: "feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A with tests", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "fuzzy match should mark completed")
}

func TestMatchTodosToAC_FuzzyFallback_ACContainsTodo(t *testing.T) {
	// AC text contains the todo content
	criteria := []AcceptanceCriterion{
		{Text: "implement the feature A and verify output"},
	}
	todos := []TodoUpdate{
		{Content: "implement the feature A and verify output something else", Status: "done"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	// "implement the feature A and verify output" is contained in the todo content
	assert.True(t, result[0].Completed)
}

func TestMatchTodosToAC_EmptyCriteria(t *testing.T) {
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
	}
	result := MatchTodosToAC(nil, todos)
	assert.Nil(t, result)

	result2 := MatchTodosToAC([]AcceptanceCriterion{}, todos)
	assert.Empty(t, result2)
}

func TestMatchTodosToAC_EmptyTodos(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
	}
	result := MatchTodosToAC(criteria, nil)
	// returned unchanged
	assert.Equal(t, criteria, result)

	result2 := MatchTodosToAC(criteria, []TodoUpdate{})
	assert.Equal(t, criteria, result2)
}

func TestMatchTodosToAC_MismatchedLengths_MoreTodosThanAC(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
		{Content: "write docs", Status: "completed"},
		{Content: "add tests", Status: "todo"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed)
}

func TestMatchTodosToAC_MismatchedLengths_MoreACThanTodos(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
		{Text: "write tests"},
		{Text: "update docs"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 3)
	assert.True(t, result[0].Completed, "positional match at index 0")
	assert.False(t, result[1].Completed, "no matching todo for index 1")
	assert.False(t, result[2].Completed, "no matching todo for index 2")
}

func TestMatchTodosToAC_AllCompleted(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "step one"},
		{Text: "step two"},
		{Text: "step three"},
	}
	todos := []TodoUpdate{
		{Content: "step one", Status: "completed"},
		{Content: "step two", Status: "done"},
		{Content: "step three", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 3)
	for i, ac := range result {
		assert.True(t, ac.Completed, "AC[%d] should be completed", i)
	}
}

func TestMatchTodosToAC_NoneCompleted(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "step one"},
		{Text: "step two"},
	}
	todos := []TodoUpdate{
		{Content: "step one", Status: "todo"},
		{Content: "step two", Status: "in-progress"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 2)
	for i, ac := range result {
		assert.False(t, ac.Completed, "AC[%d] should not be completed", i)
	}
}

func TestMatchTodosToAC_OriginalNotMutated(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A", Completed: false},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	assert.True(t, result[0].Completed)
	// original slice must not be mutated
	assert.False(t, criteria[0].Completed, "original criteria slice must not be modified")
}

func TestMatchTodosToAC_NoFuzzyMatchWhenTextUnrelated(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "completely unrelated criterion"},
	}
	todos := []TodoUpdate{
		{Content: "write docs", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.False(t, result[0].Completed, "no match should leave criterion incomplete")
}

// ---------------------------------------------------------------------------
// copyOf — AcceptanceCriteria deep copy isolation
// ---------------------------------------------------------------------------

func TestCopyOf_AcceptanceCriteriaIsolation(t *testing.T) {
	a := Agent{
		ID:        "agent-1",
		AgentType: "go-pro",
		AcceptanceCriteria: []AcceptanceCriterion{
			{Text: "criterion one", Completed: false},
			{Text: "criterion two", Completed: false},
		},
	}

	cp := a.copyOf()

	// Mutate the copy's slice.
	cp.AcceptanceCriteria[0].Completed = true
	cp.AcceptanceCriteria = append(cp.AcceptanceCriteria, AcceptanceCriterion{Text: "extra"})

	// Original must be unchanged.
	assert.False(t, a.AcceptanceCriteria[0].Completed, "original Completed must not be affected by copy mutation")
	assert.Len(t, a.AcceptanceCriteria, 2, "original slice length must not change")
}

func TestCopyOf_NilAcceptanceCriteria(t *testing.T) {
	a := Agent{ID: "agent-nil-ac"}
	cp := a.copyOf()
	assert.Nil(t, cp.AcceptanceCriteria)
}

// ---------------------------------------------------------------------------
// UpdateAcceptanceCriteria — registry write method
// ---------------------------------------------------------------------------

func TestUpdateAcceptanceCriteria_Basic(t *testing.T) {
	r := NewAgentRegistry()
	a := makeAgent("agent-1", "go-pro", "test task", "")
	a.AcceptanceCriteria = []AcceptanceCriterion{
		{Text: "implement feature"},
		{Text: "write tests"},
	}
	require.NoError(t, r.Register(a))

	r.UpdateAcceptanceCriteria("agent-1", []TodoUpdate{
		{Content: "implement feature", Status: "completed"},
		{Content: "write tests", Status: "todo"},
	})

	got := r.Get("agent-1")
	require.NotNil(t, got)
	assert.True(t, got.AcceptanceCriteria[0].Completed)
	assert.False(t, got.AcceptanceCriteria[1].Completed)
}

func TestUpdateAcceptanceCriteria_UnknownAgent_NoOp(t *testing.T) {
	r := NewAgentRegistry()
	// Should not panic or error for unknown agent.
	r.UpdateAcceptanceCriteria("ghost", []TodoUpdate{
		{Content: "some todo", Status: "completed"},
	})
}

// ---------------------------------------------------------------------------
// Pass 2: case-insensitive substring matching
// ---------------------------------------------------------------------------

func TestMatchTodosToAC_CaseInsensitiveSubstring(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "Implement Feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature a with tests", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "case-insensitive substring should match")
}

func TestMatchTodosToAC_CaseInsensitiveSubstring_NotCompleted(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "Implement Feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature a with tests", Status: "in-progress"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.False(t, result[0].Completed, "case-insensitive match with non-completed status should not complete AC")
}

// ---------------------------------------------------------------------------
// Pass 3: word-overlap scoring
// ---------------------------------------------------------------------------

func TestMatchTodosToAC_WordOverlap_Paraphrase(t *testing.T) {
	// Positional and substring passes won't fire; word overlap should.
	criteria := []AcceptanceCriterion{
		{Text: "analyze routing architecture performance bottlenecks"},
	}
	todos := []TodoUpdate{
		{Content: "routing architecture performance bottlenecks analysis", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "word overlap should match paraphrased content")
}

func TestMatchTodosToAC_WordOverlap_StopwordsFiltered(t *testing.T) {
	// Criterion is heavy on stopwords; only "implementation" and "feature"
	// are meaningful. The todo reproduces them → should match.
	criteria := []AcceptanceCriterion{
		{Text: "the implementation of the feature is done"},
	}
	todos := []TodoUpdate{
		{Content: "feature implementation finished", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "stopwords should be filtered before overlap computation")
}

func TestMatchTodosToAC_WordOverlap_ThresholdAtExactlyHalf(t *testing.T) {
	// Criterion meaningful words: routing, architecture, analysis, deep (4)
	// Todo meaningful words:      routing, architecture, implementation, plan (4)
	// Intersection: routing, architecture (2) → 2/4 = 0.5 → should match.
	criteria := []AcceptanceCriterion{
		{Text: "routing architecture analysis deep"},
	}
	todos := []TodoUpdate{
		{Content: "routing architecture implementation plan", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "overlap ratio 0.5 should match")
}

func TestMatchTodosToAC_WordOverlap_ThresholdBelowHalf(t *testing.T) {
	// Criterion meaningful words: routing, architecture, analysis, deep (4)
	// Todo meaningful words:      routing, implementation, plan, review (4)
	// Intersection: routing (1) → 1/4 = 0.25 → should NOT match.
	criteria := []AcceptanceCriterion{
		{Text: "routing architecture analysis deep"},
	}
	todos := []TodoUpdate{
		{Content: "routing implementation plan review", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.False(t, result[0].Completed, "overlap ratio 0.25 should not match")
}

func TestMatchTodosToAC_WordOverlap_OnlyMatchesCompletedTodos(t *testing.T) {
	// Word overlap would succeed on text alone, but todo is not completed.
	criteria := []AcceptanceCriterion{
		{Text: "analyze routing performance bottlenecks"},
	}
	todos := []TodoUpdate{
		{Content: "routing performance bottlenecks analysis", Status: "in_progress"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.False(t, result[0].Completed, "word overlap pass must not match non-completed todos")
}

// ---------------------------------------------------------------------------
// Regression: existing passes unaffected by new passes
// ---------------------------------------------------------------------------

func TestMatchTodosToAC_Regression_ExactMatch(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "implement feature A"},
		{Text: "write tests"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A", Status: "completed"},
		{Content: "write tests", Status: "done"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 2)
	assert.True(t, result[0].Completed, "exact match should still work")
	assert.True(t, result[1].Completed, "exact match with 'done' status should still work")
}

func TestMatchTodosToAC_Regression_SubstringMatch(t *testing.T) {
	criteria := []AcceptanceCriterion{
		{Text: "feature A"},
	}
	todos := []TodoUpdate{
		{Content: "implement feature A with tests", Status: "completed"},
	}
	result := MatchTodosToAC(criteria, todos)
	require.Len(t, result, 1)
	assert.True(t, result[0].Completed, "substring match should still work")
}
