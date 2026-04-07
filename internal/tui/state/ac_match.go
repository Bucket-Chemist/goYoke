package state

import "strings"

// MatchTodosToAC matches a slice of TodoUpdate items to a slice of
// AcceptanceCriterion values, returning a new slice with Completed fields set.
//
// Matching strategy (in order):
//  1. Positional: if todos[i].Content == criteria[i].Text exactly, mark
//     criteria[i].Completed according to todos[i].Status.
//  2. Fuzzy fallback: for each criterion that was not matched positionally,
//     scan all todos and mark it completed when any todo's Content contains
//     the criterion's Text or vice-versa.
//
// The original criteria slice is not modified; a new slice is returned.
// Empty or nil inputs are returned unchanged.
func MatchTodosToAC(criteria []AcceptanceCriterion, todos []TodoUpdate) []AcceptanceCriterion {
	if len(criteria) == 0 || len(todos) == 0 {
		return criteria
	}

	result := make([]AcceptanceCriterion, len(criteria))
	copy(result, criteria)

	matched := make([]bool, len(result))

	// Pass 1: positional exact match.
	for i := range result {
		if i >= len(todos) {
			break
		}
		if todos[i].Content == result[i].Text {
			result[i].Completed = acIsCompleted(todos[i].Status)
			matched[i] = true
		}
	}

	// Pass 2: fuzzy fallback for unmatched criteria.
	for i := range result {
		if matched[i] {
			continue
		}
		for _, todo := range todos {
			if strings.Contains(todo.Content, result[i].Text) ||
				strings.Contains(result[i].Text, todo.Content) {
				result[i].Completed = acIsCompleted(todo.Status)
				matched[i] = true
				break
			}
		}
	}

	return result
}

// acIsCompleted returns true when the TodoWrite status string indicates the
// item has been completed.
func acIsCompleted(status string) bool {
	return status == "completed" || status == "done"
}
