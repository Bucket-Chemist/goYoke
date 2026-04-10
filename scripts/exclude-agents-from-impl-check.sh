#!/usr/bin/env bash
# Add "agents/*" to excluded_patterns in routing-schema.json
# so gogent-direct-impl-check stops warning on agent definition files.

set -euo pipefail

SCHEMA="/home/doktersmol/Documents/GOgent-Fortress/.claude/routing-schema.json"

if ! command -v jq &>/dev/null; then
  echo "ERROR: jq not found" >&2
  exit 1
fi

# Check if already present
if jq -e '.direct_impl_check.excluded_patterns | index("agents/*")' "$SCHEMA" &>/dev/null; then
  echo "agents/* already in excluded_patterns, nothing to do."
  exit 0
fi

# Add it
tmp=$(mktemp)
jq '.direct_impl_check.excluded_patterns += ["agents/*"]' "$SCHEMA" > "$tmp"
mv "$tmp" "$SCHEMA"

echo "Done. Added agents/* to excluded_patterns:"
jq '.direct_impl_check.excluded_patterns' "$SCHEMA"
