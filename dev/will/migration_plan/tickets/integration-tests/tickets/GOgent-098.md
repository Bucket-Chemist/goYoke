---
id: GOgent-098
title: Performance Benchmarks
description: **Task**:
status: pending
time_estimate: 2h
dependencies: ["GOgent-094","GOgent-095"]
priority: high
week: 5
tags: ["performance", "week-5"]
tests_required: true
acceptance_criteria_count: 8
---

### GOgent-098: Performance Benchmarks

**Time**: 2 hours
**Dependencies**: GOgent-094 (harness), GOgent-095-044 (hook binaries)

**Task**:
Benchmark hook execution latency and memory usage. Target: <5ms p99 latency, <10MB memory per hook.

**File**: `test/benchmark/hooks_bench_test.go`

**Implementation**:

```go
package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// BenchmarkValidateRouting_Allow benchmarks validate-routing for allowed operations
func BenchmarkValidateRouting_Allow(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Read",
		"tool_input": {"file_path": "/tmp/test.txt"},
		"session_id": "bench-allow"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkValidateRouting_Block benchmarks validate-routing for blocked operations
func BenchmarkValidateRouting_Block(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Task",
		"tool_input": {
			"model": "opus",
			"prompt": "AGENT: einstein\n\nAnalyze",
			"subagent_type": "general-purpose"
		},
		"session_id": "bench-block"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkSessionArchive benchmarks session-archive hook
func BenchmarkSessionArchive(b *testing.B) {
	binaryPath := "../../cmd/gogent-archive/gogent-archive"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-archive binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupSessionMetricsFiles(b, projectDir)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-session",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript.jsonl"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}

		// Clean up handoff for next iteration
		handoffPath := filepath.Join(projectDir, ".claude", "memory", "last-handoff.md")
		os.Remove(handoffPath)
	}
}

// BenchmarkSharpEdgeDetector benchmarks sharp-edge-detector hook
func BenchmarkSharpEdgeDetector(b *testing.B) {
	binaryPath := "../../cmd/gogent-sharp-edge/gogent-sharp-edge"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-sharp-edge binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PostToolUse",
		"tool_name": "Edit",
		"tool_input": {"file_path": "/tmp/test.go"},
		"tool_response": {"success": false, "error": "Type error"},
		"session_id": "bench-sharp-edge"
	}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}
	}
}

// BenchmarkMemoryUsage measures peak memory usage of hooks
func BenchmarkMemoryUsage(b *testing.B) {
	hooks := []struct {
		name    string
		path    string
		event   string
	}{
		{
			name: "validate-routing",
			path: "../../cmd/gogent-validate/gogent-validate",
			event: `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"}}`,
		},
		{
			name: "session-archive",
			path: "../../cmd/gogent-archive/gogent-archive",
			event: `{"hook_event_name":"SessionEnd","session_id":"mem-test"}`,
		},
		{
			name: "sharp-edge-detector",
			path: "../../cmd/gogent-sharp-edge/gogent-sharp-edge",
			event: `{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_response":{"success":false}}`,
		},
	}

	projectDir := setupBenchmarkProject(b)

	for _, hook := range hooks {
		b.Run(hook.name, func(b *testing.B) {
			if _, err := os.Stat(hook.path); err != nil {
				b.Skipf("%s binary not found", hook.name)
			}

			var totalMem uint64

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)

				cmd := exec.Command(hook.path)
				cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
				cmd.Stdin = bytes.NewReader([]byte(hook.event))

				var stdout bytes.Buffer
				cmd.Stdout = &stdout

				if err := cmd.Run(); err != nil {
					// Some hooks may error on minimal input - that's OK for memory test
				}

				runtime.ReadMemStats(&m2)
				totalMem += (m2.TotalAlloc - m1.TotalAlloc)
			}

			avgMem := totalMem / uint64(b.N)
			b.ReportMetric(float64(avgMem)/1024/1024, "MB/op")

			// Verify <10MB target
			if avgMem > 10*1024*1024 {
				b.Errorf("%s exceeds 10MB memory target: %.2f MB", hook.name, float64(avgMem)/1024/1024)
			}
		})
	}
}

