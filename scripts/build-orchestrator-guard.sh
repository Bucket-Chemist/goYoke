#!/bin/bash
set -euo pipefail

echo "Building gogent-orchestrator-guard..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-orchestrator-guard ./cmd/gogent-orchestrator-guard

if [[ $? -eq 0 ]]; then
    echo "✓ Built: bin/gogent-orchestrator-guard"
    echo "  Size: $(du -h bin/gogent-orchestrator-guard | cut -f1)"
else
    echo "✗ Build failed"
    exit 1
fi
