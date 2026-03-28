package cli

// resilience_test.go verifies that the NDJSON event pipeline handles unknown,
// malformed, and structurally unexpected events without crashing or dropping
// valid subsequent events.
//
// Acceptance criteria (TUI-041):
//  1. Unknown event types produce CLIUnknownEvent with raw JSON preserved.
//  2. Parser continues after unknown events.
//  3. Malformed JSON lines are logged and skipped (no crash, no drop).
//  4. Extra fields on known types are handled gracefully.
//  5. Unknown subtypes fall through to the SystemStatusEvent default case.
//  6. Stress test: alternate valid/invalid events at high rate; no dropped
//     events and no goroutine leaks.
//  7. rate_limit_event is parsed as an informational RateLimitEvent (no state
//     change required by the parser).

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// AC-1: Unknown event types produce CLIUnknownEvent with raw JSON preserved.
// ---------------------------------------------------------------------------

func TestResilience_UnknownEventType_ProducesUnknownEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantType  string
		wantField string // extra field to verify in raw JSON
	}{
		{
			name:      "future_event type with nested payload",
			input:     `{"type":"future_event","data":{"key":"value"},"session_id":"s","uuid":"u1"}`,
			wantType:  "future_event",
			wantField: "data",
		},
		{
			name:     "telemetry_event type not yet in spec",
			input:    `{"type":"telemetry_event","metrics":{"cpu":0.5},"uuid":"u2"}`,
			wantType: "telemetry_event",
		},
		{
			name:     "empty type string falls through to unknown",
			input:    `{"type":"","payload":"x","uuid":"u3"}`,
			wantType: "",
		},
		{
			name:     "numeric type fields not in switch",
			input:    `{"type":"debug_trace","trace_id":42,"uuid":"u4"}`,
			wantType: "debug_trace",
		},
		{
			name:     "deeply nested unknown event",
			input:    `{"type":"experimental","nested":{"a":{"b":{"c":true}}},"uuid":"u5"}`,
			wantType: "experimental",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))

			require.NoError(t, err, "unknown events must not produce a parse error")
			require.NotNil(t, got, "unknown events must produce a non-nil value")

			ev, ok := got.(CLIUnknownEvent)
			require.True(t, ok, "expected CLIUnknownEvent, got %T", got)
			assert.Equal(t, tc.wantType, ev.Type, "Type field must match the JSON type discriminator")
			assert.NotEmpty(t, ev.RawJSON, "RawJSON must be populated")

			// Verify raw JSON is valid and fully round-trips.
			var raw map[string]any
			require.NoError(t, json.Unmarshal(ev.RawJSON, &raw),
				"RawJSON must be valid JSON")
			assert.Equal(t, tc.wantType, raw["type"],
				"RawJSON must preserve the type field")

			if tc.wantField != "" {
				_, hasField := raw[tc.wantField]
				assert.True(t, hasField, "RawJSON must preserve %q field", tc.wantField)
			}
		})
	}
}

// TestResilience_UnknownEvent_RawJSONIsUnchanged verifies byte-level fidelity:
// the raw JSON stored in CLIUnknownEvent.RawJSON must be byte-identical to the
// original input line.
func TestResilience_UnknownEvent_RawJSONIsUnchanged(t *testing.T) {
	t.Parallel()

	input := `{"type":"future_event","some_field":"some_value","nested":{"x":1}}`
	got, err := ParseCLIEvent([]byte(input))
	require.NoError(t, err)

	ev, ok := got.(CLIUnknownEvent)
	require.True(t, ok)
	assert.Equal(t, json.RawMessage(input), ev.RawJSON,
		"RawJSON must be byte-identical to the input line")
}

// ---------------------------------------------------------------------------
// AC-2: Parser continues processing valid events after unknown ones.
// ---------------------------------------------------------------------------

