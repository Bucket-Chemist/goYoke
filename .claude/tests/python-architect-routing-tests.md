# Python-Architect Routing Tests

Test cases to validate the new routing and escalation paths.

---

## Test 1: Direct python-architect Invocation

**Prompt:**
```
Should I use linear attention or full attention for cross-scale fusion in a model with 2450 sequence length?
```

**Expected Behavior:**
1. Routes to `python-architect` (opus tier)
2. Spawns `codebase-search` to find existing attention patterns
3. References `python-ml.md` attention decision matrix

**Expected Output:**
- `.claude/tmp/architecture-decision.md` with:
  - Options: Full attention, Local attention, Linear (Linformer), Performer
  - Decision: Linear attention (Linformer) with k=256
  - Rationale: O(n²) too expensive for n=2450
- `.claude/tmp/architecture-metadata.json` with confidence score

**Validation:**
```bash
# Check output files exist
ls -la .claude/tmp/architecture-decision.md
ls -la .claude/tmp/architecture-metadata.json

# Verify JSON is valid
jq . .claude/tmp/architecture-metadata.json
```

---

## Test 2: python-pro Escalation Path

**Setup:**
Trigger python-pro with a task requiring architecture decision.

**Prompt:**
```
Implement the adaptive deconvolution head in src/models/deconvolution.py
```

**Expected Behavior:**
1. `python-pro` activates (auto-activate for models/*.py)
2. Loads `python.md` + `python-ml.md` conventions
3. Recognizes multiple approaches (classification vs ordinal, independent vs iterative)
4. Spawns `python-architect` for decision
5. Reads decision, implements accordingly

**Validation Criteria:**
- python-pro spawns python-architect (check agent logs)
- architecture-decision.md created before implementation
- Final implementation follows decision guidance

---

## Test 3: Convention Loading - Data Science

**Prompt:**
```
Implement the MSPreprocessor class in src/data/preprocessing.py with Anscombe transform and MAD noise estimation
```

**Expected Behavior:**
1. `python-pro` activates
2. Loads `python.md` + `python-datasci.md` (based on path pattern `**/data/**`)
3. Uses Anscombe transform pattern from python-datasci.md
4. Uses MAD noise estimation pattern from python-datasci.md

**Validation:**
- Check implementation uses `2.0 * np.sqrt(x + 3/8)` for Anscombe
- Check implementation uses `median(|x - median(x)|) * 1.4826` for MAD
- No escalation needed (patterns are clear in conventions)

---

## Test 4: Convention Loading - ML

**Prompt:**
```
Implement the EdgeAwareEncoder in src/models/encoder.py following the existing patterns
```

**Expected Behavior:**
1. `python-pro` activates
2. Loads `python.md` + `python-ml.md` (based on path pattern `**/models/**`)
3. Uses GroupNorm (not BatchNorm)
4. Uses GELU activation
5. Includes tensor shape comments

**Validation:**
- Check `nn.GroupNorm` in output
- Check `nn.GELU()` in output
- Check shape comments like `# [batch, channels, seq_len]`

---

## Test 5: Cross-Domain Task

**Prompt:**
```
Implement differentiable preprocessing as a PyTorch nn.Module that includes Gaussian binning and Anscombe transform
```

**Expected Behavior:**
1. `python-pro` activates
2. Loads ALL conventions: `python.md` + `python-datasci.md` + `python-ml.md`
3. Combines:
   - Preprocessing patterns from python-datasci.md
   - nn.Module patterns from python-ml.md
4. May escalate if design choices are ambiguous

**Validation:**
- Output inherits from `nn.Module`
- Uses preprocessing patterns (Anscombe, Gaussian binning)
- Includes proper forward() signature with type hints

---

## Test 6: Architecture Decision with Scout

**Prompt:**
```
Design the loss function for isotope detection in our MS1 pipeline
```

**Expected Behavior:**
1. Routes to `python-architect`
2. Spawns `codebase-search` to find:
   - Existing loss functions in codebase
   - Class distribution for imbalance check
3. Spawns `librarian` if external best practices needed
4. Outputs decision with:
   - Focal loss recommendation (if imbalanced)
   - Permutation invariance consideration (if set prediction)

**Validation:**
- Scout spawned (check agent logs)
- Decision references existing code patterns
- Decision addresses class imbalance explicitly

---

## Test 7: Escalation to Einstein

**Setup:**
Simulate repeated failure scenario.

**Prompt (to python-architect):**
```
I've tried 3 attention mechanisms for cross-scale fusion:
1. Full attention: Works but too slow (150ms)
2. Linear attention: Fast but loses fine-grained patterns
3. Local attention: Misses cross-scale context

None meet both <50ms latency AND >90% pattern detection. What should I do?
```

**Expected Behavior:**
1. `python-architect` attempts analysis
2. Uses `AskUserQuestion` for clarification (max 2x)
3. If still blocked: generates GAP document
4. Outputs: `.claude/tmp/einstein-gap-{timestamp}.md`
5. Informs user to run `/einstein`

**Validation:**
- GAP document created with:
  - Primary question
  - What was tried (3 options)
  - Blocking ambiguity
  - Relevant context

---

## Test 8: Convention Conflict Detection

**Prompt:**
```
Implement a BatchNorm layer for variable batch sizes in the encoder
```

**Expected Behavior:**
1. `python-pro` activates
2. Reads `python-ml.md` which says "Use GroupNorm for variable batch"
3. Either:
   - Implements GroupNorm instead and explains
   - Escalates to python-architect for clarification

**Validation:**
- Agent does NOT blindly implement BatchNorm
- Either corrects to GroupNorm or escalates
- References convention conflict in response

---

## Execution Script

```bash
#!/bin/bash
# Run all routing tests

TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Direct invocation
echo "Test 1: Direct python-architect invocation"
# [Run test, check outputs]

# Test 2: Escalation path
echo "Test 2: python-pro escalation"
# [Run test, verify escalation occurred]

# ... continue for all tests

echo "Results: $TESTS_PASSED passed, $TESTS_FAILED failed"
```

---

## Success Criteria

All tests pass when:
1. Correct agent receives each task
2. Correct conventions are loaded based on file paths
3. Output format matches templates
4. Escalation triggers appropriately (not too often, not never)
5. Decision documents are well-formed
6. GAP documents generated when escalation to einstein needed

---

## Regression Checks

After any changes to routing, re-run:
- Test 1 (basic routing)
- Test 2 (escalation path)
- Test 3-4 (convention loading)

These are the critical path tests.
