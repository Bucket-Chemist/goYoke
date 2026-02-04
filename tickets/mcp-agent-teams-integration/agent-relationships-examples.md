# Agent Relationship Field Examples

This shows how each agent type should be updated in agents-index.json.

---

## Orchestrator Agents

### mozart
```json
{
  "id": "mozart",
  "spawned_by": ["router"],
  "can_spawn": ["haiku-scout", "codebase-search", "einstein", "staff-architect-critical-review", "beethoven"],
  "invoked_by": "skill:braintrust",
  "inputs": ["user-problem-statement", "gap-document"],
  "outputs": [".claude/braintrust/problem-brief-*.md"],
  "outputs_to": ["einstein", "staff-architect-critical-review", "beethoven"],
  "must_delegate": true,
  "min_delegations": 3,
  "max_delegations": 5
}
```

### review-orchestrator
```json
{
  "id": "review-orchestrator",
  "spawned_by": ["router"],
  "can_spawn": ["backend-reviewer", "frontend-reviewer", "standards-reviewer", "architect-reviewer", "codebase-search"],
  "invoked_by": "skill:review",
  "inputs": ["files-to-review", "review-scope"],
  "outputs": [".claude/tmp/review-result.json", ".claude/tmp/review-report.md"],
  "outputs_to": [],
  "must_delegate": true,
  "min_delegations": 2,
  "max_delegations": 4
}
```

### impl-manager
```json
{
  "id": "impl-manager",
  "spawned_by": ["router", "orchestrator"],
  "can_spawn": ["go-pro", "go-cli", "go-tui", "go-api", "go-concurrent", "python-pro", "python-ux", "r-pro", "r-shiny-pro", "typescript-pro", "react-pro", "codebase-search", "code-reviewer"],
  "invoked_by": "skill:ticket",
  "inputs": [".claude/tmp/specs.md", "conventions/*.md"],
  "outputs": [".claude/tmp/impl-progress.json", ".claude/tmp/impl-violations.jsonl"],
  "outputs_to": [],
  "must_delegate": true,
  "min_delegations": 1,
  "max_delegations": 10
}
```

### orchestrator (general)
```json
{
  "id": "orchestrator",
  "spawned_by": ["router"],
  "can_spawn": ["codebase-search", "haiku-scout", "librarian", "code-reviewer", "architect", "go-pro", "python-pro", "typescript-pro", "react-pro"],
  "invoked_by": "router",
  "inputs": ["ambiguous-request"],
  "outputs": ["synthesized-response"],
  "outputs_to": [],
  "must_delegate": false,
  "min_delegations": 0
}
```

---

## Analysis Agents (Braintrust)

### einstein
```json
{
  "id": "einstein",
  "spawned_by": ["mozart"],
  "can_spawn": ["codebase-search", "haiku-scout"],
  "invoked_by": "orchestrator:mozart",
  "inputs": ["problem-brief", "gathered-context"],
  "outputs": ["theoretical-analysis"],
  "outputs_to": ["beethoven"],
  "must_delegate": false
}
```

### staff-architect-critical-review
```json
{
  "id": "staff-architect-critical-review",
  "spawned_by": ["mozart", "router"],
  "can_spawn": ["codebase-search", "haiku-scout"],
  "invoked_by": "orchestrator:mozart",
  "inputs": ["problem-brief", "implementation-plan"],
  "outputs": [".claude/tmp/review-critique.md"],
  "outputs_to": ["beethoven"],
  "must_delegate": false
}
```

### beethoven
```json
{
  "id": "beethoven",
  "spawned_by": ["mozart"],
  "can_spawn": [],
  "invoked_by": "orchestrator:mozart",
  "inputs": ["problem-brief", "einstein-analysis", "staff-architect-review"],
  "outputs": [".claude/braintrust/analysis-*.md"],
  "outputs_to": [],
  "must_delegate": false
}
```

---

## Reviewer Agents (Leaf)

### backend-reviewer
```json
{
  "id": "backend-reviewer",
  "spawned_by": ["review-orchestrator", "orchestrator"],
  "can_spawn": [],
  "invoked_by": "orchestrator:review-orchestrator",
  "inputs": ["backend-files", "review-criteria"],
  "outputs": ["backend-findings"],
  "outputs_to": [],
  "must_delegate": false
}
```

### frontend-reviewer
```json
{
  "id": "frontend-reviewer",
  "spawned_by": ["review-orchestrator", "orchestrator"],
  "can_spawn": [],
  "invoked_by": "orchestrator:review-orchestrator",
  "inputs": ["frontend-files", "review-criteria"],
  "outputs": ["frontend-findings"],
  "outputs_to": [],
  "must_delegate": false
}
```

### standards-reviewer
```json
{
  "id": "standards-reviewer",
  "spawned_by": ["review-orchestrator", "orchestrator"],
  "can_spawn": [],
  "invoked_by": "orchestrator:review-orchestrator",
  "inputs": ["source-files", "conventions"],
  "outputs": ["standards-findings"],
  "outputs_to": [],
  "must_delegate": false
}
```

