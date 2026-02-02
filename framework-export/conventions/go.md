# GO Conventions - GoGent

## System Constraints (CRITICAL)

**This system targets desktop distribution. All GO code must:**

1. Compile to single binary with zero runtime dependencies
2. Cross-compile for darwin/amd64, darwin/arm64, windows/amd64, linux/amd64
3. Embed all static assets using `go:embed`
4. Never require users to install GO toolchain

## Project Structure

### Start Simple, Add Complexity Only When Needed

```
# Minimum viable (single binary)
myproject/
  go.mod
  main.go

# Add internal/ for private packages (compiler-enforced)
myproject/
  main.go
  internal/
    config/config.go
    handlers/handlers.go
  go.mod

# Add cmd/ only for multiple binaries
myproject/
  cmd/
    api/main.go
    worker/main.go
  internal/
    shared/
    api/
    worker/
  go.mod
```

**Rules:**

- `internal/` - Private packages, cannot be imported externally
- `pkg/` - ONLY if explicitly sharing code as library (rarely needed)
- `cmd/` - ONLY for multiple binaries
- Never use `golang-standards/project-layout` structure blindly

### Embedding Static Files

```go
// CORRECT: Package-level embed
//go:embed templates/*.html static/
var content embed.FS

// CORRECT: Single file
//go:embed version.txt
var version string

// WRONG: Embedding in function (won't compile)
func loadTemplates() {
    //go:embed templates/  // ERROR
}

// PREFER: //go:embed dirname over //go:embed dirname/* (latter includes dotfiles)
```

## Error Handling

### Wrapping Errors

```go
// CORRECT: Wrap with context using %w
if err := db.Query(ctx, query); err != nil {
    return fmt.Errorf("query users table: %w", err)
}

// CORRECT: Check specific errors
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}

// CORRECT: Extract typed errors
var apiErr *APIError
if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
    return handleRateLimit(apiErr)
}

// WRONG: String comparison
if err.Error() == "not found" {  // NEVER
    // ...
}

// WRONG: Bare error return
return err  // Add context!
```

### Sentinel Errors

```go
// Define at package level
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrInvalidInput  = errors.New("invalid input")
    ErrRateLimited   = errors.New("rate limited")
)

// Custom error types with Unwrap
type ValidationError struct {
    Field   string
    Message string
    Err     error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed on %s: %s", e.Field, e.Message)
}

func (e *ValidationError) Unwrap() error {
    return e.Err
}
```

### Panic Rules

```go
// CORRECT: Panic for programming errors only
func MustCompile(pattern string) *regexp.Regexp {
    re, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("invalid regex %q: %v", pattern, err))
    }
    return re
}

// WRONG: Panic for expected conditions
func GetUser(id int) *User {
    user, err := db.GetUser(id)
    if err != nil {
        panic(err)  // NEVER - return error instead
    }
    return user
}
```

## Concurrency Patterns

### Context Propagation

```go
// CORRECT: Accept context as first parameter
func (s *Service) ProcessTask(ctx context.Context, task Task) error {
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Pass context to all downstream calls
    result, err := s.api.Fetch(ctx, task.URL)
    if err != nil {
        return fmt.Errorf("fetch: %w", err)
    }

    return s.store.Save(ctx, result)
}

// WRONG: Ignoring context
func (s *Service) ProcessTask(task Task) error {
    result, _ := s.api.Fetch(context.Background(), task.URL)  // WRONG
    // ...
}
```

### Worker Pool Pattern

```go
type WorkerPool struct {
    numWorkers int
    jobs       chan Job
    results    chan Result
    wg         sync.WaitGroup
}

func NewWorkerPool(n int, bufferSize int) *WorkerPool {
    return &WorkerPool{
        numWorkers: n,
        jobs:       make(chan Job, bufferSize),
        results:    make(chan Result, bufferSize),
    }
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.numWorkers; i++ {
        wp.wg.Add(1)
        go wp.worker(ctx, i)
    }
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
    defer wp.wg.Done()
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-wp.jobs:
            if !ok {
                return
            }
            result := wp.process(job)
            select {
            case wp.results <- result:
            case <-ctx.Done():
                return
            }
        }
    }
}

func (wp *WorkerPool) Submit(job Job) {
    wp.jobs <- job
}

func (wp *WorkerPool) Close() {
    close(wp.jobs)
    wp.wg.Wait()
    close(wp.results)
}
```

### errgroup for Coordinated Operations

```go
import "golang.org/x/sync/errgroup"

func FetchAll(ctx context.Context, urls []string) ([]Result, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(urls))

    for i, url := range urls {
        i, url := i, url  // CRITICAL: Capture loop variables
        g.Go(func() error {
            result, err := fetch(ctx, url)
            if err != nil {
                return fmt.Errorf("fetch %s: %w", url, err)
            }
            results[i] = result
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

### Semaphore for Rate Limiting

```go
import "golang.org/x/sync/semaphore"

func ProcessWithLimit(ctx context.Context, tasks []Task, maxConcurrent int64) error {
    sem := semaphore.NewWeighted(maxConcurrent)
    g, ctx := errgroup.WithContext(ctx)

    for _, task := range tasks {
        task := task  // Capture

        if err := sem.Acquire(ctx, 1); err != nil {
            return fmt.Errorf("acquire semaphore: %w", err)
        }

        g.Go(func() error {
            defer sem.Release(1)
            return processTask(ctx, task)
        })
    }

    return g.Wait()
}
```

## HTTP Clients

### Never Use Default Client

```go
// WRONG: No timeout, can hang forever
resp, err := http.Get(url)

