---
name: init-auto
description: This skill should be used when initializing a new project with Claude Code configuration. It detects the project language from indicator files (pyproject.toml, setup.py for Python; DESCRIPTION, renv.lock for R; package.json for JavaScript; go.mod for Go), creates a ./CLAUDE.md file that references the appropriate global conventions, and scaffolds an Obsidian dev-vault folder for the project. Invoke with /init-auto or when user asks to "initialize claude config" or "set up claude for this project".
---

# Init Auto - Project Configuration Scaffolder

## Overview

This skill detects a project's primary language and:
1. Creates a `CLAUDE.md` configuration file that references the appropriate global conventions stored in `~/.claude/conventions/`
2. Creates a project folder in `~/Documents/dev-vault/` for development notes, ideas, and issue tracking

## Workflow

### Step 1: Detect Project Language and Framework

Scan the current directory for language indicator files:

| Indicators | Language/Framework |
|------------|-------------------|
| `go.mod`, `go.sum`, `*.go` in root | Go |
| `go.mod` + `cmd/*/main.go` pattern | Go (CLI with Cobra likely) |
| `go.mod` + `internal/tui/` pattern | Go (TUI with Bubbletea likely) |
| `pyproject.toml`, `setup.py`, `requirements.txt`, `uv.lock`, `Pipfile` | Python |
| `DESCRIPTION` + `inst/golem-config.yml` | R (Golem) |
| `DESCRIPTION` + (`app.R` or `ui.R`/`server.R`) | R (Shiny) |
| `DESCRIPTION`, `NAMESPACE`, `*.Rproj`, `renv.lock` | R |
| `package.json`, `tsconfig.json`, `deno.json` | JavaScript/TypeScript |

**Detection priority:**
1. Check for Go indicators first (`go.mod`, `*.go` files)
2. Then check for Golem indicators (`inst/golem-config.yml`)
3. Then check for Shiny indicators (`app.R`, `ui.R`/`server.R`)
4. Fall back to base R if DESCRIPTION found
5. Check Python indicators
6. Check JavaScript indicators

Execute detection by listing files in the project root:

```bash
ls -la
ls cmd/ 2>/dev/null
ls internal/ 2>/dev/null
ls R/ 2>/dev/null
ls inst/ 2>/dev/null
cat go.mod 2>/dev/null | head -5
```

### Step 2: Announce Detection

After identifying the language/framework, announce the result:

**Go base:**
```
[init-auto] Detected: Go project (found go.mod)
[init-auto] Creating CLAUDE.md with Go convention references...
```

**Go CLI (Cobra):**
```
[init-auto] Detected: Go CLI project (found cmd/ structure with cobra dependency)
[init-auto] Creating CLAUDE.md with Go + Cobra CLI convention references...
```

**Go TUI (Bubbletea):**
```
[init-auto] Detected: Go TUI project (found bubbletea/lipgloss dependencies)
[init-auto] Creating CLAUDE.md with Go + Bubbletea TUI convention references...
```

**Base language:**
```
[init-auto] Detected: [LANGUAGE] project
[init-auto] Creating CLAUDE.md with [LANGUAGE] convention references...
```

**R with Shiny:**
```
[init-auto] Detected: R Shiny project (found app.R/ui.R)
[init-auto] Creating CLAUDE.md with R + Shiny convention references...
```

**R with Golem:**
```
[init-auto] Detected: R Golem project (found inst/golem-config.yml)
[init-auto] Creating CLAUDE.md with R + Shiny + Golem convention references...
```

If multiple languages detected:
```
[init-auto] Detected multiple languages: [LIST]
[init-auto] Please specify primary language: Go, Go-CLI, Go-TUI, Python, R, R-Shiny, R-Golem, or JavaScript?
```

If no language detected:
```
[init-auto] No recognized language indicators found.
[init-auto] Please specify: Go, Go-CLI, Go-TUI, Python, R, R-Shiny, R-Golem, or JavaScript?
```

### Step 3: Create CLAUDE.md

Based on the detected (or specified) language, create `./CLAUDE.md` with the appropriate template.

#### Go Template

