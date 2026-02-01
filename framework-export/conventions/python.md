---
paths:
  - "**/*.py"
---

# Agent Guidelines for Python Code Quality

This document provides guidelines for maintaining high-quality Python code. These rules MUST be followed by all AI coding agents and contributors.

## Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:
- Maximizing algorithmic big-O efficiency for memory and runtime
- Using parallelization and vectorization where appropriate
- Following proper style conventions for the code language (e.g., maximizing code reuse (DRY))
- No extra code beyond what is absolutely necessary to solve the problem the user provides (i.e., no technical debt)

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

---

## Python Version and Modern Features

**MUST** target Python 3.11+ for new projects; 3.12+ preferred.

### Python 3.11+ Features to Adopt
- **Self type** for methods returning `self` (replaces verbose TypeVar patterns)
- **ExceptionGroup** and `except*` for handling multiple concurrent errors
- **tomllib** for TOML parsing (built-in, replaces external toml packages)
- **asyncio.TaskGroup** for structured concurrency (replaces `asyncio.gather()`)

### Python 3.12+ Features to Adopt
- **PEP 695 type parameter syntax**: `class Stack[T]:` instead of `TypeVar`
- **F-string improvements**: nested quotes, multi-line expressions, comments allowed
- **`@override` decorator** for explicit method overriding
- **`type` statement** for type aliases: `type Vector[T] = list[T]`

### Python 3.13+ Features (When Available)
- **TypeIs** for type narrowing (prefer over TypeGuard)
- **ReadOnly** TypedDict fields for immutable configurations
- **`@deprecated`** decorator for marking deprecated APIs

**Example - Modern Generic Syntax (Python 3.12+):**
```python
from typing import Self

# PEP 695 syntax - PREFERRED
class Builder[T]:
    def add(self, item: T) -> Self:
        self._items.append(item)
        return self

# Type alias
type JSON = dict[str, "JSON"] | list["JSON"] | str | int | float | bool | None
```

---

## Preferred Tools

### Package Management
- **MUST** use `uv` for Python package management and virtual environments
- **MUST** commit `uv.lock` to version control for reproducibility
- **MUST** use `uv run` to execute Python commands in the virtual environment
- **MUST** add dependencies with `uv add` and `uv add --dev` for dev dependencies
```bash
uv init --package myproject  # Create project with src layout
uv add requests polars       # Add dependencies
uv add --dev pytest ruff mypy  # Add dev dependencies
uv sync                       # Install all dependencies
uv run pytest                 # Run commands in venv
```

### Environment: Arch Linux / CachyOS (CRITICAL)
On this system, **pip is incompatible with the system Python** due to Arch's externally-managed Python policy.

**Running Python:**
- **MUST** use a virtual environment for all Python execution
- **NEVER** run `pip install` or bare `python` on system Python—it will fail

**General-purpose venv:** `~/.generic-python/`
```bash
# Direct invocation (preferred for scripts)
~/.generic-python/bin/python script.py

# Interactive use
genpy                    # Bash alias to activate the venv
python                   # Then run interactively
```

**When to use which:**
| Scenario | Command |
|----------|---------|
| Running a script | `~/.generic-python/bin/python script.py` |
| Interactive REPL | `genpy` then `python` |
| Project with `pyproject.toml` | `uv run python ...` |
| Installing packages | `uv add package` (in project) |

### Development Tools
- **Ruff** for linting and formatting (replaces Black, isort, flake8)
- **mypy** (or **Pyright**) for static type checking
- **pytest** for testing with **pytest-xdist** for parallel execution
- **pre-commit** for automated checks before commits
- **tqdm** for progress bars in long-running loops
- **orjson** for fast JSON loading/dumping
- Use `logger.error` instead of `print` for error reporting

### Data Science Tools
- **ALWAYS** use `polars` instead of `pandas` for data frame manipulation
- Use **lazy evaluation** (`pl.scan_csv()`) for files >50MB
- **NEVER** print DataFrame row count or schema alongside the DataFrame (redundant)
- **NEVER** ingest more than 10 rows of a DataFrame at a time in context
- For pandas (when required): enable `dtype_backend="pyarrow"` and `copy_on_write`