// BenchmarkLatency_Percentiles measures p50, p95, p99 latencies
func BenchmarkLatency_Percentiles(b *testing.B) {
	binaryPath := "../../cmd/gogent-validate/gogent-validate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-validate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := `{
		"hook_event_name": "PreToolUse",
		"tool_name": "Read",
		"tool_input": {"file_path": "/tmp/test.txt"}
	}`

	// Run 1000 iterations to get percentile data
	iterations := 1000
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Fatalf("Hook failed: %v", err)
		}

		latencies[i] = time.Since(start)
	}

	// Calculate percentiles
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	b.ReportMetric(float64(p50.Microseconds()), "p50-μs")
	b.ReportMetric(float64(p95.Microseconds()), "p95-μs")
	b.ReportMetric(float64(p99.Microseconds()), "p99-μs")

	// Verify <5ms p99 target
	if p99 > 5*time.Millisecond {
		b.Errorf("p99 latency exceeds 5ms target: %v", p99)
	}

	fmt.Printf("\nLatency Percentiles:\n")
	fmt.Printf("  p50: %v\n", p50)
	fmt.Printf("  p95: %v\n", p95)
	fmt.Printf("  p99: %v\n", p99)
}

// Helper: Setup benchmark project directory
func setupBenchmarkProject(b *testing.B) string {
	projectDir := b.TempDir()

	// Create routing schema
	schemaPath := filepath.Join(projectDir, ".claude", "routing-schema.json")
	os.MkdirAll(filepath.Dir(schemaPath), 0755)

	schema := `{
		"tiers": {
			"haiku": {"tools_allowed": ["Read", "Glob", "Grep"]},
			"sonnet": {"tools_allowed": ["Read", "Glob", "Grep", "Edit", "Write", "Bash", "Task"]},
			"opus": {"tools_allowed": ["*"], "task_invocation_blocked": true}
		},
		"agent_subagent_mapping": {
			"codebase-search": "Explore",
			"einstein": "general-purpose"
		}
	}`

	os.WriteFile(schemaPath, []byte(schema), 0644)

	// Set tier to haiku
	tierPath := filepath.Join(projectDir, ".gogent", "current-tier")
	os.MkdirAll(filepath.Dir(tierPath), 0755)
	os.WriteFile(tierPath, []byte("haiku\n"), 0644)

	return projectDir
}

// Helper: Setup session metrics files
func setupSessionMetricsFiles(b *testing.B, projectDir string) {
	// Create tool counter logs
	toolCounterPath := filepath.Join(projectDir, ".gogent", "tool-counter-read")
	os.MkdirAll(filepath.Dir(toolCounterPath), 0755)
	os.WriteFile(toolCounterPath, []byte("x\nx\nx\n"), 0644)

	// Create empty transcript
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	os.WriteFile(transcriptPath, []byte(""), 0644)
}

// Helper: Calculate percentile from sorted durations
func percentile(durations []time.Duration, p int) time.Duration {
	// Sort durations
	for i := 0; i < len(durations); i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[j] < durations[i] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	index := (p * len(durations)) / 100
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}
```

**Run benchmarks**:
```bash
go test -bench=. ./test/benchmark -benchmem -benchtime=10s
```

**Acceptance Criteria**:
- [ ] `BenchmarkValidateRouting_Allow` measures allow path latency
- [ ] `BenchmarkValidateRouting_Block` measures block path latency
- [ ] `BenchmarkSessionArchive` measures session-archive latency
- [ ] `BenchmarkSharpEdgeDetector` measures sharp-edge latency
- [ ] `BenchmarkMemoryUsage` verifies <10MB memory per hook
- [ ] `BenchmarkLatency_Percentiles` verifies <5ms p99 latency
- [ ] All benchmarks pass performance targets
- [ ] Benchmark report saved: `go test -bench=. ./test/benchmark | tee benchmark-results.txt`

**Why This Matters**: Performance regression would make hooks unusable in production. Must verify latency and memory targets met before cutover.

---
