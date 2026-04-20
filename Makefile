# Makefile for goYoke Go migration
# Project: goYoke
# Purpose: Convenient task automation

BINARY_NAME=goyoke
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: help test test-ecosystem test-unit test-integration test-race coverage build build-tui build-legacy build-hooks build-archive build-validate build-aggregate build-sharp-edge build-capture-intent build-load-context build-codebase-extract install install-archive install-aggregate install-wrapper install-load-context install-codebase-extract uninstall uninstall-aggregate check-path clean defaults dist check-size clean-defaults test-defaults test-zero-install dev-setup test-simulation test-simulation-fuzz test-simulation-deterministic test-simulation-posttooluse test-simulation-replay test-simulation-behavioral test-simulation-chaos test-simulation-behavioral-all replay-crash clean-simulation test-sharp-edge-unit test-sharp-edge-integration test-sharp-edge-coverage test-sharp-edge-all telemetry-tools check-claude-writes test-codebase-extract-coverage all

help:
	@echo "goYoke - Available targets:"
	@echo "  make test            - Run full test ecosystem (alias for test-ecosystem)"
	@echo "  make test-ecosystem  - Run complete test suite with audit trail"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-race       - Run race detector"
	@echo "  make coverage        - Generate coverage report"
	@echo "  make build           - Build TypeScript TUI and all hooks (default)"
	@echo "  make build-tui       - Build TypeScript TUI only"
	@echo "  make build-legacy    - Build legacy Go TUI"
	@echo "  make build-hooks     - Build all hook binaries"
	@echo "  make build-validate  - Build goyoke-validate binary"
	@echo "  make build-archive   - Build goyoke-archive binary"
	@echo "  make build-aggregate - Build goyoke-aggregate binary"
	@echo "  make build-sharp-edge - Build goyoke-sharp-edge binary"
	@echo "  make build-capture-intent - Build goyoke-capture-intent binary"
	@echo "  make build-load-context   - Build goyoke-load-context binary"
	@echo "  make build-agent-endstate - Build goyoke-agent-endstate binary"
	@echo "  make build-orchestrator-guard - Build goyoke-orchestrator-guard binary"
	@echo "  make build-update-review-outcome - Build goyoke-update-review-outcome binary"
	@echo "  make build-log-review     - Build goyoke-log-review binary"
	@echo "  make build-all             - Build all hook binaries"
	@echo "  make install         - Install all CLIs to ~/.local/bin"
	@echo "  make install-archive - Install goyoke-archive to ~/.local/bin"
	@echo "  make install-aggregate - Install goyoke-aggregate to ~/.local/bin"
	@echo "  make install-load-context - Install goyoke-load-context to ~/.local/bin"
	@echo "  make install-orchestrator-guard - Install goyoke-orchestrator-guard to ~/.local/bin"
	@echo "  make install-update-review-outcome - Install goyoke-update-review-outcome to ~/.local/bin"
	@echo "  make install-log-review - Install goyoke-log-review to ~/.local/bin"
	@echo "  make install-wrapper - Install session-archive wrapper hook"
	@echo "  make uninstall       - Remove all CLIs from ~/.local/bin"
	@echo "  make check-path      - Verify ~/.local/bin is in PATH"
	@echo "  make clean           - Remove build artifacts"
	@echo ""
	@echo "Simulation testing:"
	@echo "  make test-simulation              - Run mixed simulation (deterministic + fuzz)"
	@echo "  make test-simulation-deterministic - Run deterministic tests only"
	@echo "  make test-simulation-fuzz         - Run fuzz tests only"
	@echo "  make test-simulation-posttooluse  - Run posttooluse tests only (requires build-sharp-edge)"
	@echo "  make test-simulation-sessionstart - Run sessionstart tests only (requires build-load-context)"
	@echo "  make test-simulation-replay       - Run session replay tests (goYoke-042)"
	@echo "  make test-simulation-behavioral   - Run behavioral property tests (goYoke-042)"
	@echo "  make test-simulation-chaos        - Run chaos tests (goYoke-042)"
	@echo "  make test-simulation-behavioral-all - Run all behavioral tests"
	@echo "  make replay-crash CRASH=<file>    - Replay a specific crash"
	@echo "  make clean-simulation             - Clean simulation artifacts"
	@echo ""
	@echo "Sharp Edge testing:"
	@echo "  make test-sharp-edge-unit         - Run sharp edge unit tests"
	@echo "  make test-sharp-edge-integration  - Run sharp edge integration tests"
	@echo "  make test-sharp-edge-coverage     - Generate coverage report for sharp edge"
	@echo "  make test-sharp-edge-all          - Run all sharp edge tests"

