# Agent Contract End-to-End Guide

Lifecycle guide for adding a new agent with constrained decoding to the goYoke team-run pipeline.

---

## Data Flow Overview

```
Agent Identity          Formal Schema              Contract Example
.claude/agents/         .claude/schemas/stdout/    .claude/schemas/teams/
{id}/{id}.md            {role}.json                stdin-stdout/{wf}-{role}.json
       |                       |                              |
       v                       v                              v
agents-index.json       meta-fields stripped         embedded in prompt
(model, tools,          at runtime → passed          by buildPromptEnvelope()
 can_spawn, etc.)       as --json-schema arg
       |                       |                              |
       +--------+--------------+------------------------------+
                |
                v
        goyoke-team-run
        config.json (wave/member definitions)
                |
                v
        prepareSpawn()
          - loadAgentConfig()         reads agents-index.json
          - BuildFullAgentContext()   injects identity + rules + conventions
          - buildPromptEnvelope()     reads stdin file, embeds contract example
          - buildCLIArgs()            assembles --json-schema flag
                |
                v
        claude -p --output-format stream-json
               --model {model}
               --allowedTools {tools}
               [--json-schema '{stripped schema}']
                |
                v
        Agent executes, produces JSON stdout
                |
                v
        parseCLIOutput()    extracts result from stream-json
        writeStdoutFile()   extracts JSON from result text, writes {agent}.stdout.json
        validateStdout()    checks $schema and status fields present
```

---

## Step-by-Step: Adding a New Agent with Constrained Decoding

### Step 1: Create Agent Identity

Create the agent's identity file at `.claude/agents/{id}/{id}.md`. This file defines the agent's persona, operating principles, and behavioral constraints. It is injected into the prompt by `BuildFullAgentContext()` in `pkg/routing/identity_loader.go`.

The file should have YAML frontmatter (compile-time metadata, not parsed at runtime) followed by the agent's markdown identity:

```markdown
---
id: my-analyst
model: sonnet
spawned_by: [mozart, orchestrator]
can_spawn: []
---

# My Analyst Agent

You are a specialized analyst that...
```

### Step 2: Add to agents-index.json

Add an entry to `.claude/agents/agents-index.json`:

```json
{
  "id": "my-analyst",
  "model": "sonnet",
  "cli_flags": {
    "allowed_tools": ["Read", "Glob", "Grep", "Bash"],
    "additional_flags": []
  },
  "context_requirements": {
    "rules": ["agent-guidelines.md"],
    "conventions": ["go.md"]
  },
  "spawned_by": ["mozart"],
  "can_spawn": []
}
```

Note: `cli_flags.allowed_tools` is READ-ONLY for analysis agents. If the agent needs to create or modify files (implementation workflow), `augmentToolsForImplementation()` in `cmd/goyoke-team-run/spawn.go` adds `Write` and `Edit` automatically when `workflowType == "implementation"`.

### Step 3: Create the Formal Stdout Schema

Create `.claude/schemas/stdout/{role}.json`. The filename is the role, which may differ from the agent ID (multiple agents can share a schema, as all reviewer agents share `reviewer.json`).

See [STDOUT-SCHEMA-AUTHORING-GUIDE.md](STDOUT-SCHEMA-AUTHORING-GUIDE.md) for the full authoring process. Key requirements:

- Include `schema_id` as a `const` field in `required`
- Include `status` with `enum: ["complete", "partial", "failed"]`
- Set `"additionalProperties": false`
- Keep meta-fields (`$schema`, `$id`, `title`, `description`, `version`) for readability — they are stripped at runtime
- Apply enum/string decisions per [ENUM-CLASSIFICATION-GUIDE.md](ENUM-CLASSIFICATION-GUIDE.md)

### Step 4: Create the Contract Example

Create `.claude/schemas/teams/stdin-stdout/{workflow}-{role}.json`. This is the prompt-level contract embedded in the agent's envelope:

```json
{
  "$comment": "Stdin/stdout contract for my-analyst in analysis workflow",
  "stdin": {
    "agent": "my-analyst",
    "workflow": "analysis",
    "description": "Analyze {context.target}",
    "context": {
      "project_root": "/absolute/path/to/project",
      "target": "what to analyze"
    },
    "task": "Perform analysis of the given target"
  },
  "stdout": {
    "schema_id": "analysis-my-analyst",
    "status": "complete",
    "summary": "One-sentence summary of findings",
    "findings": [],
    "metadata": {
      "thinking_budget_used": 0
    }
  }
}
```

