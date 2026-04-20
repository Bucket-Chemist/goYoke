package utils

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunPlanImpl implements the goyoke-plan-impl utility.
func RunPlanImpl(_ context.Context, args []string, _ io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("plan-impl", flag.ContinueOnError)
	planPath := fs.String("plan", "", "Path to implementation-plan.json")
	projectRoot := fs.String("project-root", "", "Absolute path to project root")
	outputDir := fs.String("output", "", "Team directory for generated files")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("plan-impl: %w", err)
	}

	if *planPath == "" || *projectRoot == "" || *outputDir == "" {
		return fmt.Errorf("plan-impl: --plan, --project-root, and --output are required")
	}

	planData, err := os.ReadFile(*planPath)
	if err != nil {
		return fmt.Errorf("plan-impl: read plan file: %w", err)
	}

	var plan piPlan
	if err := json.Unmarshal(planData, &plan); err != nil {
		return fmt.Errorf("plan-impl: parse plan JSON: %w", err)
	}

	knownAgents, err := piLoadKnownAgents()
	if err != nil {
		return fmt.Errorf("plan-impl: load agents index: %w", err)
	}

	if err := piValidatePlan(&plan, knownAgents); err != nil {
		if piIsSchemaValidationError(err) {
			return fmt.Errorf("plan-impl: validation (schema): %w", err)
		}
		if piIsReferentialIntegrityError(err) {
			return fmt.Errorf("plan-impl: validation (integrity): %w", err)
		}
		return fmt.Errorf("plan-impl: validation: %w", err)
	}

	for _, w := range piWarnImplicitDeps(&plan) {
		fmt.Fprintln(os.Stderr, w)
	}

	if score := piFormatReadinessScore(&plan); score != "" {
		fmt.Fprint(stdout, score)
	}
	if w := piReadinessScoreWarning(&plan); w != "" {
		fmt.Fprintln(os.Stderr, w)
	}

	waves, err := piComputeWaves(plan.Tasks)
	if err != nil {
		if piIsReferentialIntegrityError(err) {
			return fmt.Errorf("plan-impl: compute waves (integrity): %w", err)
		}
		return fmt.Errorf("plan-impl: compute waves: %w", err)
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		return fmt.Errorf("plan-impl: create output directory: %w", err)
	}

	configPath := filepath.Join(*outputDir, "config.json")
	if err := piGenerateConfig(waves, *projectRoot, *outputDir, configPath); err != nil {
		return fmt.Errorf("plan-impl: generate config: %w", err)
	}

	if err := piGenerateStdinFiles(plan, waves, *projectRoot, *outputDir); err != nil {
		return fmt.Errorf("plan-impl: generate stdin files: %w", err)
	}

	fmt.Fprintf(stdout, "Generated %d tasks in %d waves → %s\n", len(plan.Tasks), len(waves), *outputDir)
	return nil
}

// --- Types ---

type piPlan struct {
	Version           string               `json:"version"`
	Project           piProject            `json:"project"`
	Tasks             []piTask             `json:"tasks"`
	ReviewAnnotations []piReviewAnnotation `json:"review_annotations,omitempty"`
	ReadinessScore    *piReadinessScore    `json:"readiness_score,omitempty"`
}

type piProject struct {
	Language          string   `json:"language"`
	ConventionsFile   string   `json:"conventions_file"`
	BuildVerification string   `json:"build_verification,omitempty"`
	ErrorHandling     string   `json:"error_handling,omitempty"`
	TestPattern       string   `json:"test_pattern,omitempty"`
	ArchitectureNotes string   `json:"architecture_notes,omitempty"`
	PatternsToFollow  []string `json:"patterns_to_follow,omitempty"`
	AntiPatterns      []string `json:"anti_patterns,omitempty"`
}

type piTask struct {
	TaskID               string               `json:"task_id"`
	Subject              string               `json:"subject"`
	Description          string               `json:"description"`
	Agent                string               `json:"agent"`
	TargetPackages       []string             `json:"target_packages"`
	RelatedFiles         []piRelatedFile      `json:"related_files,omitempty"`
	BlockedBy            []string             `json:"blocked_by"`
	AcceptanceCriteria   []string             `json:"acceptance_criteria"`
	TestsRequired        *bool                `json:"tests_required,omitempty"`
	CoverageTarget       *int                 `json:"coverage_target,omitempty"`
	ImplicitDependencies []piImplicitDep      `json:"implicit_dependencies,omitempty"`
}

