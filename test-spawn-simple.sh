#!/bin/bash
# test-spawn-simple.sh - Simple MCP spawn testing with separate terminals

set -eo pipefail

PROJECT_ROOT="$HOME/Documents/GOgent-Fortress"
TUI_DIR="$PROJECT_ROOT/packages/tui"

echo "=== Simple MCP Spawn Test Setup ==="
echo ""

# Cleanup
echo "Cleaning up..."
tmux kill-session -t mcp-spawn-test 2>/dev/null || true
pkill -f "claude -p" 2>/dev/null || true
echo "✓ Cleanup complete"
echo ""

# Check terminal emulator
if command -v gnome-terminal &> /dev/null; then
    TERM_CMD="gnome-terminal"
elif command -v konsole &> /dev/null; then
    TERM_CMD="konsole"
elif command -v xterm &> /dev/null; then
    TERM_CMD="xterm"
elif command -v alacritty &> /dev/null; then
    TERM_CMD="alacritty"
elif command -v kitty &> /dev/null; then
    TERM_CMD="kitty"
else
    echo "No supported terminal emulator found."
    echo "Falling back to tmux with simpler layout..."
    TERM_CMD="tmux"
fi

if [ "$TERM_CMD" = "tmux" ]; then
    # Fallback: Use tmux with 2 panes side-by-side
    echo "Using tmux fallback..."
    tmux new-session -d -s mcp-spawn-test

    # Split horizontally (side by side)
    tmux split-window -h -t mcp-spawn-test:0

    # Left pane: TUI (80% width)
    tmux resize-pane -t mcp-spawn-test:0.0 -x 120

    # Left: Start TUI
    tmux send-keys -t mcp-spawn-test:0.0 "cd $TUI_DIR && clear && echo 'Starting TUI...' && npm start" C-m

    # Right: Process monitor
    tmux send-keys -t mcp-spawn-test:0.1 "clear && echo '=== Process Monitor ===' && echo '' && echo 'Waiting for TUI to start...' && sleep 3" C-m
    tmux send-keys -t mcp-spawn-test:0.1 "watch -n 0.5 'date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi'" C-m

    # Focus TUI pane
    tmux select-pane -t mcp-spawn-test:0.0

    echo ""
    echo "✓ Tmux session created"
    echo ""
    echo "Attaching to session..."
    sleep 2

    tmux attach -t mcp-spawn-test
else
    # Use native terminal emulator
    echo "Using $TERM_CMD..."

    # Start process monitor in new terminal
    if [ "$TERM_CMD" = "gnome-terminal" ]; then
        gnome-terminal -- bash -c 'watch -n 0.5 "date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi"' &
    elif [ "$TERM_CMD" = "konsole" ]; then
        konsole -e bash -c 'watch -n 0.5 "date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi"' &
    elif [ "$TERM_CMD" = "alacritty" ]; then
        alacritty -e bash -c 'watch -n 0.5 "date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi"' &
    elif [ "$TERM_CMD" = "kitty" ]; then
        kitty bash -c 'watch -n 0.5 "date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi"' &
    else
        xterm -e bash -c 'watch -n 0.5 "date \"+%H:%M:%S\"; echo \"\"; echo \"Spawned Claude processes:\"; count=\$(ps aux | grep \"claude -p\" | grep -v grep | wc -l); echo \"\$count\"; echo \"\"; if [ \$count -gt 0 ]; then echo \"Active spawns:\"; ps aux | grep \"claude -p\" | grep -v grep | awk \"{print \\\$2, \\\$11}\" | head -5; fi"' &
    fi

    sleep 1
    echo "✓ Process monitor started in new terminal"
    echo ""

    # Start TUI in current terminal
    echo "Starting TUI in this terminal..."
    echo ""
    sleep 2
    cd "$TUI_DIR"
    npm start
fi
