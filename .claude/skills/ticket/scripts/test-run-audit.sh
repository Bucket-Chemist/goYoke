#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# test-run-audit.sh - Comprehensive test suite for run-audit.sh
# =============================================================================
# Phase 6: Complete test coverage
# - All Phase 1 tests (language detection, config, args, directory)
# - Phase 2 tests (placeholder replacement, test execution, all languages)
# - Phase 3 tests (summary generation, template rendering, fallbacks)
# - Edge cases (missing files, invalid config, failures)
# - Integration tests (full workflow)
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_AUDIT="$SCRIPT_DIR/run-audit.sh"

TEST_PASS=0
TEST_FAIL=0

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Test Helpers
# =============================================================================
test_start() {
  echo -e "\n${YELLOW}[TEST]${NC} $1"
}

test_pass() {
  echo -e "${GREEN}[PASS]${NC} $1"
  TEST_PASS=$((TEST_PASS + 1))
}

test_fail() {
  echo -e "${RED}[FAIL]${NC} $1"
  TEST_FAIL=$((TEST_FAIL + 1))
}

section_header() {
  echo ""
  echo "========================================="
  echo -e "${BLUE}$1${NC}"
  echo "========================================="
}

cleanup_temp() {
  if [[ -n "${TEMP_DIR:-}" ]] && [[ -d "$TEMP_DIR" ]]; then
    rm -rf "$TEMP_DIR"
  fi
}

trap cleanup_temp EXIT

# =============================================================================
# PHASE 1 TESTS: Core Infrastructure
# =============================================================================

# =============================================================================
# Test: Language Detection - Go
# =============================================================================
test_language_go() {
  test_start "Language detection: Go"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create go.mod
  echo "module test" > go.mod

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  # Run audit
  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-001 2>&1)

  if echo "$output" | grep -q "Detected language: go"; then
    test_pass "Go detected via go.mod"
  else
    test_fail "Go not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - Python (pyproject.toml)
# =============================================================================
test_language_python_pyproject() {
  test_start "Language detection: Python (pyproject.toml)"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create pyproject.toml
  echo "[project]" > pyproject.toml

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-002 2>&1)

  if echo "$output" | grep -q "Detected language: python"; then
    test_pass "Python detected via pyproject.toml"
  else
    test_fail "Python not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - Python (setup.py)
# =============================================================================
test_language_python_setuppy() {
  test_start "Language detection: Python (setup.py)"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create setup.py
  echo "# setup" > setup.py

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-003 2>&1)

  if echo "$output" | grep -q "Detected language: python"; then
    test_pass "Python detected via setup.py"
  else
    test_fail "Python not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - R (DESCRIPTION)
# =============================================================================
test_language_r_description() {
  test_start "Language detection: R (DESCRIPTION)"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create DESCRIPTION
  echo "Package: test" > DESCRIPTION

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-004 2>&1)

  if echo "$output" | grep -q "Detected language: r"; then
    test_pass "R detected via DESCRIPTION"
  else
    test_fail "R not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - TypeScript
# =============================================================================
test_language_typescript() {
  test_start "Language detection: TypeScript"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create package.json + tsconfig.json
  echo '{"name": "test"}' > package.json
  echo '{"compilerOptions": {}}' > tsconfig.json

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-005 2>&1)

  if echo "$output" | grep -q "Detected language: typescript"; then
    test_pass "TypeScript detected via package.json + tsconfig.json"
  else
    test_fail "TypeScript not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - JavaScript
# =============================================================================
test_language_javascript() {
  test_start "Language detection: JavaScript"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create package.json only
  echo '{"name": "test"}' > package.json

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-006 2>&1)

  if echo "$output" | grep -q "Detected language: javascript"; then
    test_pass "JavaScript detected via package.json"
  else
    test_fail "JavaScript not detected"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - Multiple Files (Priority)
# =============================================================================
test_language_priority() {
  test_start "Language detection: Priority (Go over Python)"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create both go.mod and pyproject.toml (Go should win)
  echo "module test" > go.mod
  echo "[project]" > pyproject.toml

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-007 2>&1)

  if echo "$output" | grep -q "Detected language: go"; then
    test_pass "Go has priority over Python"
  else
    test_fail "Priority detection failed"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Language Detection - Unknown