# Primary test target - runs full ecosystem with audit trail
test: test-ecosystem

test-ecosystem:
	@./scripts/test-ecosystem.sh

# Individual test targets for granular testing
test-unit:
	@echo "Running unit tests..."
	@go test -v ./cmd/... ./pkg/... ./test/...

test-integration:
	@echo "Running integration tests..."
	@go test -v -run 'TestEcosystem_' ./pkg/routing

test-race:
	@echo "Running race detector..."
	@go test -race ./cmd/... ./pkg/... ./test/...

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./cmd/... ./pkg/... ./test/...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report:"
	@echo "  go tool cover -html=coverage.out"

# Build targets
# New default: TypeScript TUI + hooks
build: build-hooks build-tui

# Build TypeScript TUI
build-tui:
	@echo "Building TypeScript TUI..."
	@cd packages/tui && npm install && npm run build
	@chmod +x packages/tui/bin/goyoke-tui.js
	@echo "✓ TypeScript TUI built at packages/tui/dist/index.js"

# Build Go TUI + MCP server
build-go-tui:
	@echo "Building Go TUI..."
	@mkdir -p bin
	@go build -ldflags "-X main.version=$$(git describe --tags --always 2>/dev/null || echo dev)" \
		-o bin/goyoke ./cmd/goyoke
	@echo "✓ Go TUI built at bin/goyoke"

build-go-mcp:
	@echo "Building Go MCP server..."
	@mkdir -p bin
	@go build -o bin/goyoke-mcp ./cmd/goyoke-mcp
	@echo "✓ Go MCP server built at bin/goyoke-mcp"

build-go: build-go-tui build-go-mcp
	@echo "✓ All Go TUI binaries built"

# Remove stale binaries from project root (C-2 fix: all outputs go to bin/)
clean-stale:
	@rm -f goyoke goyoke-mcp goyoke-mcp-standalone
	@echo "✓ Stale root binaries removed"

# Build legacy Go TUI
build-legacy:
	@echo "Building legacy Go TUI..."
	@go build -o bin/goyoke-legacy ./deprecated/cmd/goyoke
	@echo "✓ Legacy Go TUI built at bin/goyoke-legacy"

# Build all hook binaries
build-hooks: build-validate build-archive build-sharp-edge build-load-context build-agent-endstate build-orchestrator-guard build-update-review-outcome build-log-review
	@echo "✓ All hook binaries built"

build-validate:
	@echo "Building goyoke-validate binary..."
	go build -o bin/goyoke-validate ./cmd/goyoke-validate
	@echo "✅ Binary created at bin/goyoke-validate"

build-archive:
	@echo "Building goyoke-archive binary..."
	go build -o bin/goyoke-archive ./cmd/goyoke-archive
	@echo "✅ Binary created at bin/goyoke-archive"

build-aggregate:
	@echo "Building goyoke-aggregate binary..."
	go build -o bin/goyoke-aggregate ./cmd/goyoke-aggregate
	@echo "✅ Binary created at bin/goyoke-aggregate"

build-sharp-edge:
	@echo "Building goyoke-sharp-edge binary..."
	go build -o bin/goyoke-sharp-edge ./cmd/goyoke-sharp-edge
	@echo "✅ Binary created at bin/goyoke-sharp-edge"

