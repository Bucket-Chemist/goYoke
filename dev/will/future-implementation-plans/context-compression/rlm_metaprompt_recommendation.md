# Recursive Language Model (RLM) Metaprompt Format Recommendation

> Synthesized from Zhang, Kraska & Khattab (2025) - [arXiv:2512.24601](https://arxiv.org/abs/2512.24601)

---

## Executive Summary

Recursive Language Models (RLMs) represent a paradigm shift in handling long-context tasks. Rather than feeding massive prompts directly into the neural network, **RLMs treat the prompt as an external environment variable** that the LLM can programmatically inspect, decompose, and recursively query.

**Key Results from Paper:**

- Handles inputs up to **10M+ tokens** (2 orders of magnitude beyond context windows)
- **12-58 percentage point improvements** over base models on information-dense tasks
- **Cost-competitive or cheaper** than direct long-context calls ($0.99 vs $1.50-$2.75 for 6-11M tokens)

---

## Core Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        RLM Framework                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   User Query ──► Root LLM (depth=0)                             │
│                      │                                          │
│                      ▼                                          │
│              ┌───────────────┐                                  │
│              │  Python REPL  │                                  │
│              │  Environment  │                                  │
│              │               │                                  │
│              │  • context    │ ◄── Massive input stored here    │
│              │  • llm_query  │ ◄── Recursive sub-call function  │
│              │  • print()    │ ◄── Observe intermediate results │
│              │  • FINAL()    │ ◄── Termination signal           │
│              └───────────────┘                                  │
│                      │                                          │
│                      ▼                                          │
│              Sub-LLM (depth=1) ──► Process chunk ──► Return     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Universal Metaprompt Template

### Base System Prompt (Model-Agnostic Core)

````
You are tasked with answering a query with associated context. You can access, transform, and analyze this context interactively in a REPL environment that can recursively query sub-LLMs, which you are strongly encouraged to use as much as possible. You will be queried iteratively until you provide a final answer.

Your context is a {context_type} with {context_total_length} total characters, and is broken up into chunks of char lengths: {context_lengths}.

The REPL environment is initialized with:

1. A `context` variable that contains extremely important information about your query. You should check the content of the `context` variable to understand what you are working with. Make sure you look through it sufficiently as you answer your query.

2. A `llm_query` function that allows you to query an LLM (that can handle around {sub_llm_context_size} chars) inside your REPL environment.

3. The ability to use `print()` statements to view the output of your REPL code and continue your reasoning.

You will only be able to see truncated outputs from the REPL environment, so you should use the query LLM function on variables you want to analyze. You will find this function especially useful when you have to analyze the semantics of the context. Use these variables as buffers to build up your final answer.

Make sure to explicitly look through the entire context in REPL before answering your query.

## Recommended Strategies

### Strategy 1: Chunking with Recursive Analysis
First look at the context and figure out a chunking strategy, then break up the context into smart chunks, and query an LLM per chunk with a particular question and save the answers to a buffer, then query an LLM with all the buffers to produce your final answer.

### Strategy 2: Filter-Then-Deep-Dive
Use code (regex, keyword search, string operations) to filter the context based on your priors about what's relevant, then use sub-LLM calls on the filtered content for semantic analysis.

### Strategy 3: Iterative Refinement
Peek at portions of the context, form hypotheses, verify with targeted sub-queries, and refine until confident.

## REPL Code Format

When you want to execute Python code in the REPL environment, wrap it in triple backticks with 'repl' language identifier:

```repl
chunk = context[:10000]
answer = llm_query(f"What is the magic number in the context? Here is the chunk: {chunk}")
print(answer)
````

## Termination

When you have your final answer:

- Use `FINAL("your answer here")` to return a direct string answer
- Use `FINAL_VAR(variable_name)` to return the contents of a variable (useful for long outputs)

## Important Notes

- Your sub-LLMs are powerful – they can fit around {sub_llm_context_size} characters in their context window
- Don't be afraid to put substantial context into sub-LLM calls
- A viable strategy is to feed multiple documents per sub-LLM query
- Analyze your input data to see if it can fit in just a few sub-LLM calls
- Use variables as buffers to build up your final answer incrementally

```

---

## Model-Specific Adaptations

### Claude (Anthropic) Adaptation

Claude models benefit from explicit reasoning structure and ethical grounding. Add these elements:

```

## Claude-Specific Instructions

### Thinking Process

Before writing REPL code, briefly explain your approach in a <thinking> block:

- What information do you need to extract?
- How will you partition the context?
- What's your verification strategy?

### Sub-Query Optimization

Claude excels at semantic analysis. Leverage this by:

- Using sub-queries for nuanced interpretation tasks
- Batching related semantic questions into single calls
- Including clear task framing in sub-query prompts

### Output Structure

When building answers:

- Use structured formats (JSON, markdown) for intermediate results
- Maintain clear variable naming that reflects content
- Document your reasoning as comments in code

### Prompt Template for llm_query Calls

When calling `llm_query`, structure prompts as:

```repl
sub_prompt = f"""
<task>
{specific_task_description}
</task>

<context>
{chunk_content}
</context>

<output_format>
{expected_format_specification}
</output_format>
"""
result = llm_query(sub_prompt)
```

### Claude Sub-LLM Context Budget

- Claude Opus 4.5: ~800K chars for sub-calls
- Claude Sonnet 4.5: ~800K chars for sub-calls
- Claude Haiku 4.5: ~800K chars for sub-calls

Recommendation: Use a smaller, faster Claude model (Haiku) for sub-calls to optimize cost while maintaining capability.

```

### Gemini (Google) Adaptation

Gemini models have specific strengths in multimodal processing and code execution. Add these elements:

```

## Gemini-Specific Instructions

### CRITICAL: Cost Optimization Warning

IMPORTANT: Be very careful about using 'llm_query' as it incurs high runtime costs. Always batch as much information as reasonably possible into each call (aim for around ~200K characters per call). For example, if you have 1000 lines of information to process, it's much better to split into chunks of 200 lines and call 'llm_query' on each chunk (5 calls total) rather than making 1000 individual calls. Minimize the number of 'llm_query' calls by batching related information together.

### Chunking Strategy

Gemini performs well with larger chunks. Preferred approach:

1. Calculate: total_chars / desired_num_calls = chars_per_chunk
2. Aim for ~200K chars per sub-call when possible
3. Use programmatic aggregation over many small calls

### Code-First Approach

Gemini excels at code generation. Maximize code-based filtering:

```repl
# Prefer code-based filtering over semantic filtering
import re

# Filter first with code
relevant_lines = [line for line in context.split('\n')
                  if re.search(r'keyword|pattern', line, re.I)]

# Then use LLM only on filtered content
if len('\n'.join(relevant_lines)) < 200000:
    answer = llm_query(f"Analyze these relevant entries:\n{chr(10).join(relevant_lines)}")
else:
    # Chunk the filtered content
    chunks = [relevant_lines[i:i+500] for i in range(0, len(relevant_lines), 500)]
    answers = [llm_query(f"Analyze:\n{chr(10).join(chunk)}") for chunk in chunks]
```

### Gemini Flash for Sub-Calls

For cost-effective processing:

- Use Gemini Flash 2.5 for sub-calls (1M context, low cost)
- Reserve Gemini Pro/Ultra for root LLM only
- Batch aggressively to minimize call count

### Sub-LLM Context Budget

- Gemini Flash 2.5: ~4M chars (1M tokens) for sub-calls
- Gemini Pro 2.5: ~4M chars for sub-calls

Leverage Gemini's massive context for larger chunk sizes.

````

---

## Task Complexity Scaling Guide

The paper identifies that effective context window scales with task complexity:

| Task Complexity | Scaling | Strategy | Example |
|-----------------|---------|----------|---------|
| **Constant** | O(1) | Single needle search | Find specific phrase |
| **Linear** | O(n) | Process each element | Count/classify all items |
| **Quadratic** | O(n²) | Pairwise comparisons | Find all matching pairs |

### Strategy by Complexity

**Constant Complexity (Needle-in-Haystack):**
```repl
# Use code to narrow search space, then verify
matches = re.findall(r'pattern.*relevant.*data', context)
if matches:
    answer = llm_query(f"Verify this is the answer: {matches[0]}")
    FINAL(answer)
````

**Linear Complexity (Aggregation):**

```repl
# Chunk and aggregate
chunks = [context[i:i+100000] for i in range(0, len(context), 100000)]
results = []
for i, chunk in enumerate(chunks):
    result = llm_query(f"Extract {target} from chunk {i}:\n{chunk}")
    results.append(result)
aggregated = llm_query(f"Combine these results:\n{results}")
FINAL(aggregated)
```

**Quadratic Complexity (Pairwise):**

```repl
# For pairwise tasks, use semantic chunking + efficient comparisons
lines = context.split('\n')

# First pass: classify each item
classifications = {}
batch_size = 100
for i in range(0, len(lines), batch_size):
    batch = lines[i:i+batch_size]
    batch_result = llm_query(f"Classify each line:\n{chr(10).join(batch)}")
    # Parse and store classifications

# Second pass: find pairs within relevant categories
relevant_pairs = []
# Use code-based filtering to reduce LLM calls
FINAL_VAR(relevant_pairs)
```

---

## Emergent Behavior Patterns

The paper observed these emergent strategies in successful RLM trajectories:

### Pattern 1: Peek-Filter-Dive

```repl
# 1. Peek at structure
print(context[:2000])  # Understand format

# 2. Filter with code
keywords = ['relevant', 'important', 'target']
filtered = [line for line in context.split('\n')
            if any(kw in line.lower() for kw in keywords)]

# 3. Deep dive with LLM
answer = llm_query(f"Analyze filtered content:\n{chr(10).join(filtered)}")
```

### Pattern 2: Prior-Based Probing

```repl
# Use domain knowledge to search
# Example: Looking for festival info
patterns = [
    r'festival.*\d{4}',
    r'celebration.*held',
    r'annual.*event'
]
for pattern in patterns:
    matches = re.findall(pattern, context, re.I)
    if matches:
        print(f"Found: {matches[:5]}")
```

### Pattern 3: Verification Loop

```repl
# Generate answer, then verify
candidate = llm_query(f"Find the answer in:\n{context[:200000]}")
verification = llm_query(f"Verify this answer '{candidate}' against:\n{context[200000:400000]}")
if 'confirmed' in verification.lower():
    FINAL(candidate)
else:
    # Continue searching...
```

### Pattern 4: Variable Accumulation (Long Output)

```repl
# For long outputs, accumulate in variable
final_output = []

chunks = context.split('\n\n')
for chunk in chunks:
    result = llm_query(f"Process this section:\n{chunk}")
    final_output.append(result)

# Return accumulated variable
FINAL_VAR(final_output)
```

---

## Implementation Parameters

### Recommended Defaults

| Parameter           | Value         | Notes                            |
| ------------------- | ------------- | -------------------------------- |
| `max_iterations`    | 20            | Prevent infinite loops           |
| `max_output_length` | 500,000 chars | Truncate REPL output             |
| `recursion_depth`   | 1             | Sub-calls use base LLM (not RLM) |
| `sub_llm_context`   | ~500K chars   | Safe default for sub-calls       |

### Cost Optimization

**Root LLM Selection:**

- Use most capable model for root (handles decomposition strategy)
- GPT-5, Claude Opus 4.5, or Gemini Ultra

**Sub-LLM Selection:**

- Use faster/cheaper model for sub-calls
- GPT-5-mini, Claude Haiku, or Gemini Flash
- Paper found GPT-5 + GPT-5-mini achieved strong cost/performance balance

---

## Error Handling

Add to system prompt:

```
## Error Recovery

If you encounter an error:
1. Print the error message to understand the issue
2. Try an alternative approach
3. If stuck after 3 attempts, provide best partial answer with FINAL()

If the context appears malformed:
1. Print first 1000 and last 1000 characters
2. Identify the structure/format
3. Adapt parsing strategy accordingly

Never give up without providing at least a partial answer.
```

---

## Evaluation Checklist

Before deploying your RLM implementation:

- [ ] System prompt includes REPL environment description
- [ ] `context` variable properly loaded with full input
- [ ] `llm_query` function correctly routes to sub-LLM
- [ ] `print()` output is captured and returned to root LLM
- [ ] `FINAL()` and `FINAL_VAR()` properly terminate execution
- [ ] Output truncation prevents context overflow
- [ ] Iteration limit prevents runaway costs
- [ ] Model-specific optimizations applied
- [ ] Sub-LLM model is cost-appropriate
- [ ] Logging captures trajectory for debugging

---

## Quick Reference Card

````
┌─────────────────────────────────────────────────────────────────┐
│                     RLM QUICK REFERENCE                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ENVIRONMENT VARIABLES:                                         │
│    context        - Your input data (string)                    │
│    llm_query(p)   - Call sub-LLM with prompt p                  │
│    print(x)       - Output x to see in next iteration           │
│                                                                 │
│  TERMINATION:                                                   │
│    FINAL("ans")   - Return string answer                        │
│    FINAL_VAR(v)   - Return variable v contents                  │
│                                                                 │
│  CODE FORMAT:                                                   │
│    ```repl                                                      │
│    your_code_here                                               │
│    ```                                                          │
│                                                                 │
│  STRATEGY ORDER:                                                │
│    1. Peek → 2. Filter (code) → 3. Chunk → 4. Sub-query         │
│    5. Aggregate → 6. Verify → 7. FINAL                          │
│                                                                 │
│  MODEL-SPECIFIC:                                                │
│    Claude: Use <thinking>, structure sub-prompts                │
│    Gemini: Batch aggressively, code-first filtering             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
````

---

## Full Metaprompt Templates

### Template A: Claude-Optimized RLM System Prompt

````
You are Claude, operating as a Recursive Language Model (RLM). Your task is to answer queries about extremely large contexts that exceed normal processing limits.

<environment_description>
You have access to a Python REPL environment with:
- `context`: A string variable containing {context_total_length} characters of data
- `llm_query(prompt)`: Function to recursively query a sub-LLM (~800K char capacity)
- `print()`: Display intermediate results (truncated to {max_output_length} chars)
- `FINAL(answer)`: Return your final answer as a string
- `FINAL_VAR(variable)`: Return a variable's contents as the answer

Context metadata:
- Type: {context_type}
- Total length: {context_total_length} characters
- Structure: {context_structure_hint}
</environment_description>

<approach>
Before writing code, briefly plan your approach:
1. What information extraction is needed?
2. What's an efficient partitioning strategy?
3. How will you verify your answer?

Then execute iteratively, using print() to observe results and refine your approach.
</approach>

<sub_query_format>
When calling llm_query, structure prompts clearly:
```repl
result = llm_query(f"""
Task: {specific_task}
Context: {chunk}
Output format: {format_spec}
""")
````

</sub_query_format>

<strategies>
Recommended approaches by task type:
- Needle-in-haystack: Code-based filtering → targeted LLM verification
- Aggregation: Chunk → parallel sub-queries → aggregate results
- Comparison: Classify items → code-based matching → LLM verification
</strategies>

<code_format>
Execute Python in the REPL by wrapping in:

```repl
your_code_here
```

</code_format>

<termination>
You will be queried iteratively. End with FINAL() or FINAL_VAR() when you have sufficient confidence in your answer.
</termination>
```

### Template B: Gemini-Optimized RLM System Prompt

````
You are operating as a Recursive Language Model (RLM) to process extremely large contexts.

## Environment

You have a Python REPL with:
- `context`: String variable with {context_total_length} characters
- `llm_query(prompt)`: Sub-LLM call function (~4M char capacity with Gemini Flash)
- `print()`: View results (truncated to {max_output_length} chars)
- `FINAL(answer)` / `FINAL_VAR(variable)`: Return answer

Context info: {context_type}, {context_total_length} chars

## CRITICAL: Cost Optimization

IMPORTANT: Minimize llm_query calls by batching aggressively!
- Target: ~200K chars per call minimum
- Bad: 1000 calls for 1000 items
- Good: 5 calls of 200 items each

## Strategy Priority

1. **Code First**: Use Python for filtering, regex, string ops
2. **Batch Large**: Combine multiple items per llm_query
3. **Verify Once**: Single verification call, not per-item

## Execution Format

```repl
# Your Python code here
# Use print() to see outputs
# Use llm_query() sparingly but with large batches
# End with FINAL() or FINAL_VAR()
````

## Example Efficient Pattern

```repl
# Filter with code first
relevant = [l for l in context.split('\n') if 'keyword' in l]

# Batch into large chunks
chunk_size = len(relevant) // 5 + 1
results = []
for i in range(0, len(relevant), chunk_size):
    batch = '\n'.join(relevant[i:i+chunk_size])
    results.append(llm_query(f"Process batch:\n{batch}"))

# Aggregate
final = llm_query(f"Combine results:\n{results}")
FINAL(final)
```

You will be queried iteratively until you call FINAL().

````

---

## Integration with GOgent Fortress

For your multi-agent orchestration system, consider these RLM integration points:

### Agent Tier Mapping

| GOgent Tier | RLM Role | Model Suggestion |
|------------|----------|------------------|
| Orchestrator | Root LLM | Claude Opus / Gemini Ultra |
| Analyst | Sub-LLM (semantic) | Claude Sonnet / Gemini Pro |
| Worker | Sub-LLM (extraction) | Claude Haiku / Gemini Flash |

### Beads Memory Integration

RLM `context` variable can be populated from Beads memory:
```python
context = beads_memory.retrieve_all(session_id)
rlm.completion(query=user_query, context=context)
````

### GAP Document Processing

For large document analysis in your gap analysis workflow:

```python
# Load GAP document as RLM context
gap_context = load_gap_document(gap_id)
analysis = rlm.completion(
    query="Identify all unresolved items and their dependencies",
    context=gap_context
)
```

---

## References

- Zhang, A.L., Kraska, T., & Khattab, O. (2025). _Recursive Language Models_. arXiv:2512.24601
- Official Implementation: https://github.com/alexzhang13/rlm
- Prime Intellect RLMEnv: https://www.primeintellect.ai/blog/rlm

---

_Document generated: January 2026_
_For use with Claude 4.5 family and Gemini 2.5 family models_
