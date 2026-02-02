# gogent-scout Implementation Specification v2.0

> **Generated:** 2026-02-02
> **Author:** Einstein (Opus)
> **Review:** Staff Architect - APPROVED_WITH_CHANGES
> **Status:** READY FOR IMPLEMENTATION
> **Ticket Reference:** GOgent-150

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | 2026-02-02 | Addressed 5 critical findings from staff-architect review |
| 1.0 | 2026-02-02 | Initial specification |

### Key Changes from v1.0

1. **SIMPLIFIED Native Scout** - Basic metrics only, NO import density analysis
2. **Added Schema Validation** - Gemini output wrapper with version check
3. **Added Synthetic Fallback** - Graceful degradation when all backends fail
4. **Added Performance Benchmarks** - With acceptance criteria
5. **Clarified Algorithm** - Detailed pseudo-code for go-pro

---

## 1. Executive Summary

Create a unified `gogent-scout` Go binary with **smart backend routing**:

| Scope | Backend | Latency Target |
|-------|---------|----------------|
| < 20 files | Native Go | < 100ms (p50) |
| ≥ 20 files | gemini-slave (Gemini 3 Flash) | 1-3s |
| All backends fail | Synthetic fallback | < 50ms |

**Design Philosophy:** Native scout is SIMPLE (metrics only). Semantic analysis (imports, dependencies) is delegated to Gemini.

---

## 2. Interface Specification

### 2.1 CLI Interface

```bash
# Basic usage
gogent-scout <target-directory> "<instruction>"

# Pipe file list
find src -name "*.go" | gogent-scout - "<instruction>"

# Environment overrides
SCOUT_BACKEND=gemini gogent-scout ./pkg "Assess scope"
SCOUT_BACKEND=native gogent-scout ./pkg "Assess scope"
SCOUT_THRESHOLD=30 gogent-scout ./pkg "Assess scope"

# Output to specific file
gogent-scout ./pkg "Assess scope" --output=.claude/tmp/scout_metrics.json
```

### 2.2 Exit Codes

| Code | Meaning | Behavior |
|------|---------|----------|
| 0 | Success | Always return valid JSON (even degraded) |
| 1 | Invalid arguments | Print usage |
| 2 | Target not found | Print error |

**Critical:** Exit 0 with degraded report is preferable to exit non-zero. Consumers depend on scout output.

### 2.3 Output Schema (STRICT)

```json
{
  "schema_version": "1.0",
  "scout_report": {
    "backend": "native|gemini|native_fallback|synthetic_fallback",
    "target": "/path/to/target",
    "timestamp": "2026-02-02T10:30:00Z",

    "scope_metrics": {
      "total_files": 15,
      "total_lines": 2340,
      "estimated_tokens": 23400,
      "languages": ["go", "md"],
      "file_types": {".go": 12, ".md": 3},
      "max_file_lines": 450,
      "files_over_500_lines": 0
    },

    "complexity_signals": {
      "available": true,
      "import_density": "medium",
      "cross_file_dependencies": 8,
      "test_coverage_present": true,
      "note": "Semantic analysis from Gemini backend"
    },

    "routing_recommendation": {
      "recommended_tier": "sonnet",
      "confidence": "high",
      "reasoning": "15 files, 2340 lines, single language",
      "clarification_needed": null
    },

    "key_files": [
      {"path": "pkg/routing/schema.go", "lines": 450, "relevance": "Largest file"}
    ],

    "warnings": []
  }
}
```

**Native scout specifics:** When `backend: "native"`:
```json
"complexity_signals": {
  "available": false,
  "import_density": null,
  "cross_file_dependencies": null,
  "test_coverage_present": true,
  "note": "Semantic analysis unavailable - basic metrics only"
}
```

---

## 3. Backend Selection Logic (MULTI-FACTOR)

### 3.1 Routing Score Calculation

```go
type RoutingScore struct {
    FileCount     int
    TotalLines    int
    MaxFileLines  int
    LanguageCount int
}

func (rs RoutingScore) Score() int {
    // Weighted composite score
    // Each factor normalized to 0-100 range, then weighted

    fileScore := min(rs.FileCount * 5, 100)        // 20 files = 100
    lineScore := min(rs.TotalLines / 50, 100)      // 5000 lines = 100
    maxFileScore := min(rs.MaxFileLines / 10, 100) // 1000 line file = 100
    langScore := rs.LanguageCount * 25             // 4 languages = 100

    return (fileScore*40 + lineScore*30 + maxFileScore*20 + langScore*10) / 100
}

const (
    NativeThreshold = 40  // Below this → native scout
    // Score 40 ≈ 8 files, 2000 lines, max 400 lines, 1 language
)
```

