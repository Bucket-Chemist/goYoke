package cli

// BenchmarkNDJSONParsing measures the throughput of ParseCLIEvent on a
// representative mix of realistic NDJSON lines.
//
// Target threshold (TUI-040): sustain >= 10,000 lines/sec.
// In practice, JSON unmarshal for small structs is in the µs range;
// 10K lines/sec = 100 µs/line is a very conservative lower bound.
//
// Run:
//
//	go test -bench=BenchmarkNDJSON -benchmem -count=5 ./internal/tui/cli/

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// NDJSON fixtures
//
// The fixture set covers every event type exercised by ParseCLIEvent:
//   - system/init
//   - system/hook_started
//   - system/hook_response
//   - assistant (text + tool_use)
//   - user (tool_result)
//   - result/success
//   - rate_limit_event
//   - stream_event (content_block_delta)
//   - unknown type (fallback path)
// ---------------------------------------------------------------------------

var benchNDJSONLines = [][]byte{
	// system/init
	[]byte(`{"type":"system","subtype":"init","cwd":"/home/user/project","session_id":"sess-abc123","tools":["Read","Write","Edit","Bash","Glob","Grep","Task"],"model":"claude-sonnet-4-20250514","permissionMode":"acceptEdits","apiKeySource":"env","claude_code_version":"1.2.3","uuid":"uuid-init-001"}`),

	// system/hook_started
	[]byte(`{"type":"system","subtype":"hook_started","hook_id":"hook-001","hook_name":"gogent-validate","hook_event":"PreToolUse","uuid":"uuid-hook-001","session_id":"sess-abc123"}`),

	// system/hook_response
	[]byte(`{"type":"system","subtype":"hook_response","hook_id":"hook-001","hook_name":"gogent-validate","hook_event":"PreToolUse","output":"","stdout":"","stderr":"","exit_code":0,"outcome":"allow","uuid":"uuid-hook-002","session_id":"sess-abc123"}`),

	// assistant — text block only
	[]byte(`{"type":"assistant","message":{"id":"msg-001","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"I will read the file and analyze its contents."}],"stop_reason":null,"usage":{"input_tokens":1024,"output_tokens":128}},"parent_tool_use_id":null,"session_id":"sess-abc123","uuid":"uuid-asst-001"}`),

	// assistant — tool_use block (Task call)
	[]byte(`{"type":"assistant","message":{"id":"msg-002","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"tool_use","id":"tu-001","name":"Read","input":{"file_path":"/home/user/project/main.go"},"caller":{"type":"direct"}}],"stop_reason":"tool_use","usage":{"input_tokens":2048,"output_tokens":64}},"parent_tool_use_id":null,"session_id":"sess-abc123","uuid":"uuid-asst-002"}`),

	// assistant — tool_use block (Task spawning a subagent)
	[]byte(`{"type":"assistant","message":{"id":"msg-003","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"tool_use","id":"tu-002","name":"Task","input":{"description":"Analyze the codebase","subagent_type":"GO Pro","model":"claude-sonnet-4-20250514","prompt":"Review the Go code in internal/"},"caller":{"type":"direct"}}],"stop_reason":"tool_use","usage":{"input_tokens":3000,"output_tokens":200}},"parent_tool_use_id":null,"session_id":"sess-abc123","uuid":"uuid-asst-003"}`),

	// user — tool_result
	[]byte(`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu-001","content":"package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n","is_error":false}]},"parent_tool_use_id":null,"session_id":"sess-abc123","uuid":"uuid-user-001"}`),

	// rate_limit_event
	[]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","rateLimitType":"output_tokens","overageStatus":"","resetsAt":1700000000,"isUsingOverage":false},"uuid":"uuid-rl-001","session_id":"sess-abc123"}`),

	// result/success
	[]byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":12345,"duration_api_ms":10000,"num_turns":5,"result":"Completed successfully","stop_reason":"end_turn","session_id":"sess-abc123","uuid":"uuid-result-001","total_cost_usd":0.0123,"usage":{"input_tokens":5000,"output_tokens":800}}`),

	// stream_event — content_block_delta
	[]byte(`{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"erat. "}},"session_id":"sess-abc123","parent_tool_use_id":null,"uuid":"uuid-stream-001"}`),

	// unknown type (fallback to CLIUnknownEvent)
	[]byte(`{"type":"custom_extension","payload":{"key":"value"},"session_id":"sess-abc123","uuid":"uuid-unk-001"}`),
}

