# goYoke

[![Build](https://github.com/Bucket-Chemist/goYoke/actions/workflows/build-test.yml/badge.svg)](https://github.com/Bucket-Chemist/goYoke/actions/workflows/build-test.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/Bucket-Chemist/goYoke)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-linux%20%7C%20macOS%20%7C%20windows-brightgreen)]()

**Slow the vibes down. *Really* plan, review, break it into context-sound chunks with testable boundaries, and THEN implement.**

goYoke wraps your models with compiled Go enforcement: tiered agent routing, multi-agent workflows, and a terminal UI to track your work history, track work dispatch, log work relative to budget. Every request is classified, delegated to the right model tier, and tracked.

End result? You produce tested deliverables instead of a psychotic echo chamber with a sycophantic silicon voice designed to lull you into a false sense of dopamine-drunk security.

> *Built for Claude Code at launch. Multi-provider support planned.*

<p align="center">
  <img src="assets/goyoke-monkey.png" alt="Don't be this." width="400">
  <br>
  <em>^^^^ don't be their trained token monkey</em>
</p>

## Why goYoke?

goYoke encourages a development-centric workflow: scout the problem, create a plan, review it, then implement. Every handoff between agents is typed — JSON-schema-validated stdin/stdout contracts ensure that a reviewer always produces structured findings with severity, file, and recommendation; that a worker always reports which acceptance criteria were met with evidence, and that an architect always produces implementation plans with dependency graphs and testable criteria. Same contracts in, comparable quality out.

This project represents an attempt to deviate from documentation theatre as much as possible. goYoke uses numerous compiled Go binaries that intercept model events at runtime. They validate routing, track failures, enforce delegation, and capture sharp edges. Enforcement happens in code, not in text instructions that degrade over long conversations. The model can't forget a binary that blocks its tool call.

### How I use this

I built the progress tracker so I could more accurately track what I had done during the session. Initially I used a zellij template with lazygit as a sidebar but in the past few weeks I have defaulted back to vscode with goyoke at ~50% of window width. Run a 2x2 grid of these (see below) and...baby you got a stew going.

<p align="center">
  <img src="assets/HGR4doYXwAALMtm.png" alt="goYoke TUI during a long implementation session" width="800">
  <br>
  <em>A typical long implementation session in goYoke</em>
</p>

### Why I built this

As a scientist by trade, I write code for an ISO compliant mass spectrometry facility. I abhor black boxes and as a field mass spectrometry generates absurdly precise biochemical measurements. As a supportive adjunct to this calibre of science - when I write pipelines or software it is a requirement that these are entirely transparent, reproducible and follow strict stylistic and best practice conventions.

The barriers to having the same kind of experience when using inference providers are countless. Model providers silently alter quality between releases; or throttle inference quietly to a lower tier model whilst you pay for the frontier. The same prompt produces different results on Tuesday than it did on Monday because they don't have compute to cover users online.

If you care about sitting down at your desk and getting the same quality of work every session - then agentic activity needs to be standardised, measurable, and more than anything **stable and consistent**.

I started my AI journey in December 2022 with ChatGPT one-shotting some of my more obscure trading algorithm problems, and I remember thinking that humanity had entered a new era of tool discovery akin to stone tools. The real alpha laid not in those who swung the axe - but in those who found ways to abstract the sharp edge into more complex tools. This project represents my ongoing journey of using these tools (from chat interfaces, to Cursor, Windsurf, terminal, you name it) and trying to create a generalisable harness that reaches for the optimal model paired with the right schema no matter the use case. As a builder I built this to be the tool I use to build everything else in my work.

## NB

I am by no means done with this project - for those of you who are fortunate enough to have enjoyed Claude's generous subscription model - you will rightly be feeling the time of cheap frontier inference is coming to an end. I will be providing support for Codex (which I also use for various use cases) immediately following release, and then will be moving onto supporting standard tool call/message frameworks for open source models and any others that I use in my work routinely. If you have any providers you wish supported please feel free to drop me a line and I would be happy to accomodate.


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

## How It Works (current at release specifically for Claude Code)

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

- **[Workflow Guide](docs/WORKFLOWS.md)** — Reproducibility contracts, team orchestration, skills, costs, anti-patterns
- **[Roadmap](ROADMAP.md)** — Version plan: multi-provider, benchmarks, TUI editor, telemetry dashboard
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