### 3.2 Backend Selection

```go
func selectBackend(target string) (string, error) {
    // 1. Check environment override
    if backend := os.Getenv("SCOUT_BACKEND"); backend != "" {
        if backend == "native" || backend == "gemini" {
            return backend, nil
        }
        return "", fmt.Errorf("invalid SCOUT_BACKEND: %s (use 'native' or 'gemini')", backend)
    }

    // 2. Calculate routing score
    score, err := calculateRoutingScore(target)
    if err != nil {
        // Can't even count files → try native anyway
        return "native", nil
    }

    // 3. Apply threshold (configurable via env)
    threshold := NativeThreshold
    if t := os.Getenv("SCOUT_THRESHOLD"); t != "" {
        if parsed, err := strconv.Atoi(t); err == nil {
            threshold = parsed
        }
    }

    if score.Score() < threshold {
        return "native", nil
    }
    return "gemini", nil
}
```

---

## 4. Native Scout Implementation (SIMPLIFIED)

### 4.1 Scope: Basic Metrics ONLY

Native scout provides:
- ✅ File count (by extension)
- ✅ Line count (total and per-file)
- ✅ File size distribution (max, files over 500 lines)
- ✅ Language detection (by extension mapping)
- ✅ Test file presence (by naming convention)
- ❌ ~~Import density~~ (requires AST parsing)
- ❌ ~~Cross-file dependencies~~ (requires semantic analysis)

### 4.2 Implementation

```go
// cmd/gogent-scout/native_scout.go

var SupportedExtensions = map[string]string{
    ".go":   "go",
    ".py":   "python",
    ".r":    "r",
    ".R":    "r",
    ".ts":   "typescript",
    ".tsx":  "typescript",
    ".js":   "javascript",
    ".jsx":  "javascript",
    ".md":   "markdown",
}

var TestPatterns = []string{
    "_test.go",
    "test_*.py",
    "*_test.py",
    "test-*.R",
}

type NativeScout struct {
    Target      string
    Instruction string
}

func (ns *NativeScout) Run() (*ScoutReport, error) {
    // 1. Walk directory, collect file stats
    var files []FileInfo
    err := filepath.WalkDir(ns.Target, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return nil // Skip unreadable files
        }
        if d.IsDir() {
            // Skip hidden directories and vendor
            if strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || d.Name() == "node_modules" {
                return filepath.SkipDir
            }
            return nil
        }

        ext := filepath.Ext(path)
        if lang, ok := SupportedExtensions[ext]; ok {
            lines, _ := countLines(path)
            files = append(files, FileInfo{
                Path:     path,
                Lines:    lines,
                Language: lang,
                IsTest:   isTestFile(path),
            })
        }
        return nil
    })
    if err != nil {
        return nil, fmt.Errorf("failed to walk directory: %w", err)
    }

    // 2. Aggregate metrics
    metrics := aggregateMetrics(files)

    // 3. Generate routing recommendation (basic heuristics)
    recommendation := ns.generateRecommendation(metrics)

    // 4. Identify key files (top 5 by size)
    keyFiles := identifyKeyFiles(files, 5)

    return &ScoutReport{
        SchemaVersion: "1.0",
        Backend:       "native",
        Target:        ns.Target,
        Timestamp:     time.Now().Format(time.RFC3339),
        ScopeMetrics:  metrics,
        ComplexitySignals: &ComplexitySignals{
            Available:             false,
            ImportDensity:         nil,
            CrossFileDependencies: nil,
            TestCoveragePresent:   hasTestFiles(files),
            Note:                  "Semantic analysis unavailable - basic metrics only",
        },
        RoutingRecommendation: recommendation,
        KeyFiles:              keyFiles,
        Warnings:              []string{},
    }, nil
}

func countLines(path string) (int, error) {
    f, err := os.Open(path)
    if err != nil {
        return 0, err
    }
    defer f.Close()

    count := 0
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        count++
    }
    return count, scanner.Err()
}

func aggregateMetrics(files []FileInfo) *ScopeMetrics {
    metrics := &ScopeMetrics{
        FileTypes: make(map[string]int),
    }

    langSet := make(map[string]bool)
    var maxLines int

    for _, f := range files {
        metrics.TotalFiles++
        metrics.TotalLines += f.Lines
        metrics.FileTypes[filepath.Ext(f.Path)]++
        langSet[f.Language] = true

        if f.Lines > maxLines {
            maxLines = f.Lines
        }
        if f.Lines > 500 {
            metrics.FilesOver500Lines++
        }
    }

    for lang := range langSet {
        metrics.Languages = append(metrics.Languages, lang)
    }
    sort.Strings(metrics.Languages)

    metrics.MaxFileLines = maxLines
    metrics.EstimatedTokens = metrics.TotalLines * 10 // ~10 tokens per line

    return metrics
}

func (ns *NativeScout) generateRecommendation(m *ScopeMetrics) *RoutingRecommendation {
    // Simple heuristics based on routing-schema.json thresholds

    var tier, confidence, reasoning string

    switch {
    case m.TotalFiles < 5 && m.TotalLines < 500:
        tier = "haiku"
        confidence = "high"
        reasoning = fmt.Sprintf("Small scope: %d files, %d lines", m.TotalFiles, m.TotalLines)

    case m.TotalFiles <= 15 && m.TotalLines < 2000:
        tier = "sonnet"
        confidence = "high"
        reasoning = fmt.Sprintf("Medium scope: %d files, %d lines", m.TotalFiles, m.TotalLines)

    case m.TotalFiles > 15 || m.EstimatedTokens > 50000:
        tier = "external"
        confidence = "high"
        reasoning = fmt.Sprintf("Large scope: %d files, ~%d tokens - recommend gemini-slave mapper first",
            m.TotalFiles, m.EstimatedTokens)

    default:
        tier = "sonnet"
        confidence = "medium"
        reasoning = "Moderate scope"
    }

    // Adjust for complexity signals we CAN detect
    if m.FilesOver500Lines > 3 && tier == "haiku" {
        tier = "sonnet"
        reasoning += "; multiple large files detected"
    }

    return &RoutingRecommendation{
        RecommendedTier:     tier,
        Confidence:          confidence,
        Reasoning:           reasoning,
        ClarificationNeeded: nil,
    }
}
```

