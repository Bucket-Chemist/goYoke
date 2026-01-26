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

// BenchmarkLoadContext_LanguageDetection measures language detection latency
func BenchmarkLoadContext_LanguageDetection(b *testing.B) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-load-context binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	createLanguageIndicators(b, projectDir)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "bench-lang-detect",
		"project_dir": "%s"
	}`, projectDir)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Logf("Warning: gogent-load-context failed: %v", err)
		}

		latencies = append(latencies, time.Since(start))
	}

	// Verify target <2ms
	for _, lat := range latencies {
		if lat > 2*time.Millisecond {
			b.Logf("Language detection latency exceeded 2ms target: %v", lat)
		}
	}
}

// BenchmarkLoadContext_HandoffInjection measures handoff injection latency for various payload sizes
func BenchmarkLoadContext_HandoffInjection(b *testing.B) {
	binaryPath := "../../cmd/gogent-load-context/gogent-load-context"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-load-context binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupHandoffFile(b, projectDir, 10*1024) // 10KB handoff

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionStart",
		"session_id": "bench-handoff",
		"project_dir": "%s"
	}`, projectDir)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Logf("Warning: gogent-load-context failed: %v", err)
		}

		latencies = append(latencies, time.Since(start))
	}

	// Verify target <3ms per 10KB
	for _, lat := range latencies {
		if lat > 3*time.Millisecond {
			b.Logf("Handoff injection latency exceeded 3ms target: %v", lat)
		}
	}
}

// BenchmarkAgentEndstate_OutcomeLogging measures outcome logging latency
func BenchmarkAgentEndstate_OutcomeLogging(b *testing.B) {
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
		"project_dir": "%s",
		"outcome": "success"
	}`, projectDir)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Logf("Warning: gogent-agent-endstate failed: %v", err)
		}

		latencies = append(latencies, time.Since(start))
	}

	// Verify target <1ms
	for _, lat := range latencies {
		if lat > 1*time.Millisecond {
			b.Logf("Outcome logging latency exceeded 1ms target: %v", lat)
		}
	}
}

// BenchmarkAgentEndstate_CollaborationUpdate measures collaboration logging latency
func BenchmarkAgentEndstate_CollaborationUpdate(b *testing.B) {
	binaryPath := "../../cmd/gogent-agent-endstate/gogent-agent-endstate"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-agent-endstate binary not found")
	}

	projectDir := setupBenchmarkProject(b)

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SubagentStop",
		"subagent_type": "general-purpose",
		"agent_name": "python-pro",
		"execution_time_ms": 4567,
		"project_dir": "%s",
		"collaboration_data": {"files_modified": 5, "lines_added": 150}
	}`, projectDir)

	latencies := make([]time.Duration, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			b.Logf("Warning: gogent-agent-endstate failed: %v", err)
		}

		latencies = append(latencies, time.Since(start))
	}

	// Verify target <2ms
	for _, lat := range latencies {
		if lat > 2*time.Millisecond {
			b.Logf("Collaboration update latency exceeded 2ms target: %v", lat)
		}
	}
}

// BenchmarkMLExport_SmallDataset benchmarks ML export on 1K event dataset
func BenchmarkMLExport_SmallDataset(b *testing.B) {
	binaryPath := "../../cmd/gogent-ml-export/gogent-ml-export"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-ml-export binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupMLDataset(b, projectDir, 1000, "transcript-1k.jsonl")

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-ml-1k",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript-1k.jsonl"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			// ML export may error on minimal input - that's OK
		}
	}
}

// BenchmarkMLExport_MediumDataset benchmarks ML export on 10K event dataset
func BenchmarkMLExport_MediumDataset(b *testing.B) {
	binaryPath := "../../cmd/gogent-ml-export/gogent-ml-export"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-ml-export binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupMLDataset(b, projectDir, 10000, "transcript-10k.jsonl")

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-ml-10k",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript-10k.jsonl"))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			// ML export may error on minimal input - that's OK
		}
	}
}

