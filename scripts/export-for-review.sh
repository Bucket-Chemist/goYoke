#!/bin/bash
# GOgent-Fortress Framework Export for Deep Research Review
# Exports all configuration, agents, skills, conventions for external analysis

set -euo pipefail

OUTPUT_DIR="${1:-./framework-export-$(date +%Y%m%d-%H%M%S)}"
CLAUDE_DIR="${HOME}/.claude"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== GOgent-Fortress Framework Export ==="
echo "Output: ${OUTPUT_DIR}"
echo ""

mkdir -p "${OUTPUT_DIR}"/{core,agents,skills,conventions,rules,schemas,docs}

# 1. Core Configuration
echo "[1/8] Exporting core configuration..."
cp "${CLAUDE_DIR}/CLAUDE.md" "${OUTPUT_DIR}/core/" 2>/dev/null || echo "  - CLAUDE.md not found"
cp "${CLAUDE_DIR}/routing-schema.json" "${OUTPUT_DIR}/core/" 2>/dev/null || echo "  - routing-schema.json not found"

# 2. Agents Index
echo "[2/8] Exporting agents index..."
cp "${CLAUDE_DIR}/agents/agents-index.json" "${OUTPUT_DIR}/agents/" 2>/dev/null || echo "  - agents-index.json not found"

# 3. Individual Agent Configs
echo "[3/8] Exporting agent configurations..."
for agent_dir in "${CLAUDE_DIR}/agents"/*/; do
    if [[ -d "$agent_dir" ]]; then
        agent_name=$(basename "$agent_dir")
        mkdir -p "${OUTPUT_DIR}/agents/${agent_name}"

        # Copy all agent files
        for file in agent.yaml agent.md CLAUDE.md sharp-edges.yaml; do
            [[ -f "${agent_dir}${file}" ]] && cp "${agent_dir}${file}" "${OUTPUT_DIR}/agents/${agent_name}/"
        done
    fi
done
echo "  - Exported $(ls -d "${OUTPUT_DIR}/agents"/*/ 2>/dev/null | wc -l) agents"

# 4. Skills
echo "[4/8] Exporting skills..."
for skill_dir in "${CLAUDE_DIR}/skills"/*/; do
    if [[ -d "$skill_dir" ]]; then
        skill_name=$(basename "$skill_dir")
        mkdir -p "${OUTPUT_DIR}/skills/${skill_name}"

        for file in skill.yaml SKILL.md; do
            [[ -f "${skill_dir}${file}" ]] && cp "${skill_dir}${file}" "${OUTPUT_DIR}/skills/${skill_name}/"
        done
    fi
done
echo "  - Exported $(ls -d "${OUTPUT_DIR}/skills"/*/ 2>/dev/null | wc -l) skills"