// CORRECT: Configured client with timeouts
func NewHTTPClient() *http.Client {
    return &http.Client{
        Timeout: 120 * time.Second,
        Transport: &http.Transport{
            DialContext: (&net.Dialer{
                Timeout:   10 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext,
            TLSHandshakeTimeout:   10 * time.Second,
            ResponseHeaderTimeout: 30 * time.Second,
            MaxIdleConns:          100,
            MaxIdleConnsPerHost:   10,
            IdleConnTimeout:       90 * time.Second,
            ForceAttemptHTTP2:     true,
        },
    }
}
```

### Exponential Backoff with Jitter

```go
func CalculateBackoff(attempt int, base, max time.Duration) time.Duration {
    delay := float64(base) * math.Pow(2.0, float64(attempt))
    jitter := delay * (0.5 + rand.Float64())  // Â±50% randomization
    if time.Duration(jitter) > max {
        return max
    }
    return time.Duration(jitter)
}

func RetryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        backoff := CalculateBackoff(attempt, 100*time.Millisecond, 30*time.Second)
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
    }
    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

## Testing

### Table-Driven Tests

```go
func TestParseConfig(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *Config
        wantErr  bool
    }{
        {
            name:     "valid config",
            input:    `{"port": 8080}`,
            expected: &Config{Port: 8080},
        },
        {
            name:    "invalid JSON",
            input:   `{invalid}`,
            wantErr: true,
        },
        {
            name:     "empty config uses defaults",
            input:    `{}`,
            expected: &Config{Port: 3000},
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            result, err := ParseConfig([]byte(tc.input))
            if tc.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

### Parallel Tests with Variable Capture

```go
func TestConcurrent(t *testing.T) {
    tests := []struct{
        name string
        input int
    }{
        {"case1", 1},
        {"case2", 2},
    }

    for _, tc := range tests {
        tc := tc  // CRITICAL: Capture range variable
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            // test logic using tc
        })
    }
}
```

### Run With Race Detector

```bash
# ALWAYS run in development
go test -race ./...

# CI should fail on race conditions
go test -race -count=1 ./...
```

## Naming Conventions

| Element     | Convention                   | Example                          |
| ----------- | ---------------------------- | -------------------------------- |
| Package     | lowercase, single word       | `http`, `config`, `agent`        |
| Exported    | PascalCase                   | `Client`, `NewServer`, `Config`  |
| Unexported  | camelCase                    | `config`, `parseInput`, `client` |
| Receiver    | 1-2 letter abbreviation      | `func (c *Client) Do()`          |
| Interface   | -er suffix for single method | `Reader`, `Writer`, `Stringer`   |
| Getters     | No "Get" prefix              | `func (u *User) Name() string`   |
| Initialisms | Consistent case              | `userID`, `httpClient`, `apiURL` |

### Avoid Stuttering

```go
// BAD: user.UserService
package user
type UserService struct{}

// GOOD: user.Service
package user
type Service struct{}
```

## Documentation

### Doc Comments Start With Name

```go
// Client is an HTTP client for the Claude API.
// Its zero value is not usable; use NewClient instead.
type Client struct {
    // APIKey is the authentication key for the API.
    // Required.
    APIKey string

    // Timeout specifies a time limit for requests.
    // Zero means no timeout.
    Timeout time.Duration
}

// NewClient creates a Client with the given API key.
// It returns an error if apiKey is empty.
func NewClient(apiKey string) (*Client, error)
```

## Linting Configuration

### .golangci.yml

```yaml
linters:
  enable:
    - errcheck # Check error returns
    - govet # Go vet checks
    - staticcheck # Comprehensive static analysis
    - gosimple # Simplification suggestions
    - ineffassign # Detect ineffectual assignments
    - bodyclose # HTTP response body closure
    - gosec # Security issues
    - gofmt # Format checking
    - goimports # Import organization

linters-settings:
  errcheck:
    check-type-assertions: true
    check-blank: true
  govet:
    enable-all: true

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
```

## Build Commands

### Makefile Template

```makefile
BINARY_NAME=GoGent
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

.PHONY: build build-all clean test lint

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/${BINARY_NAME}

build-all:
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ./cmd/${BINARY_NAME}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ./cmd/${BINARY_NAME}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ./cmd/${BINARY_NAME}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ./cmd/${BINARY_NAME}

clean:
	rm -f ${BINARY_NAME}
	rm -rf dist/

test:
	go test -race -v ./...

lint:
	golangci-lint run
```

## Sharp Edges

### Common Gotchas

1. **Loop variable capture in goroutines**

   ```go
   // WRONG: All goroutines see same value
   for _, item := range items {
       go func() {
           process(item)  // BUG: item changes
       }()
   }

   // CORRECT: Capture variable
   for _, item := range items {
       item := item  // Capture
       go func() {
           process(item)
       }()
   }
   ```

2. **Nil slice vs empty slice**

   ```go
   var s []int        // nil slice, json: null
   s := []int{}       // empty slice, json: []
   s := make([]int,0) // empty slice, json: []
   ```

3. **defer in loops**

   ```go
   // WRONG: Defers accumulate until function returns
   for _, file := range files {
       f, _ := os.Open(file)
       defer f.Close()  // Won't close until function ends
   }

   // CORRECT: Use anonymous function
   for _, file := range files {
       func() {
           f, _ := os.Open(file)
           defer f.Close()
           // ... use f ...
       }()
   }
   ```

4. **Channel closing**

   ```go
   // Only sender should close channels
   // Never close from receiver side
   // Closing twice causes panic
   ```

5. **Context cancellation**
   ```go
   // Always check context in long-running operations
   select {
   case <-ctx.Done():
       return ctx.Err()
   default:
   }
   ```