build-capture-intent:
	@echo "Building goyoke-capture-intent binary..."
	go build -o bin/goyoke-capture-intent ./cmd/goyoke-capture-intent
	@echo "✅ Binary created at bin/goyoke-capture-intent"

build-load-context:
	@echo "Building goyoke-load-context..."
	@go build -o bin/goyoke-load-context ./cmd/goyoke-load-context
	@echo "✓ Built: bin/goyoke-load-context"

build-agent-endstate:
	@echo "Building goyoke-agent-endstate..."
	@go build -o bin/goyoke-agent-endstate ./cmd/goyoke-agent-endstate
	@echo "✓ Built: bin/goyoke-agent-endstate"

build-orchestrator-guard:
	@scripts/build-orchestrator-guard.sh

build-update-review-outcome:
	@echo "Building goyoke-update-review-outcome..."
	@go build -o bin/goyoke-update-review-outcome ./cmd/goyoke-update-review-outcome
	@echo "✓ Built: bin/goyoke-update-review-outcome"

build-log-review:
	@echo "Building goyoke-log-review..."
	@go build -o bin/goyoke-log-review ./cmd/goyoke-log-review
	@echo "✓ Built: bin/goyoke-log-review"

telemetry-tools: build-log-review build-update-review-outcome
	@echo "✓ Telemetry tools built: goyoke-log-review, goyoke-update-review-outcome"

build-codebase-extract:
	@echo "Building goyoke-codebase-extract..."
	@mkdir -p bin
	@go build -ldflags "-X main.version=$$(git describe --tags --always 2>/dev/null || echo dev)" \
		-o bin/goyoke-codebase-extract ./cmd/goyoke-codebase-extract
	@echo "✓ goyoke-codebase-extract built at bin/goyoke-codebase-extract"

install-codebase-extract: build-codebase-extract
	@echo "Installing goyoke-codebase-extract to ~/.local/bin/..."
	@mkdir -p ~/.local/bin
	@cp bin/goyoke-codebase-extract ~/.local/bin/goyoke-codebase-extract
	@echo "✓ Installed to ~/.local/bin/goyoke-codebase-extract"

build-all: build-validate build-archive build-sharp-edge build-load-context build-agent-endstate build-orchestrator-guard build-update-review-outcome build-log-review build-codebase-extract
	@echo "✓ All hook binaries built"

# Alias for build-all (matches plan documentation)
all: build-all

install: build-validate build-archive build-aggregate build-sharp-edge build-capture-intent build-load-context build-agent-endstate build-orchestrator-guard build-update-review-outcome build-log-review build-codebase-extract check-path
	@echo "Installing goYoke CLIs to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/goyoke-validate ~/.local/bin/goyoke-validate
	cp bin/goyoke-archive ~/.local/bin/goyoke-archive
	cp bin/goyoke-aggregate ~/.local/bin/goyoke-aggregate
	cp bin/goyoke-sharp-edge ~/.local/bin/goyoke-sharp-edge
	cp bin/goyoke-capture-intent ~/.local/bin/goyoke-capture-intent
	cp bin/goyoke-load-context ~/.local/bin/goyoke-load-context
	cp bin/goyoke-agent-endstate ~/.local/bin/goyoke-agent-endstate
	cp bin/goyoke-orchestrator-guard ~/.local/bin/goyoke-orchestrator-guard
	cp bin/goyoke-update-review-outcome ~/.local/bin/goyoke-update-review-outcome
	cp bin/goyoke-log-review ~/.local/bin/goyoke-log-review
	cp bin/goyoke-codebase-extract ~/.local/bin/goyoke-codebase-extract
	chmod +x ~/.local/bin/goyoke-validate
	chmod +x ~/.local/bin/goyoke-archive
	chmod +x ~/.local/bin/goyoke-aggregate
	chmod +x ~/.local/bin/goyoke-sharp-edge
	chmod +x ~/.local/bin/goyoke-capture-intent
	chmod +x ~/.local/bin/goyoke-load-context
	chmod +x ~/.local/bin/goyoke-agent-endstate
	chmod +x ~/.local/bin/goyoke-orchestrator-guard
	chmod +x ~/.local/bin/goyoke-update-review-outcome
	chmod +x ~/.local/bin/goyoke-log-review
	chmod +x ~/.local/bin/goyoke-codebase-extract
	@echo "✅ Installed goyoke-validate, goyoke-archive, goyoke-aggregate, goyoke-sharp-edge, goyoke-capture-intent, goyoke-load-context, goyoke-agent-endstate, goyoke-orchestrator-guard, goyoke-update-review-outcome, goyoke-log-review, goyoke-codebase-extract"
	@echo ""
	@$(MAKE) check-path

