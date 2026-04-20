# Windows Distribution Plan

**Status:** BLOCKED — syscall porting required before binary compiles
**Priority:** P1 (collaborators need it)
**Estimated effort:** 4-6 hours across 2 sessions
**Branch:** `feat/windows-compat`

---

## Problem Statement

The goYoke multicall binary cannot cross-compile to Windows due to POSIX-only
syscall usage across 10+ files. The affected syscalls are:

| Syscall | Used For | Files |
|---------|----------|-------|
| `syscall.Flock` | File locking (atomic counters) | `pkg/config/paths.go` ✅ DONE |
| `syscall.Kill` | Process signaling (agent lifecycle) | 6 files |
| `syscall.Setsid` | Process group isolation | 3 files |
| `syscall.Dup2` | Daemon fd redirection | 1 file |
| `syscall.Setpgid` | Process group control | 1 file |

---

## Affected Files (Complete List)

### Already Fixed
- [x] `pkg/config/paths.go` — flock → `lockFile()`/`unlockFile()` helpers (build tags)

### Needs Fix: syscall.Kill (6 files)

| File | Lines | Context |
|------|-------|---------|
| `internal/tui/state/agent.go` | 520, 564, 580, 582 | Agent SIGTERM→SIGKILL escalation |
| `internal/tui/mcp/spawner.go` | 361, 378 | MCP subprocess lifecycle |
| `internal/tui/mcp/prepare_skill.go` | 194 | Skill guard process check |
| `internal/tui/mcp/handoff.go` | 135 | Handoff process check |
| `internal/subcmd/utils/teamrun/spawn.go` | 403, 446 | Team-run agent kill |
| `internal/subcmd/utils/teamrun/daemon.go` | 241, 243, 267, 268 | Daemon shutdown |

**Windows equivalent:** `cmd.Process.Kill()` or `windows.TerminateProcess()`
Process.Signal(os.Interrupt) sends Ctrl+C on Windows.

### Needs Fix: syscall.Setsid (3 files)

| File | Lines | Context |
|------|-------|---------|
| `internal/tui/mcp/spawner.go` | 245 | `SysProcAttr{Setsid: true}` |
| `internal/tui/mcp/tools.go` | 1008 | MCP tool subprocess isolation |
| `internal/subcmd/utils/teamrun/spawn.go` | 302 | Team-run process group |

**Windows equivalent:** `CREATE_NEW_PROCESS_GROUP` flag in `SysProcAttr.CreationFlags`.
```go
// Windows:
cmd.SysProcAttr = &syscall.SysProcAttr{
    CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
}
```

### Needs Fix: syscall.Dup2 (1 file)

| File | Lines | Context |
|------|-------|---------|
| `internal/subcmd/utils/teamrun/daemon.go` | 105, 109, 131 | Redirect stdout/stderr to file |

**Windows equivalent:** Not applicable in same way. Use `cmd.Stdout = file` / `cmd.Stderr = file` instead of fd duplication. Or skip daemon mode on Windows entirely.

### Needs Fix: syscall.Setpgid (1 file)

| File | Lines | Context |
|------|-------|---------|
| `test/simulation/chaos_runner.go` | 302 | Chaos test process group |

**Windows equivalent:** Same as Setsid — `CREATE_NEW_PROCESS_GROUP`. Test-only code, low priority.

### Needs Fix: syscall.Flock in non-config packages (2 files)

| File | Lines | Context |
|------|-------|---------|
| `internal/tui/mcp/prepare_skill.go` | 72, 91 | Skill guard lock file |
| `internal/hooks/skillguard/guard_v2_test.go` | 42, 43 | Test-only |

**Windows equivalent:** Same `lockFile()`/`unlockFile()` pattern from pkg/config.

---

## Implementation Strategy

### Approach: Build-tag split files

For each affected file, create a `_unix.go` and `_windows.go` pair with platform-specific
implementations of the problematic functions.

**Pattern:**
```
internal/tui/mcp/spawner.go          → common code (no syscall)
internal/tui/mcp/spawner_unix.go     → killProcess(), newProcessGroup()
internal/tui/mcp/spawner_windows.go  → killProcess(), newProcessGroup()
```

### Phase 1: Process Management Helpers (blocks everything)

Create shared helper package `pkg/process/`:

```go
// pkg/process/kill_unix.go
//go:build !windows

package process

import "syscall"

func Kill(pid int, sig os.Signal) error {
    return syscall.Kill(pid, sig.(syscall.Signal))
}

func NewProcessGroupAttr() *syscall.SysProcAttr {
    return &syscall.SysProcAttr{Setsid: true}
}
```

```go
// pkg/process/kill_windows.go
//go:build windows

package process

import "os"

func Kill(pid int, sig os.Signal) error {
    p, err := os.FindProcess(pid)
    if err != nil {
        return err
    }
    return p.Kill()  // Windows only supports Kill, not arbitrary signals
}

func NewProcessGroupAttr() *syscall.SysProcAttr {
    return &syscall.SysProcAttr{
        CreationFlags: 0x00000200,  // CREATE_NEW_PROCESS_GROUP
    }
}
```

