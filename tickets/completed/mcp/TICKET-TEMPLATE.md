# Ticket Template - Required Structure

**Version**: 1.0
**Last Updated**: 2026-01-15
**Purpose**: Enforce consistent detail level across all Phase 0 tickets

---

## Mandatory Ticket Structure

Every ticket MUST follow this exact structure. NO shortcuts, NO "implement logic here", NO "omitted for brevity".

```markdown
#### GOgent-XXX: [Descriptive Title]
**Time**: X hours
**Dependencies**: GOgent-YYY, GOgent-ZZZ (or "None")
**Priority**: HIGH/MEDIUM/LOW (optional - only if critical path)

**Task**:
[One clear sentence describing what needs to be done]

**File**: `exact/path/to/file.go` (or multiple files if ticket spans multiple)

**Imports**:
```go
package packagename

import (
    "standard/library/package1"
    "standard/library/package2"

    "github.com/yourusername/gogent-fortress/pkg/internal"
)
```

**Implementation**:
```go
// COMPLETE, PRODUCTION-READY CODE
// Contractor should be able to copy-paste this directly
// Include:
// - Full function signatures
// - Complete error handling
// - Proper error messages (format: "[component] What. Why. How to fix.")
// - All edge cases handled
// - Comments explaining non-obvious logic

func ExampleFunction(param1 string, param2 int) (*Result, error) {
    // Validate inputs
    if param1 == "" {
        return nil, fmt.Errorf("[component] Parameter param1 empty. Required for X. Provide non-empty value.")
    }

    // Main logic
    result := &Result{
        Field1: param1,
        Field2: param2,
    }

    return result, nil
}
```

**Tests** (if applicable):
```go
package packagename

import (
    "testing"
)

// COMPLETE test implementations
// Coverage target: ≥80%
// Test naming: TestFunctionName_Scenario

func TestExampleFunction_ValidInput(t *testing.T) {
    result, err := ExampleFunction("valid", 123)

    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if result.Field1 != "valid" {
        t.Errorf("Expected Field1 'valid', got: %s", result.Field1)
    }
}

func TestExampleFunction_EmptyParam(t *testing.T) {
    _, err := ExampleFunction("", 123)

    if err == nil {
        t.Error("Expected error for empty param1, got nil")
    }

    if !strings.Contains(err.Error(), "[component]") {
        t.Errorf("Expected error with component tag, got: %v", err)
    }
}
```

**Acceptance Criteria**:
- [ ] Specific, testable requirement 1 (e.g., "Function returns error for nil input")
- [ ] Specific, testable requirement 2 (e.g., "All tests pass: `go test ./pkg/package`")
- [ ] Specific, testable requirement 3 (e.g., "Error messages follow format")
- [ ] Code coverage ≥80% (if applicable)

**Test Deliverables** (MANDATORY):
- [ ] Test file created: `path/to/file_test.go`
- [ ] Test file size: XXX lines
- [ ] Number of test functions: X
- [ ] Coverage achieved: XX%
- [ ] Tests passing: ✅ (output: `go test ./path/to/package`)
- [ ] Race detector clean: ✅ (output: `go test -race ./path/to/package`)
- [ ] **ECOSYSTEM TEST PASS REQUIRED**: Run `make test-ecosystem` and verify ALL PASS
- [ ] Ecosystem test output saved to: `test/audit/GOgent-XXX/`
- [ ] Test audit updated: `/test/INDEX.md` row added

**CRITICAL**: The `make test-ecosystem` command MUST pass before ticket can be marked complete. This is NON-NEGOTIABLE. If ecosystem tests fail, the ticket is NOT done. Save the audit output to demonstrate compliance.

**Why This Matters**:
[Context explaining why this ticket exists, what problem it solves, which critical review issue it fixes (if applicable)]
```

---

## Required Conventions (Must Apply to ALL Tickets)

### Error Message Format
```
[component] What happened. Why it was blocked/failed. How to fix.
```

**Examples**:
- ✅ GOOD: `[config] Failed to read routing schema at ~/.claude/routing-schema.json: file not found. Ensure .claude/ directory exists.`
- ❌ BAD: `config error`
- ❌ BAD: `failed to load`

### File Path Conventions
- **ALWAYS** use XDG Base Directory compliance
- Priority: `XDG_RUNTIME_DIR > XDG_CACHE_HOME > ~/.cache/gogent`
- **NEVER** hardcode `/tmp` paths (violates M-2 from critical review)