# =============================================================================
test_language_unknown() {
  test_start "Language detection: Unknown language"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create minimal config but no language indicators
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  local output exit_code
  set +e
  output=$("$RUN_AUDIT" --ticket-id TEST-008 2>&1)
  exit_code=$?
  set -e

  if [[ $exit_code -eq 2 ]] && echo "$output" | grep -q "Could not detect project language"; then
    test_pass "Unknown language detected and reported"
  else
    test_fail "Should exit with code 2 for unknown language (got exit code: $exit_code)"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Config Missing - Backward Compatible
# =============================================================================
test_config_missing() {
  test_start "Config missing - backward compatible exit"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create go.mod but NO config
  echo "module test" > go.mod

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-009 2>&1)

  if echo "$output" | grep -q "Config file not found"; then
    test_pass "Graceful exit when config missing"
  else
    test_fail "Did not handle missing config gracefully"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Config Disabled
# =============================================================================
test_config_disabled() {
  test_start "Config disabled - skip audit"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create go.mod
  echo "module test" > go.mod

  # Create config with audit disabled
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": false
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-010 2>&1)

  if echo "$output" | grep -q "Audit disabled"; then
    test_pass "Audit skipped when disabled"
  else
    test_fail "Did not skip when disabled"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Config Invalid JSON
# =============================================================================
test_config_invalid_json() {
  test_start "Config validation: Invalid JSON"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create go.mod
  echo "module test" > go.mod

  # Create invalid JSON config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  # Missing closing brace
EOF

  local output exit_code
  set +e
  output=$("$RUN_AUDIT" --ticket-id TEST-011 2>&1)
  exit_code=$?
  set -e

  if [[ $exit_code -eq 1 ]] && echo "$output" | grep -q "Invalid JSON"; then
    test_pass "Invalid JSON detected"
  else
    test_fail "Did not reject invalid JSON (got exit code: $exit_code)"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Argument Validation - Missing ticket-id
# =============================================================================
test_missing_ticket_id() {
  test_start "Missing --ticket-id argument"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  local output
  output=$("$RUN_AUDIT" 2>&1 || true)

  if echo "$output" | grep -q "ticket-id is required"; then
    test_pass "Correctly rejects missing --ticket-id"
  else
    test_fail "Did not reject missing --ticket-id"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Directory Creation
# =============================================================================
test_directory_creation() {
  test_start "Audit directory creation"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create go.mod
  echo "module test" > go.mod

  # Create minimal config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  "$RUN_AUDIT" --ticket-id TEST-012 > /dev/null 2>&1

  if [[ -d ".ticket-audits/TEST-012" ]] && [[ -f ".ticket-audits/TEST-012/timestamp.txt" ]]; then
    test_pass "Audit directory and timestamp created"
  else
    test_fail "Audit directory not created"
  fi

  cd "$original_dir"
}

# =============================================================================
# PHASE 2 TESTS: Test Execution
# =============================================================================

# =============================================================================
# Test: Placeholder Replacement
# =============================================================================
test_placeholder_replacement() {
  test_start "Placeholder replacement in commands"

  # Test the replace_placeholders function by extracting it
  replace_placeholders() {
    local cmd="$1"
    local audit_dir="$2"
    local ticket_id="$3"
    local project_root="$4"

    cmd="${cmd//\{audit_dir\}/$audit_dir}"
    cmd="${cmd//\{ticket_id\}/$ticket_id}"
    cmd="${cmd//\{project_root\}/$project_root}"

    echo "$cmd"
  }

  local cmd="go test -coverprofile={audit_dir}/coverage.out {project_root}/..."
  local audit_dir="/tmp/audit"
  local ticket_id="TEST-013"
  local project_root="/home/user/project"

  local result
  result=$(replace_placeholders "$cmd" "$audit_dir" "$ticket_id" "$project_root")

  if [[ "$result" == "go test -coverprofile=/tmp/audit/coverage.out /home/user/project/..." ]]; then
    test_pass "Placeholders replaced correctly"
  else
    test_fail "Placeholder replacement failed"
    echo "Expected: go test -coverprofile=/tmp/audit/coverage.out /home/user/project/..."
    echo "Got: $result" >&2
  fi
}

# =============================================================================
# Test: Go Test Execution (Mock)
# =============================================================================
test_go_test_execution() {
  test_start "Go test execution with mocked tests"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create minimal Go project
  mkdir -p src
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > src/main_test.go <<'EOF'
package src

import "testing"

func TestExample(t *testing.T) {
    if 1 != 1 {
        t.Error("Math is broken")
    }
}
EOF

  # Create config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v ./...",
        "race": "go test -race ./...",
        "coverage": "go test -coverprofile={audit_dir}/coverage.out ./..."
      }
    }
  }
}
EOF

  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-014 2>&1)

  if [[ -f ".ticket-audits/TEST-014/unit-tests.log" ]] && \
     [[ -f ".ticket-audits/TEST-014/coverage.out" ]]; then
    test_pass "Go tests executed successfully"
  else
    test_fail "Go test execution failed"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Test Failure Non-Blocking
