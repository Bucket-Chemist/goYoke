# GO Implementation Checklist

## Overview

This checklist tracks the implementation of the GO migration as outlined in the research documentation. Each phase builds on the previous, with clear dependencies and verification steps.

---

## Phase 0: Foundation (COMPLETED ✅)

### Conventions
- [x] Create `~/.claude/conventions/go.md` - Core GO conventions
- [x] Create `~/.claude/conventions/go-cobra.md` - Cobra CLI patterns
- [x] Create `~/.claude/conventions/go-bubbletea.md` - Bubbletea TUI patterns

### Agents
- [x] Create `~/.claude/agents/go-pro/` - General GO agent
- [x] Create `~/.claude/agents/go-cli/` - Cobra CLI specialist
- [x] Create `~/.claude/agents/go-tui/` - Bubbletea TUI specialist
- [x] Create `~/.claude/agents/go-api/` - HTTP client specialist
- [x] Create `~/.claude/agents/go-concurrent/` - Concurrency specialist

### Schema Updates
- [x] Update `routing-schema.json` with GO agents
  - [x] Add GO patterns to sonnet tier
  - [x] Add agent_subagent_mapping entries
  - [x] Add GO agents to subagent_types general-purpose list
  - [x] Fix GAP-008: Protocol-specific model selection for gemini

- [x] Update `agents-index.json` with GO agents
  - [x] Add go-pro, go-cli, go-tui, go-api, go-concurrent entries
  - [x] Add auto_activate patterns
  - [x] Add model_tiers updates

### Documentation
- [x] Create `GO_AGENTS_USAGE_GUIDE.md`
- [x] Update `init-auto/SKILL.md` with GO detection
- [x] Create this implementation checklist

---

## Phase 1: Verify Environment (Week 1)

### Claude Code Headless Test
- [x] Run: `claude --help 2>&1 | grep -i headless` (Found `-p` for non-interactive output)
- [x] Test: `timeout 30 claude -p "Say hello" 2>&1` (Confirmed `-p` works for headless)
- [x] Test: Tmux session with Claude
- [x] Document results in `~/.claude/tmp/headless-test-results.md`

**Decision Gate:** PASSED (Headless verified via `-p` flag)

### GO Environment Setup
- [ ] Verify GO installation: `go version` (require 1.22+)
- [ ] Create project: `go mod init github.com/yourname/lisan-al-gaib`
- [ ] Add dependencies:
  ```bash
  go get github.com/spf13/cobra
  go get github.com/spf13/viper
  go get github.com/charmbracelet/bubbletea
  go get github.com/charmbracelet/lipgloss
  go get golang.org/x/sync/errgroup
  ```
- [ ] Verify build: `go build ./...`

### Beads Installation
- [ ] Install beads in ~/.claude/: `bd init`
- [ ] Test basic operations:
  ```bash
  bd create "Test task" -t task -p 1 --json
  bd list --json
  bd close $(bd list --json | jq -r '.[0].id') --reason "Test"
  ```
- [ ] Create memory wrapper scripts (from IMPLEMENTATION_PRIORITIES.md P0.4)

---

## Phase 2: Minimal TUI (Week 2-3)

### Python TUI (Fast Path)
- [ ] Install textual: `pip install textual --break-system-packages`
- [ ] Create `~/.claude/tui/main.py` with status display
- [ ] Add keybindings: q=quit, r=refresh
- [ ] Test with mock agent data

### GO TUI (Parallel Track)
- [ ] Create `cmd/lisan-tui/main.go`
- [ ] Implement Model struct with agent list
- [ ] Implement View with Lipgloss styling
- [ ] Implement Update with keyboard handling
- [ ] Test: `go run cmd/lisan-tui/main.go`

### TUI Features Checklist
- [ ] Status bar (ready/working/error indicator)
- [ ] Agent list with progress bars
- [ ] Convoy/task grouping display
- [ ] Cost tracking (daily spend)
- [ ] Real-time refresh (1s interval)
- [ ] Keyboard navigation

---

## Phase 3: lisan CLI v0.1 (Month 2)

### Core Commands
- [ ] `lisan list` - Show active agents
- [ ] `lisan hook create <id>` - Create hook directory
- [ ] `lisan hook read <id>` - Read current_work.json
- [ ] `lisan status` - Overall system status

### Session Management
- [ ] `lisan session start <agent>` - Spawn tmux session
- [ ] `lisan session stop <id>` - Kill session
- [ ] `lisan session attach <id>` - Attach to session

### Integration
- [ ] Update orchestrator to call lisan CLI
- [ ] Test session spawning with go-pro agent
- [ ] Verify hook persistence across restarts

---

## Phase 4: Beads Integration (Month 2)

### Agent Updates
- [ ] Update memory-archivist for dual-write (beads + markdown)
- [ ] Update python-pro to query gotchas before work
- [ ] Update orchestrator to check `bd ready` at session start

