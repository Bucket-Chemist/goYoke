# Agent Guidelines for Rust Code Quality

This document provides guidelines for maintaining high-quality Rust code. These rules MUST be followed by all AI coding agents and contributors.

## Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:
- Maximizing algorithmic big-O efficiency for memory and runtime
- Leveraging zero-cost abstractions (iterators, traits, monomorphization)
- Following proper Rust idioms (ownership, borrowing, error handling)
- No extra code beyond what is absolutely necessary to solve the problem the user provides (i.e., no technical debt)
- No `.unwrap()` or `.expect()` in any production code path

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

---

## Rust Edition and Modern Features

**MUST** target edition 2024 (Rust 1.85+) for new projects.

### Edition 2024 Changes (CRITICAL)
- **RPIT lifetime capture**: `impl Trait` return types now capture all in-scope lifetimes—drop `+ '_` annotations
- **Use `use<>` blocks** to limit lifetime captures when needed: `-> impl Trait + use<'a>`
- **Temporary scoping**: temporaries in tail expressions and `if let` conditions now drop before locals
- **`unsafe` attributes**: `#[no_mangle]` and `#[link_section]` require `unsafe(...)` wrapper
- **`unsafe_op_in_unsafe_fn`**: warn-by-default—use explicit `unsafe {}` blocks inside `unsafe fn`
- **Reserved keyword**: `gen` is reserved—use `r#gen` for existing identifiers
- **Prelude additions**: `Future` and `IntoFuture` added to prelude
- **`std::env::set_var`** is now `unsafe`

### Stable Features to Adopt
- **Async fn in traits** (1.75+): native `async fn` in trait definitions, no proc macro needed
- **Async closures** (1.85+): `AsyncFn`, `AsyncFnMut`, `AsyncFnOnce` traits
- **`let` chains in `if let`** (1.64+): `if let Some(x) = a && let Some(y) = b { ... }`
- **GATs** (1.65+): `type Item<'a>` in trait definitions for lending iterators

**Example - Modern Trait Patterns (Edition 2024):**
```rust
// Native async fn in traits—no #[async_trait] needed
trait DataStore {
    async fn get(&self, key: &str) -> Result<Option<Vec<u8>>>;
    async fn set(&self, key: &str, value: &[u8]) -> Result<()>;
}

// RPIT captures all lifetimes automatically in 2024
fn items(&self) -> impl Iterator<Item = &str> {
    self.data.iter().map(|s| s.as_str())
}

// Limit captures with use<> when needed
fn filtered(&self) -> impl Iterator<Item = &str> + use<'_> {
    self.data.iter().filter(|s| !s.is_empty()).map(|s| s.as_str())
}
```

---

## Preferred Tools

### Package Management and Build
- **MUST** use `cargo` as build system and package manager
- **MUST** use `[workspace.dependencies]` for version centralization in workspaces
- **MUST** commit `Cargo.lock` for binaries (not for libraries)
- **MUST** specify `rust-version` (MSRV) in `Cargo.toml`
```bash
cargo init --name myproject     # Create new project
cargo add tokio --features full # Add dependency
cargo add --dev proptest        # Add dev dependency
cargo build --release           # Release build
cargo nextest run               # Run tests (preferred)
```

### Development Tools
- **Clippy** for linting (`cargo clippy --all-targets --all-features -- -D warnings`)
- **rustfmt** for formatting (`cargo fmt`)
- **cargo-nextest** for testing (process-per-test isolation, retries, parallel Miri)
- **cargo-deny** for supply chain security (licenses, vulnerabilities, banned crates)
- **cargo-audit** for vulnerability scanning
- **cargo-machete** for unused dependency detection
- **Miri** for detecting undefined behavior in unsafe code
- **tracing** for structured logging (replaces `log` crate)
- **just** as a modern Makefile alternative

