# GOgent-Fortress Installation Guide

> **Target Audience:** Users migrating from bash-based Claude Code hooks to GOgent-Fortress Go binaries.
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

---

## 1. Overview

### What This Guide Does

This guide helps you replace your existing bash-based Claude Code hooks with compiled Go binaries from GOgent-Fortress.

### Architecture Change

**Before (Bash Scripts):**
```
Claude Code → ~/.claude/hooks/validate-routing.sh → routing-schema.json
```

**After (Go Binaries):**
```
Claude Code → gogent-validate (binary) → routing-schema.json
```

### Binary to Hook Mapping

| Hook Event | Bash Script | Go Binary |
|------------|-------------|-----------|
| SessionStart | `load-routing-context.sh` | `gogent-load-context` |
| PreToolUse (Task) | `validate-routing.sh` | `gogent-validate` |
| PostToolUse | `sharp-edge-detector.sh` | `gogent-sharp-edge` |
| PostToolUse | `attention-gate.sh` | (merged into `gogent-sharp-edge`) |
| SubagentStop | `agent-endstate.sh` | `gogent-agent-endstate` |
| SubagentStop | `orchestrator-completion-guard.sh` | `gogent-orchestrator-guard` |
| SessionEnd | `session-archive.sh` | `gogent-archive` |

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
cd ~/Documents/GOgent-Fortress
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
Building gogent-validate binary...
✅ Binary created at bin/gogent-validate
Building gogent-archive binary...
✅ Binary created at bin/gogent-archive
Building gogent-sharp-edge binary...
✅ Binary created at bin/gogent-sharp-edge
Building gogent-load-context...
✓ Built: bin/gogent-load-context
Building gogent-agent-endstate...
✓ Built: bin/gogent-agent-endstate
Building gogent-orchestrator-guard...
✓ Built: bin/gogent-orchestrator-guard
Building gogent-doc-theater...
✓ Built: bin/gogent-doc-theater
✓ All hook binaries built
```

### 3.4 Verify Binaries Exist

```bash
ls -la bin/
```

**Expected output (7 binaries):**
```
-rwxr-xr-x 1 user user XXXXXXX ... gogent-agent-endstate
-rwxr-xr-x 1 user user XXXXXXX ... gogent-archive
-rwxr-xr-x 1 user user XXXXXXX ... gogent-doc-theater
-rwxr-xr-x 1 user user XXXXXXX ... gogent-load-context
-rwxr-xr-x 1 user user XXXXXXX ... gogent-orchestrator-guard
-rwxr-xr-x 1 user user XXXXXXX ... gogent-sharp-edge
-rwxr-xr-x 1 user user XXXXXXX ... gogent-validate
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
Installing GOgent-Fortress CLIs to ~/.local/bin/...
✅ Installed gogent-validate, gogent-archive, gogent-aggregate, gogent-sharp-edge, gogent-capture-intent, gogent-load-context, gogent-agent-endstate, gogent-orchestrator-guard, gogent-doc-theater
✅ ~/.local/bin is in PATH
```

### 4.2 Verify Installation

Run each binary to confirm it's accessible:

```bash
gogent-validate --help 2>/dev/null || echo '{}' | gogent-validate
gogent-load-context --help 2>/dev/null || echo '{}' | gogent-load-context
gogent-sharp-edge --help 2>/dev/null || echo '{}' | gogent-sharp-edge
gogent-archive --help 2>/dev/null || echo '{}' | gogent-archive
gogent-agent-endstate --help 2>/dev/null || echo '{}' | gogent-agent-endstate
gogent-orchestrator-guard --help 2>/dev/null || echo '{}' | gogent-orchestrator-guard
```

Each should either print help or produce JSON output (not "command not found").

### 4.3 Verify Binary Locations

```bash
which gogent-validate gogent-load-context gogent-sharp-edge gogent-archive gogent-agent-endstate gogent-orchestrator-guard
```

**Expected output:**
```
/home/YOUR_USERNAME/.local/bin/gogent-validate
/home/YOUR_USERNAME/.local/bin/gogent-load-context
/home/YOUR_USERNAME/.local/bin/gogent-sharp-edge
/home/YOUR_USERNAME/.local/bin/gogent-archive
/home/YOUR_USERNAME/.local/bin/gogent-agent-endstate
/home/YOUR_USERNAME/.local/bin/gogent-orchestrator-guard
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
ln -sf ~/.local/bin/gogent-validate ~/.claude-test/hooks/gogent-validate
ln -sf ~/.local/bin/gogent-load-context ~/.claude-test/hooks/gogent-load-context
ln -sf ~/.local/bin/gogent-sharp-edge ~/.claude-test/hooks/gogent-sharp-edge
ln -sf ~/.local/bin/gogent-archive ~/.claude-test/hooks/gogent-archive
ln -sf ~/.local/bin/gogent-agent-endstate ~/.claude-test/hooks/gogent-agent-endstate
ln -sf ~/.local/bin/gogent-orchestrator-guard ~/.claude-test/hooks/gogent-orchestrator-guard
ln -sf ~/.local/bin/gogent-doc-theater ~/.claude-test/hooks/gogent-doc-theater
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
            "command": "gogent-load-context",
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
            "command": "gogent-validate",
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
            "command": "gogent-sharp-edge",
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
            "command": "gogent-agent-endstate",
            "timeout": 15
          },
          {
            "type": "command",
            "command": "gogent-orchestrator-guard",
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
            "command": "gogent-archive",
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
mkdir -p ~/.gogent
touch ~/.gogent/failure-tracker.jsonl
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
mkdir -p ~/gogent-test-project/.claude

# Copy test config into project
cp -r ~/.claude-test/* ~/gogent-test-project/.claude/

# Navigate to test project
cd ~/gogent-test-project

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
echo '{"event":"session_start","cwd":"/tmp"}' | gogent-load-context
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
echo '{"tool_name":"Task","tool_input":{"subagent_type":"Explore","prompt":"AGENT: tech-docs-writer"}}' | gogent-validate
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
| `$CLAUDE_PROJECT_DIR/.claude/hooks/load-routing-context.sh` | `gogent-load-context` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/validate-routing.sh` | `gogent-validate` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/sharp-edge-detector.sh` | `gogent-sharp-edge` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/attention-gate.sh` | (remove - merged into gogent-sharp-edge) |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/agent-endstate.sh` | `gogent-agent-endstate` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/orchestrator-completion-guard.sh` | `gogent-orchestrator-guard` |
| `$CLAUDE_PROJECT_DIR/.claude/hooks/session-archive.sh` | `gogent-archive` |

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

### Problem: "command not found: gogent-validate"

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
  "command": "gogent-validate",
  "timeout": 30  // Increased from 10
}
```

### Problem: "invalid JSON" errors from hooks

**Cause:** Binary is outputting non-JSON or error messages to stdout.

**Debug:**
```bash
echo '{"tool_name":"Task","tool_input":{}}' | gogent-validate 2>&1
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

**Cause:** Missing `~/.gogent/` directory or permissions issue.

**Fix:**
```bash
mkdir -p ~/.gogent
chmod 755 ~/.gogent
touch ~/.gogent/failure-tracker.jsonl
chmod 644 ~/.gogent/failure-tracker.jsonl
```

### Problem: Session handoffs not generated

**Cause:** `gogent-archive` failing silently.

**Debug:**
```bash
# Test archive binary directly
echo '{"event":"session_end","session_id":"test-123"}' | gogent-archive

# Check for memory directory
ls -la ~/.claude/memory/
```

### Problem: Permission denied errors

**Cause:** Binaries not executable.

**Fix:**
```bash
chmod +x ~/.local/bin/gogent-*
```

---

## Quick Reference Card

### Build & Install
```bash
cd ~/Documents/GOgent-Fortress
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
which gogent-validate gogent-load-context gogent-sharp-edge gogent-archive
echo '{}' | gogent-validate  # Should output JSON
```

### Rollback
```bash
cp ~/.claude/settings.json.backup-* ~/.claude/settings.json
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-01-25 | Initial release |

---

**Questions?** See `docs/systems-architecture-overview.md` for technical details or open an issue on the repository.
