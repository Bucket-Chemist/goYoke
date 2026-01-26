# Part V: Implementation Roadmap 

## **12-Month Development Plan for SelfEvolving Multi-Agent Architecture** 

## **Roadmap Overview** 

┌──────────────────────────────────────────────────────────────────── 

│                            12-MONTH IMPLEMENTATION ROADMAP │ └──────────────────────────────────────────────────────────────────── 

FOUNDATION                    CAPABILITY EVOLUTION ───────────────────────────────────────────────────────────────────── 

Weeks 1-2     Month 1       Month 2       Month 3       Months 4-5 Months 6-7 

┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌───────────┐ ┌───────────┐ │ PHASE 0 │  │ PHASE 1 │  │ PHASE 2 │  │ PHASE 3 │  │  PHASE 4  │  │ PHASE 5  │ │         │  │         │  │         │  │         │  │           │  │ │ │ Founda- │→│ Observ- │→│ Memory  │→│ Weekly  │→│ Emergent  │→│ Subagent  │ │ tion    │  │ ability │  │ Enhance │  │ Review  │  │ Schema    │  │ Spawning  │ │         │  │         │  │         │  │         │  │           │  │ │ │ • Telem │  │ • Logs  │  │ • BM25  │  │ • Synth │  │ • Observe │  │ • Gap     │ │ • Valid │  │ • Costs │  │ • Capt  │  │ • Arch  │  │ • Detect  │  │ • Generate│ │ • Schema│  │ • Hooks │  │ • Front │  │ • HITL  │  │ • Propose │  │ • Shadow  │ └─────────┘  └─────────┘  └─────────┘  └─────────┘  └───────────┘ └───────────┘ Months 8-10 Months 11-12 ┌───────────┐ ┌───────────┐ │  PHASE 6  │ │  PHASE 7  │ │           │ │           │ │ Autonomy  │→│ Self-     │ │ Progress  │ │ Improving │ │           │ │           │ │ • Track   │ │ • Auto    │ │ • Promote │ │ • Deprec  │ │ • Audit   │ │ • Loop    │ └───────────┘ └───────────┘ 

|**Phase Summary**|**Phase Summary**|||
|---|---|---|---|
|**Phase**|**Timeline**|**Focus**|**Key Deliverable**|
|0|Weeks 1-2|Foundation|Telemetry, validation,<br>schemas|
|1|Month 1|Observability|Full logging, cost<br>tracking|
|2|Month 2|Memory|BM25 retrieval, decision<br>capture|
|3|Month 3|Weekly<br>Review|Automated analysis<br>pipeline|
|4|Months 4-5|Schema<br>Discovery|Pattern detection,<br>schema proposal|
|5|Months 6-7|Subagent<br>Spawning|Agent generation,<br>shadow deployment|
|6|Months 8-10|Autonomy|Level progression,<br>earned trust|
|7|Months 11-12|Self-<br>Improvement|Continuous evolution<br>loop|



**Phase 0: Foundation (Weeks 1-2)** 

|**Objectives**|||
|---|---|---|
|1. Enable telemetry|for baseline metrics||
|2. Implement inter-agent validation|||
|3. Establish schema validation on state fles|||
|4. Create foundation for all subsequent phases|||
|**Deliverables**|||
|**Deliverable**|**Path**|**Description**|
|Telemetry<br>enablement|~/.profileor~/.zshrc|Environment<br>variable<br>confguration|
|SubagentStop<br>hook|.claude/hooks/SubagentStop.sh|Validation of<br>agent handofs|
|State schemas|.claude/schemas/state/|Pydantic models<br>for state fles|
|Schema validator|.claude/scripts/validate-<br>state.py|Validation script|



**==> picture [209 x 91] intentionally omitted <==**

**----- Start of picture text -----**<br>
Updated settings .claude/settings.json Telemetry and<br>validation flags<br>Implementation Steps<br>Step 0.1: Enable OpenTelemetry (1 hour)<br># Add to ~/.profile or ~/.zshrc<br>export CLAUDE_CODE_ENABLE_TELEMETRY=1<br># Verify telemetry is active<br>claude --version # Should show telemetry indicator<br>**----- End of picture text -----**<br>


**Step 0.2: Implement SubagentStop Validation (4 hours)** 

**==> picture [265 x 368] intentionally omitted <==**

**----- Start of picture text -----**<br>
# .claude/hooks/SubagentStop.sh<br>#!/bin/bash<br>set -euo pipefail<br>AGENT_NAME="${1:-unknown}"<br>STATE_FILE="$HOME/.claude/tmp/handoff.json"<br>LOG_FILE="$HOME/.claude/tmp/validation_log.jsonl"<br>log_validation() {<br>local status="$1"<br>local message="$2"<br>echo "{\"timestamp\":\"$(date -<br>Iseconds)\",\"agent\":\"$AGENT_NAME\",\"status\":\"$status\",\"message\":\"$message\"}"<br>>> "$LOG_FILE"<br>}<br># Check handoff file exists<br>if [[ ! -f "$STATE_FILE" ]]; then<br>    log_validation "WARN" "No handoff file found"<br>exit 0   # Don't block if no handoff expected<br>fi<br># Validate JSON syntax<br>if ! jq empty "$STATE_FILE" 2>/dev/null ; then<br>    log_validation "ERROR" "Invalid JSON in handoff file"<br>exit 2   # Block - invalid state<br>fi<br># Validate required fields<br>REQUIRED=("context.task_summary" "context.success_criteria"<br>"status")<br>for  field  in "${REQUIRED[@]}" ; do<br>if ! jq -e ".$field" "$STATE_FILE" >/dev/null 2>&1 ; then<br>        log_validation "ERROR" "Missing required field: $field"<br>exit 2<br>fi<br>done<br># Validate referenced artifacts exist<br>if  jq -e '.artifacts' "$STATE_FILE" >/dev/null 2>&1 ; then<br>while IFS= read -r artifact ; do<br>if [[ -n "$artifact" && ! -f "$artifact" ]]; then<br>            log_validation "ERROR" "Missing artifact: $artifact"<br>exit 2<br>fi<br>done < <(jq -r '.artifacts | to_entries[] | .value // empty'<br>"$STATE_FILE")<br>fi<br>log_validation "OK" "Validation passed"<br>exit 0<br># Make executable<br>chmod +x ~/.claude/hooks/SubagentStop.sh<br>**----- End of picture text -----**<br>


