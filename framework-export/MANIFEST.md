# GOgent-Fortress Framework Export

**Generated:** 2026-02-01T07:43:50+07:00
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
