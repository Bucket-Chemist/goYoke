# GO Agents Usage Guide

## Overview

The GO agent suite provides specialized agents for building production-grade GO applications targeting desktop distribution. All agents auto-activate based on file patterns and dependencies.

## Agent Roster

| Agent             | Specialty     | Triggers                         | Auto-Activate                   |
| ----------------- | ------------- | -------------------------------- | ------------------------------- |
| **go-pro**        | General GO    | implement, refactor, go build    | `go.mod` present                |
| **go-cli**        | Cobra CLI     | cobra, subcommand, viper         | `cmd/*/main.go` + cobra dep     |
| **go-tui**        | Bubbletea TUI | tui, bubbletea, lipgloss         | `internal/tui/` + bubbletea dep |
| **go-api**        | HTTP Client   | http client, rate limit, backoff | `api/` or `client/` dirs        |
| **go-concurrent** | Concurrency   | goroutine, errgroup, worker pool | `worker/` or `pool/` dirs       |

## Agent Details

### go-pro (General GO)

**When to Use:**

- New GO module setup
- General implementation tasks
- Cross-compilation concerns
- Project structure questions

**Conventions Loaded:** `go.md`

**Example Triggers:**

- "create a new go module"
- "implement the config loader"
- "refactor this function"
- "write tests for this package"

### go-cli (Cobra CLI)

**When to Use:**

- Building CLI applications
- Adding subcommands
- Viper configuration
- Shell completion
- Flag handling

**Conventions Loaded:** `go.md`, `go-cobra.md`

**Example Triggers:**

- "add a serve command"
- "implement viper config loading"
- "add shell completion support"
- "create a new cobra subcommand"

**Sharp Edges Pre-loaded:**

- Viper binding timing (bind in init(), not RunE)
- Use `viper.Get*()` not `cmd.Flags().Get*()`
- SilenceUsage pattern for runtime errors

### go-tui (Bubbletea TUI)

**When to Use:**

- Terminal UI development
- Status dashboards
- Interactive CLI tools
- Component composition

**Conventions Loaded:** `go.md`, `go-bubbletea.md`

**Example Triggers:**

- "create a status dashboard"
- "add a progress bar component"
- "implement keyboard navigation"
- "build a list view with selection"

**Sharp Edges Pre-loaded:**

- NEVER modify model in goroutines
- View must be fast (no I/O)
- tea.Batch for multiple commands
- WindowSizeMsg handling

### go-api (HTTP Client)

**When to Use:**

- API client development
- Rate limiting
- Retry logic with backoff
- SSE streaming
- LLM API integration

**Conventions Loaded:** `go.md`

**Example Triggers:**

- "implement the API client"
- "add rate limiting"
- "implement exponential backoff"
- "handle SSE streaming"

**Sharp Edges Pre-loaded:**

- Never use default http.Client
- Configure all timeout types
- Jitter in exponential backoff
- Context propagation

### go-concurrent (Concurrency)

**When to Use:**

- Worker pool patterns
- Parallel processing
- errgroup coordination
- Semaphore rate limiting
- Graceful shutdown

**Conventions Loaded:** `go.md`

**Example Triggers:**

- "implement a worker pool"
- "add parallel processing"
- "coordinate goroutines with errgroup"
- "implement graceful shutdown"

**Sharp Edges Pre-loaded:**

- Loop variable capture in goroutines
- Context cancellation patterns
- Channel closing (only sender closes)
- Defer in loops pitfall

## Auto-Activation Patterns

Agents auto-activate based on these conditions:

```yaml
go-pro:
  file_patterns: ["go.mod", "*.go"]
  languages: ["Go"]

go-cli:
  patterns: ["**/cmd/**/main.go", "**/cli/**/*.go"]
  dependencies: ["github.com/spf13/cobra", "github.com/spf13/viper"]

go-tui:
  patterns: ["**/tui/**/*.go", "**/ui/**/*.go"]
  dependencies:
    ["github.com/charmbracelet/bubbletea", "github.com/charmbracelet/lipgloss"]

go-api:
  patterns: ["**/api/**/*.go", "**/client/**/*.go"]
  dependencies: ["golang.org/x/time/rate"]

go-concurrent:
  patterns: ["**/worker/**/*.go", "**/pool/**/*.go"]
  dependencies: ["golang.org/x/sync/errgroup", "golang.org/x/sync/semaphore"]
```