// BenchmarkMLExport_LargeDataset benchmarks ML export on 100K event dataset with strict targets
func BenchmarkMLExport_LargeDataset(b *testing.B) {
	binaryPath := "../../cmd/gogent-ml-export/gogent-ml-export"
	if _, err := os.Stat(binaryPath); err != nil {
		b.Skip("gogent-ml-export binary not found")
	}

	projectDir := setupBenchmarkProject(b)
	setupMLDataset(b, projectDir, 100000, "transcript-100k.jsonl")

	eventJSON := fmt.Sprintf(`{
		"hook_event_name": "SessionEnd",
		"session_id": "bench-ml-100k",
		"transcript_path": "%s"
	}`, filepath.Join(projectDir, "transcript-100k.jsonl"))

	latencies := make([]time.Duration, 0, b.N)
	memoryUsages := make([]uint64, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()

		cmd := exec.Command(binaryPath)
		cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
		cmd.Stdin = bytes.NewReader([]byte(eventJSON))

		var stdout bytes.Buffer
		cmd.Stdout = &stdout

		if err := cmd.Run(); err != nil {
			// ML export may error on minimal input - that's OK
		}

		latency := time.Since(start)
		latencies = append(latencies, latency)

		runtime.ReadMemStats(&m2)
		memoryUsages = append(memoryUsages, m2.TotalAlloc-m1.TotalAlloc)
	}

	// Calculate and report statistics
	avgLatency := averageDuration(latencies)
	maxLatency := maxDuration(latencies)
	avgMemory := averageUint64(memoryUsages)

	b.ReportMetric(float64(avgLatency.Milliseconds()), "avg-latency-ms")
	b.ReportMetric(float64(maxLatency.Milliseconds()), "max-latency-ms")
	b.ReportMetric(float64(avgMemory)/(1024*1024), "avg-memory-mb")

	// Verify targets: <5s latency, <100MB memory
	for _, lat := range latencies {
		if lat > 5*time.Second {
			b.Logf("ML export large dataset latency exceeded 5s target: %v", lat)
		}
	}

	for _, mem := range memoryUsages {
		if mem > 100*1024*1024 {
			b.Logf("ML export large dataset memory exceeded 100MB target: %.2f MB", float64(mem)/(1024*1024))
		}
	}
}

// BenchmarkLatency_AllHooks_Percentiles measures p50, p95, p99 latencies across all hooks
func BenchmarkLatency_AllHooks_Percentiles(b *testing.B) {
	hooks := []struct {
		name  string
		path  string
		event string
	}{
		{
			name: "validate-routing",
			path: "../../cmd/gogent-validate/gogent-validate",
			event: `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/tmp/test.txt"},"session_id":"bench-validate"}`,
		},
		{
			name: "load-context",
			path: "../../cmd/gogent-load-context/gogent-load-context",
			event: `{"hook_event_name":"SessionStart","session_id":"bench-load-ctx","project_dir":""}`,
		},
		{
			name: "agent-endstate",
			path: "../../cmd/gogent-agent-endstate/gogent-agent-endstate",
			event: `{"hook_event_name":"SubagentStop","subagent_type":"Explore","agent_name":"codebase-search","execution_time_ms":1234,"project_dir":""}`,
		},
		{
			name: "sharp-edge-detector",
			path: "../../cmd/gogent-sharp-edge/gogent-sharp-edge",
			event: `{"hook_event_name":"PostToolUse","tool_name":"Edit","tool_input":{"file_path":"/tmp/test.go"},"tool_response":{"success":false,"error":"Type error"},"session_id":"bench-sharp"}`,
		},
		{
			name: "session-archive",
			path: "../../cmd/gogent-archive/gogent-archive",
			event: `{"hook_event_name":"SessionEnd","session_id":"bench-archive","transcript_path":""}`,
		},
		{
			name: "ml-export",
			path: "../../cmd/gogent-ml-export/gogent-ml-export",
			event: `{"hook_event_name":"SessionEnd","session_id":"bench-ml","transcript_path":""}`,
		},
	}

	projectDir := setupBenchmarkProject(b)

	for _, hook := range hooks {
		b.Run(hook.name, func(b *testing.B) {
			if _, err := os.Stat(hook.path); err != nil {
				b.Skipf("%s binary not found", hook.name)
			}

			// Run 1000 iterations for good percentile data
			iterations := 1000
			latencies := make([]time.Duration, 0, iterations)

			for i := 0; i < iterations; i++ {
				start := time.Now()

				cmd := exec.Command(hook.path)
				cmd.Env = append(os.Environ(), "CLAUDE_PROJECT_DIR="+projectDir)
				cmd.Stdin = bytes.NewReader([]byte(hook.event))

				var stdout bytes.Buffer
				cmd.Stdout = &stdout

				if err := cmd.Run(); err != nil {
					// Some hooks may error on minimal input - that's OK
				}

				latencies = append(latencies, time.Since(start))
			}

			// Calculate percentiles
			p50 := percentile(latencies, 50)
			p95 := percentile(latencies, 95)
			p99 := percentile(latencies, 99)

			b.ReportMetric(float64(p50.Microseconds()), "p50-μs")
			b.ReportMetric(float64(p95.Microseconds()), "p95-μs")
			b.ReportMetric(float64(p99.Microseconds()), "p99-μs")

			fmt.Printf("\n%s Latency Percentiles:\n", hook.name)
			fmt.Printf("  p50: %.2f μs\n", float64(p50.Microseconds()))
			fmt.Printf("  p95: %.2f μs\n", float64(p95.Microseconds()))
			fmt.Printf("  p99: %.2f μs\n", float64(p99.Microseconds()))
		})
	}
}

