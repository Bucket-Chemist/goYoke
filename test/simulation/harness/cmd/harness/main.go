package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bucket-Chemist/GOgent-Fortress/test/simulation/harness"
)

func main() {
	// Define flags
	mode := flag.String("mode", "deterministic", "Simulation mode: deterministic, fuzz, mixed")
	filter := flag.String("filter", "", "Scenario ID prefix filter")
	iterations := flag.Int("iterations", 1000, "Fuzz iteration count")
	seed := flag.Int64("seed", time.Now().UnixNano(), "Random seed for reproducibility")
	timeout := flag.Duration("timeout", 5*time.Minute, "Fuzz timeout")
	reportFormat := flag.String("report", "json", "Report format: json, markdown, tap")
	verbose := flag.Bool("verbose", false, "Verbose output")
	replay := flag.String("replay", "", "Crash file path for replay")
	schemaPath := flag.String("schema", "", "Path to routing-schema.json")
	agentsPath := flag.String("agents", "", "Path to agents-index.json")
	outputDir := flag.String("output", "", "Report output directory")

	flag.Parse()

	// Determine paths relative to harness directory
	harnessDir, err := findHarnessDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding harness directory: %v\n", err)
		os.Exit(1)
	}

	// Set defaults based on harness location
	fixturesDir := filepath.Join(harnessDir, "..", "fixtures")
	if *schemaPath == "" {
		*schemaPath = filepath.Join(fixturesDir, "schemas", "routing-schema.json")
	}
	if *agentsPath == "" {
		*agentsPath = filepath.Join(fixturesDir, "schemas", "agents-index.json")
	}
	if *outputDir == "" {
		*outputDir = filepath.Join(harnessDir, "..", "reports")
	}

	// Build configuration
	cfg := harness.SimulationConfig{
		Mode:           *mode,
		ScenarioFilter: parseFilter(*filter),
		FuzzIterations: *iterations,
		FuzzSeed:       *seed,
		FuzzTimeout:    *timeout,
		SchemaPath:     *schemaPath,
		AgentsPath:     *agentsPath,
		TempDir:        filepath.Join(harnessDir, "..", "tmp"),
		ReportFormat:   *reportFormat,
		Verbose:        *verbose,
	}

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Create temp directory
	os.MkdirAll(cfg.TempDir, 0755)

	// Find CLI binaries
	validatePath, err := findBinary("gogent-validate")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding gogent-validate: %v\n", err)
		os.Exit(1)
	}
	archivePath, err := findBinary("gogent-archive")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding gogent-archive: %v\n", err)
		os.Exit(1)
	}

	// Initialize components
	gen := harness.NewGenerator(fixturesDir)
	runner := harness.NewRunner(cfg, validatePath, archivePath, gen)

	// Handle replay mode
	if *replay != "" {
		exitCode := runReplay(*replay, cfg, gen, runner)
		os.Exit(exitCode)
	}

	// Run simulation
	var results []harness.SimulationResult
	var fuzzCrashes []harness.FuzzCrash

	switch cfg.Mode {
	case "deterministic":
		results, err = runDeterministic(cfg, runner)
	case "fuzz":
		results, fuzzCrashes, err = runFuzz(cfg, gen, runner)
	case "mixed":
		results, fuzzCrashes, err = runMixed(cfg, gen, runner)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", cfg.Mode)
		os.Exit(1)
	}

	var simulationErr error
	if err != nil {
		simulationErr = err
		fmt.Fprintf(os.Stderr, "Simulation error: %v\n", err)
	}

	// Generate report (even on error, for debugging)
	os.MkdirAll(*outputDir, 0755)
	reporter := harness.NewReporter(cfg.ReportFormat)
	reportPath := filepath.Join(*outputDir, fmt.Sprintf("simulation-%s.%s",
		time.Now().Format("20060102-150405"),
		reportExtension(cfg.ReportFormat)))

	if err := reporter.GenerateToFile(results, cfg, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		// Don't exit here - continue to show summary
	}

	// Exit early if simulation had an error
	if simulationErr != nil {
		os.Exit(1)
	}

	// Print summary
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	fmt.Printf("\n=== Simulation Complete ===\n")
	fmt.Printf("Mode: %s\n", cfg.Mode)
	fmt.Printf("Total: %d, Passed: %d, Failed: %d\n", len(results), passed, len(results)-passed)
	if len(fuzzCrashes) > 0 {
		fmt.Printf("Fuzz crashes: %d (saved to %s)\n", len(fuzzCrashes), filepath.Join(cfg.TempDir, "fuzz", "crashes"))
	}
	fmt.Printf("Report: %s\n", reportPath)

	// Exit with appropriate code
	if passed < len(results) {
		os.Exit(1)
	}
	os.Exit(0)
}

