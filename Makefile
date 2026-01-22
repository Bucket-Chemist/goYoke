# Makefile for GOgent Fortress Go migration
# Project: GOgent-Fortress
# Purpose: Convenient task automation

BINARY_NAME=gogent
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: help test test-ecosystem test-unit test-integration test-race coverage build build-archive build-validate build-aggregate install install-archive install-aggregate install-wrapper uninstall uninstall-aggregate check-path clean test-simulation test-simulation-fuzz test-simulation-deterministic replay-crash clean-simulation

help:
	@echo "GOgent Fortress - Available targets:"
	@echo "  make test            - Run full test ecosystem (alias for test-ecosystem)"
	@echo "  make test-ecosystem  - Run complete test suite with audit trail"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-race       - Run race detector"
	@echo "  make coverage        - Generate coverage report"
	@echo "  make build           - Build binary"
	@echo "  make build-validate  - Build gogent-validate binary"
	@echo "  make build-archive   - Build gogent-archive binary"
	@echo "  make build-aggregate - Build gogent-aggregate binary"
	@echo "  make install         - Install all CLIs to ~/.local/bin"
	@echo "  make install-archive - Install gogent-archive to ~/.local/bin"
	@echo "  make install-aggregate - Install gogent-aggregate to ~/.local/bin"
	@echo "  make install-wrapper - Install session-archive wrapper hook"
	@echo "  make uninstall       - Remove all CLIs from ~/.local/bin"
	@echo "  make check-path      - Verify ~/.local/bin is in PATH"
	@echo "  make clean           - Remove build artifacts"
	@echo ""
	@echo "Simulation testing:"
	@echo "  make test-simulation              - Run mixed simulation (deterministic + fuzz)"
	@echo "  make test-simulation-deterministic - Run deterministic tests only"
	@echo "  make test-simulation-fuzz         - Run fuzz tests only"
	@echo "  make replay-crash CRASH=<file>    - Replay a specific crash"
	@echo "  make clean-simulation             - Clean simulation artifacts"

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
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/${BINARY_NAME}

build-validate:
	@echo "Building gogent-validate binary..."
	go build -o bin/gogent-validate ./cmd/gogent-validate
	@echo "✅ Binary created at bin/gogent-validate"

build-archive:
	@echo "Building gogent-archive binary..."
	go build -o bin/gogent-archive ./cmd/gogent-archive
	@echo "✅ Binary created at bin/gogent-archive"

build-aggregate:
	@echo "Building gogent-aggregate binary..."
	go build -o bin/gogent-aggregate ./cmd/gogent-aggregate
	@echo "✅ Binary created at bin/gogent-aggregate"

install: build-validate build-archive build-aggregate check-path
	@echo "Installing GOgent-Fortress CLIs to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/gogent-validate ~/.local/bin/gogent-validate
	cp bin/gogent-archive ~/.local/bin/gogent-archive
	cp bin/gogent-aggregate ~/.local/bin/gogent-aggregate
	chmod +x ~/.local/bin/gogent-validate
	chmod +x ~/.local/bin/gogent-archive
	chmod +x ~/.local/bin/gogent-aggregate
	@echo "✅ Installed gogent-validate, gogent-archive, gogent-aggregate"
	@echo ""
	@$(MAKE) check-path

install-archive: build-archive
	@echo "Installing gogent-archive to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/gogent-archive ~/.local/bin/gogent-archive
	chmod +x ~/.local/bin/gogent-archive
	@echo "✅ Installed to ~/.local/bin/gogent-archive"
	@echo "Ensure ~/.local/bin is in your PATH"

install-aggregate: build-aggregate
	@echo "Installing gogent-aggregate to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/gogent-aggregate ~/.local/bin/gogent-aggregate
	chmod +x ~/.local/bin/gogent-aggregate
	@echo "✅ Installed to ~/.local/bin/gogent-aggregate"
	@echo "Ensure ~/.local/bin is in your PATH"

install-wrapper:
	@echo "Installing session-archive wrapper hook..."
	mkdir -p ~/.claude/hooks
	mkdir -p ~/.gogent
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
	@echo "Uninstalling GOgent-Fortress CLIs from ~/.local/bin/..."
	rm -f ~/.local/bin/gogent-validate
	rm -f ~/.local/bin/gogent-archive
	rm -f ~/.local/bin/gogent-aggregate
	@echo "✅ Uninstalled all CLIs"

uninstall-aggregate:
	@echo "Removing gogent-aggregate from ~/.local/bin/..."
	rm -f ~/.local/bin/gogent-aggregate
	@echo "✅ gogent-aggregate removed"

clean:
	rm -f ${BINARY_NAME}
	rm -f bin/gogent-validate
	rm -f bin/gogent-archive
	rm -f bin/gogent-aggregate
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
