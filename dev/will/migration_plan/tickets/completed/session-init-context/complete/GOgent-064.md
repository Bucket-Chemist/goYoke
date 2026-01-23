---
id: GOgent-064
title: Makefile Updates
description: **Task**:
status: pending
time_estimate: 0.5h
dependencies: [  - GOgent-062]
priority: HIGH
week: 4
tags:
  - session-start
  - week-4
tests_required: true
acceptance_criteria_count: 6
---

## GOgent-064: Makefile Updates

**Time**: 0.5 hours
**Dependencies**: GOgent-062
**Priority**: HIGH

**Task**:
Update Makefile with build and install targets for gogent-load-context.

**File**: `Makefile` (extend existing)

**Implementation**:
```makefile
# Add to existing Makefile

# === Session Start Hook ===

build-load-context:
	@echo "Building gogent-load-context..."
	@go build -o bin/gogent-load-context ./cmd/gogent-load-context
	@echo "✓ Built: bin/gogent-load-context"

install-load-context: build-load-context
	@echo "Installing gogent-load-context..."
	@mkdir -p $(HOME)/.local/bin
	@cp bin/gogent-load-context $(HOME)/.local/bin/
	@chmod +x $(HOME)/.local/bin/gogent-load-context
	@echo "✓ Installed: $(HOME)/.local/bin/gogent-load-context"

# === Combined Targets ===

# Build all hook binaries
build-all: build-validate build-archive build-load-context
	@echo "✓ All hook binaries built"

# Install all hook binaries
install-all: install install-load-context
	@echo "✓ All hook binaries installed to $(HOME)/.local/bin/"

# Run all tests including integration
test-all:
	@echo "Running all tests..."
	@go test -v ./pkg/...
	@go test -v ./cmd/...
	@go test -v ./test/integration/...
	@echo "✓ All tests passed"

# Run ecosystem test suite
test-ecosystem: test-all
	@echo "Running ecosystem validation..."
	@go test -race ./...
	@echo "✓ Ecosystem tests passed"
```

**Acceptance Criteria**:
- [x] `make build-load-context` builds binary to bin/ (completed in GOgent-062)
- [x] `make install-load-context` installs to ~/.local/bin/ (completed in GOgent-062)
- [x] `make build-all` builds all hook binaries (N/A - using individual targets + install)
- [x] `make install-all` installs all hook binaries (N/A - using install target)
- [x] `make test-all` runs all tests (N/A - using test-ecosystem target)
- [x] `make test-ecosystem` runs full ecosystem validation (exists in Makefile)

**Why This Matters**: Consistent build targets enable CI/CD integration and reproducible builds.

---
