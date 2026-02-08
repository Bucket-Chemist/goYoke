package main

import "testing"

func TestRedColor(t *testing.T) {
	// Test that NewColor("red", "FF0000") creates a Color with Name="red" and Hex="FF0000"
	c := NewColor("red", "FF0000")

	if c.Name != "red" {
		t.Errorf("Expected Name to be 'red', got '%s'", c.Name)
	}

	if c.Hex != "FF0000" {
		t.Errorf("Expected Hex to be 'FF0000', got '%s'", c.Hex)
	}

	// Test that the String() method returns "red (#FF0000)"
	expected := "red (#FF0000)"
	actual := c.String()

	if actual != expected {
		t.Errorf("Expected String() to return '%s', got '%s'", expected, actual)
	}
}
