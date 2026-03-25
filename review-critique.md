# Critical Review: GOgent-Fortress spawn_agent Architecture

**Reviewed:** 2026-03-25
**Reviewer:** Staff Architect Critical Review
**Input:** Live codebase analysis of spawn_agent subsystem

---

## Executive Assessment

**Overall Verdict:** CONCERNS

**Confidence Level:** HIGH

- Rationale: Full source code read across all six key files in the spawn_agent data path. Every claim is backed by file + line number. The architecture is not fundamentally unsound but has a critical data path break and multiple code-level divergences between the two MCP server implementations that will cause silent failures.

**Issue Counts:**

- Critical: 2 (must fix)
- Major: 4 (should fix)
- Minor: 3 (consider fixing)

**Commendations:** 5

**Summary:** The spawn_agent architecture is well-designed in concept -- a sidecar MCP server communicating with the TUI via Unix domain socket is the right pattern. The implementation has serious execution gaps: the agent tree sidebar is architecturally broken (bridge delivers messages but the handler never writes to the registry), and the two MCP server implementations have diverged in output format, validation logic, and agent loading, creating a maintenance trap. The stale binary problem was fixed today but the Makefile still outputs the TUI binary to the project root while MCP goes to bin/, guaranteeing recurrence.

**Go/No-Go Recommendation:**
If contractor hours were on the line Monday, I would say: Fix C-1 (agent tree registration) and C-2 (binary output path) before any further feature work. Both are surgical fixes (under 20 lines each). The remaining major issues are real debt but not blocking.

---

## Issue Register

### Critical Issues (Must Fix Before Proceeding)

| ID  | Layer         | Location                                        | Issue                                      | Impact                                             | Recommendation                                       |
| --- | ------------- | ----------------------------------------------- | ------------------------------------------ | -------------------------------------------------- | ---------------------------------------------------- |
| C-1 | Failure Modes | `internal/tui/model/ui_event_handlers.go:176`   | Agent tree never receives MCP-spawned data | Agent sidebar is permanently empty for spawn_agent | Add `registry.Register()` call in handler            |
| C-2 | Dependencies  | `Makefile:109` vs `Makefile:114`                | TUI binary outputs to root, MCP to bin/    | Stale binary problem will recur                    | Unify all outputs to bin/                            |

**Detail for C-1: Agent tree sidebar is architecturally broken for MCP-spawned agents**

The data path is:

1. `handleSpawnAgent()` in `internal/tui/mcp/tools.go:605` calls `uds.notify(TypeAgentRegister, ...)` -- WORKS
2. Bridge `handleAgentRegister()` in `internal/tui/bridge/server.go:231` sends `model.AgentRegisteredMsg{...}` via `p.Send()` -- WORKS
3. `AppModel.Update()` in `internal/tui/model/app.go:350` matches `case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:` and calls `handleAgentRegistryMsg()` -- WORKS
4. `handleAgentRegistryMsg()` at `internal/tui/model/ui_event_handlers.go:176` -- **BROKEN**

The handler:
```go
func (m AppModel) handleAgentRegistryMsg() (tea.Model, tea.Cmd) {
    if m.shared.agentRegistry != nil {
        m.shared.agentRegistry.InvalidateTreeCache()
        m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
        m.statusLine.AgentCount = m.shared.agentRegistry.Count().Total
    }
    return m, nil
}
```

This refreshes the tree view **but never calls `registry.Register()`** with the agent data from `AgentRegisteredMsg`. The message arrives, but the `AgentID`, `AgentType`, and `ParentID` fields are silently discarded. The tree refresh then shows nothing new because nothing was written to the registry.

Compare with the Task-based path which DOES work: `SyncAssistantEvent()` in `internal/tui/cli/agent_sync.go:54` explicitly calls `registry.Register(agent)` when it sees a Task tool_use block.

**Fix:** Change `handleAgentRegistryMsg()` to a type-switch that processes each message type:

```go
func (m AppModel) handleAgentRegistryMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
    if m.shared.agentRegistry == nil {
        return m, nil
    }
    switch msg := msg.(type) {
    case AgentRegisteredMsg:
        _ = m.shared.agentRegistry.Register(state.Agent{
            ID:        msg.AgentID,
            AgentType: msg.AgentType,
            ParentID:  msg.ParentID,
            Status:    state.StatusRunning,
            StartedAt: time.Now(),
        })
    case AgentUpdatedMsg:
        _ = m.shared.agentRegistry.Update(msg.AgentID, func(a *state.Agent) {
            a.Status = state.StatusFromString(msg.Status)
        })
    case AgentActivityMsg:
        m.shared.agentRegistry.SetActivity(msg.AgentID, state.AgentActivity{
            Type:      "tool_use",
            Target:    msg.ToolName,
            Timestamp: time.Now(),
        })
    }
    m.shared.agentRegistry.InvalidateTreeCache()
    m.agentTree.SetNodes(m.shared.agentRegistry.Tree())
    m.statusLine.AgentCount = m.shared.agentRegistry.Count().Total
    return m, nil
}
```

This also requires updating the call site in `app.go:350-351` to pass the message:
```go
case AgentRegisteredMsg, AgentUpdatedMsg, AgentActivityMsg:
    return m.handleAgentRegistryMsg(msg)
```

**Detail for C-2: Binary output paths are inconsistent, guaranteeing stale binary recurrence**

From `Makefile`:
```makefile
# Line 109: TUI goes to PROJECT ROOT
build-go-tui:
    @go build ... -o gofortress ./cmd/gofortress

# Line 114: MCP goes to bin/
build-go-mcp:
    @go build -o bin/gofortress-mcp ./cmd/gofortress-mcp
```

Meanwhile, `findMCPBinary()` in `cmd/gofortress/main.go:412` searches relative to `os.Executable()`:
```go
candidates := []string{
    filepath.Join(exeDir, "gofortress-mcp"),              // same dir as TUI binary
    filepath.Join(exeDir, "bin", "gofortress-mcp"),       // bin/ subdir
    filepath.Join(exeDir, "..", "bin", "gofortress-mcp"), // parent/bin (dev layout)
}
```

When TUI is at `/project/gofortress` and MCP is at `/project/bin/gofortress-mcp`, the FIRST candidate is `/project/gofortress-mcp` which matches a stale binary if one exists. This is exactly what happened today.

**Fix:** Change `build-go-tui` to output to `bin/gofortress`:
```makefile
build-go-tui:
    @go build ... -o bin/gofortress ./cmd/gofortress
```

And add to `.gitignore`:
```
/gofortress
```

And add a clean target that removes the stale root binary:
```makefile
clean-stale:
    @rm -f gofortress gofortress-mcp
```

---

### Major Issues (Should Fix, Can Proceed with Caution)

| ID  | Layer          | Location                                                     | Issue                                            | Impact                                           | Recommendation                                    |
| --- | -------------- | ------------------------------------------------------------ | ------------------------------------------------ | ------------------------------------------------ | ------------------------------------------------- |
| M-1 | Assumptions    | `internal/tui/mcp/tools.go:872` vs `cmd/gofortress-mcp-standalone/spawner.go:76` | Two spawners use different output formats       | Standalone uses stream-json, interactive uses json | Converge both to `json` for -p mode               |
| M-2 | Architecture   | `internal/tui/mcp/tools.go` vs `cmd/gofortress-mcp-standalone/tools.go` | Duplicated spawner code with diverging semantics | Maintenance nightmare, bugs fixed in one not other | Extract shared spawner package                    |
| M-3 | Architecture   | `cmd/gofortress-mcp-standalone/tools.go:160-167`              | Standalone validates spawned_by/can_spawn; interactive does not | Security model inconsistency                     | Add relationship validation to interactive spawner |
| M-4 | Testing        | All spawn_agent paths                                        | No integration test for MCP spawn_agent end-to-end | Regressions caught only by user testing          | Add E2E test with mock claude binary              |

**Detail for M-1: Output format divergence between the two spawners**

The interactive spawner in `internal/tui/mcp/tools.go:872`:
```go
args := []string{"-p", "--output-format", "json", "--permission-mode", "bypassPermissions"}
```

The standalone spawner in `cmd/gofortress-mcp-standalone/spawner.go:76`:
```go
args := []string{"-p", "--output-format", "stream-json", "--verbose", "--permission-mode", "bypassPermissions"}
```

And `gogent-team-run/spawn.go:499`:
```go
args := []string{"-p", "--output-format", "stream-json"}
```

