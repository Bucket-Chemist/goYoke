#!/usr/bin/env bash
# run-naked-test.sh — Build and run the Docker isolated test
#
# This proves the single binary works from scratch:
# - Fresh Debian container
# - Only Claude CLI (npm) installed
# - Single goyoke binary copied in
# - Auth via mounted credentials
# - No ~/.claude config, no goYoke config, no git repo
#
# Usage: ./test/docker/run-naked-test.sh [--build-only | --shell]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_DIR="$SCRIPT_DIR"
CREDS="${HOME}/.claude/.credentials.json"

MODE="run"
for arg in "$@"; do
    case "$arg" in
        --build-only) MODE="build" ;;
        --shell) MODE="shell" ;;
    esac
done

# Step 1: Build the goYoke binary for linux/amd64
echo "[docker-test] Building goyoke binary (linux/amd64)..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-w -X main.version=$(git -C "$PROJECT_ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o "${DOCKER_DIR}/goyoke" \
    "${PROJECT_ROOT}/cmd/goyoke/"

SIZE=$(stat -c%s "${DOCKER_DIR}/goyoke" 2>/dev/null || stat -f%z "${DOCKER_DIR}/goyoke")
MB=$(echo "scale=1; $SIZE / 1048576" | bc)
echo "[docker-test] Binary: ${MB}MB"

# Step 2: Build Docker image
echo "[docker-test] Building Docker image..."
docker build -t goyoke-naked-test "$DOCKER_DIR"

# Clean up binary (it's in the image now)
rm -f "${DOCKER_DIR}/goyoke"

if [ "$MODE" = "build" ]; then
    echo "[docker-test] Image built. Run with: $0"
    exit 0
fi

# Step 3: Check credentials
if [ ! -f "$CREDS" ]; then
    echo "[docker-test] ERROR: ${CREDS} not found."
    echo "  Authenticate with 'claude' first, then retry."
    exit 1
fi

# Step 4: Run
echo ""
echo "============================================"
echo "[docker-test] Running in isolated container:"
echo "  Image: goyoke-naked-test (Debian + Claude CLI + goyoke binary)"
echo "  Mount: credentials only"
echo "  No ~/.claude config, no goYoke config, no .git"
echo ""
echo "  If TUI starts and hooks fire → ZERO INSTALL PROVEN"
echo "============================================"
echo ""

DOCKER_ARGS=(
    --rm
    -it
    -e "TERM=${TERM:-xterm-256color}"
)

# Auth: prefer CLAUDE_CODE_OAUTH_TOKEN, then ANTHROPIC_API_KEY, then credentials file
if [ -n "${CLAUDE_CODE_OAUTH_TOKEN:-}" ]; then
    echo "[docker-test] Auth: CLAUDE_CODE_OAUTH_TOKEN (setup-token)"
    DOCKER_ARGS+=(-e "CLAUDE_CODE_OAUTH_TOKEN=${CLAUDE_CODE_OAUTH_TOKEN}")
elif [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    echo "[docker-test] Auth: ANTHROPIC_API_KEY (env var)"
    DOCKER_ARGS+=(-e "ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}")
elif [ -f "$CREDS" ]; then
    echo "[docker-test] Auth: mounting credentials file (may not work without keychain)"
    echo "[docker-test] TIP: Set CLAUDE_CODE_OAUTH_TOKEN for reliable Docker auth"
    echo "[docker-test]      Get one with: docker run --rm -it goyoke-naked-test claude setup-token"
    DOCKER_ARGS+=(-v "${CREDS}:/home/testuser/.claude/.credentials.json:ro")
else
    echo "[docker-test] ERROR: No auth available."
    echo "  Option 1: CLAUDE_CODE_OAUTH_TOKEN=... (run 'claude setup-token' to get one)"
    echo "  Option 2: ANTHROPIC_API_KEY=sk-ant-..."
    exit 1
fi

if [ "$MODE" = "shell" ]; then
    echo "[docker-test] Dropping to shell (run 'goyoke' manually)..."
    docker run "${DOCKER_ARGS[@]}" goyoke-naked-test /bin/bash
else
    docker run "${DOCKER_ARGS[@]}" goyoke-naked-test
fi