**==> picture [149 x 8] intentionally omitted <==**

**----- Start of picture text -----**<br>
Step 0.3: Create State File Schemas (4 hours)<br>**----- End of picture text -----**<br>


**==> picture [149 x 68] intentionally omitted <==**

**----- Start of picture text -----**<br>
# .claude/schemas/state/scout_metrics.py<br>from  pydantic  import  BaseModel, Field<br>from  typing  import  Dict, List, Optional<br>from  datetime  import  datetime<br>class  ScopeMetrics(BaseModel):<br>    total_files: int = Field(ge=0)<br>    estimated_tokens: int = Field(ge=0)<br>    file_types: Optional[Dict[str, int]] = None<br>    largest_file: Optional[Dict[str, any]] = None<br>**----- End of picture text -----**<br>


**==> picture [173 x 27] intentionally omitted <==**

**----- Start of picture text -----**<br>
class  ComplexitySignals(BaseModel):<br>    cross_file_dependencies: int = Field(ge=0, default=0)<br>    module_count: int = Field(ge=1, default=1)<br>    circular_imports: int = Field(ge=0, default=0)<br>**----- End of picture text -----**<br>


**==> picture [158 x 27] intentionally omitted <==**

**----- Start of picture text -----**<br>
class  ScoutReport(BaseModel):<br>    scope_metrics: ScopeMetrics<br>    complexity_signals: ComplexitySignals<br>    recommendations: Optional[Dict[str, any]] = None<br>**----- End of picture text -----**<br>


**==> picture [101 x 14] intentionally omitted <==**

**----- Start of picture text -----**<br>
class  ScoutMetrics(BaseModel):<br>    schema_version: str = "1.0.0"<br>**----- End of picture text -----**<br>


generated_at: datetime scout_agent: str = "haiku" scout_report: ScoutReport 

_# .claude/schemas/state/handoff.py_ **from** pydantic **import** BaseModel, Field **from** typing **import** Dict, List, Optional **from** datetime **import** datetime **from** enum **import** Enum 

**class** HandoffStatus(str, Enum): PENDING = "pending" IN_PROGRESS = "in_progress" COMPLETED = "completed" FAILED = "failed" 

**class** HandoffContext(BaseModel): task_summary: str files_in_scope: List[str] = [] critical_constraints: List[str] = [] success_criteria: List[str] 

**class** HandoffMetadata(BaseModel): estimated_tokens: int = Field(ge=0) estimated_duration_minutes: int = Field(ge=0) tier_ceiling: str 

**class** Handoff(BaseModel): schema_version: str = "1.0.0" handoff_id: str from_agent: str to_agent: str created_at: datetime status: HandoffStatus context: HandoffContext artifacts: Dict[str, str] = {} metadata: HandoffMetadata 

## **Step 0.4: Create Schema Validator Script (2 hours)** 

- _#!/usr/bin/env python3_ 

- _# .claude/scripts/validate-state.py """Validate state files against schemas."""_ 

**import** sys **import** json **import** importlib.util **from** pathlib **import** Path SCHEMA_DIR = Path.home() / ".claude" / "schemas" / "state" STATE_DIR = Path.home() / ".claude" / "tmp" SCHEMA_MAP = { "scout_metrics.json": "scout_metrics.ScoutMetrics", "handoff.json": "handoff.Handoff", 

- } 

**def** load_schema_class(module_path: str): _"""Dynamically load a Pydantic model from schema directory."""_ module_name, class_name = module_path.rsplit(".", 1) spec = importlib.util.spec_from_file_location( module_name, SCHEMA_DIR / f"{module_name}.py" 

) module = importlib.util.module_from_spec(spec) spec.loader.exec_module(module) **return** getattr(module, class_name) 

**def** validate_file(file_path: Path) -> tuple[bool, str]: _"""Validate a single state file."""_ **if** file_path.name **not in** SCHEMA_MAP: **return** True, f"No schema defined for {file_path.name}" 

**try** : **with** open(file_path) **as** f: data = json.load(f) 

schema_class = load_schema_class(SCHEMA_MAP[file_path.name]) schema_class(**data) **return** True, "Valid" 

**except** json.JSONDecodeError **as** e: **return** False, f"Invalid JSON: {e}" **except** Exception **as** e: **return** False, f"Validation failed: {e}" 

**def** main(): 

- **if** len(sys.argv) > 1: 

_# Validate specific file_ file_path = Path(sys.argv[1]) valid, message = validate_file(file_path) print(f"{file_path.name}: {message}") sys.exit(0 **if** valid **else** 1) **else** : _# Validate all known state files_ errors = 0 **for** filename **in** SCHEMA_MAP: 

**==> picture [198 x 278] intentionally omitted <==**

**----- Start of picture text -----**<br>
            file_path = STATE_DIR / filename<br>if  file_path.exists():<br>                valid, message = validate_file(file_path)<br>                status = "✓" if  valid  else "✗"<br>print(f"{status} {filename}: {message}")<br>if not  valid:<br>                    errors += 1<br>        sys.exit(errors)<br>if __name__ == "__main__":<br>    main()<br>Verification Commands<br># Verify telemetry enabled<br>env | grep CLAUDE_CODE_ENABLE_TELEMETRY<br># Test SubagentStop hook<br>echo '{"status":"completed","context":<br>{"task_summary":"test","success_criteria":["done"]}}' ><br>~/.claude/tmp/handoff.json<br>~/.claude/hooks/SubagentStop.sh test_agent<br>echo "Exit code: $?" # Should be 0<br># Test schema validation<br>python3 ~/.claude/scripts/validate-state.py<br>~/.claude/tmp/handoff.json<br>Success Criteria<br>Criterion Verification<br>Telemetry active env | grep TELEMETRY returns value<br>SubagentStop validates Valid handoff returns exit 0<br>SubagentStop blocks invalid Missing fields returns exit 2<br>Schema validator works Validates known state files<br>Estimated Effort: 12-15 hours<br>**----- End of picture text -----**<br>