type piRelatedFile struct {
	Path      string `json:"path"`
	Relevance string `json:"relevance"`
}

type piImplicitDep struct {
	DependsOn  string  `json:"depends_on"`
	Reason     string  `json:"reason"`
	Confidence float64 `json:"confidence"`
	Promoted   bool    `json:"promoted"`
}

type piReviewAnnotation struct {
	FindingID                string   `json:"finding_id"`
	Severity                 string   `json:"severity"`
	Classification           string   `json:"classification"`
	ClassificationConfidence float64  `json:"classification_confidence"`
	MappedTasks              []string `json:"mapped_tasks"`
	Description              string   `json:"description"`
	Recommendation           string   `json:"recommendation"`
	AutoApplied              bool     `json:"auto_applied"`
}

type piReadinessScore struct {
	Total      int            `json:"total"`
	Dimensions map[string]int `json:"dimensions"`
}

// --- Sentinel errors ---

var (
	piErrSchemaValidation    = errors.New("schema validation")
	piErrReferentialIntegrity = errors.New("referential integrity")
)

var piTaskIDPattern = regexp.MustCompile(`^task-\d{3}$`)

// --- Validation ---

func piValidatePlan(plan *piPlan, knownAgents []string) error {
	if plan.Version != "1.0.0" {
		return fmt.Errorf("%w: version must be 1.0.0, got %q", piErrSchemaValidation, plan.Version)
	}
	if plan.Project.Language == "" {
		return fmt.Errorf("%w: project.language is required", piErrSchemaValidation)
	}
	if plan.Project.ConventionsFile == "" {
		return fmt.Errorf("%w: project.conventions_file is required", piErrSchemaValidation)
	}
	if len(plan.Tasks) == 0 {
		return fmt.Errorf("%w: at least one task is required", piErrSchemaValidation)
	}

	taskIDs := make(map[string]bool)
	for i, task := range plan.Tasks {
		if !piTaskIDPattern.MatchString(task.TaskID) {
			return fmt.Errorf("%w: tasks[%d].task_id %q does not match pattern ^task-\\d{3}$",
				piErrSchemaValidation, i, task.TaskID)
		}
		if taskIDs[task.TaskID] {
			return fmt.Errorf("%w: duplicate task_id %q", piErrSchemaValidation, task.TaskID)
		}
		taskIDs[task.TaskID] = true
		if task.Subject == "" {
			return fmt.Errorf("%w: tasks[%d].subject is required", piErrSchemaValidation, i)
		}
		if task.Description == "" {
			return fmt.Errorf("%w: tasks[%d].description is required", piErrSchemaValidation, i)
		}
		if task.Agent == "" {
			return fmt.Errorf("%w: tasks[%d].agent is required", piErrSchemaValidation, i)
		}
		if len(task.TargetPackages) == 0 {
			return fmt.Errorf("%w: tasks[%d].target_packages is required", piErrSchemaValidation, i)
		}
		if len(task.AcceptanceCriteria) == 0 {
			return fmt.Errorf("%w: tasks[%d].acceptance_criteria is required", piErrSchemaValidation, i)
		}
	}

	allTaskIDs := make([]string, 0, len(taskIDs))
	for id := range taskIDs {
		allTaskIDs = append(allTaskIDs, id)
	}

	agentMap := make(map[string]bool)
	for _, agent := range knownAgents {
		agentMap[agent] = true
	}

	for i, task := range plan.Tasks {
		if !agentMap[task.Agent] {
			return fmt.Errorf("%w: tasks[%d].agent %q is not a known agent",
				piErrSchemaValidation, i, task.Agent)
		}
		for j, depID := range task.BlockedBy {
			if !taskIDs[depID] {
				return fmt.Errorf(`Error: [referential_integrity] unknown dependency
  at: tasks[%d].blocked_by[%d]
  value: %q
  expected: one of [%s]
%w`, i, j, depID, strings.Join(allTaskIDs, ", "), piErrReferentialIntegrity)
			}
		}
	}

	return nil
}

