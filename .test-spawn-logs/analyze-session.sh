#!/bin/bash
# Analyze test session logs

echo "=== Session Analysis: 20260205_081217 ==="
echo ""

echo "Process activity:"
if [ -f "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/processes_20260205_081217.log" ]; then
    echo "  Max concurrent processes: $(sort -n "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/processes_20260205_081217.log" | tail -1)"
else
    echo "  No process log found"
fi

echo ""
echo "Spawn events:"
if [ -f "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/spawn_events_20260205_081217.log" ]; then
    grep -c "spawn" "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/spawn_events_20260205_081217.log" 2>/dev/null || echo "  0 spawn events"
else
    echo "  No event log found"
fi

echo ""
echo "TUI output summary:"
if [ -f "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/tui_output_20260205_081217.log" ]; then
    echo "  Lines: $(wc -l < "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/tui_output_20260205_081217.log")"
    echo "  Errors: $(grep -c -i error "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs/tui_output_20260205_081217.log" || echo 0)"
else
    echo "  No TUI output found"
fi

echo ""
echo "Log files:"
ls -lh "/home/doktersmol/Documents/GOgent-Fortress/.test-spawn-logs"/*_20260205_081217.*