---

## 5. Gemini Delegation (WITH VALIDATION)

### 5.1 Schema Validation

```go
// cmd/gogent-scout/gemini_delegate.go

type GeminiScoutOutput struct {
    SchemaVersion string       `json:"schema_version"`
    ScoutReport   *ScoutReport `json:"scout_report"`
}

func validateGeminiOutput(data []byte) (*ScoutReport, error) {
    var output GeminiScoutOutput
    if err := json.Unmarshal(data, &output); err != nil {
        return nil, fmt.Errorf("invalid JSON from gemini-slave: %w", err)
    }

    // Schema version check
    if output.SchemaVersion != "1.0" && output.SchemaVersion != "" {
        // Allow empty for backwards compatibility, but log warning
        if output.SchemaVersion != "" {
            return nil, fmt.Errorf("unsupported schema version: %s", output.SchemaVersion)
        }
    }

    // Validate required fields
    if output.ScoutReport == nil {
        return nil, fmt.Errorf("missing scout_report in gemini output")
    }
    if output.ScoutReport.ScopeMetrics == nil {
        return nil, fmt.Errorf("missing scope_metrics in scout_report")
    }
    if output.ScoutReport.RoutingRecommendation == nil {
        return nil, fmt.Errorf("missing routing_recommendation in scout_report")
    }

    // Mark backend
    output.ScoutReport.Backend = "gemini"
    output.ScoutReport.SchemaVersion = "1.0"

    // Ensure complexity signals marked as available
    if output.ScoutReport.ComplexitySignals != nil {
        output.ScoutReport.ComplexitySignals.Available = true
    }

    return output.ScoutReport, nil
}
```

### 5.2 Delegation with Fallback

```go
func delegateToGemini(target, instruction string) (*ScoutReport, error) {
    // Build file list
    fileList, err := generateFileList(target)
    if err != nil {
        return nil, fmt.Errorf("failed to generate file list: %w", err)
    }

    // Execute gemini-slave with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "gemini-slave", "scout", instruction)
    cmd.Stdin = strings.NewReader(fileList)

    output, err := cmd.Output()
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, fmt.Errorf("gemini-slave timed out after 30s")
        }
        return nil, fmt.Errorf("gemini-slave failed: %w", err)
    }

    // Validate and parse output
    return validateGeminiOutput(output)
}

func generateFileList(target string) (string, error) {
    var files []string
    err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return nil
        }
        if _, ok := SupportedExtensions[filepath.Ext(path)]; ok {
            files = append(files, path)
        }
        return nil
    })
    if err != nil {
        return "", err
    }
    return strings.Join(files, "\n"), nil
}
```

