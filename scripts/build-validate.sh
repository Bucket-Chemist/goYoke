#!/bin/bash
# Build goyoke-validate binary

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

echo "Building goyoke-validate..."
go build -o bin/goyoke-validate cmd/goyoke-validate/main.go

echo "✓ Built: bin/goyoke-validate"
echo ""
echo "Test with:"
echo "  echo '{\"tool_name\":\"Task\",\"tool_input\":{\"model\":\"opus\"},\"session_id\":\"test\"}' | ./bin/goyoke-validate"
