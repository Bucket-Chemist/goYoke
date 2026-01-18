#!/bin/bash
# Build gogent-validate binary

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

echo "Building gogent-validate..."
go build -o bin/gogent-validate cmd/gogent-validate/main.go

echo "✓ Built: bin/gogent-validate"
echo ""
echo "Test with:"
echo "  echo '{\"tool_name\":\"Task\",\"tool_input\":{\"model\":\"opus\"},\"session_id\":\"test\"}' | ./bin/gogent-validate"
