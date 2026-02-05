#!/bin/bash
# test-mcp-spawn.sh - Comprehensive MCP spawn testing with full visibility

set -eo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Directories
PROJECT_ROOT="$HOME/Documents/GOgent-Fortress"
TUI_DIR="$PROJECT_ROOT/packages/tui"
LOG_DIR="$PROJECT_ROOT/.test-spawn-logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Log files
PROCESS_LOG="$LOG_DIR/processes_$TIMESTAMP.log"
NESTING_LOG="$LOG_DIR/nesting_$TIMESTAMP.log"
TUI_OUTPUT="$LOG_DIR/tui_output_$TIMESTAMP.log"
SPAWN_EVENTS="$LOG_DIR/spawn_events_$TIMESTAMP.log"

# Create log directory
mkdir -p "$LOG_DIR"

echo -e "${BOLD}${CYAN}=== MCP spawn_agent Test Harness ===${NC}"
echo ""
echo -e "${BLUE}Test Session: $TIMESTAMP${NC}"
echo -e "${BLUE}Log Directory: $LOG_DIR${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $1 found"
        return 0
    else
        echo -e "  ${RED}✗${NC} $1 not found"
        return 1
    fi
}

PREREQS_OK=true
check_command tmux || PREREQS_OK=false
check_command jq || PREREQS_OK=false
check_command claude || PREREQS_OK=false

if [ "$PREREQS_OK" = false ]; then
    echo -e "${RED}Missing prerequisites. Install and try again.${NC}"
    exit 1
fi

echo ""

# Check TUI build
if [ ! -f "$TUI_DIR/dist/index.js" ]; then
    echo -e "${YELLOW}TUI not built. Building now...${NC}"
    cd "$TUI_DIR"
    npm run build
    echo -e "${GREEN}✓ TUI built${NC}"
else
    echo -e "${GREEN}✓ TUI already built${NC}"
fi

echo ""

# Check feature flag
if [ "${GOGENT_MCP_SPAWN_ENABLED:-true}" = "false" ]; then
    echo -e "${RED}⚠️  GOGENT_MCP_SPAWN_ENABLED=false${NC}"
    echo -e "${YELLOW}spawn_agent tool will NOT be available${NC}"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo -e "${GREEN}✓ spawn_agent enabled (GOGENT_MCP_SPAWN_ENABLED not set to false)${NC}"
fi

echo ""
echo -e "${BOLD}${CYAN}=== Starting Test Environment ===${NC}"
echo ""

# Kill any existing tmux session
tmux kill-session -t mcp-spawn-test 2>/dev/null || true

# Create tmux session with multiple panes
echo -e "${YELLOW}Creating tmux session with 4 panes...${NC}"

tmux new-session -d -s mcp-spawn-test -n "MCP Spawn Test"

# Layout:
# +-------------------+-------------------+
# |                   |                   |
# |   TUI             |   Process Monitor |
# |                   |                   |
# +-------------------+-------------------+
# |                   |                   |
# |   Nesting Watch   |   Event Log       |
# |                   |                   |
# +-------------------+-------------------+

# Split into 4 panes
tmux split-window -h -t mcp-spawn-test:0
tmux split-window -v -t mcp-spawn-test:0.0
tmux split-window -v -t mcp-spawn-test:0.2

# Pane 0 (top-left): TUI output
tmux send-keys -t mcp-spawn-test:0.0 "cd $TUI_DIR" C-m
tmux send-keys -t mcp-spawn-test:0.0 "clear" C-m
tmux send-keys -t mcp-spawn-test:0.0 "echo -e '${BOLD}${CYAN}=== TUI Output ===${NC}'" C-m
tmux send-keys -t mcp-spawn-test:0.0 "echo 'Starting TUI... (logs to $TUI_OUTPUT)'" C-m
tmux send-keys -t mcp-spawn-test:0.0 "npm start 2>&1 | tee $TUI_OUTPUT" C-m

