# Stdout Schema Authoring Guide

Practical guide for creating and maintaining stdout schemas in the goYoke team-run pipeline.

---

## The Two Schema Systems

### Formal Schemas (`schemas/stdout/*.json`)

Location: `.claude/schemas/stdout/`

Full JSON Schema documents used for constrained decoding enforcement. These are the source of truth for what an agent must produce. They include meta-fields (`$schema`, `$id`, `title`, `description`, `version`) for human readability and tooling, but these are stripped at runtime before the schema is passed to `--json-schema`.

Current schemas:

| File | Agent(s) | Workflow |
|------|----------|---------|
| `einstein.json` | einstein | braintrust |
| `staff-architect.json` | staff-architect-critical-review | braintrust |
| `beethoven.json` | beethoven | braintrust |
| `reviewer.json` | backend-reviewer, frontend-reviewer, standards-reviewer, architect-reviewer | review |
| `worker.json` | go-pro, python-pro, and other implementation agents | implementation |

### Contract Examples (`schemas/teams/stdin-stdout/*.json`)

Location: `.claude/schemas/teams/stdin-stdout/`

Simplified JSON documents containing both the stdin structure and stdout structure for a specific workflow+agent combination. These are embedded in the agent's prompt envelope by `buildPromptEnvelope()` in `cmd/goyoke-team-run/envelope.go`. They provide prompt-level guidance; they are not used for constrained decoding.

Current contract files:

| File | Workflow | Agent |
|------|----------|-------|
| `braintrust-einstein.json` | braintrust | einstein |
| `braintrust-staff-architect.json` | braintrust | staff-architect-critical-review |
| `braintrust-beethoven.json` | braintrust | beethoven |
| `review-backend.json` | review | backend-reviewer |
| `review-frontend.json` | review | frontend-reviewer |
| `review-standards.json` | review | standards-reviewer |
| `review-architect.json` | review | architect-reviewer |
| `implementation-worker.json` | implementation | (generic worker) |

---

## Creating a New Formal Stdout Schema

### Step 1: Define the required fields

List every field your agent must always produce. Fields that are optional (nice-to-have but not required for downstream processing) should be defined in `properties` but omitted from `required`.

### Step 2: Apply the field type decision guide

See the [Enum Classification Guide](ENUM-CLASSIFICATION-GUIDE.md) for the complete principle. In brief:

| Field Characteristic | Type Choice |
|---------------------|-------------|
| Finite set, machine-consumed, protocol value | `enum` |
| Human-judgment label, context-dependent | `string` |
| Fixed identifier (always the same value) | `const` |
| Structured string format (IDs, prefixes) | `string` + `pattern` |
| Numeric with a floor of zero | `integer` + `minimum: 0` |

### Step 3: Write the schema file

Save to `.claude/schemas/stdout/{role}.json`. The filename is the role name, not the agent ID. Multiple agents can share a schema (e.g., all four reviewer agents use `reviewer.json`).

