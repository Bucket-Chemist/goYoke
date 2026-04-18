# Bug Hunter Report

- Findings reviewed: 12
- Confirmed: 12
- Dismissed: 0
- Manual review: 0

## Confirmed Bugs
- BUG-1 | Medium | pkg/config/paths.go | GetGOgentDir accepted XDG runtime/cache directories after mkdir alone, so an existing but unwritable runtime directory caused guard, counter, and other state writes to fail at runtime.
  Confidence: 95 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. The original XDG path selection trusted directory existence but not writability, which breaks runtime state creation under read-only runtime dirs and can drop guard enforcement into degraded behavior.
- BUG-3 | Medium | pkg/session/scanner.go | Session JSONL readers used the default scanner token limit, so large pending-learning and sharp-edge records failed during resume, query, and artifact analysis flows.
  Confidence: 96 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Session artifact parsing reused default bufio.Scanner behavior across multiple JSONL readers, so valid large artifact lines broke context-resume and sharp-edge query flows in the same way the transcript parser did.
- BUG-2 | Medium | pkg/routing/transcript.go, pkg/routing/events.go | Routing transcript readers used the default bufio.Scanner token limit, so any single JSONL event above 64 KiB caused enforcement, metadata extraction, and task-analysis paths to fail on valid transcripts.
  Confidence: 97 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Routing transcript parsing treated each JSONL line as a scanner token without increasing the default cap, so valid high-volume tool events broke both routing/task-analysis code paths and metadata extraction for workflow endstate handling.
- BUG-4 | Medium | cmd/gogent-team-run/wave.go | runWaves only returned an error when every member in a wave failed, so final-wave partial failures were reported as successful team runs even though failed members remained.
  Confidence: 97 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Team-run execution returned success for final-wave partial failures because only all-member failure was treated as fatal. That leaks failed member state past the command boundary and misreports the overall run outcome.
- BUG-6 | Medium | pkg/telemetry/scanner.go | Telemetry JSONL readers used the default scanner token limit, so large valid records were dropped or caused load failures across invocation, escalation, scout, collaboration, and lifecycle analytics.
  Confidence: 96 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Telemetry analytics reused default bufio.Scanner behavior across multiple JSONL readers, so valid large telemetry records were dropped or surfaced as read failures across the major loader paths.
- BUG-7 | Medium | pkg/workflow/logging.go | ReadEndstateLogs used the default scanner token limit, so large valid endstate JSONL entries were skipped or failed during workflow analytics.
  Confidence: 95 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Workflow endstate log reading relied on the default scanner cap even though serialized recommendation payloads can exceed it, breaking valid analytics input.
- BUG-8 | Medium | pkg/memory/failure_tracking.go | Failure tracker scanning used the default scanner token limit, so large valid failure entries stopped loop-detection and clearing logic from seeing matching records.
  Confidence: 95 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. The failure-tracker scan path powered both counting and clearing operations and previously used the default scanner limit, making large valid entries effectively invisible.
- BUG-9 | Medium | pkg/config/paths.go | GetGOgentDataDir accepted XDG data directories after mkdir alone, so existing but unwritable data directories caused telemetry and ML log writes to fail later at runtime.
  Confidence: 96 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Persistent data directory selection still trusted directory existence instead of actual writability, so telemetry and ML logging could fail at first write even after path selection succeeded.
- BUG-10 | Medium | cmd/gogent-permission-gate/cache.go | Permission-gate cache filenames embedded raw session_id values, allowing path separator and traversal characters to escape the intended runtime directory.
  Confidence: 96 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Permission-gate cache filenames incorporated raw session IDs from stdin events, so slash-containing IDs altered the resolved cache path and escaped the intended runtime directory.
- BUG-11 | Medium | cmd/gogent-aggregate/main.go | The weekly aggregation tool counted JSONL records with the default scanner limit, so a single large archived record caused line counts to fail and summary stats to degrade.
  Confidence: 95 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. Weekly aggregation counted archived JSONL entries with the default scanner cap, so a single large valid line degraded artifact counts in the generated summary.
- BUG-12 | Medium | internal/tui/components/telemetry/telemetry.go | The TUI telemetry panel still used the default scanner limit when reading routing-decisions JSONL, so large valid decisions could disappear from the UI or raise load errors.
  Confidence: 95 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. The TUI routing telemetry panel still used the default scanner cap after backend readers were fixed, so valid large decisions could vanish from the UI or surface as load errors.
- BUG-5 | Medium | internal/tui/bridge/server.go | IPC bridge startup built Unix socket paths directly from XDG runtime/temp directories without checking pathname limits, so long base paths failed before the TUI accepted MCP connections.
  Confidence: 94 (high) | INDEPENDENTLY_VERIFIED
  Analysis: Confirmed. The bridge's socket-path builder did not account for Unix pathname socket length limits, so certain valid runtime/temp directory configurations prevented the TUI bridge from binding at startup.

## Manual Review
- None

## Dismissed Findings
- None

## Coverage Note
- Partial coverage only. This run expanded into `pkg/config`, `pkg/routing`, `pkg/session`, `pkg/telemetry`, `pkg/workflow`, `pkg/memory`, `cmd/gogent-team-run`, `cmd/gogent-permission-gate`, `cmd/gogent-aggregate`, `internal/tui/bridge`, `internal/tui/mcp`, and `internal/tui/components/telemetry`, with verification on the touched package slices.
- Non-finding test hardening was also applied where the suite depended on ambient path length, writable-home assumptions, or the local presence of the `claude` binary.
- Large-codebase loop automation was not available in this runtime, so the remaining queued source files in `cmd/`, `internal/`, and most of `pkg/` were not audited in this pass.
