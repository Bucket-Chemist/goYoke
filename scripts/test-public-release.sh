#!/usr/bin/env bash
# test-public-release.sh — End-user perspective test of the public release
#
# Tests every install method against the public repo to verify
# a real user can install and run goYoke.
#
# Usage: ./scripts/test-public-release.sh [--tag v0.5.1] [--skip-brew] [--skip-go-install]
#
# Requires: curl, jq, go
# Optional: brew (for Homebrew test)

set -euo pipefail

PUBLIC_REPO="Bucket-Chemist/goYoke"
TAG=""
SKIP_BREW=false
SKIP_GO_INSTALL=false

for arg in "$@"; do
    case "$arg" in
        --tag=*) TAG="${arg#--tag=}" ;;
        --skip-brew) SKIP_BREW=true ;;
        --skip-go-install) SKIP_GO_INSTALL=true ;;
    esac
done

if [ -z "$TAG" ]; then
    TAG=$(gh release list --repo "$PUBLIC_REPO" --limit 1 --json tagName -q '.[0].tagName' 2>/dev/null || echo "")
    if [ -z "$TAG" ]; then
        echo "ERROR: Could not detect latest tag. Pass --tag=vX.Y.Z"
        exit 1
    fi
    echo "Auto-detected latest release: $TAG"
fi

VERSION="${TAG#v}"
PASS=0
FAIL=0
SKIP=0
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP: $1"; SKIP=$((SKIP + 1)); }

echo "[test-public-release] Testing goYoke $TAG"
echo "  Repo: $PUBLIC_REPO"
echo "  Work dir: $WORK_DIR"
echo ""

# ============================================================
# 1. Release assets exist and download
# ============================================================
echo "=== 1. Release asset verification ==="

EXPECTED_ASSETS=(
    "goYoke_${VERSION}_linux_amd64.tar.gz"
    "goYoke_${VERSION}_darwin_amd64.tar.gz"
    "goYoke_${VERSION}_darwin_arm64.tar.gz"
    "goYoke_${VERSION}_windows_amd64.zip"
    "checksums.txt"
)

for asset in "${EXPECTED_ASSETS[@]}"; do
    URL="https://github.com/${PUBLIC_REPO}/releases/download/${TAG}/${asset}"
    STATUS=$(curl -sL -o /dev/null -w "%{http_code}" "$URL")
    if [ "$STATUS" = "200" ]; then
        pass "$asset (HTTP $STATUS)"
    else
        fail "$asset (HTTP $STATUS)"
    fi
done

# ============================================================
# 2. Binary download and smoke test
# ============================================================
echo ""
echo "=== 2. Binary download and smoke test ==="

ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$ARCH" in
    x86_64) GOARCH="amd64" ;;
    aarch64|arm64) GOARCH="arm64" ;;
    *) GOARCH="$ARCH" ;;
esac

TARBALL="goYoke_${VERSION}_${OS}_${GOARCH}.tar.gz"
DL_URL="https://github.com/${PUBLIC_REPO}/releases/download/${TAG}/${TARBALL}"

echo "  Downloading: $TARBALL"
if curl -sL "$DL_URL" | tar xz -C "$WORK_DIR" 2>/dev/null; then
    pass "download and extract"
else
    fail "download and extract ($DL_URL)"
fi

BINARY="$WORK_DIR/goyoke"
if [ -x "$BINARY" ]; then
    pass "binary is executable"
else
    fail "binary is executable"
fi

if "$BINARY" version >/dev/null 2>&1; then
    VER_OUT=$("$BINARY" version 2>&1 || true)
    pass "goyoke version ($VER_OUT)"
else
    fail "goyoke version"
fi

# Hook dispatch smoke test
MOCK_EVENT='{"hook_event_name":"test","tool_name":"Bash","session_id":"test"}'
if echo "$MOCK_EVENT" | "$BINARY" hook validate >/dev/null 2>&1; then
    pass "hook dispatch (validate)"
else
    OUTPUT=$(echo "$MOCK_EVENT" | "$BINARY" hook validate 2>&1 || true)
    if echo "$OUTPUT" | grep -q 'panic:'; then
        fail "hook dispatch panicked"
    else
        pass "hook dispatch (non-zero exit OK)"
    fi
fi

# ============================================================
# 3. SHA256 verification
# ============================================================
echo ""
echo "=== 3. Checksum verification ==="

CHECKSUMS_URL="https://github.com/${PUBLIC_REPO}/releases/download/${TAG}/checksums.txt"
curl -sL "$CHECKSUMS_URL" > "$WORK_DIR/checksums.txt"

if [ -f "$WORK_DIR/$TARBALL" ] 2>/dev/null || curl -sL "$DL_URL" -o "$WORK_DIR/$TARBALL"; then
    EXPECTED_SHA=$(grep "$TARBALL" "$WORK_DIR/checksums.txt" | awk '{print $1}')
    ACTUAL_SHA=$(sha256sum "$WORK_DIR/$TARBALL" 2>/dev/null | awk '{print $1}' || shasum -a 256 "$WORK_DIR/$TARBALL" | awk '{print $1}')
    if [ "$EXPECTED_SHA" = "$ACTUAL_SHA" ]; then
        pass "SHA256 matches checksums.txt"
    else
        fail "SHA256 mismatch: expected=$EXPECTED_SHA actual=$ACTUAL_SHA"
    fi
