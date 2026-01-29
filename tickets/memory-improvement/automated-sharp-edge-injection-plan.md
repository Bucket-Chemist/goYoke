---
id: GOgent-MEM-002
title: "Automated Sharp Edge Injection System"
version: 2.0
time: "20-30 hours"
priority: HIGH (after MEM-001 proven for 3+ months)
dependencies: "GOgent-MEM-001 (must be deployed and producing quality recommendations)"
status: vision
created: 2026-01-30
target_date: "Q3 2026 (6+ months post MEM-001)"
---

# GOgent-MEM-002: Automated Sharp Edge Injection System

**Vision:** Close the learning loop by automatically updating agent sharp-edges.yaml files from Gemini recommendations, transforming GOgent-Fortress from "memory-aware" to "self-improving."

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Problem Statement](#problem-statement)
3. [Architecture Vision](#architecture-vision)
4. [Phase 1: Parser & Validator](#phase-1-parser--validator)
5. [Phase 2: Injection Engine](#phase-2-injection-engine)
6. [Phase 3: Approval Workflow](#phase-3-approval-workflow)
7. [Phase 4: Safety & Rollback](#phase-4-safety--rollback)
8. [Integration Points](#integration-points)
9. [Risk Mitigation](#risk-mitigation)
10. [Success Metrics](#success-metrics)
11. [Rollout Strategy](#rollout-strategy)

---

## Executive Summary

### The Opportunity

With GOgent-MEM-001 deployed, the system can:
- ✅ Detect recurring problems (via solved-problems.jsonl)
- ✅ Generate sharp edge recommendations (via Gemini /memory-improvement)
- ❌ **Cannot auto-apply recommendations** (manual copy-paste required)

**This enhancement automates the final step:** Gemini recommendations → agent sharp-edges.yaml updates.

### Value Proposition

| Metric | Current (Manual) | With Automation | Improvement |
|--------|------------------|-----------------|-------------|
| Time to add sharp edge | 15-30 min (find file, edit, format, test) | 2-5 min (review + approve) | **6-15x faster** |
| Sharp edges added/month | 2-5 (manual effort bottleneck) | 10-20 (no bottleneck) | **4-5x more** |
| Quality consistency | Variable (manual formatting errors) | High (validated schema) | **Eliminates errors** |
| Knowledge compounding | Linear (manual curation) | Exponential (automated feedback) | **Step function** |

### Phased Delivery

| Phase | Time | MVP Deliverable | Full Deliverable |
|-------|------|-----------------|------------------|
| 1. Parser & Validator | 8h | Parse Gemini YAML, validate schema | + Conflict detection, diff preview |
| 2. Injection Engine | 8h | Insert sharp edge into YAML | + Preserve formatting, handle edge cases |
| 3. Approval Workflow | 6h | CLI approval prompt | + Git integration, batch operations |
| 4. Safety & Rollback | 8h | Pre-injection backup | + Validation suite, auto-rollback |
| **Total** | **30h** | **MVP: 12h** | **Full: 30h** |

**MVP Scope (12 hours):** Parse, validate, inject with manual git workflow
**Full Scope (30 hours):** + Approval UI, git automation, comprehensive safety

---

## Problem Statement

### Current Workflow (Manual)

```
User runs /memory-improvement
    ↓
Gemini outputs YAML with sharp_edge_recommendations
    ↓
User reads YAML output
    ↓
User manually:
  1. Identifies target agent (e.g., go-pro)
  2. Opens ~/.claude/agents/go-pro/sharp-edges.yaml
  3. Finds appropriate section (category: runtime)
  4. Copies recommendation, reformats if needed
  5. Validates YAML syntax
  6. Tests agent still works
  7. Git commits change
    ↓
Repeat for each recommendation (5-10 recommendations)
    ↓
Total time: 30-60 minutes per /memory-improvement run
```

**Pain points:**
- High friction (7 manual steps per edge)
- Error-prone (YAML syntax errors, wrong section)
- Tedious (copy-paste, formatting)
- Inconsistent (skipped due to effort)

### Desired Workflow (Automated)

```
User runs /memory-improvement
    ↓
Gemini outputs YAML with sharp_edge_recommendations
    ↓
System automatically:
  1. Parses recommendations
  2. Validates schema compliance
  3. Detects conflicts with existing edges
  4. Generates diff preview
    ↓
User reviews diff (one screen, all changes)
    ↓
User: gogent-memory approve-edges
    ↓
System:
  - Injects all approved edges
  - Validates YAML post-injection
  - Creates git commit
  - Runs validation tests
    ↓
Done in 2-5 minutes
```

**Benefits:**
- ✅ 6-15x faster
- ✅ Zero syntax errors (validated)
- ✅ Consistent formatting
- ✅ Safe (pre-backup, post-validation)
- ✅ Auditable (git history)

---

## Architecture Vision

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    /memory-improvement                       │
│                    (Gemini generates YAML)                   │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│              Phase 1: Parser & Validator                     │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐          │
│  │ Parse YAML │→ │ Validate   │→ │ Detect       │          │
│  │ Output     │  │ Schema     │  │ Conflicts    │          │
│  └────────────┘  └────────────┘  └──────────────┘          │
│         ↓                                                    │
│  recommendations.json (intermediate format)                 │
└────────────────────────────┬────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│              Phase 2: Injection Engine                       │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐          │
│  │ Load Agent │→ │ Insert     │→ │ Preserve     │          │
│  │ YAML       │  │ Sharp Edge │  │ Formatting   │          │
│  └────────────┘  └────────────┘  └──────────────┘          │
│         ↓                                                    │
│  staged-changes/ (backup + new YAML)                        │
└────────────────────────────┬────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│              Phase 3: Approval Workflow                      │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐          │
│  │ Show Diff  │→ │ User       │→ │ Git Commit   │          │
│  │ Preview    │  │ Approves   │  │ Changes      │          │
│  └────────────┘  └────────────┘  └──────────────┘          │
└────────────────────────────┬────────────────────────────────┘
                             ↓
┌─────────────────────────────────────────────────────────────┐
│              Phase 4: Safety & Rollback                      │
│  ┌────────────┐  ┌────────────┐  ┌──────────────┐          │
│  │ Validate   │→ │ Test Agent │→ │ Rollback if  │          │
│  │ YAML       │  │ Load       │  │ Fails        │          │
│  └────────────┘  └────────────┘  └──────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
Input: Gemini YAML output
  ↓
recommendations.json (normalized)
  ↓
staged-changes/
  ├── go-pro.sharp-edges.yaml.backup
  ├── go-pro.sharp-edges.yaml.new
  ├── go-pro.diff
  └── injection-plan.json
  ↓
~/.claude/agents/go-pro/sharp-edges.yaml (updated)
  ↓
git commit with metadata
```

### File Locations

| File | Purpose | Location |
|------|---------|----------|
| `recommendations.json` | Parsed Gemini output | `.claude/tmp/edge-injection/` |
| `injection-plan.json` | Approved changes | `.claude/tmp/edge-injection/` |
| `*.backup` | Pre-injection backup | `.claude/tmp/edge-injection/staged-changes/` |
| `*.new` | Staged new YAML | `.claude/tmp/edge-injection/staged-changes/` |
| `*.diff` | Unified diff | `.claude/tmp/edge-injection/staged-changes/` |
| Logs | Injection audit trail | `.claude/memory/edge-injection-log.jsonl` |

---

## Phase 1: Parser & Validator

**Time:** 8 hours
**MVP:** Parse YAML, validate required fields (4 hours)
**Full:** + Conflict detection, diff generation (8 hours)

### 1.1 Input: Gemini YAML Output

**Source:** `/memory-improvement` skill output

**Expected format** (from GOgent-MEM-001 Phase 4):
```yaml
sharp_edge_recommendations:
  - priority: 1
    agent: go-pro
    action: add_sharp_edge
    sharp_edge:
      id: nil-pointer-struct-init
      severity: high
      category: runtime
      description: "Nil pointer when accessing uninitialized struct fields"
      symptom: "panic: runtime error: invalid memory address or nil pointer dereference"
      solution: |
        Always initialize struct with make() or literal:
        user := User{}  // GOOD
        var user *User  // BAD - nil
      auto_inject: true
    evidence:
      occurrence_count: 5
      unique_sessions: 4
      temporal_spread:
        distribution: "distributed"
```

### 1.2 Parser Implementation

**File:** `cmd/gogent-memory/parser.go`

```go
package main

import (
    "fmt"
    "gopkg.in/yaml.v3"
)

// RecommendationInput is the Gemini output format
type RecommendationInput struct {
    SharpEdgeRecommendations []Recommendation `yaml:"sharp_edge_recommendations"`
}

type Recommendation struct {
    Priority   int        `yaml:"priority"`
    Agent      string     `yaml:"agent"`
    Action     string     `yaml:"action"`  // "add_sharp_edge" or "skip"
    Reason     string     `yaml:"reason,omitempty"`  // If action=skip
    SharpEdge  SharpEdge  `yaml:"sharp_edge,omitempty"`
    Evidence   Evidence   `yaml:"evidence"`
}

type SharpEdge struct {
    ID          string `yaml:"id"`
    Severity    string `yaml:"severity"`
    Category    string `yaml:"category"`
    Description string `yaml:"description"`
    Symptom     string `yaml:"symptom"`
    Solution    string `yaml:"solution"`
    AutoInject  bool   `yaml:"auto_inject"`
}

type Evidence struct {
    OccurrenceCount  int              `yaml:"occurrence_count"`
    UniqueSessions   int              `yaml:"unique_sessions"`
    TemporalSpread   TemporalSpread   `yaml:"temporal_spread"`
}

type TemporalSpread struct {
    Distribution string `yaml:"distribution"`  // "distributed" or "clustered"
}

// ParseRecommendations reads Gemini YAML output
func ParseRecommendations(yamlData []byte) (*RecommendationInput, error) {
    var input RecommendationInput
    if err := yaml.Unmarshal(yamlData, &input); err != nil {
        return nil, fmt.Errorf("parse YAML: %w", err)
    }
    return &input, nil
}
```

### 1.3 Validation Logic

**File:** `cmd/gogent-memory/validator.go`

```go
package main

import (
    "fmt"
    "strings"
)

// Validation rules
var (
    ValidAgents     = []string{"go-pro", "go-cli", "go-tui", "go-api", "go-concurrent", "python-pro", "python-ux", "r-pro", "r-shiny-pro"}
    ValidActions    = []string{"add_sharp_edge", "skip"}
    ValidSeverities = []string{"critical", "high", "medium", "low"}
    ValidCategories = []string{"runtime", "build", "test", "concurrency", "type", "logic", "config", "performance"}
)

type ValidationError struct {
    RecommendationIndex int
    Field               string
    Message             string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("recommendation[%d].%s: %s", e.RecommendationIndex, e.Field, e.Message)
}

// ValidateRecommendations checks schema compliance
func ValidateRecommendations(input *RecommendationInput) []error {
    var errors []error

    for i, rec := range input.SharpEdgeRecommendations {
        // Validate agent
        if !contains(ValidAgents, rec.Agent) {
            errors = append(errors, &ValidationError{
                RecommendationIndex: i,
                Field:               "agent",
                Message:             fmt.Sprintf("invalid agent '%s', must be one of: %v", rec.Agent, ValidAgents),
            })
        }

        // Validate action
        if !contains(ValidActions, rec.Action) {
            errors = append(errors, &ValidationError{
                RecommendationIndex: i,
                Field:               "action",
                Message:             fmt.Sprintf("invalid action '%s', must be 'add_sharp_edge' or 'skip'", rec.Action),
            })
        }

        // If action=skip, no further validation needed
        if rec.Action == "skip" {
            continue
        }

        // Validate sharp_edge fields (only for add_sharp_edge)
        if rec.SharpEdge.ID == "" {
            errors = append(errors, &ValidationError{i, "sharp_edge.id", "required field missing"})
        }
        if rec.SharpEdge.Description == "" {
            errors = append(errors, &ValidationError{i, "sharp_edge.description", "required field missing"})
        }
        if rec.SharpEdge.Symptom == "" {
            errors = append(errors, &ValidationError{i, "sharp_edge.symptom", "required field missing"})
        }
        if rec.SharpEdge.Solution == "" {
            errors = append(errors, &ValidationError{i, "sharp_edge.solution", "required field missing"})
        }

        // Validate severity
        if !contains(ValidSeverities, rec.SharpEdge.Severity) {
            errors = append(errors, &ValidationError{
                RecommendationIndex: i,
                Field:               "sharp_edge.severity",
                Message:             fmt.Sprintf("invalid severity '%s'", rec.SharpEdge.Severity),
            })
        }

        // Validate category
        if !contains(ValidCategories, rec.SharpEdge.Category) {
            errors = append(errors, &ValidationError{
                RecommendationIndex: i,
                Field:               "sharp_edge.category",
                Message:             fmt.Sprintf("invalid category '%s'", rec.SharpEdge.Category),
            })
        }

        // Validate ID format (kebab-case, no spaces)
        if strings.Contains(rec.SharpEdge.ID, " ") {
            errors = append(errors, &ValidationError{
                RecommendationIndex: i,
                Field:               "sharp_edge.id",
                Message:             "ID must be kebab-case (no spaces)",
            })
        }
    }

    return errors
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 1.4 Conflict Detection

**File:** `cmd/gogent-memory/conflicts.go`

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

// ExistingSharpEdge represents an edge already in agent YAML
type ExistingSharpEdge struct {
    ID          string `yaml:"id"`
    Category    string `yaml:"category"`
    Description string `yaml:"description"`
}

// ConflictType categorizes the conflict
type ConflictType string

const (
    ConflictDuplicateID          ConflictType = "duplicate_id"
    ConflictSimilarDescription   ConflictType = "similar_description"
    ConflictSameSymptom          ConflictType = "same_symptom"
)

// Conflict represents a detected conflict
type Conflict struct {
    Type                ConflictType
    RecommendationIndex int
    RecommendedID       string
    ExistingID          string
    ExistingCategory    string
    Message             string
}

// DetectConflicts checks if recommendations conflict with existing edges
func DetectConflicts(recommendations []Recommendation, agentName string) ([]Conflict, error) {
    var conflicts []Conflict

    // Load existing sharp edges for this agent
    existingEdges, err := loadExistingSharpEdges(agentName)
    if err != nil {
        return nil, fmt.Errorf("load existing edges: %w", err)
    }

    for i, rec := range recommendations {
        if rec.Action == "skip" {
            continue
        }

        // Check for duplicate ID
        for _, existing := range existingEdges {
            if existing.ID == rec.SharpEdge.ID {
                conflicts = append(conflicts, Conflict{
                    Type:                ConflictDuplicateID,
                    RecommendationIndex: i,
                    RecommendedID:       rec.SharpEdge.ID,
                    ExistingID:          existing.ID,
                    ExistingCategory:    existing.Category,
                    Message: fmt.Sprintf(
                        "Sharp edge ID '%s' already exists in %s/%s",
                        rec.SharpEdge.ID,
                        agentName,
                        existing.Category,
                    ),
                })
            }

            // Check for similar descriptions (fuzzy match)
            if similarity(existing.Description, rec.SharpEdge.Description) > 0.8 {
                conflicts = append(conflicts, Conflict{
                    Type:                ConflictSimilarDescription,
                    RecommendationIndex: i,
                    RecommendedID:       rec.SharpEdge.ID,
                    ExistingID:          existing.ID,
                    Message: fmt.Sprintf(
                        "Sharp edge '%s' very similar to existing '%s'",
                        rec.SharpEdge.ID,
                        existing.ID,
                    ),
                })
            }
        }
    }

    return conflicts, nil
}

func loadExistingSharpEdges(agentName string) ([]ExistingSharpEdge, error) {
    home := os.Getenv("HOME")
    yamlPath := filepath.Join(home, ".claude", "agents", agentName, "sharp-edges.yaml")

    data, err := os.ReadFile(yamlPath)
    if err != nil {
        return nil, err
    }

    // Parse YAML (simplified - actual implementation needs to handle nested structure)
    var edges struct {
        SharpEdges []ExistingSharpEdge `yaml:"sharp_edges"`
    }

    if err := yaml.Unmarshal(data, &edges); err != nil {
        return nil, err
    }

    return edges.SharpEdges, nil
}

// similarity calculates string similarity (Levenshtein distance normalized)
func similarity(a, b string) float64 {
    // Simplified implementation - use levenshtein library in production
    if a == b {
        return 1.0
    }
    // TODO: Implement proper fuzzy matching
    return 0.0
}
```

### 1.5 Output: recommendations.json

**Normalized intermediate format:**

```json
{
  "timestamp": 1738195200,
  "source": "/memory-improvement run 2026-01-30",
  "recommendations": [
    {
      "index": 0,
      "priority": 1,
      "agent": "go-pro",
      "action": "add_sharp_edge",
      "sharp_edge": {
        "id": "nil-pointer-struct-init",
        "severity": "high",
        "category": "runtime",
        "description": "Nil pointer when accessing uninitialized struct fields",
        "symptom": "panic: runtime error: invalid memory address or nil pointer dereference",
        "solution": "Always initialize struct with make() or literal:\nuser := User{}  // GOOD\nvar user *User  // BAD - nil",
        "auto_inject": true
      },
      "evidence": {
        "occurrence_count": 5,
        "unique_sessions": 4
      },
      "validation_status": "valid",
      "conflicts": []
    },
    {
      "index": 1,
      "priority": 2,
      "agent": "go-pro",
      "action": "add_sharp_edge",
      "sharp_edge": {
        "id": "channel-close-race",
        "severity": "critical",
        "category": "concurrency",
        "description": "Race condition closing channels",
        "symptom": "panic: send on closed channel",
        "solution": "Use sync.Once to ensure single close:\nvar closeOnce sync.Once\ncloseOnce.Do(func() { close(ch) })",
        "auto_inject": true
      },
      "validation_status": "valid",
      "conflicts": []
    },
    {
      "index": 2,
      "priority": 3,
      "agent": "go-pro",
      "action": "skip",
      "reason": "Single session dominated - likely one-off debugging loop",
      "validation_status": "valid"
    }
  ],
  "validation_summary": {
    "total": 3,
    "valid": 3,
    "invalid": 0,
    "conflicts": 0,
    "actionable": 2
  }
}
```

---

## Phase 2: Injection Engine

**Time:** 8 hours
**MVP:** Insert sharp edge into YAML (4 hours)
**Full:** + Preserve formatting, handle edge cases (8 hours)

### 2.1 YAML Structure Analysis

**Existing sharp-edges.yaml format:**

```yaml
# ~/.claude/agents/go-pro/sharp-edges.yaml

sharp_edges:
  # Runtime errors
  - id: goroutine-leak-context
    severity: high
    category: runtime
    description: "Goroutines leak when context not canceled"
    symptom: "Memory grows unbounded with concurrent operations"
    solution: |
      Always cancel context when done:
      ctx, cancel := context.WithCancel(parent)
      defer cancel()

  - id: nil-pointer-defer
    severity: medium
    category: runtime
    description: "Deferred function panics on nil receiver"
    symptom: "panic: runtime error: invalid memory address"
    solution: |
      Check nil before deferring:
      if obj != nil {
          defer obj.Close()
      }

  # Concurrency issues
  - id: map-concurrent-write
    severity: critical
    category: concurrency
    description: "Concurrent map writes without mutex"
    symptom: "fatal error: concurrent map writes"
    solution: |
      Use sync.RWMutex:
      var mu sync.RWMutex
      mu.Lock()
      m[key] = value
      mu.Unlock()
```

**Key observations:**
1. Comments separate categories (`# Runtime errors`, `# Concurrency issues`)
2. Each edge is a list item with consistent indentation
3. `solution` field uses YAML multiline string (`|`)
4. Order matters (typically severity-sorted within category)

### 2.2 Insertion Strategy

**Goals:**
1. Preserve existing comments
2. Maintain category grouping
3. Insert in correct position (severity-sorted)
4. Preserve indentation and formatting

**Algorithm:**

```go
package main

import (
    "bytes"
    "fmt"
    "gopkg.in/yaml.v3"
)

// InjectSharpEdge inserts a new edge into existing YAML
func InjectSharpEdge(existingYAML []byte, newEdge SharpEdge, targetAgent string) ([]byte, error) {
    // Parse existing YAML with comment preservation
    var node yaml.Node
    if err := yaml.Unmarshal(existingYAML, &node); err != nil {
        return nil, fmt.Errorf("parse existing YAML: %w", err)
    }

    // Find sharp_edges list
    sharpEdgesNode, err := findNode(&node, "sharp_edges")
    if err != nil {
        return nil, err
    }

    // Find insertion point (by category, then severity)
    insertionIndex := findInsertionPoint(sharpEdgesNode, newEdge)

    // Create new edge node
    newEdgeNode := createEdgeNode(newEdge)

    // Insert at correct position
    sharpEdgesNode.Content = append(
        sharpEdgesNode.Content[:insertionIndex],
        append([]*yaml.Node{newEdgeNode}, sharpEdgesNode.Content[insertionIndex:]...)...,
    )

    // Marshal back to YAML with preserved formatting
    var buf bytes.Buffer
    encoder := yaml.NewEncoder(&buf)
    encoder.SetIndent(2)  // Match existing indentation

    if err := encoder.Encode(&node); err != nil {
        return nil, fmt.Errorf("marshal YAML: %w", err)
    }

    return buf.Bytes(), nil
}

// findInsertionPoint determines where to insert new edge
func findInsertionPoint(sharpEdgesNode *yaml.Node, newEdge SharpEdge) int {
    severityOrder := map[string]int{
        "critical": 0,
        "high":     1,
        "medium":   2,
        "low":      3,
    }

    newSeverityRank := severityOrder[newEdge.Severity]

    // Iterate through existing edges
    for i, edgeNode := range sharpEdgesNode.Content {
        // Parse edge to get category and severity
        var existingEdge ExistingSharpEdge
        if err := edgeNode.Decode(&existingEdge); err != nil {
            continue
        }

        // Same category?
        if existingEdge.Category == newEdge.Category {
            existingSeverityRank := severityOrder[existingEdge.Severity]

            // New edge has higher severity (lower rank)? Insert before
            if newSeverityRank < existingSeverityRank {
                return i
            }
        }

        // Moved past target category? Insert before
        if categoryComesBefore(existingEdge.Category, newEdge.Category) {
            return i
        }
    }

    // No match found - append to end
    return len(sharpEdgesNode.Content)
}

func categoryComesBefore(a, b string) bool {
    categoryOrder := []string{"runtime", "concurrency", "build", "test", "type", "logic", "config", "performance"}

    indexA := indexOf(categoryOrder, a)
    indexB := indexOf(categoryOrder, b)

    return indexA < indexB
}

func indexOf(slice []string, item string) int {
    for i, s := range slice {
        if s == item {
            return i
        }
    }
    return len(slice)  // Not found - place at end
}

func createEdgeNode(edge SharpEdge) *yaml.Node {
    // Create YAML node structure
    node := &yaml.Node{
        Kind: yaml.MappingNode,
    }

    // Add fields with proper formatting
    addField(node, "id", edge.ID)
    addField(node, "severity", edge.Severity)
    addField(node, "category", edge.Category)
    addField(node, "description", edge.Description)
    addField(node, "symptom", edge.Symptom)
    addFieldMultiline(node, "solution", edge.Solution)

    return node
}

func addField(node *yaml.Node, key, value string) {
    node.Content = append(node.Content,
        &yaml.Node{Kind: yaml.ScalarNode, Value: key},
        &yaml.Node{Kind: yaml.ScalarNode, Value: value},
    )
}

func addFieldMultiline(node *yaml.Node, key, value string) {
    node.Content = append(node.Content,
        &yaml.Node{Kind: yaml.ScalarNode, Value: key},
        &yaml.Node{Kind: yaml.ScalarNode, Value: value, Style: yaml.LiteralStyle},  // | multiline
    )
}
```

### 2.3 Comment Preservation

**Challenge:** YAML comments are fragile - standard parsers often strip them.

**Solution:** Use `yaml.v3` with `LineComment` and `HeadComment` preservation:

```go
func preserveComments(node *yaml.Node) {
    // yaml.v3 preserves comments by default
    // But we need to ensure they're maintained during insertion

    for i, child := range node.Content {
        // If this is a category boundary, check for header comment
        if child.LineComment != "" || child.HeadComment != "" {
            // Preserve when inserting new node nearby
            // Implementation depends on insertion position
        }
    }
}
```

**Alternative (if comment preservation is too fragile):**
- Parse YAML into struct
- Inject new edge
- Use template to regenerate YAML with comments manually inserted
- Trade-off: Simpler but loses custom comments

### 2.4 Diff Generation

**File:** `cmd/gogent-memory/diff.go`

```go
package main

import (
    "fmt"
    "os/exec"
)

// GenerateDiff creates unified diff between original and modified YAML
func GenerateDiff(originalPath, modifiedPath string) (string, error) {
    // Use system diff command
    cmd := exec.Command("diff", "-u", originalPath, modifiedPath)

    output, err := cmd.CombinedOutput()

    // diff returns exit code 1 when files differ (not an error)
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
            return string(output), nil
        }
        return "", fmt.Errorf("diff command failed: %w", err)
    }

    // No differences
    return "", nil
}

// FormatDiffForDisplay adds color and context
func FormatDiffForDisplay(diff string) string {
    // Add ANSI color codes
    // + lines: green
    // - lines: red
    // @ lines: cyan

    // Implementation uses terminal color library
    return diff  // Simplified
}
```

---

## Phase 3: Approval Workflow

**Time:** 6 hours
**MVP:** CLI approval prompt (3 hours)
**Full:** + Git integration, batch operations (6 hours)

### 3.1 CLI Command Structure

```bash
# After /memory-improvement generates recommendations
$ gogent-memory approve-edges

Found 3 sharp edge recommendations in latest /memory-improvement run:
  - 2 actionable (add_sharp_edge)
  - 1 skipped (low confidence)

Review recommendations? [y/n] y

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Recommendation 1/2 (priority: 1)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Agent: go-pro
Category: runtime
ID: nil-pointer-struct-init
Severity: high

Description:
  Nil pointer when accessing uninitialized struct fields

Symptom:
  panic: runtime error: invalid memory address or nil pointer dereference

Solution:
  Always initialize struct with make() or literal:
  user := User{}  // GOOD
  var user *User  // BAD - nil

Evidence:
  - Occurred 5 times across 4 sessions (distributed pattern)
  - Avg time to resolve: 45 seconds

Conflicts: None

Diff preview:
  ~/.claude/agents/go-pro/sharp-edges.yaml
  @@ -12,6 +12,14 @@
       defer obj.Close()
     }

  +  - id: nil-pointer-struct-init
  +    severity: high
  +    category: runtime
  +    description: "Nil pointer when accessing uninitialized struct fields"
  +    symptom: "panic: runtime error: invalid memory address or nil pointer dereference"
  +    solution: |
  +      Always initialize struct with make() or literal:
  +      user := User{}  // GOOD
  +      var user *User  // BAD - nil
  +
     # Concurrency issues
     - id: map-concurrent-write

Actions: [a]pprove, [r]eject, [e]dit, [s]kip, [q]uit
> a

✅ Approved (1/2)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Recommendation 2/2 (priority: 2)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Agent: go-pro
Category: concurrency
ID: channel-close-race
Severity: critical

[... similar display ...]

> a

✅ Approved (2/2)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Summary
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Approved: 2
Rejected: 0
Skipped: 0

Apply changes? [y/n] y

Applying changes...
  ✓ Backing up go-pro/sharp-edges.yaml
  ✓ Injecting 2 sharp edges
  ✓ Validating YAML syntax
  ✓ Testing agent load

All validations passed.

Create git commit? [y/n] y

Git commit created:
  Commit: abc1234
  Message: Add 2 sharp edges from memory-improvement

  - nil-pointer-struct-init (high, runtime)
  - channel-close-race (critical, concurrency)

  Evidence: 9 occurrences across 7 sessions
  Source: /memory-improvement 2026-01-30

  Co-Authored-By: Gemini Analysis <noreply@google.com>

Done! 2 sharp edges added to go-pro.

Run 'git show abc1234' to review changes.
```

### 3.2 Implementation

**File:** `cmd/gogent-memory/approve.go`

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

// ApprovalWorkflow guides user through recommendation review
func ApprovalWorkflow(recommendations []Recommendation) error {
    reader := bufio.NewReader(os.Stdin)

    var approved []Recommendation
    var rejected []Recommendation

    fmt.Printf("Found %d recommendations\n", len(recommendations))

    for i, rec := range recommendations {
        if rec.Action == "skip" {
            continue  // Don't show skipped recommendations
        }

        // Display recommendation
        displayRecommendation(i+1, len(recommendations), rec)

        // Show diff
        diff, err := generateDiffPreview(rec)
        if err != nil {
            return err
        }
        fmt.Println(diff)

        // Prompt for action
        fmt.Print("\nActions: [a]pprove, [r]eject, [e]dit, [s]kip, [q]uit\n> ")
        response, _ := reader.ReadString('\n')
        response = strings.TrimSpace(strings.ToLower(response))

        switch response {
        case "a", "approve":
            approved = append(approved, rec)
            fmt.Printf("✅ Approved (%d/%d)\n\n", len(approved), len(recommendations))

        case "r", "reject":
            rejected = append(rejected, rec)
            fmt.Printf("❌ Rejected\n\n")

        case "e", "edit":
            // Open editor for manual modification
            edited, err := editRecommendation(rec)
            if err != nil {
                return err
            }
            approved = append(approved, edited)
            fmt.Printf("✏️  Edited and approved (%d/%d)\n\n", len(approved), len(recommendations))

        case "s", "skip":
            fmt.Printf("⏭️  Skipped\n\n")
            continue

        case "q", "quit":
            return fmt.Errorf("user canceled")

        default:
            fmt.Println("Invalid response. Skipping...")
        }
    }

    // Summary
    fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Println("Summary")
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("Approved: %d\n", len(approved))
    fmt.Printf("Rejected: %d\n", len(rejected))
    fmt.Printf("Skipped: %d\n", len(recommendations)-len(approved)-len(rejected))

    if len(approved) == 0 {
        fmt.Println("\nNo changes to apply.")
        return nil
    }

    // Confirm application
    fmt.Print("\nApply changes? [y/n] ")
    response, _ := reader.ReadString('\n')
    if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(response)), "y") {
        fmt.Println("Canceled.")
        return nil
    }

    // Apply approved changes
    if err := applyChanges(approved); err != nil {
        return fmt.Errorf("apply changes: %w", err)
    }

    fmt.Println("\n✓ All changes applied successfully")
    return nil
}

func displayRecommendation(current, total int, rec Recommendation) {
    fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("Recommendation %d/%d (priority: %d)\n", current, total, rec.Priority)
    fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
    fmt.Printf("\nAgent: %s\n", rec.Agent)
    fmt.Printf("Category: %s\n", rec.SharpEdge.Category)
    fmt.Printf("ID: %s\n", rec.SharpEdge.ID)
    fmt.Printf("Severity: %s\n\n", rec.SharpEdge.Severity)

    fmt.Printf("Description:\n  %s\n\n", rec.SharpEdge.Description)
    fmt.Printf("Symptom:\n  %s\n\n", rec.SharpEdge.Symptom)
    fmt.Printf("Solution:\n%s\n\n", indentLines(rec.SharpEdge.Solution, "  "))

    fmt.Printf("Evidence:\n")
    fmt.Printf("  - Occurred %d times across %d sessions\n",
        rec.Evidence.OccurrenceCount,
        rec.Evidence.UniqueSessions)

    if len(rec.Conflicts) > 0 {
        fmt.Printf("\n⚠️  Conflicts:\n")
        for _, conflict := range rec.Conflicts {
            fmt.Printf("  - %s\n", conflict.Message)
        }
    } else {
        fmt.Printf("\nConflicts: None\n")
    }

    fmt.Println()
}

func indentLines(text, prefix string) string {
    lines := strings.Split(text, "\n")
    for i, line := range lines {
        lines[i] = prefix + line
    }
    return strings.Join(lines, "\n")
}
```

### 3.3 Batch Operations

**Feature:** Approve all non-conflicting recommendations at once

```bash
$ gogent-memory approve-edges --batch --auto-approve-safe

Auto-approving 5 recommendations with no conflicts...
  ✓ nil-pointer-struct-init (go-pro)
  ✓ channel-close-race (go-pro)
  ✓ missing-defer-cancel (go-concurrent)
  ✓ test-isolation-cleanup (go-pro)
  ✓ race-map-access (go-concurrent)

Skipping 2 recommendations with conflicts:
  ⚠️  duplicate-context-check (conflicts with existing edge)
  ⚠️  http-client-timeout (similar to existing edge)

Applied 5 changes. Git commit: def5678
```

### 3.4 Git Integration

**File:** `cmd/gogent-memory/git.go`

```go
package main

import (
    "fmt"
    "os/exec"
    "strings"
)

// CreateGitCommit commits approved changes with metadata
func CreateGitCommit(approved []Recommendation) (string, error) {
    // Stage changes
    for _, rec := range approved {
        agentPath := fmt.Sprintf("~/.claude/agents/%s/sharp-edges.yaml", rec.Agent)
        cmd := exec.Command("git", "add", agentPath)
        if err := cmd.Run(); err != nil {
            return "", fmt.Errorf("git add failed: %w", err)
        }
    }

    // Build commit message
    message := buildCommitMessage(approved)

    // Create commit
    cmd := exec.Command("git", "commit", "-m", message)
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("git commit failed: %w", err)
    }

    // Get commit hash
    cmd = exec.Command("git", "rev-parse", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("get commit hash: %w", err)
    }

    commitHash := strings.TrimSpace(string(output))
    return commitHash[:7], nil  // Short hash
}

func buildCommitMessage(approved []Recommendation) string {
    var lines []string

    // Title
    title := fmt.Sprintf("Add %d sharp edge(s) from memory-improvement", len(approved))
    lines = append(lines, title, "")

    // List edges
    for _, rec := range approved {
        line := fmt.Sprintf("- %s (%s, %s)",
            rec.SharpEdge.ID,
            rec.SharpEdge.Severity,
            rec.SharpEdge.Category)
        lines = append(lines, line)
    }

    // Evidence summary
    totalOccurrences := 0
    totalSessions := 0
    for _, rec := range approved {
        totalOccurrences += rec.Evidence.OccurrenceCount
        totalSessions += rec.Evidence.UniqueSessions
    }

    lines = append(lines, "")
    lines = append(lines, fmt.Sprintf("Evidence: %d occurrences across %d sessions",
        totalOccurrences, totalSessions))

    // Metadata
    lines = append(lines, fmt.Sprintf("Source: /memory-improvement %s", currentDate()))
    lines = append(lines, "")
    lines = append(lines, "Co-Authored-By: Gemini Analysis <noreply@google.com>")

    return strings.Join(lines, "\n")
}

func currentDate() string {
    // Return YYYY-MM-DD
    return "2026-01-30"  // Simplified
}
```

---

## Phase 4: Safety & Rollback

**Time:** 8 hours
**MVP:** Pre-injection backup (2 hours)
**Full:** + Validation suite, auto-rollback (8 hours)

### 4.1 Pre-Injection Backup

**File:** `cmd/gogent-memory/backup.go`

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

// BackupAgentFile creates timestamped backup before modification
func BackupAgentFile(agentName string) (string, error) {
    home := os.Getenv("HOME")
    originalPath := filepath.Join(home, ".claude", "agents", agentName, "sharp-edges.yaml")

    // Create backup directory
    backupDir := filepath.Join(home, ".claude", "tmp", "edge-injection", "backups")
    if err := os.MkdirAll(backupDir, 0755); err != nil {
        return "", fmt.Errorf("create backup dir: %w", err)
    }

    // Backup filename with timestamp
    timestamp := time.Now().Format("20060102-150405")
    backupFilename := fmt.Sprintf("%s-sharp-edges-%s.yaml", agentName, timestamp)
    backupPath := filepath.Join(backupDir, backupFilename)

    // Copy original to backup
    originalData, err := os.ReadFile(originalPath)
    if err != nil {
        return "", fmt.Errorf("read original: %w", err)
    }

    if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
        return "", fmt.Errorf("write backup: %w", err)
    }

    return backupPath, nil
}
```

### 4.2 Post-Injection Validation

**File:** `cmd/gogent-memory/validate_injection.go`

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "gopkg.in/yaml.v3"
)

// ValidationResult contains validation outcomes
type ValidationResult struct {
    Valid          bool
    Errors         []string
    Warnings       []string
    BackupPath     string
}

// ValidateInjection runs comprehensive checks on modified YAML
func ValidateInjection(agentName string, backupPath string) (*ValidationResult, error) {
    result := &ValidationResult{
        Valid:      true,
        BackupPath: backupPath,
    }

    home := os.Getenv("HOME")
    modifiedPath := filepath.Join(home, ".claude", "agents", agentName, "sharp-edges.yaml")

    // Test 1: YAML syntax valid
    data, err := os.ReadFile(modifiedPath)
    if err != nil {
        result.Valid = false
        result.Errors = append(result.Errors, fmt.Sprintf("read file: %v", err))
        return result, nil
    }

    var parsed map[string]interface{}
    if err := yaml.Unmarshal(data, &parsed); err != nil {
        result.Valid = false
        result.Errors = append(result.Errors, fmt.Sprintf("YAML parse error: %v", err))
        return result, nil
    }

    // Test 2: Required fields present
    if _, ok := parsed["sharp_edges"]; !ok {
        result.Valid = false
        result.Errors = append(result.Errors, "missing 'sharp_edges' key")
        return result, nil
    }

    // Test 3: All edges have required fields
    edges, ok := parsed["sharp_edges"].([]interface{})
    if !ok {
        result.Valid = false
        result.Errors = append(result.Errors, "'sharp_edges' is not a list")
        return result, nil
    }

    for i, edge := range edges {
        edgeMap, ok := edge.(map[string]interface{})
        if !ok {
            result.Errors = append(result.Errors, fmt.Sprintf("edge %d: not a map", i))
            result.Valid = false
            continue
        }

        // Check required fields
        requiredFields := []string{"id", "severity", "category", "description", "symptom", "solution"}
        for _, field := range requiredFields {
            if _, ok := edgeMap[field]; !ok {
                result.Errors = append(result.Errors,
                    fmt.Sprintf("edge %d: missing field '%s'", i, field))
                result.Valid = false
            }
        }
    }

    // Test 4: No duplicate IDs
    seenIDs := make(map[string]int)
    for i, edge := range edges {
        edgeMap := edge.(map[string]interface{})
        if id, ok := edgeMap["id"].(string); ok {
            if prevIndex, exists := seenIDs[id]; exists {
                result.Errors = append(result.Errors,
                    fmt.Sprintf("duplicate ID '%s' at indices %d and %d", id, prevIndex, i))
                result.Valid = false
            }
            seenIDs[id] = i
        }
    }

    // Test 5: Agent can load the file (integration test)
    // This would involve actually loading the agent config
    // Simplified here
    result.Warnings = append(result.Warnings, "Agent load test not yet implemented")

    return result, nil
}
```

### 4.3 Automatic Rollback

**File:** `cmd/gogent-memory/rollback.go`

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
)

// RollbackInjection restores from backup if validation fails
func RollbackInjection(agentName, backupPath string) error {
    home := os.Getenv("HOME")
    targetPath := filepath.Join(home, ".claude", "agents", agentName, "sharp-edges.yaml")

    // Read backup
    backupData, err := os.ReadFile(backupPath)
    if err != nil {
        return fmt.Errorf("read backup: %w", err)
    }

    // Restore original
    if err := os.WriteFile(targetPath, backupData, 0644); err != nil {
        return fmt.Errorf("restore from backup: %w", err)
    }

    fmt.Printf("✓ Rolled back %s to backup: %s\n", agentName, backupPath)
    return nil
}

// AutoRollbackOnFailure wraps injection with automatic rollback
func AutoRollbackOnFailure(agentName string, injectionFunc func() error) error {
    // Create backup
    backupPath, err := BackupAgentFile(agentName)
    if err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }

    fmt.Printf("✓ Backup created: %s\n", backupPath)

    // Attempt injection
    if err := injectionFunc(); err != nil {
        fmt.Printf("❌ Injection failed: %v\n", err)
        fmt.Println("Rolling back...")

        if rollbackErr := RollbackInjection(agentName, backupPath); rollbackErr != nil {
            return fmt.Errorf("CRITICAL: injection failed AND rollback failed: %v (original error: %v)",
                rollbackErr, err)
        }

        return fmt.Errorf("injection failed (rolled back): %w", err)
    }

    // Validate post-injection
    fmt.Println("Validating injection...")
    validation, err := ValidateInjection(agentName, backupPath)
    if err != nil {
        return fmt.Errorf("validation error: %w", err)
    }

    if !validation.Valid {
        fmt.Printf("❌ Validation failed:\n")
        for _, e := range validation.Errors {
            fmt.Printf("  - %s\n", e)
        }
        fmt.Println("Rolling back...")

        if rollbackErr := RollbackInjection(agentName, backupPath); rollbackErr != nil {
            return fmt.Errorf("CRITICAL: validation failed AND rollback failed: %v", rollbackErr)
        }

        return fmt.Errorf("validation failed (rolled back)")
    }

    // Success - show warnings if any
    if len(validation.Warnings) > 0 {
        fmt.Printf("⚠️  Warnings:\n")
        for _, w := range validation.Warnings {
            fmt.Printf("  - %s\n", w)
        }
    }

    fmt.Println("✓ Validation passed")
    return nil
}
```

### 4.4 Audit Logging

**File:** `.claude/memory/edge-injection-log.jsonl`

```jsonl
{"timestamp":1738195200,"action":"inject","agent":"go-pro","edge_id":"nil-pointer-struct-init","status":"success","backup_path":".claude/tmp/edge-injection/backups/go-pro-sharp-edges-20260130-143000.yaml","commit":"abc1234"}
{"timestamp":1738195201,"action":"inject","agent":"go-pro","edge_id":"channel-close-race","status":"success","backup_path":".claude/tmp/edge-injection/backups/go-pro-sharp-edges-20260130-143000.yaml","commit":"abc1234"}
{"timestamp":1738195400,"action":"inject","agent":"go-concurrent","edge_id":"missing-defer-cancel","status":"failed","error":"validation failed: duplicate ID","rollback":true}
```

**Purpose:**
- Track all injection attempts
- Audit trail for debugging
- Metrics for success rate
- Compliance (who approved what when)

---

## Integration Points

### 5.1 Integration with /memory-improvement Skill

**Current flow:**
```
User: /memory-improvement
  ↓
Gemini analyzes solved-problems.jsonl
  ↓
Gemini outputs YAML to terminal
  ↓
User manually copies recommendations
```

**New flow:**
```
User: /memory-improvement
  ↓
Gemini analyzes solved-problems.jsonl
  ↓
Gemini outputs YAML to terminal AND writes to file
  ↓
System: "3 recommendations saved. Run 'gogent-memory approve-edges' to apply."
  ↓
User: gogent-memory approve-edges
  ↓
Interactive approval workflow
```

**Modification to memory-improvement SKILL.md:**

```bash
# Add to Phase 3: Output Generation

# Write recommendations to file for automation
TIMESTAMP=$(date +%s)
echo "$GEMINI_OUTPUT" > "$HOME/.claude/tmp/edge-injection/recommendations-$TIMESTAMP.yaml"

# Symlink to latest
ln -sf "$HOME/.claude/tmp/edge-injection/recommendations-$TIMESTAMP.yaml" \
       "$HOME/.claude/tmp/edge-injection/recommendations-latest.yaml"

# Notify user
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📋 $ACTIONABLE_COUNT recommendations saved"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "To review and apply:"
echo "  gogent-memory approve-edges"
echo ""
echo "To preview without applying:"
echo "  gogent-memory preview-edges"
echo ""
```

### 5.2 Integration with gogent-sharp-edge Hook

**No changes required.** Sharp edge injection happens outside the hook execution path.

**Validation:** After injection, sharp edges are loaded normally by the hook via `memory.LoadSharpEdgesIndex()`.

### 5.3 Integration with Git Workflow

**Branching strategy (optional but recommended):**

```bash
# Create branch for automated changes
git checkout -b auto/sharp-edges-$(date +%Y%m%d)

# Apply changes
gogent-memory approve-edges

# Review and merge
git diff master
git checkout master
git merge auto/sharp-edges-20260130
```

---

## Risk Mitigation

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| YAML corruption | MEDIUM | CRITICAL | Pre-backup, post-validation, auto-rollback |
| Duplicate ID injection | LOW | HIGH | Conflict detection in Phase 1 |
| Comment loss | MEDIUM | LOW | Use yaml.v3 with comment preservation OR template-based generation |
| Wrong category placement | LOW | LOW | Category ordering logic with fallback to append |
| Git conflict with manual edits | MEDIUM | MEDIUM | Warning on dirty working tree, branch-based workflow |
| Agent fails to load after injection | LOW | CRITICAL | Agent load test in validation, auto-rollback |
| Gemini generates invalid YAML | LOW | MEDIUM | Schema validation rejects invalid input |

### Mitigation Strategies

**1. YAML Corruption Protection**
- ✅ Backup before every injection
- ✅ Validate YAML syntax after injection
- ✅ Test agent can load modified file
- ✅ Auto-rollback on any validation failure
- ✅ Audit log of all operations

**2. Conflict Prevention**
- ✅ Detect duplicate IDs before injection
- ✅ Fuzzy match similar descriptions
- ✅ Warn on same symptom
- ✅ User can reject conflicting recommendations

**3. Comment Preservation**
- ✅ Use yaml.v3 with comment support
- ✅ Fallback: Template-based generation with known comment structure
- ✅ Document: Comments between categories may be lost (acceptable tradeoff)

**4. Git Safety**
- ✅ Check for dirty working tree before injection
- ✅ Recommend branch-based workflow
- ✅ Git commit includes full metadata for reverting

**5. Testing Before Production**
- ✅ Dry-run mode: `gogent-memory approve-edges --dry-run`
- ✅ Preview mode: `gogent-memory preview-edges` (shows diffs without applying)
- ✅ Test on copy of sharp-edges.yaml first

---

## Success Metrics

### Quantitative Metrics

| Metric | Baseline (Manual) | Target (Automated) | Measurement |
|--------|-------------------|-------------------|-------------|
| Time to add sharp edge | 15-30 min | 2-5 min | Track from recommendation to commit |
| Sharp edges added/month | 2-5 | 10-20 | Count commits with "memory-improvement" tag |
| Injection success rate | N/A | >95% | Successful injections / total attempts |
| Rollback rate | N/A | <5% | Rollbacks / total attempts |
| YAML corruption incidents | 0 | 0 | Zero tolerance |

### Qualitative Metrics

| Metric | Measurement |
|--------|-------------|
| User satisfaction | Survey: "Easier to maintain sharp edges?" |
| Sharp edge quality | Review: Are auto-generated edges helpful? |
| System trustworthiness | Survey: "Comfortable with automated injection?" |

### Success Criteria (6 months post-deployment)

- [ ] >50 sharp edges injected via automation
- [ ] <3 rollbacks due to validation failures
- [ ] Zero YAML corruption incidents
- [ ] Average injection time <5 minutes
- [ ] User satisfaction score >4/5

---

## Rollout Strategy

### Phase 0: Pre-Requisites (Before Starting)

- [ ] GOgent-MEM-001 deployed for ≥3 months
- [ ] ≥100 entries in solved-problems.jsonl
- [ ] ≥10 Gemini recommendations generated manually
- [ ] Sharp edge format stable (no schema changes planned)

### MVP Rollout (12 hours, Week 1-2)

**Goal:** Core functionality without advanced features

**Deliverables:**
- [ ] Parser & validator (Phase 1 MVP)
- [ ] Basic injection engine (Phase 2 MVP)
- [ ] Simple approval CLI (Phase 3 MVP)
- [ ] Pre-backup only (Phase 4 MVP)

**Testing:**
- [ ] Test on 5 recommendations from real /memory-improvement run
- [ ] Verify YAML syntax valid
- [ ] Manual git commit

**Go/No-Go Decision:** If MVP works reliably on 10+ injections → proceed to Full

### Full Implementation (18 hours, Week 3-4)

**Deliverables:**
- [ ] Conflict detection (Phase 1 Full)
- [ ] Comment preservation (Phase 2 Full)
- [ ] Git integration (Phase 3 Full)
- [ ] Validation suite + auto-rollback (Phase 4 Full)

**Testing:**
- [ ] Integration test: End-to-end from /memory-improvement to git commit
- [ ] Edge case test: Duplicate ID, conflicting description
- [ ] Failure test: Intentionally corrupt YAML, verify rollback
- [ ] Load test: Inject 20 edges in batch mode

### Production Deployment (Week 5)

**Monitoring:**
- Watch `.claude/memory/edge-injection-log.jsonl` for failures
- Track rollback rate
- Monitor git commits for quality

**Rollback Plan:**
If injection success rate <80% in first month:
- Disable automation
- Debug root cause
- Fix and re-deploy

---

## Future Enhancements (v3.0+)

### Enhancement 1: ML-Driven Conflict Resolution

**Problem:** Conflict detection is rule-based (exact ID match, fuzzy description)

**Enhancement:** Use embeddings to detect semantic similarity
- Compare new edge description with existing edges
- Detect "this is the same problem, different wording"
- Suggest merging instead of duplicate injection

**Time:** 8-12 hours

### Enhancement 2: Cross-Agent Pattern Detection

**Problem:** Same pattern might apply to multiple agents (e.g., nil-pointer for go-pro AND go-concurrent)

**Enhancement:** Gemini recommends cross-agent injection
- "This pattern affects 3 agents: go-pro, go-concurrent, go-api"
- Batch inject into all relevant agents

**Time:** 4-6 hours

### Enhancement 3: Sharp Edge Effectiveness Tracking

**Problem:** No feedback on whether injected sharp edges actually help

**Enhancement:** Track if problems recur after edge injection
- Link solved-problems.jsonl to injected edge IDs
- If same problem_type + root_cause recurs → edge was ineffective
- Flag for review/refinement

**Time:** 6-8 hours

### Enhancement 4: Visual Diff UI

**Problem:** CLI diff is functional but not beautiful

**Enhancement:** Web-based approval UI
- Rich diff viewer with syntax highlighting
- Side-by-side comparison
- Click to approve/reject
- Accessible via `gogent-memory serve`

**Time:** 12-16 hours

---

## Appendix: Testing Strategy

### Unit Tests

**File:** `cmd/gogent-memory/parser_test.go`

```go
func TestParseRecommendations(t *testing.T)
func TestValidateRecommendations_ValidInput(t *testing.T)
func TestValidateRecommendations_MissingFields(t *testing.T)
func TestValidateRecommendations_InvalidSeverity(t *testing.T)
func TestDetectConflicts_DuplicateID(t *testing.T)
func TestDetectConflicts_SimilarDescription(t *testing.T)
```

**File:** `cmd/gogent-memory/inject_test.go`

```go
func TestInjectSharpEdge_SimpleCase(t *testing.T)
func TestInjectSharpEdge_PreserveComments(t *testing.T)
func TestInjectSharpEdge_CorrectPosition(t *testing.T)
func TestInjectSharpEdge_MultipleEdges(t *testing.T)
```

**File:** `cmd/gogent-memory/validate_injection_test.go`

```go
func TestValidateInjection_ValidYAML(t *testing.T)
func TestValidateInjection_InvalidYAML(t *testing.T)
func TestValidateInjection_DuplicateID(t *testing.T)
func TestRollbackInjection(t *testing.T)
```

### Integration Tests

**Test 1: End-to-End Happy Path**
```bash
#!/bin/bash
# Test full workflow from Gemini YAML to git commit

# 1. Create test recommendations.yaml
cat > test-recommendations.yaml <<EOF
sharp_edge_recommendations:
  - priority: 1
    agent: go-pro
    action: add_sharp_edge
    sharp_edge:
      id: test-edge-001
      severity: high
      category: runtime
      description: "Test edge"
      symptom: "Test symptom"
      solution: "Test solution"
    evidence:
      occurrence_count: 5
      unique_sessions: 3
EOF

# 2. Run approval workflow in test mode
gogent-memory approve-edges --input test-recommendations.yaml --auto-approve --test

# 3. Verify injection
if grep -q "test-edge-001" ~/.claude/agents/go-pro/sharp-edges.yaml; then
    echo "✓ Edge injected"
else
    echo "✗ Edge not found"
    exit 1
fi

# 4. Verify git commit
if git log -1 --pretty=%B | grep -q "test-edge-001"; then
    echo "✓ Git commit created"
else
    echo "✗ Git commit not found"
    exit 1
fi

# 5. Cleanup (rollback)
git reset --hard HEAD~1
```

**Test 2: Validation Failure → Auto-Rollback**
```bash
#!/bin/bash
# Test that rollback works on validation failure

# 1. Backup original
cp ~/.claude/agents/go-pro/sharp-edges.yaml /tmp/backup.yaml

# 2. Create invalid recommendation (duplicate ID)
cat > test-invalid.yaml <<EOF
sharp_edge_recommendations:
  - agent: go-pro
    action: add_sharp_edge
    sharp_edge:
      id: goroutine-leak-context  # Already exists
      severity: high
      category: runtime
      description: "Duplicate"
      symptom: "Test"
      solution: "Test"
EOF

# 3. Attempt injection (should fail)
if gogent-memory approve-edges --input test-invalid.yaml --auto-approve 2>&1 | grep -q "rolled back"; then
    echo "✓ Rollback triggered"
else
    echo "✗ Rollback not triggered"
    exit 1
fi

# 4. Verify file unchanged
if diff ~/.claude/agents/go-pro/sharp-edges.yaml /tmp/backup.yaml; then
    echo "✓ File unchanged after rollback"
else
    echo "✗ File was modified despite rollback"
    exit 1
fi
```

---

## Appendix: CLI Reference

### Commands

```bash
# Primary workflow
gogent-memory approve-edges                    # Interactive approval
gogent-memory approve-edges --auto-approve-safe # Batch approve non-conflicting
gogent-memory approve-edges --dry-run          # Preview without applying

# Utilities
gogent-memory preview-edges                    # Show diffs without approving
gogent-memory list-recommendations             # List pending recommendations
gogent-memory rollback-last                    # Undo last injection

# Advanced
gogent-memory approve-edges --input custom.yaml   # Use custom YAML input
gogent-memory validate-recommendation file.yaml   # Validate without injecting
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed |
| 2 | Injection failed (rolled back) |
| 3 | User canceled |
| 4 | No recommendations found |

---

## Conclusion

**Automated Sharp Edge Injection transforms GOgent-Fortress from:**
- "Generates recommendations" → "Self-improves"
- Manual curation (slow, tedious) → Automated feedback loop (fast, scalable)
- Linear knowledge growth → Exponential knowledge compounding

**When to start:** After GOgent-MEM-001 proves value for 3+ months

**MVP first:** 12 hours gets you 80% of value (parse, inject, approve)

**Full implementation:** 30 hours for production-ready system with safety guarantees

**ROI:** First 10 auto-injected sharp edges pays for implementation time

**Next steps:**
1. Deploy GOgent-MEM-001
2. Monitor solved-problems.jsonl growth
3. Run /memory-improvement monthly
4. After 3 months: Start MEM-002 MVP
5. After MVP proves stable: Complete full implementation

---

**End of Vision Document**

*This document serves as the blueprint for GOgent-MEM-002. Update as implementation reveals new insights.*