### Preferred Crates
| Purpose | Crate | Notes |
|---------|-------|-------|
| Async runtime | `tokio` | Use `features = ["full"]` for applications |
| HTTP server | `axum` | Prefer over actix-web for new projects |
| HTTP client | `reqwest` | With `rustls-tls` feature, not native-tls |
| Serialization | `serde` + `serde_json` | Derive macros, `deny_unknown_fields` |
| CLI parsing | `clap` | With derive feature |
| Error (library) | `thiserror` v2 | Typed, matchable errors |
| Error (app) | `anyhow` | Ad-hoc context with `.context()` |
| Error (CLI) | `miette` | Compiler-quality diagnostics |
| Logging | `tracing` | Structured, async-aware |
| TUI | `ratatui` | Replaces tui-rs |
| Data | `polars` | DataFrame operations |
| Testing | `proptest` | Property-based testing |
| TLS | `rustls` | No system OpenSSL dependency |
| Allocator | `mimalloc` | Replace musl default for perf |
| Async utils | `futures` | `FutureExt` combinators for zero-cost future composition |

---

## Code Style and Formatting

- **MUST** use meaningful, descriptive variable and function names
- **MUST** run `cargo fmt` before every commit
- **MUST** follow Rust naming conventions strictly (see below)
- **NEVER** use emoji or unicode emulating emoji except in tests
- Use `snake_case` for functions/variables/modules, `PascalCase` for types/traits, `SCREAMING_SNAKE_CASE` for constants
- Keep items ordered: types → traits → impls → functions → tests

### Naming Conventions
| Element | Convention | Example |
|---------|-----------|---------|
| Crate | snake_case | `my_crate` |
| Module | snake_case, single word preferred | `config`, `routing` |
| Type/Struct/Enum | PascalCase | `ClientConfig`, `RouteTable` |
| Trait | PascalCase, adjective/capability | `Readable`, `Configurable` |
| Function | snake_case | `parse_input`, `build_client` |
| Method getter | no `get_` prefix | `fn name(&self) -> &str` |
| Method builder | `with_` prefix | `fn with_timeout(self, d: Duration) -> Self` |
| Fallible method | `try_` prefix | `fn try_parse(s: &str) -> Result<Self>` |
| Constant | SCREAMING_SNAKE_CASE | `MAX_RETRIES`, `DEFAULT_PORT` |
| Lifetime | short lowercase | `'a`, `'ctx`, `'input` |
| Type param | single uppercase | `T`, `K`, `V` |
| Receiver | `&self`, `&mut self`, `self` | Never `this` |

---

## Type System

### Requirements
- **MUST** leverage the type system to make invalid states unrepresentable
- **MUST** use `pub(crate)` for internal items—minimize public API surface
- **MUST** derive standard traits where applicable: `Debug`, `Clone`, `PartialEq`
- **NEVER** derive `Default` on configuration structs (creates invalid zero-states like port 0)
- **NEVER** derive `Debug` on types with sensitive fields (leaks passwords into logs)
- **MUST** prefer `&str` over `&String` and `&[T]` over `&Vec<T>` in function parameters

### Newtype Pattern
```rust
// Wrap primitive types to prevent misuse
pub struct UserId(u64);
pub struct OrderId(u64);

// Now the compiler prevents mixing them up
fn get_order(user: UserId, order: OrderId) -> Result<Order> { ... }
```

### Builder Pattern
```rust
pub struct ServerConfig {
    host: String,
    port: u16,
    max_connections: usize,
}

impl ServerConfig {
    pub fn builder() -> ServerConfigBuilder {
        ServerConfigBuilder::default()
    }
}

pub struct ServerConfigBuilder {
    host: String,
    port: u16,
    max_connections: usize,
}

impl Default for ServerConfigBuilder {
    fn default() -> Self {
        Self {
            host: "127.0.0.1".into(),
            port: 8080,
            max_connections: 1024,
        }
    }
}

impl ServerConfigBuilder {
    pub fn host(mut self, host: impl Into<String>) -> Self {
        self.host = host.into();
        self
    }

    pub fn port(mut self, port: u16) -> Self {
        self.port = port;
        self
    }

    pub fn build(self) -> ServerConfig {
        ServerConfig {
            host: self.host,
            port: self.port,
            max_connections: self.max_connections,
        }
    }
}
```

