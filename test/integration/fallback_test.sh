#!/bin/bash
# Fallback Test - Verifies Go → Bash fallback mechanism
# Tests that wrapper correctly falls back to bash hook when Go CLI fails
set -euo pipefail

echo "Testing Go → Bash fallback..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# Create fake goyoke-archive that fails
mkdir -p "$TMPDIR/bin"
cat > "$TMPDIR/bin/goyoke-archive" <<'EOF'
#!/bin/bash
# Fake goyoke-archive that always fails
echo "FATAL: Mock failure for testing" >&2
exit 1
EOF
chmod +x "$TMPDIR/bin/goyoke-archive"

# Put fake binary first in PATH
export PATH="$TMPDIR/bin:$PATH"

# Create bash hook mock
mkdir -p "$TMPDIR/.claude/hooks"
cat > "$TMPDIR/.claude/hooks/session-archive.sh" <<'EOF'
#!/bin/bash
# Mock bash hook
cat >/dev/null  # Consume STDIN
echo '{"hookSpecificOutput":{"source":"bash_fallback"}}'
EOF
chmod +x "$TMPDIR/.claude/hooks/session-archive.sh"

# Run wrapper (should fall back to bash)
export HOME="$TMPDIR"
cat > "$TMPDIR/session.json" <<EOF
{"session_id":"test","timestamp":123,"hook_event_name":"SessionEnd"}
EOF

OUTPUT=$(scripts/session-archive-wrapper.sh < "$TMPDIR/session.json" 2>&1)

if echo "$OUTPUT" | grep -q "bash_fallback"; then
    echo "✅ Fallback to bash hook succeeded"
else
    echo "❌ Fallback failed"
    echo "$OUTPUT"
    exit 1
fi
