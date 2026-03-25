# Phase 10b: Visual Foundation

> **Tickets:** TUI-044, TUI-045, TUI-046, TUI-047
> **Status:** All complete
> **Packages:** `config/`, `util/`, `model/`
> **Purpose:** Establish the design language that every subsequent Phase 10 feature depends on.

---

## TUI-044: Semantic Color System

### Purpose

Replace ad-hoc color usage with a consistent semantic color API. Every component uses the same colors for the same meaning: red for errors, green for success, yellow for warnings, cyan for info.

### Design Decision

**Method-based API on Theme** (`theme.ErrorStyle()`) was chosen over a standalone `semantic` package. The rationale: simpler, co-located with color definitions, and components already receive the Theme struct.

| Alternative | Why Not Chosen |
|-------------|----------------|
| Standalone `semantic.ErrorStyle(theme)` | More testable but adds indirection |
| Global function registry | Too magical, harder to trace |

### Implementation

Extended `config/theme.go` with:

- **ThemeVariant enum:** Dark, Light, HighContrast
- **NewTheme() factory:** Creates theme from variant
- **5 semantic style methods:** `ErrorStyle()`, `WarningStyle()`, `SuccessStyle()`, `InfoStyle()`, `DangerStyle()`
- **HighContrast colors:** WCAG AA hex codes — error=#FF0000, success=#00AA00, warning=#FFAA00, info=#0088FF

`DefaultTheme()` is unchanged — fully backward compatible with Phase 1-9 code.

### Usage

```go
theme := config.NewTheme(config.ThemeVariantDark)
errorText := theme.ErrorStyle().Render("Connection failed")
successText := theme.SuccessStyle().Render("Build passed")
```

### Testing

- 100% statement coverage
- Table-driven tests for each ThemeVariant
- All 23 TUI packages pass after change

---

## TUI-045: Icon Library with Unicode/ASCII Fallback

### Purpose

Standardize icon usage across all components. Provide Unicode icons for modern terminals with automatic ASCII fallback for minimal terminals (TTY, SSH without UTF-8).

### Design Decision

**IconSet struct** with 12 fields, dual-mode (Unicode + ASCII). Theme.Icons() returns the appropriate set based on UseASCII bool.

### Implementation

- **IconSet struct:** 12 fields — Running, Complete, Error, Pending, Cancelled, Paused, Info, Warning, Success, Search, Settings, Arrow
- **UnicodeIcons:** Predefined set with symbols (spinners, checkmarks, X marks, etc.)
- **ASCIIIcons:** Safe fallback set using standard ASCII characters
- **Theme.Icons():** Returns UnicodeIcons or ASCIIIcons based on `UseASCII` flag
- Old `Icon*` rune constants kept with `// Deprecated:` comments for backward compatibility

### Usage

```go
icons := theme.Icons()
fmt.Println(icons.Complete, "Task done")    // ✓ Task done (Unicode)
fmt.Println(icons.Error, "Build failed")     // ✗ Build failed (Unicode)
// With UseASCII: true → [x] Build failed
```

### Testing

- 100% coverage
- Tests verify both Unicode and ASCII variants
- Integration: used by TUI-047 (error formatting), TUI-063 (breadcrumbs), TUI-066 (modal headers), TUI-068 (dashboard sections)

---

## TUI-046: Theme Switching Infrastructure

### Purpose

Enable runtime theme switching between Dark, Light, and HighContrast variants. Persist the user's theme choice across sessions.

### Design Decision

**Theme stored in sharedState** via `activeTheme *config.Theme` pointer. Components with `HandleMsg` pointer receiver read from shared state. Theme variant persisted in SessionData as `int` (avoids config→session import cycle).

| Approach | Choice | Reason |
|----------|--------|--------|
| Theme storage | sharedState pointer | Survives Bubbletea value-copy |
| Persistence | SessionData.ThemeVariant as int | Avoids import cycle |
| Propagation | ThemeChangedMsg → handleThemeChanged() | Standard Bubbletea message pattern |

### Implementation

- `activeTheme *config.Theme` + `themeVariant` fields in sharedState
- `SetTheme(*AppModel)` — pointer receiver for mutation
- `Theme(AppModel)` — value receiver with nil fallback to DefaultTheme
- `ThemeChangedMsg` triggers re-render with new theme
- `SessionData.ThemeVariant` persisted as integer

### Usage

```go
// Switching theme (triggered by settings panel)
msg := config.ThemeChangedMsg{Variant: config.ThemeVariantHighContrast}
// → handleThemeChanged() updates sharedState.activeTheme
// → All components receive new theme on next View() cycle
```

### Keyboard Shortcuts

None (triggered via settings panel, see [[phase10-settings-accessibility#TUI-050]]).

### Testing

- 8 tests in `theme_switching_test.go` + 3 in `persistence_test.go`
- model/ 88.9% coverage
- Verified: theme persists across session save/restore, nil fallback works

---

## TUI-047: Contextual Error Message Formatting

### Purpose

Provide consistent, theme-aware error and warning formatting across the TUI. Classify errors into categories for appropriate visual treatment.

### Design Decision

New `util/errors.go` file with ErrorDisplay struct. Uses theme semantic styles and icons for rendering. Classification via case-insensitive string matching.

### Implementation

- **ErrorDisplay struct:** Wraps formatted error output
- **FormatError(theme, msg):** Renders with `theme.ErrorStyle()` + `Icons().Error`
- **FormatWarning(theme, msg):** Renders with `theme.WarningStyle()` + `Icons().Warning`
- **ClassifyError(msg):** 4 categories — Network, Permission, Timeout, Unknown
- Case-insensitive `strings.Contains` matching for classification

### Usage

```go
display := util.FormatError(theme, "Connection refused: server unavailable")
// Renders: ✗ Connection refused: server unavailable (red, themed)

category := util.ClassifyError("connection timed out after 30s")
// Returns: ErrorCategoryTimeout
```

### Testing

- 10 table-driven tests covering all error categories
- 94% util coverage
- Completes Track A (visual foundation chain: TUI-044 → 045 → 047)

---

## Cross-References

- **Consumed by:** [[phase10-settings-accessibility]] (TUI-048–051 use semantic colors), [[phase10-parity-features]] (TUI-055 task board uses semantic colors for badges), [[phase10-polish-animation]] (TUI-066 rich modals use theme icons)
- **Depends on:** TUI-043 (app.go decomposition, see [[phase10-overview]])

---

_Part of [[phase10-overview|Phase 10 UX Overhaul]]. Generated by TUI-069._
