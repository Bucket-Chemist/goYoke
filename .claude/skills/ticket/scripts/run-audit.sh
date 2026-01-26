#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# run-audit.sh - Execute automated ticket audit tests
# =============================================================================
# Phase 1: Core Infrastructure
# - Argument parsing
# - Config loading and validation
# - Language detection
# - Directory creation
# =============================================================================

# Default values
TICKET_ID=""
CONFIG_FILE=".ticket-config.json"
PROJECT_ROOT="$(pwd)"

# =============================================================================
# Usage
# =============================================================================
usage() {
  cat <<EOF
Usage: run-audit.sh --ticket-id TICKET_ID [--config CONFIG_FILE]

Execute automated audit tests for a ticket.

Options:
  --ticket-id ID      Ticket ID to audit (required)
  --config FILE       Path to .ticket-config.json (default: .ticket-config.json)
  -h, --help          Show this help message

Environment:
  PROJECT_ROOT        Project root directory (default: current directory)

Exit codes:
  0   Success or audit disabled
  1   Invalid arguments or configuration error
  2   Language detection failed
  3   Audit execution failed
EOF
  exit 0
}

# =============================================================================
# Argument Parsing
# =============================================================================
parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --ticket-id)
        TICKET_ID="$2"
        shift 2
        ;;
      --config)
        CONFIG_FILE="$2"
        shift 2
        ;;
      -h|--help)
        usage
        ;;
      *)
        echo "Error: Unknown argument: $1" >&2
        echo "Run with --help for usage information" >&2
        exit 1
        ;;
    esac
  done

  # Validate required arguments
  if [[ -z "$TICKET_ID" ]]; then
    echo "Error: --ticket-id is required" >&2
    echo "Run with --help for usage information" >&2
    exit 1
  fi
}

# =============================================================================
# Config Loading and Validation
# =============================================================================
load_config() {
  # Backward compatibility: if config doesn't exist, skip audit gracefully
  if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "[INFO] Config file not found: $CONFIG_FILE. Skipping audit (backward compatible)."
    exit 0
  fi

  # Validate JSON syntax
  if ! jq empty "$CONFIG_FILE" 2>/dev/null; then
    echo "Error: Invalid JSON in config file: $CONFIG_FILE" >&2
    exit 1
  fi

  # Check if audit is enabled
  local enabled
  enabled=$(jq -r '.audit_config.enabled // false' "$CONFIG_FILE")

  if [[ "$enabled" != "true" ]]; then
    echo "[INFO] Audit disabled in config. Skipping audit."
    exit 0
  fi

  echo "[INFO] Audit enabled. Config loaded from: $CONFIG_FILE"
}

# =============================================================================
# Language Detection
# =============================================================================
detect_language() {
  local lang="unknown"

  # Priority-based detection
  # 1. Go: go.mod presence
  if [[ -f "$PROJECT_ROOT/go.mod" ]]; then
    lang="go"
  # 2. Python: pyproject.toml OR setup.py
  elif [[ -f "$PROJECT_ROOT/pyproject.toml" ]] || [[ -f "$PROJECT_ROOT/setup.py" ]]; then
    lang="python"
  # 3. R: DESCRIPTION OR *.Rproj files
  elif [[ -f "$PROJECT_ROOT/DESCRIPTION" ]] || compgen -G "$PROJECT_ROOT/*.Rproj" > /dev/null; then
    lang="r"
  # 4. TypeScript: package.json + tsconfig.json
  elif [[ -f "$PROJECT_ROOT/package.json" ]] && [[ -f "$PROJECT_ROOT/tsconfig.json" ]]; then
    lang="typescript"
  # 5. JavaScript: package.json alone
  elif [[ -f "$PROJECT_ROOT/package.json" ]]; then
    lang="javascript"
  fi

  echo "$lang"
}

# =============================================================================
# Directory Creation
# =============================================================================
create_audit_directory() {
  local timestamp
  timestamp=$(date -Iseconds)

  local audit_dir
  audit_dir=$(jq -r '.audit_config.audit_dir // ".ticket-audits"' "$CONFIG_FILE")/${TICKET_ID}

  # Create directory structure
  mkdir -p "$audit_dir"

  # Create timestamp file
  echo "$timestamp" > "$audit_dir/timestamp.txt"

  echo "[INFO] Created audit directory: $audit_dir" >&2
  echo "$audit_dir"
}