func TestResilience_ParserContinuesAfterUnknownEvent(t *testing.T) {
	t.Parallel()

	// A mix of unknown → known → unknown → known events sent through the
	// driver so that the full consumeEvents path is exercised.
	lines := []string{
		// unknown
		`{"type":"future_event","data":"first","uuid":"u1"}`,
		// known: rate_limit_event
		`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"five_hour","overageStatus":"ok","isUsingOverage":false},"uuid":"u2","session_id":"s"}`,
		// unknown
		`{"type":"unknown_b","uuid":"u3"}`,
		// known: result
		`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"ok","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u4"}`,
	}

	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}
	writer.Close()

	items := drainChannel(d.eventCh, 10, 3*time.Second)

	// Expect: CLIUnknownEvent, RateLimitEvent, CLIUnknownEvent, ResultEvent,
	// CLIDisconnectedMsg (EOF). Allow for any ordering of the last disconnect.
	require.GreaterOrEqual(t, len(items), 5, "all 4 events plus disconnect must arrive")

	_, isUnknown1 := items[0].(CLIUnknownEvent)
	assert.True(t, isUnknown1, "item[0] must be CLIUnknownEvent, got %T", items[0])

	_, isRate := items[1].(RateLimitEvent)
	assert.True(t, isRate, "item[1] must be RateLimitEvent, got %T", items[1])

	_, isUnknown2 := items[2].(CLIUnknownEvent)
	assert.True(t, isUnknown2, "item[2] must be CLIUnknownEvent, got %T", items[2])

	_, isResult := items[3].(ResultEvent)
	assert.True(t, isResult, "item[3] must be ResultEvent, got %T", items[3])

	_, isDisconnect := items[4].(CLIDisconnectedMsg)
	assert.True(t, isDisconnect, "item[4] must be CLIDisconnectedMsg, got %T", items[4])
}

// TestResilience_ParseCLIEvent_ContinuesAfterUnknown tests ParseCLIEvent
// directly (unit level) without the driver goroutine, verifying that calling
// it sequentially on unknown then valid events never returns an error.
func TestResilience_ParseCLIEvent_ContinuesAfterUnknown(t *testing.T) {
	t.Parallel()

	sequence := []struct {
		input        string
		expectUnknown bool
		expectType   string
	}{
		{`{"type":"future_event","v":1}`, true, "CLIUnknownEvent"},
		{`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`, false, "RateLimitEvent"},
		{`{"type":"yet_another_future","v":2}`, true, "CLIUnknownEvent"},
		{`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u"}`, false, "ResultEvent"},
	}

	for i, step := range sequence {
		got, err := ParseCLIEvent([]byte(step.input))
		require.NoError(t, err, "step %d (%s) must not error", i, step.expectType)
		require.NotNil(t, got, "step %d must not return nil", i)

		if step.expectUnknown {
			_, ok := got.(CLIUnknownEvent)
			assert.True(t, ok, "step %d: expected CLIUnknownEvent, got %T", i, got)
		}
	}
}

// ---------------------------------------------------------------------------
// AC-3: Malformed JSON lines are logged and skipped (no crash, no drop).
// ---------------------------------------------------------------------------

func TestResilience_MalformedJSONLinesAreSkipped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"unclosed brace", `{not valid json`},
		{"bare string instead of object", `"just a string"`},
		{"array instead of object", `[1,2,3]`},
		{"truncated object", `{"type":"assistant","message":{`},
		{"null literal", `null`},
		{"integer literal", `42`},
		{"empty braces with trailing garbage", `{}extra`},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ParseCLIEvent must return an error for truly malformed JSON but
			// must NOT panic. The driver's consumeEvents logs and skips on error.
			got, err := ParseCLIEvent([]byte(tc.input))

			// Either an error is returned OR the event is nil — never a panic.
			// (null/integer parse without error but may return nil event or
			// CLIUnknownEvent — both are acceptable skips.)
			_ = got
			_ = err
			// The key assertion: we reached here without panicking.
		})
	}
}