**Example - Polars Lazy Evaluation:**
```python
import polars as pl

# PREFERRED: Lazy evaluation with query optimization
df = (
    pl.scan_csv("large_file.csv")
    .filter(pl.col("value") > 100)
    .group_by("category")
    .agg(pl.col("amount").sum())
    .collect()  # Execute optimized query at the end
)

# For larger-than-memory: use streaming
result = lazy_df.collect(streaming=True)
```

### Database Guidelines
- Do not denormalize unless explicitly prompted
- Use appropriate datatypes: `DATETIME/TIMESTAMP` for dates, `ARRAY` for nested fields
- **NEVER** save arrays as `TEXT/STRING`

---

## Code Style and Formatting

- **MUST** use meaningful, descriptive variable and function names
- **MUST** follow PEP 8 style guidelines
- **MUST** use 4 spaces for indentation (never tabs)
- **NEVER** use emoji or unicode emulating emoji (checkmarks, X marks) except in tests
- Use snake_case for functions/variables, PascalCase for classes, UPPER_CASE for constants
- Limit line length to 88 characters (Ruff formatter standard)

---

## Type Hints

### Requirements
- **MUST** use type hints for all function signatures (parameters and return values)
- **MUST** run mypy with `--strict` and resolve all type errors
- **NEVER** use `Any` type unless absolutely necessary—prefer `object` or union types
- **MUST** include `# type: ignore[error-code]` with specific codes, never bare ignores

### Modern Type Hint Patterns
```python
from typing import Self, TypedDict, Protocol, Literal, Final, TypeIs
from collections.abc import Sequence, Mapping

# Use Self for method chaining
class QueryBuilder:
    def where(self, condition: str) -> Self:
        return self

# Use TypedDict for structured dicts
class Config(TypedDict):
    host: str
    port: int
    debug: bool

# Use Protocol for structural subtyping
class Closeable(Protocol):
    def close(self) -> None: ...

# Use Literal for constrained values
Mode = Literal["read", "write", "append"]

# Use Final for constants
MAX_RETRIES: Final = 3

# Accept broad types, return specific types
def process(items: Sequence[str], config: Mapping[str, int]) -> list[str]:
    return [item.upper() for item in items]
```

### Type Hints to Avoid
- **NEVER** use `typing.List`, `typing.Dict`—use built-in `list`, `dict`
- **NEVER** use `Optional[X]`—use `X | None`
- **NEVER** use `Union[X, Y]`—use `X | Y`
- **NEVER** use old TypeVar syntax when PEP 695 is available (Python 3.12+)

---

## Documentation

- **MUST** include docstrings for all public functions, classes, and methods
- **MUST** document function parameters, return values, and exceptions raised
- Keep comments up-to-date with code changes
- Include examples in docstrings for complex functions
```python
def calculate_total(items: list[dict], tax_rate: float = 0.0) -> float:
    """Calculate the total cost of items including tax.

    Args:
        items: List of item dictionaries with 'price' keys
        tax_rate: Tax rate as decimal (e.g., 0.08 for 8%)

    Returns:
        Total cost including tax

    Raises:
        ValueError: If items is empty or tax_rate is negative

    Example:
        >>> calculate_total([{"price": 10}, {"price": 20}], 0.1)
        33.0
    """
```

---

## Error Handling

- **NEVER** silently swallow exceptions without logging
- **NEVER** use bare `except:` clauses
- **MUST** catch specific exceptions rather than broad exception types
- **MUST** use context managers (`with` statements) for resource cleanup
- Provide meaningful error messages

### Exception Groups (Python 3.11+)
Use `ExceptionGroup` and `except*` for concurrent operations:
```python
import asyncio

async def fetch_all(urls: list[str]) -> list[str]:
    try:
        async with asyncio.TaskGroup() as tg:
            tasks = [tg.create_task(fetch(url)) for url in urls]
        return [t.result() for t in tasks]
    except* ValueError as eg:
        for exc in eg.exceptions:
            logger.error(f"Validation error: {exc}")
        raise
    except* ConnectionError as eg:
        for exc in eg.exceptions:
            logger.error(f"Connection error: {exc}")
        raise
```

---

## Function Design

- **MUST** keep functions focused on a single responsibility
- **NEVER** use mutable objects (lists, dicts) as default argument values
- Limit function parameters to 5 or fewer
- Return early to reduce nesting

