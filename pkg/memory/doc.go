// Package memory provides session memory and failure tracking for GOgent-Fortress.
//
// The failure tracking subsystem tracks consecutive failures per file+error_type
// composite key to detect debugging loops. When a threshold is reached (default: 3),
// a sharp edge is captured for later analysis.
//
// # Environment Variables
//
//   - GOGENT_MAX_FAILURES: Threshold for sharp edge capture (default: 3)
//   - GOGENT_FAILURE_WINDOW: Time window in seconds (default: 300)
//
// # Storage
//
// Tracker state is stored at ~/.gogent/failure-tracker.jsonl
//
// # Usage
//
//	tracker := memory.DefaultFailureTracker()
//	key := memory.FailureKey{
//	    FilePath:  "path/to/file.go",
//	    ErrorType: "typeerror",
//	}
//	tracker.LogFailure(key, "Bash")
//	count, _ := tracker.CountRecentFailures(key)
package memory