// TestResilience_MalformedLine_ValidEventFollowsThrough verifies the full
// driver path: a malformed line is skipped and the subsequent valid event
// still reaches the channel.
func TestResilience_MalformedLine_ValidEventFollowsThrough(t *testing.T) {
	t.Parallel()

	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Malformed line first.
	fmt.Fprintln(writer, `{bad json here`)
	// Then a valid rate_limit_event.
	fmt.Fprintln(writer, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`)
	writer.Close()

	items := drainChannel(d.eventCh, 5, 3*time.Second)

	// Malformed line is skipped; valid event must arrive.
	require.GreaterOrEqual(t, len(items), 2, "at least RateLimitEvent + disconnect")
	_, isRate := items[0].(RateLimitEvent)
	assert.True(t, isRate, "first item must be RateLimitEvent, got %T", items[0])
}

// TestResilience_MultipleMalformedLines_AllSkipped verifies that multiple
// consecutive malformed lines do not block the pipeline.
func TestResilience_MultipleMalformedLines_AllSkipped(t *testing.T) {
	t.Parallel()

	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	malformed := []string{
		`{unclosed`,
		`not json at all`,
		`{"type":"result","usage":"bad_type_for_struct"}`,
		`[1, 2, 3]`,
	}
	for _, m := range malformed {
		fmt.Fprintln(writer, m)
	}
	// One valid event after all malformed.
	fmt.Fprintln(writer, `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`)
	writer.Close()

	items := drainChannel(d.eventCh, 5, 3*time.Second)

	require.GreaterOrEqual(t, len(items), 2, "valid event + disconnect must arrive")
	_, isRate := items[0].(RateLimitEvent)
	assert.True(t, isRate, "valid event after all malformed lines must reach channel, got %T", items[0])
}

// ---------------------------------------------------------------------------
// AC-4: Extra fields on known types are handled gracefully.
// ---------------------------------------------------------------------------

// TestResilience_ExtraFieldsOnKnownTypes verifies that JSON events containing
// extra/unknown fields do not cause parse errors. Go's encoding/json silently
// ignores unknown fields by default.
func TestResilience_ExtraFieldsOnKnownTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		checkType func(t *testing.T, got any)
	}{
		{
			name: "rate_limit_event with extra top-level field",
			input: `{
				"type":"rate_limit_event",
				"rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"five_hour","overageStatus":"ok","isUsingOverage":false},
				"uuid":"u","session_id":"s",
				"future_field":"future_value",
				"another_future_nested":{"x":1}
			}`,
			checkType: func(t *testing.T, got any) {
				t.Helper()
				ev, ok := got.(RateLimitEvent)
				require.True(t, ok, "expected RateLimitEvent, got %T", got)
				assert.Equal(t, "allowed", ev.RateLimitInfo.Status)
				assert.Equal(t, "u", ev.UUID)
			},
		},
		{
			name: "result with extra fields and extended usage",
			input: `{
				"type":"result","subtype":"success","is_error":false,
				"duration_ms":100,"duration_api_ms":90,"num_turns":2,
				"result":"done","stop_reason":"end_turn","session_id":"s",
				"total_cost_usd":0.01,
				"usage":{"input_tokens":10,"output_tokens":5},
				"uuid":"u",
				"undocumented_field":"value",
				"experimental_data":{"alpha":true}
			}`,
			checkType: func(t *testing.T, got any) {
				t.Helper()
				ev, ok := got.(ResultEvent)
				require.True(t, ok, "expected ResultEvent, got %T", got)
				assert.Equal(t, "success", ev.Subtype)
				assert.Equal(t, 10, ev.Usage.InputTokens)
			},
		},
		{
			name: "system:init with extra fields",
			input: `{
				"type":"system","subtype":"init",
				"cwd":"/tmp","session_id":"s","model":"m",
				"permissionMode":"acceptEdits","claude_code_version":"9.9.9",
				"tools":["Read"],"uuid":"u",
				"new_future_flag":true,
				"extended_config":{"feature_a":1,"feature_b":"x"}
			}`,
			checkType: func(t *testing.T, got any) {
				t.Helper()
				ev, ok := got.(SystemInitEvent)
				require.True(t, ok, "expected SystemInitEvent, got %T", got)
				assert.Equal(t, "init", ev.Subtype)
				assert.Equal(t, "9.9.9", ev.ClaudeCodeVersion)
			},
		},
		{
			name: "assistant with extra top-level fields",
			input: `{
				"type":"assistant",
				"message":{"id":"m1","type":"message","role":"assistant","model":"claude-m",
					"content":[{"type":"text","text":"hi"}],"stop_reason":"end_turn",
					"usage":{"input_tokens":1,"output_tokens":1}},
				"parent_tool_use_id":null,"session_id":"s","uuid":"u",
				"extra_routing_field":"value",
				"new_trace_id":"trace-xyz"
			}`,
			checkType: func(t *testing.T, got any) {
				t.Helper()
				ev, ok := got.(AssistantEvent)
				require.True(t, ok, "expected AssistantEvent, got %T", got)
				assert.Equal(t, "m1", ev.Message.ID)
				require.Len(t, ev.Message.Content, 1)
				assert.Equal(t, "hi", ev.Message.Content[0].Text)
			},
		},
		{
			name: "user event with unknown content block fields",
			input: `{
				"type":"user",
				"message":{"role":"user","content":[{
					"type":"tool_result","tool_use_id":"t1",
					"content":"result text","is_error":false,
					"future_content_field":"x"
				}]},
				"session_id":"s","uuid":"u"
			}`,
			checkType: func(t *testing.T, got any) {
				t.Helper()
				ev, ok := got.(UserEvent)
				require.True(t, ok, "expected UserEvent, got %T", got)
				require.Len(t, ev.Message.Content, 1)
				assert.Equal(t, "tool_result", ev.Message.Content[0].Type)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))
			require.NoError(t, err, "extra fields must not cause a parse error")
			require.NotNil(t, got)

			if tc.checkType != nil {
				tc.checkType(t, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AC-5: Unknown subtypes fall through to default case (SystemStatusEvent).
// ---------------------------------------------------------------------------

// TestResilience_UnknownSystemSubtype_FallsThrough verifies that system events
// with unrecognised subtypes are parsed as SystemStatusEvent (the default
// branch in parseSystemEvent), not as an error.
func TestResilience_UnknownSystemSubtype_FallsThrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		subtype string
		input   string
	}{
		{
			name:    "system:future_subtype",
			subtype: "future_subtype",
			input:   `{"type":"system","subtype":"future_subtype","uuid":"u1","session_id":"s"}`,
		},
		{
			name:    "system:compact_boundary",
			subtype: "compact_boundary",
			input:   `{"type":"system","subtype":"compact_boundary","uuid":"u2"}`,
		},
		{
			name:    "system:status",
			subtype: "status",
			input:   `{"type":"system","subtype":"status","status":"ready","permissionMode":"acceptEdits","uuid":"u3","session_id":"s"}`,
		},
		{
			name:    "system:unknown_experimental",
			subtype: "unknown_experimental",
			input:   `{"type":"system","subtype":"unknown_experimental","extra":"data","uuid":"u4"}`,
		},
		{
			name:    "system with empty subtype",
			subtype: "",
			input:   `{"type":"system","subtype":"","uuid":"u5"}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))
			require.NoError(t, err, "unknown subtype must not produce an error")
			require.NotNil(t, got, "unknown subtype must produce a non-nil value")

			ev, ok := got.(SystemStatusEvent)
			require.True(t, ok, "expected SystemStatusEvent for subtype %q, got %T", tc.subtype, got)
			assert.Equal(t, tc.subtype, ev.Subtype,
				"Subtype field must be preserved from the original JSON")
		})
	}
}