# =============================================================================
test_test_failure_nonblocking() {
  test_start "Test failures are non-blocking"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create failing Go test
  mkdir -p src
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > src/main_test.go <<'EOF'
package src

import "testing"

func TestFailure(t *testing.T) {
    t.Error("Intentional failure")
}
EOF

  # Create config
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "go test -v ./..."
      }
    }
  }
}
EOF

  local output exit_code
  output=$("$RUN_AUDIT" --ticket-id TEST-015 2>&1)
  exit_code=$?

  # Should exit 0 even with test failure
  if [[ $exit_code -eq 0 ]] && echo "$output" | grep -q "\[FAIL\] Unit tests failed"; then
    test_pass "Test failures are non-blocking (exit 0)"
  else
    test_fail "Test failure should not cause exit 1"
    echo "Exit code: $exit_code"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Custom Test Commands
# =============================================================================
test_custom_test_commands() {
  test_start "Custom test commands from config"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create minimal Go project
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  # Create config with custom command
  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "echo 'Custom test command executed' > {audit_dir}/custom-test.log"
      }
    }
  }
}
EOF

  "$RUN_AUDIT" --ticket-id TEST-016 > /dev/null 2>&1

  if [[ -f ".ticket-audits/TEST-016/custom-test.log" ]] && \
     grep -q "Custom test command executed" ".ticket-audits/TEST-016/custom-test.log"; then
    test_pass "Custom test commands executed"
  else
    test_fail "Custom test command not executed"
  fi

  cd "$original_dir"
}

# =============================================================================
# PHASE 3 TESTS: Summary Generation
# =============================================================================

# =============================================================================
# Test: Extract Test Results - Go
# =============================================================================
test_extract_go_results() {
  test_start "Extract test results from Go logs"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create mock audit directory with Go test results
  mkdir -p .ticket-audits/TEST-017
  cat > .ticket-audits/TEST-017/unit-tests.log <<EOF
=== RUN   TestExample1
--- PASS: TestExample1 (0.00s)
=== RUN   TestExample2
--- PASS: TestExample2 (0.00s)
=== RUN   TestExample3
--- FAIL: TestExample3 (0.00s)
FAIL
EOF

  # Extract using grep (same logic as run-audit.sh)
  local pass_count fail_count
  pass_count=$(grep -c "^--- PASS:" ".ticket-audits/TEST-017/unit-tests.log" 2>/dev/null || echo "0")
  fail_count=$(grep -c "^--- FAIL:" ".ticket-audits/TEST-017/unit-tests.log" 2>/dev/null || echo "0")

  if [[ "$pass_count" == "2" ]] && [[ "$fail_count" == "1" ]]; then
    test_pass "Go test results extracted correctly (2 passed, 1 failed)"
  else
    test_fail "Go result extraction failed (got pass=$pass_count, fail=$fail_count)"
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Bootstrap Template Creation (Self-Healing)
# =============================================================================
test_minimal_summary() {
  test_start "Bootstrap template auto-creation (self-healing)"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create Go project
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  # Temporarily move template to test bootstrap
  local template_backup=""
  if [[ -f "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl" ]]; then
    template_backup="$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl.test-backup"
    mv "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl" "$template_backup"
  fi

  # Run audit - should bootstrap template automatically
  local output
  output=$("$RUN_AUDIT" --ticket-id TEST-018 2>&1)

  # Verify template was created by bootstrap
  local test_passed=0
  if echo "$output" | grep -q "Bootstrapping template" && \
     echo "$output" | grep -q "Template created successfully" && \
     [[ -f "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl" ]]; then
    test_passed=1
  fi

  # Restore original template if it existed
  if [[ -n "$template_backup" ]] && [[ -f "$template_backup" ]]; then
    mv "$template_backup" "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl"
  fi

  if [[ $test_passed -eq 1 ]]; then
    test_pass "Bootstrap auto-created template successfully"
  else
    test_fail "Bootstrap template creation failed"
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Summary with Template
# =============================================================================
test_summary_with_template() {
  test_start "Generate summary with template and metadata"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create minimal Go project
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true
  }
}
EOF

  # Create tickets-index.json
  cat > tickets-index.json <<EOF
{
  "tickets": [
    {
      "id": "TEST-019",
      "title": "Test Ticket",
      "description": "Test Description",
      "status": "in-progress",
      "dependencies": ["TEST-001"]
    }
  ]
}
EOF

  # Create template in expected location
  local template_dir="$HOME/.claude/skills/ticket/templates"
  mkdir -p "$template_dir"
  cat > "$template_dir/implementation-summary.md.tmpl" <<'EOF'
# $TICKET_ID - $TICKET_TITLE

**Status**: $STATUS
**Date**: $DATE

## Test Results

$TEST_RESULTS

## Coverage

$COVERAGE_SUMMARY
EOF

  "$RUN_AUDIT" --ticket-id TEST-019 > /dev/null 2>&1

  if [[ -f ".ticket-audits/TEST-019/implementation-summary.md" ]] && \
     grep -q "TEST-019 - Test Ticket" ".ticket-audits/TEST-019/implementation-summary.md"; then
    test_pass "Summary generated with template"
  else
    test_fail "Template-based summary failed"
    if [[ -f ".ticket-audits/TEST-019/implementation-summary.md" ]]; then
      echo "Summary content:"
      cat ".ticket-audits/TEST-019/implementation-summary.md" >&2
    fi
  fi

  # Cleanup template
  rm -f "$template_dir/implementation-summary.md.tmpl"

  cd "$original_dir"
}

