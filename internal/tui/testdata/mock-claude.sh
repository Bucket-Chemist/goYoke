#!/usr/bin/env bash
# mock-claude.sh — A mock Claude CLI for integration testing.
# Emits known NDJSON events on stdout and accepts stream-json on stdin.
#
# Behavior:
#   1. Emit system.init event
#   2. Wait for stdin input (a user message in stream-json format)
#   3. Emit assistant response + result event
#   4. Exit cleanly on SIGINT (trap handler)
#
# Environment variables:
#   MOCK_SESSION_ID  — session ID to report (default: "mock-session-001")
#   MOCK_COST        — cost to report (default: "0.0042")
#   MOCK_RESPONSE    — assistant text (default: "Mock response from test script")
#   MOCK_EXIT_CODE   — exit code (default: 0)
#   MOCK_CRASH       — if "true", exit immediately without events
#   MOCK_UNKNOWN     — if "true", emit an unknown event type before response

set -euo pipefail

SESSION_ID="${MOCK_SESSION_ID:-mock-session-001}"
COST="${MOCK_COST:-0.0042}"
RESPONSE="${MOCK_RESPONSE:-Mock response from test script}"
EXIT_CODE="${MOCK_EXIT_CODE:-0}"

# Signal handling: clean exit on SIGINT/SIGTERM
trap 'exit 0' INT TERM

# Crash mode: exit immediately
if [[ "${MOCK_CRASH:-}" == "true" ]]; then
    exit 1
fi

# Phase 1: Emit system.init
echo '{"type":"system","subtype":"init","session_id":"'"$SESSION_ID"'","model":"claude-sonnet-4-20250514","tools":["Read","Write","Bash"]}'

# Phase 2: If MOCK_UNKNOWN is set, emit an unknown event type
if [[ "${MOCK_UNKNOWN:-}" == "true" ]]; then
    echo '{"type":"xyzzy","subtype":"alien","data":"unknown event"}'
fi

# Phase 3: Wait for user input on stdin (one line of stream-json)
if read -r input_line; then
    # Phase 4: Emit assistant response
    echo '{"type":"assistant","message":{"id":"msg-mock-001","type":"message","role":"assistant","content":[{"type":"text","text":"'"$RESPONSE"'"}],"model":"claude-sonnet-4-20250514","stop_reason":"end_turn"}}'

    # Phase 5: Emit result
    echo '{"type":"result","subtype":"success","session_id":"'"$SESSION_ID"'","total_cost_usd":'"$COST"',"duration_ms":1234}'
fi

exit "$EXIT_CODE"