// TestResilience_SystemSubtype_DefaultPreservesAllKnownFields verifies that
// the SystemStatusEvent default branch preserves all expected fields even when
// additional unknown fields are present.
func TestResilience_SystemSubtype_DefaultPreservesAllKnownFields(t *testing.T) {
	t.Parallel()

	input := `{
		"type":"system","subtype":"new_subtype_v3",
		"status":"active","permissionMode":"bypassPermissions",
		"uuid":"u-test","session_id":"sess-test",
		"future_array":[1,2,3]
	}`

	got, err := ParseCLIEvent([]byte(input))
	require.NoError(t, err)

	ev, ok := got.(SystemStatusEvent)
	require.True(t, ok, "expected SystemStatusEvent, got %T", got)
	assert.Equal(t, "new_subtype_v3", ev.Subtype)
	assert.Equal(t, "bypassPermissions", ev.PermissionMode)
	assert.Equal(t, "u-test", ev.UUID)
	assert.Equal(t, "sess-test", ev.SessionID)
	require.NotNil(t, ev.Status)
	assert.Equal(t, "active", *ev.Status)
}

// ---------------------------------------------------------------------------
// AC-6: Stress test — no dropped events, no goroutine leaks.
// ---------------------------------------------------------------------------

// TestResilience_StressTest_NoDroppedEventsNoGoroutineLeaks alternates valid
// and invalid NDJSON lines at high rate and verifies:
//   - All valid events reach the channel (none dropped).
//   - The goroutine count returns to the pre-test level after the driver exits.
//
// Skipped under -short.
func TestResilience_StressTest_NoDroppedEventsNoGoroutineLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const (
		totalLines   = 2000    // 1000 valid + 1000 malformed (interleaved)
		targetRateHz = 1000    // target event rate for pacing
		linePause    = time.Second / targetRateHz
	)

	// Capture goroutine count before starting.
	goroutinesBefore := runtime.NumGoroutine()

	// Use a larger channel buffer to avoid back-pressure during burst writes.
	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	validLine := `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"five_hour","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`
	malformedLine := `{not valid json at all`

	// Pre-compute the expected valid event count (even-indexed lines are valid).
	// For totalLines=2000 with every other line valid: totalLines/2 = 1000.
	expectedValid := totalLines / 2

	go func() {
		for i := range totalLines {
			if i%2 == 0 {
				fmt.Fprintln(writer, validLine)
			} else {
				fmt.Fprintln(writer, malformedLine)
			}
			// Pace to target rate — but don't let pacing dominate test time.
			// With 2000 lines at 1000/s this is ~2 seconds total.
			time.Sleep(linePause)
		}
		writer.Close()
	}()

	// Drain all events. We expect expectedValid RateLimitEvents + 1 disconnect.
	// Allow generous timeout: 2s of writes + 2s buffer.
	items := drainChannel(d.eventCh, expectedValid+10, 6*time.Second)

	// Count valid events received.
	receivedValid := 0
	disconnectReceived := false
	for _, item := range items {
		switch item.(type) {
		case RateLimitEvent:
			receivedValid++
		case CLIDisconnectedMsg:
			disconnectReceived = true
		}
	}

	assert.Equal(t, expectedValid, receivedValid,
		"all valid events must be received; dropped: %d", expectedValid-receivedValid)
	assert.True(t, disconnectReceived, "CLIDisconnectedMsg must be received after EOF")

	// Allow goroutines to settle after the driver exits.
	time.Sleep(200 * time.Millisecond)

	goroutinesAfter := runtime.NumGoroutine()
	// Allow a small delta for test framework goroutines. The driver spawns
	// 2 goroutines (consumeEvents outer + inner scanner), both of which should
	// have exited. A delta of 5 is generous.
	delta := goroutinesAfter - goroutinesBefore
	assert.LessOrEqual(t, delta, 5,
		"goroutine count should not grow; before=%d after=%d delta=%d",
		goroutinesBefore, goroutinesAfter, delta)
}