### Workflow Testing
- [ ] Test: Task creation → completion → archival
- [ ] Test: Dependency blocking (`bd dep add`)
- [ ] Test: Work discovery (`bd ready --json`)
- [ ] Test: Gotcha creation and retrieval

---

## Phase 5: GUPP Implementation (Month 2-3)

### Hook Persistence
- [ ] Create hook directory structure
- [ ] Write `current_work.json` on spawn
- [ ] Read and resume on restart
- [ ] Test: Kill session, restart, verify resume

### Crash Recovery
- [ ] Test: Force kill during work
- [ ] Verify hook file persists
- [ ] Verify GUPP activates on restart

---

## Phase 6: Parallel Spawning (Month 3)

### Multi-Agent Support
- [ ] `lisan spawn --background` implementation
- [ ] `lisan wait --any/--all` implementation
- [ ] Test: 3+ agents in parallel
- [ ] Verify context isolation (no inheritance)

### Witness Daemon
- [ ] Implement stuck detection
- [ ] Implement alerting
- [ ] Test: Agent idle for 30+ minutes
- [ ] Verify escalation to orchestrator

---

## Phase 7: Full GO Migration (Month 4+)

### Routing Engine Port
- [ ] Port routing logic to GO
- [ ] Port complexity scoring
- [ ] Port tier enforcement
- [ ] Validate: Compare GO routing vs Python routing

### Hook Enforcement Port
- [ ] Port validate-routing.sh to GO
- [ ] Port sharp-edge-detector.sh to GO
- [ ] Port session-archive.sh to GO

### Deprecation
- [ ] Add deprecation warnings to Python tools
- [ ] 4-week stability testing
- [ ] Remove Python dependencies (if stable)

---

## Verification Commands

### After Phase 0 (Foundation)
```bash
# Verify conventions exist
ls ~/.claude/conventions/go*.md

# Verify agents exist
ls -la ~/.claude/agents/go-*/

# Verify schema updated
jq '.agent_subagent_mapping | keys | map(select(startswith("go")))' ~/.claude/routing-schema.json

# Verify agents index
jq '.agents | map(select(.id | startswith("go"))) | .[].id' ~/.claude/agents/agents-index.json
```

### After Phase 1 (Environment)
```bash
# Verify GO
go version

# Verify beads
bd list --json

# Verify Claude headless (if available)
cat ~/.claude/tmp/headless-test-results.md
```

### After Phase 2 (TUI)
```bash
# Test Python TUI
python ~/.claude/tui/main.py

# Test GO TUI
go run cmd/lisan-tui/main.go
```

### After Phase 3 (lisan CLI)
```bash
# Test lisan commands
lisan list
lisan hook create test-agent
lisan hook read test-agent
lisan status
```

---

## Rollback Points

| Phase | Rollback Command | What You Lose |
|-------|------------------|---------------|
| P0 | N/A (additive) | Nothing |
| P1 | `rm -rf ~/.claude/.beads/` | Beads integration |
| P2 | `rm -rf ~/.claude/tui/` | TUI (Python) |
| P3 | `rm $(which lisan)` | lisan CLI |
| P4-6 | Disable lisan, re-enable Task() | Advanced coordination |
| P7 | `git checkout v2.1.0` | GO migration (full revert) |

---

## Success Criteria

### Phase 0 (Foundation)
- [ ] GO agents respond to triggers
- [ ] Conventions load correctly
- [ ] Schema validates without errors

### Phase 2 (TUI)
- [ ] TUI displays agent status
- [ ] Refresh works without crash
- [ ] Can quit cleanly with 'q'

### Phase 3 (lisan)
- [ ] lisan spawns tmux sessions
- [ ] Hooks persist across restarts
- [ ] Integration with Python orchestrator works

### Phase 5 (GUPP)
- [ ] Agent resumes work after crash 100% of time
- [ ] No data loss on unexpected termination

### Phase 6 (Parallel)
- [ ] 3+ agents run concurrently
- [ ] Context isolation verified (no 60K inheritance)
- [ ] Stuck detection catches idle agents

### Phase 7 (Full Migration)
- [ ] All routing in GO
- [ ] 4 weeks stable production use
- [ ] Python dependencies removed

---

## Current Status

**Phase:** 0 (Foundation) - COMPLETING
**Next Action:** Apply schema patch, verify agent activation
**Blockers:** None
**Risk Level:** Low

---

## Notes

- Each phase is independently valuable
- Stop at any phase if priorities change
- Refer to MIGRATION_RISKS_REVIEW.md for detailed risk analysis
- Refer to IMPLEMENTATION_PRIORITIES.md for P0-P3 details
- Refer to TUI_DESIGN_PRINCIPLES.md for TUI specifications
