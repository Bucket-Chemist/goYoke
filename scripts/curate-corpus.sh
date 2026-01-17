#!/bin/bash
# Curate event corpus for GOgent-008b
# Converts raw JSONL to structured JSON array with validation

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
    exit 1
fi

CORPUS_OUTPUT="test/fixtures/event-corpus.json"

echo "🔄 Curating Event Corpus"
echo "========================"
echo ""
echo "Source: $CORPUS_RAW"
echo "Target: $CORPUS_OUTPUT"
echo ""

# Count raw events
RAW_COUNT=$(wc -l < "$CORPUS_RAW")
echo "Raw events: $RAW_COUNT"

# Filter and curate:
# - Remove null/invalid entries
# - Keep only events with tool_name (valid tool events)
# - Convert JSONL to JSON array
cat "$CORPUS_RAW" \
  | jq -s '[.[] | select(.tool_name != null and .tool_name != "")]' \
  > "$CORPUS_OUTPUT"

# Validate output
if ! jq empty "$CORPUS_OUTPUT" 2>/dev/null; then
    echo "❌ Generated corpus is not valid JSON"
    exit 1
fi

CURATED_COUNT=$(jq 'length' "$CORPUS_OUTPUT")
echo "Curated events: $CURATED_COUNT"

# Check if target met
TARGET=95
if (( CURATED_COUNT >= TARGET )); then
    echo "✅ Target met! ($CURATED_COUNT/$TARGET)"
else
    REMAINING=$((TARGET - CURATED_COUNT))
    echo "⚠️  Short of target: $CURATED_COUNT/$TARGET"
    echo "   Need $REMAINING more events"
    echo ""
    echo "Continue capturing and re-run this script."
    exit 1
fi

echo ""
echo "📋 Sample Events"
echo "==============="
jq '.[0:3]' "$CORPUS_OUTPUT"

echo ""
echo "📊 Event Type Distribution"
echo "========================="
jq -r '.[] | .hook_event_name' "$CORPUS_OUTPUT" | sort | uniq -c | sort -rn

echo ""
echo "📊 Tool Name Distribution"
echo "========================"
jq -r '.[] | .tool_name' "$CORPUS_OUTPUT" | sort | uniq -c | sort -rn | head -10

echo ""
echo "✅ Corpus curation complete!"
echo ""
echo "Next steps:"
echo "  1. Review: jq '.' $CORPUS_OUTPUT | less"
echo "  2. Commit: git add $CORPUS_OUTPUT"
echo "  3. Proceed with GOgent-006, GOgent-007, GOgent-008, GOgent-009"