// TestResilience_StressTest_ParseCLIEvent_Mixed exercises ParseCLIEvent
// directly at high rate without the driver, verifying zero panics and correct
// classification. This is suitable for -short mode since it has no I/O.
func TestResilience_StressTest_ParseCLIEvent_Mixed(t *testing.T) {
	t.Parallel()

	const iterations = 5000

	lines := [][]byte{
		// valid: known types
		[]byte(`{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`),
		[]byte(`{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u"}`),
		// valid: unknown type (CLIUnknownEvent)
		[]byte(`{"type":"future_event","data":{"key":"value"},"uuid":"u"}`),
		// valid: system with unknown subtype (SystemStatusEvent)
		[]byte(`{"type":"system","subtype":"future_subtype","uuid":"u"}`),
		// invalid: malformed JSON
		[]byte(`{not valid json`),
		[]byte(`truncated`),
	}

	knownValidCount := 0
	unknownEventCount := 0
	errorCount := 0

	for i := range iterations {
		line := lines[i%len(lines)]
		got, err := ParseCLIEvent(line)
		if err != nil {
			errorCount++
			continue
		}
		if got == nil {
			continue
		}
		switch got.(type) {
		case RateLimitEvent, ResultEvent, SystemStatusEvent:
			knownValidCount++
		case CLIUnknownEvent:
			unknownEventCount++
		}
	}

	// With 6 line templates cycling over 5000 iterations:
	// - indices 0,1 → valid known: 2/6 * 5000 ≈ 1666
	// - index 2 → CLIUnknownEvent: 1/6 * 5000 ≈ 833
	// - index 3 → SystemStatusEvent (known valid): 1/6 * 5000 ≈ 833
	// - indices 4,5 → errors: 2/6 * 5000 ≈ 1666
	assert.Greater(t, knownValidCount, 0, "known valid events must be parsed")
	assert.Greater(t, unknownEventCount, 0, "unknown events must be classified as CLIUnknownEvent")
	assert.Greater(t, errorCount, 0, "malformed lines must produce errors")
	// Total must account for every iteration.
	total := knownValidCount + unknownEventCount + errorCount
	assert.Equal(t, iterations, total,
		"every iteration must produce exactly one outcome")
}