The interactive spawner was RECENTLY changed to `json` (this was one of today's fixes -- stream-json was hanging). But the standalone spawner still uses `stream-json` with `--verbose`. The `parseCLIOutput()` functions in both handle both formats, so this works today, but:

1. The comment in `tools.go:865-869` documents WHY `json` is correct for one-shot `-p` mode
2. The standalone spawner ignores this reasoning and still uses the harder-to-parse format
3. A future change to one parser won't be applied to the other

**Detail for M-2: Code duplication between the two MCP servers**

Files with near-identical code:

| File                                         | Lines | What it does              |
| -------------------------------------------- | ----- | ------------------------- |
| `internal/tui/mcp/spawner.go`                | 275   | runSubprocess, parseCLI   |
| `cmd/gofortress-mcp-standalone/spawner.go`   | 327   | runSubprocess, parseCLI   |
| `internal/tui/mcp/tools.go` (SpawnAgentInput)| ~30   | Input type definition     |
| `cmd/gofortress-mcp-standalone/tools.go`     | ~30   | Input type definition     |

The spawner files are 80% identical. But they've already diverged:
- Interactive uses `routing.LoadAgentIndex()` (uncached); standalone uses `routing.LoadAgentsIndexCached()` (cached)
- Interactive skips relationship validation; standalone has it at line 160-167
- Interactive uses `--output-format json`; standalone uses `--output-format stream-json --verbose`

This is textbook "distributed monolith" code duplication. Both binaries link the same `pkg/routing` package, so a shared `pkg/spawn` package is trivial to extract.

**Detail for M-3: Relationship validation missing from interactive spawner**

The standalone spawner validates the agent relationship:
```go
// cmd/gofortress-mcp-standalone/tools.go:159-167
vr := validateRelationship(index, parentType, input.Agent, input.CallerType)
if !vr.Valid {
    return nil, SpawnAgentOutput{
        Agent:   input.Agent,
        Success: false,
        Error:   "spawn validation failed: " + strings.Join(vr.Errors, "; "),
    }, nil
}
```

The interactive spawner at `internal/tui/mcp/tools.go:558-670` has NO relationship validation. Any agent can spawn any other agent. This means the `spawned_by`/`can_spawn` constraints in `agents-index.json` are only enforced when using the standalone binary, not when using the TUI.

---

### Minor Issues (Consider Addressing)

| ID  | Layer        | Location                                      | Issue                                          | Impact                          | Recommendation                    |
| --- | ------------ | --------------------------------------------- | ---------------------------------------------- | ------------------------------- | --------------------------------- |
| m-1 | Assumptions  | `internal/tui/mcp/tools.go:611`                | `BuildFullAgentContext` called with nil requirements | Context requirements not loaded from agent config | Pass `agent.ContextRequirements`  |
| m-2 | Contractor   | CLAUDE.md MCP server description               | CLAUDE.md says "gofortress-interactive" is TS   | LLM is confused about the architecture           | Update CLAUDE.md to reflect Go implementation |
| m-3 | Architecture | `internal/tui/mcp/tools.go:605`                | `AgentRegisterPayload.ParentID` is never set    | Tree shows flat list, no hierarchy               | Set ParentID from GOGENT_PARENT_AGENT env var |

**Detail for m-1: Context requirements not passed through**

At `internal/tui/mcp/tools.go:611`:
```go
augmented, err := routing.BuildFullAgentContext(agent.ID, nil, nil, input.Prompt)
```

The second argument is `nil` but should be `agent.ContextRequirements`:
```go
augmented, err := routing.BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, input.Prompt)
```

The standalone version at `cmd/gofortress-mcp-standalone/tools.go:176` does this correctly:
```go
augmented, err := routing.BuildFullAgentContext(agent.ID, agent.ContextRequirements, nil, input.Prompt)
```

**Detail for m-2: CLAUDE.md is factually wrong about the MCP server**

CLAUDE.md currently says:
```
**`gofortress-interactive`** (TS, runs inside TUI process):
- Primary spawn_agent with `buildFullAgentContext()`, relationship validation, Zustand store, cost tracking
```

This is wrong on every count:
1. It's Go, not TS
2. It runs as a SEPARATE process, not inside the TUI
3. It uses `BuildFullAgentContext()` (Go), not `buildFullAgentContext()` (TS)
4. It does NOT have relationship validation (see M-3)
5. There is no Zustand store (that's the TS TUI, which is deprecated/parallel)
6. Cost tracking is passive (extracted from CLI output), not active

This actively misleads the LLM into generating incorrect code and expecting features that don't exist.

**Detail for m-3: Agent tree hierarchy is flat**

`handleSpawnAgent()` sends:
```go
uds.notify(TypeAgentRegister, AgentRegisterPayload{
    AgentID:   agentID,
    AgentType: agent.ID,
})
```

`ParentID` is never set. The `AgentRegisterPayload` struct supports it:
```go
type AgentRegisterPayload struct {
    AgentID   string `json:"agentId"`
    AgentType string `json:"agentType"`
    ParentID  string `json:"parentId,omitempty"`
}
```

The fix is to use `GOGENT_PARENT_AGENT` environment variable (set by `buildSpawnEnv()` for nested spawns) or the current agent's ID.

---

## Assumption Register

| #   | Assumption                                                  | Source                       | Verified? | Risk if False                                | Mitigation                                    |
| --- | ----------------------------------------------------------- | ---------------------------- | --------- | -------------------------------------------- | --------------------------------------------- |
| A-1 | claude CLI supports `--output-format json` for -p mode      | tools.go:865 comment         | Yes       | Spawned agents hang                          | Verified in today's fixes                     |
| A-2 | `--permission-mode bypassPermissions` is a valid flag       | tools.go:872                 | Unverified| Spawned agents cannot write files            | Test with `claude -p --permission-mode bypassPermissions` |
| A-3 | `--verbose` is NOT needed for json output format            | tools.go:865-869 comment     | Unverified| If needed, spawned agents may fail silently  | Test without --verbose                        |
| A-4 | GOFORTRESS_SOCKET propagates to MCP server subprocess       | main.go:315-319              | Yes       | MCP server can't connect to TUI             | setenv before subprocess start                |
| A-5 | UDS connection survives for full agent lifecycle (10min)     | tools.go:95                  | Likely    | Long agents lose UDS, notifications dropped  | Monitor for connection reset errors           |
| A-6 | agents-index.json v2.6.0 is present on all machines         | agents.go:12                 | Project-specific | Hard crash at runtime                  | Version validation exists (good)              |
| A-7 | `--allowedTools` flag accepts comma-separated values        | tools.go:887                 | Yes       | Agent gets wrong tool set                    | Claude CLI docs confirm                       |

---

## Commendations

1. **Sidecar MCP architecture is correct.** Having the MCP server as a separate Go binary that communicates with the TUI via Unix domain socket is the right pattern. It cleanly separates the MCP protocol handling from the TUI event loop, avoids blocking the UI during agent spawns, and allows the MCP server to outlive or be replaced independently of the TUI.

2. **Subprocess lifecycle management is well-implemented.** The `runSubprocess()` function in `internal/tui/mcp/spawner.go` handles SIGTERM -> SIGKILL escalation, process group isolation via `Setsid`, stdout buffer limits (10MB), stderr capture, and graceful timeout management. This is production-quality process management.

3. **Error handling follows the soft-error pattern correctly.** Subprocess failures are returned as `SpawnAgentOutput{Success: false, Error: msg}` rather than Go errors. This ensures the MCP protocol doesn't see a tool invocation failure -- the caller gets structured data about what went wrong. This matches the MCP spec's intent.

4. **Double-injection prevention is thorough.** Both `BuildFullAgentContext()` and `BuildAugmentedPrompt()` check for marker strings before injecting, preventing context bloat on re-spawns. This is a common pitfall in agent-spawning systems and it's handled well.

5. **IPC bridge shutdown handling is robust.** The `IPCBridge.Shutdown()` in `internal/tui/bridge/server.go:308-335` correctly handles the tricky edge cases: it signals blocked modal handlers via `close(b.done)`, drains the pending modal map with non-blocking sends to avoid double-close panics, and removes the socket file. This avoids both goroutine leaks and deadlocks.

---

## Architecture Assessment: Is This Over-Engineered?

**No. The complexity is justified, but the execution has gaps.**

The spawn_agent architecture involves 4 processes:
```
User -> Go TUI (Bubbletea) -> claude CLI (stream-json) -> gofortress-mcp (MCP stdio) -> claude -p (agent)
```

This looks like a lot, but each layer is necessary:
- **Go TUI**: Renders the terminal UI, manages the session
- **claude CLI**: Anthropic's runtime -- can't be replaced
- **gofortress-mcp**: MCP tools (spawn_agent, ask_user, etc.) -- must be a separate process because MCP uses stdio
- **agent claude -p**: The actual agent work -- must be a separate process because it's a different session

What IS over-engineered is having TWO MCP server implementations (`gofortress-mcp` and `gofortress-mcp-standalone`) with duplicated spawner code. The standalone binary should either:
1. Share a common `pkg/spawn` package with the interactive version, OR
2. Be eliminated entirely if the Go TUI is the primary interface

The UDS bridge is also correct -- the MCP server runs as a subprocess of the claude CLI (because that's how MCP servers work), so it can't directly call into the TUI. The UDS is the only viable IPC mechanism that doesn't require the MCP server to know about Bubbletea.

---

## What's Still Broken After Today's Fixes

### Definitely Broken

1. **Agent tree sidebar shows nothing for MCP-spawned agents** (C-1). The bridge delivers `AgentRegisteredMsg` but `handleAgentRegistryMsg()` never writes the data to the registry. This is a data path break, not a rendering issue.

2. **CLAUDE.md describes the wrong architecture** (m-2). The documentation says `gofortress-interactive` is a TypeScript implementation inside the TUI process with Zustand and relationship validation. None of that is true for the Go implementation.

3. **Context requirements not injected for interactive spawner** (m-1). The interactive spawner passes `nil` for context requirements, so agent conventions and rules are partially missing from the augmented prompt.

### Probably Still Broken

4. **Stale binary in project root** (`/home/doktersmol/Documents/GOgent-Fortress/gofortress-mcp` exists alongside `/home/doktersmol/Documents/GOgent-Fortress/bin/gofortress-mcp`). The Glob results confirm both exist. If the TUI binary is in the project root, `findMCPBinary()` will find the stale root one first.

5. **Relationship validation only in standalone** (M-3). Any agent can spawn any other agent through the interactive MCP server, bypassing `spawned_by`/`can_spawn` constraints.

### Likely Working

6. **Basic spawn_agent flow** -- agent spawning, output collection, and cost extraction appear correct after today's fixes to output format and permission mode.

7. **Interactive tools (ask_user, confirm_action)** -- the UDS bridge handles modal request/response flow correctly.

---

## Binary Management Assessment

The binary layout is fragmented:

| Binary                     | Built by           | Output location            | Found by                    |
| -------------------------- | ------------------ | -------------------------- | --------------------------- |
| `gofortress` (TUI)         | `build-go-tui`     | `./gofortress` (root)      | launcher script             |
| `gofortress-mcp`           | `build-go-mcp`     | `bin/gofortress-mcp`       | `findMCPBinary()`           |
| `gofortress-mcp-standalone`| manual              | `bin/gofortress-mcp-standalone` | settings.json          |
| `gofortress-mcp` (stale)   | previous build      | `./gofortress-mcp` (root)  | `findMCPBinary()` candidate |

The root cause of today's stale binary problem: `build-go-tui` outputs to the project root, `build-go-mcp` outputs to bin/. The `findMCPBinary()` search starts from the TUI binary's directory, so if the TUI is in root and a stale MCP binary is also in root, it finds the stale one first.

The `--mcp-binary` flag exists as an escape hatch and works correctly, but requiring users to know about it defeats the purpose of auto-discovery.

---

## MCP Server Naming Confusion

There are three names in play:

| Name                     | Where used                                              | What it refers to          |
| ------------------------ | ------------------------------------------------------- | -------------------------- |
| `gofortress-interactive` | `writeMCPConfig()` in main.go:453, CLAUDE.md            | Go MCP server in TUI       |
| `gofortress-standalone`  | settings.json:140                                       | Go standalone MCP server   |
| `gofortress-mcp`         | Binary name, MCP Implementation.Name, Makefile          | Same as gofortress-interactive |

The confusion: the BINARY is called `gofortress-mcp` but is REGISTERED as `gofortress-interactive`. This was one of today's bugs (the old name was just `gofortress`, causing tool-not-found errors because CLAUDE.md told the LLM to call `mcp__gofortress-interactive__spawn_agent`).

The naming is now CONSISTENT between `writeMCPConfig()` and CLAUDE.md, but the binary name `gofortress-mcp` doesn't match the registration name `gofortress-interactive`. This is confusing but not broken -- MCP allows arbitrary server names independent of binary names.

---

## Permission Model for Spawned Agents

Current: `--permission-mode bypassPermissions` on all spawned agents (tools.go:872).

This is CORRECT for `-p` (print/pipe) mode because there's no interactive terminal to approve permissions. Without it, any Write/Edit/Bash call would block forever waiting for user approval that can never arrive.

Risk: if `bypassPermissions` is not a valid claude CLI flag (or its behavior changes), all spawned agents will silently fail on write operations. The fix applied today (adding `--permission-mode`) addressed the symptom; the underlying assumption (A-2) should be verified.

---

## Agent Tree Sidebar Feasibility

**Can the agent tree work with the current IPC architecture?** Yes, once C-1 is fixed.

The infrastructure is all in place:
- `AgentRegisterPayload` has the right fields (agentId, agentType, parentId)
- `IPCBridge.handleAgentRegister()` correctly sends `AgentRegisteredMsg` via `p.Send()`
- `state.AgentRegistry` has `Register()`, `Update()`, `SetActivity()`, `Tree()`
- The tree component (`agentTree.SetNodes()`) and detail panel (`agentDetail.SetAgent()`) are wired

The ONLY gap is the handler at `ui_event_handlers.go:176` which discards the message data instead of processing it. Fix that, and the tree should populate.

For hierarchical display (parent-child relationships), m-3 also needs fixing (set `ParentID` in the register payload).

---

## Recommendations

### High Priority (Fix This Week)

1. **C-1: Fix `handleAgentRegistryMsg()` to actually register agents from bridge messages.** Surgical fix: ~15 lines of code. Without this, the agent tree sidebar is useless and users have no visibility into spawned agents.

2. **C-2: Unify binary output paths in Makefile.** Change `build-go-tui` to output to `bin/gofortress`. Delete stale `./gofortress-mcp` from project root. Add both to `.gitignore`.

3. **m-1: Pass `agent.ContextRequirements` to `BuildFullAgentContext()` in interactive spawner.** One-word fix: change `nil` to `agent.ContextRequirements`.

### Medium Priority (Fix This Sprint)

4. **M-1: Converge standalone spawner to `--output-format json` for `-p` mode.** Both spawners should use the same format. The interactive spawner's reasoning (in the comment at tools.go:865) is correct.

5. **M-3: Add relationship validation to interactive spawner.** Copy the `validateRelationship()` call from the standalone spawner. Without this, the `spawned_by`/`can_spawn` configuration is security theater.

6. **m-2: Update CLAUDE.md to reflect actual Go MCP server architecture.** The current description is factually wrong and actively causes the LLM to generate incorrect code.

### Low Priority (Next Iteration)

7. **M-2: Extract shared `pkg/spawn` package.** The spawner code is already 80% identical. Factor out `runSubprocess()`, `parseCLIOutput()`, `buildSpawnArgs()`, `buildSpawnEnv()`, `validateNestingDepth()` into a shared package. Both MCP servers import it.

8. **M-4: Add integration test for MCP spawn_agent.** Create a mock `claude` binary that echoes a known result JSON, then test the full path: MCP tool call -> handleSpawnAgent -> runSubprocess -> parseCLIOutput -> SpawnAgentOutput.

9. **m-3: Set ParentID in AgentRegisterPayload.** Read `GOGENT_PARENT_AGENT` from environment to populate the parent agent ID for tree hierarchy.

---

## Final Sign-Off

**Reviewed By:** Staff Architect Critical Review
**Review Date:** 2026-03-25
**Review Duration:** ~25 minutes (including source code analysis)

**Conditions for Approval:**

- [ ] C-1: `handleAgentRegistryMsg()` processes message data and calls `registry.Register()`
- [ ] C-2: All Go binaries output to `bin/` directory; stale root binaries removed

**Recommended Actions:**

1. Fix C-1 and C-2 immediately (both are < 20 lines)
2. Fix m-1 (one word change) as part of the C-1 commit
3. Update CLAUDE.md (m-2) to prevent future LLM confusion
4. Add relationship validation (M-3) in the next sprint
5. Plan shared spawner extraction (M-2) as a refactoring ticket

**Post-Approval Monitoring:**

- After C-1 fix: Verify agent tree sidebar shows spawned agents by triggering a spawn_agent call through the TUI
- After C-2 fix: Run `make build-go` and verify no binaries appear in the project root
- Watch for: UDS connection timeouts on long-running agents (>5 minutes)
- Watch for: `parseCLIOutput` failures if claude CLI output format changes