---

## Class Design

- **MUST** keep classes focused on a single responsibility
- **MUST** keep `__init__` simple; avoid complex logic
- Use `@dataclass(slots=True)` for simple data containers (memory efficient)
- Prefer composition over inheritance
- Use `@property` for computed attributes
- Use `@override` decorator when overriding parent methods (Python 3.12+)
```python
from dataclasses import dataclass
from typing import override

@dataclass(slots=True)
class Point:
    x: float
    y: float

class Animal:
    def speak(self) -> str:
        return "..."

class Dog(Animal):
    @override
    def speak(self) -> str:
        return "Woof!"
```

---

## Concurrency and Parallelization

### asyncio (I/O-bound tasks)
- **MUST** use `asyncio.TaskGroup` instead of `asyncio.gather()` for structured concurrency
- **MUST** use `asyncio.timeout()` for bounded async operations
- **MUST** let `CancelledError` propagate—do not swallow it
- **MUST** use `asyncio.to_thread()` for blocking operations in async code
```python
import asyncio

async def fetch_with_timeout(urls: list[str]) -> list[str]:
    async with asyncio.timeout(30.0):
        async with asyncio.TaskGroup() as tg:
            tasks = [tg.create_task(fetch(url)) for url in urls]
        return [t.result() for t in tasks]
```

### Multiprocessing (CPU-bound tasks)
- **MUST** use `ProcessPoolExecutor` for CPU-bound work
- **MUST** use context managers for executor lifecycle
- **MUST** use `if __name__ == "__main__":` guard
- **MUST** check `future.result()` to surface errors
```python
from concurrent.futures import ProcessPoolExecutor, as_completed

def parallel_process(items: list) -> list:
    with ProcessPoolExecutor() as executor:
        futures = {executor.submit(process, item): item for item in items}
        results = []
        for future in as_completed(futures):
            try:
                results.append(future.result())
            except Exception as e:
                logger.error(f"Failed: {futures[future]}: {e}")
        return results
```

### Concurrency Selection Guide
| Workload | Solution |
|----------|----------|
| I/O-bound (network, files) | `asyncio.TaskGroup` or `ThreadPoolExecutor` |
| CPU-bound (computation) | `ProcessPoolExecutor` |
| ML data preprocessing | `joblib` with loky backend |
| Large datasets | `Polars` (parallel by default) or `Dask` |

---

## Testing

### pytest Configuration
- **MUST** use pytest as the testing framework
- **MUST** use `pytest-xdist` for parallel test execution (`pytest -n auto`)
- **MUST** use `pytest-cov` with branch coverage (`--cov-branch`)
- **MUST** mock external dependencies (APIs, databases, file systems)
- **NEVER** run tests without saving them as discrete files first
- **NEVER** delete files created as part of testing
```bash
pytest -n auto --cov=src --cov-branch --cov-fail-under=80
```

### Property-Based Testing with Hypothesis
Use Hypothesis for testing invariants and discovering edge cases:
```python
from hypothesis import given, strategies as st

@given(st.lists(st.integers()))
def test_sorted_is_idempotent(lst: list[int]) -> None:
    result = sorted(lst)
    assert sorted(result) == result
    assert len(result) == len(lst)

@given(st.text(), st.text())
def test_string_concatenation_length(s1: str, s2: str) -> None:
    assert len(s1 + s2) == len(s1) + len(s2)
```

### Testing Best Practices
- Follow the Arrange-Act-Assert pattern
- Use `pytest.raises` for exception testing
- Use `pytest.mark.parametrize` for multiple inputs
- Use `autospec=True` when mocking to catch API mismatches
- **NEVER** write tests that depend on execution order

---

## Performance Optimization

### Profiling First
- **MUST** profile before optimizing—never optimize without data
- Use `cProfile` for function-level profiling
- Use `py-spy` for production profiling (zero code changes)
- Use `scalene` for combined CPU/memory/GPU profiling
```bash
# Generate flamegraph
py-spy record -o flame.svg -- python script.py

# Scalene profiling
scalene script.py
```

