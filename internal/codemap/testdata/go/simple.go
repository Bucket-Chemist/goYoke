package testfixture

// Hello is an exported function.
func Hello() string { return "hello" }

func add(a, b int) int { return a + b }

// Config holds application settings.
type Config struct {
	Name    string
	Timeout int
}

// Handler processes incoming requests.
type Handler interface {
	Handle(req string) error
}
