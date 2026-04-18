package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImplementationPlan represents the input JSON schema
type ImplementationPlan struct {
	Version           string             `json:"version"`
	Project           Project            `json:"project"`
	Tasks             []Task             `json:"tasks"`
	ReviewAnnotations []ReviewAnnotation `json:"review_annotations,omitempty"`
	ReadinessScore    *ReadinessScore    `json:"readiness_score,omitempty"`
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
	TaskID               string               `json:"task_id"`
	Subject              string               `json:"subject"`
	Description          string               `json:"description"`
	Agent                string               `json:"agent"`
	TargetPackages       []string             `json:"target_packages"`
	RelatedFiles         []RelatedFile        `json:"related_files,omitempty"`
	BlockedBy            []string             `json:"blocked_by"`
	AcceptanceCriteria   []string             `json:"acceptance_criteria"`
	TestsRequired        *bool                `json:"tests_required,omitempty"`
	CoverageTarget       *int                 `json:"coverage_target,omitempty"`
	ImplicitDependencies []ImplicitDependency `json:"implicit_dependencies,omitempty"`
}

// RelatedFile represents a file related to a task
type RelatedFile struct {
	Path      string `json:"path"`
	Relevance string `json:"relevance"`
}

// ImplicitDependency represents an inferred (not promoted) dependency from plan-harmonizer
type ImplicitDependency struct {
	DependsOn  string  `json:"depends_on"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Promoted   bool    `json:"promoted"`
}

// ReviewAnnotation represents a review finding mapped to tasks by plan-harmonizer
type ReviewAnnotation struct {
	FindingID                string   `json:"finding_id"`
	Severity                 string   `json:"severity"`
	Classification           string   `json:"classification"`
	ClassificationConfidence float64  `json:"classification_confidence"`
	MappedTasks              []string `json:"mapped_tasks"`
	Description              string   `json:"description"`
	Recommendation           string   `json:"recommendation"`
	AutoApplied              bool     `json:"auto_applied"`
}

// ReadinessScore represents the enriched plan readiness score from plan-harmonizer
type ReadinessScore struct {
	Total      int            `json:"total"`
	Dimensions map[string]int `json:"dimensions"`
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

	// Emit implicit dependency warnings (enrichment — warnings only, never blocks)
	for _, w := range warnImplicitDeps(&plan) {
		fmt.Fprintln(os.Stderr, w)
	}

	// Emit readiness score summary (enrichment — before wave execution)
	if score := formatReadinessScore(&plan); score != "" {
		fmt.Print(score)
	}
	if w := readinessScoreWarning(&plan); w != "" {
		fmt.Fprintln(os.Stderr, w)
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

// warnImplicitDeps returns warning lines for unpromoted implicit dependencies.
// Returns nil when no tasks have unpromoted implicit dependencies.
func warnImplicitDeps(plan *ImplementationPlan) []string {
	var warnings []string
	for _, task := range plan.Tasks {
		for _, dep := range task.ImplicitDependencies {
			if dep.Promoted {
				continue
			}
			warnings = append(warnings,
				fmt.Sprintf("⚠ %s has unpromoted implicit dependency on %s (confidence: %.2f)",
					task.TaskID, dep.DependsOn, dep.Confidence),
				fmt.Sprintf("  Reason: %s", dep.Reason),
				fmt.Sprintf("  Consider: /refine-plan --promote-dep %s:%s", task.TaskID, dep.DependsOn),
			)
		}
	}
	return warnings
}

// formatReadinessScore returns the readiness score summary string for stdout.
// Returns "" when no readiness_score is present in the plan.
func formatReadinessScore(plan *ImplementationPlan) string {
	if plan.ReadinessScore == nil {
		return ""
	}
	rs := plan.ReadinessScore

	label := "not ready"
	switch {
	case rs.Total >= 70:
		label = "ready"
	case rs.Total >= 50:
		label = "caveats"
	}

	// Sort dimension names for deterministic output (map iteration is random)
	dimNames := make([]string, 0, len(rs.Dimensions))
	for k := range rs.Dimensions {
		dimNames = append(dimNames, k)
	}
	sort.Strings(dimNames)

	dimParts := make([]string, len(dimNames))
	for i, k := range dimNames {
		dimParts[i] = fmt.Sprintf("%s: %d/5", k, rs.Dimensions[k])
	}

	return fmt.Sprintf("Readiness Score: %d/100 (%s)\n  %s\n",
		rs.Total, label, strings.Join(dimParts, " | "))
}

// readinessScoreWarning returns a warning string when readiness_score.total < 50.
// Returns "" when no readiness_score is present or the score is >= 50.
func readinessScoreWarning(plan *ImplementationPlan) string {
	if plan.ReadinessScore == nil || plan.ReadinessScore.Total >= 50 {
		return ""
	}
	return fmt.Sprintf("⚠ Readiness score %d/100 < 50 — implementation may encounter significant gaps",
		plan.ReadinessScore.Total)
}
