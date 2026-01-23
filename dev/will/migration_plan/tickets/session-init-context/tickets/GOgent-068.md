---
id: GOgent-068
title: Create SessionStart Test Fixtures
description: **Task**:
status: pending
time_estimate: 2h
dependencies: [\n  - GOgent-067]
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 8
---

## GOgent-068: Create SessionStart Test Fixtures

**Time**: 2 hours
**Dependencies**: GOgent-067
**Priority**: HIGH

**Task**:
Create deterministic test fixtures for SessionStart scenarios.

**Directory**: `test/simulation/fixtures/deterministic/sessionstart/`

**Fixture: startup-basic.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-001",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "SESSION INITIALIZED (startup)",
      "hooks are ACTIVE"
    ],
    "additional_context_not_contains": [
      "PREVIOUS SESSION HANDOFF",
      "ERROR"
    ]
  }
}
```

**Fixture: startup-go-project.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-002",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "files": {
      "go.mod": "module test\n\ngo 1.21"
    }
  },
  "expected": {
    "exit_code": 0,
    "project_type_equals": "go",
    "additional_context_contains": [
      "PROJECT TYPE: go",
      "go.mod"
    ]
  }
}
```

**Fixture: startup-python-project.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-003",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "files": {
      "pyproject.toml": "[project]\nname = \"test\"\nversion = \"1.0.0\""
    }
  },
  "expected": {
    "exit_code": 0,
    "project_type_equals": "python",
    "additional_context_contains": [
      "PROJECT TYPE: python",
      "pyproject.toml"
    ]
  }
}
```

**Fixture: startup-r-shiny-project.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-004",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "files": {
      "DESCRIPTION": "Package: myapp\nTitle: Shiny App\nVersion: 1.0.0\nImports: shiny",
      "app.R": "library(shiny)\nshinyApp(ui, server)"
    }
  },
  "expected": {
    "exit_code": 0,
    "project_type_equals": "r-shiny",
    "additional_context_contains": [
      "PROJECT TYPE: r-shiny",
      "R.md",
      "R-shiny.md"
    ]
  }
}
```

**Fixture: resume-with-handoff.json**
```json
{
  "input": {
    "type": "resume",
    "session_id": "sim-resume-001",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/last-handoff.md": "# Session Handoff\n\n## Last Session\nImplemented feature XYZ.\n\n## Next Steps\n- Complete testing"
    }
  },
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "SESSION INITIALIZED (resume)",
      "PREVIOUS SESSION HANDOFF",
      "feature XYZ"
    ]
  }
}
```

**Fixture: resume-no-handoff.json**
```json
{
  "input": {
    "type": "resume",
    "session_id": "sim-resume-002",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "SESSION INITIALIZED (resume)"
    ],
    "additional_context_not_contains": [
      "PREVIOUS SESSION HANDOFF"
    ]
  }
}
```

**Fixture: startup-with-pending-learnings.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-005",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "create_dirs": [".claude/memory"],
    "files": {
      ".claude/memory/pending-learnings.jsonl": "{\"file\":\"test.go\",\"error_type\":\"type_mismatch\",\"consecutive_failures\":3,\"timestamp\":1705000000}\n{\"file\":\"main.go\",\"error_type\":\"nil_pointer\",\"consecutive_failures\":3,\"timestamp\":1705000010}\n"
    }
  },
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "PENDING LEARNINGS",
      "2 sharp edge"
    ]
  }
}
```

**Fixture: startup-git-repo.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-006",
    "hook_event_name": "SessionStart"
  },
  "setup": {
    "create_dirs": [".git"],
    "files": {
      ".git/HEAD": "ref: refs/heads/feature-branch",
      ".git/config": "[core]\nrepositoryformatversion = 0"
    }
  },
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "GIT:"
    ]
  }
}
```

**Fixture: startup-empty-input.json**
```json
{
  "input": {},
  "expected": {
    "exit_code": 0,
    "additional_context_contains": [
      "SESSION INITIALIZED (startup)"
    ]
  }
}
```

**Fixture: startup-tool-counter.json**
```json
{
  "input": {
    "type": "startup",
    "session_id": "sim-startup-007",
    "hook_event_name": "SessionStart"
  },
  "expected": {
    "exit_code": 0,
    "tool_counter_initialized": true
  }
}
```

**Acceptance Criteria**:
- [ ] 10 fixture files created in `test/simulation/fixtures/deterministic/sessionstart/`
- [ ] Fixtures cover: startup, resume, project detection, pending learnings, git status
- [ ] Each fixture has valid JSON input and expected output
- [ ] Setup sections create required directories and files
- [ ] All fixtures pass when run against `gogent-load-context`

**Test Deliverables**:
- [ ] Files created: 10 JSON fixtures
- [ ] Manual verification: `go run ./test/simulation/harness/cmd/harness -mode=deterministic -filter=sim-startup`
- [ ] All fixtures passing: ✅

**Why This Matters**: Deterministic fixtures form the foundation of L1 testing and provide reproducible regression tests.

---
