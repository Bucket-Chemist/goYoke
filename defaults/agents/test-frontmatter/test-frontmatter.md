---
name: Test Frontmatter Agent
description: Agent to test if Claude Code ignores unknown frontmatter fields
model: haiku
x-goyoke-tier: haiku
x-goyoke-custom-field: "this should be ignored"
unknown_field_123: "testing tolerance"
tools:
  - Read
  - Glob
---

# Test Frontmatter Agent

This is a test agent to validate that Claude Code's native agent discovery ignores unknown YAML frontmatter fields.

If Claude Code loads this agent without errors or warnings, the assumption is validated.