install-archive: build-archive
	@echo "Installing goyoke-archive to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/goyoke-archive ~/.local/bin/goyoke-archive
	chmod +x ~/.local/bin/goyoke-archive
	@echo "✅ Installed to ~/.local/bin/goyoke-archive"
	@echo "Ensure ~/.local/bin is in your PATH"

install-aggregate: build-aggregate
	@echo "Installing goyoke-aggregate to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/goyoke-aggregate ~/.local/bin/goyoke-aggregate
	chmod +x ~/.local/bin/goyoke-aggregate
	@echo "✅ Installed to ~/.local/bin/goyoke-aggregate"
	@echo "Ensure ~/.local/bin is in your PATH"

install-load-context: build-load-context
	@echo "Installing goyoke-load-context..."
	@mkdir -p $(HOME)/.local/bin
	cp bin/goyoke-load-context $(HOME)/.local/bin/
	chmod +x $(HOME)/.local/bin/goyoke-load-context
	@echo "✓ Installed: $(HOME)/.local/bin/goyoke-load-context"

install-orchestrator-guard: build-orchestrator-guard
	@echo "Installing goyoke-orchestrator-guard to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/goyoke-orchestrator-guard ~/.local/bin/
	@chmod +x ~/.local/bin/goyoke-orchestrator-guard
	@echo "✓ Installed: ~/.local/bin/goyoke-orchestrator-guard"

install-update-review-outcome: build-update-review-outcome
	@echo "Installing goyoke-update-review-outcome to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/goyoke-update-review-outcome ~/.local/bin/
	@chmod +x ~/.local/bin/goyoke-update-review-outcome
	@echo "✓ Installed: ~/.local/bin/goyoke-update-review-outcome"

install-log-review: build-log-review
	@echo "Installing goyoke-log-review to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/goyoke-log-review ~/.local/bin/
	@chmod +x ~/.local/bin/goyoke-log-review
	@echo "✓ Installed: ~/.local/bin/goyoke-log-review"

install-wrapper:
	@echo "Installing session-archive wrapper hook..."
	mkdir -p ~/.claude/hooks
	mkdir -p ~/.goyoke
	cp scripts/session-archive-wrapper.sh ~/.claude/hooks/session-archive-wrapper.sh
	chmod +x ~/.claude/hooks/session-archive-wrapper.sh
	@echo "✅ Wrapper installed"
	@echo ""
	@echo "Update hook config to use:"
	@echo "    command = \"~/.claude/hooks/session-archive-wrapper.sh\""

check-path:
	@if echo $$PATH | grep -q "/.local/bin"; then \
		echo "✅ ~/.local/bin is in PATH"; \
	else \
		echo "⚠️  ~/.local/bin is NOT in PATH"; \
		echo "Add this to your ~/.bashrc or ~/.zshrc:"; \
		echo "    export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
		echo "Then run: source ~/.bashrc"; \
	fi

uninstall:
	@echo "Uninstalling goYoke CLIs from ~/.local/bin/..."
	rm -f ~/.local/bin/goyoke-validate
	rm -f ~/.local/bin/goyoke-archive
	rm -f ~/.local/bin/goyoke-aggregate
	rm -f ~/.local/bin/goyoke-sharp-edge
	rm -f ~/.local/bin/goyoke-capture-intent
	rm -f ~/.local/bin/goyoke-load-context
	rm -f ~/.local/bin/goyoke-agent-endstate
	rm -f ~/.local/bin/goyoke-orchestrator-guard
	rm -f ~/.local/bin/goyoke-update-review-outcome
	rm -f ~/.local/bin/goyoke-log-review
	rm -f ~/.local/bin/goyoke-codebase-extract
	@echo "✅ Uninstalled all CLIs"

