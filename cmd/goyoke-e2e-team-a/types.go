package main

import "fmt"

// Greeting represents a greeting message in a specific language
type Greeting struct {
	Language string
	Text     string
}

// NewGreeting creates a new Greeting with the specified language and text
func NewGreeting(lang, text string) Greeting {
	return Greeting{
		Language: lang,
		Text:     text,
	}
}

// String returns the greeting formatted as "[lang] text"
func (g Greeting) String() string {
	return fmt.Sprintf("[%s] %s", g.Language, g.Text)
}