else
    fail "could not download tarball for checksum verification"
fi

# ============================================================
# 4. Build from public source
# ============================================================
echo ""
echo "=== 4. Build from public source (clone + go build) ==="

if gh repo clone "$PUBLIC_REPO" "$WORK_DIR/source" -- --depth=1 2>/dev/null; then
    pass "clone public repo"
    if (cd "$WORK_DIR/source" && go build ./... 2>&1); then
        pass "go build ./..."
    else
        fail "go build ./... (public source doesn't compile)"
    fi
else
    fail "clone public repo"
fi

# ============================================================
# 5. go install
# ============================================================
echo ""
echo "=== 5. go install ==="

if [ "$SKIP_GO_INSTALL" = true ]; then
    skip "go install (--skip-go-install)"
else
    export GOBIN="$WORK_DIR/gobin"
    mkdir -p "$GOBIN"
    if go install "github.com/${PUBLIC_REPO}/cmd/goyoke@${TAG}" 2>&1; then
        pass "go install @$TAG"
        if "$GOBIN/goyoke" version >/dev/null 2>&1; then
            pass "go install binary runs"
        else
            fail "go install binary runs"
        fi
    else
        fail "go install @$TAG"
    fi
fi

# ============================================================
# 6. Homebrew formula
# ============================================================
echo ""
echo "=== 6. Homebrew ==="

if [ "$SKIP_BREW" = true ]; then
    skip "Homebrew (--skip-brew)"
elif ! command -v brew >/dev/null 2>&1; then
    skip "Homebrew (brew not installed)"
else
    FORMULA_URL="https://raw.githubusercontent.com/Bucket-Chemist/homebrew-tap/main/goyoke.rb"
    FORMULA=$(curl -sL "$FORMULA_URL")
    if echo "$FORMULA" | grep -q "version \"${VERSION}\""; then
        pass "formula version matches $VERSION"
    else
        fail "formula version doesn't match $VERSION"
    fi
    if echo "$FORMULA" | grep -q "github.com/Bucket-Chemist/goYoke/releases"; then
        pass "formula URLs point to public repo"
    else
        fail "formula URLs don't point to public repo"
    fi
    # Don't actually install — just verify the formula is valid
    skip "brew install (manual test: brew install Bucket-Chemist/tap/goyoke)"
fi

# ============================================================
# 7. AUR package
# ============================================================
echo ""
echo "=== 7. AUR ==="

AUR_INFO=$(curl -s "https://aur.archlinux.org/rpc/v5/info?arg[]=goyoke-bin" 2>/dev/null)
AUR_VERSION=$(echo "$AUR_INFO" | jq -r '.results[0].Version // empty' 2>/dev/null)
if [ -n "$AUR_VERSION" ]; then
    if [[ "$AUR_VERSION" == "${VERSION}"* ]]; then
        pass "AUR goyoke-bin version $AUR_VERSION"
    else
        fail "AUR version mismatch: expected ${VERSION}*, got $AUR_VERSION"
    fi
else
    fail "AUR goyoke-bin not found"
fi

# ============================================================
# 8. No private content in public repo
# ============================================================
echo ""
echo "=== 8. Private content check ==="

if [ -d "$WORK_DIR/source" ]; then
    PAT="genomics-reviewer|proteomics-reviewer|proteogenomics-reviewer|proteoform-reviewer|mass-spec-reviewer|bioinformatician-reviewer|pasteur|staff-bioinformatician|python-architect|llm-inference-architect"

    # Check agent directories
    PRIVATE_DIRS=$(find "$WORK_DIR/source/.claude/agents" -maxdepth 1 -type d | xargs -I{} basename {} | grep -E "$PAT" || true)
    if [ -z "$PRIVATE_DIRS" ]; then
        pass "no private agent directories"
    else
        fail "private agent dirs found: $PRIVATE_DIRS"
    fi

    # Check defaults
    PRIVATE_DEFAULTS=$(find "$WORK_DIR/source/defaults" -name "*bioinformatics*" 2>/dev/null || true)
    if [ -z "$PRIVATE_DEFAULTS" ]; then
        pass "no bioinformatics content in defaults"
    else
        fail "bioinformatics in defaults: $PRIVATE_DEFAULTS"
    fi

    # Check private conventions
    if [ -f "$WORK_DIR/source/.claude/conventions/python-datasci.md" ] || [ -f "$WORK_DIR/source/.claude/conventions/python-ml.md" ]; then
        fail "private conventions found"
    else
        pass "no private conventions"
    fi
fi

# ============================================================
# Summary
# ============================================================
echo ""
echo "================================"
echo "  Passed:  $PASS"
echo "  Failed:  $FAIL"
echo "  Skipped: $SKIP"
echo "================================"

if [ "$FAIL" -gt 0 ]; then
    echo ""
    echo "PUBLIC RELEASE TEST FAILED"
    exit 1
fi

echo ""
echo "PUBLIC RELEASE TEST PASSED"
exit 0
