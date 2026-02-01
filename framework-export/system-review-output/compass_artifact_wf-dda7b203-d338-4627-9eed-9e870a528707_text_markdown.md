# Multi-Language Code Review Framework: 2024-2025 Best Practices Update

Modern development paradigms have shifted dramatically since 2023. **React 19** introduces automatic memoization via React Compiler, **TypeScript 5.5** infers type predicates automatically, **Go 1.22** fixes the notorious loop variable capture bug, **Python 3.12** brings clean generic syntax, and **R's S7** emerges as the new OOP standard. This comprehensive guide covers best practices, anti-patterns, and sharp edges for all technologies in your code review framework.

---

## React 19+ / React 18+ Best Practices

React 19, released December 2024, represents the most significant evolution since hooks. The **React Compiler** (formerly React Forget) automatically handles memoization by analyzing components and applying optimization at build time—eliminating most manual `useMemo`, `useCallback`, and `memo` calls. To benefit, components must follow the "Rules of React": pure rendering, immutable props/state, and side effects outside render.

### New hooks and patterns to embrace

React 19 introduces transformative hooks. **`use()`** reads promises and context during render with Suspense support—uniquely, it can be called conditionally. **`useOptimistic()`** enables optimistic UI updates before server confirmation. **`useFormStatus()`** and **`useActionState()`** streamline form handling with automatic pending/error state management.

```jsx
// Server Action with new form handling
'use server'
async function createUser(formData) {
  const result = await db.insert({ name: formData.get('name') });
  return result.error ? { error: result.error } : { success: true };
}

function Form() {
  const [state, formAction, isPending] = useActionState(createUser, {});
  return (
    <form action={formAction}>
      {state.error && <p className="error">{state.error}</p>}
      <input name="name" required />
      <button disabled={isPending}>{isPending ? 'Saving...' : 'Submit'}</button>
    </form>
  );
}
```

### Anti-patterns to flag in code review

| Anti-Pattern | Better Approach |
|--------------|-----------------|
| `useEffect` for derived state | Calculate during render |
| Effect chains updating state | Single event handler with all updates |
| Server state in Zustand/Redux | TanStack Query for server data |
| `'use server'` for Server Components | Only for Server Actions |
| Importing Server into Client components | Pass as children (composition) |

### State management recommendations

Use **TanStack Query** for server/async data, **Zustand** (~3KB) for simple global state, **Jotai** for fine-grained atomic state, and native **Context** only for simple prop drilling. Redux Toolkit remains valid for large enterprise apps requiring structured middleware.

### Testing patterns

Prefer **Vitest** for new Vite-based projects (10x faster than Jest). Use React Testing Library with accessibility-first queries (`getByRole` preferred over `getByTestId`). For Server Components, extract logic to pure functions for unit testing, and use Playwright/Cypress for E2E.

---

## TypeScript 5.x Best Practices

TypeScript 5.5 (June 2024) is a "blockbuster release" with **inferred type predicates**—the compiler now automatically infers `x is Type` from function bodies, eliminating boilerplate type guards. The `.filter()` method finally narrows types correctly.

```typescript
// Before 5.5: Required explicit type predicate
function isNumber(x: unknown): x is number { return typeof x === 'number'; }

// After 5.5: TypeScript infers automatically
const isNumber = (x: unknown) => typeof x === 'number';
// Inferred: (x: unknown) => x is number

// .filter() now works correctly!
const birds = items.filter(bird => bird !== undefined);
// Type: Bird[] (not (Bird | undefined)[] anymore)
```

### Critical features since 2023

**`NoInfer`** (5.4) prevents inference from specific positions—essential for generic functions with multiple inference candidates. **`using`** keyword (5.2) brings explicit resource management like C#'s using statement. **`satisfies`** operator validates type conformance without changing the inferred type, preserving literal types.

