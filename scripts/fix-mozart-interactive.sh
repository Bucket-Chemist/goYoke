#!/usr/bin/env bash
# Fix Mozart's agents-index.json entry so it can spawn children via MCP
set -euo pipefail

INDEX="$HOME/.claude/agents/agents-index.json"

echo "=== Fixing Mozart in agents-index.json ==="

# 1. Set interactive: true so buildSpawnArgs injects --mcp-config
jq '(.agents[] | select(.id == "mozart")).interactive = true' "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  Set interactive: true"

# 2. Expand cli_flags.allowed_tools to match the tools Mozart needs
# Mozart needs: Read, Glob, Grep (recon) + Write (problem-brief) + AskUserQuestion (interview)
# spawn_agent comes via MCP, not cli_flags
jq '(.agents[] | select(.id == "mozart")).cli_flags.allowed_tools = ["Read", "Glob", "Grep", "Write", "AskUserQuestion"]' "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  Updated cli_flags.allowed_tools"

# Validate
jq . "$INDEX" > /dev/null 2>&1 && echo "  JSON valid: OK" || echo "  JSON valid: FAIL"

echo ""
echo "=== Verification ==="
jq '.agents[] | select(.id == "mozart") | {id, interactive, cli_flags}' "$INDEX"

echo ""
echo "=== Also check: do Einstein, Beethoven, Staff-Architect need interactive? ==="
for agent in einstein beethoven staff-architect-critical-review; do
  interactive=$(jq -r --arg id "$agent" '.agents[] | select(.id == $id) | .interactive // "null"' "$INDEX")
  echo "  $agent: interactive=$interactive"
done
echo ""
echo "NOTE: Einstein/Beethoven/Staff-Architect do NOT need interactive."
echo "They are terminal agents — they analyze and write output, they don't spawn children."
echo "Only Mozart needs it because it orchestrates the braintrust workflow."
