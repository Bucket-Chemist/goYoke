#!/bin/bash
# Check corpus capture progress for GOgent-008b

set -euo pipefail

# Determine corpus location (XDG-compliant)
CORPUS_RAW=""
if [[ -n "${XDG_RUNTIME_DIR:-}" && -f "$XDG_RUNTIME_DIR/gogent/event-corpus-raw.jsonl" ]]; then
    CORPUS_RAW="$XDG_RUNTIME_DIR/gogent/event-corpus-raw.jsonl"
elif [[ -n "${XDG_CACHE_HOME:-}" && -f "$XDG_CACHE_HOME/gogent/event-corpus-raw.jsonl" ]]; then
    CORPUS_RAW="$XDG_CACHE_HOME/gogent/event-corpus-raw.jsonl"
elif [[ -f "$HOME/.cache/gogent/event-corpus-raw.jsonl" ]]; then
    CORPUS_RAW="$HOME/.cache/gogent/event-corpus-raw.jsonl"
else
    echo "❌ No corpus file found in XDG locations"
    echo ""
    echo "Expected locations:"
    echo "  - \$XDG_RUNTIME_DIR/gogent/event-corpus-raw.jsonl"
    echo "  - \$XDG_CACHE_HOME/gogent/event-corpus-raw.jsonl"
    echo "  - ~/.cache/gogent/event-corpus-raw.jsonl"
    echo ""
    echo "The corpus logger may not have captured any events yet."
    echo "Use Claude Code normally and check again later."
    exit 1
fi

echo "📊 Corpus Capture Progress (GOgent-008b)"
echo "======================================="
echo ""
echo "Corpus file: $CORPUS_RAW"
echo ""

# Count total events
TOTAL_EVENTS=$(wc -l < "$CORPUS_RAW")
echo "Total events captured: $TOTAL_EVENTS"

# Calculate progress toward goal
TARGET=95
if (( TOTAL_EVENTS >= TARGET )); then
    echo "✅ Target met! ($TOTAL_EVENTS/$TARGET)"
    echo ""
    echo "Ready to curate corpus. Run:"
    echo "  ./scripts/curate-corpus.sh"
else
    REMAINING=$((TARGET - TOTAL_EVENTS))
    PERCENT=$((TOTAL_EVENTS * 100 / TARGET))
    echo "⏳ Progress: $PERCENT% ($TOTAL_EVENTS/$TARGET)"
    echo "   Need $REMAINING more events"
    echo ""
    echo "Continue using Claude Code normally."
    echo "Check progress again later with:"
    echo "  ./scripts/check-corpus-progress.sh"
fi

echo ""
echo "Last 3 events:"
echo "-------------"
tail -3 "$CORPUS_RAW" | jq -c '{tool_name, hook_event_name, captured_at}'

echo ""
echo "File size: $(ls -lh "$CORPUS_RAW" | awk '{print $5}')"
