#!/usr/bin/env bash
# Full agent alignment: id + subagent_type across frontmatter, agents-index.json, routing-schema.json
set -euo pipefail

AGENTS_DIR="$HOME/.claude/agents"
INDEX="$AGENTS_DIR/agents-index.json"
SCHEMA="$HOME/.claude/routing-schema.json"

echo "=== Phase 1: Add missing agents to routing-schema.json agent_subagent_mapping ==="

# Add architect-reviewer and gogent-scout to agent_subagent_mapping
jq '.agent_subagent_mapping += {
  "architect-reviewer": "Architect Reviewer",
  "gogent-scout": "GOgent Smart Scout"
}' "$SCHEMA" > "$SCHEMA.tmp" && mv "$SCHEMA.tmp" "$SCHEMA"
echo "  Added architect-reviewer + gogent-scout to routing-schema mapping"

echo ""
echo "=== Phase 2: Populate subagent_type in agents-index.json from routing-schema ==="

# Build the mapping from routing-schema, then apply to each agent in index
# jq: for each agent, look up its id in the mapping and set subagent_type
jq --argjson mapping "$(jq '.agent_subagent_mapping | del(.description)' "$SCHEMA")" '
  .agents |= map(
    if $mapping[.id] then
      .subagent_type = $mapping[.id]
    else
      .
    end
  )
' "$INDEX" > "$INDEX.tmp" && mv "$INDEX.tmp" "$INDEX"
echo "  Updated subagent_type for all agents in agents-index.json"

echo ""
echo "=== Phase 3: Add/fix id + subagent_type in all agent frontmatter ==="

# Build associative array of agent -> subagent_type from routing-schema
declare -A SAT_MAP
while IFS=$'\t' read -r agent_id sat; do
  SAT_MAP["$agent_id"]="$sat"
done < <(jq -r '.agent_subagent_mapping | del(.description) | to_entries[] | [.key, .value] | @tsv' "$SCHEMA")

for dir in "$AGENTS_DIR"/*/; do
  agent=$(basename "$dir")
  file="$dir/$agent.md"

  # Skip dirs without a matching .md file
  if [ ! -f "$file" ]; then
    echo "  SKIP: $file not found"
    continue
  fi

  # Skip test/orphan agents not in mapping
  if [ -z "${SAT_MAP[$agent]+x}" ]; then
    echo "  SKIP: $agent not in routing-schema mapping (orphan/test)"
    continue
  fi

  expected_sat="${SAT_MAP[$agent]}"
  changed=false

  # --- Fix id ---
  current_id=$(awk '/^---$/{n++; next} n==1 && /^id:/{print $2; exit}' "$file")
  if [ -z "$current_id" ]; then
    # Add id as first field after opening ---
    sed -i "0,/^---$/!{0,/^---$/!b; :a; /^---$/{a\\id: $agent
b}; n; ba}" "$file"
    # Simpler approach: add after first ---
    sed -i "1,/^---$/{/^---$/{a\\id: $agent
}}" "$file"
    # Verify it was added
    verify_id=$(awk '/^---$/{n++; next} n==1 && /^id:/{print $2; exit}' "$file")
    if [ "$verify_id" = "$agent" ]; then
      echo "  $agent: added id"
      changed=true
    else
      echo "  $agent: WARN - id insertion may have failed, trying alternate method"
      # Alternate: use line-based insertion after first ---
      first_dash=$(grep -n '^---$' "$file" | head -1 | cut -d: -f1)
      if [ -n "$first_dash" ]; then
        sed -i "${first_dash}a\\id: $agent" "$file"
        echo "  $agent: added id (alternate method)"
        changed=true
      fi
    fi
  elif [ "$current_id" != "$agent" ]; then
    sed -i "s/^id: .*/id: $agent/" "$file"
    echo "  $agent: fixed id ($current_id -> $agent)"
    changed=true
  fi

  # --- Fix subagent_type ---
  # Get current value (raw line after "subagent_type:")
  current_sat=$(awk '/^---$/{n++; next} n==1 && /^subagent_type:/{$1=""; gsub(/^ +/,""); print; exit}' "$file")

  if [ -z "$current_sat" ]; then
    # Find the line with "tier:" or "category:" or "triggers:" to insert before
    # Prefer inserting after category if it exists, otherwise after tier
    insert_after=$(grep -n '^category:' "$file" | head -1 | cut -d: -f1)
    if [ -z "$insert_after" ]; then
      insert_after=$(grep -n '^tier:' "$file" | head -1 | cut -d: -f1)
    fi
    if [ -n "$insert_after" ]; then
      sed -i "${insert_after}a\\subagent_type: $expected_sat" "$file"
      echo "  $agent: added subagent_type: $expected_sat"
      changed=true
    else
      # Fallback: add after model line
      model_line=$(grep -n '^model:' "$file" | head -1 | cut -d: -f1)
      if [ -n "$model_line" ]; then
        sed -i "${model_line}a\\subagent_type: $expected_sat" "$file"
        echo "  $agent: added subagent_type: $expected_sat (after model)"
        changed=true
      else
        echo "  $agent: WARN - could not find insertion point for subagent_type"
      fi
    fi
  elif [ "$current_sat" != "$expected_sat" ]; then
    # Replace the subagent_type line (handles arrays, comments, etc.)
    sed -i "s|^subagent_type:.*|subagent_type: $expected_sat|" "$file"
    echo "  $agent: fixed subagent_type ($current_sat -> $expected_sat)"
    changed=true
  fi

  if [ "$changed" = false ]; then
    echo "  $agent: OK (no changes needed)"
  fi