### architect-reviewer
```json
{
  "id": "architect-reviewer",
  "spawned_by": ["review-orchestrator", "orchestrator"],
  "can_spawn": [],
  "invoked_by": "orchestrator:review-orchestrator",
  "inputs": ["source-files", "architecture-context"],
  "outputs": ["architecture-findings"],
  "outputs_to": [],
  "must_delegate": false
}
```

---

## Implementation Agents (Leaf)

### go-pro
```json
{
  "id": "go-pro",
  "spawned_by": ["impl-manager", "orchestrator", "router"],
  "can_spawn": ["codebase-search"],
  "invoked_by": "orchestrator:impl-manager",
  "inputs": ["task-spec", "conventions/go.md"],
  "outputs": ["implemented-code"],
  "outputs_to": [],
  "must_delegate": false
}
```

### python-pro
```json
{
  "id": "python-pro",
  "spawned_by": ["impl-manager", "orchestrator", "router"],
  "can_spawn": ["codebase-search", "python-architect"],
  "invoked_by": "orchestrator:impl-manager",
  "inputs": ["task-spec", "conventions/python.md"],
  "outputs": ["implemented-code"],
  "outputs_to": [],
  "must_delegate": false
}
```

---

## Scout Agents (Universal)

### haiku-scout
```json
{
  "id": "haiku-scout",
  "spawned_by": ["any"],
  "can_spawn": [],
  "invoked_by": "any",
  "inputs": ["exploration-query"],
  "outputs": [".claude/tmp/scout_metrics.json"],
  "outputs_to": [],
  "must_delegate": false
}
```

### codebase-search
```json
{
  "id": "codebase-search",
  "spawned_by": ["any"],
  "can_spawn": [],
  "invoked_by": "any",
  "inputs": ["search-query"],
  "outputs": ["search-results"],
  "outputs_to": [],
  "must_delegate": false
}
```

---

## Visualization Examples

### AgentTree with Relationships

```
Epic: braintrust (running)

▼ 🏃 → mozart (opus)                    [can_spawn: einstein, staff-arch, beethoven]
  ├─ 📡 ⚡ einstein (opus)              [spawned_by: mozart ✓]
  │      └─ outputs_to: beethoven
  ├─ 📡 ⚡ staff-architect (opus)       [spawned_by: mozart ✓]
  │      └─ outputs_to: beethoven
  └─ ⏳ ⚡ beethoven (opus)              [spawned_by: mozart ✓]
         └─ inputs: einstein-analysis, staff-architect-review

Delegation: 3/3 ✓ (must_delegate met)
```

### AgentDetail with Validation

```
┌─ Agent Detail ──────────────────────────────────────────────┐
│                                                             │
│ einstein • opus • streaming                                 │
│                                                             │
│ Hierarchy:                                                  │
│   Epic: braintrust-abc123                                   │
│   Parent: mozart                                            │
│   Depth: 2                                                  │
│   Spawn: mcp-cli                                            │
│                                                             │
│ Relationships (from schema):                                │
│   Spawned by: mozart                                        │
│   Can spawn: codebase-search, haiku-scout                   │
│   Outputs to: beethoven                                     │
│   Inputs: problem-brief, gathered-context                   │
│                                                             │
│ Validation:                                                 │
│   ✓ Parent allowed (mozart in spawned_by)                   │
│   ✓ No unexpected children spawned                          │
│   ✓ Not a delegating agent                                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### EpicOverview with Data Flow

```
┌─ Epic: braintrust ──────────────────────────────────────────┐
│                                                             │
│ Status: running • 3 agents • $0.0891                        │
│                                                             │
│ Data Flow:                                                  │
│   mozart ──────────┬──→ einstein ────────┐                  │
│   (problem-brief)  │                     │                  │
│                    └──→ staff-architect ─┼──→ beethoven     │
│                                          │   (synthesis)    │
│                                          └───────┘          │
│                                                             │
│ Delegation Progress:                                        │
│   mozart: ███████████ 3/3 ✓                                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## MCP spawn_agent Validation

The spawn_agent tool should validate relationships before spawning:

```typescript
async function validateSpawn(parentId: string, childType: string): Promise<ValidationResult> {
  const parentConfig = getAgentConfig(getAgentType(parentId));
  const childConfig = getAgentConfig(childType);

  const errors: string[] = [];
  const warnings: string[] = [];

  // Check if parent is allowed to spawn this child
  if (parentConfig.can_spawn && !parentConfig.can_spawn.includes(childType)) {
    errors.push(`${parentConfig.id} cannot spawn ${childType} (not in can_spawn)`);
  }

  // Check if child expects to be spawned by this parent
  if (childConfig.spawned_by &&
      !childConfig.spawned_by.includes('any') &&
      !childConfig.spawned_by.includes(parentConfig.id)) {
    warnings.push(`${childType} expects to be spawned by ${childConfig.spawned_by.join('|')}, not ${parentConfig.id}`);
  }

  // Check delegation limits
  if (parentConfig.max_delegations) {
    const currentChildren = getChildCount(parentId);
    if (currentChildren >= parentConfig.max_delegations) {
      errors.push(`${parentConfig.id} at max delegations (${currentChildren}/${parentConfig.max_delegations})`);
    }
  }

  return { valid: errors.length === 0, errors, warnings };
}
```
