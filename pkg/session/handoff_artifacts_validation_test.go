package session

import (
	"strings"
	"testing"
)

func TestValidateSharpEdge_Valid(t *testing.T) {
	validJSON := `{"file":"test.go","error_type":"parse_error","consecutive_failures":3,"timestamp":1234567890}`

	err := ValidateSharpEdge([]byte(validJSON))
	if err != nil {
		t.Errorf("Valid SharpEdge failed validation: %v", err)
	}
}

func TestValidateSharpEdge_WithOptionalFields(t *testing.T) {
	validJSON := `{"file":"test.go","error_type":"type_error","consecutive_failures":5,"timestamp":1234567890,"context":"line 42","error_message":"test error","severity":"high","resolution":"Fix type","resolved_at":1234567900}`

	err := ValidateSharpEdge([]byte(validJSON))
	if err != nil {
		t.Errorf("Valid SharpEdge with optional fields failed validation: %v", err)
	}
}

func TestValidateSharpEdge_MissingFile(t *testing.T) {
	invalidJSON := `{"error_type":"parse_error","consecutive_failures":3,"timestamp":1234567890}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for missing 'file' field")
	}
	if !strings.Contains(err.Error(), "file") {
		t.Errorf("Error should mention missing 'file' field: %v", err)
	}
}

func TestValidateSharpEdge_InvalidFailureCount(t *testing.T) {
	invalidJSON := `{"file":"test.go","error_type":"parse_error","consecutive_failures":2,"timestamp":1234567890}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for consecutive_failures < 3")
	}
	if !strings.Contains(err.Error(), "consecutive_failures") {
		t.Errorf("Error should mention invalid consecutive_failures: %v", err)
	}
}

func TestValidateSharpEdge_InvalidJSON(t *testing.T) {
	invalidJSON := `{not valid json`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for malformed JSON")
	}
}

func TestValidateSharpEdge_MissingTimestamp(t *testing.T) {
	invalidJSON := `{"file":"test.go","error_type":"parse_error","consecutive_failures":3}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for missing 'timestamp' field")
	}
	if !strings.Contains(err.Error(), "timestamp") {
		t.Errorf("Error should mention missing 'timestamp' field: %v", err)
	}
}

func TestValidateSharpEdge_InvalidSeverity(t *testing.T) {
	invalidJSON := `{"file":"test.go","error_type":"parse_error","consecutive_failures":3,"timestamp":1234567890,"severity":"critical"}`

	err := ValidateSharpEdge([]byte(invalidJSON))
	if err == nil {
		t.Error("Expected error for invalid severity")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("Error should mention invalid severity: %v", err)
	}
}

func TestValidateSharpEdge_ValidSeverities(t *testing.T) {
	severities := []string{"high", "medium", "low"}
	for _, sev := range severities {
		validJSON := `{"file":"test.go","error_type":"parse_error","consecutive_failures":3,"timestamp":1234567890,"severity":"` + sev + `"}`

		err := ValidateSharpEdge([]byte(validJSON))
		if err != nil {
			t.Errorf("Valid severity '%s' failed validation: %v", sev, err)
		}
	}
}
