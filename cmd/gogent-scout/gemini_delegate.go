package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// delegateToGemini sends the file list to gemini-slave scout protocol.
func delegateToGemini(target, instruction string) (*ScoutReport, error) {
	// Build file list
	fileList, err := generateFileList(target)
	if err != nil {
		return nil, fmt.Errorf("failed to generate file list: %w", err)
	}

	if fileList == "" {
		return nil, fmt.Errorf("no supported files found in %s", target)
	}

	// Execute gemini-slave with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gemini-slave", "scout", instruction)
	cmd.Stdin = strings.NewReader(fileList)

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("gemini-slave timed out after 30s")
		}
		// Include stderr if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gemini-slave failed: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gemini-slave failed: %w", err)
	}

	// Validate and parse output
	return validateGeminiOutput(output)
}

// generateFileList creates a newline-separated list of supported files.
func generateFileList(target string) (string, error) {
	var files []string
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			// Skip hidden and vendor directories
			if d != nil && d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if _, ok := SupportedExtensions[filepath.Ext(path)]; ok {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(files, "\n"), nil
}

// validateGeminiOutput parses and validates the gemini-slave scout output.
func validateGeminiOutput(data []byte) (*ScoutReport, error) {
	// gemini-slave might output other text before/after the JSON.
	// We try to find the first '{' and last '}' to isolate the JSON object.
	input := string(data)
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")

	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON object found in gemini-slave output")
	}

	jsonData := input[start : end+1]

	var output GeminiScoutOutput
	if err := json.Unmarshal([]byte(jsonData), &output); err != nil {
		return nil, fmt.Errorf("invalid JSON from gemini-slave: %w", err)
	}

	// Schema version check (allow empty for backwards compatibility)
	if output.SchemaVersion != "1.0" && output.SchemaVersion != "" {
		return nil, fmt.Errorf("unsupported schema version: %s", output.SchemaVersion)
	}

	// Validate required fields
	if output.ScoutReport == nil {
		return nil, fmt.Errorf("missing scout_report in gemini output")
	}
	if output.ScoutReport.ScopeMetrics == nil {
		return nil, fmt.Errorf("missing scope_metrics in scout_report")
	}
	if output.ScoutReport.RoutingRecommendation == nil {
		return nil, fmt.Errorf("missing routing_recommendation in scout_report")
	}

	// Mark backend and ensure schema version
	output.ScoutReport.Backend = "gemini"
	output.ScoutReport.SchemaVersion = "1.0"

	// Ensure complexity signals marked as available if present
	if output.ScoutReport.ComplexitySignals != nil {
		output.ScoutReport.ComplexitySignals.Available = true
	}

	return output.ScoutReport, nil
}
