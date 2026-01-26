# Sharp Edge: Project-Local System File References in Hooks

## Issue Summary

**Discovered**: 2026-01-13
**Severity**: Medium (causes silent degradation and output suppression)
**Status**: ✅ FIXED
**Affected Hooks**: 3 (validate-routing.sh, load-routing-context.sh, attention-gate.sh)

## The Problem

Three global hooks incorrectly referenced **system-level configuration files** using `$PROJECT_DIR/.claude/` paths instead of `$HOME/.claude/` paths.

This caused two failure modes:

### 1. Silent Validation Degradation

When `routing-schema.json` was not found in project directory, hooks would silently exit with:
```bash
if [[ ! -f "$SCHEMA_FILE" ]]; then
    exit 0  # Silent pass
fi
```

**Result**: Routing validation would be **disabled** in projects without local schema copies.

### 2. Output Suppression

When hooks errored (especially on session stop), Claude Code CLI would suppress agent output, preventing user from seeing important messages like user interview questions.

**Manifestation**:
```
● Plan(Orchestrate MultiScholaR documentation project)
  ⎿  PreToolUse:Task hook error
  ⎿  PostToolUse:Task hook error
  ⎿  PostToolUse:Task hook error
```

User never saw orchestrator's output containing 5 clarifying questions.

## Root Cause

Hooks confused **project-local state files** with **global system configuration**.

### What Should Be Global (System-Level)

These files define **architecture**, not project state:
- `routing-schema.json` - Agent tier definitions, tool permissions
- `scripts/calculate-complexity.sh` - Complexity scoring algorithm
- Anything defining **how the system works**

**Correct path**: `${HOME}/.claude/`

### What Should Be Project-Local

These files track **project-specific state**:
- `memory/` - Project learnings, sharp edges
- `tmp/` - Session state (scout metrics, complexity scores)
- `plans/` - Implementation specs
- Anything tracking **what was done in this project**

**Correct path**: `$PROJECT_DIR/.claude/`

## The Architectural Bug

Original assumption: "Projects might want to override routing schema."

**Why this is wrong**:
1. Routing schema defines **agent capabilities** (system constant)
2. Allowing per-project overrides creates **inconsistent behavior**
3. User calling Claude in **any directory** should get **identical routing rules**
4. Project-specific adaptations belong in `./CLAUDE.md`, not schema duplication

## Affected Code Locations

### 1. validate-routing.sh (2 locations)

**Line 13** (FIXED):
```bash
# BEFORE
SCHEMA_FILE="$PROJECT_DIR/.claude/routing-schema.json"

# AFTER
SCHEMA_FILE="${HOME}/.claude/routing-schema.json"
```

**Line 74** (FIXED):
```bash
# BEFORE
CALC_SCRIPT="$PROJECT_DIR/.claude/scripts/calculate-complexity.sh"

# AFTER
CALC_SCRIPT="${HOME}/.claude/scripts/calculate-complexity.sh"
```

### 2. load-routing-context.sh (1 location)

**Line 13** (FIXED):
```bash
# BEFORE
SCHEMA_FILE="$PROJECT_DIR/.claude/routing-schema.json"

# AFTER
SCHEMA_FILE="${HOME}/.claude/routing-schema.json"
```

### 3. attention-gate.sh (1 location)

**Lines 83-84** (FIXED):
```bash
# BEFORE
if [[ -f "$PROJECT_DIR/.claude/routing-schema.json" ]]; then
    routing_hint=$(jq -r '.tiers | keys | join(", ")' "$PROJECT_DIR/.claude/routing-schema.json" 2>/dev/null || echo "haiku, sonnet, opus")
fi

# AFTER
if [[ -f "${HOME}/.claude/routing-schema.json" ]]; then
    routing_hint=$(jq -r '.tiers | keys | join(", ")' "${HOME}/.claude/routing-schema.json" 2>/dev/null || echo "haiku, sonnet, opus")
fi
```

## Correct Path Architecture

```
~/.claude/                              # Global system configuration
├── routing-schema.json                 # Agent tier rules (GLOBAL)
├── scripts/
│   └── calculate-complexity.sh         # Scoring logic (GLOBAL)
├── hooks/                              # Event handlers (GLOBAL)
│   ├── validate-routing.sh
│   └── ...
└── agents/                             # Agent definitions (GLOBAL)
    └── ...

$PROJECT_DIR/.claude/                   # Project-local state
├── memory/                             # What we learned HERE
│   ├── pending-learnings.jsonl
│   └── sharp-edges/
├── tmp/                                # Ephemeral session state
│   ├── scout_metrics.json
│   ├── complexity_score
│   └── recommended_tier
└── plans/                              # Implementation specs
    └── *.md
```

