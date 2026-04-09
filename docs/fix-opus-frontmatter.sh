#!/usr/bin/env bash
# Fix opus agent frontmatter: prune invalid SDK fields, add effort: high
set -euo pipefail

AGENTS_DIR="$HOME/.claude/agents"

# --- 1. llm-inference-architect: prune invalid fields, fix subagent_type, simplify effort ---
FILE="$AGENTS_DIR/llm-inference-architect/llm-inference-architect.md"
echo "Fixing $FILE"

# Remove the effort block (lines with nested effort fields) and replace with flat effort: high
sed -i '/^effort:$/,/^[^ #]/{
  /^effort:$/d
  /^  /d
}' "$FILE"

# Remove max_tokens line
sed -i '/^max_tokens:/d' "$FILE"

# Remove context_window line
sed -i '/^context_window:/d' "$FILE"

# Remove the "# Opus 4.6 specific capabilities" comment and the 4 fields after it
sed -i '/^# Opus 4.6 specific capabilities$/d' "$FILE"
sed -i '/^interleaved_thinking:/d' "$FILE"
sed -i '/^compaction:/d' "$FILE"
sed -i '/^structured_outputs:/d' "$FILE"
sed -i '/^fast_mode:/d' "$FILE"

# Fix subagent_type
sed -i 's/^subagent_type: \["Plan", "Explore"\]/subagent_type: Analyst/' "$FILE"

# Add effort: high after model line (before tier)
sed -i '/^tier: 3$/i effort: high' "$FILE"

# Clean up any double blank lines left behind
sed -i '/^$/N;/^\n$/d' "$FILE"

echo "  Done: pruned invalid fields, fixed subagent_type, added effort: high"

# --- 2. Add effort: high to other opus agents that lack it ---
for agent in architect einstein mozart beethoven planner staff-architect-critical-review python-architect; do
  FILE="$AGENTS_DIR/$agent/$agent.md"
  if [ ! -f "$FILE" ]; then
    echo "SKIP: $FILE not found"
    continue
  fi

  # Check if effort already exists
  if grep -q '^effort:' "$FILE"; then
    echo "SKIP: $agent already has effort field"
    continue
  fi

  # Add effort: high after the model line
  sed -i '/^model: opus$/a effort: high' "$FILE"
  echo "Added effort: high to $agent"
done

echo ""
echo "=== Verification ==="
for agent in llm-inference-architect architect einstein mozart beethoven planner staff-architect-critical-review python-architect; do
  FILE="$AGENTS_DIR/$agent/$agent.md"
  echo "--- $agent ---"
  # Show first 25 lines of frontmatter
  sed -n '1,/^---$/{ /^---$/q; p }' "$FILE" | head -25
  echo "---"
  echo ""
done
