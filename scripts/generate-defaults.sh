#!/usr/bin/env bash
# generate-defaults.sh — Populate defaults/ from .claude/ with distribution filtering
#
# Dependencies: jq (https://jqlang.github.io/jq/)
# Usage: ./scripts/generate-defaults.sh
#
# Reads distribution flags from .claude/agents/agents-index.json and copies
# only public content to defaults/. Private bioinformatics agents and
# domain-specific conventions are excluded.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SOURCE="${PROJECT_ROOT}/.claude"
DEST="${PROJECT_ROOT}/defaults"
AGENTS_INDEX="${SOURCE}/agents/agents-index.json"

# Verify jq is available
if ! command -v jq &>/dev/null; then
    echo "ERROR: jq is required. Install with: sudo pacman -S jq" >&2
    exit 1
fi

# Verify source exists
if [[ ! -f "$AGENTS_INDEX" ]]; then
    echo "ERROR: agents-index.json not found at $AGENTS_INDEX" >&2
    exit 1
fi

echo "[generate-defaults] Starting..."

# Clean destination (idempotent)
rm -rf "${DEST}/agents" "${DEST}/conventions" "${DEST}/rules" "${DEST}/schemas" "${DEST}/skills"
mkdir -p "${DEST}/agents" "${DEST}/conventions" "${DEST}/rules" "${DEST}/schemas" "${DEST}/skills"

# --- 1. Filter agents-index.json ---
echo "[generate-defaults] Filtering agents-index.json..."
# Get private agent IDs, private conventions, private skills, private schema prefixes
PRIVATE_AGENTS=$(jq -r '.agents[] | select(.distribution == "private") | .id' "$AGENTS_INDEX")
PRIVATE_CONVENTIONS=$(jq -r '.distribution.private_conventions[]' "$AGENTS_INDEX" 2>/dev/null || true)
PRIVATE_SKILLS=$(jq -r '.distribution.private_skills[]' "$AGENTS_INDEX" 2>/dev/null || true)
PRIVATE_SCHEMA_PREFIXES=$(jq -r '.distribution.private_schema_prefixes[]' "$AGENTS_INDEX" 2>/dev/null || true)

# Filter agents-index.json: remove private agents, remove distribution metadata
jq '
  del(.distribution) |
  .agents = [.agents[] | select(.distribution != "private") | del(.distribution)]
' "$AGENTS_INDEX" > "${DEST}/agents/agents-index.json"

# --- 2. Copy public agent directories ---
echo "[generate-defaults] Copying public agent directories..."
for agent_dir in "${SOURCE}/agents"/*/; do
    agent_name=$(basename "$agent_dir")
    # Skip if private
    if echo "$PRIVATE_AGENTS" | grep -qx "$agent_name"; then
        continue
    fi
    # Skip non-directories and hidden/underscore dirs
    if [[ "$agent_name" == .* ]] || [[ "$agent_name" == _* ]]; then
        continue
    fi
    # Copy entire agent directory (md + sharp-edges.yaml + references/)
    cp -r "$agent_dir" "${DEST}/agents/${agent_name}"
done

# --- 3. Copy conventions (excluding private) ---
echo "[generate-defaults] Copying conventions..."
for conv in "${SOURCE}/conventions"/*.md; do
    [[ -f "$conv" ]] || continue
    conv_name=$(basename "$conv")
    if echo "$PRIVATE_CONVENTIONS" | grep -qx "$conv_name"; then
        continue
    fi
    cp "$conv" "${DEST}/conventions/${conv_name}"
done

# --- 4. Copy rules ---
echo "[generate-defaults] Copying rules..."
cp "${SOURCE}/rules"/*.md "${DEST}/rules/" 2>/dev/null || true

# --- 5. Copy schemas (excluding private prefixes) ---
echo "[generate-defaults] Copying schemas..."
# Copy schema subdirectories, filtering by prefix
for schema_dir in "${SOURCE}/schemas"/*/; do
    [[ -d "$schema_dir" ]] || continue
    schema_name=$(basename "$schema_dir")
    skip=false
    for prefix in $PRIVATE_SCHEMA_PREFIXES; do
        if [[ "$schema_name" == "$prefix"* ]]; then
            skip=true
            break
        fi
    done
    if [[ "$skip" == "true" ]]; then
        continue
    fi
    cp -r "$schema_dir" "${DEST}/schemas/${schema_name}"
done
# Also copy any top-level schema files
for schema_file in "${SOURCE}/schemas"/*.json; do
    [[ -f "$schema_file" ]] && cp "$schema_file" "${DEST}/schemas/" || true
done

# --- 6. Copy skills (excluding private) ---
echo "[generate-defaults] Copying skills..."
for skill_dir in "${SOURCE}/skills"/*/; do
    [[ -d "$skill_dir" ]] || continue
    skill_name=$(basename "$skill_dir")
    if echo "$PRIVATE_SKILLS" | grep -qx "$skill_name"; then
        continue
    fi
    if [[ "$skill_name" == .* ]] || [[ "$skill_name" == _* ]]; then
        continue
    fi
    mkdir -p "${DEST}/skills/${skill_name}"
    # Copy SKILL.md only (not scripts or other internal files)
    if [[ -f "${skill_dir}/SKILL.md" ]]; then
        cp "${skill_dir}/SKILL.md" "${DEST}/skills/${skill_name}/"
    fi
done

# --- 7. Copy root-level files ---
echo "[generate-defaults] Copying root files..."
cp "${SOURCE}/routing-schema.json" "${DEST}/routing-schema.json" 2>/dev/null || true
# CLAUDE.md: embed the real one so the binary has full routing knowledge
cp "${SOURCE}/CLAUDE.md" "${DEST}/CLAUDE.md"
# settings-template.json: extract hooks, convert to multicall format (goyoke hook <name>)
jq '{hooks: .hooks}' "${SOURCE}/settings.json" | \
    sed 's|/[^"]*bin/\(goyoke-[^"]*\)|\1|g' | \
    sed 's|"goyoke-\([^"]*\)"|"goyoke hook \1"|g' > "${DEST}/settings-template.json"

# --- 8. Post-copy cleanup ---
# Remove dotfiles copied transitively from source (not allowed in distribution)
find "${DEST}" -name ".*" -not -name ".gitkeep" -not -path "${DEST}" -exec rm -rf {} + 2>/dev/null || true

# --- 9. Post-copy validation ---
echo "[generate-defaults] Validating..."
DOTFILES=$(find "${DEST}" -name ".*" -not -name ".gitkeep" -not -path "${DEST}" 2>/dev/null)
if [[ -n "$DOTFILES" ]]; then
    echo "ERROR: Dotfiles found in defaults/ (not allowed):" >&2
    echo "$DOTFILES" >&2
    exit 1
fi

# Count results
AGENT_COUNT=$(ls -d "${DEST}/agents"/*/ 2>/dev/null | wc -l || echo 0)
CONV_COUNT=$(ls "${DEST}/conventions"/*.md 2>/dev/null | wc -l || echo 0)
SKILL_COUNT=$(ls -d "${DEST}/skills"/*/ 2>/dev/null | wc -l || echo 0)

echo "[generate-defaults] Done!"
echo "  Agents: $AGENT_COUNT"
echo "  Conventions: $CONV_COUNT"
echo "  Skills: $SKILL_COUNT"
