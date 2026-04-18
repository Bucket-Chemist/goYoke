#!/usr/bin/env bash
# Add routing-schema.json and agents-index.json to excluded_patterns
# in routing-schema.json so the router can edit these config files.

set -euo pipefail

SCHEMA="/home/doktersmol/Documents/goYoke/.claude/routing-schema.json"

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq not found" >&2
  exit 1
fi

for pattern in "routing-schema.json" "agents-index.json"; do
  if jq -e --arg p "$pattern" '.direct_impl_check.excluded_patterns | index($p)' "$SCHEMA" &>/dev/null; then
    echo "$pattern already in excluded_patterns, skipping."
  else
    tmp=$(mktemp)
    jq --arg p "$pattern" '.direct_impl_check.excluded_patterns += [$p]' "$SCHEMA" > "$tmp"
    mv "$tmp" "$SCHEMA"
    echo "Added $pattern"
  fi
done

echo ""
echo "Current excluded_patterns:"
jq '.direct_impl_check.excluded_patterns' "$SCHEMA"