# =============================================================================
# Bootstrap Templates (Self-Healing)
# =============================================================================
bootstrap_templates() {
  # Safety checks
  if [[ -z "${HOME:-}" ]]; then
    echo "[WARN] \$HOME not set, cannot bootstrap templates" >&2
    return 0
  fi

  local template_dir="$HOME/.claude/skills/ticket/templates"
  local template_file="$template_dir/implementation-summary.md.tmpl"

  # Create template directory if missing
  if [[ ! -d "$template_dir" ]]; then
    echo "[INFO] Creating template directory: $template_dir" >&2
    mkdir -p "$template_dir" || {
      echo "[WARN] Failed to create template directory" >&2
      return 0
    }
  fi

  # Check if template exists
  if [[ -f "$template_file" ]]; then
    # Template exists, nothing to do
    return 0
  fi

  # Check for symlinks (safety)
  if [[ -L "$template_file" ]]; then
    echo "[WARN] Template path is a symlink, skipping bootstrap" >&2
    return 0
  fi

  # Embed template and write atomically
  echo "[INFO] Bootstrapping template: $template_file" >&2

  local temp_file="${template_file}.tmp.$$"

  # Write template from heredoc
  cat > "$temp_file" <<'TEMPLATE_EOF'
# ${TICKET_ID}: ${TICKET_TITLE} - Implementation Summary

**Ticket ID**: ${TICKET_ID}
**Title**: ${TICKET_TITLE}
**Status**: ${STATUS}
**Date**: ${DATE}
**Dependencies**: ${DEPENDENCIES}

## Overview

${OVERVIEW}

## Implementation Details

### Files Modified

${FILES_MODIFIED}

### Key Components

${KEY_COMPONENTS}

## Test Results

${TEST_RESULTS}

## Coverage Report

${COVERAGE_SUMMARY}

## Acceptance Criteria

${ACCEPTANCE_CRITERIA}

## Integration Points

${INTEGRATION_POINTS}

## Next Steps

${NEXT_STEPS}

---

**Implementation by**: ${AGENT}
**Agent ID**: ${AGENT_ID}
**Verified by**: ${VERIFIED_BY}
TEMPLATE_EOF

  # Atomic move
  if mv "$temp_file" "$template_file" 2>/dev/null; then
    echo "[INFO] Template created successfully" >&2
  else
    echo "[WARN] Failed to move template file" >&2
    rm -f "$temp_file" 2>/dev/null
    return 0
  fi

  return 0
}

# =============================================================================
# Test Execution Functions (Phase 2)
# =============================================================================

# Replace placeholders in command strings
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