# Pane 1 (top-right): Process monitor
tmux send-keys -t mcp-spawn-test:0.1 "clear" C-m
tmux send-keys -t mcp-spawn-test:0.1 "echo -e '${BOLD}${GREEN}=== Process Monitor ===${NC}'" C-m
tmux send-keys -t mcp-spawn-test:0.1 "echo 'Monitoring claude processes...'" C-m
tmux send-keys -t mcp-spawn-test:0.1 "sleep 2" C-m
tmux send-keys -t mcp-spawn-test:0.1 "watch -n 0.5 -c 'echo -e \"${BOLD}Claude Processes:${NC}\"; echo \"\"; ps aux | grep -E \"claude|node.*tui\" | grep -v grep | grep -v watch | awk \"{printf \\\"%s %5s %s\\\\n\\\", \\$2, \\$3, \\$11}\" | head -20; echo \"\"; echo -e \"${BOLD}Total Claude processes:${NC} \$(ps aux | grep \"claude -p\" | grep -v grep | wc -l)\"; echo -e \"${BOLD}Nesting levels detected:${NC} \$(ps e | grep GOGENT_NESTING_LEVEL | grep -oP \"GOGENT_NESTING_LEVEL=\\K[0-9]+\" | sort -u | tr \"\\n\" \",\" | sed \"s/,\$//\")\"'" C-m

# Pane 2 (bottom-left): Nesting level watch
tmux send-keys -t mcp-spawn-test:0.2 "clear" C-m
tmux send-keys -t mcp-spawn-test:0.2 "echo -e '${BOLD}${BLUE}=== Nesting Level Watch ===${NC}'" C-m
tmux send-keys -t mcp-spawn-test:0.2 "echo 'Monitoring GOGENT_NESTING_LEVEL...'" C-m
tmux send-keys -t mcp-spawn-test:0.2 "sleep 2" C-m
tmux send-keys -t mcp-spawn-test:0.2 "while true; do clear; echo -e '${BOLD}${BLUE}=== Nesting Levels ===${NC}'; echo ''; ps e -o pid,cmd | grep -E 'claude|node' | grep -v grep | grep -E 'GOGENT_NESTING_LEVEL|node.*dist/index.js' | sed 's/GOGENT_NESTING_LEVEL=/LEVEL:/g' | head -20; echo ''; echo 'Press Ctrl+C to stop'; sleep 1; done" C-m

# Pane 3 (bottom-right): Spawn event log
tmux send-keys -t mcp-spawn-test:0.3 "clear" C-m
tmux send-keys -t mcp-spawn-test:0.3 "echo -e '${BOLD}${YELLOW}=== Spawn Event Log ===${NC}'" C-m
tmux send-keys -t mcp-spawn-test:0.3 "echo 'Watching for spawn events...'" C-m
tmux send-keys -t mcp-spawn-test:0.3 "echo ''" C-m
tmux send-keys -t mcp-spawn-test:0.3 "echo 'Events will appear here when spawn_agent is called'" C-m
tmux send-keys -t mcp-spawn-test:0.3 "echo ''" C-m
tmux send-keys -t mcp-spawn-test:0.3 "tail -f $TUI_OUTPUT 2>/dev/null | grep --line-buffered -E 'spawn|agent|MCP|tool' || echo 'Waiting for TUI output...'" C-m

echo -e "${GREEN}✓ Tmux session created${NC}"
echo ""

# Create helper scripts for common test prompts
cat > "$LOG_DIR/test-prompts.txt" <<'EOF'
=== MCP spawn_agent Test Prompts ===

TEST 1: Simple Single Agent Spawn
==================================
Please use the spawn_agent tool to spawn a codebase-search agent to find all
TypeScript files in packages/tui/src/mcp that contain the word "spawn".

Use these parameters:
- agent: "codebase-search"
- model: "haiku"
- description: "Find spawn references in MCP code"

Expected: See 1 additional process in Process Monitor, then it disappears


TEST 2: Nested Orchestrator (Complex)
======================================
I want to test nested agent spawning. Please use spawn_agent to spawn a
review-orchestrator that will review the MCP-SPAWN-009 implementation.

Files to review:
- packages/tui/src/mcp/server.ts
- packages/tui/src/index.tsx
- packages/tui/src/mcp/server.test.ts

Use:
- agent: "review-orchestrator"
- model: "sonnet"
- description: "Review MCP-SPAWN-009 implementation"

Expected: See up to 4 additional processes (orchestrator + 3 reviewers)


TEST 3: Simple Orchestrator Test
=================================
Use spawn_agent to spawn a general-purpose orchestrator agent. Give it this task:

"Analyze the spawn_agent implementation in packages/tui/src/mcp/tools/spawnAgent.ts
and create a 3-point summary of how it works."

agent: "orchestrator"
model: "sonnet"
description: "Analyze spawn_agent implementation"

Expected: See 2-3 processes as orchestrator may spawn helpers


TEST 4: Quick Verification
===========================
Use spawn_agent to spawn a haiku agent with this simple task:
"Count to 5 and return the result"

agent: "general-purpose"
model: "haiku"
description: "Quick test"
prompt: "AGENT: general-purpose\n\nCount from 1 to 5 and return the numbers."

Expected: Fast execution (<5 seconds), minimal cost (<$0.01)
EOF

cat > "$LOG_DIR/monitor-commands.sh" <<'EOF'
#!/bin/bash
# Helper commands for manual monitoring

echo "=== Monitoring Commands ==="
echo ""
echo "1. Count active Claude processes:"
echo "   ps aux | grep 'claude -p' | grep -v grep | wc -l"
echo ""
echo "2. Show all Claude processes with nesting levels:"
echo "   ps e -o pid,cmd | grep claude | grep GOGENT_NESTING_LEVEL"
echo ""
echo "3. Watch TUI output in real-time:"
echo "   tail -f $TUI_OUTPUT"
echo ""
echo "4. Check spawn registry (if exists):"
echo "   cat /tmp/gogent-spawn-registry.json 2>/dev/null | jq"
echo ""
echo "5. Kill all Claude processes (emergency):"
echo "   pkill -f 'claude -p'"
echo ""
EOF
chmod +x "$LOG_DIR/monitor-commands.sh"

# Create summary script
cat > "$LOG_DIR/analyze-session.sh" <<EOF
#!/bin/bash
# Analyze test session logs

echo "=== Session Analysis: $TIMESTAMP ==="
echo ""

echo "Process activity:"
if [ -f "$PROCESS_LOG" ]; then
    echo "  Max concurrent processes: \$(sort -n "$PROCESS_LOG" | tail -1)"
else
    echo "  No process log found"
fi

echo ""
echo "Spawn events:"
if [ -f "$SPAWN_EVENTS" ]; then
    grep -c "spawn" "$SPAWN_EVENTS" 2>/dev/null || echo "  0 spawn events"
else
    echo "  No event log found"
fi

echo ""
echo "TUI output summary:"
if [ -f "$TUI_OUTPUT" ]; then
    echo "  Lines: \$(wc -l < "$TUI_OUTPUT")"
    echo "  Errors: \$(grep -c -i error "$TUI_OUTPUT" || echo 0)"
else
    echo "  No TUI output found"
fi

echo ""
echo "Log files:"
ls -lh "$LOG_DIR"/*_$TIMESTAMP.*
EOF
chmod +x "$LOG_DIR/analyze-session.sh"

echo -e "${BOLD}${GREEN}=== Test Environment Ready ===${NC}"
echo ""
echo -e "${CYAN}Tmux session: ${BOLD}mcp-spawn-test${NC}"
echo -e "${CYAN}Log directory: ${BOLD}$LOG_DIR${NC}"
echo ""
echo -e "${YELLOW}Commands:${NC}"
echo -e "  ${BOLD}tmux attach -t mcp-spawn-test${NC}  - Attach to test session"
echo -e "  ${BOLD}tmux kill-session -t mcp-spawn-test${NC}  - Stop test session"
echo -e "  ${BOLD}cat $LOG_DIR/test-prompts.txt${NC}  - View test prompts"
echo -e "  ${BOLD}$LOG_DIR/analyze-session.sh${NC}  - Analyze results"
echo ""
echo -e "${YELLOW}Tmux controls (when attached):${NC}"
echo -e "  ${BOLD}Ctrl+B then arrow keys${NC}  - Navigate between panes"
echo -e "  ${BOLD}Ctrl+B then d${NC}  - Detach (keeps running)"
echo -e "  ${BOLD}Ctrl+C in TUI pane${NC}  - Stop TUI"
echo ""
echo -e "${GREEN}Starting tmux session now...${NC}"
echo ""

sleep 2

# Select the TUI pane (top-left) as active before attaching
tmux select-pane -t mcp-spawn-test:0.0

# Attach to session
tmux attach -t mcp-spawn-test
