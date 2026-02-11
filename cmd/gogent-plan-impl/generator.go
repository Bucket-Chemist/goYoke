package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// genTeamConfig matches the TeamConfig structure expected by gogent-team-run
type genTeamConfig struct {
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
	Waves               []genWave `json:"waves"`
}

// genWave represents a wave of parallel tasks
type genWave struct {
	WaveNumber       int         `json:"wave_number"`
	Description      string      `json:"description"`
	Members          []genMember `json:"members"`
	OnCompleteScript *string     `json:"on_complete_script"`
}

// genMember represents a team member (task executor)
type genMember struct {
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

// generateConfig creates the config.json file
func generateConfig(waves [][]Task, projectRoot, teamDir, configPath string) error {
	now := time.Now()
	timestamp := now.Unix()

	config := genTeamConfig{
		TeamName:            fmt.Sprintf("implementation-%d", timestamp),
		WorkflowType:        "implementation",
		ProjectRoot:         projectRoot,
		SessionID:           uuid.New().String(),
		CreatedAt:           now.Format(time.RFC3339),
		BudgetMaxUSD:        10.0,
		BudgetRemainingUSD:  10.0,
		WarningThresholdUSD: 8.0,
		Status:              "pending",
		BackgroundPID:       nil,
		StartedAt:           nil,
		CompletedAt:         nil,
		Waves:               make([]genWave, len(waves)),
	}

	// Generate waves (1-indexed)
	for i, waveTasks := range waves {
		waveNum := i + 1
		wave := genWave{
			WaveNumber:       waveNum,
			Description:      fmt.Sprintf("Wave %d: %d tasks", waveNum, len(waveTasks)),
			Members:          make([]genMember, len(waveTasks)),
			OnCompleteScript: nil,
		}

		for j, task := range waveTasks {
			wave.Members[j] = genMember{
				Name:         task.TaskID,
				Agent:        task.Agent,
				Model:        "sonnet",
				StdinFile:    fmt.Sprintf("stdin_%s.json", task.TaskID),
				StdoutFile:   fmt.Sprintf("stdout_%s.json", task.TaskID),
				Status:       "pending",
				ProcessPID:   nil,
				ExitCode:     nil,
				CostUSD:      0,
				CostStatus:   "",
				ErrorMessage: "",
				RetryCount:   0,
				MaxRetries:   2,
				TimeoutMs:    300000,
				StartedAt:    nil,
				CompletedAt:  nil,
			}
		}

		config.Waves[i] = wave
	}

	// Write config with pretty printing
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// stdinSchema represents the stdin JSON structure for each task
type stdinSchema struct {
	Agent              string                 `json:"agent"`
	Workflow           string                 `json:"workflow"`
	Context            stdinContext           `json:"context"`
	Task               stdinTask              `json:"task"`
	ImplementationScope stdinImplScope        `json:"implementation_scope"`
	Conventions        stdinConventions       `json:"conventions"`
	CodebaseContext    stdinCodebaseContext   `json:"codebase_context"`
	Description        string                 `json:"description"`
}

type stdinContext struct {
	ProjectRoot string `json:"project_root"`
	TeamDir     string `json:"team_dir"`
}

type stdinTask struct {
	TaskID             string   `json:"task_id"`
	Subject            string   `json:"subject"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	BlockedBy          []string `json:"blocked_by"`
	Blocks             []string `json:"blocks"`
}

type stdinImplScope struct {
	TargetPackages     []string      `json:"target_packages"`
	RelatedFiles       []RelatedFile `json:"related_files,omitempty"`
	TestsRequired      bool          `json:"tests_required"`
	BuildVerification  string        `json:"build_verification,omitempty"`
	CoverageTarget     *int          `json:"coverage_target,omitempty"`
}

type stdinConventions struct {
	Language        string `json:"language"`
	ConventionsFile string `json:"conventions_file"`
	ErrorHandling   string `json:"error_handling,omitempty"`
	TestPattern     string `json:"test_pattern,omitempty"`
}

type stdinCodebaseContext struct {
	ArchitectureNotes string   `json:"architecture_notes,omitempty"`
	PatternsToFollow  []string `json:"patterns_to_follow,omitempty"`
	AntiPatterns      []string `json:"anti_patterns,omitempty"`
}

// generateStdinFiles creates stdin JSON files for each task
func generateStdinFiles(plan ImplementationPlan, waves [][]Task, projectRoot, teamDir string) error {
	// Build reverse-lookup map for blocks relationships
	blocksMap := make(map[string][]string)
	for _, task := range plan.Tasks {
		for _, dep := range task.BlockedBy {
			blocksMap[dep] = append(blocksMap[dep], task.TaskID)
		}
	}

	// Generate stdin file for each task
	for _, wave := range waves {
		for _, task := range wave {
			stdin := stdinSchema{
				Agent:    task.Agent,
				Workflow: "implementation",
				Context: stdinContext{
					ProjectRoot: projectRoot,
					TeamDir:     teamDir,
				},
				Task: stdinTask{
					TaskID:             task.TaskID,
					Subject:            task.Subject,
					Description:        task.Description,
					AcceptanceCriteria: task.AcceptanceCriteria,
					BlockedBy:          task.BlockedBy,
					Blocks:             blocksMap[task.TaskID],
				},
				ImplementationScope: stdinImplScope{
					TargetPackages:    task.TargetPackages,
					RelatedFiles:      task.RelatedFiles,
					TestsRequired:     true,
					BuildVerification: plan.Project.BuildVerification,
					CoverageTarget:    task.CoverageTarget,
				},
				Conventions: stdinConventions{
					Language:        inferLanguage(task.Agent, plan.Project.Language),
					ConventionsFile: inferConventionsFile(task.Agent, plan.Project.ConventionsFile),
					ErrorHandling:   plan.Project.ErrorHandling,
					TestPattern:     plan.Project.TestPattern,
				},
				CodebaseContext: stdinCodebaseContext{
					ArchitectureNotes: plan.Project.ArchitectureNotes,
					PatternsToFollow:  plan.Project.PatternsToFollow,
					AntiPatterns:      plan.Project.AntiPatterns,
				},
				Description: fmt.Sprintf("Implement: %s", task.Subject),
			}

			// Override tests_required if explicitly set
			if task.TestsRequired != nil {
				stdin.ImplementationScope.TestsRequired = *task.TestsRequired
			}

			// Ensure blocks is never nil
			if stdin.Task.Blocks == nil {
				stdin.Task.Blocks = []string{}
			}
			if stdin.Task.BlockedBy == nil {
				stdin.Task.BlockedBy = []string{}
			}

			// Write stdin file
			filename := fmt.Sprintf("stdin_%s.json", task.TaskID)
			filePath := filepath.Join(teamDir, filename)

			data, err := json.MarshalIndent(stdin, "", "  ")
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

// inferLanguage infers language from agent if not provided
func inferLanguage(agent, projectLanguage string) string {
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

// inferConventionsFile infers conventions file from agent if not provided
func inferConventionsFile(agent, projectConventions string) string {
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
