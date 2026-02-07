#!/bin/bash
# validate-cli-flags.sh — Validates cli_flags consistency in agents-index.json
# Run from project root: .claude/scripts/validate-cli-flags.sh
set -e

AGENTS_INDEX=".claude/agents/agents-index.json"

if [ ! -f "$AGENTS_INDEX" ]; then
  echo "ERROR: $AGENTS_INDEX not found"
  exit 1
fi

echo "=== cli_flags Validation ==="
errors=0

# 1. JSON validity
if ! python3 -c "import json; json.load(open('$AGENTS_INDEX'))" 2>/dev/null; then
  echo "FAIL: Invalid JSON"
  exit 1
fi
echo "OK: Valid JSON"

# 2. Count agents with/without cli_flags
total=$(jq '.agents | length' "$AGENTS_INDEX")
with_flags=$(jq '[.agents[] | select(.cli_flags)] | length' "$AGENTS_INDEX")
without_flags=$(jq -r '.agents[] | select(.cli_flags == null) | .id' "$AGENTS_INDEX")
echo "OK: $with_flags/$total agents have cli_flags"

# 3. Verify excluded agents are expected (bash-invoked only)
expected_excluded="gemini-slave gogent-scout"
for agent in $without_flags; do
  if ! echo "$expected_excluded" | grep -qw "$agent"; then
    echo "FAIL: Agent '$agent' missing cli_flags (not in expected exclusion list)"
    errors=$((errors + 1))
  fi
done
if [ $errors -eq 0 ]; then
  echo "OK: Excluded agents are expected: $without_flags"
fi

# 4. No reviewers have Bash
reviewers_with_bash=$(jq -r '.agents[] | select(.category == "review") | select(.cli_flags.allowed_tools | index("Bash")) | .id' "$AGENTS_INDEX")
if [ -n "$reviewers_with_bash" ]; then
  echo "FAIL: Reviewers with Bash in allowed_tools:"
  echo "  $reviewers_with_bash"
  errors=$((errors + 1))
else
  echo "OK: No reviewers have Bash"
fi

# 5. All cli_flags have additional_flags
missing_addl=$(jq -r '.agents[] | select(.cli_flags) | select(.cli_flags.additional_flags == null) | .id' "$AGENTS_INDEX")
if [ -n "$missing_addl" ]; then
  echo "FAIL: Agents with cli_flags but no additional_flags:"
  echo "  $missing_addl"
  errors=$((errors + 1))
else
  echo "OK: All cli_flags include additional_flags"
fi

# 6. Validate tool names are known CLI tools
known='["Read","Write","Edit","Glob","Grep","Bash","WebFetch","WebSearch"]'
unknown=$(jq -r --argjson known "$known" \
  '.agents[] | select(.cli_flags) | .id as $id | .cli_flags.allowed_tools[] | select(. as $t | $known | index($t) | not) | "\($id): \(.)"' \
  "$AGENTS_INDEX")
if [ -n "$unknown" ]; then
  echo "WARN: Unknown tool names in allowed_tools:"
  echo "  $unknown"
else
  echo "OK: All tool names are recognized CLI tools"
fi

# 7. Informational: tools vs cli_flags differences
echo ""
echo "=== tools vs cli_flags.allowed_tools differences (expected for team pattern) ==="
jq -r '.agents[] | select(.cli_flags and .tools) | select((.tools | sort) != (.cli_flags.allowed_tools | sort)) | "  \(.id): tools=\(.tools | sort | join(",")) cli=\(.cli_flags.allowed_tools | sort | join(","))"' "$AGENTS_INDEX"

echo ""
if [ $errors -gt 0 ]; then
  echo "FAILED: $errors error(s) found"
  exit 1
else
  echo "PASSED: All checks OK"
fi
