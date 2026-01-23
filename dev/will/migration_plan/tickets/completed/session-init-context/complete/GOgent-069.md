---
id: GOgent-069
title: Update Harness CLI for SessionStart
description: Update harness CLI to find and use gogent-load-context binary
status: pending
time_estimate: 1h
dependencies:
  - GOgent-067
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 9
---

## GOgent-069: Update Harness CLI for SessionStart

**Time**: 1 hour
**Dependencies**: GOgent-067
**Priority**: HIGH

**Task**:
Update harness CLI to find and use `gogent-load-context` binary.

**File**: `test/simulation/harness/cmd/harness/main.go` (modify existing)

**Implementation**:
```go
// Add after finding sharpEdgePath (~line 104):

	// Find optional load-context binary for sessionstart scenarios
	loadContextPath, loadContextErr := findBinary("gogent-load-context")
	if loadContextErr != nil && *verbose {
		fmt.Printf("[INFO] gogent-load-context not found, sessionstart scenarios will be skipped\n")
	}

// After setting sharp-edge path (~line 116):

	// Set load-context path if available
	if loadContextErr == nil {
		runner.SetLoadContextPath(loadContextPath)
	}
```

**Update Makefile** - add build target:
```makefile
# Add to existing build targets
build-load-context:
	@echo "Building gogent-load-context..."
	@go build -o bin/gogent-load-context ./cmd/gogent-load-context
	@echo "✓ Built: bin/gogent-load-context"

# Update build-all to include new binary
build-all: build-validate build-archive build-sharp-edge build-load-context
	@echo "✓ All hook binaries built"

# Add simulation target for sessionstart
test-simulation-sessionstart:
	@echo "Running SessionStart simulation tests..."
	@go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-filter=sim-startup,sim-resume \
		-verbose
	@echo "✓ SessionStart simulation tests passed"
```

**Tests**: Add to `test/simulation/harness/cmd/harness/main_test.go`

```go
func TestFindBinary_LoadContext(t *testing.T) {
	// This test verifies findBinary can locate gogent-load-context
	// when it exists in expected locations

	// Create temp bin directory
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "bin", "gogent-load-context")
	os.MkdirAll(filepath.Dir(binPath), 0755)

	// Create mock binary
	os.WriteFile(binPath, []byte("#!/bin/bash\necho 'mock'"), 0755)

	// Save and restore working directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	path, err := findBinary("gogent-load-context")
	if err != nil {
		t.Fatalf("findBinary failed: %v", err)
	}

	if !strings.Contains(path, "gogent-load-context") {
		t.Errorf("Expected path to contain binary name, got: %s", path)
	}
}
```

**Acceptance Criteria**:
- [x] Harness CLI finds `gogent-load-context` in bin/ or PATH
- [x] Verbose mode logs when binary not found
- [x] Runner receives load-context path when available
- [x] `make build-all` builds all 4 hook binaries
- [x] `make test-simulation-sessionstart` runs SessionStart tests
- [x] Tests verify binary discovery

**Test Deliverables**:
- [x] Tests added to: `cmd/harness/main_test.go`
- [x] Makefile targets added: `build-load-context`, `test-simulation-sessionstart`
- [x] Tests passing: ✅

**Why This Matters**: CLI integration enables harness to execute SessionStart tests in CI/CD.

---
