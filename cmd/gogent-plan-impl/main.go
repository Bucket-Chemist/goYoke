package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// ImplementationPlan represents the input JSON schema
type ImplementationPlan struct {
	Version string  `json:"version"`
	Project Project `json:"project"`
	Tasks   []Task  `json:"tasks"`
}

// Project contains project-level metadata
type Project struct {
	Language          string   `json:"language"`
	ConventionsFile   string   `json:"conventions_file"`
	BuildVerification string   `json:"build_verification,omitempty"`
	ErrorHandling     string   `json:"error_handling,omitempty"`
	TestPattern       string   `json:"test_pattern,omitempty"`
	ArchitectureNotes string   `json:"architecture_notes,omitempty"`
	PatternsToFollow  []string `json:"patterns_to_follow,omitempty"`
	AntiPatterns      []string `json:"anti_patterns,omitempty"`
}

// Task represents a single implementation task
type Task struct {
	TaskID             string        `json:"task_id"`
	Subject            string        `json:"subject"`
	Description        string        `json:"description"`
	Agent              string        `json:"agent"`
	TargetPackages     []string      `json:"target_packages"`
	RelatedFiles       []RelatedFile `json:"related_files,omitempty"`
	BlockedBy          []string      `json:"blocked_by"`
	AcceptanceCriteria []string      `json:"acceptance_criteria"`
	TestsRequired      *bool         `json:"tests_required,omitempty"`
	CoverageTarget     *int          `json:"coverage_target,omitempty"`
}

// RelatedFile represents a file related to a task
type RelatedFile struct {
	Path      string `json:"path"`
	Relevance string `json:"relevance"`
}

func main() {
	// Parse CLI flags
	planPath := flag.String("plan", "", "Path to implementation-plan.json")
	projectRoot := flag.String("project-root", "", "Absolute path to project root")
	outputDir := flag.String("output", "", "Team directory for generated files")
	flag.Parse()

	// Validate flags
	if *planPath == "" || *projectRoot == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --plan, --project-root, and --output are required")
		os.Exit(1)
	}

	// Read and unmarshal implementation plan
	planData, err := os.ReadFile(*planPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading plan file: %v\n", err)
		os.Exit(1)
	}

	var plan ImplementationPlan
	if err := json.Unmarshal(planData, &plan); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing plan JSON: %v\n", err)
		os.Exit(1)
	}

	// Load known agents
	knownAgents, err := loadKnownAgents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading agents index: %v\n", err)
		os.Exit(1)
	}

	// Validate plan
	if err := validatePlan(&plan, knownAgents); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if isSchemaValidationError(err) {
			os.Exit(2)
		}
		if isReferentialIntegrityError(err) {
			os.Exit(3)
		}
		os.Exit(1)
	}

	// Compute waves
	waves, err := computeWaves(plan.Tasks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if isReferentialIntegrityError(err) {
			os.Exit(3)
		}
		os.Exit(1)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate config.json
	configPath := filepath.Join(*outputDir, "config.json")
	if err := generateConfig(waves, *projectRoot, *outputDir, configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating config: %v\n", err)
		os.Exit(1)
	}

	// Generate stdin files
	if err := generateStdinFiles(plan, waves, *projectRoot, *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating stdin files: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("Generated %d tasks in %d waves → %s\n", len(plan.Tasks), len(waves), *outputDir)
}