**Minimal valid formal schema template:**

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://goyoke-fortress/schemas/stdout/{role}.json",
  "title": "{Role} Stdout Schema",
  "description": "Validates stdout for {agent description}",
  "version": "1.0.0",
  "type": "object",
  "required": [
    "schema_id",
    "status",
    "summary"
  ],
  "properties": {
    "schema_id": {
      "type": "string",
      "const": "{workflow}-{role}",
      "description": "Self-identification field for this output"
    },
    "status": {
      "type": "string",
      "enum": ["complete", "partial", "failed"],
      "description": "Task completion status"
    },
    "summary": {
      "type": "string",
      "description": "1-2 sentence summary of output"
    },
    "metadata": {
      "type": "object",
      "required": ["thinking_budget_used"],
      "properties": {
        "thinking_budget_used": {
          "type": "integer",
          "minimum": 0
        }
      }
    }
  },
  "additionalProperties": false
}
```

### Step 4: Wire up schema resolution

Schema resolution in `resolveStdoutSchema()` (`cmd/goyoke-team-run/envelope.go`) uses the contract example files, not the formal schema files directly. The formal schema gets passed to `--json-schema` via a separate resolution path.

For the contract example, create `.claude/schemas/teams/stdin-stdout/{workflow}-{role}.json` with the structure:

```json
{
  "$comment": "Stdin/stdout contract for {agent} in {workflow} workflow",
  "stdin": { ... },
  "stdout": {
    "schema_id": "{workflow}-{role}",
    "status": "complete",
    "summary": "Example summary text"
  }
}
```

The `stdout` value in this file is the example contract embedded in prompts. Keep it concise — it is for prompt guidance, not exhaustive schema specification.

### Step 5: Add to agents-index.json

Ensure the agent entry in `.claude/agents/agents-index.json` has the correct `model`, `cli_flags`, and `context_requirements` fields. The schema resolution is driven by the agent ID at runtime; no explicit schema pointer is stored in `agents-index.json`.

---

## The `schema_id` Field

Every agent stdout must include a `schema_id` field for self-identification. This field replaced `$schema` as the self-ID mechanism.

**Why not `$schema`?**

`$schema` is a JSON Schema meta-field. The formal schemas in `.claude/schemas/stdout/` use it to declare their own schema dialect. When the formal schema is stripped of meta-fields before passing to `--json-schema`, `$schema` is removed. If the agent's output also used `$schema` for self-identification, that field would conflict with JSON Schema semantics and be stripped during meta-field processing.

**`schema_id` convention:**

- Value: `{workflow}-{role}` (e.g., `"braintrust-einstein"`, `"review-reviewer"`)
- Type: always `const` in the formal schema (one correct value per schema)
- Required: yes, always include in `required` array

```json
"schema_id": {
  "type": "string",
  "const": "braintrust-einstein",
  "description": "Self-identification field for this output"
}
```

---

## Schema Resolution Algorithm

`resolveStdoutSchema(workflowType, agentID)` in `cmd/goyoke-team-run/envelope.go` finds the contract example to embed in the prompt. The resolution order is:

1. **Exact match**: `{workflowType}-{agentID}.json`
   - Example: `review-backend-reviewer.json` for agentID `backend-reviewer`

2. **Suffix-stripped variants**: strip common suffixes from agentID, try each
   - Suffixes: `-reviewer`, `-critical-review`, `-pro`, `-writer`, `-archivist`
   - Example: `staff-architect-critical-review` → strips `-critical-review` → tries `braintrust-staff-architect.json`

3. **Generic worker fallback**: `{workflowType}-worker.json`
   - Example: `implementation-worker.json` for any implementation agent without an exact match

The function returns the `stdout` key from the matched contract file, pretty-printed for embedding in the prompt envelope.

**Resolution lookup directory**: `$HOME/.claude/schemas/teams/stdin-stdout/` (overridable via `stdoutSchemasBaseDir` for tests)

---

## Validation Checklist Before Deployment

Before adding a new formal schema:

- [ ] All fields agents must produce are in `required`
- [ ] `schema_id` uses `const`, not `enum`, not `string`
- [ ] `status` uses `enum: ["complete", "partial", "failed"]`
- [ ] Protocol values use `enum`; classification labels use `string` (see Enum Classification Guide)
- [ ] `additionalProperties: false` is present (prevents extra fields silently passing)
- [ ] All array items have explicit `type` and `required` fields
- [ ] `minimum: 0` is set on all count/integer fields
- [ ] Meta-fields are present (`$schema`, `$id`, `title`, `description`, `version`) for human readability
- [ ] Schema parses as valid JSON (`python3 -m json.tool schema.json`)
- [ ] A corresponding contract example exists in `.claude/schemas/teams/stdin-stdout/`
- [ ] Schema resolution produces a match (test with `buildSchemaCandidates(workflowType, agentID)`)