**Example**:
```go
func GetGOgentDir() string {
    if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        os.MkdirAll(dir, 0755)
        return dir
    }

    if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
        dir := filepath.Join(xdg, "gogent")
        os.MkdirAll(dir, 0755)
        return dir
    }

    home, _ := os.UserHomeDir()
    dir := filepath.Join(home, ".cache", "gogent")
    os.MkdirAll(dir, 0755)
    return dir
}
```

### STDIN Timeout Handling
- **ALL** hooks reading from STDIN MUST have timeout (default: 5 seconds)
- Prevents hanging hooks (fixes M-6 from critical review)

**Example**:
```go
func ParseToolEvent(r io.Reader, timeout time.Duration) (*ToolEvent, error) {
    ch := make(chan result, 1)
    go func() {
        data, err := io.ReadAll(r)
        ch <- result{data, err}
    }()

    select {
    case res := <-ch:
        // Process data
        return parseData(res.data)
    case <-time.After(timeout):
        return nil, fmt.Errorf("[event-parser] STDIN read timeout after %v. Hook may be stuck waiting for input.", timeout)
    }
}
```

### Test Conventions
- **Coverage target**: ≥80%
- **Naming**: `TestFunctionName_Scenario`
- **Must test**: Valid input, invalid input, edge cases, error conditions
- **Assertions**: Use clear error messages showing expected vs actual

**Example**:
```go
func TestLoadRoutingSchema_ValidFile(t *testing.T) {
    schema, err := LoadRoutingSchema()
    if err != nil {
        t.Fatalf("Expected no error, got: %v", err)
    }

    if schema.Version == "" {
        t.Error("Expected version field to be populated")
    }
}
```

### Go Version and Module
- **Go Version**: 1.21+
- **Module Path**: `github.com/yourusername/gogent-fortress`
- Contractor replaces "yourusername" with their GitHub username during GOgent-001

---

## Anti-Patterns (FORBIDDEN)

### ❌ Pseudocode or Placeholders
```go
// BAD - contractor doesn't know what to implement
func ValidateRouting(event ToolEvent) error {
    // TODO: Implement validation logic here
    return nil
}
```

### ❌ "Omitted for Brevity"
```go
// BAD - contractor has to guess struct fields
type Schema struct {
    Version string
    // ... other fields omitted for brevity
}
```

### ❌ Incomplete Error Handling
```go
// BAD - no context, no guidance
if err != nil {
    return err
}

// GOOD - structured error with guidance
if err != nil {
    return fmt.Errorf("[config] Failed to load schema: %w. Check file exists at ~/.claude/routing-schema.json", err)
}
```

### ❌ Missing Test Cases
```go
// BAD - only tests happy path
func TestParseEvent(t *testing.T) {
    event, _ := ParseEvent(`{"valid": "json"}`)
    // only checks success case
}

// GOOD - tests multiple scenarios
func TestParseEvent_ValidJSON(t *testing.T) { ... }
func TestParseEvent_InvalidJSON(t *testing.T) { ... }
func TestParseEvent_MissingFields(t *testing.T) { ... }
```

---

## Checklist Before Marking Ticket Complete

- [ ] All code is copy-paste ready (no placeholders)
- [ ] All imports are complete
- [ ] Error messages follow `[component] What. Why. How.` format
- [ ] File paths use XDG compliance (no hardcoded /tmp)
- [ ] STDIN reads have timeout (if applicable)
- [ ] Tests are complete with ≥80% coverage
- [ ] **ECOSYSTEM TEST PASSED**: `make test-ecosystem` shows ALL PASS
- [ ] Audit trail saved to `test/audit/GOgent-XXX/`
- [ ] Acceptance criteria are specific and testable
- [ ] "Why This Matters" section explains context
- [ ] Dependencies are listed correctly
- [ ] Time estimate is realistic (1-2 hours per ticket)

---

## Cross-References

- **Testing Strategy**: See [00-overview.md](00-overview.md#testing-strategy)
- **Rollback Plan**: See [00-overview.md](00-overview.md#rollback-plan)
- **Error Standards**: See [00-overview.md](00-overview.md#error-handling-standards)
- **Critical Review**: See [../CRITICAL_REVIEW.md](../CRITICAL_REVIEW.md)
- **Migration Plan**: See [../gogent_migration_plan_v3_FINAL.md](../gogent_migration_plan_v3_FINAL.md)

---

**Remember**: The contractor should be able to implement ANY ticket without asking clarifying questions. If you find yourself writing "implement X", you haven't provided enough detail.
