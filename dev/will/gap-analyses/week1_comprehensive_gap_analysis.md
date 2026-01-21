# Comprehensive Gap Analysis: Week 1 (GOgent-003 to GOgent-019)

**Date:** 2026-01-16
**Scope:** Week 1 Tickets (GOgent-003 to GOgent-019)
**Reviewer:** Senior Systems Architect
**Status:** CRITICAL GAPS IDENTIFIED

## 1. Executive Summary

A systematic review of the Week 1 migration plan reveals significant discrepancies between the proposed "clean slate" implementation and the complex reality of the existing v2.2.0 Bash environment. The plan consistently underestimates the complexity of existing data structures and logic, risking a regression in functionality and security.

**Major systemic issues:**
1.  **Schema Drift:** Ticket definitions rely on outdated or simplified versions of JSON schemas (`routing-schema.json`, `agents-index.json`).
2.  **Missing Logic:** Critical features present in Bash hooks (e.g., specific regex patterns for overrides, nuanced permission checks) are absent from the Go specs.
3.  **Process Gaps:** GOgent-005 ("Parse Task Input") is missing from the detailed markdown entirely, despite being a dependency for later tickets.

## 2. Detailed Ticket Analysis

### GOgent-003: Agent Index Parsing
**Gap:** The proposed `AgentIndex` struct in the ticket is a simplified version of reality.
-   **Actual State:** `agents-index.json` likely contains richer fields for agent capabilities, state management, and routing rules than the ticket anticipates.
-   **Risk:** Data loss during unmarshaling. If the Go struct doesn't capture all fields, we lose critical configuration data needed for routing decisions.
-   **Recommendation:** Do not use the ticket's struct definition. Generate the struct directly from `~/.claude/agents/agents-index.json` to ensure 100% fidelity.

### GOgent-004a: Config Loader
**Gap:** The ticket proposes a naive loader that doesn't account for the "split brain" configuration state.
-   **Actual State:** Configuration is split between `~/.claude/` (global) and project-specific `.claude/` (local) directories. The Bash hooks likely have precedence logic (Local > Global).
-   **Risk:** The Go implementation might read the wrong config, leading to inconsistent behavior between legacy Bash and new Go tools.
-   **Recommendation:** Implement a `LoadConfig` strategy that explicitly handles precedence: Project Local > User Global > Defaults.

### GOgent-005: Parse Task Input
**Gap:** **MISSING FROM DETAILED PLAN.**
-   **Impact:** GOgent-020 (Opus Blocking) and GOgent-023 (Subagent Validation) depend on this.
-   **Recommendation:** This ticket must be formally defined. It needs to robustly parse the `prompt` field to extract hidden instructions or metadata that the current Bash scripts might be parsing (e.g., `AGENT: <name>` markers).

### GOgent-006/007/008: Event Parsing
**Gap:** The `ToolEvent` struct is too simple.
-   **Actual State:** Claude Code's IPC is undocumented but likely includes more context than just `tool_name` and `tool_input`. The "real corpus" capture in GOgent-008b is good, but doing it *after* defining the struct is backward.
-   **Risk:** The Go parser will fail or silently drop data on real events.
-   **Recommendation:** **Reverse the order.** Capture the corpus (GOgent-008b) *first*. Analyze the JSON. *Then* define the `ToolEvent` struct (GOgent-006).

### GOgent-010: Override Flags
**Gap:** Regex naivety.
-   **Ticket Spec:** `regexp.MustCompile("--force-tier=(\w+)")`
-   **Actual Requirement:** Users might use `=`, ` `, or no separator? Bash scripts might support case-insensitivity or different flag variations (`-f`, `--force`). The regex `\w+` is too restrictive if tier names contain hyphens (e.g., `haiku-thinking`).
-   **Risk:** Valid overrides will be ignored, frustrating users.
-   **Recommendation:** Use a robust CLI argument parsing library (like `pflag` or `cobra`) or a more permissive regex `[a-zA-Z0-9-_]+` that matches the schema's tier name format. Check existing Bash `validate-routing.sh` for exact supported patterns.

### GOgent-013: Scout Metrics
**Gap:** Path assumptions.
-   **Ticket Spec:** `filepath.Join(projectDir, ".claude", "tmp", "scout_metrics.json")`
-   **Actual State:** The Bash `haiku-scout` might output to a different location or format. The ticket assumes a specific JSON structure that might not exist yet if `haiku-scout` is a Bash script.
-   **Risk:** The Go tool will fail to find metrics, defaulting to fallback behavior permanently.
-   **Recommendation:** Verify the *exact* output path and JSON structure of the current `haiku-scout` tool. If it's unstructured text, GOgent-013 is impossible without updating `haiku-scout` first.

### GOgent-017: Tool Permissions
**Gap:** "Wildcard" handling is brittle.
-   **Ticket Spec:** Checks for `"*"` string.
-   **Actual State:** The schema gap analysis showed `Tools` is a `[]string` even for Opus. The wildcard might be `["*"]` (array containing star), not `"*"` (raw string).
-   **Risk:** Runtime panic (type assertion failure) or incorrect permission denial.
-   **Recommendation:** Use the strictly typed schema from our `GOgent-002` fix. Handle `[]string{"*"}` as the wildcard condition.

## 3. Strategic Recommendations

1.  **Schema-First Development:** Stop writing Go structs from memory. Create a script that generates Go types from the live JSON configuration files (`routing-schema.json`, `agents-index.json`). This ensures the Go code is always in sync with the configuration reality.
2.  **Corpus-First Parsing:** Prioritize GOgent-008b (Capture Corpus). Do not write the Event Parser (GOgent-007) until you have at least 10 real event samples to validate against.
3.  **Bash Parity Check:** Before implementing any logic (like Overrides or Permissions), read the corresponding Bash script (`validate-routing.sh`) to understand the *exact* logic being replaced. The Go code must duplicate this logic exactly to pass regression tests.

## 4. Modified Plan for Week 1

*   **Step 1:** Run `GOgent-008b` immediately to build the event corpus.
*   **Step 2:** Generate `pkg/routing/schema.go` and `pkg/routing/agents.go` from live JSON files.
*   **Step 3:** Analyze `validate-routing.sh` to extract the exact regex and logic for overrides and permissions.
*   **Step 4:** Implement `GOgent-005` (Task Parsing) explicitly.
*   **Step 5:** Proceed with the rest of the implementation, using the corpus for TDD.