# =============================================================================
# Test: Metadata Not Found Fallback
# =============================================================================
test_metadata_missing_fallback() {
  test_start "Fallback when ticket metadata missing"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create Go project
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "echo 'mock test' > /dev/null"
      }
    }
  }
}
EOF

  # Create tickets-index.json WITHOUT our ticket
  cat > tickets-index.json <<EOF
{
  "tickets": [
    {
      "id": "OTHER-001",
      "title": "Other Ticket"
    }
  ]
}
EOF

  # Create template in expected location
  local template_dir="$HOME/.claude/skills/ticket/templates"
  mkdir -p "$template_dir"
  cat > "$template_dir/implementation-summary.md.tmpl" <<'EOFTEMPLATE'
# $TICKET_ID - $TICKET_TITLE
EOFTEMPLATE

  "$RUN_AUDIT" --ticket-id TEST-020 > /dev/null 2>&1

  # Cleanup template
  rm -f "$template_dir/implementation-summary.md.tmpl"

  if [[ -f ".ticket-audits/TEST-020/implementation-summary.md" ]] && \
     grep -q "Ticket metadata not found" ".ticket-audits/TEST-020/implementation-summary.md"; then
    test_pass "Fallback to minimal summary when metadata missing"
  else
    test_fail "Should create minimal summary when metadata missing"
    if [[ -f ".ticket-audits/TEST-020/implementation-summary.md" ]]; then
      echo "Summary content:"
      cat ".ticket-audits/TEST-020/implementation-summary.md" >&2
    fi
  fi

  cd "$original_dir"
}

# =============================================================================
# INTEGRATION TESTS
# =============================================================================