uninstall-aggregate:
	@echo "Removing goyoke-aggregate from ~/.local/bin/..."
	rm -f ~/.local/bin/goyoke-aggregate
	@echo "✅ goyoke-aggregate removed"

clean:
	rm -f ${BINARY_NAME}
	rm -f bin/goyoke-validate
	rm -f bin/goyoke-archive
	rm -f bin/goyoke-aggregate
	rm -f bin/goyoke-sharp-edge
	rm -f bin/goyoke-capture-intent
	rm -f bin/goyoke-load-context
	rm -f bin/goyoke-agent-endstate
	rm -f bin/goyoke-orchestrator-guard
	rm -f bin/goyoke-update-review-outcome
	rm -f bin/goyoke-log-review
	rm -f bin/goyoke-legacy
	rm -rf packages/tui/dist
	rm -rf packages/tui/node_modules
	rm -f coverage.out
	rm -f *.test

# ==============================================================================
# Simulation Testing
# ==============================================================================

# Run mixed simulation (deterministic + fuzz)
test-simulation: build-validate build-archive
	@echo "Running simulation tests (mixed mode)..."
	@mkdir -p test/simulation/reports
	go run ./test/simulation/harness/cmd/harness \
		-mode=mixed \
		-iterations=500 \
		-report=markdown \
		-output=test/simulation/reports
	@echo "Report: test/simulation/reports/"

# Run deterministic tests only
test-simulation-deterministic: build-validate build-archive
	@echo "Running deterministic simulation tests..."
	go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-report=tap

# Run fuzz tests only
test-simulation-fuzz: build-validate build-archive
	@echo "Running fuzz simulation tests..."
	go run ./test/simulation/harness/cmd/harness \
		-mode=fuzz \
		-iterations=1000 \
		-verbose

# Run posttooluse tests only (sharp-edge detection)
test-simulation-posttooluse: build-validate build-archive build-sharp-edge
	@echo "Running posttooluse simulation tests..."
	go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-filter=F \
		-report=tap \
		-verbose

# Run sessionstart tests only (context loading)
test-simulation-sessionstart: build-validate build-archive build-load-context
	@echo "Running SessionStart simulation tests..."
	@go run ./test/simulation/harness/cmd/harness \
		-mode=deterministic \
		-filter=startup,resume \
		-verbose
	@echo "✓ SessionStart simulation tests passed"

# Replay a specific crash
# Usage: make replay-crash CRASH=path/to/crash.json
replay-crash: build-validate build-archive
	@if [ -z "$(CRASH)" ]; then \
		echo "Usage: make replay-crash CRASH=path/to/crash.json"; \
		exit 1; \
	fi
	go run ./test/simulation/harness/cmd/harness -replay=$(CRASH)

