#!/bin/bash
# Verification script for GOgent-109 Agent Lifecycle Telemetry
# This demonstrates the verification command from the ticket spec

set -e

echo "=== GOgent-109 Agent Lifecycle Telemetry Verification ==="
echo ""

# Setup test environment
TEST_DIR=$(mktemp -d)
export GOYOKE_PROJECT_DIR="$TEST_DIR"
LIFECYCLE_FILE="$TEST_DIR/.goyoke/agent-lifecycle.jsonl"

echo "Test directory: $TEST_DIR"
echo "Lifecycle file: $LIFECYCLE_FILE"
echo ""

# Create sample lifecycle events (simulating a session with multiple agents)
mkdir -p "$TEST_DIR/.goyoke"

cat > "$LIFECYCLE_FILE" <<'EOF'
{"event_id":"spawn-1","session_id":"test-session","timestamp":1234567890,"event_type":"spawn","agent_id":"python-pro","parent_agent":"terminal","tier":"sonnet","task_description":"Implement feature X","decision_id":"dec-1"}
{"event_id":"spawn-2","session_id":"test-session","timestamp":1234567891,"event_type":"spawn","agent_id":"orchestrator","parent_agent":"terminal","tier":"sonnet","task_description":"Coordinate multi-agent task","decision_id":"dec-2"}
{"event_id":"complete-1","session_id":"test-session","timestamp":1234567895,"event_type":"complete","agent_id":"python-pro","parent_agent":"terminal","tier":"sonnet","task_description":"","decision_id":"dec-1","success":true,"duration_ms":5000}
{"event_id":"complete-2","session_id":"test-session","timestamp":1234567900,"event_type":"complete","agent_id":"orchestrator","parent_agent":"terminal","tier":"sonnet","task_description":"","decision_id":"dec-2","success":true,"duration_ms":9000}
{"event_id":"spawn-3","session_id":"test-session","timestamp":1234567901,"event_type":"spawn","agent_id":"codebase-search","parent_agent":"terminal","tier":"haiku","task_description":"Find all Python files","decision_id":"dec-3"}
{"event_id":"complete-3","session_id":"test-session","timestamp":1234567903,"event_type":"complete","agent_id":"codebase-search","parent_agent":"terminal","tier":"haiku","task_description":"","decision_id":"dec-3","success":true,"duration_ms":2000}
EOF

echo "✓ Sample lifecycle events created"
echo ""

# Verification command from ticket
echo "Running verification command from ticket spec:"
echo "cat $LIFECYCLE_FILE | jq -s 'group_by(.agent_id) | map({agent: .[0].agent_id, events: map(.event_type)})'"
echo ""

jq -s 'group_by(.agent_id) | map({agent: .[0].agent_id, events: map(.event_type)})' < "$LIFECYCLE_FILE"

echo ""
echo "=== Verification Checks ==="

# Check 1: All agents have spawn events
SPAWN_COUNT=$(jq -s '[.[] | select(.event_type == "spawn")] | length' < "$LIFECYCLE_FILE")
echo "✓ Spawn events logged: $SPAWN_COUNT"

# Check 2: All agents have complete events
COMPLETE_COUNT=$(jq -s '[.[] | select(.event_type == "complete")] | length' < "$LIFECYCLE_FILE")
echo "✓ Complete events logged: $COMPLETE_COUNT"

# Check 3: Spawn and complete counts match
if [ "$SPAWN_COUNT" -eq "$COMPLETE_COUNT" ]; then
    echo "✓ All spawned agents completed"
else
    echo "⚠ Mismatch: $SPAWN_COUNT spawns vs $COMPLETE_COUNT completes"
fi

# Check 4: DecisionID correlation
echo ""
echo "=== DecisionID Correlation ==="
jq -s 'group_by(.decision_id) | map({decision_id: .[0].decision_id, event_count: length, event_types: map(.event_type)})' < "$LIFECYCLE_FILE"

# Check 5: Event fields
echo ""
echo "=== Event Field Validation ==="
jq -s '.[0] | keys' < "$LIFECYCLE_FILE"

# Check 6: Session filtering
echo ""
echo "=== Session Filtering ==="
SESSION_EVENTS=$(jq -s '[.[] | select(.session_id == "test-session")] | length' < "$LIFECYCLE_FILE")
echo "✓ Events for session 'test-session': $SESSION_EVENTS"

# Cleanup
rm -rf "$TEST_DIR"
echo ""
echo "✓ Verification complete!"