**Phase 1: Observability & Validation (Month 1)** 

|**Objectives**|||
|---|---|---|
|1. Complete hook|coverage for all lifecycle events||
|2. Implement comprehensive routing decision logging|||
|3. Create cost tracking per session and tier|||
|4. Establish baseline metrics for optimization|||
|**Deliverables**|||
|**Deliverable**|**Path**|**Description**|
|Enhanced<br>PreToolUse|.claude/hooks/PreToolUse.sh|Full routing<br>logging|
|Enhanced<br>PostToolUse|.claude/hooks/PostToolUse.sh|Outcome<br>capture|
|SessionEnd<br>hook|.claude/hooks/SessionEnd.sh|Session<br>summary|
|Routing log<br>schema|.claude/schemas/routing_log.json|Log format<br>defnition|
|Cost report|.claude/scripts/generate-cost-|Cost|
|script|report.sh|aggregation|
|Metrics<br>dashboard data|.claude/tmp/metrics/|Aggregated<br>metrics|
|**Implementation Steps**|||
|**Step 1.1: Enhanced Routing Log (4 hours)**|||



_# .claude/hooks/PreToolUse.sh (enhanced) #!/bin/bash_ set -euo pipefail TOOL_NAME="${CLAUDE_TOOL_NAME:-unknown}" SESSION_ID="${CLAUDE_SESSION_ID:-$(date +%s)}" LOG_FILE="$HOME/.claude/tmp/routing_log.jsonl" METRICS_FILE="$HOME/.claude/tmp/scout_metrics.json" SCORE_FILE="$HOME/.claude/tmp/complexity_score" TIER_FILE="$HOME/.claude/tmp/recommended_tier" _# Read current state_ COMPLEXITY_SCORE=$(cat "$SCORE_FILE" 2>/dev/null **||** echo "0") RECOMMENDED_TIER=$(cat "$TIER_FILE" 2>/dev/null **||** echo "sonnet") _# Determine requested tier from tool (simplified - actual implementation varies)_ REQUESTED_TIER="${CLAUDE_REQUESTED_TIER:-sonnet}" 

**==> picture [229 x 171] intentionally omitted <==**

**----- Start of picture text -----**<br>
# Make routing decision<br>if [[ "$REQUESTED_TIER" == "opus" && "$RECOMMENDED_TIER" != "opus"<br>]]; then<br>DECISION="BLOCK"<br>REASON="requested opus but ceiling is $RECOMMENDED_TIER"<br>EXIT_CODE=2<br>else<br>DECISION="PERMIT"<br>REASON="within tier ceiling"<br>EXIT_CODE=0<br>fi<br># Estimate cost (simplified)<br>case "$RECOMMENDED_TIER" in<br>haiku ) COST_PER_1K=0.00025  ;;<br>sonnet ) COST_PER_1K=0.003  ;;<br>opus ) COST_PER_1K=0.015  ;;<br>* ) COST_PER_1K=0.003  ;;<br>esac<br>ESTIMATED_TOKENS=$(jq -r<br>'.scout_report.scope_metrics.estimated_tokens // 10000'<br>"$METRICS_FILE" 2>/dev/null  || echo "10000")<br>ESTIMATED_COST=$(echo "scale=4; $ESTIMATED_TOKENS / 1000 *<br>$COST_PER_1K" | bc)<br>**----- End of picture text -----**<br>


_# Log decision_ cat >> "$LOG_FILE" << EOF {"timestamp":"$(date - Iseconds)","session_id":"$SESSION_ID","tool_name":"$TOOL_NAME","routing": {"complexity_score":$COMPLEXITY_SCORE,"calculated_tier":"$RECOMMENDED_TIER","requested_tier":"$REQUESTE {"estimated_tokens":$ESTIMATED_TOKENS,"estimated_cost_usd":$ESTIMATED_COST},"outcome": {"actual_tokens":null,"actual_cost_usd":null,"task_success":null}} EOF 

