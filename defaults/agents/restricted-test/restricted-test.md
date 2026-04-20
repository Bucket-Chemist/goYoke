---
name: Restricted Test Agent
description: Agent with tool restrictions to test pipe mode behavior
model: haiku
tools:
  - Read
---

# Restricted Test Agent

This agent should ONLY have access to the Read tool.

If pipe mode respects this, Bash should be blocked.
