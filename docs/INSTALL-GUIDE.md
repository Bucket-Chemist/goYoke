---
title: Installation Guide
type: guide
tags: [installation, setup]
created: 2026-04-18
---
# goYoke Installation Guide

> **Target Audience:** Users migrating from bash-based Claude Code hooks to goYoke Go binaries.
>
> **Prerequisites:** Go 1.21+, Claude Code CLI installed, `~/.local/bin` in PATH.
>
> **Time Required:** 15-30 minutes for first-time setup.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites Check](#2-prerequisites-check)
3. [Build the Binaries](#3-build-the-binaries)
4. [Install the Binaries](#4-install-the-binaries)
5. [Create Test Environment](#5-create-test-environment)
6. [Configure Test Environment](#6-configure-test-environment)
7. [Run Claude with Test Config](#7-run-claude-with-test-config)
8. [Validate Installation](#8-validate-installation)
9. [Production Cutover](#9-production-cutover)
10. [Rollback Procedure](#10-rollback-procedure)
11. [Troubleshooting](#11-troubleshooting)
12. [Running as `claudeGO` Command](#12-running-as-claudego-command)

---

## 1. Overview

### What This Guide Does

This guide helps you replace your existing bash-based Claude Code hooks with compiled Go binaries from goYoke.

### Architecture Change

**Before (Bash Scripts):**
```
Claude Code → ~/.claude/hooks/validate-routing.sh → routing-schema.json
```

**After (Go Binaries):**
```
Claude Code → goyoke-validate (binary) → routing-schema.json
```

### Binary to Hook Mapping

| Hook Event | Bash Script | Go Binary |
|------------|-------------|-----------|
| SessionStart | `load-routing-context.sh` | `goyoke-load-context` |
| PreToolUse (Task) | `validate-routing.sh` | `goyoke-validate` |
| PostToolUse | `sharp-edge-detector.sh` | `goyoke-sharp-edge` |
| PostToolUse | `attention-gate.sh` | (merged into `goyoke-sharp-edge`) |
| SubagentStop | `agent-endstate.sh` | `goyoke-agent-endstate` |
| SubagentStop | `orchestrator-completion-guard.sh` | `goyoke-orchestrator-guard` |
| SessionEnd | `session-archive.sh` | `goyoke-archive` |

---

## 2. Prerequisites Check

### 2.1 Check Go Installation

```bash
go version
```

**Expected output:**
```
go version go1.21.0 linux/amd64
```

If Go is not installed:
```bash
# Arch Linux / CachyOS
sudo pacman -S go

# Ubuntu/Debian
sudo apt install golang-go

# macOS
brew install go
```

### 2.2 Check Claude Code Installation

```bash
claude --version
```

**Expected output:**
```
claude-code version X.X.X
```

### 2.3 Check PATH Configuration

```bash
echo $PATH | tr ':' '\n' | grep -E "local/bin|go/bin"
```

**Expected output should include:**
```
/home/YOUR_USERNAME/.local/bin
```

If `~/.local/bin` is NOT in PATH, add it now:

```bash
# For bash users
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# For zsh users
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

### 2.4 Verify PATH Change

```bash
echo $PATH | grep -q ".local/bin" && echo "SUCCESS: ~/.local/bin is in PATH" || echo "FAILED: ~/.local/bin NOT in PATH"
```

---

## 3. Build the Binaries

### 3.1 Navigate to Project Directory

```bash
cd ~/Documents/goYoke
```

Verify you're in the right place:
```bash
ls -la go.mod Makefile
```

**Expected output:**
```
-rw-r--r-- 1 user user XXXX ... go.mod
-rw-r--r-- 1 user user XXXX ... Makefile
```

### 3.2 Download Dependencies

```bash
go mod download
```

### 3.3 Build All Binaries

```bash
make build-all
```

**Expected output:**
```
Building goyoke-validate binary...
✅ Binary created at bin/goyoke-validate
Building goyoke-archive binary...
✅ Binary created at bin/goyoke-archive
Building goyoke-sharp-edge binary...
✅ Binary created at bin/goyoke-sharp-edge
Building goyoke-load-context...
✓ Built: bin/goyoke-load-context
Building goyoke-agent-endstate...
✓ Built: bin/goyoke-agent-endstate
Building goyoke-orchestrator-guard...
✓ Built: bin/goyoke-orchestrator-guard
Building goyoke-doc-theater...
✓ Built: bin/goyoke-doc-theater
✓ All hook binaries built
```

### 3.4 Verify Binaries Exist

```bash
ls -la bin/
```

**Expected output (7 binaries):**
```
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-agent-endstate
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-archive
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-doc-theater
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-load-context
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-orchestrator-guard
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-sharp-edge
-rwxr-xr-x 1 user user XXXXXXX ... goyoke-validate
```

### 3.5 Run Tests (Optional but Recommended)

```bash
make test-unit
```

All tests should pass before proceeding.

---

## 4. Install the Binaries

### 4.1 Install to ~/.local/bin

```bash
make install
```

**Expected output:**
```
Installing goYoke CLIs to ~/.local/bin/...
✅ Installed goyoke-validate, goyoke-archive, goyoke-aggregate, goyoke-sharp-edge, goyoke-capture-intent, goyoke-load-context, goyoke-agent-endstate, goyoke-orchestrator-guard, goyoke-doc-theater
✅ ~/.local/bin is in PATH
```

### 4.2 Verify Installation

Run each binary to confirm it's accessible:

```bash
goyoke-validate --help 2>/dev/null || echo '{}' | goyoke-validate
goyoke-load-context --help 2>/dev/null || echo '{}' | goyoke-load-context
goyoke-sharp-edge --help 2>/dev/null || echo '{}' | goyoke-sharp-edge
goyoke-archive --help 2>/dev/null || echo '{}' | goyoke-archive
goyoke-agent-endstate --help 2>/dev/null || echo '{}' | goyoke-agent-endstate
goyoke-orchestrator-guard --help 2>/dev/null || echo '{}' | goyoke-orchestrator-guard
```

Each should either print help or produce JSON output (not "command not found").

### 4.3 Verify Binary Locations

```bash
which goyoke-validate goyoke-load-context goyoke-sharp-edge goyoke-archive goyoke-agent-endstate goyoke-orchestrator-guard
```

**Expected output:**
```
/home/YOUR_USERNAME/.local/bin/goyoke-validate
/home/YOUR_USERNAME/.local/bin/goyoke-load-context
/home/YOUR_USERNAME/.local/bin/goyoke-sharp-edge
/home/YOUR_USERNAME/.local/bin/goyoke-archive
/home/YOUR_USERNAME/.local/bin/goyoke-agent-endstate
/home/YOUR_USERNAME/.local/bin/goyoke-orchestrator-guard
```

---

## 5. Create Test Environment

**IMPORTANT:** Do NOT modify your production `~/.claude/` yet. Create a separate test environment first.

### 5.1 Create Test Directory Structure

```bash
mkdir -p ~/.claude-test/{hooks,memory,tmp,session-archive}
```

### 5.2 Copy Configuration Files

```bash
# Core configuration (REQUIRED)
cp ~/.claude/CLAUDE.md ~/.claude-test/
cp ~/.claude/routing-schema.json ~/.claude-test/

# Agents (REQUIRED)
cp -r ~/.claude/agents ~/.claude-test/

# Conventions (REQUIRED)
cp -r ~/.claude/conventions ~/.claude-test/

# Rules (REQUIRED)
cp -r ~/.claude/rules ~/.claude-test/

# Skills (REQUIRED for slash commands)
cp -r ~/.claude/skills ~/.claude-test/

# Documentation (OPTIONAL)
cp -r ~/.claude/docs ~/.claude-test/ 2>/dev/null || true
```

### 5.3 Create Symlinks to Go Binaries

```bash
# Create symlinks from test hooks directory to installed binaries
ln -sf ~/.local/bin/goyoke-validate ~/.claude-test/hooks/goyoke-validate
ln -sf ~/.local/bin/goyoke-load-context ~/.claude-test/hooks/goyoke-load-context
ln -sf ~/.local/bin/goyoke-sharp-edge ~/.claude-test/hooks/goyoke-sharp-edge
ln -sf ~/.local/bin/goyoke-archive ~/.claude-test/hooks/goyoke-archive
ln -sf ~/.local/bin/goyoke-agent-endstate ~/.claude-test/hooks/goyoke-agent-endstate
ln -sf ~/.local/bin/goyoke-orchestrator-guard ~/.claude-test/hooks/goyoke-orchestrator-guard
ln -sf ~/.local/bin/goyoke-doc-theater ~/.claude-test/hooks/goyoke-doc-theater
```

### 5.4 Verify Test Environment Structure

```bash
find ~/.claude-test -maxdepth 2 -type d | sort
```

**Expected output:**
```
/home/YOUR_USERNAME/.claude-test
/home/YOUR_USERNAME/.claude-test/agents
/home/YOUR_USERNAME/.claude-test/agents/architect
/home/YOUR_USERNAME/.claude-test/agents/codebase-search
... (more agent directories)
/home/YOUR_USERNAME/.claude-test/conventions
/home/YOUR_USERNAME/.claude-test/docs
/home/YOUR_USERNAME/.claude-test/hooks
/home/YOUR_USERNAME/.claude-test/memory
/home/YOUR_USERNAME/.claude-test/rules
/home/YOUR_USERNAME/.claude-test/session-archive
/home/YOUR_USERNAME/.claude-test/skills
/home/YOUR_USERNAME/.claude-test/tmp
```

---

## 6. Configure Test Environment

### 6.1 Create settings.json for Test Environment

Create the file `~/.claude-test/settings.json` with the following content:

```bash
cat > ~/.claude-test/settings.json << 'SETTINGS_EOF'
{
  "permissions": {
    "allow": [
      "Read(**)",
      "Glob(**)",
      "Grep(**)",
      "Bash(wc:*)",
      "Bash(find:*)",
      "Bash(head:*)",
      "Bash(tail:*)",
      "Bash(cat:*)",
      "Bash(ls:*)",
      "Bash(stat:*)",
      "Bash(git status:*)",
      "Bash(git diff:*)",
      "Bash(git log:*)",
      "Bash(go test:*)",
      "Bash(go build:*)",
      "Bash(make:*)"
    ],
    "deny": [
      "Write(.env*)",
      "Write(**/secrets/**)",
      "Bash(*rm -rf*)",
      "Bash(*> /dev/*)",
      "Bash(*sudo*)"
    ]
  },
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-load-context",
            "timeout": 10
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-validate",
            "timeout": 10
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|Task",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-sharp-edge",
            "timeout": 5
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-agent-endstate",
            "timeout": 15
          },
          {
            "type": "command",
            "command": "goyoke-orchestrator-guard",
            "timeout": 10
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-archive",
            "timeout": 30
          }
        ]
      }
    ]
  },
  "trustedDirectories": [
    "/home/doktersmol",
    "/home/doktersmol/Documents"
  ]
}
SETTINGS_EOF
```

### 6.2 Verify settings.json is Valid JSON

```bash
cat ~/.claude-test/settings.json | python3 -m json.tool > /dev/null && echo "SUCCESS: settings.json is valid JSON" || echo "FAILED: settings.json has JSON syntax errors"
```

### 6.3 Create Required Runtime Directories

```bash
mkdir -p ~/.goyoke
touch ~/.goyoke/failure-tracker.jsonl
```

---

## 7. Run Claude with Test Config

### Method A: Directory Swap (Recommended for Testing)

This method temporarily swaps your config directories.

#### Step 1: Close any running Claude sessions

```bash
# Ensure no Claude processes are running
pkill -f "claude" 2>/dev/null || true
```

#### Step 2: Swap directories

```bash
# Backup production config
mv ~/.claude ~/.claude-production-backup

# Activate test config
mv ~/.claude-test ~/.claude
```

#### Step 3: Run Claude

```bash
claude
```

#### Step 4: After testing, restore production config

```bash
# Deactivate test config
mv ~/.claude ~/.claude-test

# Restore production config
mv ~/.claude-production-backup ~/.claude
```

### Method B: Project-Local Config

Create a test project with its own `.claude/` directory:

```bash
# Create test project
mkdir -p ~/goyoke-test-project/.claude

# Copy test config into project
cp -r ~/.claude-test/* ~/goyoke-test-project/.claude/

# Navigate to test project
cd ~/goyoke-test-project

# Run Claude (it will use the local .claude/ directory)
claude
```

---

## 8. Validate Installation

### 8.1 Check SessionStart Hook

When you start a Claude session, you should see the session initialization output.

**Expected behavior:**
- The ASCII art fortress banner appears
- `[Session Init]` line shows detected language and conventions

**If NOT working:**
```bash
# Test the binary directly
echo '{"event":"session_start","cwd":"/tmp"}' | goyoke-load-context
```

### 8.2 Check PreToolUse Hook (Task Validation)

In a Claude session, try to invoke a Task with wrong subagent_type:

**Test prompt:**
```
Use Task tool with subagent_type "Explore" and agent "tech-docs-writer"
```

**Expected behavior:**
- Hook should BLOCK the call
- Error message mentions wrong subagent_type

**If NOT working:**
```bash
# Test the binary directly
echo '{"tool_name":"Task","tool_input":{"subagent_type":"Explore","prompt":"AGENT: tech-docs-writer"}}' | goyoke-validate
```

### 8.3 Check PostToolUse Hook (Sharp Edge Detection)

The sharp edge detector runs after every Bash/Edit/Write/Task.

**Test by checking if files are created:**
```bash
ls -la /tmp/claude-tool-counter-*.log 2>/dev/null
```

**If working:** You'll see a tool counter file.

### 8.4 Check SessionEnd Hook (Archive)

Exit Claude session normally (Ctrl+D or type "exit").

**Check for handoff generation:**
```bash
ls -la ~/.claude/memory/
cat ~/.claude/memory/handoffs.jsonl | tail -1 | python3 -m json.tool
```

**Expected:** A new entry in `handoffs.jsonl` with session metadata.

### 8.5 Full Validation Checklist

Run through this checklist:

- [ ] `claude` starts without errors
- [ ] Session Init banner displays
- [ ] Language detection works (shows Python/Go/R when in appropriate project)
- [ ] Task validation blocks wrong subagent_type
- [ ] Tool counter increments (check `/tmp/claude-tool-counter-*.log`)
- [ ] Session end creates handoff in `~/.claude/memory/`
- [ ] No hook timeout errors in Claude output

---

## 9. Production Cutover

**Only proceed after successful testing.**

### 9.1 Backup Current Production Config

```bash
cp ~/.claude/settings.json ~/.claude/settings.json.backup-$(date +%Y%m%d)
```

### 9.2 Backup Bash Scripts

```bash
mkdir -p ~/.claude/hooks/bash-backup
cp ~/.claude/hooks/*.sh ~/.claude/hooks/bash-backup/
```

### 9.3 Update Production settings.json

Edit `~/.claude/settings.json` to replace bash scripts with Go binaries:

**Find and replace these patterns:**

| Old (Bash) | New (Go Binary) |
|------------|-----------------|
| `$CLAUDE_PROJECT_DIR/.claude/hooks/load-routing-context.sh` | `goyoke-load-context` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/validate-routing.sh` | `goyoke-validate` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/sharp-edge-detector.sh` | `goyoke-sharp-edge` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/attention-gate.sh` | (remove - merged into goyoke-sharp-edge) |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/agent-endstate.sh` | `goyoke-agent-endstate` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/orchestrator-completion-guard.sh` | `goyoke-orchestrator-guard` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/session-archive.sh` | `goyoke-archive` |

### 9.4 Verify Production Config

```bash
cat ~/.claude/settings.json | python3 -m json.tool > /dev/null && echo "SUCCESS" || echo "FAILED"
```

### 9.5 Test Production

```bash
claude
```

Run a few simple commands to verify hooks work.

---

## 10. Rollback Procedure

If something goes wrong, rollback immediately.

### 10.1 Quick Rollback

```bash
# Restore backup settings.json
cp ~/.claude/settings.json.backup-* ~/.claude/settings.json

# Restart Claude
claude
```

### 10.2 Full Rollback to Bash Scripts

```bash
# Restore bash scripts from backup
cp ~/.claude/hooks/bash-backup/*.sh ~/.claude/hooks/

# Edit settings.json to use bash scripts again (manually or restore backup)
cp ~/.claude/settings.json.backup-* ~/.claude/settings.json
```

---

## 11. Troubleshooting

### Problem: "command not found: goyoke-validate"

**Cause:** `~/.local/bin` is not in PATH.

**Fix:**
```bash
export PATH="$HOME/.local/bin:$PATH"
# Add to shell config permanently
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Problem: Hook timeout errors

**Cause:** Binary is slow or hanging.

**Fix:** Increase timeout in settings.json:
```json
{
  "type": "command",
  "command": "goyoke-validate",
  "timeout": 30  // Increased from 10
}
```

### Problem: "invalid JSON" errors from hooks

**Cause:** Binary is outputting non-JSON or error messages to stdout.

**Debug:**
```bash
echo '{"tool_name":"Task","tool_input":{}}' | goyoke-validate 2>&1
```

Check stderr for errors separate from stdout JSON.

### Problem: Hooks not triggering at all

**Cause:** settings.json hook configuration syntax error.

**Debug:**
```bash
# Validate JSON syntax
cat ~/.claude/settings.json | python3 -m json.tool

# Check hook matchers match your tool usage
grep -A5 '"hooks"' ~/.claude/settings.json
```

### Problem: Sharp edges not being captured

**Cause:** Missing `~/.goyoke/` directory or permissions issue.

**Fix:**
```bash
mkdir -p ~/.goyoke
chmod 755 ~/.goyoke
touch ~/.goyoke/failure-tracker.jsonl
chmod 644 ~/.goyoke/failure-tracker.jsonl
```

### Problem: Session handoffs not generated

**Cause:** `goyoke-archive` failing silently.

**Debug:**
```bash
# Test archive binary directly
echo '{"event":"session_end","session_id":"test-123"}' | goyoke-archive

# Check for memory directory
ls -la ~/.claude/memory/
```

### Problem: Permission denied errors

**Cause:** Binaries not executable.

**Fix:**
```bash
chmod +x ~/.local/bin/goyoke-*
```

---

## Quick Reference Card

### Build & Install
```bash
cd ~/Documents/goYoke
make build-all
make install
```

### Test Environment Setup
```bash
mkdir -p ~/.claude-test/{hooks,memory,tmp}
cp ~/.claude/{CLAUDE.md,routing-schema.json} ~/.claude-test/
cp -r ~/.claude/{agents,conventions,rules,skills} ~/.claude-test/
# Create settings.json (see Section 6.1)
```

### Directory Swap for Testing
```bash
mv ~/.claude ~/.claude-backup && mv ~/.claude-test ~/.claude  # Activate test
mv ~/.claude ~/.claude-test && mv ~/.claude-backup ~/.claude  # Restore
```

### Verify Installation
```bash
which goyoke-validate goyoke-load-context goyoke-sharp-edge goyoke-archive
echo '{}' | goyoke-validate  # Should output JSON
```

### Rollback
```bash
cp ~/.claude/settings.json.backup-* ~/.claude/settings.json
```

---

## 12. Running as `claudeGO` Command

This section explains how to run Claude with goYoke hooks using the command `claudeGO`, while keeping your production `claude` command unchanged.

### 12.1 How It Works

```
claude     → Uses ~/.claude/ (your production bash-based config)
claudeGO   → Uses ~/.claude-goyoke/ (goYoke Go binaries)
```

Both commands run the same Claude Code CLI, but with different configuration directories.

### 12.2 Prerequisites

Before proceeding, ensure:
- [ ] You completed sections 1-4 (binaries built and installed)
- [ ] You completed section 5-6 (test environment created at `~/.claude-test/`)
- [ ] You validated the installation (section 8)

### 12.3 Create Permanent goYoke Config Directory

Move your test config to a permanent location:

```bash
# If you still have ~/.claude-test from earlier testing
mv ~/.claude-test ~/.claude-goyoke

# OR if starting fresh, create it now
mkdir -p ~/.claude-goyoke/{hooks,memory,tmp,session-archive}
cp ~/.claude/CLAUDE.md ~/.claude-goyoke/
cp ~/.claude/routing-schema.json ~/.claude-goyoke/
cp -r ~/.claude/agents ~/.claude-goyoke/
cp -r ~/.claude/conventions ~/.claude-goyoke/
cp -r ~/.claude/rules ~/.claude-goyoke/
cp -r ~/.claude/skills ~/.claude-goyoke/
cp -r ~/.claude/docs ~/.claude-goyoke/ 2>/dev/null || true
```

### 12.4 Create the Wrapper Script

Create the `claudeGO` wrapper script:

```bash
cat > ~/.local/bin/claudeGO << 'WRAPPER_EOF'
#!/bin/bash
#
# claudeGO - Run Claude Code with goYoke hooks
#
# This wrapper temporarily swaps ~/.claude with ~/.claude-goyoke,
# runs Claude, then restores the original config on exit.
#
# Usage: claudeGO [claude arguments...]
#

set -e

# Configuration
PRODUCTION_CONFIG="$HOME/.claude"
GOYOKE_CONFIG="$HOME/.claude-goyoke"
BACKUP_CONFIG="$HOME/.claude-production-tmp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Cleanup function - ALWAYS restore production config
cleanup() {
    local exit_code=$?

    echo ""
    echo -e "${YELLOW}[claudeGO]${NC} Restoring production config..."

    # Restore production config
    if [[ -d "$BACKUP_CONFIG" ]]; then
        # Move goYoke config back
        if [[ -d "$PRODUCTION_CONFIG" ]]; then
            rm -rf "$GOYOKE_CONFIG"
            mv "$PRODUCTION_CONFIG" "$GOYOKE_CONFIG"
        fi

        # Restore production
        mv "$BACKUP_CONFIG" "$PRODUCTION_CONFIG"
        echo -e "${GREEN}[claudeGO]${NC} Production config restored."
    else
        echo -e "${RED}[claudeGO]${NC} WARNING: Backup not found. Config may be in inconsistent state."
        echo -e "${RED}[claudeGO]${NC} Check ~/.claude and ~/.claude-goyoke manually."
    fi

    exit $exit_code
}

# Register cleanup on ANY exit (normal, error, interrupt)
trap cleanup EXIT INT TERM

# Preflight checks
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}  goYoke - Go Hook Orchestration Framework${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check goYoke config exists
if [[ ! -d "$GOYOKE_CONFIG" ]]; then
    echo -e "${RED}[claudeGO]${NC} ERROR: goYoke config not found at $GOYOKE_CONFIG"
    echo -e "${RED}[claudeGO]${NC} Run the installation steps first. See INSTALL-GUIDE.md section 12.3"
    exit 1
fi

# Check settings.json exists in goYoke config
if [[ ! -f "$GOYOKE_CONFIG/settings.json" ]]; then
    echo -e "${RED}[claudeGO]${NC} ERROR: settings.json not found in $GOYOKE_CONFIG"
    echo -e "${RED}[claudeGO]${NC} Create settings.json per INSTALL-GUIDE.md section 6.1"
    exit 1
fi

# Check production config exists
if [[ ! -d "$PRODUCTION_CONFIG" ]]; then
    echo -e "${RED}[claudeGO]${NC} ERROR: Production config not found at $PRODUCTION_CONFIG"
    echo -e "${RED}[claudeGO]${NC} This is unusual. Is Claude Code installed?"
    exit 1
fi

# Check no backup already exists (previous run crashed?)
if [[ -d "$BACKUP_CONFIG" ]]; then
    echo -e "${YELLOW}[claudeGO]${NC} WARNING: Found stale backup at $BACKUP_CONFIG"
    echo -e "${YELLOW}[claudeGO]${NC} A previous claudeGO session may have crashed."
    echo ""
    echo -e "${YELLOW}[claudeGO]${NC} Options:"
    echo -e "  1. Restore production: mv $BACKUP_CONFIG $PRODUCTION_CONFIG"
    echo -e "  2. Delete stale backup: rm -rf $BACKUP_CONFIG"
    echo ""
    read -p "Delete stale backup and continue? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$BACKUP_CONFIG"
        echo -e "${GREEN}[claudeGO]${NC} Stale backup removed."
    else
        echo -e "${RED}[claudeGO]${NC} Aborting. Resolve manually."
        exit 1
    fi
fi

# Check binaries are installed
REQUIRED_BINARIES=(
    "goyoke-validate"
    "goyoke-load-context"
    "goyoke-sharp-edge"
    "goyoke-archive"
    "goyoke-agent-endstate"
    "goyoke-orchestrator-guard"
)

MISSING_BINARIES=()
for bin in "${REQUIRED_BINARIES[@]}"; do
    if ! command -v "$bin" &> /dev/null; then
        MISSING_BINARIES+=("$bin")
    fi
done

if [[ ${#MISSING_BINARIES[@]} -gt 0 ]]; then
    echo -e "${RED}[claudeGO]${NC} ERROR: Missing binaries:"
    for bin in "${MISSING_BINARIES[@]}"; do
        echo -e "  - $bin"
    done
    echo ""
    echo -e "${RED}[claudeGO]${NC} Run: cd ~/Documents/goYoke && make install"
    exit 1
fi

echo -e "${GREEN}[claudeGO]${NC} Preflight checks passed."
echo ""

# Swap configurations
echo -e "${YELLOW}[claudeGO]${NC} Activating goYoke config..."

# Step 1: Move production to backup
mv "$PRODUCTION_CONFIG" "$BACKUP_CONFIG"

# Step 2: Move goYoke to production location
mv "$GOYOKE_CONFIG" "$PRODUCTION_CONFIG"

echo -e "${GREEN}[claudeGO]${NC} goYoke active."
echo -e "${GREEN}[claudeGO]${NC} Starting Claude..."
echo ""

# Run Claude with any passed arguments
claude "$@"

# cleanup() runs automatically on exit via trap
WRAPPER_EOF
```

### 12.5 Make the Script Executable

```bash
chmod +x ~/.local/bin/claudeGO
```

### 12.6 Verify the Script Exists

```bash
which claudeGO
```

**Expected output:**
```
/home/YOUR_USERNAME/.local/bin/claudeGO
```

### 12.7 Ensure settings.json Exists in goYoke Config

If you haven't already created `~/.claude-goyoke/settings.json`, do it now:

```bash
cat > ~/.claude-goyoke/settings.json << 'SETTINGS_EOF'
{
  "permissions": {
    "allow": [
      "Read(**)",
      "Glob(**)",
      "Grep(**)",
      "Bash(wc:*)",
      "Bash(find:*)",
      "Bash(head:*)",
      "Bash(tail:*)",
      "Bash(cat:*)",
      "Bash(ls:*)",
      "Bash(stat:*)",
      "Bash(git status:*)",
      "Bash(git diff:*)",
      "Bash(git log:*)",
      "Bash(go test:*)",
      "Bash(go build:*)",
      "Bash(make:*)"
    ],
    "deny": [
      "Write(.env*)",
      "Write(**/secrets/**)",
      "Bash(*rm -rf*)",
      "Bash(*> /dev/*)",
      "Bash(*sudo*)"
    ]
  },
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-load-context",
            "timeout": 10
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Task",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-validate",
            "timeout": 10
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|Task",
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-sharp-edge",
            "timeout": 5
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-agent-endstate",
            "timeout": 15
          },
          {
            "type": "command",
            "command": "goyoke-orchestrator-guard",
            "timeout": 10
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "goyoke-archive",
            "timeout": 30
          }
        ]
      }
    ]
  },
  "trustedDirectories": [
    "/home/doktersmol",
    "/home/doktersmol/Documents"
  ]
}
SETTINGS_EOF
```

### 12.8 Test the claudeGO Command

```bash
claudeGO
```

**Expected behavior:**
1. You see the goYoke banner
2. Preflight checks pass
3. "goYoke active" message appears
4. Claude starts normally
5. Session Init shows with language detection
6. When you exit (Ctrl+D), production config is automatically restored

### 12.9 Verify Restoration After Exit

After exiting claudeGO, verify your production config is restored:

```bash
# Check production config is back
ls ~/.claude/settings.json

# Check goYoke config is back in its place
ls ~/.claude-goyoke/settings.json

# Both should exist
```

### 12.10 Usage Examples

```bash
# Start claudeGO in current directory
claudeGO

# Start claudeGO with a specific prompt
claudeGO -p "Explain this codebase"

# Start claudeGO in a specific directory
cd ~/my-project && claudeGO

# Continue a previous session (if Claude supports it)
claudeGO --continue
```

### 12.11 What Happens If claudeGO Crashes?

The wrapper has a cleanup trap that runs on ANY exit, including:
- Normal exit (Ctrl+D, typing "exit")
- Error exit (Claude crashes)
- Interrupt (Ctrl+C)
- Kill signal

If the cleanup somehow fails, you'll see a warning next time you run `claudeGO`. Follow the prompts to resolve.

**Manual recovery if needed:**
```bash
# Check current state
ls -la ~/.claude ~/.claude-goyoke ~/.claude-production-tmp

# If ~/.claude-production-tmp exists, production config is there
mv ~/.claude ~/.claude-goyoke
mv ~/.claude-production-tmp ~/.claude
```

### 12.12 Updating goYoke

When you update the Go binaries, both `claude` (if using Go) and `claudeGO` will use the new versions:

```bash
cd ~/Documents/goYoke
git pull
make build-all
make install
```

The `~/.claude-goyoke/` config directory remains unchanged; only the binaries in `~/.local/bin/` are updated.

### 12.13 Quick Reference

| Command | Config Used | Hooks |
|---------|-------------|-------|
| `claude` | `~/.claude/` | Your production config (bash or Go) |
| `claudeGO` | `~/.claude-goyoke/` (swapped to `~/.claude/`) | goYoke Go binaries |

### 12.14 Alternative: Shell Alias (Simpler but Less Safe)

If you prefer a simpler approach without the safety features, you can use an alias:

```bash
# Add to ~/.bashrc or ~/.zshrc
alias claudeGO='mv ~/.claude ~/.claude-bak && mv ~/.claude-goyoke ~/.claude && claude; mv ~/.claude ~/.claude-goyoke && mv ~/.claude-bak ~/.claude'
```

**WARNING:** This alias has no error handling. If Claude crashes, your config will be in the wrong state. The wrapper script (section 12.4) is strongly recommended.

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.1.0 | 2026-01-25 | Added claudeGO command (section 12) |
| 1.0.0 | 2026-01-25 | Initial release |

---

**Questions?** See `docs/systems-architecture-overview.md` for technical details or open an issue on the repository.


---

## See Also

- [[concepts/hook-system]] — Hook architecture
- [[hook-configuration]] — Hook setup details
- [[concepts/distribution-model]] — Planned single-binary distribution
