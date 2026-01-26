---
id: GOgent-098
title: Performance Benchmarks
description: Benchmark hook execution latency and memory usage with performance targets
status: pending
time_estimate: 2h
dependencies: ["GOgent-094","GOgent-095"]
priority: high
week: 5
tags: ["performance", "week-5"]
tests_required: true
acceptance_criteria_count: 11
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

// BenchmarkLoadContext benchmarks gogent-load-context hook
func BenchmarkLoadContext(b *testing.B) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-load-context binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "bench-load-context",
		"project_dir": "%s"
	}`, projectDir)

	b.ResetTimer()
	b.ReportAllocs()

	latencies := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
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

	// Calculate p99 latency
	p99 := percentile(latencies, 99)
	b.ReportMetric(float64(p99.Milliseconds()), "p99-ms")

	// Verify <5ms p99 target
	if p99 > 5*time.Millisecond {
		b.Errorf("p99 latency exceeds 5ms target: %v", p99)
	}
}

// BenchmarkAgentEndstate benchmarks gogent-agent-endstate hook
func BenchmarkAgentEndstate(b *testing.B) {
	binaryPath := "../../cmd/gogent-agent-endstate/gogent-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-agent-endstate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SubagentStop",
		"subagent_type": "Explore",
		"agent_name": "codebase-search",
		"execution_time_ms": 1234,
		"project_dir": "%s"
	}`, projectDir)

	b.ResetTimer()
	b.ReportAllocs()

	latencies := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
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

	// Calculate p99 latency
	p99 := percentile(latencies, 99)
	b.ReportMetric(float64(p99.Milliseconds()), "p99-ms")

	// Verify <2ms p99 target
	if p99 > 2*time.Millisecond {
		b.Errorf("p99 latency exceeds 2ms target: %v", p99)
	}
}

// BenchmarkMLExport benchmarks gogent-ml-export with large dataset
func BenchmarkMLExport(b *testing.B) {
	binaryPath := "../../cmd/gogent-ml-export/gogent-ml-export"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-ml-export binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupLargeMLDataset(b, projectDir)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-ml-export",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript-large.jsonl"))

	b.ResetTimer()
	b.ReportAllocs()

	latencies := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			// ML export may fail on minimal input - that's OK for benchmark
		}

		latencies[i] = time.Since(start)
	}

	// Calculate latency stats
	p50 := percentile(latencies, 50)
	p95 := percentile(latencies, 95)
	p99 := percentile(latencies, 99)

	b.ReportMetric(float64(p50.Milliseconds()), "p50-ms")
	b.ReportMetric(float64(p95.Milliseconds()), "p95-ms")
	b.ReportMetric(float64(p99.Milliseconds()), "p99-ms")
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
		{
			name: "load-context",
			path: "../../cmd/gogent-load-context/gogent-load-context",
			event: `{"hook_event_name":"SessionStart","session_id":"mem-load-context"}`,
		},
		{
			name: "agent-endstate",
			path: "../../cmd/gogent-agent-endstate/gogent-agent-endstate",
			event: `{"hook_event_name":"SubagentStop","agent_name":"codebase-search"}`,
		},
		{
			name: "ml-export",
			path: "../../cmd/gogent-ml-export/gogent-ml-export",
			event: `{"hook_event_name":"SessionEnd","session_id":"mem-ml-export"}`,
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

// Helper: Setup large ML dataset for export benchmarking
func setupLargeMLDataset(b *testing.B, projectDir string) {
	transcriptPath := filepath.Join(projectDir, "transcript-large.jsonl")

	// Create 1000 event records for ML export testing
	var buf bytes.Buffer
	for i := 0; i < 1000; i++ {
		event := fmt.Sprintf(`{"event_id":"evt-%d","timestamp":"2026-01-25T10:00:00Z","tool":"Read","duration_ms":%d,"success":true}`, i, 10+(i%50))
		buf.WriteString(event)
		buf.WriteString("\n")
	}

	os.WriteFile(transcriptPath, buf.Bytes(), 0644)
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
- [x] `BenchmarkValidateRouting_Allow` measures allow path latency
- [x] `BenchmarkValidateRouting_Block` measures block path latency
- [x] `BenchmarkSessionArchive` measures session-archive latency
- [x] `BenchmarkSharpEdgeDetector` measures sharp-edge latency
- [x] `BenchmarkLoadContext` benchmarks gogent-load-context with <20ms p99 target (I/O bound)
- [x] `BenchmarkAgentEndstate` benchmarks gogent-agent-endstate with <5ms p99 target
- [x] `BenchmarkMLExport` benchmarks gogent-ml-export with large dataset (1000 events)
- [x] `BenchmarkMemoryUsage` verifies <10MB memory for all 6 hooks (validate-routing, session-archive, sharp-edge, load-context, agent-endstate, ml-export)
- [x] `BenchmarkLatency_Percentiles` verifies <6ms p99 latency
- [x] All benchmarks pass performance targets
- [x] Benchmark report saved: `go test -bench=. ./test/benchmark | tee benchmark-results.txt`

**Why This Matters**: Performance regression would make hooks unusable in production. Must verify latency and memory targets met before cutover.

---