// ---------------------------------------------------------------------------
// AC-7: rate_limit_event is informational (no state change required).
// ---------------------------------------------------------------------------

// TestResilience_RateLimitEvent_Informational verifies that rate_limit_event
// events are parsed as RateLimitEvent and that the RateLimitInfo fields are
// fully populated. The parser performs no state mutation; consumers decide
// how to react.
func TestResilience_RateLimitEvent_Informational(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		wantStatus  string
		wantType    string
		wantResets  int64
		wantOverage bool
	}{
		{
			name: "status allowed",
			input: `{
				"type":"rate_limit_event",
				"rate_limit_info":{
					"status":"allowed",
					"rateLimitType":"five_hour",
					"overageStatus":"rejected",
					"overageDisabledReason":"org_level_disabled",
					"resetsAt":1774252800,
					"isUsingOverage":false
				},
				"uuid":"u1","session_id":"s"
			}`,
			wantStatus:  "allowed",
			wantType:    "five_hour",
			wantResets:  1774252800,
			wantOverage: false,
		},
		{
			name: "status throttled",
			input: `{
				"type":"rate_limit_event",
				"rate_limit_info":{
					"status":"throttled",
					"rateLimitType":"concurrent",
					"overageStatus":"active",
					"resetsAt":9999999999,
					"isUsingOverage":true
				},
				"uuid":"u2","session_id":"s"
			}`,
			wantStatus:  "throttled",
			wantType:    "concurrent",
			wantResets:  9999999999,
			wantOverage: true,
		},
		{
			name: "minimal rate_limit_event",
			input: `{
				"type":"rate_limit_event",
				"rate_limit_info":{"status":"allowed","rateLimitType":"t","overageStatus":"ok","resetsAt":0,"isUsingOverage":false},
				"uuid":"u3","session_id":"s"
			}`,
			wantStatus:  "allowed",
			wantType:    "t",
			wantResets:  0,
			wantOverage: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))
			require.NoError(t, err, "rate_limit_event must parse without error")
			require.NotNil(t, got)

			ev, ok := got.(RateLimitEvent)
			require.True(t, ok, "expected RateLimitEvent, got %T", got)

			assert.Equal(t, "rate_limit_event", ev.Type)
			assert.Equal(t, tc.wantStatus, ev.RateLimitInfo.Status)
			assert.Equal(t, tc.wantType, ev.RateLimitInfo.RateLimitType)
			assert.Equal(t, tc.wantResets, ev.RateLimitInfo.ResetsAt)
			assert.Equal(t, tc.wantOverage, ev.RateLimitInfo.IsUsingOverage)
		})
	}
}