```typescript
// NoInfer prevents invalid defaults from widening type
function createStreetLight<C extends string>(
  colors: C[],
  defaultColor?: NoInfer<C>
) {}
createStreetLight(["red", "yellow", "green"], "blue");
// Error! "blue" not assignable to "red" | "yellow" | "green"

// satisfies preserves literal types
const routes = {
  HOME: { path: "/" },
  ABOUT: { path: "/about" }
} satisfies Record<string, { path: string }>;
routes.HOME.path; // Type: "/" (exact literal preserved!)
```

### Recommended tsconfig.json for 2025

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "verbatimModuleSyntax": true,
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "target": "es2022",
    "isolatedModules": true,
    "skipLibCheck": true
  }
}
```

Use `moduleResolution: "bundler"` for frontend apps with Vite/webpack (doesn't require extensions), but `"NodeNext"` for Node.js libraries to ensure compatibility.

### Anti-patterns to flag

- **`any` abuse**: Use `unknown` with type guards instead
- **Type assertions overuse**: Proper construction beats `as Type`
- **Single-use generics**: If type parameter appears once, you don't need it
- **Overly complex types**: If hover tooltips are unreadable, simplify

---

## Ink (React for CLIs) Best Practices

Ink is at **v6.6.0** (December 2025), used by Claude Code (Anthropic), Gemini CLI (Google), GitHub Copilot CLI, and Cloudflare Wrangler. It's now **ESM-only** (requires `"type": "module"` in package.json) with full **React 18** support—React 19 support is pending.

### Component architecture essentials

All text must be inside `<Text>` components—bare strings cause errors. Use `<Box>` for Flexbox layouts (it's `display: flex` by default). The `<Static>` component permanently renders items above the UI, ideal for logs and completed tasks that shouldn't re-render.

```tsx
import { render, Box, Text, Static, useInput, useApp } from 'ink';

const App = () => {
  const { exit } = useApp();
  const [logs, setLogs] = useState<string[]>([]);
  
  useInput((input, key) => {
    if (input === 'q' || (key.ctrl && input === 'c')) {
      exit();
    }
  });

  return (
    <>
      <Static items={logs}>
        {(log, i) => <Text key={i} dimColor>{log}</Text>}
      </Static>
      <Box borderStyle="round" borderColor="cyan" padding={1}>
        <Text>Press q to quit</Text>
      </Box>
    </>
  );
};
```

### Key hooks and patterns

| Hook | Purpose |
|------|---------|
| `useInput(handler, options?)` | Handle keyboard input |
| `useApp()` | Access `exit()` to unmount |
| `useFocus({ id, autoFocus })` | Make component focusable |
| `useFocusManager()` | Control focus: `focusNext()`, `focus(id)` |

Use `useInput` with `isActive` option to conditionally enable input handling based on focus state.

### Testing with ink-testing-library

```tsx
import { render } from 'ink-testing-library';

test('renders correctly', () => {
  const { lastFrame, stdin } = render(<MyComponent />);
  expect(lastFrame()).toContain('Expected text');
  
  stdin.write('a'); // Simulate input
  expect(lastFrame()).toContain('Updated text');
});
```

### Anti-patterns to avoid

- **Not wrapping text in `<Text>`** — causes runtime errors
- **Using `<Box>` inside `<Text>`** — Box cannot be nested in Text
- **Ignoring `isActive` in `useInput`** — causes multiple handlers to fire
- **Blocking main thread** — use async/await for I/O
- **Not handling Ctrl+C** — always provide clean exit handling

---

## Go Best Practices (1.21-1.23+)

Go 1.22 (February 2024) includes a **critical behavioral change**: the for loop variable bug is fixed—each iteration now creates new variables. Code that relied on the old behavior (rare) may break, but closures in loops now work correctly without explicit capture.

```go
// Go 1.22+: Each goroutine correctly gets its own copy
for _, v := range values {
    go func() {
        fmt.Println(v) // Now works correctly!
    }()
}
```

### New language features to adopt

**Range over integers** simplifies counting loops. **Enhanced HTTP routing** in `net/http` supports method matching and wildcards without third-party routers. **Structured logging with `slog`** replaces ad-hoc logging patterns.

```go
// Range over integers (Go 1.22)
for i := range 10 {
    fmt.Println(i) // Prints 0-9
}

