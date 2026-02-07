package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

// Spawner defines the interface for spawning Claude CLI processes.
// Production code uses claudeSpawner; tests inject fakes.
type Spawner interface {
	Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error
}

// claudeSpawner is the production Spawner implementation.
// TC-008 will replace the Spawn method body with actual CLI invocation.
type claudeSpawner struct{}

func (s *claudeSpawner) Spawn(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int) error {
	return fmt.Errorf("spawnClaude not yet implemented (TC-008)")
}

// spawnAndWait spawns a Claude CLI process for a team member and waits for completion.
// Uses iterative retry (NOT recursive) to avoid WaitGroup panics.
//
// CONTRACT: The caller MUST call wg.Add(1) exactly once before invoking this function.
// This function calls wg.Done() exactly once via defer, matching the single Add(1).
// Violating this contract causes a WaitGroup counter underflow panic.
//
// Usage:
//
//	wg.Add(1)
//	go spawnAndWait(ctx, tr, 0, 0, &wg)
//	wg.Wait()
//
// Retry logic:
//   - Attempts up to member.MaxRetries + 1 times (0-indexed)
//   - Checks context cancellation before each attempt
//   - Updates member status and error history throughout
//   - Returns on first success or after exhausting all retries
func spawnAndWait(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get member info snapshot for retry loop
	tr.configMu.RLock()
	if tr.config == nil || waveIdx >= len(tr.config.Waves) || memIdx >= len(tr.config.Waves[waveIdx].Members) {
		tr.configMu.RUnlock()
		log.Printf("ERROR: Invalid wave/member indices: wave=%d, member=%d", waveIdx, memIdx)
		return
	}
	member := tr.config.Waves[waveIdx].Members[memIdx]
	tr.configMu.RUnlock()

	var errorHistory []string

	// Iterative retry loop (NOT recursive)
	for attempt := 0; attempt <= member.MaxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "failed"
				ctxErr := fmt.Sprintf("context cancelled: %v", ctx.Err())
				if len(errorHistory) > 0 {
					m.ErrorMessage = strings.Join(errorHistory, "; ") + "; " + ctxErr
				} else {
					m.ErrorMessage = ctxErr
				}
			}); err != nil {
				log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			}
			return
		}

		// Mark as running
		if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
			m.Status = "running"
			m.RetryCount = attempt
		}); err != nil {
			log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			continue
		}

		// Attempt spawn
		err := tr.spawner.Spawn(ctx, tr, waveIdx, memIdx)
		if err == nil {
			// Success - mark as completed and return
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "completed"
			}); err != nil {
				log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
			}
			log.Printf("Member %s completed successfully (attempt %d)", member.Name, attempt)
			return
		}

		// Spawn failed - log and update error message
		log.Printf("Spawn attempt %d for %s failed: %v", attempt, member.Name, err)
		errMsg := fmt.Sprintf("attempt %d: %v", attempt, err)
		errorHistory = append(errorHistory, errMsg)
		if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
			m.ErrorMessage = strings.Join(errorHistory, "; ")
		}); err != nil {
			log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
		}

		// Continue to next retry (if any left)
	}

	// All retries exhausted - mark as failed
	if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
		m.Status = "failed"
	}); err != nil {
		log.Printf("ERROR: Failed to update member %s: %v", member.Name, err)
	}
	log.Printf("Member %s failed after %d retries", member.Name, member.MaxRetries+1)
}