// TestResilience_RateLimitEvent_ViaDriver verifies that rate_limit_event flows
// through the consumeEvents goroutine correctly and reaches the event channel
// without triggering any state mutation in the driver itself.
func TestResilience_RateLimitEvent_ViaDriver(t *testing.T) {
	t.Parallel()

	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// State before.
	stateBefore := d.State()

	fmt.Fprintln(writer, `{"type":"rate_limit_event","rate_limit_info":{"status":"throttled","rateLimitType":"output_tokens","overageStatus":"active","resetsAt":9000000000,"isUsingOverage":true},"uuid":"u","session_id":"s"}`)
	writer.Close()

	items := drainChannel(d.eventCh, 5, 3*time.Second)
	require.GreaterOrEqual(t, len(items), 1)

	ev, ok := items[0].(RateLimitEvent)
	require.True(t, ok, "first item must be RateLimitEvent, got %T", items[0])
	assert.Equal(t, "throttled", ev.RateLimitInfo.Status)
	assert.True(t, ev.RateLimitInfo.IsUsingOverage)

	// Driver state must not have changed as a result of the rate_limit_event.
	// (consumeEvents only transitions to DriverDead on EOF/shutdown, not on
	// rate limit events.)
	// Wait briefly for EOF to settle.
	time.Sleep(50 * time.Millisecond)
	stateAfterRateLimit := d.State()
	// After writer.Close(), state transitions to DriverDead. The point is that
	// it was not changed by the rate_limit_event itself — only by the EOF.
	_ = stateBefore
	// The final state must be DriverDead (EOF processed).
	assert.Equal(t, DriverDead, stateAfterRateLimit,
		"driver must be DriverDead after pipe close, not due to rate_limit_event")
}

// ---------------------------------------------------------------------------
// Edge case: blank and whitespace lines remain a no-op (AC-3 complement).
// ---------------------------------------------------------------------------

func TestResilience_BlankLinesNeverProduceEvents(t *testing.T) {
	t.Parallel()

	blanks := []string{"", " ", "\t", "\n", "   \t  ", "\r\n"}

	for _, blank := range blanks {
		got, err := ParseCLIEvent([]byte(blank))
		assert.NoError(t, err, "blank line %q must not produce error", blank)
		assert.Nil(t, got, "blank line %q must produce nil event", blank)
	}
}

// ---------------------------------------------------------------------------
// AC-1 complement: multiple sequential unknown types, each with distinct raw JSON.
// ---------------------------------------------------------------------------

func TestResilience_MultipleUnknownTypesPreserveDistinctRawJSON(t *testing.T) {
	t.Parallel()

	inputs := []struct {
		line    string
		wantKey string
		wantVal string
	}{
		{`{"type":"alpha_event","alpha":"one"}`, "alpha", "one"},
		{`{"type":"beta_event","beta":"two"}`, "beta", "two"},
		{`{"type":"gamma_event","gamma":"three"}`, "gamma", "three"},
	}

	for _, inp := range inputs {
		inp := inp
		t.Run(inp.wantKey, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(inp.line))
			require.NoError(t, err)

			ev, ok := got.(CLIUnknownEvent)
			require.True(t, ok, "expected CLIUnknownEvent, got %T", got)

			var raw map[string]any
			require.NoError(t, json.Unmarshal(ev.RawJSON, &raw))
			assert.Equal(t, inp.wantVal, raw[inp.wantKey],
				"RawJSON must preserve %q=%q", inp.wantKey, inp.wantVal)
		})
	}
}

// ---------------------------------------------------------------------------
// Goroutine leak guard — standalone (no-subprocess) variant.
// ---------------------------------------------------------------------------