```markdown
# Project Configuration

## Language Conventions

This is a Go project. The global Go conventions from `~/.claude/conventions/go.md` apply.

At session start, read and apply those conventions.

## System Constraint

**Target: Desktop distribution to non-technical users.**

- Single binary output (zero runtime dependencies)
- Cross-compilation support (darwin/windows/linux)
- Static asset embedding via go:embed

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

- `go.mod` - Module definition
- `main.go` - Entry point (or cmd/*/main.go)
```

#### Go CLI (Cobra) Template

```markdown
# Project Configuration

## Language Conventions

This is a Go CLI project using Cobra. The following global conventions apply:
- `~/.claude/conventions/go.md` (Go core)
- `~/.claude/conventions/go-cobra.md` (Cobra CLI conventions)

At session start, read and apply those conventions.

## System Constraint

**Target: Desktop distribution to non-technical users.**

- Single binary output (zero runtime dependencies)
- Cross-compilation support (darwin/windows/linux)
- Static asset embedding via go:embed
- Shell completion generation

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

- `go.mod` - Module definition
- `cmd/*/main.go` - CLI entry point
- `internal/cli/` - Command implementations
```

#### Go TUI (Bubbletea) Template

```markdown
# Project Configuration

## Language Conventions

This is a Go TUI project using Bubbletea. The following global conventions apply:
- `~/.claude/conventions/go.md` (Go core)
- `~/.claude/conventions/go-bubbletea.md` (Bubbletea TUI conventions)

At session start, read and apply those conventions.

## System Constraint

**Target: Desktop distribution to non-technical users.**

- Single binary output (zero runtime dependencies)
- Cross-compilation support (darwin/windows/linux)
- Static asset embedding via go:embed
- The Elm Architecture (MVU): Model-View-Update

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

- `go.mod` - Module definition
- `main.go` or `cmd/*/main.go` - Entry point
- `internal/tui/` - TUI components
```

#### Python Template

```markdown
# Project Configuration

## Language Conventions

This is a Python project. The global Python conventions from `~/.claude/conventions/python.md` apply.

At session start, read and apply those conventions.

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

<!-- Document important files and directories -->
```

#### R Template

```markdown
# Project Configuration

## Language Conventions

This is an R project. The global R conventions from `~/.claude/conventions/R.md` apply.

At session start, read and apply those conventions.

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

<!-- Document important files and directories -->
```

#### R-Shiny Template

```markdown
# Project Configuration

## Language Conventions

This is an R Shiny project. The following global conventions apply:
- `~/.claude/conventions/R.md` (R core)
- `~/.claude/conventions/R-shiny.md` (Shiny conventions)

At session start, read and apply those conventions.

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

- `app.R` - Main application entry point
```

#### R-Golem Template

```markdown
# Project Configuration

## Language Conventions

This is an R Golem project. The following global conventions apply:
- `~/.claude/conventions/R.md` (R core)
- `~/.claude/conventions/R-shiny.md` (Shiny conventions)
- `~/.claude/conventions/R-golem.md` (Golem conventions)

At session start, read and apply those conventions.

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

- `R/app_ui.R` - Main UI
- `R/app_server.R` - Main server
- `R/mod_*.R` - Golem modules
- `inst/golem-config.yml` - Configuration
- `dev/02_dev.R` - Development utilities
```

#### JavaScript/TypeScript Template

```markdown
# Project Configuration

## Language Conventions

This is a JavaScript/TypeScript project. No language-specific global rules are currently configured.

## Dev Vault

Development notes, ideas, and issues are tracked in Obsidian:
- **Vault**: `~/Documents/dev-vault/[PROJECT_NAME]/`
- **Journal**: `~/Documents/dev-vault/[PROJECT_NAME]/journal/`
- **Ideas**: `~/Documents/dev-vault/[PROJECT_NAME]/ideas/`
- **Issues**: `~/Documents/dev-vault/[PROJECT_NAME]/issues/`

Read relevant notes when context is needed for a task.

## Debug 66

Verbose step-trace debugging is available. Invoke with "execute debug 66" or "debug 66 [target]".

## Project-Specific Rules

<!-- Add project-specific overrides below -->

## Key Files

<!-- Document important files and directories -->
```

### Step 4: Create Obsidian Dev Vault Project Folder

After creating CLAUDE.md, scaffold the Obsidian project folder:

```bash
# Get project name from current directory
PROJECT_NAME=$(basename "$PWD")

# Create project structure in dev-vault
mkdir -p ~/Documents/dev-vault/${PROJECT_NAME}/{journal,ideas,issues,decisions}
```

