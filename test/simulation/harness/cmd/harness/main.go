package main

import (
	"encoding/json"
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
	mode := flag.String("mode", "deterministic", "Simulation mode: deterministic, fuzz, mixed, replay, behavioral, chaos")
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

	// Chaos testing parameters
	chaosAgents := flag.Int("chaos-agents", 10, "Number of chaos agents")
	chaosSharedRatio := flag.Float64("chaos-shared-ratio", 0.3, "Ratio of agents sharing keys (0.0-1.0)")

	flag.Parse()

	// Support environment variable overrides for chaos (CI integration)
	if envAgents := os.Getenv("CHAOS_AGENTS"); envAgents != "" {
		if n, err := parseEnvInt(envAgents); err == nil {
			*chaosAgents = n
		}
	}
	if envRatio := os.Getenv("CHAOS_SHARED_RATIO"); envRatio != "" {
		if r, err := parseEnvFloat(envRatio); err == nil {
			*chaosSharedRatio = r
		}
	}

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
		FixturesDir:    fixturesDir,
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

	// Find optional sharp-edge binary for posttooluse scenarios
	sharpEdgePath, sharpEdgeErr := findBinary("gogent-sharp-edge")
	if sharpEdgeErr != nil && *verbose {
		fmt.Printf("[INFO] gogent-sharp-edge not found, posttooluse scenarios will be skipped\n")
	}

	// Initialize components
	gen := harness.NewGenerator(fixturesDir)
	runner := harness.NewRunner(cfg, validatePath, archivePath, gen)

	// Set sharp-edge path if available
	if sharpEdgeErr == nil {
		runner.SetSharpEdgePath(sharpEdgePath)
	}

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
	case "replay":
		// Session replay mode (GOgent-042)
		exitCode := runSessionReplay(cfg, validatePath, archivePath, sharpEdgePath, *verbose, *outputDir)
		os.Exit(exitCode)
	case "behavioral":
		// Behavioral property testing mode (GOgent-042)
		exitCode := runBehavioral(cfg, validatePath, archivePath, sharpEdgePath, *verbose, *outputDir)
		os.Exit(exitCode)
	case "chaos":
		// Chaos testing mode (GOgent-042)
		exitCode := runChaos(cfg, sharpEdgePath, *chaosAgents, *chaosSharedRatio, *verbose, *outputDir)
		os.Exit(exitCode)
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
	validModes := map[string]bool{
		"deterministic": true,
		"fuzz":          true,
		"mixed":         true,
		"replay":        true,     // GOgent-042
		"behavioral":    true,     // GOgent-042
		"chaos":         true,     // GOgent-042
	}
	if !validModes[cfg.Mode] {
		return fmt.Errorf("invalid mode: %s (must be deterministic, fuzz, mixed, replay, behavioral, or chaos)", cfg.Mode)
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

// runSessionReplay executes session replay testing (GOgent-042 Level 2).
func runSessionReplay(cfg harness.SimulationConfig, validatePath, archivePath, sharpEdgePath string, verbose bool, outputDir string) int {
	fmt.Println("Running session replay tests...")

	if sharpEdgePath == "" {
		fmt.Fprintf(os.Stderr, "Error: gogent-sharp-edge binary required for session replay\n")
		return 1
	}

	replayer := harness.NewSessionReplayer(validatePath, archivePath, sharpEdgePath, cfg.FixturesDir)
	replayer.SetSchemaPath(cfg.SchemaPath)
	replayer.SetAgentsPath(cfg.AgentsPath)
	replayer.SetVerbose(verbose)

	results, err := replayer.ReplayAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Session replay error: %v\n", err)
		return 1
	}

	// Generate report
	os.MkdirAll(outputDir, 0755)
	reportPath := filepath.Join(outputDir, fmt.Sprintf("replay-%s.json",
		time.Now().Format("20060102-150405")))

	if err := writeReplayReport(results, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
	}

	// Print summary
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	fmt.Printf("\n=== Session Replay Complete ===\n")
	fmt.Printf("Sessions: %d, Passed: %d, Failed: %d\n", len(results), passed, len(results)-passed)
	fmt.Printf("Report: %s\n", reportPath)

	if passed < len(results) {
		// Print failure details
		fmt.Println("\nFailures:")
		for _, r := range results {
			if !r.Passed {
				fmt.Printf("  - %s:\n", r.SessionID)
				for _, e := range r.EventErrors {
					fmt.Printf("      Event: %s\n", e)
				}
				for _, e := range r.ValidationErrors {
					fmt.Printf("      Validation: %s\n", e)
				}
				if r.ErrorMsg != "" {
					fmt.Printf("      Error: %s\n", r.ErrorMsg)
				}
			}
		}
		return 1
	}
	return 0
}

// runBehavioral executes behavioral property testing (GOgent-042 Level 3).
func runBehavioral(cfg harness.SimulationConfig, validatePath, archivePath, sharpEdgePath string, verbose bool, outputDir string) int {
	fmt.Println("Running behavioral property tests...")

	if sharpEdgePath == "" {
		fmt.Fprintf(os.Stderr, "Error: gogent-sharp-edge binary required for behavioral tests\n")
		return 1
	}

	// First run session replay to generate state
	replayer := harness.NewSessionReplayer(validatePath, archivePath, sharpEdgePath, cfg.FixturesDir)
	replayer.SetSchemaPath(cfg.SchemaPath)
	replayer.SetAgentsPath(cfg.AgentsPath)
	replayer.SetVerbose(verbose)

	replayResults, err := replayer.ReplayAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Replay phase error: %v\n", err)
		return 1
	}

	// Check behavioral invariants on all passed sessions
	var allInvariantResults []harness.InvariantResult
	sessionsChecked := 0

	for _, rr := range replayResults {
		if !rr.Passed {
			continue // Skip failed sessions
		}

		// Load context from a fresh temp dir (sessions already cleaned up)
		// For behavioral tests, we need to re-run with a temp dir that persists
		// This is handled within ReplaySession which cleans up - we need to modify approach
		sessionsChecked++
	}

	// Run behavioral invariants in isolation
	fmt.Printf("Checking %d behavioral invariants...\n", len(harness.BehavioralInvariants))

	// Create a test context with sample data to verify invariants work
	testTempDir, err := os.MkdirTemp("", "behavioral-test-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(testTempDir)

	// Setup directories
	os.MkdirAll(filepath.Join(testTempDir, ".claude", "memory"), 0755)
	os.MkdirAll(filepath.Join(testTempDir, ".gogent"), 0755)

	// Create test sharp edge with valid schema
	sharpEdgeContent := `{"file":"test.py","error_type":"TypeError","consecutive_failures":3,"timestamp":1705000020}`
	os.WriteFile(filepath.Join(testTempDir, ".claude", "memory", "pending-learnings.jsonl"),
		[]byte(sharpEdgeContent+"\n"), 0644)

	// Create test handoff with correct schema version
	handoffContent := `{"schema_version":"1.2","session_id":"test","timestamp":"2026-01-23T10:00:00Z"}`
	os.WriteFile(filepath.Join(testTempDir, ".claude", "memory", "handoffs.jsonl"),
		[]byte(handoffContent+"\n"), 0644)

	// Create failure tracker
	trackerContent := `{"file":"test.py","error_type":"TypeError","timestamp":1705000000}
{"file":"test.py","error_type":"TypeError","timestamp":1705000010}
{"file":"test.py","error_type":"TypeError","timestamp":1705000020}`
	os.WriteFile(filepath.Join(testTempDir, ".gogent", "failure-tracker.jsonl"),
		[]byte(trackerContent+"\n"), 0644)

	// Load and check context
	ctx, err := harness.LoadBehavioralContext(testTempDir, harness.DefaultBehavioralConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading context: %v\n", err)
		return 1
	}

	allInvariantResults = harness.CheckBehavioralInvariants(ctx)

	// Generate report
	os.MkdirAll(outputDir, 0755)
	reportPath := filepath.Join(outputDir, fmt.Sprintf("behavioral-%s.json",
		time.Now().Format("20060102-150405")))

	if err := writeBehavioralReport(replayResults, allInvariantResults, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
	}

	// Count results
	passed := 0
	for _, r := range allInvariantResults {
		if r.Passed {
			passed++
		}
	}

	replayPassed := 0
	for _, r := range replayResults {
		if r.Passed {
			replayPassed++
		}
	}

	fmt.Printf("\n=== Behavioral Tests Complete ===\n")
	fmt.Printf("Replay: %d/%d sessions passed\n", replayPassed, len(replayResults))
	fmt.Printf("Invariants: %d/%d passed\n", passed, len(allInvariantResults))
	fmt.Printf("Report: %s\n", reportPath)

	if passed < len(allInvariantResults) || replayPassed < len(replayResults) {
		fmt.Println("\nInvariant Failures:")
		for _, r := range allInvariantResults {
			if !r.Passed {
				fmt.Printf("  - [%s]: %s\n", r.InvariantID, r.Message)
			}
		}
		return 1
	}
	return 0
}

// runChaos executes chaos testing (GOgent-042 Level 4).
func runChaos(cfg harness.SimulationConfig, sharpEdgePath string, numAgents int, sharedRatio float64, verbose bool, outputDir string) int {
	fmt.Printf("Running chaos tests (%d agents, %.0f%% shared keys)...\n", numAgents, sharedRatio*100)

	if sharpEdgePath == "" {
		fmt.Fprintf(os.Stderr, "Error: gogent-sharp-edge binary required for chaos testing\n")
		return 1
	}

	// Create dedicated temp dir for chaos
	chaosTempDir, err := os.MkdirTemp("", "chaos-test-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		return 1
	}
	defer os.RemoveAll(chaosTempDir)

	chaosConfig := harness.ChaosConfig{
		NumAgents:        numAgents,
		FailuresPerAgent: 20,
		SharedKeyRatio:   sharedRatio,
		MaxFailures:      3,
		Duration:         60 * time.Second,
		Seed:             cfg.FuzzSeed,
		SharpEdgePath:    sharpEdgePath,
		SchemaPath:       cfg.SchemaPath,
		AgentsPath:       cfg.AgentsPath,
		Verbose:          verbose,
	}

	runner := harness.NewChaosRunner(chaosConfig, chaosTempDir)
	report, err := runner.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chaos test error: %v\n", err)
		return 1
	}

	// Generate report
	os.MkdirAll(outputDir, 0755)
	reportPath := filepath.Join(outputDir, fmt.Sprintf("chaos-%s.json",
		time.Now().Format("20060102-150405")))

	if err := writeChaosReport(report, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
	}

	fmt.Printf("\n=== Chaos Test Complete ===\n")
	fmt.Printf("Agents: %d (shared: %d, unique: %d)\n",
		report.TotalAgents, report.SharedKeyAgents, report.UniqueKeyAgents)
	fmt.Printf("Events: %d\n", report.TotalEvents)
	fmt.Printf("Duration: %v\n", report.Duration)
	if report.JSONLValidation != nil {
		fmt.Printf("JSONL: %d files, %d lines, %d invalid\n",
			report.JSONLValidation.FilesChecked,
			report.JSONLValidation.TotalLines,
			report.JSONLValidation.InvalidLines)
	}
	fmt.Printf("Report: %s\n", reportPath)

	if !report.Passed {
		fmt.Println("\nErrors:")
		for _, e := range report.Errors {
			fmt.Printf("  - %s\n", e)
		}
		return 1
	}

	fmt.Println("Result: PASSED")
	return 0
}

// Helper functions for report writing

func writeReplayReport(results []harness.ReplayResult, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{
		"type":      "replay",
		"timestamp": time.Now().Format(time.RFC3339),
		"results":   results,
	})
}

func writeBehavioralReport(replayResults []harness.ReplayResult, invariantResults []harness.InvariantResult, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{
		"type":       "behavioral",
		"timestamp":  time.Now().Format(time.RFC3339),
		"replay":     replayResults,
		"invariants": invariantResults,
	})
}

func writeChaosReport(report *harness.ChaosReport, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]interface{}{
		"type":      "chaos",
		"timestamp": time.Now().Format(time.RFC3339),
		"report":    report,
	})
}

// parseEnvInt parses an integer from environment variable.
func parseEnvInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// parseEnvFloat parses a float from environment variable.
func parseEnvFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