// Enhanced HTTP routing (Go 1.22)
mux.HandleFunc("GET /posts/{id}", getPost)
mux.HandleFunc("POST /posts", createPost)
mux.HandleFunc("/files/{path...}", serveFiles) // Rest parameter

// Structured logging with slog (Go 1.21)
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("request processed",
    "method", "GET",
    "status", 200,
    "duration_ms", 158,
)
```

### Standard library additions (Go 1.21)

Use the new **`slices`** and **`maps`** packages for common operations: `slices.Sort()`, `slices.Contains()`, `slices.Clone()`, `maps.Clone()`, `maps.Equal()`. The built-in **`min`**, **`max`**, and **`clear`** functions work with any ordered/comparable types.

### Error handling patterns

```go
// Error wrapping with context
if err != nil {
    return fmt.Errorf("failed to open config %s: %w", path, err)
}

// Use errors.Is for sentinel errors
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}

// Use errors.As for error types
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    fmt.Println("Failed path:", pathErr.Path)
}
```

### Concurrency best practices

Use **`errgroup`** for coordinated goroutine error handling with automatic cancellation. Always pass **context** as the first parameter and check for cancellation. Use `signal.NotifyContext` for graceful shutdown.

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10) // Limit concurrent goroutines

for _, url := range urls {
    g.Go(func() error {
        return fetch(ctx, url)
    })
}

if err := g.Wait(); err != nil {
    return err // First error from any goroutine
}
```

### Anti-patterns to flag

- **Panic for recoverable errors** — only panic for programmer errors
- **Forgetting `defer cancel()`** for context.WithTimeout/Cancel
- **Large interfaces** — prefer small, focused interfaces
- **`utils/` packages** — group by domain, not by type

---

## Go Bubbletea + Cobra CLI Best Practices

### Bubbletea (Elm Architecture for TUIs)

Bubbletea follows the Model-Update-View pattern. **Critical rule**: Keep `Update()` and `View()` fast—offload expensive operations to commands.

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q":
            return m, tea.Quit
        }
    case dataLoadedMsg:
        m.data = msg.data
    }
    return m, nil
}

// DON'T do expensive work in Update(). Use commands:
func loadData() tea.Cmd {
    return func() tea.Msg {
        data, _ := fetchFromAPI() // Runs in separate goroutine
        return dataLoadedMsg{data: data}
    }
}
```

### Lipgloss styling patterns

Use `lipgloss.Height()` and `lipgloss.Width()` for responsive layouts—never hard-code dimensions.

```go
func (m model) View() string {
    header := headerStyle.Render("Header")
    footer := footerStyle.Render("Footer")
    
    // Calculate remaining space dynamically
    contentHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
    content := contentStyle.Height(contentHeight).Render("Content")
    
    return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}
```

### Cobra CLI patterns

Use `RunE` (not `Run`) for commands that can fail. Integrate with Viper for configuration, but **always read from Viper, not flag variables** after binding.

```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    RunE: func(cmd *cobra.Command, args []string) error {
        port := viper.GetInt("port") // NOT the flag variable!
        return startServer(port)
    },
}

// Flag groups (2024 feature)
rootCmd.MarkFlagsRequiredTogether("username", "password")
rootCmd.MarkFlagsMutuallyExclusive("json", "yaml")
```

### Sharp edges

- **Message ordering**: Results from concurrent commands are not guaranteed ordered
- **Pointer receivers**: Both work, but pointer receivers risk race conditions if misused
- **Viper binding**: `BindPFlag` flows flag→Viper only, not bidirectional
- **Testing**: Use `teatest` for Bubbletea; set `lipgloss.SetColorProfile(termenv.Ascii)` in CI

---

## Python Best Practices (3.11-3.13+)

Python 3.12 introduces **PEP 695: Type Parameter Syntax**—a cleaner way to write generics without importing TypeVar.

```python
# OLD (pre-3.12)
from typing import TypeVar, Generic
T = TypeVar('T')
class Stack(Generic[T]):
    def push(self, item: T) -> None: ...

