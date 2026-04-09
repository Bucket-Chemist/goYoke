package state

import "strings"

// stopwords is the set of common English words excluded from word-overlap
// scoring to avoid false positives from high-frequency terms.
var stopwords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "with": {},
	"for": {}, "to": {}, "in": {}, "of": {}, "is": {}, "are": {},
	"was": {}, "were": {}, "be": {}, "been": {}, "has": {}, "have": {},
	"had": {}, "do": {}, "does": {}, "did": {}, "will": {}, "would": {},
	"could": {}, "should": {}, "may": {}, "might": {}, "must": {}, "shall": {},
	"can": {}, "that": {}, "this": {}, "it": {}, "its": {}, "by": {},
	"on": {}, "at": {}, "from": {}, "as": {}, "but": {}, "not": {},
	"no": {}, "all": {}, "each": {}, "every": {}, "both": {}, "any": {},
	"into": {},
}

// MatchTodosToAC matches a slice of TodoUpdate items to a slice of
// AcceptanceCriterion values, returning a new slice with Completed fields set.
//
// Matching strategy (in order):
//  1. Positional: if todos[i].Content == criteria[i].Text exactly, mark
//     criteria[i].Completed according to todos[i].Status.
//  2. Case-insensitive fuzzy fallback: for each criterion not matched
//     positionally, scan all todos and mark it completed when any todo's
//     Content contains the criterion's Text or vice-versa (case-insensitive).
//  3. Word-overlap: for each criterion still unmatched, compute the ratio of
//     meaningful-word overlap between criterion and each completed todo.
//     Match if ratio >= 0.5.
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

	// Pass 2: case-insensitive fuzzy fallback for unmatched criteria.
	for i := range result {
		if matched[i] {
			continue
		}
		criterionLower := strings.ToLower(result[i].Text)
		for _, todo := range todos {
			todoLower := strings.ToLower(todo.Content)
			if strings.Contains(todoLower, criterionLower) ||
				strings.Contains(criterionLower, todoLower) {
				result[i].Completed = acIsCompleted(todo.Status)
				matched[i] = true
				break
			}
		}
	}

	// Pass 3: word-overlap scoring for remaining unmatched criteria.
	// Only matches against todos that are already completed/done.
	for i := range result {
		if matched[i] {
			continue
		}
		criterionWords := tokenize(result[i].Text)
		if len(criterionWords) == 0 {
			continue
		}
		for _, todo := range todos {
			if !acIsCompleted(todo.Status) {
				continue
			}
			todoWords := tokenize(todo.Content)
			if len(todoWords) == 0 {
				continue
			}
			if wordOverlapRatio(criterionWords, todoWords) >= 0.5 {
				result[i].Completed = true
				matched[i] = true
				break
			}
		}
	}

	return result
}

// tokenize splits text into a set of lowercase words, stripping punctuation
// and filtering stopwords and single-character tokens.
func tokenize(text string) map[string]struct{} {
	words := make(map[string]struct{})
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.Trim(word, ".,!?;:()[]{}\"'`-")
		if len(word) <= 1 {
			continue
		}
		if _, isStop := stopwords[word]; isStop {
			continue
		}
		words[word] = struct{}{}
	}
	return words
}

// wordOverlapRatio returns len(intersection) / min(len(a), len(b)).
func wordOverlapRatio(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersection := 0
	for word := range a {
		if _, ok := b[word]; ok {
			intersection++
		}
	}
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	return float64(intersection) / float64(minLen)
}

// acIsCompleted returns true when the TodoWrite status string indicates the
// item has been completed.
func acIsCompleted(status string) bool {
	return status == "completed" || status == "done"
}
