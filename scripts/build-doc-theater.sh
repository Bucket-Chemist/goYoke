#!/bin/bash
set -euo pipefail

echo "Building goyoke-doc-theater..."

cd "$(dirname "$0")/.."

go build -o bin/goyoke-doc-theater ./cmd/goyoke-doc-theater

if [[ $? -eq 0 ]]; then
    echo "✓ Built: bin/goyoke-doc-theater"
    echo "  Size: $(du -h bin/goyoke-doc-theater | cut -f1)"
else
    echo "✗ Build failed"
    exit 1
fi