done

echo ""
echo "=== Phase 4: Verification ==="

echo ""
echo "--- agents-index.json: agents with NULL subagent_type ---"
null_count=$(jq '[.agents[] | select(.subagent_type == null or .subagent_type == "")] | length' "$INDEX")
if [ "$null_count" -eq 0 ]; then
  echo "  PASS: All agents have subagent_type"
else
  echo "  FAIL: $null_count agents still have NULL subagent_type:"
  jq -r '.agents[] | select(.subagent_type == null or .subagent_type == "") | "    " + .id' "$INDEX"
fi

echo ""
echo "--- agents-index.json: subagent_type matches routing-schema ---"
mismatch=0
while IFS=$'\t' read -r agent_id idx_sat; do
  schema_sat=$(jq -r --arg id "$agent_id" '.agent_subagent_mapping[$id] // "NOT_IN_SCHEMA"' "$SCHEMA")
  if [ "$schema_sat" = "NOT_IN_SCHEMA" ]; then
    echo "  WARN: $agent_id not in routing-schema mapping"
  elif [ "$idx_sat" != "$schema_sat" ]; then
    echo "  MISMATCH: $agent_id index='$idx_sat' schema='$schema_sat'"
    mismatch=$((mismatch + 1))
  fi
done < <(jq -r '.agents[] | [.id, (.subagent_type // "NULL")] | @tsv' "$INDEX")
if [ "$mismatch" -eq 0 ]; then
  echo "  PASS: All subagent_types match between index and schema"
fi

echo ""
echo "--- frontmatter: id + subagent_type spot check ---"
fm_issues=0
for dir in "$AGENTS_DIR"/*/; do
  agent=$(basename "$dir")
  file="$dir/$agent.md"
  [ ! -f "$file" ] && continue
  [ -z "${SAT_MAP[$agent]+x}" ] && continue

  fm_id=$(awk '/^---$/{n++; next} n==1 && /^id:/{print $2; exit}' "$file")
  fm_sat=$(awk '/^---$/{n++; next} n==1 && /^subagent_type:/{$1=""; gsub(/^ +/,""); print; exit}' "$file")

  issues=""
  [ "$fm_id" != "$agent" ] && issues="id=$fm_id(want $agent) "
  [ "$fm_sat" != "${SAT_MAP[$agent]}" ] && issues="${issues}sat=$fm_sat(want ${SAT_MAP[$agent]})"

  if [ -n "$issues" ]; then
    echo "  FAIL: $agent - $issues"
    fm_issues=$((fm_issues + 1))
  fi
done
if [ "$fm_issues" -eq 0 ]; then
  echo "  PASS: All frontmatter id + subagent_type aligned"
fi

echo ""
echo "--- JSON validity ---"
jq . "$INDEX" > /dev/null 2>&1 && echo "  PASS: agents-index.json valid JSON" || echo "  FAIL: agents-index.json invalid JSON"
jq . "$SCHEMA" > /dev/null 2>&1 && echo "  PASS: routing-schema.json valid JSON" || echo "  FAIL: routing-schema.json invalid JSON"

echo ""
echo "--- Orphan check ---"
echo "  restricted-test: orphan dir (no index entry) - consider removing"
echo "  test-frontmatter: orphan dir (no index entry) - consider removing"
echo "  gogent-scout: in index, no frontmatter dir (native binary, expected)"

echo ""
echo "=== Done ==="