// findHarnessDir locates the harness directory.
func findHarnessDir() (string, error) {
	// Try current directory first
	if _, err := os.Stat("types.go"); err == nil {
		return ".", nil
	}

	// Try test/simulation/harness
	if _, err := os.Stat("test/simulation/harness/types.go"); err == nil {
		return "test/simulation/harness", nil
	}

	// Try relative to executable
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "types.go")); err == nil {
		return dir, nil
	}

	return "", fmt.Errorf("cannot find harness directory")
}

// findBinary locates a CLI binary.
func findBinary(name string) (string, error) {
	// Try project bin/ directory first (for CI/local builds)
	// Walk up from current dir to find project root with bin/
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		binPath := filepath.Join(dir, "bin", name)
		if _, err := os.Stat(binPath); err == nil {
			return binPath, nil
		}
	}

	// Try ~/.local/bin (for installed binaries)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		homePath := filepath.Join(homeDir, ".local", "bin", name)
		if _, err := os.Stat(homePath); err == nil {
			return homePath, nil
		}
	}

	// Try PATH using exec.LookPath
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("binary not found: %s", name)
}

// parseFilter splits comma-separated filter into slice.
func parseFilter(filter string) []string {
	if filter == "" {
		return nil
	}
	parts := strings.Split(filter, ",")
	// Trim whitespace from each part
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateConfig checks configuration validity.
func validateConfig(cfg harness.SimulationConfig) error {
	validModes := map[string]bool{"deterministic": true, "fuzz": true, "mixed": true}
	if !validModes[cfg.Mode] {
		return fmt.Errorf("invalid mode: %s (must be deterministic, fuzz, or mixed)", cfg.Mode)
	}

	validFormats := map[string]bool{"json": true, "markdown": true, "tap": true}
	if !validFormats[cfg.ReportFormat] {
		return fmt.Errorf("invalid report format: %s (must be json, markdown, or tap)", cfg.ReportFormat)
	}

	if cfg.FuzzIterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}

	if cfg.FuzzTimeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// reportExtension returns the file extension for a report format.
func reportExtension(format string) string {
	switch format {
	case "markdown":
		return "md"
	case "tap":
		return "tap"
	default:
		return "json"
	}
}

// runDeterministic executes deterministic scenarios.
func runDeterministic(cfg harness.SimulationConfig, runner *harness.DefaultRunner) ([]harness.SimulationResult, error) {
	fmt.Println("Running deterministic scenarios...")
	return runner.Run(cfg)
}

// runFuzz executes fuzz testing.
func runFuzz(cfg harness.SimulationConfig, gen *harness.DefaultGenerator, runner *harness.DefaultRunner) ([]harness.SimulationResult, []harness.FuzzCrash, error) {
	fmt.Printf("Running fuzz testing (%d iterations, seed: %d)...\n", cfg.FuzzIterations, cfg.FuzzSeed)

	fuzzRunner := harness.NewFuzzRunner(cfg, gen, runner)
	results, err := fuzzRunner.RunFuzz()
	if err != nil {
		return nil, nil, err
	}

	return results, fuzzRunner.GetCrashes(), nil
}

// runMixed executes both deterministic and fuzz testing.
func runMixed(cfg harness.SimulationConfig, gen *harness.DefaultGenerator, runner *harness.DefaultRunner) ([]harness.SimulationResult, []harness.FuzzCrash, error) {
	var allResults []harness.SimulationResult

	// Run deterministic first
	detResults, err := runDeterministic(cfg, runner)
	if err != nil {
		return nil, nil, fmt.Errorf("deterministic phase: %w", err)
	}
	allResults = append(allResults, detResults...)

	// Run fuzz
	fuzzResults, crashes, err := runFuzz(cfg, gen, runner)
	if err != nil {
		return nil, nil, fmt.Errorf("fuzz phase: %w", err)
	}
	allResults = append(allResults, fuzzResults...)

	return allResults, crashes, nil
}

// runReplay re-executes a crash file.
func runReplay(crashPath string, cfg harness.SimulationConfig, gen *harness.DefaultGenerator, runner *harness.DefaultRunner) int {
	fmt.Printf("Replaying crash: %s\n", crashPath)

	fuzzRunner := harness.NewFuzzRunner(cfg, gen, runner)
	result, err := fuzzRunner.ReplayCrash(crashPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Replay error: %v\n", err)
		return 1
	}

	if result.Passed {
		fmt.Println("Replay: PASS (crash no longer reproduces)")
		return 0
	}

	fmt.Println("Replay: FAIL (crash reproduced)")
	fmt.Printf("Diff:\n%s\n", result.Diff)
	return 1
}
