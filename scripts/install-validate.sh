#!/bin/bash
# Install goyoke-validate to ~/.local/bin

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Build first
"$SCRIPT_DIR/build-validate.sh"

# Install
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

cp "$PROJECT_ROOT/bin/goyoke-validate" "$INSTALL_DIR/goyoke-validate"
chmod +x "$INSTALL_DIR/goyoke-validate"

echo "✓ Installed to: $INSTALL_DIR/goyoke-validate"
echo ""
echo "Make sure $INSTALL_DIR is in your PATH:"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