// benchLineCount is the number of NDJSON lines in the fixture set.
var benchLineCount = len(benchNDJSONLines)

// BenchmarkNDJSONParsing benchmarks ParseCLIEvent on a single representative
// line of each event type.  b.N iterations cycle through the fixture set.
//
// The metric to watch is ns/op — divide 1e9 by ns/op to get lines/sec.
func BenchmarkNDJSONParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := range b.N {
		line := benchNDJSONLines[i%benchLineCount]
		ev, err := ParseCLIEvent(line)
		if err != nil {
			b.Fatalf("ParseCLIEvent failed: %v", err)
		}
		_ = ev
	}
}

// BenchmarkNDJSONParsing1000Lines benchmarks parsing a synthetic 1000-line
// NDJSON stream built from the fixture set to measure sustained throughput.
//
// The b.N outer loop repeats the 1000-line pass so the benchmark framework
// can calibrate iteration count correctly.  b.SetBytes(1000) reports
// throughput in lines/sec via the MB/s column (1 "byte" = 1 line).
func BenchmarkNDJSONParsing1000Lines(b *testing.B) {
	b.ReportAllocs()

	const batchSize = 1000
	lines := make([][]byte, batchSize)
	for i := range batchSize {
		lines[i] = benchNDJSONLines[i%benchLineCount]
	}

	b.SetBytes(batchSize) // report throughput as "lines/s" via MB/s column
	b.ResetTimer()
	for range b.N {
		for _, line := range lines {
			ev, err := ParseCLIEvent(line)
			if err != nil {
				b.Fatalf("ParseCLIEvent failed: %v", err)
			}
			_ = ev
		}
	}
}

// BenchmarkNDJSONParsingByType benchmarks each event type individually so
// that hot and cold paths can be compared.
func BenchmarkNDJSONParsingByType(b *testing.B) {
	cases := []struct {
		name string
		line []byte
	}{
		{"SystemInit", benchNDJSONLines[0]},
		{"SystemHook", benchNDJSONLines[1]},
		{"Assistant/Text", benchNDJSONLines[3]},
		{"Assistant/ToolUse", benchNDJSONLines[4]},
		{"Assistant/Task", benchNDJSONLines[5]},
		{"User/ToolResult", benchNDJSONLines[6]},
		{"Result/Success", benchNDJSONLines[8]},
		{"StreamEvent", benchNDJSONLines[9]},
		{"Unknown", benchNDJSONLines[10]},
	}

	for _, tc := range cases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				ev, err := ParseCLIEvent(tc.line)
				if err != nil {
					b.Fatalf("ParseCLIEvent(%s) failed: %v", tc.name, err)
				}
				_ = ev
			}
		})
	}
}

// BenchmarkNDJSONParsingBlankLine confirms that the fast path for blank lines
// (skip immediately without JSON unmarshal) has negligible overhead.
func BenchmarkNDJSONParsingBlankLine(b *testing.B) {
	b.ReportAllocs()

	blank := []byte("   ")
	b.ResetTimer()
	for b.Loop() {
		ev, err := ParseCLIEvent(blank)
		if err != nil {
			b.Fatal("blank line must not error")
		}
		if ev != nil {
			b.Fatal("blank line must return nil event")
		}
	}
}

// Compile-time guard: ensure the fixture slice has the expected number of
// entries so that index accesses above are always in range.
func init() {
	const expected = 11
	if len(benchNDJSONLines) != expected {
		panic(fmt.Sprintf("benchNDJSONLines has %d entries, expected %d", len(benchNDJSONLines), expected))
	}
}
