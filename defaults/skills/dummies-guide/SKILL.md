---
name: dummies-guide
description: This skill should be used when the user needs help understanding their Claude Code configuration system. It displays a comprehensive guide covering the orchestrator, rules, skills, and how to extend the system. Invoke with /dummies_guide or when user asks "how does my config work", "explain my setup", or "I forgot how this works".
---

# Dummies Guide - Claude Code Configuration Reference

## Overview

This skill displays the comprehensive system guide that explains how your Claude Code configuration works, including the orchestrator pattern, rule files, skills, and how to extend everything.

## Workflow

### Step 1: Announce

When invoked, announce:
```
[dummies-guide] Loading system configuration guide...
```

### Step 2: Read and Present the Guide

Read the full guide from `~/.claude/docs/system-guide.md` using the Read tool.

Present the content to the user. The guide covers:

1. **The Big Picture** - Three-layer architecture (Orchestrator -> Rules -> Project)
2. **Directory Structure** - Where everything lives
3. **How the Orchestrator Works** - Session init, debug 66, always-active guidelines
4. **Understanding Rule Files** - Types, anatomy, preloaded vs on-demand
5. **Creating New Rulesets** - Step-by-step for adding languages
6. **Updating the Orchestrator** - How to add languages and invocation patterns
7. **Skills System** - What skills are, where they live
8. **Creating Skills** - Full step-by-step with compound-engineering
9. **Quick Reference Commands** - Cheat sheet
10. **Troubleshooting** - Common issues and fixes

### Step 3: Offer Navigation

After presenting, ask:
```
[dummies-guide] Guide loaded. Would you like me to focus on a specific section?
- Orchestrator system
- Creating new rulesets
- Skills and slash commands
- Troubleshooting
```

## Quick Answers

For quick questions without loading the full guide:

**"Where do always-loaded rules go?"**
→ `~/.claude/rules/` (only LLM-guidelines.md lives here now)

**"Where do language conventions go?"**
→ `~/.claude/conventions/` (loaded on-demand based on project type)

**"Where do skills go?"**
→ `~/.claude/plugins/marketplaces/every-marketplace/plugins/compound-engineering/skills/`
→ Also copy to: `~/.claude/plugins/cache/every-marketplace/compound-engineering/*/skills/`

**"How do I add a new language?"**
→ 1. Create `~/.claude/conventions/newlang.md`
→ 2. Update `~/.claude/CLAUDE.md` with detection and loading instructions
→ 3. Optionally create `~/.claude/conventions/debug66/newlang.md`
→ 4. Update `/init-auto` skill for the new language

**"How do I create a skill?"**
→ Run: `python3 ~/.claude/plugins/cache/every-marketplace/compound-engineering/*/skills/skill-creator/scripts/init_skill.py skill-name --path ~/.claude/skills/`
→ Edit SKILL.md, delete unused folders
→ Copy to plugin skills directory
→ Restart Claude Code

## File Location

The full guide is stored at: `~/.claude/docs/system-guide.md`

To read it manually:
```bash
cat ~/.claude/docs/system-guide.md
```

Or open in editor:
```bash
code ~/.claude/docs/system-guide.md
```
