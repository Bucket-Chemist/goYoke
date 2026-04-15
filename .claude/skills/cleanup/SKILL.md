---
name: cleanup
description: Systematic codebase hygiene through 8 specialist cleanup reviewers with Opus synthesis
---

# Cleanup Skill v1.0

## Purpose

Systematic codebase cleanup through 8 coordinated specialist reviewers, each examining the code through a different quality lens. Findings are synthesized by an Opus-tier agent that deduplicates, identifies causal chains, and produces a phased remediation plan.

**What this skill does:**

1. **Scope** — Determine cleanup target (whole project, specific path, or glob)
2. **Detect** — Identify languages present and gather file inventory
3. **Execute** — Dispatch all 8 reviewers (wave 0) + synthesizer (wave 1) via background team-run
4. **Launch** — Start \`gogent-team-run\` in background, return immediately

**What this skill does NOT do:**

- Implement fixes (generates phased remediation plan only)
- Replace human judgment on which findings to act on
- Run on vendor/node_modules/generated code (excluded by default)

---

## Invocation

- \`/cleanup\` — Analyze entire project
- \`/cleanup path/to/module\` — Analyze specific directory
- \`/cleanup --scope="**/*.go"\` — Analyze files matching glob
- \`/cleanup --scope="**/*.ts,**/*.tsx"\` — Multiple globs (comma-separated)

---

## Prerequisites

**Required tools:**

- \`git\` (for project root detection)
- \`jq\` (JSON processing)
- \`gogent-team-run\` (team execution)

**Optional tools (enhance analysis):**

- \`knip\` (TS/JS dead code detection — dead-code-reviewer)
- \`madge\` (TS/JS dependency graph — dependency-reviewer)
- \`staticcheck\` (Go static analysis — dead-code-reviewer)
- \`vulture\` (Python dead code — dead-code-reviewer)

---

## Workflow

When \`/cleanup\` is invoked, the \`gogent-skill-guard\` PreToolUse hook has already:
- Created the team directory (\`{gogent_session_dir}/teams/{timestamp}.cleanup/\`)
- Written \`active-skill.json\` with guard restrictions + \`team_dir\` path
- Restricted the router to: Task, Bash, Read, AskUserQuestion, Skill

The \`gogent_session_dir\` lives under \`{project_root}/.gogent/sessions/\`, NOT \`.claude/sessions/\`. It is resolved by reading \`{project_root}/.gogent/current-session\`.

### Phase 1: Read Guard File and Determine Scope

#### Step 1: Read Team Directory from Guard File

\`\`\`javascript
Read({ file_path: \`\${gogent_session_dir}/active-skill.json\` })
// Extract team_dir from JSON response
\`\`\`

#### Step 2: Determine Scope

\`\`\`bash
cleanup_scope="project"  # default
if [[ "\$1" == --scope=* ]]; then
    cleanup_scope="glob"
    glob_pattern="\${1#--scope=}"
elif [[ -n "\$1" ]]; then
    cleanup_scope="explicit"
    cleanup_target="\$1"
fi

# Build file list
case "\$cleanup_scope" in
    project)
        files=\$(find . -type f \\( -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.py" -o -name "*.rs" -o -name "*.R" \\) \
            ! -path "*/vendor/*" ! -path "*/node_modules/*" ! -path "*/dist/*" ! -path "*/.git/*" ! -path "*/generated/*")
        ;;
    glob)
        IFS=',' read -ra patterns <<< "\$glob_pattern"
        files=""
        for pat in "\${patterns[@]}"; do
            files+=\$'\\n'\$(find . -type f -path "\$pat" ! -path "*/vendor/*" ! -path "*/node_modules/*")
        done
        ;;
    explicit)
        if [[ -d "\$cleanup_target" ]]; then
            files=\$(find "\$cleanup_target" -type f \\( -name "*.go" -o -name "*.ts" -o -name "*.tsx" -o -name "*.py" \\) \
                ! -path "*/vendor/*" ! -path "*/node_modules/*")
        else
            files="\$cleanup_target"
        fi
        ;;
esac

# Detect languages
declare -A langs
while IFS= read -r file; do
    [[ -z "\$file" ]] && continue
    ext="\${file##*.}"
    langs["\$ext"]=1
done <<< "\$files"

file_count=\$(echo "\$files" | grep -c '.' || echo 0)
echo "[cleanup] Found \$file_count files to analyze"
echo "[cleanup] Languages: \${!langs[*]}"
\`\`\`

---

### Phase 2: Generate Team Config and Stdin Files

#### Step 1: Generate config.json

Read the template from \`.claude/schemas/teams/cleanup.json\` and populate:

- \`team_name\`: \`"cleanup-\$(date +%Y%m%d-%H%M%S)"\`
- \`workflow_type\`: \`"cleanup"\`
- \`project_root\`: \`\$(git rev-parse --show-toplevel)\`
- \`session_id\`: basename of \`\$GOGENT_SESSION_DIR\`
- \`created_at\`: \`\$(date -u +%Y-%m-%dT%H:%M:%SZ)\`
- \`budget_max_usd\`: \`25.0\`
- \`budget_remaining_usd\`: \`25.0\`
- \`warning_threshold_usd\`: \`20.0\`
- \`status\`: \`"pending"\`

All 8 reviewers in wave 0 (parallel). Synthesizer in wave 1 (sequential, after wave 0).

Write to \`\$team_dir/config.json\`.

#### Step 2: Generate Stdin Files for Reviewers

For each of the 8 reviewers, generate a stdin JSON file:

\`\`\`json
{
  "agent": "<reviewer-id>",
  "workflow": "cleanup",
  "description": "Analyze codebase through <lens-name> lens",
  "context": {
    "project_root": "<absolute project root>",
    "team_dir": "<absolute team directory>"
  },
  "scope": {
    "target": "<project|glob|explicit>",
    "include": ["<glob patterns>"],
    "exclude": ["vendor/", "node_modules/", "dist/", ".git/", "generated/"],
    "file_count": 0,
    "languages_detected": ["<languages>"]
  },
  "project_conventions": {
    "languages": ["<detected languages>"],
    "conventions_files": ["<matching convention files>"]
  }
}
\`\`\`

Write each to \`\$team_dir/stdin_<reviewer-id>.json\`.

#### Step 3: Generate Stdin File for Synthesizer

\`\`\`json
{
  "agent": "cleanup-synthesizer",
  "workflow": "cleanup",
  "description": "Synthesize findings from 8 cleanup reviewers",
  "context": {
    "project_root": "<absolute project root>",
    "team_dir": "<absolute team directory>"
  },
  "reviewer_outputs": [
    "stdout_dedup-reviewer.json",
    "stdout_type-consolidator.json",
    "stdout_dead-code-reviewer.json",
    "stdout_dependency-reviewer.json",
    "stdout_type-safety-reviewer.json",
    "stdout_error-hygiene-reviewer.json",
    "stdout_legacy-code-reviewer.json",
    "stdout_slop-reviewer.json"
  ]
}
\`\`\`

Write to \`\$team_dir/stdin_cleanup-synthesizer.json\`.

---

### Phase 3: Launch and Return

\`\`\`
result = mcp__gofortress-interactive__team_run({
    team_dir: "\$team_dir",
    wait_for_start: true,
    timeout_ms: 10000
})
if !result.success:
    echo "[cleanup] ERROR: \${result.result}"
    rm -f "\$gogent_session_dir/active-skill.json"
    exit 1
background_pid = result.background_pid
\`\`\`

#### Step 4: Remove Skill Guard

\`\`\`bash
rm -f "\$gogent_session_dir/active-skill.json"
\`\`\`

#### Step 5: Return to User

\`\`\`
[cleanup] Cleanup analysis launched in background
  Reviewers (wave 0): dedup, types, dead-code, dependencies, type-safety, error-hygiene, legacy, slop
  Synthesizer (wave 1): cleanup-synthesizer (Opus)
  Scope: {file_count} files across {language_count} languages
  Team: {team_dir}
  PID: {background_pid}

Use /team-status to check progress
Use /team-result to view the remediation plan when complete
\`\`\`

---

## The 8 Cleanup Lenses

| # | Agent | Lens | What It Finds |
|---|-------|------|---------------|
| 1 | dedup-reviewer | Deduplication | Copy-paste, structural similarity, DRY violations |
| 2 | type-consolidator | Type Organization | Scattered types, redundant aliases, misplaced definitions |
| 3 | dead-code-reviewer | Dead Code | Unused exports, orphaned files, unused dependencies |
| 4 | dependency-reviewer | Dependency Health | Circular imports, god modules, layer violations |
| 5 | type-safety-reviewer | Type Safety | any/unknown/interface{}, unsafe assertions, missing annotations |
| 6 | error-hygiene-reviewer | Error Hygiene | Empty catch, error hiding, unnecessary try/catch |
| 7 | legacy-code-reviewer | Legacy Code | Deprecated code, compat shims, stale feature flags |
| 8 | slop-reviewer | Slop & Stubs | AI slop, stubs, LARP code, unhelpful comments |

---

## Synthesizer Output

The cleanup-synthesizer (Opus, wave 1) produces:

1. **Executive Summary** — overall health score, top 3 priorities
2. **Causal Chains** — root causes with their downstream symptoms
3. **Conflict Resolution** — when agents disagree, resolution with reasoning
4. **Phased Remediation Plan** — 8 phases ordered by dependency:
   - Phase 1: Structural (break cycles)
   - Phase 2: Pruning (remove dead code)
   - Phase 3: Legacy (remove fallbacks)
   - Phase 4: Consolidation (merge duplicates)
   - Phase 5: Types (consolidate type definitions)
   - Phase 6: Safety (strengthen types)
   - Phase 7: Errors (clean error handling)
   - Phase 8: Polish (remove slop)
5. **Per-File Action List** — what to do in each file

---

## Cost Model

| Component | Model | Est. Tokens | Cost |
|-----------|-------|-------------|------|
| Scope detection | Bash | 0 | \$0.00 |
| Config generation | Router | ~2K | \$0.00 |
| 8x Sonnet Reviewers | Sonnet | 20-40K each | \$1.50-\$3.00 each |
| Cleanup Synthesizer | Opus | 30-60K | \$2.50-\$5.00 |
| **Typical total** | | 190-380K | **\$15-\$30** |
| Budget cap | | | **\$25.00** |

---

## Partial Failure Handling

If one or more wave 0 reviewers fail:
- Synthesizer works with available results
- Failed reviewers noted prominently in report
- Caveat: "Analysis incomplete — N of 8 reviewers completed"
- Health scores adjusted for gaps

If synthesizer fails:
- Individual reviewer stdout files available via \`/team-result\`
- No cross-lens synthesis, but per-lens findings are intact

---

## State Files

| File | Purpose | Format |
|------|---------|--------|
| \`{team_dir}/config.json\` | Team execution config | JSON |
| \`{team_dir}/stdin_*.json\` | Per-reviewer/synthesizer input | JSON |
| \`{team_dir}/stdout_*.json\` | Per-reviewer/synthesizer output | JSON |
| \`{team_dir}/runner.log\` | Execution log | Text |

---

## Common Output Contract

All 8 reviewers produce JSON with this structure (synthesizer consumes it):

\`\`\`json
{
  "agent": "<agent-id>",
  "lens": "<lens-name>",
  "status": "complete|partial|failed",
  "summary": {
    "files_analyzed": "<int>",
    "findings_count": "<int>",
    "by_severity": {"critical": 0, "high": 0, "medium": 0, "low": 0},
    "health_score": "<float 0-10>",
    "top_concern": "<one sentence>"
  },
  "findings": [{
    "id": "<prefix-NNN>",
    "severity": "critical|high|medium|low",
    "category": "<agent-specific>",
    "title": "<short>",
    "locations": [{
      "file": "<relative path>",
      "line_start": 0, "line_end": 0,
      "snippet": "<max 10 lines>",
      "role": "primary|duplicate|dependency|consumer|related"
    }],
    "description": "<detailed>",
    "impact": "<consequence>",
    "recommendation": "<actionable>",
    "action_type": "<delete|merge|extract|move|retype|narrow|simplify|remove-guard|remove-fallback|rewrite-comment|delete-comment|invert-dependency|extract-interface>",
    "effort": "trivial|small|medium|large",
    "confidence": "<float 0-1>",
    "tags": ["<freeform>"],
    "language": "<go|typescript|python|rust|r>",
    "sharp_edge_id": "<optional>"
  }],
  "caveats": [],
  "tools_used": []
}
\`\`\`

---

## Troubleshooting

**"No files to analyze"**
- Check the scope — are there source files matching the patterns?
- Use \`--scope=\` to specify files explicitly

**"Reviewer not found"**
- Ensure agents-index.json includes all 9 cleanup agents
- Check that agent .md files exist in .claude/agents/

**"Team launch failed"**
- Check \`\$team_dir/runner.log\` for errors
- Verify \`gogent-team-run\` is built and in PATH
- Validate \`\$team_dir/config.json\`: \`jq . "\$team_dir/config.json"\`

**"Synthesizer has no data"**
- Check wave 0 reviewer stdout files exist
- At least 1 reviewer must complete for synthesis to work

---

## Example Session

\`\`\`bash
$ /cleanup

[cleanup] Found 127 files to analyze
[cleanup] Languages: go ts tsx
[cleanup] Cleanup analysis launched in background
  Reviewers (wave 0): dedup, types, dead-code, dependencies, type-safety, error-hygiene, legacy, slop
  Synthesizer (wave 1): cleanup-synthesizer (Opus)
  Scope: 127 files across 3 languages
  Team: .gogent/sessions/.../teams/1713196800.cleanup
  PID: 67890

Use /team-status to check progress
Use /team-result to view the remediation plan when complete

$ /team-status
[team-status] Team: cleanup
Status: running
Started: 45 seconds ago
Progress: 5/8 reviewers complete, synthesizer pending

Wave 0:
  dedup-reviewer (completed 12s ago, 4 findings)
  type-consolidator (completed 8s ago, 2 findings)
  dead-code-reviewer (completed 5s ago, 6 findings)
  dependency-reviewer (completed 3s ago, 1 finding)
  type-safety-reviewer (running, 40s elapsed)
  error-hygiene-reviewer (completed 15s ago, 3 findings)
  legacy-code-reviewer (running, 35s elapsed)
  slop-reviewer (completed 20s ago, 8 findings)

Wave 1:
  cleanup-synthesizer (pending — waiting for wave 0)

$ /team-result
[team-result] Team: cleanup (completed 2 minutes ago)

## Executive Summary
Overall Health: 6.8/10
Raw Findings: 31 | Deduplicated: 24 | Causal Chains: 3

Top 3 Priorities:
1. Circular dependency between internal/tui/model and internal/tui/mcp (blocks 4 other fixes)
2. 6 unused exported functions across cmd/ packages
3. 11 instances of AI slop comments

## Phase 1: Structural (1 finding)
  dep-001 [HIGH] Circular import: model → mcp → model
  ...

## Phase 2: Pruning (6 findings)
  dead-001 [HIGH] Unused export: HandleLegacyAuth in cmd/auth/
  ...
\`\`\`

---

**Skill Version:** 1.0
**Last Updated:** 2026-04-15