## Routing Priority

When multiple agents could match:

1. **Most specific wins**: go-tui over go-pro for TUI work
2. **Trigger keywords**: "bubbletea" → go-tui, "cobra" → go-cli
3. **Dependency detection**: Presence of charmbracelet imports → go-tui

## Example Workflows

### New CLI Application

```
User: Create a new CLI tool for managing agents

[ROUTING] → go-cli (detected: "cli tool" trigger)

go-cli: I'll create a Cobra-based CLI structure:
1. cmd/GoGent/main.go - Entry point
2. internal/cli/root.go - Root command
3. internal/cli/serve/command.go - Subcommand example
...
```

### Adding TUI Dashboard

```
User: Add a status dashboard to show running agents

[ROUTING] → go-tui (detected: "dashboard" trigger)

go-tui: I'll create a Bubbletea-based TUI:
1. internal/tui/model.go - Application state
2. internal/tui/view.go - Rendering
3. internal/tui/components/status.go - Status component
...
```

### Worker Pool Implementation

```
User: Implement parallel agent spawning with concurrency limit

[ROUTING] → go-concurrent (detected: "parallel", "concurrency limit" triggers)

go-concurrent: I'll implement a worker pool pattern:
1. Create WorkerPool struct
2. Use semaphore for concurrency control
3. errgroup for error coordination
...
```

## Integration with Orchestrator

When tasks span multiple concerns:

```
User: Build a CLI tool with a TUI dashboard that spawns workers

[ROUTING] → orchestrator (compound triggers from multiple agents)

orchestrator:
1. Spawns go-cli for CLI structure
2. Spawns go-tui for dashboard
3. Spawns go-concurrent for worker pool
4. Coordinates outputs
```

## Conventions Files

All conventions are in `~/.claude/conventions/`:

- `go.md` - Core GO conventions (all agents load this)
- `go-cobra.md` - Cobra CLI patterns (go-cli loads this)
- `go-bubbletea.md` - Bubbletea TUI patterns (go-tui loads this)

## Initializing GO Projects

Use `/init-auto` to scaffold a new GO project:

```bash
# In your project directory
/init-auto go      # Basic GO project
/init-auto go-cli  # Cobra CLI project
/init-auto go-tui  # Bubbletea TUI project
```

This creates `CLAUDE.md` with appropriate convention references.

## Testing GO Code

All agents apply these testing standards:

1. **Table-driven tests** - Standard pattern
2. **Race detection** - Always run with `-race`
3. **Variable capture** - `tc := tc` in parallel tests
4. **testify assertions** - `require` for fatal, `assert` for non-fatal

## Build Standards

All agents enforce these build requirements:

```makefile
# Cross-compilation targets
GOOS=darwin GOARCH=amd64 go build ...
GOOS=darwin GOARCH=arm64 go build ...
GOOS=windows GOARCH=amd64 go build ...
GOOS=linux GOARCH=amd64 go build ...

# Version injection
go build -ldflags "-X main.version=${VERSION}" ...
```

## Sharp Edges Catalog

Each agent has pre-loaded gotchas. View them:

```bash
cat ~/.claude/agents/go-pro/sharp-edges.yaml
cat ~/.claude/agents/go-cli/sharp-edges.yaml
cat ~/.claude/agents/go-tui/sharp-edges.yaml
cat ~/.claude/agents/go-api/sharp-edges.yaml
cat ~/.claude/agents/go-concurrent/sharp-edges.yaml
```

## Troubleshooting

### Agent Not Activating

Check file patterns:

```bash
ls go.mod           # go-pro should activate
ls cmd/*/main.go    # go-cli should activate if cobra in go.mod
ls internal/tui/    # go-tui should activate if bubbletea in go.mod
```

### Wrong Agent Selected

Use explicit routing:

```
User: [go-tui] Add a spinner to the status component
```

### Missing Conventions

Verify conventions exist:

```bash
ls ~/.claude/conventions/go*.md
# Should show: go.md, go-cobra.md, go-bubbletea.md
```