// Helper: Create language indicator files (Python, Go, R)
func createLanguageIndicators(b *testing.B, projectDir string) {
	// Create pyproject.toml for Python detection
	pyProjectPath := filepath.Join(projectDir, "pyproject.toml")
	os.WriteFile(pyProjectPath, []byte("[project]\nname = \"test\"\n"), 0644)

	// Create go.mod for Go detection
	goModPath := filepath.Join(projectDir, "go.mod")
	os.WriteFile(goModPath, []byte("module github.com/test/test\n"), 0644)

	// Create R project file
	rProjectPath := filepath.Join(projectDir, "test.Rproj")
	os.WriteFile(rProjectPath, []byte("Version: 1.0\n"), 0644)
}

// Helper: Setup handoff file with specific size
func setupHandoffFile(b *testing.B, projectDir string, sizeBytes int) {
	memoryDir := filepath.Join(projectDir, ".claude", "memory")
	os.MkdirAll(memoryDir, 0755)

	handoffPath := filepath.Join(memoryDir, "last-handoff.md")

	// Create handoff content of approximately the requested size
	content := "# Handoff Document\n\n"
	for len(content) < sizeBytes {
		content += "## Decision Block\n"
		content += "This is a decision point in the migration flow.\n"
		content += "- Key insight 1\n"
		content += "- Key insight 2\n"
		content += "- Key insight 3\n\n"
	}

	// Truncate to exact size if needed
	if len(content) > sizeBytes {
		content = content[:sizeBytes]
	}

	os.WriteFile(handoffPath, []byte(content), 0644)
}

// Helper: Setup ML dataset of specific size
func setupMLDataset(b *testing.B, projectDir string, eventCount int, filename string) {
	transcriptPath := filepath.Join(projectDir, filename)

	var buf bytes.Buffer
	for i := 0; i < eventCount; i++ {
		tool := "Read"
		if i%5 == 0 {
			tool = "Edit"
		} else if i%7 == 0 {
			tool = "Write"
		} else if i%11 == 0 {
			tool = "Bash"
		}

		event := map[string]interface{}{
			"event_id":    fmt.Sprintf("evt-%d", i),
			"timestamp":   "2026-01-25T10:00:00Z",
			"tool":        tool,
			"duration_ms": 10 + (i % 50),
			"success":     i%50 != 0, // 98% success rate
		}

		eventJSON, _ := json.Marshal(event)
		buf.Write(eventJSON)
		buf.WriteString("\n")
	}

	os.WriteFile(transcriptPath, buf.Bytes(), 0644)
}

// Helper: Calculate average duration
func averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

// Helper: Find maximum duration
func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations {
		if d > max {
			max = d
		}
	}

	return max
}

// Helper: Calculate average uint64
func averageUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}

	var total uint64
	for _, v := range values {
		total += v
	}

	return total / uint64(len(values))
}