**==> picture [46 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
exit $EXIT_CODE<br>**----- End of picture text -----**<br>


**Step 1.2: PostToolUse Outcome Capture (4 hours)** 

**==> picture [223 x 212] intentionally omitted <==**

**----- Start of picture text -----**<br>
# .claude/hooks/PostToolUse.sh<br>#!/bin/bash<br>set -euo pipefail<br>TOOL_NAME="${CLAUDE_TOOL_NAME:-unknown}"<br>EXIT_CODE="${CLAUDE_TOOL_EXIT_CODE:-0}"<br>SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"<br>LOG_FILE="$HOME/.claude/tmp/routing_log.jsonl"<br># Determine success<br>if [[ "$EXIT_CODE" == "0" ]]; then<br>SUCCESS="true"<br>else<br>SUCCESS="false"<br>fi<br># Update the last log entry with outcome<br># (In practice, would use a proper log ID; this is simplified)<br>TEMP_FILE=$(mktemp)<br>head -n -1 "$LOG_FILE" > "$TEMP_FILE" 2>/dev/null  || true<br># Get last entry, update outcome<br>LAST_ENTRY=$(tail -1 "$LOG_FILE")<br>UPDATED_ENTRY=$(echo "$LAST_ENTRY" |  jq --arg success "$SUCCESS"<br>'.outcome.task_success = ($success == "true")')<br>cat "$TEMP_FILE" > "$LOG_FILE"<br>echo "$UPDATED_ENTRY" >> "$LOG_FILE"<br>rm "$TEMP_FILE"<br>exit 0<br>**----- End of picture text -----**<br>


**Step 1.3: SessionEnd Hook (3 hours)** 

**==> picture [232 x 151] intentionally omitted <==**

**----- Start of picture text -----**<br>
# .claude/hooks/SessionEnd.sh<br>#!/bin/bash<br>set -euo pipefail<br>SESSION_ID="${CLAUDE_SESSION_ID:-$(date +%s)}"<br>LOG_FILE="$HOME/.claude/tmp/routing_log.jsonl"<br>SUMMARY_DIR="$HOME/.claude/tmp/session_summaries"<br>mkdir -p "$SUMMARY_DIR"<br># Generate session summary<br>SUMMARY_FILE="$SUMMARY_DIR/${SESSION_ID}.json"<br># Aggregate session metrics<br>jq -s --arg sid "$SESSION_ID" '<br>{<br>    session_id: $sid,<br>    timestamp: (now | todate),<br>    metrics: {<br>        total_tool_calls: length,<br>        by_tier: (group_by(.routing.final_tier) | map({(.<br>[0].routing.final_tier): length}) | add),<br>        total_estimated_cost: (map(.cost.estimated_cost_usd // 0) |<br>**----- End of picture text -----**<br>


**==> picture [235 x 435] intentionally omitted <==**

**----- Start of picture text -----**<br>
add),<br>        success_rate: ((map(select(.outcome.task_success == true)) |<br>length) / (length + 0.001)),<br>        blocked_requests: (map(select(.routing.decision == "BLOCK"))<br>| length)<br>    }<br>}' "$LOG_FILE" > "$SUMMARY_FILE"<br># Archive routing log for this session<br>mv "$LOG_FILE" "$HOME/.claude/tmp/routing_log_${SESSION_ID}.jsonl"<br>echo "Session summary: $SUMMARY_FILE"<br>Step 1.4: Cost Report Generator (3 hours)<br>#!/bin/bash<br># .claude/scripts/generate-cost-report.sh<br>SUMMARY_DIR="$HOME/.claude/tmp/session_summaries"<br>OUTPUT_FILE="$HOME/.claude/tmp/metrics/cost_report.json"<br>mkdir -p "$(dirname "$OUTPUT_FILE")"<br># Aggregate all session summaries<br>find "$SUMMARY_DIR" -name "*.json" -mtime -7 | \<br>xargs cat 2>/dev/null  | \<br>jq -s '<br>{<br>    report_date: (now | todate),<br>    period: "last_7_days",<br>    sessions: length,<br>    total_cost_usd: (map(.metrics.total_estimated_cost // 0) | add),<br>    avg_cost_per_session: ((map(.metrics.total_estimated_cost // 0)<br>| add) / (length + 0.001)),<br>    by_tier: (<br>        map(.metrics.by_tier // {}) |<br>        add |<br>        to_entries |<br>        map({(.key): .value}) |<br>        add<br>    ),<br>    overall_success_rate: ((map(.metrics.success_rate // 0) | add) /<br>(length + 0.001)),<br>    total_blocked: (map(.metrics.blocked_requests // 0) | add)<br>}' > "$OUTPUT_FILE"<br>echo "Cost report generated: $OUTPUT_FILE"<br>cat "$OUTPUT_FILE" |  jq .<br>Verification Commands<br># Test full hook chain<br>echo '{"scout_report":{"scope_metrics":<br>{"total_files":10,"estimated_tokens":25000},"complexity_signals":<br>{"module_count":2}}}' > ~/.claude/tmp/scout_metrics.json<br>~/.claude/scripts/calculate-complexity.sh<br>~/.claude/hooks/PreToolUse.sh   # Should permit<br>cat ~/.claude/tmp/routing_log.jsonl  |  jq .<br># Generate cost report<br>~/.claude/scripts/generate-cost-report.sh<br>Success Criteria<br>**----- End of picture text -----**<br>


**Criterion Verification** All hooks execute No errors on tool invocation Routing logged routing_log.jsonl contains entries Cost tracked Cost report shows tier breakdown Session summaries Summary files created per session **Estimated Effort: 20-25 hours** 

**Phase 2: Memory Enhancement (Month 2)** 

**Objectives** 1. Implement BM25-based memory retrieval 2. Deploy decision capture schema 3. Standardize YAML frontmatter across memory files 4. Enable structured pattern accumulation **Deliverables Deliverable Path Description** BM25 retriever .claude/scripts/queryImproved memory-bm25.py retrieval Decision capture .claude/schemas/decision.py Decision schema Frontmatter .claude/docs/memoryFormat standard format.md specification 

Observation .claude/scripts/logRaw event logger observation.py capture Memory index .claude/memory/index.json Searchable index 

**Implementation Steps** 

**Step 2.1: BM25 Memory Retrieval (6 hours)** 

- _#!/usr/bin/env python3 # .claude/scripts/query-memory-bm25.py """BM25-based memory retrieval."""_ 

- **import** sys 

**import** json **import** re **from** pathlib **import** Path **from** typing **import** List, Dict, Tuple **from** dataclasses **import** dataclass 

_# Install: pip install rank-bm25 pyyaml_ **from** rank_bm25 **import** BM25Okapi **import** yaml MEMORY_DIR = Path.home() / ".claude" / "memory" INDEX_FILE = MEMORY_DIR / "index.json" 

@dataclass **class** MemoryDocument: path: Path title: str 

content: str category: str tags: List[str] created: str 

- **def** parse_frontmatter(content: str) -> Tuple[Dict, str]: _"""Extract YAML frontmatter and body from markdown."""_ **if not** content.startswith("---"): **return** {}, content 

parts = content.split("---", 2) **if** len(parts) < 3: **return** {}, content 

**try** : frontmatter = yaml.safe_load(parts[1]) body = parts[2].strip() **return** frontmatter **or** {}, body **except** yaml.YAMLError: **return** {}, content 

**def** load_memory_documents() -> List[MemoryDocument]: _"""Load all memory documents."""_ documents = [] 

**for** md_file **in** MEMORY_DIR.rglob("*.md"): **if** md_file.name.startswith("."): **continue** 

content = md_file.read_text() frontmatter, body = parse_frontmatter(content) 

doc = MemoryDocument( path=md_file, title=frontmatter.get("title", md_file.stem), content=body, category=frontmatter.get("category", "uncategorized"), tags=frontmatter.get("tags", []), created=frontmatter.get("created", "") 

) documents.append(doc) 

**return** documents 

**def** tokenize(text: str) -> List[str]: _"""Simple tokenization."""_ **return** re.findall(r'\w+', text.lower()) **def** search(query: str, top_k: int = 5, category: str = None) -> List[Dict]: _"""Search memory using BM25."""_ documents = load_memory_documents() 

_# Filter by category if specified_ **if** category: documents = [d **for** d **in** documents **if** d.category == category] **if not** documents: **return** [] 

_# Build corpus_ corpus = [tokenize(f"{d.title} {d.content} {' '.join(d.tags)}") **for** d **in** documents] 

_# Create BM25 index_ 

bm25 = BM25Okapi(corpus) 

_# Search_ query_tokens = tokenize(query) scores = bm25.get_scores(query_tokens) 

_# Rank results_ 

ranked = sorted(zip(documents, scores), key= **lambda** x: x[1], reverse=True) 

results = [] **for** doc, score **in** ranked[:top_k]: **if** score > 0: results.append({ "path": str(doc.path.relative_to(MEMORY_DIR)), "title": doc.title, "category": doc.category, "tags": doc.tags, "score": round(score, 3), "preview": doc.content[:200] + "..." **if** 

len(doc.content) > 200 **else** doc.content 

}) 

**return** results 

**def** main(): **import** argparse parser = argparse.ArgumentParser(description="Search memory with BM25") parser.add_argument("query", help="Search query") parser.add_argument("-k", "--top-k", type=int, default=5, help="Number of results") parser.add_argument("-c", "--category", help="Filter by category") parser.add_argument("--json", action="store_true", help="Output as JSON") 

args = parser.parse_args() results = search(args.query, args.top_k, args.category) **if** args.json: print(json.dumps(results, indent=2)) **else** : **for** i, r **in** enumerate(results, 1): print(f"\n{i}. [{r['score']}] {r['title']}") print(f"   Category: {r['category']} | Tags: {', '.join(r['tags'])}") print(f"   Path: {r['path']}") print(f"   {r['preview']}") 

**if** __name__ == "__main__": main() 

**Step 2.2: Decision Capture Schema (4 hours)** 

_# .claude/schemas/decision.py_ **from** pydantic **import** BaseModel, Field **from** typing **import** List, Dict, Optional, Any **from** datetime **import** datetime **from** enum **import** Enum 

**class** DecisionCategory(str, Enum): ROUTING = "routing_override" SCOPE = "scope_modification" APPROVAL = "plan_approval" REJECTION = "plan_rejection" ESCALATION = "human_escalation" DELEGATION = "task_delegation" 

**class** SystemRecommendation(BaseModel): action: str confidence: float = Field(ge=0, le=1) reasoning: str 

**class** HumanDecision(BaseModel): action: str reasoning_provided: Optional[str] = None time_to_decide_seconds: Optional[int] = None 

**class** DecisionOutcome(BaseModel): task_success: Optional[bool] = None quality_rating: Optional[int] = Field(None, ge=1, le=5) issues_encountered: List[str] = [] would_recommend_same: Optional[bool] = None 

**class** LearningMetadata(BaseModel): autonomy_level_at_time: int = Field(ge=1, le=5) pattern_match_candidates: List[str] = [] should_automate_similar: Optional[bool] = None requires_human_always: bool = False 

**class** Decision(BaseModel): _"""Schema for capturing human decisions for apprenticeship learning."""_ 

schema_version: str = "1.0.0" decision_id: str 

timestamp: datetime session_id: str decision_category: DecisionCategory context: Dict[str, Any] = Field( description="Task context at decision time" ) 

system_recommendation: Optional[SystemRecommendation] = None human_decision: HumanDecision outcome: DecisionOutcome = DecisionOutcome() learning_metadata: LearningMetadata 

**Step 2.3: Observation Logger (3 hours)** 

- _#!/usr/bin/env python3_ 

- _# .claude/scripts/log-observation.py_ 

**==> picture [217 x 205] intentionally omitted <==**

**----- Start of picture text -----**<br>
"""Log raw behavioral observations for pattern discovery."""<br>import  json<br>import  sys<br>from  datetime  import  datetime<br>from  pathlib  import  Path<br>from  typing  import  Dict, Any<br>OBSERVATIONS_DIR = Path.home() / ".claude" / "memory" /<br>"observations"<br>def  log_observation(<br>    event_type: str,<br>    context: Dict[str, Any],<br>    action: Dict[str, Any],<br>    outcome: Dict[str, Any] = None<br>) -> str:<br>"""Log a single observation event."""<br>    OBSERVATIONS_DIR.mkdir(parents=True, exist_ok=True)<br>    today = datetime.now().strftime("%Y-%m-%d")<br>    log_file = OBSERVATIONS_DIR / f"{today}-observations.jsonl"<br>    observation = {<br>"timestamp": datetime.now().isoformat(),<br>"event_type": event_type,<br>"context": context,<br>"action": action,<br>"outcome": outcome  or  {}<br>**----- End of picture text -----**<br>


} 

**with** open(log_file, "a") **as** f: f.write(json.dumps(observation) + "\n") **return** str(log_file) **def** main(): _"""CLI interface for logging observations."""_ **import** argparse parser = argparse.ArgumentParser(description="Log behavioral observation") parser.add_argument("event_type", help="Type of event") parser.add_argument("--context", type=json.loads, default={}, help="Context JSON") parser.add_argument("--action", type=json.loads, default={}, help="Action JSON") parser.add_argument("--outcome", type=json.loads, default={}, help="Outcome JSON") 

args = parser.parse_args() log_file = log_observation( args.event_type, args.context, args.action, args.outcome 

- ) 

print(f"Logged to: {log_file}") **if** __name__ == "__main__": main() 

## **Step 2.4: Memory Format Standard (2 hours)** 

- # .claude/docs/memory-format.md 

# Memory File Format Standard 

All memory files in _**`.claude/memory/`**_ MUST follow this format. ## Required YAML Frontmatter _**```yaml**_ --title **:** "Brief descriptive title" created **:** YYYY-MM-DD 

**==> picture [209 x 554] intentionally omitted <==**

**----- Start of picture text -----**<br>
category :  decisions|sharp-edges|facts|preferences<br>tags : [ tag1 ,  tag2 ,  tag3 ]<br>status :  active|deprecated|archived<br>summary : "One-line searchable summary for BM25 retrieval"<br>---<br>Optional Frontmatter Fields<br>updated :  YYYY-MM-DD<br>related : [ ./other-file.md ,  ../category/file.md ]<br>confidence :  high|medium|low<br>source :  session-id or "manual"<br>expires :  YYYY-MM-DD   # For time-sensitive facts<br>Body Format<br>Use standard markdown. Structure recommendations:<br>For Decisions<br>Context (why decision was needed)<br>Decision (what was decided)<br>Rationale (why this choice)<br>Consequences (what follows from this)<br>For Sharp Edges<br>Problem (what went wrong)<br>Symptoms (how it manifested)<br>Solution (how to fix/avoid)<br>Prevention (how to prevent recurrence)<br>For Facts<br>Statement (the fact)<br>Evidence (how we know)<br>Scope (when this applies)<br>For Preferences<br>Preference (what is preferred)<br>Context (when this applies)<br>Rationale (why this preference)<br>### Verification Commands<br>```bash<br># Test BM25 search<br>python3 ~/.claude/scripts/query-memory-bm25.py "authentication JWT"<br>--json<br># Log test observation<br>python3 ~/.claude/scripts/log-observation.py "test_event" \<br>   --context '{"task":"test"}' \<br>   --action '{"type":"test"}'<br># Verify observation logged<br>cat ~/.claude/memory/observations/$(date +%Y-%m-%d)-<br>observations.jsonl<br>Success Criteria<br>Criterion Verification<br>BM25 returns relevant results Test query returns expectedfiles<br>Decision schema validates Sample decision passes<br>Pydantic<br>Observations accumulating JSONL files growing<br>Frontmatter standard<br>Format doc exists<br>documented<br>Estimated Effort: 20-25 hours<br>**----- End of picture text -----**<br>


**Phase 3: Weekly Review System (Month 3)** 

**==> picture [41 x 9] intentionally omitted <==**

**----- Start of picture text -----**<br>
Objectives<br>**----- End of picture text -----**<br>


1. Implement Memory Synthesis agent 

2. Implement Systems Architect agent 

**==> picture [156 x 32] intentionally omitted <==**

**----- Start of picture text -----**<br>
3. Create human interview protocol<br>4. Establish automated recommendation generation<br>Deliverables<br>**----- End of picture text -----**<br>


|**Deliverable**|**Path**|**Description**|
|---|---|---|
|Memory<br>Synthesis agent|.claude/agents/memory-synthesis/|Aggregation<br>agent|
|Systems<br>Architect agent|.claude/agents/systems-architect/|Analysis<br>agent|
|Review<br>orchestrator|.claude/scripts/weekly-review.sh|Review<br>trigger|
|Interview<br>template|.claude/docs/review-interview.md|Human<br>interaction|
|Recommendation<br>schema|.claude/schemas/recommendation.py|Output<br>format|



## **Implementation Steps** 

**Step 3.1: Memory Synthesis Agent (6 hours)** 

_# .claude/agents/memory-synthesis/agent.yaml_ name **:** memory-synthesis version **:** 1.0.0 tier **:** sonnet purpose **:** | Aggregate and synthesize memory artifacts from the past week to prepare input for systems architect analysis. triggers **: -** manual **:** "/weekly-review" **-** scheduled **:** "0 9 * * 1" _# Monday 9 AM_ inputs **: -** .claude/memory/decisions/*.md (last 7 days) **-** .claude/memory/sharp-edges/*.md (last 7 days) **-** .claude/tmp/session_summaries/*.json (last 7 days) **-** .claude/tmp/routing_log_*.jsonl (last 7 days) **-** .claude/memory/observations/*.jsonl (last 7 days) outputs **: -** .claude/tmp/weekly_synthesis.json constraints **:** 

- max_input_tokens **:** 50000 

- max_output_tokens **:** 5000 

- timeout_minutes **:** 5 

**==> picture [131 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
# .claude/agents/memory-synthesis/CLAUDE.md<br>**----- End of picture text -----**<br>


- # Memory Synthesis Agent 

You are the Memory Synthesis agent. Your role is to aggregate and summarize the past week's system activity to prepare input for architectural analysis. 

## Your Task 

1. Read all provided memory files from the past week 

2. Read all session summaries and routing logs 

3. Read accumulated observations 

4. Produce a synthesis document containing: 

- Total sessions and tool invocations 

- Cost breakdown by tier 

- Success rate overall and by task type 

- Key decisions made and their outcomes 

- Sharp edges encountered 

- Human override patterns 

- Observation counts by event type 

- ## Output Format 

Produce JSON matching this structure: _**```json**_ { 

"period": {"start": "YYYY-MM-DD", "end": "YYYY-MM-DD"}, "sessions": {"count": **N** , "avg_duration_minutes": **N** }, "costs": {"total_usd": **N** , "by_tier": {"haiku": **N** , "sonnet": **N** , "opus": **N** }}, "performance": {"success_rate": **N** , "blocked_requests": **N** }, "decisions": [{"title": "", "outcome": "", "category": ""}], "sharp_edges": [{"title": "", "severity": "", "resolved": **bool** }], "overrides": {"count": **N** , "patterns": ["pattern1", "pattern2"]}, "observations": {"total": **N** , "by_type": {"type1": **N** , "type2": **N** }} } 

Be concise. Focus on patterns, not individual events. 

#### Step 3.2: Systems Architect Agent (8 hours) 

```yaml 

# .claude/agents/systems-architect/agent.yaml name: systems-architect version: 1.0.0 tier: opus purpose: | Analyze weekly synthesis and generate architectural 

recommendations 

including gap identification and improvement proposals. 

triggers: 

- after: memory-synthesis 

- manual: "/architect-review" 

inputs: - .claude/tmp/weekly_synthesis.json 

- .claude/memory/observations/*.jsonl (for pattern analysis) 

- .claude/schemas/ (current schema definitions) 

- .claude/agents/ (current agent registry) 

outputs: 

- .claude/tmp/architect_report.md 

- .claude/tmp/recommendations.json 

constraints: 

- extended_thinking: true 

- max_input_tokens: 100000 

- max_output_tokens: 10000 

- timeout_minutes: 10 

   - # .claude/agents/systems-architect/CLAUDE.md 

   - # Systems Architect Agent 

- You are the Systems Architect agent. Your role is to perform deep 

- analysis of system behavior and generate actionable improvement 

- recommendations. 

   - ## Your Responsibilities 

   - ### 1. Gap Analysis 

   - Identify recurring failure patterns (≥3 occurrences) 

   - Flag tasks that consistently exceed time/cost budgets 

   - Detect human overrides that suggest missing capabilities 

   - Find context boundary issues causing synthesis gaps 

   - ### 2. For Each Gap, Determine 

   - Can this be resolved by configuration change? → Generate config 

diff 

- Does this require new subagent capability? → Generate agent 

- template 

   - Is this a one-off or recurring pattern? → Prioritize accordingly 

   - ### 3. Schema Emergence Check (if observations ≥ 200) 

   - Analyze observation patterns 

   - Identify clusters with ≥10 instances 

   - Propose schema if patterns are statistically significant 

   - ### 4. Autonomy Assessment 

   - Review decision categories 

   - Calculate success rates per category 

   - Recommend level promotions where thresholds met 

## Output Format 

- ### architect_report.md Human-readable analysis with sections: 

- Executive Summary 

- Gap Analysis Findings 

- Schema Emergence Status 

- Autonomy Assessment 

- Recommendations Summary 

- ### recommendations.json _**```json**_ 

{ 

   - "generated_at": "ISO8601", "recommendations": [ 

   - { 

- "id": "uuid", "type": 

- "config_change|new_agent|schema_activation|autonomy_promotion", "priority": "high|medium|low", "title": "Brief title", "description": "Detailed description", "implementation": { "files_affected": ["path1", "path2"], "diff_or_template": "...", "effort_hours": **N** 

   - }, "evidence": ["observation1", "observation2"] 

   - } 

   - ] 

   - } 

Be thorough but actionable. Every recommendation must be implementable. 

#### Step 3.3: Review Orchestrator (4 hours) ```bash #!/bin/bash # .claude/scripts/weekly-review.sh 

**==> picture [53 x 7] intentionally omitted <==**

**----- Start of picture text -----**<br>
set -euo pipefail<br>**----- End of picture text -----**<br>


echo 

"═══════════════════════════════════════════════════════════════" 

echo "                    WEEKLY REVIEW PROCESS " echo 

"═══════════════════════════════════════════════════════════════" 

REVIEW_DATE=$(date +%Y-%m-%d) REVIEW_DIR="$HOME/.claude/tmp/reviews/$REVIEW_DATE" mkdir -p "$REVIEW_DIR" 

# Phase 1: Memory Synthesis 

echo "" echo "Phase 1: Memory Synthesis" echo "─────────────────────────" 

# Invoke memory synthesis agent (placeholder - actual implementation via Claude) claude --agent memory-synthesis --output "$REVIEW_DIR/synthesis.json" 

if [[ ! -f "$REVIEW_DIR/synthesis.json" ]]; then echo "ERROR: Memory synthesis failed" exit 1 fi 

echo "✓ Synthesis complete: $REVIEW_DIR/synthesis.json" 

# Phase 2: Systems Architect Analysis 

echo "" echo "Phase 2: Systems Architect Analysis" echo "────────────────────────────────────" 

claude --agent systems-architect \ --input "$REVIEW_DIR/synthesis.json" \ --output-report "$REVIEW_DIR/architect_report.md" \ --output-recommendations "$REVIEW_DIR/recommendations.json" 

echo "✓ Analysis complete" 

# Phase 3: Human Interview 

echo "" echo "Phase 3: Review Findings" echo "────────────────────────" echo "" 

cat "$REVIEW_DIR/architect_report.md" echo "" echo 

"═══════════════════════════════════════════════════════════════" echo "                    RECOMMENDATIONS " echo 

"═══════════════════════════════════════════════════════════════" 

# Present recommendations interactively 

jq -r '.recommendations[] | "[\(.priority)] \(.title)\n    \ (.description)\n"' \ "$REVIEW_DIR/recommendations.json" 

echo "" echo "Review complete. Recommendations saved to:" echo "  $REVIEW_DIR/recommendations.json" echo "" echo "To implement recommendations, run:" echo "  claude --implement-recommendation <id>" 

**==> picture [90 x 9] intentionally omitted <==**

**----- Start of picture text -----**<br>
Verification Commands<br>**----- End of picture text -----**<br>


_# Test weekly review (after accumulating data)_ 

~/.claude/scripts/weekly-review.sh 

_# View generated recommendations_ cat ~/.claude/tmp/reviews/$(date +%Y-%m-%d)/recommendations.json **|** jq . **Success Criteria Criterion Verification** Synthesis agent produces synthesis.json exists and valid output Architect agent analyzes architect_report.md exists recommendations.json has Recommendations generated entries Human review possible Report is readable and actionable 

**Estimated Effort: 25-30 hours** 

## **Phase 4: Emergent Schema Discovery (Months 4-5)** 

|**Objectives**||
|---|---|
|1. Implement observation accumulation infrastructure||
|2. Create pattern detection subagent||
|3. Build schema proposal mechanism||
|4. Establish human approval workfow||
|**Deliverables**||
|**Deliverable**<br>**Path**|**Description**|
|Schema Discovery<br>agent<br>.claude/agents/schema-<br>discovery/<br>Pattern analyzer||
|Pattern detector<br>.claude/scripts/detect-<br>patterns.py<br>Statistical analysis||
|Schema proposer<br>.claude/scripts/propose-<br>schema.py<br>Schema generation||
|Approval workfow<br>.claude/docs/schema-<br>approval.md|Human process|
|_Detailed implementation in Technical Specifcations (Part VI)_||
|**Key Thresholds (from research)**||
|**Threshold**<br>**Value**|**Source**|
|Minimum observations for<br>analysis<br>200|Statistical power<br>research|
|Observations per cluster<br>30|80% statistical power|
|Silhouette score threshold<br>0.5|Cluster validity|
|Bootstrap stability<br>80%|Reproducibility<br>requirement|



**Estimated Effort: 40-50 hours** 

**Phase 5: Subagent Spawning (Months 6-7)** 

|**Objectives**||
|---|---|
|1. Implement gap-to-agent generation||
|2. Create shadow deployment infrastructure||
|3. Build promotion criteria system||
|4. Establish deprecation lifecycle||
|**Deliverables**||
|**Deliverable**<br>**Path**|**Description**|
|Agent generator<br>.claude/scripts/generate-<br>agent.py<br>Template creation||
|Shadow runner<br>.claude/scripts/shadow-<br>deploy.sh|Parallel execution|
|Promotion<br>evaluator<br>.claude/scripts/evaluate-<br>promotion.py<br>Criteria check||
|Agent registry<br>.claude/agents-index.json<br>Lifecycle tracking||
|_Detailed implementation in Technical Specifcations (Part VI)_||
|**Key Thresholds (from research)**||
|**Threshold**<br>**Value**|**Source**|
|Shadow period minimum<br>10 invocations|Statistical<br>signifcance|
|Success rate for<br>promotion<br>90%|Quality gate|
|Error rate vs baseline<br>≤0.1%<br>increase|Risk management|
|Maximum shadow<br>duration<br>14 days|Prevent limbo|
|Agent population cap<br>15 active|Coordination<br>overhead|



**Estimated Effort: 50-60 hours** 

**Phase 6: Autonomy Progression (Months 8-10)** 

**Objectives** 

1. Implement decision category tracking 

2. Build success rate monitoring per category 

3. Create level promotion logic 

4. Establish audit trail system 

**Deliverables** 

|**Deliverable**|**Path**|**Description**|
|---|---|---|
|Autonomy tracker|.claude/autonomy-<br>levels.yaml|Level state|
|Category analyzer|.claude/scripts/analyze-<br>categories.py|Success rates|
|Promotion engine|.claude/scripts/promote-<br>autonomy.py|Level changes|
|Audit logger|.claude/memory/autonomy-<br>audit.jsonl|Decision trail|



_Detailed implementation in Technical Specifications (Part VI)_ 

**Key Thresholds (from research)** 

||**Level**<br>**Transition**|**Requirement**|**Source**|
|---|---|---|---|
|L1|→ L2|100 decisions captured|Suficient data|
|L2|→ L3|95% success over 200<br>decisions|Collaborator<br>trust|
|L3|→ L4|98% success over 500<br>decisions|Consultant trust|
|L4|→ L5|Domain-specifc, human-<br>defned|Full trust|



**Estimated Effort: 40-50 hours** 

## **Phase 7: Self-Improving System (Months 11-12)** 

**Objectives** 

1. Implement schema versioning automation 

2. Create subagent deprecation lifecycle 

3. Build continuous improvement loop 

4. Establish human oversight dashboard 

|**Deliverables**|||
|---|---|---|
|**Deliverable**|**Path**|**Description**|
|Schema migrator|.claude/scripts/migrate-<br>schema.py|Version handling|
|Deprecation<br>manager|.claude/scripts/manage-<br>deprecation.py|Lifecycle|
|Improvement loop|.claude/scripts/continuous-<br>improve.sh|Automation|
|Dashboard data|.claude/tmp/dashboard/|Visualization data|



_Detailed implementation in Technical Specifications (Part VI)_ 

**Estimated Effort: 35-45 hours** 

## **Dependency Graph** 

┌──────────────────────────────────────────────────────────────────── 

**==> picture [265 x 14] intentionally omitted <==**

**----- Start of picture text -----**<br>
||
|---|
|│                              IMPLEMENTATION DEPENDENCIES|
|│|

**----- End of picture text -----**<br>


└──────────────────────────────────────────────────────────────────── 

Phase 0 

───────────────────────────────────────────────────────────────────── 

│ │ Telemetry, Validation, Schemas │ └──► Phase 1 ───────────────────────────────────────────────────────────────────── 

- │ 

│ Observability, Logging, Cost Tracking 

- │ ├──► Phase 2 

- ────────────────────────────────────────────────────────────────► 

│       │ 

│       │ BM25, Decision Capture, Observations 

**==> picture [207 x 260] intentionally omitted <==**

**----- Start of picture text -----**<br>
          │       │<br>          │       └──► Phase 3<br>─────────────────────────────────────────────────────────►<br>          │               │<br>          │               │ Weekly Review, Synthesis, Architect<br>          │               │<br>          │               ├──► Phase 4<br>──────────────────────────────────────────────────►<br>          │               │       │<br>          │               │       │ Schema Discovery, Pattern<br>Detection<br>          │               │       │<br>          │               │       └──► Phase 5<br>───────────────────────────────────────────►<br>          │               │               │<br>          │               │               │ Subagent Spawning,<br>Shadow Deploy<br>          │               │               │<br>          │               │               └──► Phase 6<br>────────────────────────────────────►<br>          │               │                       │<br>          │               │                       │ Autonomy<br>Progression<br>          │               │                       │<br>          │               │                       └──► Phase 7<br>─────────────────────────────►<br>          │               │                               │<br>          │               │                               │ Self-<br>Improving Loop<br>          │               │                               │<br>          │               │                               ▼<br>          │               │                           COMPLETE<br>          │               │<br>          │               │ (Weekly Review continues independently)<br>          │<br>└──────────────────────────────────────────────────────────────►<br>          │<br>          │ (Observability continues independently)<br>**----- End of picture text -----**<br>


└──────────────────────────────────────────────────────────────────── 

LEGEND: ───► Sequential dependency (must complete before next) │   Parallel workstream (continues independently) 

**==> picture [209 x 6] intentionally omitted <==**

## **Total Effort Estimate** 

|**Phase**|**Timeline**|**Efort (hours)**|
|---|---|---|
|Phase 0|Weeks 1-2|12-15|
|Phase 1|Month 1|20-25|
|Phase 2|Month 2|20-25|
|Phase 3|Month 3|25-30|
|Phase 4|Months 4-5|40-50|
|Phase 5|Months 6-7|50-60|
|Phase 6|Months 8-10|40-50|
|Phase 7|Months 11-12|35-45|
|**Total**|**12 months**|**242-300 hours**|



At 10-15 hours/week of focused development: **20-30 weeks of active work** spread across 12 months, allowing for iteration, testing, and refinement. 