# Clean simulation artifacts
clean-simulation:
	rm -rf test/simulation/reports/*
	rm -rf test/simulation/tmp/*

# ==============================================================================
# Sharp Edge Testing
# ==============================================================================

test-sharp-edge-unit:
	@echo "Running sharp edge unit tests..."
	go test -v -race ./pkg/routing -run Failure
	go test -v -race ./pkg/memory -run Failure

test-sharp-edge-integration:
	@echo "Running sharp edge integration tests..."
	go test -v -race ./test/integration -run SharpEdge

test-sharp-edge-coverage:
	@echo "Generating sharp edge coverage reports..."
	go test -coverprofile=coverage-routing.out ./pkg/routing
	go test -coverprofile=coverage-memory.out ./pkg/memory
	@echo ""
	@echo "=== pkg/routing Coverage ==="
	go tool cover -func=coverage-routing.out
	@echo ""
	@echo "=== pkg/memory Coverage ==="
	go tool cover -func=coverage-memory.out
	@echo ""
	@echo "To view HTML coverage reports:"
	@echo "  go tool cover -html=coverage-routing.out"
	@echo "  go tool cover -html=coverage-memory.out"

test-sharp-edge-all: test-sharp-edge-unit test-sharp-edge-integration
	@echo "✅ All sharp edge tests passed"

# ==============================================================================
# Behavioral Simulation Testing (goYoke-042)
# 4-level pipeline: Unit -> Session Replay -> Behavioral Properties -> Chaos
# ==============================================================================

# Session Replay Tests (Level 2)
# Tests multi-turn session sequences against recorded fixtures
test-simulation-replay: build-validate build-archive build-sharp-edge
	@echo "Running session replay tests..."
	@mkdir -p test/simulation/reports
	go run ./test/simulation/harness/cmd/harness \
		-mode=replay \
		-report=json \
		-output=test/simulation/reports

# Behavioral Property Tests (Level 3)
# Tests system invariants B1, B4-B7 across sessions
test-simulation-behavioral: build-validate build-archive build-sharp-edge
	@echo "Running behavioral property tests..."
	@mkdir -p test/simulation/reports
	go run ./test/simulation/harness/cmd/harness \
		-mode=behavioral \
		-report=json \
		-output=test/simulation/reports

# Chaos Testing (Level 4)
# Tests concurrent agent scenarios with shared-key contention
test-simulation-chaos: build-validate build-archive build-sharp-edge
	@echo "Running chaos tests..."
	@mkdir -p test/simulation/reports
	@CHAOS_AGENTS=$${CHAOS_AGENTS:-10} \
	CHAOS_SHARED_RATIO=$${CHAOS_SHARED_RATIO:-0.3} \
	go run ./test/simulation/harness/cmd/harness \
		-mode=chaos \
		-report=json \
		-output=test/simulation/reports

# All behavioral tests (for CI/manual comprehensive testing)
test-simulation-behavioral-all: test-simulation-deterministic test-simulation-replay test-simulation-behavioral
	@echo "✅ All behavioral simulation tests passed"

# Full simulation suite including chaos (use sparingly - takes longer)
test-simulation-all: test-simulation-deterministic test-simulation-fuzz test-simulation-replay test-simulation-behavioral
	@echo "✅ All simulation tests passed (excluding chaos)"

test-codebase-extract-coverage:
	@echo "Running codebase-extract coverage..."
	@go test -coverprofile=coverage-codemap.out ./internal/codemap/...
	@go tool cover -func=coverage-codemap.out

# ==============================================================================
# Distribution / Defaults
# ==============================================================================

defaults:
	@scripts/generate-defaults.sh

dist: defaults
	@echo "Building distribution binaries..."
	@mkdir -p bin
	@go build -o bin/ ./cmd/...
	@echo "✓ Distribution build complete"

check-size: dist
	@echo "Checking embedded binary sizes (3MB limit)..."
	@for bin in goyoke goyoke-mcp goyoke-team-run goyoke-load-context goyoke-validate; do \
		size=$$(stat -c%s bin/$$bin 2>/dev/null || stat -f%z bin/$$bin); \
		mb=$$(echo "scale=1; $$size / 1048576" | bc); \
		echo "  $$bin: $${mb}MB ($$size bytes)"; \
		if [ $$size -gt 3145728 ]; then echo "ERROR: $$bin exceeds 3MB"; exit 1; fi; \
	done
	@echo "✓ All binaries under 3MB"

dev-setup:
	@scripts/dev-setup.sh

test-defaults: defaults
	@scripts/test-defaults.sh

test-zero-install: defaults
	@scripts/test-zero-install.sh --skip-claude

clean-defaults:
	@echo "Cleaning defaults/ generated content..."
	@find defaults/ -mindepth 1 -not -name embed.go -delete 2>/dev/null || true
	@echo "✓ defaults/ cleaned (embed.go preserved)"

# CI check: ensure no residual .claude/ runtime write paths in production code
check-claude-writes:
	@scripts/check-claude-writes.sh
