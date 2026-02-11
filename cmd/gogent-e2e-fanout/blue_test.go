package main

import "testing"

func TestBlueColor(t *testing.T) {
	// Test NewColor creates correct Color instance
	blue := NewColor("blue", "0000FF")

	if blue.Name != "blue" {
		t.Errorf("Expected Name to be 'blue', got '%s'", blue.Name)
	}

	if blue.Hex != "0000FF" {
		t.Errorf("Expected Hex to be '0000FF', got '%s'", blue.Hex)
	}

	// Test String() method returns correct format
	expected := "blue (#0000FF)"
	result := blue.String()

	if result != expected {
		t.Errorf("Expected String() to return '%s', got '%s'", expected, result)
	}
}