### Optimization Techniques
- Use NumPy/Polars vectorization over Python loops
- Use `@functools.cache` or `@lru_cache` for pure functions with repeated calls
- Use `__slots__` for classes with many instances
- Use generators for large sequence processing
- **NEVER** concatenate strings in loops—use `"".join()` or `io.StringIO`

### Memory Management
- Use generators instead of list comprehensions for large datasets
- Use `__slots__` for memory-constrained classes
- Process large files in chunks
- Use `numpy.memmap` for large numeric arrays

---

## Security

### Secrets Management
- **NEVER** store secrets, API keys, or passwords in code
- **MUST** use environment variables via `python-dotenv` or `pydantic-settings`
- **MUST** add `.env` to `.gitignore`
- **NEVER** print or log URLs containing API keys
- **NEVER** log sensitive information (passwords, tokens, PII)

### Input Validation
- **MUST** validate all external inputs with Pydantic
- **MUST** use parameterized queries for all database operations
- **MUST** use `os.path.basename()` for user-provided filenames
- **NEVER** use `pickle.loads()` with untrusted data (RCE vulnerability)
- **NEVER** use `yaml.load()`—use `yaml.safe_load()`
- **NEVER** use `eval()`, `exec()`, or `subprocess` with `shell=True` on user input

### Dependency Security
- **MUST** run `pip-audit` in CI pipeline
- **MUST** run `bandit` for static security analysis
- **MUST** use pre-commit hooks for secret scanning (detect-secrets, gitleaks)
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/PyCQA/bandit
    rev: 1.7.9
    hooks:
      - id: bandit
        args: [-r, -ll, -x, tests]
  - repo: https://github.com/gitleaks/gitleaks
    rev: v8.22.1
    hooks:
      - id: gitleaks
```

---

## Project Structure

Use the **src layout** for all packages:
```
myproject/
├── src/
│   └── mypackage/
│       ├── __init__.py
│       ├── py.typed          # For mypy
│       └── module.py
├── tests/
│   ├── __init__.py
│   └── test_module.py
├── pyproject.toml
├── uv.lock
├── .python-version
├── .pre-commit-config.yaml
└── README.md
```

---

## Configuration (pyproject.toml)
```toml
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "mypackage"
version = "0.1.0"
requires-python = ">=3.11"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=8.0",
    "pytest-cov>=4.0",
    "pytest-xdist>=3.0",
    "hypothesis>=6.0",
    "ruff>=0.5",
    "mypy>=1.10",
    "pre-commit>=3.7",
    "bandit>=1.7",
    "pip-audit>=2.7",
]

[tool.ruff]
target-version = "py311"
line-length = 88
src = ["src", "tests"]

[tool.ruff.lint]
select = ["E", "W", "F", "I", "B", "C4", "UP", "ARG", "SIM", "TCH", "PTH", "RUF"]
ignore = ["E501"]

[tool.ruff.lint.isort]
known-first-party = ["mypackage"]

[tool.mypy]
python_version = "3.11"
strict = true
warn_return_any = true
warn_unused_ignores = true
show_error_codes = true

[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = "-ra --strict-markers"
asyncio_mode = "auto"

[tool.coverage.run]
source = ["src"]
branch = true

[tool.coverage.report]
fail_under = 80
show_missing = true
exclude_lines = ["pragma: no cover", "if TYPE_CHECKING:"]
```

---

## Imports and Dependencies

- **MUST** avoid wildcard imports (`from module import *`)
- **MUST** document dependencies in `pyproject.toml`
- Organize imports: standard library, third-party, local imports
- Ruff handles import sorting automatically

---

## Version Control

- **MUST** write clear, descriptive commit messages
- **NEVER** commit commented-out code; delete it
- **NEVER** commit debug print statements or breakpoints
- **NEVER** commit credentials or sensitive data

---

## Before Committing Checklist

- [ ] All tests pass (`uv run pytest -n auto`)
- [ ] Type checking passes (`uv run mypy src`)
- [ ] Linter and formatter pass (`uv run ruff check . && uv run ruff format .`)
- [ ] Security scan passes (`uv run bandit -r src && uv run pip-audit`)
- [ ] All functions have docstrings and type hints
- [ ] No commented-out code or debug statements
- [ ] No hardcoded credentials

---

**Remember:** Prioritize clarity and maintainability over cleverness.
