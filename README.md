# goYoke

**Stop vibe coding. Plan, review, then implement.**

goYoke wraps Claude Code with compiled Go enforcement: tiered agent routing, multi-agent workflows, and a terminal UI. Every request is classified, delegated to the right model tier, and tracked — so you produce deliverables, not chat.

> *Built for Claude Code at launch. Multi-provider support planned.*

<p align="center">
  <img src="assets/goYoke_v1.png" alt="goYoke — multi-agent terminal orchestration" width="480">
  <br>
  <em>Stop being the model's token-spending monkey.</em>
</p>

## Why goYoke?

LLMs are powerful but easy to misuse. You jump into implementation without a plan, burn premium tokens on grep, lose context in long conversations, and end up with untested code you're not confident in. Each chat feels productive until you realize you've spent $50 and have no working feature.

goYoke enforces a development workflow: scout the problem, create a plan, review it, then implement. Each step uses the cheapest model tier that can handle it. The router never implements — it coordinates. Agents get fresh context with injected conventions. The cheapest tier handles file search. Mid-tier implements. Premium tier only makes architectural decisions. No degradation, no wasted tokens.

This isn't prompt engineering. goYoke uses 11 compiled Go binaries that intercept Claude Code events at runtime. They validate routing, track failures, enforce delegation, and capture sharp edges. Enforcement happens in code, not in text instructions that Claude forgets.

## Quick Start

**Prerequisites:** [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated.

**Homebrew** (macOS + Linux):
```bash
brew install Bucket-Chemist/tap/goyoke
goyoke
```

**Arch Linux:**
```bash
paru -S goyoke-bin
goyoke
```

**Binary download:** See [Releases](../../releases) for all platforms.

## What You Can Do

| Command | What Happens | Cost |
|---------|-------------|------|
| "fix the login bug" | Router delegates to language-specific agent (Sonnet) | ~$0.10 |
| `/explore "how does auth work?"` | Scout reconnoiters, architect analyzes | ~$0.20 |
| `/review` | 4 specialist reviewers examine code in parallel | ~$0.30 |
| `/review-plan` | Staff architect applies 7-layer critical review | ~$0.25 |
| `/braintrust "Redis vs DB cache?"` | 4 Opus agents analyze from different perspectives | ~$1.00 |
| `/plan-tickets "notifications"` | Full pipeline: scout → plan → architect → review → tickets | ~$3.00 |
| `/implement` | Architect plans, background workers execute | ~$2.00 |
| `/cleanup` | 8 specialist reviewers audit code hygiene | ~$0.50 |

See [Workflow Guide](docs/WORKFLOWS.md) for detailed examples with sample output.

## How It Works

### Standalone wrapper — your Claude Code stays clean

goYoke is a single Go binary. It wraps Claude Code CLI as a subprocess, injecting its hooks, agents, and settings at runtime via CLI flags. Your Claude Code installation, your `~/.claude/` directory, and your project's `.claude/` config are never modified. Everything goYoke needs is embedded in the binary and applied ephemerally — close goYoke, and Claude Code is exactly as it was.

This means goYoke can sit on top of any Claude Code setup without conflict. No agents polluting your native config. No hooks left behind. No MCP servers registered in your settings. Install it, run it, remove it — zero residue.

### MCP compliance boundary — one choke point for all agent interactions

Claude Code has built-in tools for spawning subagents (`Agent`, `Task`). goYoke deliberately blocks these because they bypass all enforcement — no hook fires, no conventions load, no identity gets injected. The subagent just runs naked.

Instead, goYoke provides its own `spawn_agent` tool through an MCP server that lives inside the TUI ecosystem. This creates a single compliance boundary: every agent spawn goes through the MCP server, which validates the tier, injects the agent's identity and conventions, checks parent-child authorization, and logs telemetry — before the agent ever starts.

The router and all agents can only interact through these MCP tools. There is no back door. The enforcement is structural, not behavioral.

### The full picture

```
You → goYoke TUI → Router (classifies request)
                        │
                        ├─ MCP spawn_agent ──→ Agent (executes) → Your codebase
                        │    validates tier       ↑
                        │    injects identity     Conventions
                        │    logs telemetry       injected per language
                        │
                        └─ 11 Go hooks fire on every Claude Code event
                             validate, track, enforce, capture
```

For the full technical architecture, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Installation

### Homebrew (macOS + Linux)
```bash
brew install Bucket-Chemist/tap/goyoke
```

### Arch Linux (AUR)
```bash
paru -S goyoke-bin
```

### Standalone binary
Download from [Releases](../../releases) and add to PATH.

### Go install
```bash
go install github.com/Bucket-Chemist/goYoke/cmd/goyoke@latest
```

### Build from source
```bash
git clone https://github.com/Bucket-Chemist/goYoke.git
cd goYoke
make dist && make install
```

See [INSTALL.md](INSTALL.md) for detailed setup including auth configuration.

## Platform Support

| Platform | Status | Install Methods |
|----------|--------|-----------------|
| Linux x86_64 | Supported | Homebrew, AUR, binary, go install |
| macOS Intel | Supported | Homebrew, binary, go install |
| macOS Apple Silicon | Supported | Homebrew, binary, go install |
| Windows x86_64 | Supported | Binary, go install |

## Documentation

- **[Workflow Guide](docs/WORKFLOWS.md)** — Skills, workflows, costs, anti-patterns
- **[Installation](INSTALL.md)** — Auth setup, platform details
- **[Architecture](docs/ARCHITECTURE.md)** — System internals, hooks, team orchestration
- **[Contributing](CONTRIBUTING.md)** — How to contribute agents, conventions, and skills

## Language Support

goYoke auto-loads coding conventions based on file context:

| Language | Agents | Specialized Conventions |
|----------|--------|------------------------|
| Go | `go-pro`, `go-cli`, `go-tui`, `go-api`, `go-concurrent` | Cobra, Bubbletea |
| Python | `python-pro`, `python-ux` | — |
| TypeScript | `typescript-pro`, `react-pro` | React |
| Rust | `rust-pro` | — |
| R | `r-pro`, `r-shiny-pro` | Shiny, Golem |

Custom domain agents can be added via `.claude/agents/` — see [Contributing](CONTRIBUTING.md).

## Author

Created by [Dokter Smol](https://github.com/Bucket-Chemist)

## License

[MIT](LICENSE)