Then create the project index file at `~/Documents/dev-vault/${PROJECT_NAME}/_index.md`:

```markdown
---
project: [PROJECT_NAME]
created: [TODAY'S DATE]
type: index
repo: [FULL PATH TO PROJECT]
---

# [PROJECT_NAME]

## Overview
Brief description of the project.

## Quick Links
- **Repo**: [FULL PATH]
- **CLAUDE.md**: [PATH]/CLAUDE.md

## Current Focus
What are we working on right now?

## Recent Journal
- [[journal/|Latest entries]]

## Session Handoff
Notes for next Claude session:
-
```

Announce:
```
[init-auto] Created Obsidian project folder at ~/Documents/dev-vault/[PROJECT_NAME]/
[init-auto] Subdirectories: journal/, ideas/, issues/, decisions/
```

### Step 5: Confirm Completion

After creating all files:

```
[init-auto] Created ./CLAUDE.md for [LANGUAGE] project
[init-auto] Created ~/Documents/dev-vault/[PROJECT_NAME]/ for dev notes
[init-auto] Edit CLAUDE.md to add project-specific rules
[init-auto] Open dev-vault in Obsidian for journaling and idea capture
```

## Manual Override

To create configuration for a specific language/framework regardless of detection:

- `/init-auto go` - Force Go template
- `/init-auto go-cli` - Force Go CLI (Cobra) template
- `/init-auto go-tui` - Force Go TUI (Bubbletea) template
- `/init-auto python` - Force Python template
- `/init-auto r` - Force R template
- `/init-auto shiny` - Force R-Shiny template
- `/init-auto golem` - Force R-Golem template
- `/init-auto js` - Force JavaScript template

## Examples

**Example 1: Go project with go.mod**
```
User: /init-auto
Claude: [init-auto] Detected: Go project (found go.mod)
        [init-auto] Creating CLAUDE.md with Go convention references...
        [init-auto] Created ./CLAUDE.md for Go project
        [init-auto] Edit the file to add project-specific rules and key files
```

**Example 2: Go CLI project with Cobra**
```
User: /init-auto
Claude: [init-auto] Detected: Go CLI project (found cmd/ structure, github.com/spf13/cobra in go.mod)
        [init-auto] Creating CLAUDE.md with Go + Cobra CLI convention references...
        [init-auto] Created ./CLAUDE.md for Go CLI project
```

**Example 3: Go TUI project with Bubbletea**
```
User: /init-auto
Claude: [init-auto] Detected: Go TUI project (found github.com/charmbracelet/bubbletea in go.mod)
        [init-auto] Creating CLAUDE.md with Go + Bubbletea TUI convention references...
        [init-auto] Created ./CLAUDE.md for Go TUI project
```

**Example 4: Python project with pyproject.toml**
```
User: /init-auto
Claude: [init-auto] Detected: Python project (found pyproject.toml)
        [init-auto] Creating CLAUDE.md with Python convention references...
        [init-auto] Created ./CLAUDE.md for Python project
        [init-auto] Edit the file to add project-specific rules and key files
```

**Example 5: Ambiguous project**
```
User: /init-auto
Claude: [init-auto] Detected multiple languages: Python (requirements.txt), JavaScript (package.json)
        [init-auto] Please specify primary language: Python or JavaScript?
User: Python
Claude: [init-auto] Creating CLAUDE.md with Python convention references...
        [init-auto] Created ./CLAUDE.md for Python project
```

**Example 6: Golem project**
```
User: /init-auto
Claude: [init-auto] Detected: R Golem project (found inst/golem-config.yml, R/app_ui.R)
        [init-auto] Creating CLAUDE.md with R + Shiny + Golem convention references...
        [init-auto] Created ./CLAUDE.md for R Golem project
        [init-auto] Edit the file to add project-specific rules
```

**Example 7: Manual specification for Go TUI**
```
User: /init-auto go-tui
Claude: [init-auto] Creating CLAUDE.md with Go + Bubbletea TUI convention references...
        [init-auto] Created ./CLAUDE.md for Go TUI project
```

## Not For
- Understanding an existing goYoke setup (use /dummies-guide instead)
- Adding skills to an initialized project (use /explore-add instead)
- Modifying existing CLAUDE.md (edit directly or use /explore for planning)