### Enum State Machines
```rust
// Use enums to model states—compiler enforces valid transitions
enum Connection {
    Disconnected,
    Connecting { attempt: u32 },
    Connected { session: Session },
}
```

---

## Error Handling

- **NEVER** use `.unwrap()` or `.expect()` in production code paths
- **NEVER** use bare `?` without adding context at module boundaries
- **MUST** use `thiserror` for library error types, `anyhow` for application errors
- **MUST** implement `std::error::Error` for all custom error types
- **MUST** use `#[source]` or `#[from]` to preserve error chains

### Library Errors (thiserror)
```rust
#[derive(Debug, thiserror::Error)]
pub enum StorageError {
    #[error("key not found: {key}")]
    NotFound { key: String },

    #[error("connection failed after {attempts} attempts")]
    ConnectionFailed {
        attempts: u32,
        #[source]
        source: std::io::Error,
    },

    #[error("serialization failed")]
    Serialization(#[from] serde_json::Error),
}
```

### Application Errors (anyhow)
```rust
use anyhow::{Context, Result};

fn load_config(path: &Path) -> Result<Config> {
    let raw = std::fs::read_to_string(path)
        .with_context(|| format!("reading config from {}", path.display()))?;
    let config: Config = toml::from_str(&raw)
        .context("parsing config TOML")?;
    config.validate()
        .context("validating config")?;
    Ok(config)
}
```

---

## Documentation

- **MUST** include doc comments for all public functions, types, traits, and methods
- **MUST** document errors with `# Errors` section
- **MUST** document panics with `# Panics` section (if any)
- **MUST** include `# Examples` with compilable doctests for complex APIs
- Keep comments up-to-date with code changes
```rust
/// Parses a duration string like "5s", "100ms", or "2m".
///
/// Supports suffixes: `ms` (milliseconds), `s` (seconds), `m` (minutes), `h` (hours).
///
/// # Errors
///
/// Returns [`ParseError::InvalidFormat`] if the string doesn't match
/// the expected pattern, or [`ParseError::Overflow`] if the value
/// exceeds `u64::MAX` milliseconds.
///
/// # Examples
///
/// ```
/// use mylib::parse_duration;
///
/// let d = parse_duration("5s")?;
/// assert_eq!(d, Duration::from_secs(5));
///
/// let d = parse_duration("100ms")?;
/// assert_eq!(d, Duration::from_millis(100));
/// # Ok::<(), mylib::ParseError>(())
/// ```
pub fn parse_duration(s: &str) -> Result<Duration, ParseError> { ... }
```

---

## Function Design

- **MUST** keep functions focused on a single responsibility
- **MUST** accept the most general type that works: `&str` not `&String`, `&[T]` not `&Vec<T>`
- **MUST** return concrete types from public APIs (avoid `impl Trait` in return position for libraries unless intentional)
- Limit function parameters to 5 or fewer—use a config struct beyond that
- Return early to reduce nesting
- Prefer iterators over manual loops where readability is maintained

---

## Struct and Trait Design

- **MUST** keep structs focused on a single responsibility
- **MUST** prefer composition over trait inheritance
- Use marker traits sparingly
- Implement `From`/`Into` for natural type conversions
- Use `impl Into<T>` parameters for ergonomic APIs

```rust
// Prefer composition
struct HttpClient {
    transport: Transport,
    auth: AuthProvider,
    retry: RetryPolicy,
}

// Trait objects for runtime polymorphism
trait Handler: Send + Sync {
    fn handle(&self, req: Request) -> Result<Response>;
}

// Generics for compile-time polymorphism (zero-cost)
fn process<H: Handler>(handler: &H, req: Request) -> Result<Response> {
    handler.handle(req)
}
```

---

## Concurrency and Async

### Tokio (I/O-bound tasks)
- **MUST** use `tokio::task::spawn_blocking` for file I/O and CPU-bound work
- **MUST** scope RAII guards (MutexGuard, etc.) before `.await` points
- **MUST** use `tokio::select!` with cancellation safety in mind
- **NEVER** call `block_on` inside an async context (panics or deadlocks)
- **NEVER** hold locks across `.await` points
```rust
use tokio::sync::Mutex;