# Run Go tests
run_go_tests() {
  local audit_dir="$1"
  local test_failed=0

  echo "[INFO] Running Go tests..."

  # Get test commands from config (with defaults)
  local unit_test_cmd
  local integration_test_cmd
  local race_test_cmd
  local coverage_cmd

  unit_test_cmd=$(jq -r '.audit_config.test_commands.go.unit // "go test -v ./..."' "$CONFIG_FILE")
  integration_test_cmd=$(jq -r '.audit_config.test_commands.go.integration // ""' "$CONFIG_FILE")
  race_test_cmd=$(jq -r '.audit_config.test_commands.go.race // "go test -race ./..."' "$CONFIG_FILE")
  coverage_cmd=$(jq -r '.audit_config.test_commands.go.coverage // "go test -coverprofile={audit_dir}/coverage.out ./..."' "$CONFIG_FILE")

  # Replace placeholders
  unit_test_cmd=$(replace_placeholders "$unit_test_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")
  integration_test_cmd=$(replace_placeholders "$integration_test_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")
  race_test_cmd=$(replace_placeholders "$race_test_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")
  coverage_cmd=$(replace_placeholders "$coverage_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")

  # Run unit tests (non-blocking)
  echo "[INFO] Executing unit tests..."
  if eval "$unit_test_cmd" > "$audit_dir/unit-tests.log" 2>&1; then
    echo "[PASS] Unit tests passed"
  else
    echo "[FAIL] Unit tests failed (see $audit_dir/unit-tests.log)"
    test_failed=1
  fi

  # Run integration tests if configured (non-blocking)
  if [[ -n "$integration_test_cmd" && "$integration_test_cmd" != "null" ]]; then
    echo "[INFO] Executing integration tests..."
    if eval "$integration_test_cmd" > "$audit_dir/integration-tests.log" 2>&1; then
      echo "[PASS] Integration tests passed"
    else
      echo "[FAIL] Integration tests failed (see $audit_dir/integration-tests.log)"
      test_failed=1
    fi
  fi

  # Run race detector (non-blocking)
  echo "[INFO] Executing race detector..."
  if eval "$race_test_cmd" > "$audit_dir/race-detector.log" 2>&1; then
    echo "[PASS] Race detector passed"
  else
    echo "[FAIL] Race detector found issues (see $audit_dir/race-detector.log)"
    test_failed=1
  fi

  # Generate coverage report (non-blocking)
  echo "[INFO] Generating coverage report..."
  if eval "$coverage_cmd" > /dev/null 2>&1; then
    echo "[PASS] Coverage data generated"

    # Generate coverage report text
    if go tool cover -func="$audit_dir/coverage.out" > "$audit_dir/coverage-report.txt" 2>&1; then
      echo "[INFO] Coverage report created"

      # Extract total coverage percentage
      local total_coverage
      total_coverage=$(grep "total:" "$audit_dir/coverage-report.txt" | awk '{print $3}')
      echo "$total_coverage" > "$audit_dir/coverage-summary.txt"
      echo "[INFO] Total coverage: $total_coverage"
    else
      echo "[WARN] Failed to generate coverage report"
    fi
  else
    echo "[FAIL] Coverage generation failed"
    test_failed=1
  fi

  return $test_failed
}

# Run Python tests
run_python_tests() {
  local audit_dir="$1"
  local test_failed=0

  echo "[INFO] Running Python tests..."

  # Get test commands from config (with defaults)
  local pytest_cmd
  pytest_cmd=$(jq -r '.audit_config.test_commands.python.pytest // "uv run pytest --cov=src --cov-branch --cov-report=term"' "$CONFIG_FILE")

  # Replace placeholders
  pytest_cmd=$(replace_placeholders "$pytest_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")

  # Run pytest with coverage (non-blocking)
  echo "[INFO] Executing pytest..."
  if eval "$pytest_cmd" > "$audit_dir/pytest-output.log" 2>&1; then
    echo "[PASS] Python tests passed"

    # Extract coverage summary if available
    if grep -q "TOTAL" "$audit_dir/pytest-output.log"; then
      grep "TOTAL" "$audit_dir/pytest-output.log" > "$audit_dir/coverage-summary.txt"
      echo "[INFO] Coverage summary extracted"
    fi
  else
    echo "[FAIL] Python tests failed (see $audit_dir/pytest-output.log)"
    test_failed=1
  fi

  return $test_failed
}

# Run R tests
run_r_tests() {
  local audit_dir="$1"
  local test_failed=0

  echo "[INFO] Running R tests..."

  # Get test commands from config (with defaults)
  local testthat_cmd
  local coverage_cmd

  testthat_cmd=$(jq -r '.audit_config.test_commands.r.testthat // "Rscript -e \"devtools::test()\""' "$CONFIG_FILE")
  coverage_cmd=$(jq -r '.audit_config.test_commands.r.coverage // ""' "$CONFIG_FILE")

  # Replace placeholders
  testthat_cmd=$(replace_placeholders "$testthat_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")
  coverage_cmd=$(replace_placeholders "$coverage_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")

  # Run testthat (non-blocking)
  echo "[INFO] Executing testthat..."
  if eval "$testthat_cmd" > "$audit_dir/testthat-output.log" 2>&1; then
    echo "[PASS] R tests passed"
  else
    echo "[FAIL] R tests failed (see $audit_dir/testthat-output.log)"
    test_failed=1
  fi

  # Run coverage if configured (non-blocking)
  if [[ -n "$coverage_cmd" && "$coverage_cmd" != "null" ]]; then
    echo "[INFO] Generating coverage report..."
    if eval "$coverage_cmd" > "$audit_dir/coverage-report.txt" 2>&1; then
      echo "[PASS] Coverage report generated"
    else
      echo "[WARN] Coverage generation failed"
    fi
  fi

  return $test_failed
}

