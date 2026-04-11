#!/usr/bin/env bash
# Fix cli_flags.allowed_tools for all braintrust agents so they have the tools
# they actually need to complete their tasks.
#
# cli_flags.allowed_tools is what gets passed as --allowedTools to claude -p.
# The "tools" field in agents-index.json is documentation only.
set -euo pipefail

INDEX="$HOME/.claude/agents/agents-index.json"

echo "=== Fixing braintrust agent tools ==="

# Mozart: Read, Glob, Grep (recon) + Write (problem-brief, config files)
# spawn_agent + ask_user come via MCP (interactive: true already set)
jq '(.agents[] | select(.id == "mozart")).cli_flags.allowed_tools = ["Read", "Glob", "Grep", "Write"]' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  mozart: Read, Glob, Grep, Write (+ MCP spawn_agent/ask_user via interactive)"

# Einstein: Read, Glob, Grep (analysis) + Write (theoretical analysis output)
# Einstein can spawn scouts — needs interactive for MCP spawn_agent
jq '(.agents[] | select(.id == "einstein")).cli_flags.allowed_tools = ["Read", "Glob", "Grep", "Write"]' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
jq '(.agents[] | select(.id == "einstein")).interactive = true' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  einstein: Read, Glob, Grep, Write + interactive (can spawn scouts)"

# Beethoven: Read, Glob, Grep (read analyses) + Write (synthesis document)
# Terminal agent — no spawning, no interactive needed
jq '(.agents[] | select(.id == "beethoven")).cli_flags.allowed_tools = ["Read", "Glob", "Grep", "Write"]' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  beethoven: Read, Glob, Grep, Write (terminal, no interactive)"

# Staff-Architect: Read, Glob, Grep (review) + Write (critique + metadata)
# Can spawn llm-inference-architect — needs interactive for MCP spawn_agent
jq '(.agents[] | select(.id == "staff-architect-critical-review")).cli_flags.allowed_tools = ["Read", "Glob", "Grep", "Write"]' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
jq '(.agents[] | select(.id == "staff-architect-critical-review")).interactive = true' \
  "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  staff-architect-critical-review: Read, Glob, Grep, Write + interactive (can spawn llm-inference-architect)"

# Validate
jq . "$INDEX" > /dev/null 2>&1 && echo "  JSON valid: OK" || echo "  JSON valid: FAIL"

echo ""
echo "=== Verification ==="
for agent in mozart einstein beethoven staff-architect-critical-review; do
  echo "--- $agent ---"
  jq --arg id "$agent" '.agents[] | select(.id == $id) | {interactive, "allowed_tools": .cli_flags.allowed_tools}' "$INDEX"
done