# NEW (3.12+)
class Stack[T]:
    def push(self, item: T) -> None: ...

# Type aliases with new syntax
type Point = tuple[float, float]
type Vector[T] = list[T]
```

### Structured concurrency with TaskGroups (3.11+)

**Always prefer `TaskGroup` over `asyncio.gather()`** for proper error handling and automatic cancellation.

```python
async def fetch_all(urls: list[str]) -> list[dict]:
    async with asyncio.TaskGroup() as tg:
        tasks = [tg.create_task(fetch(url)) for url in urls]
    return [t.result() for t in tasks]
    # If any task fails, all others are cancelled automatically
```

### Package management: uv is the new standard

**uv** from Astral (Ruff creators) is 10-100x faster than pip and replaces pip, pip-tools, virtualenv, pipx, and pyenv.

```bash
uv init --package myproject  # src layout (recommended)
uv add requests httpx
uv add --dev pytest ruff mypy
uv run pytest
uvx ruff check .  # Run tools without installing
```

### Pydantic v2 patterns

```python
from pydantic import BaseModel, ConfigDict, Field, field_validator
from typing import Annotated

class User(BaseModel):
    model_config = ConfigDict(
        from_attributes=True,      # ORM mode
        extra='forbid',            # Fail on unknown fields
        str_strip_whitespace=True,
    )
    
    email: Annotated[str, Field(pattern=r'^[^@\s]+@[^@\s]+\.[^@\s]+$')]
    
    @field_validator('email')
    @classmethod
    def validate_email(cls, v: str) -> str:
        return v.lower()
```

### Anti-patterns to flag

- **`asyncio.gather()` without error handling** — use TaskGroup
- **Mutable default arguments** — use `None` and create inside function
- **`from typing import List, Dict`** — use built-in `list`, `dict`
- **`Optional[X]`** — use `X | None` (3.10+)
- **Black + isort + Flake8** — use Ruff (single tool, 10-100x faster)

---

## PySide6/Qt Best Practices

PySide6 6.9+ supports **pyproject.toml** (replacing deprecated `.pyproject`). The **QtAsyncio module** provides native asyncio integration.

### Always use @Slot decorator

Essential for thread-safe signal handling and QML compatibility.

```python
from PySide6.QtCore import QObject, Signal, Slot

class Worker(QObject):
    finished = Signal()
    progress = Signal(int)
    
    @Slot()
    def run(self):
        for i in range(100):
            self.progress.emit(i)
        self.finished.emit()
```

### Correct threading pattern (Worker pattern)

```python
class MainWindow(QMainWindow):
    def start_work(self):
        self.thread = QThread()
        self.worker = Worker()
        self.worker.moveToThread(self.thread)
        
        self.thread.started.connect(self.worker.run)
        self.worker.finished.connect(self.thread.quit)
        self.worker.finished.connect(self.worker.deleteLater)
        self.thread.finished.connect(self.thread.deleteLater)
        
        self.thread.start()
```

### Anti-patterns to avoid

- **Subclassing QThread and implementing logic in run()** — use Worker pattern
- **Accessing GUI from worker threads** — emit signals instead
- **Using `QApplication.processEvents()` in loops** — blocks event loop
- **Forgetting `deleteLater()`** — causes memory leaks

---

## R Best Practices (4.3+)

R 4.4.0 introduces the **`%||%` operator** (null coalescing) and enhanced native pipe placeholder support.

### S7: The new OOP system

S7 is designed to supersede S3 and S4, developed by the R Consortium OOP Working Group. Use it for new projects.

```r
library(S7)

Range <- new_class("Range",
  properties = list(
    start = class_double,
    end = class_double
  ),
  validator = function(self) {
    if (self@end < self@start) "@end must be >= @start"
  }
)