The `stdout` value is embedded verbatim in the prompt with instructions to follow this schema. Keep it minimal — it is for agent guidance, not exhaustive documentation.

### Step 5: Schema Resolution — How team-run Finds the Schema

`resolveStdoutSchema(workflowType, agentID)` in `cmd/goyoke-team-run/envelope.go` builds candidates in order:

```
1. "{workflowType}-{agentID}.json"           → "analysis-my-analyst.json"
2. Strip "-reviewer" suffix (if applicable)  → not applicable here
3. Strip "-critical-review" suffix            → not applicable here
4. Strip "-pro" suffix                        → not applicable here
5. Strip "-writer" suffix                     → not applicable here
6. Strip "-archivist" suffix                  → not applicable here
7. "{workflowType}-worker.json"               → "analysis-worker.json" (generic fallback)
```

The first file found wins. The function reads the `stdout` key from the matched contract file.

For the example above with agentID `my-analyst` and workflowType `analysis`, the exact match `analysis-my-analyst.json` is found at candidate 1.

For `staff-architect-critical-review` with workflowType `braintrust`, candidate 1 (`braintrust-staff-architect-critical-review.json`) does not exist, candidate 2 strips `-critical-review` to produce `braintrust-staff-architect.json`, which exists.

### Step 6: Runtime Flow

When `goyoke-team-run` spawns the agent:

1. `prepareSpawn()` reads member config from `config.json`
2. `loadAgentConfig()` reads `agents-index.json` for tools, model, context requirements
3. `buildPromptEnvelope()` reads the stdin file, embeds the contract example from `resolveStdoutSchema()`
4. `BuildFullAgentContext()` injects agent identity + rules + conventions
5. `buildCLIArgs()` assembles the CLI argument list including `--json-schema` with the stripped formal schema (meta-fields removed)
6. `executeSpawn()` starts `claude -p --output-format stream-json [args]`
7. `parseCLIOutput()` scans NDJSON output for the `"type": "result"` entry, extracts `result` field
8. `writeStdoutFile()` extracts JSON from the result text, writes `{agent}.stdout.json` to the team directory
9. `validateStdout()` checks that `$schema` and `status` fields are present in the written file

---

## Troubleshooting

### Agent output does not match the schema

Check these in order:

1. **`schema_id` missing from `required`**: The agent will not produce `schema_id` if it is not required. Add it to `required` and set type to `const`.

2. **Enum vs string mismatch**: If the agent produces `"high confidence"` but the schema has `"enum": ["high", "medium", "low"]`, constrained decoding will force a pick. Review the enum classification decision — if this is a judgment field, relax to `string`.

3. **`additionalProperties: false` absent**: The agent may produce extra fields that cause downstream parsing to fail unexpectedly. Always include `"additionalProperties": false`.

4. **Schema file has meta-fields causing CLI rejection**: If `--json-schema` throws a parse error, the schema may contain unsupported fields (`$schema`, `$id`, `title`, `description`, `version`). Verify stripping logic removes these before passing.

### `--json-schema` flag not appearing in CLI args

`buildCLIArgs()` in `cmd/goyoke-team-run/spawn.go` adds `--json-schema` only when a formal schema is found for the agent. Check:

1. Does `.claude/schemas/stdout/{role}.json` exist for this agent's role?
2. Does the schema resolution chain in `buildSchemaCandidates()` produce a match? Add logging to verify which candidate matches.
3. Is the schema file valid JSON? A parse error during stripping would silently skip the flag.

### `structured_output` is null in CLI output

The `structured_output` field in the Claude CLI result entry is populated only when constrained decoding is active. If it is null:

- `--json-schema` was not passed (schema not found — see above)
- The schema was rejected by the CLI (meta-fields not stripped)
- The CLI version does not support `--json-schema`

The pipeline currently reads from the `result` field (text output) rather than `structured_output`. A null `structured_output` does not necessarily mean the output is unusable — `writeStdoutFile()` attempts to parse JSON from the text result regardless.

### Legacy agents without a formal schema

Agents added before constrained decoding was introduced do not have entries in `.claude/schemas/stdout/`. They continue to work: `buildCLIArgs()` omits `--json-schema` when no schema is found, and the agent receives only the contract example in the prompt (prompt-level guidance, no token-level enforcement).

To upgrade a legacy agent to constrained decoding:
1. Create the formal schema in `.claude/schemas/stdout/{role}.json`
2. Verify the contract example in `.claude/schemas/teams/stdin-stdout/` exists
3. Test by running a team-run and checking that `--json-schema` appears in the spawned process args (visible in process output / stream NDJSON)