---

## 6. Synthetic Fallback (GRACEFUL DEGRADATION)

```go
// cmd/gogent-scout/fallback.go

func generateSyntheticReport(target string, primaryErr, fallbackErr error) *ScoutReport {
    // Quick file count - absolute minimum
    fileCount := 0
    filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
        if err == nil && !d.IsDir() {
            if _, ok := SupportedExtensions[filepath.Ext(path)]; ok {
                fileCount++
            }
        }
        return nil
    })

    // Conservative estimates
    estimatedLines := fileCount * 100
    estimatedTokens := estimatedLines * 10

    warnings := []string{
        fmt.Sprintf("Primary backend failed: %v", primaryErr),
    }
    if fallbackErr != nil {
        warnings = append(warnings, fmt.Sprintf("Fallback backend failed: %v", fallbackErr))
    }
    warnings = append(warnings, "Using synthetic metrics - review recommended tier manually")

    return &ScoutReport{
        SchemaVersion: "1.0",
        Backend:       "synthetic_fallback",
        Target:        target,
        Timestamp:     time.Now().Format(time.RFC3339),
        ScopeMetrics: &ScopeMetrics{
            TotalFiles:      fileCount,
            TotalLines:      estimatedLines,
            EstimatedTokens: estimatedTokens,
            Languages:       []string{"unknown"},
            FileTypes:       map[string]int{},
        },
        ComplexitySignals: &ComplexitySignals{
            Available: false,
            Note:      "Scout backends unavailable",
        },
        RoutingRecommendation: &RoutingRecommendation{
            RecommendedTier:     "sonnet", // Safe default
            Confidence:          "low",
            Reasoning:           "Scout failure - using conservative sonnet tier",
            ClarificationNeeded: nil,
        },
        KeyFiles: []KeyFile{},
        Warnings: warnings,
    }
}
```

---

## 7. Main Orchestration with Fallback Chain

```go
// cmd/gogent-scout/main.go

func runScout(target, instruction string) (*ScoutReport, error) {
    // 1. Select primary backend
    backend, _ := selectBackend(target)

    var report *ScoutReport
    var primaryErr, fallbackErr error

    // 2. Try primary backend
    switch backend {
    case "native":
        scout := &NativeScout{Target: target, Instruction: instruction}
        report, primaryErr = scout.Run()

    case "gemini":
        report, primaryErr = delegateToGemini(target, instruction)
    }

    // 3. Primary succeeded
    if primaryErr == nil {
        return report, nil
    }

    // 4. Try fallback (opposite backend)
    log.Printf("Primary backend (%s) failed: %v, trying fallback", backend, primaryErr)

    switch backend {
    case "native":
        // Native failed, try Gemini
        report, fallbackErr = delegateToGemini(target, instruction)
        if fallbackErr == nil {
            report.Warnings = append(report.Warnings,
                fmt.Sprintf("Native scout failed (%v), used Gemini", primaryErr))
            return report, nil
        }

    case "gemini":
        // Gemini failed, try native
        scout := &NativeScout{Target: target, Instruction: instruction}
        report, fallbackErr = scout.Run()
        if fallbackErr == nil {
            report.Backend = "native_fallback"
            report.Warnings = append(report.Warnings,
                fmt.Sprintf("Gemini scout failed (%v), used native", primaryErr))
            return report, nil
        }
    }

    // 5. Both failed - synthetic fallback
    log.Printf("Both backends failed, generating synthetic report")
    return generateSyntheticReport(target, primaryErr, fallbackErr), nil
}

func main() {
    if len(os.Args) < 3 {
        fmt.Fprintf(os.Stderr, "Usage: gogent-scout <target> \"<instruction>\"\n")
        os.Exit(1)
    }

    target := os.Args[1]
    instruction := os.Args[2]

    // Handle piped input
    if target == "-" {
        // Read file list from stdin
        scanner := bufio.NewScanner(os.Stdin)
        var files []string
        for scanner.Scan() {
            files = append(files, scanner.Text())
        }
        target = filepath.Dir(files[0]) // Use directory of first file
    }

    // Validate target exists
    if _, err := os.Stat(target); os.IsNotExist(err) {
        fmt.Fprintf(os.Stderr, "Error: target not found: %s\n", target)
        os.Exit(2)
    }

    // Run scout
    report, err := runScout(target, instruction)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    // Output JSON
    outputPath := os.Getenv("SCOUT_OUTPUT")
    if outputPath == "" {
        // Check for --output flag
        for i, arg := range os.Args {
            if strings.HasPrefix(arg, "--output=") {
                outputPath = strings.TrimPrefix(arg, "--output=")
            } else if arg == "--output" && i+1 < len(os.Args) {
                outputPath = os.Args[i+1]
            }
        }
    }

    data, _ := json.MarshalIndent(report, "", "  ")

    if outputPath != "" {
        // Atomic write
        tmpPath := outputPath + ".tmp"
        if err := os.WriteFile(tmpPath, data, 0644); err != nil {
            fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
            os.Exit(4)
        }
        if err := os.Rename(tmpPath, outputPath); err != nil {
            fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
            os.Exit(4)
        }
    }

    // Always also output to stdout
    fmt.Println(string(data))
}
```