# Run JavaScript/TypeScript tests
run_js_tests() {
  local audit_dir="$1"
  local test_failed=0

  echo "[INFO] Running JavaScript/TypeScript tests..."

  # Get test commands from config (with defaults)
  local npm_test_cmd
  npm_test_cmd=$(jq -r '.audit_config.test_commands.javascript.test // "npm test"' "$CONFIG_FILE")

  # Replace placeholders
  npm_test_cmd=$(replace_placeholders "$npm_test_cmd" "$audit_dir" "$TICKET_ID" "$PROJECT_ROOT")

  # Run npm test (non-blocking)
  echo "[INFO] Executing npm test..."
  if eval "$npm_test_cmd" > "$audit_dir/npm-test-output.log" 2>&1; then
    echo "[PASS] JavaScript tests passed"

    # Extract coverage summary if available
    if grep -q "Coverage summary" "$audit_dir/npm-test-output.log"; then
      grep -A 10 "Coverage summary" "$audit_dir/npm-test-output.log" > "$audit_dir/coverage-summary.txt"
      echo "[INFO] Coverage summary extracted"
    fi
  else
    echo "[FAIL] JavaScript tests failed (see $audit_dir/npm-test-output.log)"
    test_failed=1
  fi

  return $test_failed
}

# =============================================================================
# Summary Generation Functions (Phase 3)
# =============================================================================

# Extract test results from logs
extract_test_results() {
  local audit_dir="$1"
  local language="$2"
  local results=""

  case "$language" in
    go)
      # Extract unit test results
      if [[ -f "$audit_dir/unit-tests.log" ]]; then
        local pass_count
        local fail_count
        pass_count=$(grep -c "^--- PASS:" "$audit_dir/unit-tests.log" 2>/dev/null || true)
        fail_count=$(grep -c "^--- FAIL:" "$audit_dir/unit-tests.log" 2>/dev/null || true)

        # Handle case where grep -c returns nothing (no matches)
        [[ -z "$pass_count" ]] && pass_count="0"
        [[ -z "$fail_count" ]] && fail_count="0"

        results+="### Unit Tests\n\n"
        results+="- Passed: $pass_count\n"
        results+="- Failed: $fail_count\n\n"

        # Extract test output snippet
        if grep -q "^PASS$" "$audit_dir/unit-tests.log"; then
          results+="\`\`\`\n"
          results+="$(tail -10 "$audit_dir/unit-tests.log")\n"
          results+="\`\`\`\n\n"
        fi
      fi

      # Check integration tests
      if [[ -f "$audit_dir/integration-tests.log" ]]; then
        results+="\n### Integration Tests\n\n"
        if grep -q "^PASS$" "$audit_dir/integration-tests.log"; then
          results+="- Status: ✅ Passed\n"
        else
          results+="- Status: ❌ Failed\n"
        fi
      fi

      # Check race detector
      if [[ -f "$audit_dir/race-detector.log" ]]; then
        if grep -q "WARNING: DATA RACE" "$audit_dir/race-detector.log"; then
          results+="\n### Race Detector\n\n"
          results+="- Status: ⚠️  Race conditions detected\n"
        fi
      fi
      ;;

    python)
      if [[ -f "$audit_dir/pytest-output.log" ]]; then
        results+="### Pytest Results\n\n"

        # Extract pass/fail counts
        if grep -q "passed" "$audit_dir/pytest-output.log"; then
          local pytest_summary
          pytest_summary=$(grep -E "[0-9]+ passed" "$audit_dir/pytest-output.log" | tail -1)
          results+="$pytest_summary\n\n"
        fi

        # Extract coverage if available
        if grep -q "TOTAL" "$audit_dir/pytest-output.log"; then
          results+="\`\`\`\n"
          results+="$(grep "TOTAL" "$audit_dir/pytest-output.log")\n"
          results+="\`\`\`\n"
        fi
      fi
      ;;

    r)
      if [[ -f "$audit_dir/testthat-output.log" ]]; then
        results+="### Testthat Results\n\n"

        # Extract test summary
        if grep -q "OK:" "$audit_dir/testthat-output.log"; then
          local test_summary
          test_summary=$(grep "OK:" "$audit_dir/testthat-output.log" | tail -1)
          results+="$test_summary\n"
        fi
      fi
      ;;

    javascript|typescript)
      if [[ -f "$audit_dir/npm-test-output.log" ]]; then
        results+="### NPM Test Results\n\n"

        # Extract test summary
        if grep -q "Tests:" "$audit_dir/npm-test-output.log"; then
          results+="\`\`\`\n"
          results+="$(grep "Tests:" "$audit_dir/npm-test-output.log")\n"
          results+="\`\`\`\n"
        fi
      fi
      ;;
  esac

  if [[ -z "$results" ]]; then
    results="No test results available"
  fi

  echo -e "$results"
}