func piLoadKnownAgents() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	indexPath := filepath.Join(homeDir, ".claude", "agents", "agents-index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read agents-index.json: %w", err)
	}

	var index struct {
		Agents []struct {
			ID string `json:"id"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse agents-index.json: %w", err)
	}

	agents := make([]string, len(index.Agents))
	for i, agent := range index.Agents {
		agents[i] = agent.ID
	}
	return agents, nil
}

func piIsSchemaValidationError(err error) bool {
	return errors.Is(err, piErrSchemaValidation)
}

func piIsReferentialIntegrityError(err error) bool {
	return errors.Is(err, piErrReferentialIntegrity)
}

// --- Wave computation ---

func piComputeWaves(tasks []piTask) ([][]piTask, error) {
	if len(tasks) == 0 {
		return [][]piTask{}, nil
	}

	taskMap := make(map[string]piTask)
	for _, task := range tasks {
		taskMap[task.TaskID] = task
	}

	inDegree := make(map[string]int)
	for _, task := range tasks {
		if _, exists := inDegree[task.TaskID]; !exists {
			inDegree[task.TaskID] = 0
		}
		for range task.BlockedBy {
			inDegree[task.TaskID]++
		}
	}

	processed := make(map[string]bool)
	waves := [][]piTask{}

	for len(processed) < len(tasks) {
		var currentWave []piTask
		for taskID, degree := range inDegree {
			if degree == 0 && !processed[taskID] {
				currentWave = append(currentWave, taskMap[taskID])
			}
		}

		if len(currentWave) == 0 {
			remaining := []string{}
			for _, task := range tasks {
				if !processed[task.TaskID] {
					remaining = append(remaining, task.TaskID)
				}
			}
			sort.Strings(remaining)
			return nil, fmt.Errorf("%w: circular dependency detected among: %s",
				piErrReferentialIntegrity, strings.Join(remaining, ", "))
		}

		sort.Slice(currentWave, func(i, j int) bool {
			return currentWave[i].TaskID < currentWave[j].TaskID
		})
		waves = append(waves, currentWave)

		for _, task := range currentWave {
			processed[task.TaskID] = true
			for _, otherTask := range tasks {
				for _, dep := range otherTask.BlockedBy {
					if dep == task.TaskID {
						inDegree[otherTask.TaskID]--
					}
				}
			}
		}
	}

	return waves, nil
}

// --- Enrichment helpers ---

func piWarnImplicitDeps(plan *piPlan) []string {
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

func piFormatReadinessScore(plan *piPlan) string {
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

func piReadinessScoreWarning(plan *piPlan) string {
	if plan.ReadinessScore == nil || plan.ReadinessScore.Total >= 50 {
		return ""
	}
	return fmt.Sprintf("⚠ Readiness score %d/100 < 50 — implementation may encounter significant gaps",
		plan.ReadinessScore.Total)
}

// --- Config generation types ---

type piTeamConfig struct {
	TeamName            string    `json:"team_name"`
	WorkflowType        string    `json:"workflow_type"`
	ProjectRoot         string    `json:"project_root"`
	SessionID           string    `json:"session_id"`
	CreatedAt           string    `json:"created_at"`
	BudgetMaxUSD        float64   `json:"budget_max_usd"`
	BudgetRemainingUSD  float64   `json:"budget_remaining_usd"`
	WarningThresholdUSD float64   `json:"warning_threshold_usd"`
	Status              string    `json:"status"`
	BackgroundPID       *int      `json:"background_pid"`
	StartedAt           *string   `json:"started_at"`
	CompletedAt         *string   `json:"completed_at"`
	Waves               []piWave  `json:"waves"`
}

type piWave struct {
	WaveNumber       int        `json:"wave_number"`
	Description      string     `json:"description"`
	Members          []piMember `json:"members"`
	OnCompleteScript *string    `json:"on_complete_script"`
}

type piMember struct {
	Name         string  `json:"name"`
	Agent        string  `json:"agent"`
	Model        string  `json:"model"`
	StdinFile    string  `json:"stdin_file"`
	StdoutFile   string  `json:"stdout_file"`
	Status       string  `json:"status"`
	ProcessPID   *int    `json:"process_pid"`
	ExitCode     *int    `json:"exit_code"`
	CostUSD      float64 `json:"cost_usd"`
	CostStatus   string  `json:"cost_status"`
	ErrorMessage string  `json:"error_message"`
	RetryCount   int     `json:"retry_count"`
	MaxRetries   int     `json:"max_retries"`
	TimeoutMs    int     `json:"timeout_ms"`
	StartedAt    *string `json:"started_at"`
	CompletedAt  *string `json:"completed_at"`
}

func piGenerateConfig(waves [][]piTask, projectRoot, teamDir, configPath string) error {
	now := time.Now()
	timestamp := now.Unix()

	config := piTeamConfig{
		TeamName:            fmt.Sprintf("implementation-%d", timestamp),
		WorkflowType:        "implementation",
		ProjectRoot:         projectRoot,
		SessionID:           uuid.New().String(),
		CreatedAt:           now.Format(time.RFC3339),
		BudgetMaxUSD:        10.0,
		BudgetRemainingUSD:  10.0,
		WarningThresholdUSD: 8.0,
		Status:              "pending",
		Waves:               make([]piWave, len(waves)),
	}

	for i, waveTasks := range waves {
		waveNum := i + 1
		wave := piWave{
			WaveNumber:  waveNum,
			Description: fmt.Sprintf("Wave %d: %d tasks", waveNum, len(waveTasks)),
			Members:     make([]piMember, len(waveTasks)),
		}

		for j, task := range waveTasks {
			wave.Members[j] = piMember{
				Name:         task.TaskID,
				Agent:        task.Agent,
				Model:        "sonnet",
				StdinFile:    fmt.Sprintf("stdin_%s.json", task.TaskID),
				StdoutFile:   fmt.Sprintf("stdout_%s.json", task.TaskID),
				Status:       "pending",
				CostUSD:      0,
				RetryCount:   0,
				MaxRetries:   2,
				TimeoutMs:    300000,
			}
		}
		config.Waves[i] = wave
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// --- Stdin file generation ---

type piStdinSchema struct {
	Agent               string             `json:"agent"`
	Workflow            string             `json:"workflow"`
	Context             piStdinContext      `json:"context"`
	Task                piStdinTask        `json:"task"`
	ImplementationScope piStdinImplScope   `json:"implementation_scope"`
	Conventions         piStdinConventions `json:"conventions"`
	CodebaseContext     piStdinCodebase    `json:"codebase_context"`
	Description         string             `json:"description"`
	ReviewFindings      *piReviewFindings  `json:"review_findings,omitempty"`
}

type piStdinContext struct {
	ProjectRoot string `json:"project_root"`
	TeamDir     string `json:"team_dir"`
}

type piStdinTask struct {
	TaskID             string   `json:"task_id"`
	Subject            string   `json:"subject"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	BlockedBy          []string `json:"blocked_by"`
	Blocks             []string `json:"blocks"`
}

type piStdinImplScope struct {
	TargetPackages    []piRelatedFile `json:"related_files,omitempty"`
	TestsRequired     bool            `json:"tests_required"`
	BuildVerification string          `json:"build_verification,omitempty"`
	CoverageTarget    *int            `json:"coverage_target,omitempty"`
	TargetPkgs        []string        `json:"target_packages"`
}

type piStdinConventions struct {
	Language        string `json:"language"`
	ConventionsFile string `json:"conventions_file"`
	ErrorHandling   string `json:"error_handling,omitempty"`
	TestPattern     string `json:"test_pattern,omitempty"`
}

type piStdinCodebase struct {
	ArchitectureNotes string   `json:"architecture_notes,omitempty"`
	PatternsToFollow  []string `json:"patterns_to_follow,omitempty"`
	AntiPatterns      []string `json:"anti_patterns,omitempty"`
}

type piReviewFindings struct {
	CorrectionsToAddress []string `json:"corrections_to_address,omitempty"`
	ReviewNotes          []string `json:"review_notes,omitempty"`
	FixesIncorporated    []string `json:"fixes_incorporated,omitempty"`
}

func piGenerateStdinFiles(plan piPlan, waves [][]piTask, projectRoot, teamDir string) error {
	blocksMap := make(map[string][]string)
	for _, task := range plan.Tasks {
		for _, dep := range task.BlockedBy {
			blocksMap[dep] = append(blocksMap[dep], task.TaskID)
		}
	}

	type taskAnnotations struct {
		corrections []string
		notes       []string
		fixes       []string
	}
	annotationsByTask := make(map[string]*taskAnnotations)
	for _, ann := range plan.ReviewAnnotations {
		var entry, bucket string
		if ann.AutoApplied {
			entry = fmt.Sprintf("[%s] %s (auto-applied)", ann.FindingID, ann.Recommendation)
			bucket = "fixes"
		} else if ann.Classification == "correction" {
			entry = fmt.Sprintf("[%s] %s", ann.FindingID, ann.Recommendation)
			bucket = "corrections"
		} else {
			entry = fmt.Sprintf("[%s] %s", ann.FindingID, ann.Recommendation)
			bucket = "notes"
		}
		for _, taskID := range ann.MappedTasks {
			if annotationsByTask[taskID] == nil {
				annotationsByTask[taskID] = &taskAnnotations{}
			}
			ta := annotationsByTask[taskID]
			switch bucket {
			case "fixes":
				ta.fixes = append(ta.fixes, entry)
			case "corrections":
				ta.corrections = append(ta.corrections, entry)
			default:
				ta.notes = append(ta.notes, entry)
			}
		}
	}

	for _, wave := range waves {
		for _, task := range wave {
			stdinDoc := piStdinSchema{
				Agent:    task.Agent,
				Workflow: "implementation",
				Context: piStdinContext{
					ProjectRoot: projectRoot,
					TeamDir:     teamDir,
				},
				Task: piStdinTask{
					TaskID:             task.TaskID,
					Subject:            task.Subject,
					Description:        task.Description,
					AcceptanceCriteria: task.AcceptanceCriteria,
					BlockedBy:          task.BlockedBy,
					Blocks:             blocksMap[task.TaskID],
				},
				ImplementationScope: piStdinImplScope{
					TargetPkgs:        task.TargetPackages,
					TargetPackages:    task.RelatedFiles,
					TestsRequired:     true,
					BuildVerification: plan.Project.BuildVerification,
					CoverageTarget:    task.CoverageTarget,
				},
				Conventions: piStdinConventions{
					Language:        piInferLanguage(task.Agent, plan.Project.Language),
					ConventionsFile: piInferConventionsFile(task.Agent, plan.Project.ConventionsFile),
					ErrorHandling:   plan.Project.ErrorHandling,
					TestPattern:     plan.Project.TestPattern,
				},
				CodebaseContext: piStdinCodebase{
					ArchitectureNotes: plan.Project.ArchitectureNotes,
					PatternsToFollow:  plan.Project.PatternsToFollow,
					AntiPatterns:      plan.Project.AntiPatterns,
				},
				Description: fmt.Sprintf("Implement: %s", task.Subject),
			}

			if task.TestsRequired != nil {
				stdinDoc.ImplementationScope.TestsRequired = *task.TestsRequired
			}

			if stdinDoc.Task.Blocks == nil {
				stdinDoc.Task.Blocks = []string{}
			}
			if stdinDoc.Task.BlockedBy == nil {
				stdinDoc.Task.BlockedBy = []string{}
			}

			if ta := annotationsByTask[task.TaskID]; ta != nil {
				rf := &piReviewFindings{}
				if len(ta.corrections) > 0 {
					rf.CorrectionsToAddress = ta.corrections
				}
				if len(ta.notes) > 0 {
					rf.ReviewNotes = ta.notes
				}
				if len(ta.fixes) > 0 {
					rf.FixesIncorporated = ta.fixes
				}
				stdinDoc.ReviewFindings = rf
			}

			filename := fmt.Sprintf("stdin_%s.json", task.TaskID)
			filePath := filepath.Join(teamDir, filename)

			data, err := json.MarshalIndent(stdinDoc, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal stdin for %s: %w", task.TaskID, err)
			}
			if err := os.WriteFile(filePath, data, 0644); err != nil {
				return fmt.Errorf("write stdin for %s: %w", task.TaskID, err)
			}
		}
	}

	return nil
}

func piInferLanguage(agent, projectLanguage string) string {
	if projectLanguage != "" {
		return projectLanguage
	}
	switch agent {
	case "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent":
		return "go"
	case "python-pro", "python-ux":
		return "python"
	case "typescript-pro", "react-pro":
		return "typescript"
	case "r-pro", "r-shiny-pro":
		return "r"
	default:
		return ""
	}
}

func piInferConventionsFile(agent, projectConventions string) string {
	if projectConventions != "" {
		return projectConventions
	}
	switch agent {
	case "go-pro", "go-cli", "go-tui", "go-api", "go-concurrent":
		return "go.md"
	case "python-pro", "python-ux":
		return "python.md"
	case "typescript-pro", "react-pro":
		return "typescript.md"
	case "r-pro", "r-shiny-pro":
		return "R.md"
	default:
		return ""
	}
}
