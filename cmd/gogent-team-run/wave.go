package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// runWaves executes all waves sequentially.
// Within each wave, members execute in parallel via spawnAndWait goroutines.
// Budget gates prevent spawns when budget is insufficient.
func runWaves(ctx context.Context, tr *TeamRunner) error {
	// Snapshot wave count under lock — config.Waves length is immutable after load
	tr.configMu.RLock()
	if tr.config == nil {
		tr.configMu.RUnlock()
		return fmt.Errorf("config not loaded")
	}
	waves := tr.config.Waves // snapshot slice header (safe: waves are never appended after load)
	tr.configMu.RUnlock()

	for waveIdx, wave := range waves {
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

		// Check for wave failures and skip subsequent waves if any failed
		if failed := checkWaveFailures(tr, waveIdx); len(failed) > 0 {
			log.Printf("[WARN] wave: Wave %d had failed members: %v", wave.WaveNumber, failed)
			skipRemainingWaves(tr, waveIdx)
			return fmt.Errorf("wave %d had %d failed member(s), skipping subsequent waves", wave.WaveNumber, len(failed))
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
// Passes teamDir as the first argument to the script.
func runInterWaveScript(ctx context.Context, scriptPath string, teamDir string) error {
	log.Printf("[INFO] wave: executing inter-wave script: %s %s", scriptPath, teamDir)
	cmd := exec.CommandContext(ctx, scriptPath, teamDir)
	cmd.Dir = teamDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("inter-wave script %s failed: %w\noutput: %s", scriptPath, err, string(output))
	}
	log.Printf("[INFO] wave: inter-wave script %s completed successfully", scriptPath)
	return nil
}

// checkWaveFailures returns names of failed members in the given wave.
// Reads member statuses under RLock.
func checkWaveFailures(tr *TeamRunner, waveIdx int) []string {
	tr.configMu.RLock()
	defer tr.configMu.RUnlock()

	if tr.config == nil || waveIdx >= len(tr.config.Waves) {
		return nil
	}

	var failed []string
	for _, member := range tr.config.Waves[waveIdx].Members {
		if member.Status == "failed" {
			failed = append(failed, member.Name)
		}
	}
	return failed
}

// skipRemainingWaves sets all members in waves after fromWaveIdx to "skipped".
// Uses updateMember for thread-safe status updates.
func skipRemainingWaves(tr *TeamRunner, fromWaveIdx int) {
	tr.configMu.RLock()
	if tr.config == nil || fromWaveIdx+1 >= len(tr.config.Waves) {
		tr.configMu.RUnlock()
		return
	}
	totalWaves := len(tr.config.Waves)
	tr.configMu.RUnlock()

	// Skip all waves after fromWaveIdx
	for waveIdx := fromWaveIdx + 1; waveIdx < totalWaves; waveIdx++ {
		tr.configMu.RLock()
		memberCount := len(tr.config.Waves[waveIdx].Members)
		tr.configMu.RUnlock()

		for memIdx := 0; memIdx < memberCount; memIdx++ {
			// updateMember acquires writeMu then configMu - do NOT call while holding either lock
			if err := tr.updateMember(waveIdx, memIdx, func(m *Member) {
				m.Status = "skipped"
			}); err != nil {
				log.Printf("[ERROR] wave: failed to skip member at wave %d, member %d: %v", waveIdx, memIdx, err)
			}
		}
	}
}
