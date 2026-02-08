package main

import (
	"fmt"
	"sort"
	"strings"
)

// computeWaves performs topological sort using Kahn's algorithm
func computeWaves(tasks []Task) ([][]Task, error) {
	if len(tasks) == 0 {
		return [][]Task{}, nil
	}

	// Build task map for quick lookup
	taskMap := make(map[string]Task)
	for _, task := range tasks {
		taskMap[task.TaskID] = task
	}

	// Compute in-degree for each task
	inDegree := make(map[string]int)
	for _, task := range tasks {
		if _, exists := inDegree[task.TaskID]; !exists {
			inDegree[task.TaskID] = 0
		}
		for range task.BlockedBy {
			inDegree[task.TaskID]++
		}
	}

	// Track processed tasks
	processed := make(map[string]bool)
	waves := [][]Task{}

	// Iterate until all tasks processed or circular dependency detected
	for len(processed) < len(tasks) {
		// Find all tasks with in-degree 0
		currentWave := []Task{}
		for taskID, degree := range inDegree {
			if degree == 0 && !processed[taskID] {
				currentWave = append(currentWave, taskMap[taskID])
			}
		}

		// No tasks available but not all processed = circular dependency
		if len(currentWave) == 0 {
			remaining := []string{}
			for _, task := range tasks {
				if !processed[task.TaskID] {
					remaining = append(remaining, task.TaskID)
				}
			}
			sort.Strings(remaining)
			return nil, fmt.Errorf("%w: circular dependency detected among: %s",
				ErrReferentialIntegrity, strings.Join(remaining, ", "))
		}

		// Sort tasks within wave by task_id for deterministic output
		sort.Slice(currentWave, func(i, j int) bool {
			return currentWave[i].TaskID < currentWave[j].TaskID
		})

		waves = append(waves, currentWave)

		// Mark current wave tasks as processed and update in-degrees
		for _, task := range currentWave {
			processed[task.TaskID] = true

			// Find all tasks that depend on this task
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