# 5. Conventions
echo "[5/8] Exporting conventions..."
cp "${CLAUDE_DIR}/conventions"/*.md "${OUTPUT_DIR}/conventions/" 2>/dev/null || echo "  - No conventions found"
echo "  - Exported $(ls "${OUTPUT_DIR}/conventions"/*.md 2>/dev/null | wc -l) convention files"

# 6. Rules
echo "[6/8] Exporting rules..."
cp "${CLAUDE_DIR}/rules"/*.md "${OUTPUT_DIR}/rules/" 2>/dev/null || echo "  - No rules found"
echo "  - Exported $(ls "${OUTPUT_DIR}/rules"/*.md 2>/dev/null | wc -l) rule files"

# 7. Go Schema Definitions
echo "[7/8] Exporting Go schema definitions..."
cp "${PROJECT_DIR}/pkg/routing/schema.go" "${OUTPUT_DIR}/schemas/" 2>/dev/null || echo "  - schema.go not found"
cp "${PROJECT_DIR}/pkg/routing/agents.go" "${OUTPUT_DIR}/schemas/" 2>/dev/null || true

# 8. Architecture Documentation
echo "[8/8] Exporting architecture docs..."
cp "${PROJECT_DIR}/docs/ARCHITECTURE.md" "${OUTPUT_DIR}/docs/" 2>/dev/null || echo "  - ARCHITECTURE.md not found"
cp "${PROJECT_DIR}/docs/systems-architecture-overview.md" "${OUTPUT_DIR}/docs/" 2>/dev/null || true

# Generate manifest
echo ""
echo "=== Generating manifest ==="
cat > "${OUTPUT_DIR}/MANIFEST.md" << 'MANIFEST'
# GOgent-Fortress Framework Export

**Generated:** $(date -Iseconds)
**Purpose:** Deep research review of agent framework configuration

## Directory Structure

```
framework-export/
├── core/                    # Core configuration
│   ├── CLAUDE.md           # Router identity & dispatch tables
│   └── routing-schema.json # Tier definitions, agent mappings
├── agents/                  # Agent definitions
│   ├── agents-index.json   # Agent registry with metadata
│   └── {agent-name}/       # Per-agent configuration
│       ├── agent.yaml      # Model, tier, tools, triggers
│       ├── agent.md        # Detailed instructions
│       ├── CLAUDE.md       # Agent-specific context
│       └── sharp-edges.yaml# Known pitfalls
├── skills/                  # Skill workflows
│   └── {skill-name}/
│       ├── skill.yaml      # Skill metadata
│       └── SKILL.md        # Workflow definition
├── conventions/             # Language conventions
│   └── *.md                # go.md, python.md, etc.
├── rules/                   # Behavioral rules
│   ├── LLM-guidelines.md   # Cross-cutting guidelines
│   └── agent-behavior.md   # Agent behavioral rules
├── schemas/                 # Go type definitions
│   └── schema.go           # Routing schema structs
└── docs/                    # Architecture documentation
    └── ARCHITECTURE.md     # System architecture v1.2
```

## Review Focus Areas

1. **Agent Scope**: Are agents appropriately scoped? Too broad? Too narrow?
2. **Trigger Overlap**: Do agent triggers conflict or create ambiguity?
3. **Tier Alignment**: Are agents assigned to appropriate cost tiers?
4. **Sharp Edges**: Are known pitfalls well-documented?
5. **Convention Consistency**: Do conventions align across languages?
6. **Skill Completeness**: Are workflows well-defined and complete?

## Key Files for Review

| Priority | File | Purpose |
|----------|------|---------|
| P0 | `core/routing-schema.json` | Source of truth for tiers |
| P0 | `agents/agents-index.json` | Agent registry |
| P1 | `rules/LLM-guidelines.md` | Multi-model strategy |
| P1 | `rules/agent-behavior.md` | Behavioral guidelines |
| P2 | `agents/*/agent.yaml` | Individual agent configs |
| P2 | `agents/*/sharp-edges.yaml` | Known pitfalls |
| P3 | `conventions/*.md` | Language conventions |
MANIFEST

# Add actual timestamp
sed -i "s/\$(date -Iseconds)/$(date -Iseconds)/" "${OUTPUT_DIR}/MANIFEST.md"

# Create combined single-file export for easy LLM consumption
echo ""
echo "=== Creating combined export file ==="
COMBINED="${OUTPUT_DIR}/COMBINED-EXPORT.md"

cat > "${COMBINED}" << 'HEADER'
# GOgent-Fortress Framework - Combined Export

This file contains all framework configuration for deep research review.

---

HEADER

echo "## Core Configuration" >> "${COMBINED}"
echo "" >> "${COMBINED}"

echo "### CLAUDE.md (Router Identity)" >> "${COMBINED}"
echo '```markdown' >> "${COMBINED}"
cat "${OUTPUT_DIR}/core/CLAUDE.md" >> "${COMBINED}" 2>/dev/null || echo "(not found)"
echo '```' >> "${COMBINED}"
echo "" >> "${COMBINED}"

echo "### routing-schema.json" >> "${COMBINED}"
echo '```json' >> "${COMBINED}"
cat "${OUTPUT_DIR}/core/routing-schema.json" >> "${COMBINED}" 2>/dev/null || echo "(not found)"
echo '```' >> "${COMBINED}"
echo "" >> "${COMBINED}"

echo "## Agents Index" >> "${COMBINED}"
echo '```json' >> "${COMBINED}"
cat "${OUTPUT_DIR}/agents/agents-index.json" >> "${COMBINED}" 2>/dev/null || echo "(not found)"
echo '```' >> "${COMBINED}"
echo "" >> "${COMBINED}"

echo "## Rules" >> "${COMBINED}"
for rule in "${OUTPUT_DIR}/rules"/*.md; do
    if [[ -f "$rule" ]]; then
        echo "### $(basename "$rule")" >> "${COMBINED}"
        echo '```markdown' >> "${COMBINED}"
        cat "$rule" >> "${COMBINED}"
        echo '```' >> "${COMBINED}"
        echo "" >> "${COMBINED}"
    fi
done

echo "## Conventions" >> "${COMBINED}"
for conv in "${OUTPUT_DIR}/conventions"/*.md; do
    if [[ -f "$conv" ]]; then
        echo "### $(basename "$conv")" >> "${COMBINED}"
        echo '```markdown' >> "${COMBINED}"
        cat "$conv" >> "${COMBINED}"
        echo '```' >> "${COMBINED}"
        echo "" >> "${COMBINED}"
    fi
done

echo ""
echo "=== Export Complete ==="
echo ""
echo "Directory export: ${OUTPUT_DIR}/"
echo "Combined file:    ${OUTPUT_DIR}/COMBINED-EXPORT.md"
echo ""
echo "File counts:"
find "${OUTPUT_DIR}" -type f | wc -l | xargs echo "  Total files:"
du -sh "${OUTPUT_DIR}" | cut -f1 | xargs echo "  Total size: "
wc -l "${COMBINED}" 2>/dev/null | cut -d' ' -f1 | xargs echo "  Combined file lines:"
