# TUI Implementation - Ticket Overview

> **Status:** Ready for Implementation
> **Total Estimated Hours:** 26.0
> **Last Updated:** 2026-01-26
> **Architecture:** Einstein Analysis (GAP-TUI-003)

---

## Quick Navigation

| Phase | Tickets | Hours |
|-------|---------|-------|
| [Infrastructure](#phase-0-infrastructure) | TUI-INFRA-01 | 2.0 |
| [Foundation](#phase-1-foundation) | TUI-CLI-01, TUI-CLI-01a, TUI-PERF-01, TUI-TELEM-01 | 8.0 |
| [Event System](#phase-2-event-system) | TUI-CLI-02 | 1.5 |
| [Agent Tree](#phase-3-agent-tree) | TUI-AGENT-01, TUI-AGENT-02, TUI-AGENT-03 | 6.5 |
| [Integration](#phase-4-integration) | TUI-CLI-03, TUI-CLI-04, TUI-MAIN-01 | 7.5 |
| [Session](#phase-5-session) | TUI-CLI-05 | 1.5 |
| **Total** | | **26.0** |

---

## Visual Layout Target

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ [1] Claude  [2] Agents  [3] Stats  [4] Query    Session: abc │ Cost: $0.34 │
├────────────────────────────────────────────────────┬────────────────────────┤
│                                                    │ Agent Delegation Tree  │
│  Claude Conversation (70%)                         │ (30%)                  │
│  ┌──────────────────────────────────────────────┐  │                        │
│  │ You: Implement the TUI panel                 │  │ ▸ terminal             │
│  │                                              │  │   ├─ orchestrator ⏳   │
│  │ Claude: I'll help you implement the TUI...  │  │   │  └─ python-pro ✓   │
│  │ [streaming...]                              │  │   └─ go-tui ⏳        │
│  │                                              │  │      "Implement panel"│
│  └──────────────────────────────────────────────┘  │                        │
│  ┌──────────────────────────────────────────────┐  ├────────────────────────┤
│  │ Hook Events                                  │  │ Selected: go-tui      │
│  │ ✓ gogent-validate (Task → orchestrator)     │  │ Tier: sonnet          │
│  │ ⏳ gogent-validate (Task → go-tui)          │  │ Duration: 2.3s...     │
│  └──────────────────────────────────────────────┘  │ Task: "Implement..."  │
│  ┌──────────────────────────────────────────────┐  │                        │
│  │ > Type your message here...          [Enter] │  │ [Enter] Expand        │
│  └──────────────────────────────────────────────┘  │ [q] Query agent       │
└────────────────────────────────────────────────────┴────────────────────────┘
```

---

## Dependency Graph

```
TUI-INFRA-01 ────────────────────────┐
(Agent Lifecycle)                    │
        │                            │
        ▼                            │
TUI-TELEM-01 ◄───────────────────────┤
(File Watchers)                      │
        │                            │
        ▼                            │
TUI-AGENT-01                         │
(Agent Tree Model)                   │
        │                            │
        ├──────────────────┐         │
        ▼                  ▼         │
TUI-AGENT-02          TUI-PERF-01    │
(Tree View)           (Dashboard)    │
        │                  │         │
        ▼                  │         │
TUI-AGENT-03               │         │
(Detail Sidebar)           │         │
        │                  │         │
        └────────┬─────────┘         │
                 │                   │
TUI-CLI-01 ──────┼───────────────────┘
(Subprocess)     │
    │            │
    ▼            │
TUI-CLI-01a      │
(Auto-Restart)   │
    │            │
    ▼            │
TUI-CLI-02       │
(Events)         │
    │            │
    ▼            │
TUI-CLI-03       │
(Claude Panel)   │
    │            │
    ├────────────┘
    ▼
TUI-CLI-04
(Layout: 70/30)
    │
    ▼
TUI-MAIN-01
(Banner)
    │
    ▼
TUI-CLI-05
(Sessions)
```

---

## Execution Order (Recommended)

### Week 1: Infrastructure + Foundation
1. TUI-INFRA-01 (agent lifecycle telemetry)
2. TUI-CLI-01 (subprocess) + TUI-PERF-01 (shell) in parallel
3. TUI-CLI-01a (auto-restart)
4. TUI-TELEM-01 (file watchers)

### Week 2: Agent Tree + Events
5. TUI-CLI-02 (events)
6. TUI-AGENT-01 (tree model)
7. TUI-AGENT-02 (tree view)
8. TUI-AGENT-03 (detail sidebar)

### Week 3: Integration
9. TUI-CLI-03 (Claude panel)
10. TUI-CLI-04 (70/30 layout)
11. TUI-MAIN-01 (persistent banner)
12. TUI-CLI-05 (session management)

---

## Known Limitations

### Chain Depth Visibility

**Current Constraint:** Claude Code hooks only fire for terminal-level agent delegations. If `orchestrator` spawns `python-pro`, which internally spawns `haiku-scout`, we only see:

```
terminal → orchestrator (visible)
terminal → python-pro (visible, but appears as sibling not child)
           python-pro → haiku-scout (NOT visible - internal to subagent)
```

**Accepted for MVP.** The agent tree will show a flat list of terminal-spawned agents rather than true nested delegation chains.

---

## Future Implementation Ideas

### 1. Agent Status Polling / Messaging

**Concept:** Allow the TUI to send status query messages to running agents and receive progress updates.

**Potential Approaches:**

a) **Named Pipe Communication**
   - Each spawned agent creates a named pipe at `/tmp/gogent-agent-{id}.pipe`
   - TUI can write status query messages
   - Agent responds with current state (tool count, last action, progress %)

b) **Shared Memory / mmap**
   - Agent writes status to shared memory region
   - TUI reads without blocking agent execution
   - Lower latency than file I/O

c) **Unix Domain Sockets**
   - Agent opens socket for bidirectional communication
   - TUI can query state and potentially send control messages (pause, priority change)

d) **Transcript Tailing**
   - Parse agent transcript files in real-time
   - Extract tool calls and progress from JSONL stream
   - Requires transcript path from spawn event

**Implementation Considerations:**
- Must not impact agent performance
- Need protocol for status message format
- Consider agent busy/idle states
- Handle agent crash gracefully

### 2. Nested Agent Visibility

**Concept:** Track internal agent-to-agent delegations within subagent processes.

**Potential Approaches:**

a) **Transcript Parsing**
   - Parse completed transcripts for Task() tool calls
   - Reconstruct delegation tree post-hoc
   - Limited to completed sessions

b) **Agent Instrumentation**
   - Modify agent spawning to emit lifecycle events
   - Requires changes to Claude Code or wrapper scripts

c) **Claude Code API Enhancement**
   - Request nested SubagentStart/SubagentStop events from Anthropic
   - Would provide native support

### 3. Agent Query Interface

**Concept:** Interactive interface to query agents about their current task, ask for summaries, or request priority changes.

**Features:**
- Select agent from tree
- Send natural language query
- Receive response without interrupting main task
- Potentially pause/resume agents

### 4. Cost Projection & Budgets

**Concept:** Real-time cost tracking with alerts and hard stops.

**Features:**
- Per-agent cost tracking
- Session budget limits
- Warning thresholds (50%, 75%, 90%)
- Hard stop option at budget limit

### 5. Performance Flame Graphs

**Concept:** Visual representation of agent execution time as nested flame graph.

**Features:**
- Width = duration
- Depth = delegation chain
- Color = success/failure
- Interactive drill-down

---

## Related Documents

| Document | Purpose |
|----------|---------|
| `GAP-TUI-CLI-EMBEDDING.md` | Original CLI embedding architecture |
| `GAP-TUI-TELEMETRY-EXPANSION.md` | Telemetry wiring requirements |
| `TUI-PERF-INDEX.md` | Performance view specifications |
| `~/.claude/conventions/go-bubbletea.md` | Bubble Tea conventions |

---

**Document Version:** 3.0
**Created:** 2026-01-22
**Last Updated:** 2026-01-26
