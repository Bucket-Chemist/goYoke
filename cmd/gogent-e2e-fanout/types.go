package main

import "fmt"

// Color represents a color with a name and hexadecimal value.
type Color struct {
	Name string
	Hex  string
}

// NewColor creates a new Color instance.
func NewColor(name, hex string) Color {
	return Color{
		Name: name,
		Hex:  hex,
	}
}

// String returns a string representation of the Color in the format "name (#hex)".
func (c Color) String() string {
	return fmt.Sprintf("%s (#%s)", c.Name, c.Hex)
}