async fn update_state(state: &Mutex<AppState>, data: Data) -> Result<()> {
    // Lock scoped before any .await
    let processed = {
        let guard = state.lock().await;
        guard.prepare(&data)
    };

    // Now safe to .await
    save_to_disk(processed).await?;
    Ok(())
}
```

### Rayon (CPU-bound tasks)
- **MUST** use `rayon` for CPU-bound parallel iteration
- **MUST** use `par_iter()` / `par_bridge()` for embarrassingly parallel work
- **NEVER** mix rayon and tokio without `spawn_blocking`
```rust
use rayon::prelude::*;

let results: Vec<_> = items
    .par_iter()
    .map(|item| expensive_computation(item))
    .collect();
```

### Concurrency Selection Guide
| Workload | Solution |
|----------|----------|
| I/O-bound (network, files) | `tokio` async tasks |
| CPU-bound (computation) | `rayon` parallel iterators |
| CPU in async context | `tokio::task::spawn_blocking` |
| Shared state in async | `tokio::sync::Mutex` (not `std::sync::Mutex`) |
| Channel communication | `tokio::sync::mpsc` / `broadcast` / `watch` |
| One-shot results | `tokio::sync::oneshot` |

### Async Future Size Optimization

Async functions generate state machines. Careless signatures inflate future size and binary weight.

**Eliminate unnecessary async state machines:**
```rust
// WRONG: async fn with no .await — generates dead state machine
async fn default_config() -> Config {
    Config::new()
}

// CORRECT: zero-cost ready future (Future is in 2024 prelude)
fn default_config() -> impl Future<Output = Config> {
    std::future::ready(Config::new())
}
```

**Eliminate async pass-through wrappers:**
```rust
// WRONG: wrapper state machine just to forward
impl Store for CachedStore {
    async fn get(&self, key: &str) -> Result<Option<Vec<u8>>> {
        self.inner.get(key).await
    }
}

// CORRECT: return inner future directly — zero-cost
impl Store for CachedStore {
    fn get(&self, key: &str) -> impl Future<Output = Result<Option<Vec<u8>>>> {
        self.inner.get(key)
    }
}

// With postamble: use FutureExt::map() instead of .await + transform
use futures::FutureExt;

fn get(&self, key: &str) -> impl Future<Output = Result<Option<Vec<u8>>>> {
    self.inner.get(key).map(|res| res.map(|opt| opt.map(|v| v.to_vec())))
}
```

**Pass references, not owned values, to async functions:**
```rust
// WRONG: owned buffer duplicated in state machine (2080 bytes for 1024-byte array)
async fn process(mut buffer: [u8; 1024]) { /* ... */ }

// CORRECT: reference — future is 40 bytes (98% reduction)
async fn process(buffer: &mut [u8; 1024]) { /* ... */ }
```

**Factor common await points out of match arms:**
```rust
// WRONG: duplicates send_response in state machine per arm
async fn handle() {
    match command().await {
        Cmd::A => send(123).await,
        Cmd::B => send(456).await,
    }
}

// CORRECT: single await point, ~11% smaller assembly
async fn handle() {
    let payload = match command().await {
        Cmd::A => 123,
        Cmd::B => 456,
    };
    send(payload).await;
}
```

---

## Testing

### Configuration
- **MUST** use `cargo nextest` as the test runner (process isolation, retries)
- **MUST** use `proptest` for property-based testing of serialization roundtrips and invariants
- **MUST** use `#[cfg(test)]` module in each source file for unit tests
- **MUST** place integration tests in `tests/` directory
- **NEVER** write tests that depend on execution order
- **NEVER** delete test files

