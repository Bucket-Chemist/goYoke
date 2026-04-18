package main

// Pair represents a pair of strings.
type Pair struct {
	Left  string
	Right string
}

// NewPair creates a new Pair with the given left and right values.
func NewPair(l, r string) Pair {
	return Pair{Left: l, Right: r}
}
