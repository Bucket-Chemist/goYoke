# run-audit.sh Test Suite

Comprehensive test suite for the ticket audit automation system.

## Running Tests

### Full Test Suite

```bash
~/.claude/skills/ticket/scripts/test-run-audit.sh
```

### Test Coverage

**Total: 24 tests**

- **Phase 1: Core Infrastructure (13 tests)**
  - Language detection (Go, Python, R, TypeScript, JavaScript)
  - Language priority handling
  - Unknown language detection
  - Config loading and validation
  - Argument parsing
  - Directory creation

- **Phase 2: Test Execution (4 tests)**
  - Placeholder replacement in commands
  - Go test execution
  - Non-blocking test failures
  - Custom test commands

- **Phase 3: Summary Generation (4 tests)**
  - Test result extraction
  - Minimal summary fallback
  - Template rendering with metadata
  - Metadata missing fallback

- **Integration Tests (3 tests)**
  - Full workflow with Go project
  - Backward compatibility (no config)
  - Backward compatibility (disabled audit)

## Test Results

Expected output:

```
=========================================
run-audit.sh Phase 6 Comprehensive Tests
=========================================

PHASE 1: Core Infrastructure Tests
  ✓ Language detection: Go
  ✓ Language detection: Python (pyproject.toml)
  ✓ Language detection: Python (setup.py)
  ✓ Language detection: R (DESCRIPTION)
  ✓ Language detection: TypeScript
  ✓ Language detection: JavaScript
  ✓ Language detection: Priority
  ✓ Language detection: Unknown
  ✓ Config missing
  ✓ Config disabled
  ✓ Config invalid JSON
  ✓ Missing --ticket-id argument
  ✓ Audit directory creation

PHASE 2: Test Execution Tests
  ✓ Placeholder replacement
  ✓ Go test execution
  ✓ Test failures non-blocking
  ✓ Custom test commands

PHASE 3: Summary Generation Tests
  ✓ Extract Go test results
  ✓ Minimal summary generation
  ✓ Template rendering
  ✓ Metadata missing fallback

INTEGRATION TESTS
  ✓ Full workflow
  ✓ Backward compat (no config)
  ✓ Backward compat (disabled)

=========================================
Test Summary
=========================================
PASSED: 24
FAILED: 0

✅ All tests passed!
```

## Test Isolation

Each test runs in an isolated temporary directory created with `mktemp -d`. Tests clean up after themselves using a trap handler.

## Edge Cases Tested

1. **Missing config file** - Graceful skip with exit 0 (backward compatible)
2. **Disabled audit** - Graceful skip with exit 0
3. **Invalid JSON config** - Error with exit 1
4. **Unknown language** - Error with exit 2
5. **Missing ticket metadata** - Falls back to minimal summary
6. **Missing template** - Creates minimal summary
7. **Test failures** - Non-blocking, logs failure, continues execution

## Adding New Tests

To add a new test:

1. Create test function following naming convention:
   ```bash
   test_description_of_what_it_tests() {
     test_start "Description shown to user"

     local original_dir="$PWD"
     TEMP_DIR=$(mktemp -d)
     cd "$TEMP_DIR"

     # Test setup
     # ...

     # Run test
     # ...

     # Assert results
     if [[ condition ]]; then
       test_pass "Description of pass"
     else
       test_fail "Description of failure"
     fi

     cd "$original_dir"
   }
   ```

2. Add test to appropriate section in `main()`:
   ```bash
   section_header "PHASE X: Category Name"
   test_description_of_what_it_tests
   ```

3. Update test count in summary section

## Test Framework

The test suite uses a simple bash-based framework with:

- **test_start()** - Announces test beginning
- **test_pass()** - Records test pass
- **test_fail()** - Records test failure with optional debug output
- **section_header()** - Organizes tests into phases
- **cleanup_temp()** - Cleans up temporary directories

## CI/CD Integration

This test suite can be integrated into CI/CD pipelines:

```bash
# Run tests with timeout
timeout 120 ~/.claude/skills/ticket/scripts/test-run-audit.sh

# Check exit code
if [ $? -eq 0 ]; then
  echo "Tests passed"
else
  echo "Tests failed"
  exit 1
fi
```

## Debugging Failed Tests

When a test fails, debug output is printed to stderr showing:

- The failing condition
- Actual vs expected values
- Full output from run-audit.sh (when relevant)

To debug a specific test:

1. Run the test suite
2. Note which test failed
3. Run that test function in isolation:
   ```bash
   source ~/.claude/skills/ticket/scripts/test-run-audit.sh
   test_specific_function_name
   ```

## Performance

Test suite execution time: ~15-30 seconds for all 24 tests.

Individual test performance:
- Core infrastructure tests: <1s each
- Test execution tests: 1-2s each
- Summary generation tests: <1s each
- Integration tests: 2-5s each
