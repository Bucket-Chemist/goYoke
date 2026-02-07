package main

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// runWaves executes all waves sequentially.
// Within each wave, members execute in parallel via spawnAndWait goroutines.
// Budget gates prevent spawns when budget is insufficient.
func runWaves(ctx context.Context, tr *TeamRunner) error {
	for waveIdx, wave := range tr.config.Waves {
		log.Printf("[INFO] wave: Wave %d starting with %d members", wave.WaveNumber, len(wave.Members))

		// Check budget before wave
		if tr.BudgetRemaining() <= 0 {
			return fmt.Errorf("budget exhausted before wave %d", wave.WaveNumber)
		}

		// Execute wave members in parallel
		var wg sync.WaitGroup
		for memberIdx := range wave.Members {
			// Select context check
			select {
			case <-ctx.Done():
				log.Printf("[INFO] wave: context cancelled, stopping wave %d", wave.WaveNumber)
				wg.Wait()
				return ctx.Err()
			default:
			}

			estimated := tr.estimateCost(wave.Members[memberIdx].Agent)
			if !tr.tryReserveBudget(estimated) {
				log.Printf("[WARN] wave: budget gate blocked %s ($%.2f needed, $%.2f remaining)",
					wave.Members[memberIdx].Name, estimated, tr.BudgetRemaining())
				break // Stop spawning this wave
			}

			wg.Add(1)
			go func(wIdx, mIdx int, est float64) {
				defer wg.Done()
				spawnAndWaitWithBudget(ctx, tr, wIdx, mIdx, est)
			}(waveIdx, memberIdx, estimated)
		}

		wg.Wait()

		// Inter-wave script (TC-010 hook point)
		if wave.OnCompleteScript != nil && *wave.OnCompleteScript != "" {
			if err := runInterWaveScript(ctx, *wave.OnCompleteScript, tr.teamDir); err != nil {
				return fmt.Errorf("wave %d inter-wave script: %w", wave.WaveNumber, err)
			}
		}

		log.Printf("[INFO] wave: Wave %d completed", wave.WaveNumber)
	}

	return nil
}

// spawnAndWaitWithBudget wraps spawnAndWait with budget reconciliation.
// Reconciles estimated cost reservation with actual cost after spawn completes.
func spawnAndWaitWithBudget(ctx context.Context, tr *TeamRunner, waveIdx, memIdx int, estimated float64) {
	// Get member info for actual cost extraction
	tr.configMu.RLock()
	if tr.config == nil || waveIdx >= len(tr.config.Waves) || memIdx >= len(tr.config.Waves[waveIdx].Members) {
		tr.configMu.RUnlock()
		log.Printf("ERROR: Invalid wave/member indices in budget reconciliation: wave=%d, member=%d", waveIdx, memIdx)
		return
	}
	memberName := tr.config.Waves[waveIdx].Members[memIdx].Name
	tr.configMu.RUnlock()

	// Spawn and wait (this calls the WaitGroup.Done internally)
	var wg sync.WaitGroup
	wg.Add(1)
	spawnAndWait(ctx, tr, waveIdx, memIdx, &wg)
	wg.Wait()

	// Extract actual cost from member after completion
	tr.configMu.RLock()
	actual := tr.config.Waves[waveIdx].Members[memIdx].CostUSD
	tr.configMu.RUnlock()

	// Reconcile budget: return reservation, deduct actual
	if err := tr.reconcileCost(estimated, actual); err != nil {
		log.Printf("[ERROR] wave: budget reconciliation failed for %s: %v", memberName, err)
	} else {
		log.Printf("[INFO] wave: reconciled budget for %s (estimated=$%.2f, actual=$%.2f, remaining=$%.2f)",
			memberName, estimated, actual, tr.BudgetRemaining())
	}
}

// runInterWaveScript executes a script between waves.
// Stub implementation for TC-010 — logs and returns nil.
func runInterWaveScript(ctx context.Context, scriptPath string, teamDir string) error {
	log.Printf("[INFO] wave: inter-wave script %s (TC-010 stub, not yet implemented)", scriptPath)
	// TC-010 will implement real execution here
	return nil
}
