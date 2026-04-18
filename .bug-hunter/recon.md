## Architecture Summary
GOgent-Fortress is a Go monorepo centered on a Bubble Tea TUI plus a set of CLI utilities that enforce routing, permissions, session handoff, and transcript analysis. The highest-risk paths are XDG-backed runtime state in `pkg/config`, skill/tool guard enforcement in `cmd/gogent-skill-guard`, and transcript-driven policy logic in `pkg/routing`.

## Risk Map
### HIGH PRIORITY (scanned)
- `pkg/config/paths.go` — runtime/cache directory resolution for guard files, counters, and routing state
- `cmd/gogent-skill-guard/main.go` — policy enforcement falls back to unguarded operation on state path failures
- `pkg/routing/transcript.go` — transcript parsing and background-task analysis use line-oriented scanning on unbounded JSONL
- `pkg/routing/delegation.go` — delegation enforcement fail-opens when transcript parsing fails
- `cmd/gogent-permission-gate/uds.go` — permission bridge I/O and timeout handling
- `internal/tui/bridge/server.go` — UDS bridge startup depends on runtime socket path selection
- `internal/tui/mcp/spawner.go` — subprocess output parsing and activity extraction
- `pkg/session/handoff.go` — session artifact generation reads transcript-derived state

### CONTEXT-ONLY
- `pkg/config/paths_test.go`
- `pkg/routing/transcript_test.go`
- `cmd/gogent-skill-guard/guard_v2_test.go`

## Detected Patterns
- Framework: Go CLI/TUI with Bubble Tea and MCP bridge
- Auth / policy: local guard files, permission-gate UDS requests, transcript-based enforcement
- Storage: XDG runtime/cache/data dirs plus project-local `.gogent/`
- Trust boundaries: env vars (`XDG_RUNTIME_DIR`, `XDG_CACHE_HOME`), transcript JSONL, UDS sockets, spawned subprocess output

## Coverage
Focused audit only. Read coverage included the files above plus `go.mod`, but not the remaining queued domains in `cmd/`, `internal/tui/`, and most of `pkg/`.