## Detection Pattern

**How to spot this issue in the future:**

1. Global hook references `$PROJECT_DIR/.claude/<system-file>`
2. System file defines **behavior**, not **state**
3. Hook has `exit 0` fallback when file missing
4. User reports "hook errors" or "missing output"

**Test**:
```bash
# Run Claude in empty directory
cd /tmp/test-empty-dir
claude

# Check if hooks error
# If they do, they're likely looking for project-local system files
```

## Prevention Guidelines

When writing hooks, ask:

| Question | Answer | Path |
|----------|--------|------|
| Does this define how agents work? | Yes | `${HOME}/.claude/` |
| Does this track what happened in this project? | Yes | `$PROJECT_DIR/.claude/` |
| Would this be identical across all projects? | Yes | `${HOME}/.claude/` |
| Should different projects have different versions? | Yes | `$PROJECT_DIR/.claude/` |

**Examples**:
- `routing-schema.json` - Defines agents → `${HOME}/.claude/`
- `last-handoff.md` - Tracks session → `$PROJECT_DIR/.claude/memory/`
- `calculate-complexity.sh` - Scoring algorithm → `${HOME}/.claude/scripts/`
- `scout_metrics.json` - Session state → `$PROJECT_DIR/.claude/tmp/`

## Testing the Fix

### Verify hooks reference correct paths

```bash
# Should return ONLY $HOME paths
grep -n "SCHEMA_FILE\|CALC_SCRIPT" ~/.claude/hooks/*.sh | grep -v PROJECT_DIR

# Should show correct references
grep "HOME.*routing-schema.json" ~/.claude/hooks/*.sh
```

### Test in clean directory

```bash
mkdir -p /tmp/test-claude-clean
cd /tmp/test-claude-clean
claude

# Should NOT see hook errors
# Routing validation should work
```

### Verify routing still works

```bash
# In any project directory
jq '.tiers.sonnet.tools' ~/.claude/routing-schema.json
# Should return tool list
```

## Impact Assessment

**Before fix**:
- Projects without local schema → routing disabled silently
- Hook errors suppressed agent output → user blind to questions
- Inconsistent behavior across directories

**After fix**:
- ✅ All projects use same routing rules (from `~/.claude/`)
- ✅ Hook errors eliminated
- ✅ Agent output displays correctly
- ✅ Validation works in all directories

## Related Sharp Edges

This is **NOT** the same issue as:
- Claude Code looking for `./.claude/hooks/` (that's a CLI design flaw)
- Project-local conventions in `./CLAUDE.md` (that's intentional)

This is specifically about **global hooks** incorrectly using **project paths** for **system configuration**.

## Lessons Learned

1. **System vs State**: Always distinguish architecture (global) from history (project-local)
2. **Silent Failures**: `exit 0` fallbacks hide bugs. Better to fail loudly.
3. **Path Hygiene**: Use `${HOME}/.claude/` for system files, `$PROJECT_DIR/.claude/` for state
4. **Output Suppression**: Hook errors can silently break user communication

## Verification Checklist

Before considering a hook "fixed":

- [ ] All system config files use `${HOME}/.claude/` paths
- [ ] All project state files use `$PROJECT_DIR/.claude/` paths
- [ ] Hook works in empty directory
- [ ] Hook works in project directory
- [ ] Hook failures are loud (not `exit 0`)
- [ ] Hook output reaches user (not suppressed)

## References

- Original issue discovered: Session 2026-01-13 15:45 GMT+11
- Fixed in: validate-routing.sh v2.2, load-routing-context.sh v1.1, attention-gate.sh v1.1
- Discussion: User question "why are all hooks looking for files in project directory?"
- Root cause analysis: 4 parallel Bash investigations
- Fix verification: Manual testing in MultiScholaR project

## Future Work

Consider:
1. **Hook testing framework** - Automated tests for path hygiene
2. **Schema validation** - Verify hooks reference correct files at startup
3. **Documentation audit** - Ensure all docs distinguish global vs project-local
4. **Symlink detection** - Warn if project has symlinked system files (anti-pattern)
