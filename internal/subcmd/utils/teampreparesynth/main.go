// Command goyoke-team-prepare-synthesis bridges analyst waves and synthesis waves.
// It reads the team config to determine workflow type, finds the completed wave,
// and dispatches to the appropriate preparation handler.
//
// Usage:
//
//	goyoke-team-prepare-synthesis <team-directory>
//
// Supported workflow types:
//   - braintrust: reads Einstein + Staff-Arch stdout, writes pre-synthesis.md
//   - review-bioinformatics: updates stdin_pasteur.json with wave_0_outputs
//   - unknown: logs warning and exits 0 (graceful no-op)
//
// Exit codes:
//   - 0: Success (including graceful degradation for missing/malformed input)
//   - 1: Fatal error (team directory not found, write failure)
//   - 2: Usage error (wrong number of arguments)
package teampreparesynth

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

// prepareBraintrust runs the braintrust preparation logic.
// Reads Einstein + Staff-Arch stdout and writes pre-synthesis.md.
func prepareBraintrust(teamDir string) error {
	einsteinPath := filepath.Join(teamDir, einsteinStdoutFile)
	staffArchPath := filepath.Join(teamDir, staffArchStdoutFile)
	outputPath := filepath.Join(teamDir, outputFile)

	log.Printf("Preparing braintrust synthesis from team directory: %s", teamDir)

	einstein := extractEinstein(einsteinPath)
	staffArch := extractStaffArch(staffArchPath)
	markdown := generateMarkdown(einstein, staffArch)

	if err := os.WriteFile(outputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outputPath, err)
	}

	log.Printf("Successfully wrote pre-synthesis document to: %s", outputPath)
	return nil
}

func Main() {
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

	// Load config to determine workflow type
	config, err := loadConfig(teamDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to load config: %v\n", err)
		os.Exit(1)
	}

	switch config.WorkflowType {
	case "braintrust":
		if err := prepareBraintrust(teamDir); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: braintrust preparation failed: %v\n", err)
			os.Exit(1)
		}
	case "review-bioinformatics":
		completedWaveIdx := findCompletedWaveIdx(config)
		if completedWaveIdx == -1 {
			log.Printf("[WARN] review-bioinformatics: no completed wave with pending next wave found, skipping")
			return
		}
		if err := prepareBioinformaticsReview(teamDir, config, completedWaveIdx); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: bioinformatics review preparation failed: %v\n", err)
			os.Exit(1)
		}
	default:
		log.Printf("WARNING: unknown workflow_type %q, skipping synthesis", config.WorkflowType)
	}
}
