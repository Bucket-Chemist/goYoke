# Constrained Decoding Reference

Technical reference for `--json-schema` constrained decoding in the GOgent-Fortress team-run pipeline.

---

## What Constrained Decoding Is

Standard LLM output is probabilistic — the model samples tokens from a probability distribution. With prompt-level guidance ("output valid JSON"), the model is instructed to self-constrain, but there is no mechanism preventing structurally invalid output.

**Constrained decoding** operates at the token-sampling level. The model's probability distribution is masked at each sampling step so that only tokens that continue a valid JSON document (matching the provided schema) have non-zero probability. The model cannot produce invalid JSON because invalid tokens are simply unavailable during generation.

| Mechanism | Where It Operates | Reliability |
|-----------|------------------|-------------|
| Prompt instruction | Text layer (LLM guidance) | Probabilistic — model can deviate |
| `--json-schema` flag | Token-sampling layer (runtime) | Deterministic — invalid tokens blocked |

---

## How `--json-schema` Works in Claude CLI

The flag accepts an **inline JSON string**, not a file path:

```bash
# Correct: inline JSON string
claude -p --output-format stream-json --json-schema '{"type":"object","required":["status"],"properties":{"status":{"type":"string","enum":["complete","failed"]}}}'

# Incorrect: file path (not supported)
claude -p --output-format stream-json --json-schema /path/to/schema.json
```

The schema string must be passed directly on the command line. In `gogent-team-run`, this string is constructed at runtime by reading the formal schema from `.claude/schemas/stdout/`, stripping meta-fields, and serializing the result to a single-line JSON string.

The constrained output appears in the `structured_output` field of the CLI's result entry (in `--output-format stream-json`), not in `result`. `gogent-team-run` extracts the agent's text response from the `result` field in `parseCLIOutput()` (`cmd/gogent-team-run/spawn.go`).

---

## What Constrained Decoding Guarantees

Structural validity only:

| Property | Guaranteed | Notes |
|----------|-----------|-------|
| Output parses as valid JSON | Yes | Token masking prevents malformed JSON |
| Required fields are present | Yes | Schema `required` array is enforced |
| Field types are correct | Yes | `"type": "string"` prevents integers, etc. |
| Enum values are exact | Yes | Only listed values are producible |
| Pattern constraints match | Yes | `"pattern"` regex is enforced |
| Numeric bounds respected | Yes | `minimum`/`maximum` enforced |
| `additionalProperties: false` | Yes | Extra keys cannot be added |
| Content is correct or useful | No | The model may produce valid-but-wrong content |
| Reasoning is coherent | No | Structural validity says nothing about logic |
| Field values are meaningful | No | A string field can contain any valid string |

**Semantic correctness is not guaranteed.** An agent can produce a structurally valid JSON object with plausible-looking but incorrect analysis. Constrained decoding eliminates parse failures and missing-field errors; it does not verify the quality of reasoning.

---

## Supported JSON Schema Features

Empirically verified against Claude CLI (tested with schemas up to 248 lines, 7 nesting levels, 4.9KB):

| Feature | Supported | Example |
|---------|-----------|---------|
| `type` | Yes | `"type": "object"` |
| `required` | Yes | `"required": ["status", "summary"]` |
| `properties` | Yes | Nested property definitions |
| `enum` | Yes | `"enum": ["complete", "partial", "failed"]` |
| `const` | Yes | `"const": "einstein"` |
| `pattern` | Yes | `"pattern": "^einstein-"` |
| `minimum` / `maximum` | Yes | `"minimum": 0` on integer fields |
| `items` (arrays) | Yes | Array element type definitions |
| `additionalProperties` | Yes | `"additionalProperties": false` |
| Nested objects | Yes | Multi-level nesting works |
| Nested arrays | Yes | Arrays of objects with their own required fields |
| `$schema` (meta) | No | Must be stripped before passing |
| `$id` (meta) | No | Must be stripped before passing |
| `title` (meta) | No | Must be stripped before passing |
| `description` (meta) | No | Must be stripped; nested `description` in properties also stripped |
| `version` (custom) | No | Must be stripped before passing |

---

## Schema Size Considerations

The beethoven schema (`schemas/stdout/beethoven.json`) at 248 lines / 4.9KB with 7 nesting levels passes constrained decoding without issues. Very large schemas (tens of kilobytes) may increase generation latency as the token masker evaluates more schema states per step, but no hard limit has been established in testing.

---

## Meta-Field Stripping Requirement

Claude CLI rejects schemas containing meta-fields. Before passing a schema to `--json-schema`, the following top-level keys must be removed:

- `$schema`
- `$id`
- `title`
- `description` (top-level only; nested `description` inside `properties` entries should also be stripped to avoid wasting tokens in constrained mode)
- `version` (GOgent-Fortress custom field)

The formal schemas in `.claude/schemas/stdout/` retain these fields for human readability and tooling compatibility. Stripping happens at runtime when building the CLI argument.

**Example — formal schema (stored on disk):**
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://gogent-fortress/schemas/stdout/einstein.json",
  "title": "Einstein Stdout Schema",
  "description": "Validates stdout for Einstein in braintrust workflow",
  "version": "1.0.0",
  "type": "object",
  "required": ["analysis_id", "status"],
  "properties": { ... }
}
```

**Example — stripped schema (passed to CLI):**
```json
{
  "type": "object",
  "required": ["analysis_id", "status"],
  "properties": { ... }
}
```

---

## The `structured_output` Field

When `--json-schema` is active, Claude CLI populates a `structured_output` field in the result entry of the stream-json output. This field contains the constrained-decoded JSON object.

In `parseCLIOutput()` (`cmd/gogent-team-run/spawn.go`), the pipeline currently extracts the agent response from the `result` field (the text result), not `structured_output`. If `structured_output` is null, constrained decoding may not have been active — typically because no schema was resolved for this agent.

---

## Two Schema Systems

GOgent-Fortress maintains two distinct schema systems with different purposes:

| System | Location | Format | Used For |
|--------|----------|--------|----------|
| Formal schemas | `.claude/schemas/stdout/*.json` | Full JSON Schema with meta-fields | Constrained decoding (meta-fields stripped at runtime) |
| Contract examples | `.claude/schemas/teams/stdin-stdout/*.json` | Simplified stdin+stdout contracts | Prompt-level output guidance embedded in agent envelope |

Formal schemas are the authoritative definition of what an agent must output. Contract examples are the human-readable template embedded in the agent's prompt. Both point to the same structural requirements, but serve different roles in the pipeline.