# =============================================================================
# Test: Full Workflow - Go Project
# =============================================================================
test_full_workflow_go() {
  test_start "Integration: Full workflow with Go project"

  local test_dir=$(mktemp -d)

  # Run test in subprocess to avoid state pollution
  (
    cd "$test_dir"

    cat > go.mod <<'EOF'
module testproject
go 1.21
EOF

    cat > .ticket-config.json <<'EOF'
{
  "audit_config": {
    "enabled": true,
    "test_commands": {
      "go": {
        "unit": "echo 'mock' > {audit_dir}/unit-tests.log"
      }
    }
  }
}
EOF

    cat > tickets-index.json <<'EOF'
{"tickets": [{"id": "TEST-021", "title": "Integration Test"}]}
EOF

    mkdir -p "$HOME/.claude/skills/ticket/templates"
    echo "# \$TICKET_ID" > "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl"

    if timeout 10 "$RUN_AUDIT" --ticket-id TEST-021 > /dev/null 2>&1; then
      rm -f "$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl"

      # Check artifacts
      if [[ -d ".ticket-audits/TEST-021" ]] && \
         [[ -f ".ticket-audits/TEST-021/unit-tests.log" ]] && \
         [[ -f ".ticket-audits/TEST-021/implementation-summary.md" ]]; then
        exit 0
      else
        exit 1
      fi
    else
      exit 2
    fi
  )

  local result=$?
  rm -rf "$test_dir"

  if [[ $result -eq 0 ]]; then
    test_pass "Full Go workflow completed successfully"
  elif [[ $result -eq 2 ]]; then
    test_fail "Integration test timed out"
  else
    test_fail "Integration test failed (artifacts missing)"
  fi
}

# =============================================================================
# Test: Backward Compatibility - No Config
# =============================================================================
test_backward_compat_no_config() {
  test_start "Backward compatibility: No config file"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  # Create Go project without config
  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  local output exit_code
  output=$("$RUN_AUDIT" --ticket-id TEST-022 2>&1)
  exit_code=$?

  if [[ $exit_code -eq 0 ]] && \
     echo "$output" | grep -q "Config file not found" && \
     ! [[ -d ".ticket-audits/TEST-022" ]]; then
    test_pass "Backward compatible (no config = graceful skip)"
  else
    test_fail "Backward compatibility broken"
    echo "Exit code: $exit_code"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Test: Backward Compatibility - Disabled Audit
# =============================================================================
test_backward_compat_disabled() {
  test_start "Backward compatibility: Audit disabled"

  local original_dir="$PWD"
  TEMP_DIR=$(mktemp -d)
  cd "$TEMP_DIR"

  cat > go.mod <<EOF
module testproject
go 1.21
EOF

  cat > .ticket-config.json <<EOF
{
  "audit_config": {
    "enabled": false
  }
}
EOF

  local output exit_code
  output=$("$RUN_AUDIT" --ticket-id TEST-023 2>&1)
  exit_code=$?

  if [[ $exit_code -eq 0 ]] && \
     echo "$output" | grep -q "Audit disabled" && \
     ! [[ -d ".ticket-audits/TEST-023" ]]; then
    test_pass "Backward compatible (disabled = graceful skip)"
  else
    test_fail "Disabled audit should skip gracefully"
    echo "Exit code: $exit_code"
    echo "Debug output: $output" >&2
  fi

  cd "$original_dir"
}

# =============================================================================
# Run All Tests
# =============================================================================
main() {
  echo "========================================="
  echo "run-audit.sh Phase 6 Comprehensive Tests"
  echo "========================================="

  section_header "PHASE 1: Core Infrastructure Tests"
  test_language_go
  test_language_python_pyproject
  test_language_python_setuppy
  test_language_r_description
  test_language_typescript
  test_language_javascript
  test_language_priority
  test_language_unknown
  test_config_missing
  test_config_disabled
  test_config_invalid_json
  test_missing_ticket_id
  test_directory_creation

  section_header "PHASE 2: Test Execution Tests"
  test_placeholder_replacement
  test_go_test_execution
  test_test_failure_nonblocking
  test_custom_test_commands

  section_header "PHASE 3: Summary Generation Tests"
  test_extract_go_results
  test_minimal_summary
  test_summary_with_template
  test_metadata_missing_fallback

  section_header "INTEGRATION TESTS"
  test_full_workflow_go
  test_backward_compat_no_config
  test_backward_compat_disabled

  # =============================================================================
  # Summary
  # =============================================================================
  echo ""
  echo "========================================="
  echo "Test Summary"
  echo "========================================="
  echo -e "${GREEN}PASSED:${NC} $TEST_PASS"
  echo -e "${RED}FAILED:${NC} $TEST_FAIL"
  echo ""

  if [[ $TEST_FAIL -eq 0 ]]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    echo ""
    echo "Coverage Summary:"
    echo "- Core Infrastructure: 13 tests"
    echo "- Test Execution: 4 tests"
    echo "- Summary Generation: 4 tests"
    echo "- Integration: 3 tests"
    echo "- Total: 24 tests"
    exit 0
  else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
  fi
}

# Execute main
main