---

## 8. Testing Requirements

### 8.1 Unit Tests

```go
// cmd/gogent-scout/main_test.go

func TestCountLines(t *testing.T) {
    // Create temp file, count lines, verify
}

func TestAggregateMetrics(t *testing.T) {
    files := []FileInfo{
        {Path: "a.go", Lines: 100, Language: "go"},
        {Path: "b.go", Lines: 200, Language: "go"},
        {Path: "c.py", Lines: 50, Language: "python"},
    }
    metrics := aggregateMetrics(files)

    assert.Equal(t, 3, metrics.TotalFiles)
    assert.Equal(t, 350, metrics.TotalLines)
    assert.Contains(t, metrics.Languages, "go")
    assert.Contains(t, metrics.Languages, "python")
}

func TestGenerateRecommendation(t *testing.T) {
    tests := []struct {
        name     string
        files    int
        lines    int
        wantTier string
    }{
        {"tiny", 3, 200, "haiku"},
        {"small", 10, 1000, "sonnet"},
        {"large", 25, 10000, "external"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            metrics := &ScopeMetrics{TotalFiles: tt.files, TotalLines: tt.lines}
            rec := (&NativeScout{}).generateRecommendation(metrics)
            assert.Equal(t, tt.wantTier, rec.RecommendedTier)
        })
    }
}
```

### 8.2 Integration Tests

```go
func TestSmallScopeUsesNative(t *testing.T) {
    // Create temp dir with 5 Go files
    tmpDir := createTestFiles(t, 5)
    defer os.RemoveAll(tmpDir)

    report, err := runScout(tmpDir, "Test instruction")
    require.NoError(t, err)
    assert.Equal(t, "native", report.Backend)
}

func TestLargeScopeUsesGemini(t *testing.T) {
    if testing.Short() {
        t.Skip("Requires gemini-slave")
    }

    tmpDir := createTestFiles(t, 30)
    defer os.RemoveAll(tmpDir)

    report, err := runScout(tmpDir, "Test instruction")
    require.NoError(t, err)
    assert.Equal(t, "gemini", report.Backend)
}

func TestGeminiFallbackToNative(t *testing.T) {
    // Mock gemini-slave to fail
    t.Setenv("PATH", "/nonexistent:"+os.Getenv("PATH"))

    tmpDir := createTestFiles(t, 30)
    defer os.RemoveAll(tmpDir)

    report, err := runScout(tmpDir, "Test instruction")
    require.NoError(t, err)
    assert.Equal(t, "native_fallback", report.Backend)
    assert.Contains(t, report.Warnings[0], "failed")
}

func TestBothBackendsFail(t *testing.T) {
    // Force both to fail by using unreadable target
    report, _ := runScout("/nonexistent/path", "Test")
    assert.Equal(t, "synthetic_fallback", report.Backend)
    assert.Equal(t, "low", report.RoutingRecommendation.Confidence)
}
```

### 8.3 Performance Benchmarks

```go
func BenchmarkNativeScout5Files(b *testing.B) {
    tmpDir := createTestFiles(b, 5)
    defer os.RemoveAll(tmpDir)

    scout := &NativeScout{Target: tmpDir, Instruction: "Benchmark"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := scout.Run()
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkNativeScout15Files(b *testing.B) {
    tmpDir := createTestFiles(b, 15)
    defer os.RemoveAll(tmpDir)

    scout := &NativeScout{Target: tmpDir, Instruction: "Benchmark"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = scout.Run()
    }
}

// Acceptance criteria (check in CI):
// BenchmarkNativeScout5Files: p50 < 50ms, p99 < 100ms
// BenchmarkNativeScout15Files: p50 < 100ms, p99 < 200ms
```

