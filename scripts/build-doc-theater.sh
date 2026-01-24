#!/bin/bash
set -euo pipefail

echo "Building gogent-doc-theater..."

cd "$(dirname "$0")/.."

go build -o bin/gogent-doc-theater ./cmd/gogent-doc-theater

if [[ $? -eq 0 ]]; then
    echo "✓ Built: bin/gogent-doc-theater"
    echo "  Size: $(du -h bin/gogent-doc-theater | cut -f1)"
else
    echo "✗ Build failed"
    exit 1
fi
