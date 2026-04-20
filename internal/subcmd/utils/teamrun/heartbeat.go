package teamrun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// HeartbeatInterval defines how often the heartbeat file is touched.
// C6: Per TC-012 requirement (was 30s in original spec, now 10s).
const HeartbeatInterval = 10 * time.Second

// startHeartbeat begins a background goroutine that touches the heartbeat file
// at HeartbeatInterval. Stops when ctx is cancelled.
//
// The heartbeat file is written immediately on start, then periodically updated.
// This allows monitoring tools to detect daemon health.
//
// File location: {teamDir}/heartbeat
// File content: Unix timestamp (seconds since epoch)
func startHeartbeat(ctx context.Context, teamDir string) {
	startHeartbeatWithInterval(ctx, teamDir, HeartbeatInterval)
}

// startHeartbeatWithInterval is the testable version of startHeartbeat with
// configurable interval. Used by tests to run heartbeat with short intervals
// (e.g., 50ms) for fast verification.
//
// Production code should use startHeartbeat() which uses the HeartbeatInterval constant.
func startHeartbeatWithInterval(ctx context.Context, teamDir string, interval time.Duration) {
	heartbeatPath := filepath.Join(teamDir, "heartbeat")

	// Touch immediately on start
	writeHeartbeat(heartbeatPath)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				writeHeartbeat(heartbeatPath)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// writeHeartbeat writes the current Unix timestamp to the heartbeat file.
// Errors are logged but do not stop the heartbeat goroutine.
func writeHeartbeat(path string) {
	now := time.Now().Unix()
	content := fmt.Sprintf("%d\n", now)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		// Log error but don't stop heartbeat
		// This is a monitoring feature, not critical to execution
		fmt.Fprintf(os.Stderr, "heartbeat: failed to write %s: %v\n", path, err)
	}
}