### 8.4 Golden File Tests

```
test/fixtures/scout/
├── small_go_project/          # 5 Go files
│   ├── main.go
│   ├── handler.go
│   └── ...
├── medium_mixed_project/      # 15 files, Go + Python
├── large_go_project/          # 30 Go files
└── expected/
    ├── small_go_project.json  # Expected native output
    ├── medium_mixed.json      # Expected native output
    └── large_go_project.json  # Expected gemini output (mocked)
```

---

## 9. Configuration Updates

### 9.1 routing-schema.json

Update the `scout_protocol` section:

```json
"scout_protocol": {
  "description": "Pre-routing reconnaissance via unified gogent-scout binary",
  "primary": "gogent-scout",
  "backends": {
    "native": {
      "threshold_score": 40,
      "capabilities": ["file_count", "line_count", "language_detection", "test_detection"],
      "latency_target_ms": 100
    },
    "gemini": {
      "min_score": 40,
      "capabilities": ["import_analysis", "dependency_graph", "semantic_complexity"],
      "model": "gemini-3-flash-preview"
    }
  },
  "fallback_chain": ["native", "gemini", "synthetic"],
  "invocation": "gogent-scout <target> \"<instruction>\"",
  "output_file": ".claude/tmp/scout_metrics.json",
  "deprecated": "Direct gemini-slave scout calls (use gogent-scout instead)"
}
```

### 9.2 agents-index.json

Add new entry (insert after haiku-scout):

```json
{
  "id": "gogent-scout",
  "name": "GOgent Smart Scout",
  "model": "hybrid",
  "tier": "hybrid",
  "category": "reconnaissance",
  "path": "cmd/gogent-scout",
  "triggers": [
    "assess scope",
    "scout",
    "how big is",
    "estimate complexity",
    "pre-route",
    "check size"
  ],
  "tools": ["Bash"],
  "invocation": "Bash: gogent-scout <target> \"<instruction>\"",
  "backends": {
    "native": "< 40 score (fast, basic metrics)",
    "gemini": ">= 40 score (semantic analysis)"
  },
  "state_files": {
    "scout_output": ".claude/tmp/scout_metrics.json"
  },
  "description": "Unified scout with smart backend routing. Native Go for small scopes, Gemini for large."
}
```

### 9.3 ARCHITECTURE.md Update

Add to Section 9.1 (Hook Binaries) or create new utility section:

```markdown
### 9.4 Scout Utilities

| Binary | Purpose | Backends | Output |
|--------|---------|----------|--------|
| `gogent-scout` | Unified pre-routing reconnaissance | native (Go), gemini (Flash) | JSON to stdout + .claude/tmp/scout_metrics.json |

**Usage:**
```bash
gogent-scout ./pkg "Assess auth module scope"
```

**Backend Selection:**
- Score < 40: Native Go scout (fast, basic metrics)
- Score >= 40: gemini-slave scout (Gemini 3 Flash, semantic analysis)
- Fallback chain: native → gemini → synthetic
```

---

## 10. Implementation Checklist for go-pro

### Phase 1: Core Binary
- [ ] Create `cmd/gogent-scout/` directory structure
- [ ] Implement `types.go` with ScoutReport, ScopeMetrics, etc.
- [ ] Implement `native_scout.go` with file counting and metrics
- [ ] Implement `main.go` with CLI parsing
- [ ] Add unit tests for native scout

### Phase 2: Gemini Integration
- [ ] Implement `gemini_delegate.go` with validation
- [ ] Implement `fallback.go` for synthetic reports
- [ ] Add integration tests for backend selection
- [ ] Add tests for fallback chain

### Phase 3: Performance & Polish
- [ ] Add benchmark tests
- [ ] Verify latency targets (p50 < 100ms native)
- [ ] Add golden file tests

### Phase 4: Documentation
- [ ] Update `routing-schema.json`
- [ ] Update `agents-index.json`
- [ ] Update `ARCHITECTURE.md`
- [ ] Update `/explore` skill to use gogent-scout

---

## 11. Success Criteria

| Metric | Target | Verification |
|--------|--------|--------------|
| Native scout latency | p50 < 100ms | Benchmark tests |
| Gemini delegation overhead | < 500ms additional | Integration tests |
| Output schema compliance | 100% | Golden file tests |
| Test coverage | > 85% | `go test -cover` |
| Fallback reliability | Always return JSON | Failure injection tests |

---

**End of Specification v2.0**

*Ready for go-pro Implementation*
