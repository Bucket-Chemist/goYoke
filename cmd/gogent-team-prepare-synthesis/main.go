// Command gogent-team-prepare-synthesis bridges Wave 1 (Einstein + Staff-Architect)
// and Wave 2 (Beethoven) in the braintrust workflow. It extracts key analysis sections
// from parallel analyst JSON outputs and generates a pre-synthesis markdown document.
//
// Usage:
//
//	gogent-team-prepare-synthesis <team-directory>
//
// The binary reads stdout_einstein.json and stdout_staff-arch.json from the team
// directory, extracts sections with graceful degradation for missing or malformed
// data, and writes pre-synthesis.md for Beethoven to consume.
//
// Exit codes:
//   - 0: Success (including graceful degradation for missing/malformed input)
//   - 1: Fatal error (team directory not found, write failure)
//   - 2: Usage error (wrong number of arguments)
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	einsteinStdoutFile  = "stdout_einstein.json"
	staffArchStdoutFile = "stdout_staff-arch.json"
	outputFile          = "pre-synthesis.md"
)

// validateTeamDir validates and resolves the team directory path.
// Returns the cleaned absolute path or an error.
func validateTeamDir(path string) (string, error) {
	cleaned := filepath.Clean(path)

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("team directory not found: %s", abs)
	}
	if err != nil {
		return "", fmt.Errorf("cannot access team directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", abs)
	}

	return abs, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <team-directory>\n", os.Args[0])
		os.Exit(2)
	}

	teamDir, err := validateTeamDir(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Pre-flight: verify team directory is writable
	testFile := filepath.Join(teamDir, ".write-test")
	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Team directory is not writable: %v\n", err)
		os.Exit(1)
	}
	os.Remove(testFile)

	// Build file paths
	einsteinPath := filepath.Join(teamDir, einsteinStdoutFile)
	staffArchPath := filepath.Join(teamDir, staffArchStdoutFile)
	outputPath := filepath.Join(teamDir, outputFile)

	log.Printf("Preparing synthesis from team directory: %s", teamDir)

	// Extract sections from both analysts
	einstein := extractEinstein(einsteinPath)
	staffArch := extractStaffArch(staffArchPath)

	// Generate markdown
	markdown := generateMarkdown(einstein, staffArch)

	// Write output file
	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to write %s: %v\n", outputPath, err)
		os.Exit(1)
	}

	log.Printf("Successfully wrote pre-synthesis document to: %s", outputPath)
}
