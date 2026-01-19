package session

import (
	"strings"
	"testing"
)

func TestValidateSharpEdge_Valid(t *testing.T) {
	validJSON := `{"file":"test.go","error_type":"parse_error","failure_count":3,"last_occurrence":1234567890}`

	err := ValidateSharpEdge([]byte(validJSON))
	if err != nil {
		t.Errorf("Valid SharpEdge failed validation: %v", err)
	}
}

func TestValidateSharpEdge_WithOptionalFields(t *testing.T) {
	validJSON := `{"file":"test.go","error_type":"type_error","failure_count":5,"last_occurrence":1234567890,"context":"line 42","remediation":"Fix type"}`

	err := ValidateSharpEdge([]byte(validJSON))
	if err != nil {
		t.Errorf("Valid SharpEdge with optional fields failed validation: %v", err)
	}
}

func TestValidateSharpEdge_MissingFile(t *testing.T) {
	invalidJSON := `{"error_type":"parse_error","failure_count":3,"last_occurrence":1234567890}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for missing 'file' field")
	}
	if !strings.Contains(err.Error(), "file") {
		t.Errorf("Error should mention missing 'file' field: %v", err)
	}
}

func TestValidateSharpEdge_InvalidFailureCount(t *testing.T) {
	invalidJSON := `{"file":"test.go","error_type":"parse_error","failure_count":2,"last_occurrence":1234567890}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for failure_count < 3")
	}
	if !strings.Contains(err.Error(), "failure_count") {
		t.Errorf("Error should mention invalid failure_count: %v", err)
	}
}

func TestValidateSharpEdge_InvalidJSON(t *testing.T) {
	invalidJSON := `{not valid json`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for malformed JSON")
	}
}
