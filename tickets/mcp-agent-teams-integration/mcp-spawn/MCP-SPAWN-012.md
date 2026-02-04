```yaml
---
id: MCP-SPAWN-012
title: Integration Testing and Documentation
description: End-to-end testing of MCP spawning and documentation updates.
status: pending
time_estimate: 4h
dependencies: [MCP-SPAWN-010, MCP-SPAWN-011]
phase: 3
tags: [testing, documentation, phase-3]
needs_planning: false
agent: typescript-pro
priority: HIGH
coverage_target: 80
---
```

# MCP-SPAWN-012: Integration Testing and Documentation

## Description

End-to-end testing of the complete MCP spawning system and documentation updates for CLAUDE.md and troubleshooting guide.

**Source**: Staff-Architect Analysis §4.5.2, §4.7.2

## Task

1. Create E2E test for Braintrust workflow
2. Create E2E test for /review workflow
3. Test timeout and error scenarios
4. Update CLAUDE.md with spawning documentation
5. Create troubleshooting guide

## Files

- `packages/tui/tests/e2e/braintrust.test.ts` — E2E test
- `packages/tui/tests/e2e/review.test.ts` — E2E test
- `~/.claude/CLAUDE.md` — Documentation update
- `~/.claude/docs/mcp-spawning-troubleshooting.md` — New guide

## Acceptance Criteria

- [ ] E2E Braintrust test passes (may use mock CLI for CI)
- [ ] E2E Review test passes
- [ ] Timeout scenarios tested
- [ ] Error propagation tested
- [ ] CLAUDE.md updated with spawn_agent documentation
- [ ] Troubleshooting guide created
- [ ] 3 successful real Braintrust runs without intervention