x <- Range(start = 1, end = 10)
x@start  # Access with @ (like S4)
```

### Modern tidyverse patterns (dplyr 1.1+)

**`.by` argument** replaces `group_by() |> ... |> ungroup()`:

```r
# Modern approach
expenses |> summarise(total = sum(cost), .by = region)

# join_by() for non-equi joins
transactions |> inner_join(companies, join_by(company == id, year >= since))

# reframe() for multi-row results (summarise() now warns)
starwars |> reframe(quantile_df(height), .by = homeworld)
```

### Native pipe vs magrittr

| Feature | Native `|>` | magrittr `%>%` |
|---------|-------------|----------------|
| Placeholder | `_` (requires named arg) | `.` (flexible) |
| Performance | Faster (syntax transform) | Slower (function call) |
| Parentheses | Required | Optional |

### Anti-patterns to flag

- **`across()` without function** — use `pick()` instead
- **`summarise()` with multi-row results** — use `reframe()`
- **Growing vectors in loops** — always pre-allocate
- **`group_by()` without `ungroup()`** — use `.by` argument

---

## R Shiny + Golem Best Practices

### Shiny 1.8+ key features

**ExtendedTask** (1.8.1) enables truly non-blocking async operations—the session remains responsive during long-running tasks.

```r
task <- ExtendedTask$new(function(x) {
  future_promise({ expensive_computation(x) })
})
observeEvent(input$go, { task$invoke(input$x) })
output$result <- renderText({ task$result() })
```

### Modern reactive patterns

Prefer `bindCache()` and `bindEvent()` over legacy `observeEvent()`:

```r
# Modern composable pattern
data <- reactive({
  expensive_query(input$year, input$region)
}) |> bindCache(input$year, input$region) |> bindEvent(input$go)
```

### Module communication

Use the **"Stratégie du petit r"**: pass `reactiveValues()` between modules:

```r
r <- reactiveValues(shared_data = NULL)
mod_A_server("modA", r = r)
mod_B_server("modB", r = r)
```

### Golem framework patterns

Golem app = R Package. Use `golem::add_module("name", with_test = TRUE)` to create properly namespaced modules. Configuration lives in `inst/golem-config.yml`:

```yaml
default:
  app_prod: no
  MONGODBURL: localhost
production:
  app_prod: yes
  MONGODBURL: mongo.production.com
```

### Testing with shinytest2

shinytest2 replaces deprecated shinytest (which used PhantomJS):

```r
test_that("app works", {
  app <- AppDriver$new()
  app$set_inputs(name = "Test User")
  app$click("submit")
  app$expect_values()  # Snapshot testing
})
```

### Anti-patterns to flag

- **Forgetting `ns()` in modules** — silent failure, IDs don't connect
- **Using `callModule()`** — deprecated, use `moduleServer()`
- **`observe()` modifying values it depends on** — infinite loops
- **Editing NAMESPACE manually** — always auto-generated

---

## Configuration recommendations summary

| Technology | Key Config |
|------------|------------|
| React | Enable React Compiler in Babel; Vite for dev |
| TypeScript | `strict: true`, `noUncheckedIndexedAccess: true`, `moduleResolution: "NodeNext"` |
| Ink | `"type": "module"` in package.json; React 18 (not 19 yet) |
| Go | `go 1.22` minimum; enable PGO for performance |
| Python | pyproject.toml with uv; Ruff for linting/formatting |
| PySide6 | Use pyproject.toml (6.9+); @Slot on all signal handlers |
| R | `testthat 3e`; S7 for new OOP; renv for dependencies |
| Shiny/Golem | shinytest2; bslib with Bootstrap 5; ExtendedTask for async |

---

## Conclusion

The 2024-2025 landscape shows clear trends: **automatic optimization** (React Compiler, TypeScript inference), **structured concurrency** (Python TaskGroups, Go errgroup), **simpler syntax** (Python generics, R's `.by` argument), and **unified tooling** (uv for Python, Ruff, S7 for R OOP). Code review frameworks should prioritize flagging deprecated patterns while encouraging adoption of these modern idioms that improve both developer experience and application robustness.