### Phase 2: Replace all call sites

Replace all `syscall.Kill(pid, sig)` with `process.Kill(pid, sig)`.
Replace all `&syscall.SysProcAttr{Setsid: true}` with `process.NewProcessGroupAttr()`.

### Phase 3: Daemon mode (Dup2)

Two options:
- **A.** Skip daemon mode on Windows (simpler, team-run doesn't need it for basic operation)
- **B.** Rewrite daemon to use `cmd.Stdout = file` instead of fd dup (more correct)

Recommend **A** for initial port.

### Phase 4: File locking (remaining files)

Apply same `lockFile()`/`unlockFile()` pattern from `pkg/config/` to:
- `internal/tui/mcp/prepare_skill.go`

Or move the helpers to `pkg/process/` so all packages can use them.

---

## Goreleaser Changes

Once Windows compiles, re-enable in `.goreleaser.yml`:

```yaml
builds:
  - id: goyoke
    goos: [linux, darwin, windows]  # add windows back
    goarch: [amd64, arm64]
    ignore:
      - goos: linux
        goarch: arm64
      - goos: windows
        goarch: arm64  # skip Windows ARM for now
```

And uncomment the format override:
```yaml
format_overrides:
  - goos: windows
    formats:
      - zip
```

---

## Desktop Launchers (Post-Compile)

Once the binary compiles for each platform:

### Windows (.exe)
- Already opens a console window when double-clicked
- Add icon resource via `go-winres` or `rsrc` tool in goreleaser hooks:
  ```yaml
  before:
    hooks:
      - go-winres make --icon assets/goyoke.ico
  ```
- Optionally create Windows Terminal profile entry

### macOS (.app bundle)
- Create `goYoke.app/Contents/MacOS/goyoke-launcher.sh`:
  ```bash
  #!/bin/bash
  DIR="$(dirname "$0")"
  open -a Terminal "$DIR/goyoke"
  ```
- `Info.plist` with CFBundleIconFile pointing to `goyoke.icns`
- Goreleaser custom archive hook packages it as `.app` inside the tar.gz

### Linux (.desktop file)
- Include in archive:
  ```ini
  [Desktop Entry]
  Name=goYoke
  Exec=goyoke
  Terminal=true
  Type=Application
  Categories=Development;
  Comment=AI-powered development assistant
  ```
- Users copy to `~/.local/share/applications/`

---

## Task Breakdown

| # | Task | Depends On | Estimate |
|---|------|-----------|----------|
| 1 | Create `pkg/process/` with Kill + NewProcessGroupAttr (unix + windows) | — | 30min |
| 2 | Replace syscall.Kill in 6 files with process.Kill | 1 | 1hr |
| 3 | Replace SysProcAttr{Setsid} in 3 files with process.NewProcessGroupAttr | 1 | 30min |
| 4 | Handle Dup2 in daemon.go (skip on Windows or rewrite) | — | 30min |
| 5 | Move lockFile/unlockFile to pkg/process, use in prepare_skill.go | — | 20min |
| 6 | Verify cross-compile: `GOOS=windows go build ./...` passes | 1-5 | 10min |
| 7 | Re-enable windows in goreleaser | 6 | 5min |
| 8 | Add icon resource (go-winres) | 7 | 30min |
| 9 | Create macOS .app bundle hook | — | 1hr |
| 10 | Create Linux .desktop file | — | 10min |
| 11 | Test on actual Windows machine | 7 | 1hr |

**Critical path:** 1 → 2+3 (parallel) → 6 → 7 → 11

---

## Testing Strategy

1. **Cross-compile check** (CI): `GOOS=windows GOARCH=amd64 go build ./...`
2. **Docker + Wine** (stretch goal): Run .exe in Wine container for basic smoke test
3. **Real Windows** (required): Collaborator tests TUI on Windows Terminal
4. **Bubbletea rendering**: Verify TUI renders correctly in Windows Terminal vs cmd.exe

---

## Known Limitations (Post-Port)

- No SIGTERM on Windows — shutdown is Kill-only (less graceful)
- File locking is no-op on Windows (acceptable for counters, may cause rare double-counts)
- Daemon mode may not work identically (Dup2 not available)
- `.app` bundle requires code signing for macOS Gatekeeper (unsigned = right-click → Open)

---

## References

- [Go Windows syscall docs](https://pkg.go.dev/syscall?GOOS=windows)
- [CREATE_NEW_PROCESS_GROUP](https://learn.microsoft.com/en-us/windows/win32/procthread/process-creation-flags)
- [go-winres](https://github.com/tc-hib/go-winres) — embed Windows resources
- [Bubbletea Windows support](https://github.com/charmbracelet/bubbletea/issues/19)