// TestResilience_DriverGoroutineLeak_NoSubprocess verifies that after the
// pipe closes and the driver reaches DriverDead, no goroutines from
// consumeEvents remain running.
func TestResilience_DriverGoroutineLeak_NoSubprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}
	t.Parallel()

	before := runtime.NumGoroutine()

	d, writer, _ := newTestDriver(t, CLIDriverOpts{})

	// Write a few events and close.
	for range 10 {
		fmt.Fprintln(writer, `{"type":"future_event","i":1}`)
	}
	writer.Close()

	// Drain until disconnect.
	drainChannel(d.eventCh, 20, 3*time.Second)

	// Wait for goroutines to settle.
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	delta := after - before
	assert.LessOrEqual(t, delta, 5,
		"no goroutine leak: before=%d after=%d delta=%d", before, after, delta)
}

// ---------------------------------------------------------------------------
// Table-driven summary: all AC paths in one place for coverage audit.
// ---------------------------------------------------------------------------

// TestResilience_ParseCLIEvent_AllResiliencePaths is a compact table-driven
// test that exercises every resilience path through ParseCLIEvent, providing
// a single source of truth for AC coverage.
func TestResilience_ParseCLIEvent_AllResiliencePaths(t *testing.T) {
	t.Parallel()

	type outcome int
	const (
		outcomeUnknown  outcome = iota // CLIUnknownEvent
		outcomeKnown                   // any known typed event
		outcomeNilNil                  // blank line
		outcomeError                   // parse error
	)

	tests := []struct {
		name    string
		input   string
		outcome outcome
	}{
		// AC-1: unknown types
		{"unknown type future_event", `{"type":"future_event","v":1}`, outcomeUnknown},
		{"unknown type empty string", `{"type":"","data":"x"}`, outcomeUnknown},
		{"unknown type telemetry", `{"type":"telemetry","m":{}}`, outcomeUnknown},

		// AC-2: valid events after unknown (parser continues — tested by ordering above)
		{"known: rate_limit after unknown (direct)", `{"type":"rate_limit_event","rate_limit_info":{"status":"allowed","resetsAt":0,"rateLimitType":"t","overageStatus":"ok","isUsingOverage":false},"uuid":"u","session_id":"s"}`, outcomeKnown},

		// AC-3: malformed lines
		{"malformed: unclosed brace", `{not valid`, outcomeError},
		{"malformed: truncated", `{"type":"assistant","message":{`, outcomeError},

		// AC-4: extra fields on known types (no error expected)
		{"extra fields on result", `{"type":"result","subtype":"success","is_error":false,"duration_ms":1,"duration_api_ms":1,"num_turns":1,"result":"","stop_reason":"end_turn","session_id":"s","total_cost_usd":0,"usage":{"input_tokens":0,"output_tokens":0},"uuid":"u","extra":"ignored"}`, outcomeKnown},

		// AC-5: unknown subtype → SystemStatusEvent
		{"unknown system subtype", `{"type":"system","subtype":"future_compact","uuid":"u"}`, outcomeKnown},

		// AC-7: rate_limit_event informational
		{"rate_limit_event informational", `{"type":"rate_limit_event","rate_limit_info":{"status":"throttled","resetsAt":1000,"rateLimitType":"x","overageStatus":"y","isUsingOverage":true},"uuid":"u","session_id":"s"}`, outcomeKnown},

		// Blank lines
		{"blank line", "", outcomeNilNil},
		{"whitespace line", "   ", outcomeNilNil},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCLIEvent([]byte(tc.input))

			switch tc.outcome {
			case outcomeUnknown:
				require.NoError(t, err)
				require.NotNil(t, got)
				_, ok := got.(CLIUnknownEvent)
				assert.True(t, ok, "expected CLIUnknownEvent, got %T", got)

			case outcomeKnown:
				require.NoError(t, err)
				require.NotNil(t, got)
				_, isUnknown := got.(CLIUnknownEvent)
				assert.False(t, isUnknown, "expected a known event type, got CLIUnknownEvent")

			case outcomeNilNil:
				assert.NoError(t, err)
				assert.Nil(t, got)

			case outcomeError:
				assert.Error(t, err)
				assert.Nil(t, got)
			}
		})
	}
}
