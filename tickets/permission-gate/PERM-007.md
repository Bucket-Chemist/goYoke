---
id: PERM-007
title: "Session Cache Lifecycle"
status: pending
dependencies: [PERM-004]
time_estimate: "30min"
phase: 6
tags: [plan-generated, phase-6, cache, session]
needs_planning: false
---

# PERM-007: Session Cache Lifecycle

## Description

Implement the "Allow for Session" cache and its cleanup. When a user clicks
"Allow for Session" in the permission modal, the hook caches the approval so
subsequent Bash commands auto-allow without prompting. The cache is cleaned up
by `gogent-archive` at session end.

## Acceptance Criteria

- [ ] Cache file: `$XDG_RUNTIME_DIR/gofortress-perm-cache-{session_id}.json`
- [ ] Format: `{"Bash": true}` (tool name → allowed boolean)
- [ ] Hook reads cache BEFORE contacting bridge (fast path)
- [ ] `gogent-archive` deletes cache file at SessionEnd
- [ ] Cache is scoped to session_id — no cross-session leakage
- [ ] Cache file created on first "Allow for Session"; absent by default

## Files

- `cmd/gogent-permission-gate/cache.go` — Read/write cache (part of PERM-004 binary)
- `cmd/gogent-archive/main.go` — Add cache cleanup

## Technical Spec

### Cache Read (in hook)

```go
func readCache(sessionID string) map[string]bool {
    path := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"),
        fmt.Sprintf("gofortress-perm-cache-%s.json", sessionID))
    data, err := os.ReadFile(path)
    if err != nil { return nil }
    var cache map[string]bool
    json.Unmarshal(data, &cache)
    return cache
}
```

### Cache Write (in hook, on allow_session response)

```go
func writeCache(sessionID, toolName string) {
    cache := readCache(sessionID)
    if cache == nil { cache = make(map[string]bool) }
    cache[toolName] = true
    data, _ := json.Marshal(cache)
    path := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"),
        fmt.Sprintf("gofortress-perm-cache-%s.json", sessionID))
    os.WriteFile(path, data, 0600)
}
```

### Cleanup (in gogent-archive)

```go
// During session end cleanup
pattern := filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "gofortress-perm-cache-*.json")
matches, _ := filepath.Glob(pattern)
for _, m := range matches {
    os.Remove(m)
}
```

## Review Notes

- m-1: Session cache is tool-name-level (not command-pattern-level). Rationale:
  command-pattern matching would require regex/glob engine and is fragile with
  arbitrary commands. Tool-name-level matches Claude Code's own model. Users who
  want per-command control can decline "Allow for Session".
- File permissions: 0600 (user-only read/write)

---

_Generated from: Revised Architecture v2, Phase 6_