# Create minimal summary when template or metadata missing
create_minimal_summary() {
  local audit_dir="$1"
  local ticket_id="$2"
  local reason="$3"

  cat > "$audit_dir/implementation-summary.md" <<EOF
# $ticket_id - Implementation Summary (Minimal)

**Ticket ID**: $ticket_id
**Status**: Audit completed
**Date**: $(date -I)
**Note**: $reason

## Audit Results

Audit execution completed successfully. See audit logs for details:
- \`$audit_dir/unit-tests.log\`
- \`$audit_dir/coverage-report.txt\`
- \`$audit_dir/coverage-summary.txt\`

For full implementation summary, ensure:
1. Template exists at: \`~/.claude/skills/ticket/templates/implementation-summary.md.tmpl\`
2. Ticket metadata exists in: \`tickets-index.json\`

---

**Generated**: $(date -Iseconds)
EOF

  echo "[INFO] Created minimal summary: $audit_dir/implementation-summary.md"
}

# Generate implementation summary
generate_implementation_summary() {
  local audit_dir="$1"
  local ticket_id="$2"
  local language="$3"

  echo "[INFO] Phase 3: Generating implementation summary..."

  # Template path
  local template_path="$HOME/.claude/skills/ticket/templates/implementation-summary.md.tmpl"

  # Check if template exists
  if [[ ! -f "$template_path" ]]; then
    create_minimal_summary "$audit_dir" "$ticket_id" "Template not found: $template_path"
    return 0
  fi

  # Find tickets-index.json (search from project root upward)
  local tickets_index=""
  local search_dir="$PROJECT_ROOT"

  while [[ "$search_dir" != "/" ]]; do
    if [[ -f "$search_dir/tickets-index.json" ]]; then
      tickets_index="$search_dir/tickets-index.json"
      break
    elif [[ -f "$search_dir/migration_plan/tickets/tickets-index.json" ]]; then
      tickets_index="$search_dir/migration_plan/tickets/tickets-index.json"
      break
    fi
    search_dir=$(dirname "$search_dir")
  done

  if [[ -z "$tickets_index" || ! -f "$tickets_index" ]]; then
    create_minimal_summary "$audit_dir" "$ticket_id" "tickets-index.json not found"
    return 0
  fi

  # Extract metadata from tickets-index.json
  local ticket_title
  local ticket_description
  local ticket_dependencies
  local ticket_status

  ticket_title=$(jq -r ".tickets[] | select(.id == \"$ticket_id\") | .title // \"Unknown\"" "$tickets_index")
  ticket_description=$(jq -r ".tickets[] | select(.id == \"$ticket_id\") | .description // \"No description\"" "$tickets_index")
  ticket_status=$(jq -r ".tickets[] | select(.id == \"$ticket_id\") | .status // \"in-progress\"" "$tickets_index")

  # Extract dependencies array as comma-separated string
  ticket_dependencies=$(jq -r ".tickets[] | select(.id == \"$ticket_id\") | .dependencies // [] | join(\", \")?" "$tickets_index")
  if [[ -z "$ticket_dependencies" || "$ticket_dependencies" == "null" ]]; then
    ticket_dependencies="None"
  fi

  # Fallback if ticket not found
  if [[ -z "$ticket_title" || "$ticket_title" == "Unknown" || "$ticket_title" == "null" ]]; then
    create_minimal_summary "$audit_dir" "$ticket_id" "Ticket metadata not found in tickets-index.json"
    return 0
  fi

  # Extract test results
  local test_results
  test_results=$(extract_test_results "$audit_dir" "$language")

  # Load coverage summary
  local coverage_summary="No coverage data available"
  if [[ -f "$audit_dir/coverage-summary.txt" ]]; then
    coverage_summary=$(cat "$audit_dir/coverage-summary.txt")
  fi

  # Set environment variables for envsubst
  export TICKET_ID="$ticket_id"
  export TICKET_TITLE="$ticket_title"
  export STATUS="✅ Completed"
  export DATE="$(date -I)"
  export DEPENDENCIES="$ticket_dependencies"
  export OVERVIEW="$ticket_description"
  export FILES_MODIFIED="See audit logs for details"
  export KEY_COMPONENTS="See audit logs for details"
  export TEST_RESULTS="$test_results"
  export COVERAGE_SUMMARY="$coverage_summary"
  export ACCEPTANCE_CRITERIA="See ticket for full criteria"
  export INTEGRATION_POINTS="See ticket for integration points"
  export NEXT_STEPS="Review audit results and proceed to next ticket"
  export AGENT="run-audit.sh"
  export AGENT_ID="automated"
  export VERIFIED_BY="Audit system"

  # Render template
  if envsubst < "$template_path" > "$audit_dir/implementation-summary.md"; then
    echo "[INFO] Implementation summary generated: $audit_dir/implementation-summary.md"
  else
    echo "[WARN] Failed to render template, creating minimal summary"
    create_minimal_summary "$audit_dir" "$ticket_id" "Template rendering failed"
  fi

  # Clean up environment variables
  unset TICKET_ID TICKET_TITLE STATUS DATE DEPENDENCIES OVERVIEW FILES_MODIFIED
  unset KEY_COMPONENTS TEST_RESULTS COVERAGE_SUMMARY ACCEPTANCE_CRITERIA
  unset INTEGRATION_POINTS NEXT_STEPS AGENT AGENT_ID VERIFIED_BY
}

# =============================================================================
# Main Execution
# =============================================================================
main() {
  parse_args "$@"

  echo "[INFO] Starting audit for ticket: $TICKET_ID"

  # Load and validate config
  load_config

  # Bootstrap templates (self-healing)
  bootstrap_templates

  # Detect project language
  local language
  language=$(detect_language)

  if [[ "$language" == "unknown" ]]; then
    echo "Error: Could not detect project language" >&2
    echo "Checked for: go.mod, pyproject.toml, setup.py, DESCRIPTION, *.Rproj, package.json, tsconfig.json" >&2
    exit 2
  fi

  echo "[INFO] Detected language: $language"

  # Create audit directory
  local audit_dir
  audit_dir=$(create_audit_directory)

  echo "[INFO] Phase 1 (Core Infrastructure) complete"
  echo "[INFO] Audit directory: $audit_dir"

  # Phase 2: Execute tests
  local test_status=0
  echo "[INFO] Phase 2: Executing tests..."

  case "$language" in
    go)
      run_go_tests "$audit_dir" || test_status=$?
      ;;
    python)
      run_python_tests "$audit_dir" || test_status=$?
      ;;
    r)
      run_r_tests "$audit_dir" || test_status=$?
      ;;
    javascript|typescript)
      run_js_tests "$audit_dir" || test_status=$?
      ;;
    *)
      echo "[WARN] No test runner configured for language: $language"
      ;;
  esac

  if [[ $test_status -eq 0 ]]; then
    echo "[INFO] Phase 2 (Test Execution) complete - All tests passed"
  else
    echo "[WARN] Phase 2 (Test Execution) complete - Some tests failed (non-blocking)"
  fi

  # Phase 3: Generate implementation summary
  generate_implementation_summary "$audit_dir" "$TICKET_ID" "$language"

  echo "[INFO] Audit complete. Results in: $audit_dir"

  exit 0
}

# Execute main with all arguments
main "$@"
