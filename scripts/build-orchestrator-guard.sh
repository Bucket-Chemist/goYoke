#!/bin/bash
set -euo pipefail

echo "Building goyoke-orchestrator-guard..."

cd "$(dirname "$0")/.."

go build -o bin/goyoke-orchestrator-guard ./cmd/goyoke-orchestrator-guard

if [[ $? -eq 0 ]]; then
    echo "✓ Built: bin/goyoke-orchestrator-guard"
    echo "  Size: $(du -h bin/goyoke-orchestrator-guard | cut -f1)"
else
    echo "✗ Build failed"
    exit 1
fi
