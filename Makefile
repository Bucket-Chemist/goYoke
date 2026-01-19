# Makefile for GOgent Fortress Go migration
# Project: GOgent-Fortress
# Purpose: Convenient task automation

BINARY_NAME=gogent
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: help test test-ecosystem test-unit test-integration test-race coverage build build-archive install-archive clean

help:
	@echo "GOgent Fortress - Available targets:"
	@echo "  make test            - Run full test ecosystem (alias for test-ecosystem)"
	@echo "  make test-ecosystem  - Run complete test suite with audit trail"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration - Run integration tests only"
	@echo "  make test-race       - Run race detector"
	@echo "  make coverage        - Generate coverage report"
	@echo "  make build           - Build binary"
	@echo "  make build-archive   - Build gogent-archive binary"
	@echo "  make install-archive - Install gogent-archive to ~/.local/bin"
	@echo "  make clean           - Remove build artifacts"

# Primary test target - runs full ecosystem with audit trail
test: test-ecosystem

test-ecosystem:
	@./scripts/test-ecosystem.sh

# Individual test targets for granular testing
test-unit:
	@echo "Running unit tests..."
	@go test -v ./pkg/routing/...

test-integration:
	@echo "Running integration tests..."
	@go test -v -run 'TestEcosystem_' ./pkg/routing

test-race:
	@echo "Running race detector..."
	@go test -race ./pkg/routing/...

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./pkg/routing/...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report:"
	@echo "  go tool cover -html=coverage.out"

# Build targets
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/${BINARY_NAME}

build-archive:
	@echo "Building gogent-archive binary..."
	go build -o bin/gogent-archive ./cmd/gogent-archive
	@echo "✅ Binary created at bin/gogent-archive"

install-archive: build-archive
	@echo "Installing gogent-archive to ~/.local/bin/..."
	mkdir -p ~/.local/bin
	cp bin/gogent-archive ~/.local/bin/gogent-archive
	chmod +x ~/.local/bin/gogent-archive
	@echo "✅ Installed to ~/.local/bin/gogent-archive"
	@echo "Ensure ~/.local/bin is in your PATH"

clean:
	rm -f ${BINARY_NAME}
	rm -f bin/gogent-archive
	rm -f coverage.out
	rm -f *.test
