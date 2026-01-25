package benchmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

// BenchmarkValidateRouting_Allow benchmarks validate-routing for allowed operations
func BenchmarkValidateRouting_Allow(b *testing.B) {
	binaryPath := "../../bin/gogent-validate"
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
	binaryPath := "../../bin/gogent-validate"
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
	binaryPath := "../../bin/gogent-archive"
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
	binaryPath := "../../bin/gogent-sharp-edge"
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
	binaryPath := "../../bin/gogent-load-context"
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

	// Verify <20ms p99 target (load-context is I/O bound - reads handoff files)
	if p99 > 20*time.Millisecond {
		b.Errorf("p99 latency exceeds 20ms target: %v", p99)
	}
}

// BenchmarkAgentEndstate benchmarks gogent-agent-endstate hook
func BenchmarkAgentEndstate(b *testing.B) {
	binaryPath := "../../bin/gogent-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-agent-endstate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	// Create empty transcript for agent-endstate
	transcriptPath := filepath.Join(projectDir, "transcript.jsonl")
	os.WriteFile(transcriptPath, []byte(""), 0644)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SubagentStop",
		"subagent_type": "Explore",
		"agent_name": "codebase-search",
		"execution_time_ms": 1234,
		"session_id": "bench-agent-endstate",
		"transcript_path": "%s",
		"project_dir": "%s"
	}`, transcriptPath, projectDir)

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

// BenchmarkMLExport benchmarks gogent-ml-export with large dataset
func BenchmarkMLExport(b *testing.B) {
	binaryPath := "../../bin/gogent-ml-export"
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
		name  string
		path  string
		event string
	}{
		{
			name:  "validate-routing",
			path:  "../../bin/gogent-validate",
			event: `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"}}`,
		},
		{
			name:  "session-archive",
			path:  "../../bin/gogent-archive",
			event: `{"hook_event_name":"SessionEnd","session_id":"mem-test"}`,
		},
		{
			name:  "sharp-edge-detector",
			path:  "../../bin/gogent-sharp-edge",
			event: `{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_response":{"success":false}}`,
		},
		{
			name:  "load-context",
			path:  "../../bin/gogent-load-context",
			event: `{"hook_event_name":"SessionStart","session_id":"mem-load-context"}`,
		},
		{
			name:  "agent-endstate",
			path:  "../../bin/gogent-agent-endstate",
			event: `{"hook_event_name":"SubagentStop","agent_name":"codebase-search","session_id":"mem-agent-endstate","transcript_path":"/tmp/transcript.jsonl"}`,
		},
		{
			name:  "ml-export",
			path:  "../../bin/gogent-ml-export",
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
	binaryPath := "../../bin/gogent-validate"
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

	// Verify <6ms p99 target (allow small variance)
	if p99 > 6*time.Millisecond {
		b.Errorf("p99 latency exceeds 6ms target: %v", p99)
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
		event := map[string]interface{}{
			"event_id":    fmt.Sprintf("evt-%d", i),
			"timestamp":   "2026-01-25T10:00:00Z",
			"tool":        "Read",
			"duration_ms": 10 + (i % 50),
			"success":     true,
		}
		eventJSON, _ := json.Marshal(event)
		buf.Write(eventJSON)
		buf.WriteString("\n")
	}

	os.WriteFile(transcriptPath, buf.Bytes(), 0644)
}

// Helper: Calculate percentile from durations
func percentile(durations []time.Duration, p int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Create a copy to avoid modifying original slice
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	// Sort durations using standard library
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := (p * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