```bash
cargo nextest run                           # Run all tests
cargo nextest run --workspace               # Entire workspace
cargo nextest run -p my-crate               # Single crate
cargo llvm-cov nextest --branch             # Coverage with branch tracking
```

### Property-Based Testing
```rust
use proptest::prelude::*;

proptest! {
    #[test]
    fn roundtrip_json(value in any::<MyType>()) {
        let json = serde_json::to_string(&value)?;
        let decoded: MyType = serde_json::from_str(&json)?;
        prop_assert_eq!(value, decoded);
    }

    #[test]
    fn sort_is_idempotent(mut vec in prop::collection::vec(any::<i32>(), 0..100)) {
        vec.sort();
        let sorted = vec.clone();
        vec.sort();
        prop_assert_eq!(vec, sorted);
    }
}
```

### Testing Best Practices
- Follow Arrange-Act-Assert pattern
- Use descriptive test names: `test_parse_rejects_empty_input`
- Use `#[should_panic(expected = "...")]` for panic tests
- Use `assert_matches!` macro for enum variant assertions
- Mock external dependencies with traits + test implementations
- **NEVER** use `#[ignore]` without a comment explaining why

---

## Performance Optimization

### Profile First
- **MUST** profile before optimizing—never optimize without data
- Use `criterion` for microbenchmarks
- Use `cargo flamegraph` for CPU profiling
- Use `dhat` for heap profiling
- Use `cargo-bloat` for binary size analysis and function-level weight
```bash
cargo bench                              # Run benchmarks
cargo flamegraph --bin myapp             # Generate flamegraph
cargo bench -- --baseline main           # Compare against baseline
cargo bloat --release --crates           # Binary size by crate
cargo bloat --release -n 20             # Top 20 largest functions
```

### Optimization Techniques
- Use `Vec::with_capacity()` when the size is known
- Use `Cow<'_, str>` to avoid unnecessary allocations
- Use `SmallVec` for small, stack-allocated collections
- Use `&str` instead of `String` in function parameters
- Prefer `iter()` chains over manual `for` loops (optimizer-friendly)
- Use `#[inline]` sparingly—only for small, hot functions in library crates
- **NEVER** use `collect()` when you can chain iterators instead

### Integer Safety
- **NEVER** rely on debug-mode overflow panics—release mode silently wraps
- **MUST** use `checked_*`, `saturating_*`, or `wrapping_*` methods for arithmetic
- **MUST** use `TryFrom`/`TryInto` instead of `as` for numeric conversions
- Consider `overflow-checks = true` in `[profile.release]`
```rust
// WRONG: silent truncation in release
let small = big_value as u8;

// CORRECT: explicit handling
let small = u8::try_from(big_value).context("value out of range")?;

// CORRECT: saturating arithmetic
let total = a.saturating_add(b);
```

### Memory Management
- Use `Arc` only when shared ownership is genuinely needed
- Prefer `Box<dyn Trait>` over `Arc<dyn Trait>` for single-owner polymorphism
- Use `Weak<T>` for back-references to break reference cycles
- Process large files in chunks with streaming iterators

---

## Security

### Secrets Management
- **NEVER** store secrets, API keys, or passwords in code
- **MUST** use environment variables or config files outside version control
- **MUST** add `.env` to `.gitignore`
- **NEVER** derive `Debug` on types with sensitive fields
- **NEVER** log sensitive information (passwords, tokens, PII)
- **MUST** use `secrecy::SecretString` for sensitive values in memory

### Input Validation
- **MUST** validate all external inputs at system boundaries
- **MUST** use `TryFrom` for validated type conversions
- **NEVER** use `unsafe` to bypass validation
- **NEVER** trust deserialized data—use `#[serde(deny_unknown_fields)]`
- Use newtypes to enforce validation at the type level

### Dependency Security
- **MUST** run `cargo audit` in CI pipeline
- **MUST** run `cargo deny check` for license and advisory compliance
- **MUST** prefer pure-Rust dependencies over C bindings when possible
```toml
# deny.toml
[advisories]
vulnerability = "deny"

[licenses]
unlicensed = "deny"
allow = ["MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"]

[bans]
multiple-versions = "warn"
deny = [
    { name = "openssl" },
    { name = "openssl-sys" },
]
```

