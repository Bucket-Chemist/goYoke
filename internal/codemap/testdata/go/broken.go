package testfixture

// ValidFunc is a complete, parseable function.
func ValidFunc() string { return "valid" }

// Incomplete is missing its closing brace — the parser produces a partial AST.
func Incomplete() {
