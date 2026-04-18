package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// ErrSchemaValidation indicates schema validation failure
	ErrSchemaValidation = errors.New("schema validation")
	// ErrReferentialIntegrity indicates referential integrity failure
	ErrReferentialIntegrity = errors.New("referential integrity")
)

var taskIDPattern = regexp.MustCompile(`^task-\d{3}$`)

// validatePlan orchestrates all validation checks
func validatePlan(plan *ImplementationPlan, knownAgents []string) error {
	// Check version
	if plan.Version != "1.0.0" {
		return fmt.Errorf("%w: version must be 1.0.0, got %q", ErrSchemaValidation, plan.Version)
	}

	// Check required project fields
	if plan.Project.Language == "" {
		return fmt.Errorf("%w: project.language is required", ErrSchemaValidation)
	}
	if plan.Project.ConventionsFile == "" {
		return fmt.Errorf("%w: project.conventions_file is required", ErrSchemaValidation)
	}

	// Check tasks exist
	if len(plan.Tasks) == 0 {
		return fmt.Errorf("%w: at least one task is required", ErrSchemaValidation)
	}

	// Track task IDs for uniqueness and referential integrity
	taskIDs := make(map[string]bool)

	// First pass: validate task structure and collect IDs
	for i, task := range plan.Tasks {
		// Check task_id format
		if !taskIDPattern.MatchString(task.TaskID) {
			return fmt.Errorf("%w: tasks[%d].task_id %q does not match pattern ^task-\\d{3}$",
				ErrSchemaValidation, i, task.TaskID)
		}

		// Check task_id uniqueness
		if taskIDs[task.TaskID] {
			return fmt.Errorf("%w: duplicate task_id %q", ErrSchemaValidation, task.TaskID)
		}
		taskIDs[task.TaskID] = true

		// Check required fields
		if task.Subject == "" {
			return fmt.Errorf("%w: tasks[%d].subject is required", ErrSchemaValidation, i)
		}
		if task.Description == "" {
			return fmt.Errorf("%w: tasks[%d].description is required", ErrSchemaValidation, i)
		}
		if task.Agent == "" {
			return fmt.Errorf("%w: tasks[%d].agent is required", ErrSchemaValidation, i)
		}
		if len(task.TargetPackages) == 0 {
			return fmt.Errorf("%w: tasks[%d].target_packages is required", ErrSchemaValidation, i)
		}
		if len(task.AcceptanceCriteria) == 0 {
			return fmt.Errorf("%w: tasks[%d].acceptance_criteria is required", ErrSchemaValidation, i)
		}
	}

	// Build list of all task IDs for error messages
	allTaskIDs := make([]string, 0, len(taskIDs))
	for id := range taskIDs {
		allTaskIDs = append(allTaskIDs, id)
	}

	// Second pass: validate agents and referential integrity
	agentMap := make(map[string]bool)
	for _, agent := range knownAgents {
		agentMap[agent] = true
	}

	for i, task := range plan.Tasks {
		// Check agent exists
		if !agentMap[task.Agent] {
			return fmt.Errorf("%w: tasks[%d].agent %q is not a known agent",
				ErrSchemaValidation, i, task.Agent)
		}

		// Check blocked_by references
		for j, depID := range task.BlockedBy {
			if !taskIDs[depID] {
				return fmt.Errorf(`Error: [referential_integrity] unknown dependency
  at: tasks[%d].blocked_by[%d]
  value: %q
  expected: one of [%s]
%w`, i, j, depID, strings.Join(allTaskIDs, ", "), ErrReferentialIntegrity)
			}
		}
	}

	return nil
}

// loadKnownAgents reads agents-index.json and returns all agent IDs
func loadKnownAgents() ([]string, error) {
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

// isSchemaValidationError checks if error is a schema validation error
func isSchemaValidationError(err error) bool {
	return errors.Is(err, ErrSchemaValidation)
}

// isReferentialIntegrityError checks if error is a referential integrity error
func isReferentialIntegrityError(err error) bool {
	return errors.Is(err, ErrReferentialIntegrity)
}