---

## Project Structure

Use the **workspace layout** for anything beyond a single binary:
```
myproject/
├── Cargo.toml              # [workspace] definition
├── Cargo.lock
├── deny.toml               # cargo-deny config
├── clippy.toml             # Clippy disallowed-methods
├── .config/
│   └── nextest.toml        # nextest config
├── crates/
│   ├── myproject-core/     # Core library
│   │   ├── Cargo.toml
│   │   └── src/
│   │       ├── lib.rs
│   │       ├── error.rs
│   │       └── types.rs
│   └── myproject-cli/      # Binary crate
│       ├── Cargo.toml
│       └── src/
│           └── main.rs
├── tests/                  # Integration tests
│   └── integration.rs
└── README.md
```

---

## Configuration (Cargo.toml)

### Workspace Root
```toml
[workspace]
members = ["crates/*"]
resolver = "2"

[workspace.package]
edition = "2024"
rust-version = "1.85"
license = "MIT"

[workspace.dependencies]
tokio = { version = "1", features = ["full"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
anyhow = "1"
thiserror = "2"
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
clap = { version = "4", features = ["derive"] }
reqwest = { version = "0.12", default-features = false, features = ["rustls-tls", "json"] }

[workspace.dependencies.proptest]
version = "1"

[profile.release]
lto = true
strip = true
codegen-units = 1
overflow-checks = true
```

### Crate Cargo.toml
```toml
[package]
name = "myproject-core"
version = "0.1.0"
edition.workspace = true
rust-version.workspace = true
license.workspace = true

[dependencies]
tokio.workspace = true
serde.workspace = true
thiserror.workspace = true
tracing.workspace = true

[dev-dependencies]
proptest.workspace = true
tokio = { workspace = true, features = ["test-util"] }
```

### Clippy Configuration (clippy.toml)
```toml
disallowed-methods = [
    { path = "std::env::set_var", reason = "unsafe in edition 2024, not thread-safe" },
    { path = "std::env::remove_var", reason = "unsafe in edition 2024, not thread-safe" },
    { path = "std::thread::sleep", reason = "use tokio::time::sleep in async code" },
]
```

### Crate-Level Clippy Lints (lib.rs / main.rs)
```rust
#![warn(clippy::all, clippy::pedantic)]
#![allow(clippy::module_name_repetitions)]  // Common false positive
#![deny(clippy::unwrap_used, clippy::expect_used)]
#![deny(unsafe_code)]  // Remove only where justified
```

---

## Imports and Dependencies

- **MUST** avoid wildcard imports (`use module::*`) except in test modules and preludes
- **MUST** group imports: std → external crates → local modules
- **MUST** prefer `use crate::` for absolute paths within the crate
- **MUST** prefer re-exporting (`pub use`) from `lib.rs` for clean public API
- Let `rustfmt` handle import ordering automatically

---

## Version Control

- **MUST** write clear, descriptive commit messages
- **NEVER** commit commented-out code; delete it
- **NEVER** commit debug print statements (`dbg!`, `println!` for debugging)
- **NEVER** commit credentials or sensitive data
- **MUST** commit `Cargo.lock` for binary crates

---

## Before Committing Checklist

- [ ] All tests pass (`cargo nextest run`)
- [ ] Clippy clean (`cargo clippy --all-targets --all-features -- -D warnings`)
- [ ] Formatted (`cargo fmt -- --check`)
- [ ] Supply chain audit passes (`cargo deny check && cargo audit`)
- [ ] All public items have doc comments
- [ ] No `.unwrap()` / `.expect()` in production paths
- [ ] No `dbg!()` or debugging `println!()`
- [ ] No hardcoded credentials
- [ ] No `unsafe` without safety comments

---

**Remember:** Leverage the type system and the compiler as your first line of defense. If the compiler accepts it, it should be correct.
