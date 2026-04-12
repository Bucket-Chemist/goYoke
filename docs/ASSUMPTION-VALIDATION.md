# Braintrust Assumption Validation

Generated: 2026-02-22
Source: Braintrust analysis `20260222.150000.braintrust`

---

## Assumption 1: Native agent discovery ignores unknown YAML frontmatter fields

**Status:** NEEDS MANUAL VALIDATION
**Priority:** HIGH
**Blocking:** Yes (affects whether we can add GOgent-specific metadata to frontmatter)

### Test Setup
Created test agent at `.claude/agents/test-frontmatter/test-frontmatter.md` with:
- Standard fields: `name`, `description`, `model`, `tools`
- Unknown fields: `x-gogent-tier`, `x-gogent-custom-field`, `unknown_field_123`

### Manual Test Procedure
```bash
# In a fresh terminal (not inside Claude Code session):
claude --print "list available agents" 2>&1 | tee /tmp/agent-list.txt

# Check for warnings:
grep -i "test-frontmatter\|warning\|error\|unknown" /tmp/agent-list.txt
```

### Expected Results
- ✅ **PASS:** Agent appears in list without warnings about unknown fields
- ❌ **FAIL:** Warnings/errors about `x-gogent-*` or `unknown_field_123`

### If FAIL
Migrate GOgent-specific fields to a separate metadata file or remove from frontmatter entirely.

---

## Assumption 2: Pipe mode `-p` doesn't respect agent definitions

**Status:** NEEDS MANUAL VALIDATION
**Priority:** HIGH
**Blocking:** No (confirms team-run must handle injection itself)

### Test Setup
Created test agent at `.claude/agents/restricted-test/restricted-test.md` with:
- `tools: [Read]` only (Bash should be blocked if respected)

### Manual Test Procedure
```bash
# In a fresh terminal:
unset CLAUDECODE

# Test 1: Does pipe mode see the agent definitions?
echo "List all agents you can see" | claude -p --output-format stream-json 2>&1 | head -50

# Test 2: Are tool restrictions applied?
echo "Use Bash to run: echo PIPE_TEST" | claude -p --output-format stream-json 2>&1 | grep -E "(Bash|PIPE_TEST)"
```

### Expected Results
- **If Bash executes:** Pipe mode does NOT respect agent tool restrictions
  - ✅ Confirms: team-run must inject restrictions via `--allowedTools`
- **If Bash blocked:** Pipe mode DOES respect agent definitions
  - 🔄 Re-evaluate: team-run might be able to use native injection

### Current Hypothesis
Based on Braintrust analysis: pipe mode likely does NOT load `.claude/agents/*.md` files because:
1. Pipe mode is designed for non-interactive batch processing
2. Agent discovery is an interactive feature
3. The `-p` flag documentation mentions nothing about agent loading

---

## Assumption 3: GOgent-Fortress maintenance cost is <2 hours/month

**Status:** TRACKING SETUP NEEDED
**Priority:** MEDIUM
**Blocking:** No (informational for future migration decisions)

### Tracking Method
Add to `.claude/memory/MEMORY.md` monthly:

```yaml
## Monthly Maintenance Log

### 2026-02
- agents-index.json changes: [count] edits, [minutes] total
- identity_loader.go changes: [count] edits, [minutes] total
- gogent-validate changes: [count] edits, [minutes] total
- Total agent-related maintenance: [minutes]
```

### Git Log Analysis
```bash
# Check change frequency over past 30 days:
git log --since="30 days ago" --oneline -- \
  .claude/agents/agents-index.json \
  pkg/routing/identity_loader.go \
  cmd/gogent-validate/main.go
```

### Decision Trigger
If maintenance exceeds 4 hours/month consistently:
- Re-evaluate Generated Catalog approach (Einstein's Approach 1)
- Consider reducing agent count or consolidating similar agents

---

## Cleanup After Validation

Once assumptions are validated, remove test agents:
```bash
rm -rf .claude/agents/test-frontmatter/
rm -rf .claude/agents/restricted-test/
```

---

## Summary

| Assumption | Status | Action |
|------------|--------|--------|
| Unknown frontmatter fields tolerated | PENDING | Manual test |
| Pipe mode ignores agent definitions | PENDING | Manual test |
| Maintenance <2h/month | TRACKING | Setup monthly log |

Run manual tests, update this doc with results, then delete test agents.
