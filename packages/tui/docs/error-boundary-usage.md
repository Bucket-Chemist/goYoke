# ErrorBoundary Usage Guide

The `ErrorBoundary` component catches React errors and prevents the entire TUI from crashing. It integrates with the logger system to capture error details for debugging.

## Basic Usage

Wrap any component that might throw errors:

```tsx
import { ErrorBoundary } from "./components/ErrorBoundary.js";
import { MyComponent } from "./components/MyComponent.js";

function App() {
  return (
    <ErrorBoundary>
      <MyComponent />
    </ErrorBoundary>
  );
}
```

## Custom Fallback UI

Provide a custom error display:

```tsx
import { ErrorBoundary } from "./components/ErrorBoundary.js";
import { Box, Text } from "ink";

function App() {
  return (
    <ErrorBoundary
      fallback={
        <Box borderStyle="double" borderColor="yellow" padding={1}>
          <Text color="yellow">
            Something went wrong. Press Ctrl+C to exit.
          </Text>
        </Box>
      }
    >
      <MyComponent />
    </ErrorBoundary>
  );
}
```

## Nested Error Boundaries

Use multiple error boundaries to isolate failures:

```tsx
import { ErrorBoundary } from "./components/ErrorBoundary.js";
import { Layout } from "./components/Layout.js";
import { AgentTree } from "./components/AgentTree.js";
import { AgentDetail } from "./components/AgentDetail.js";

function App() {
  return (
    <ErrorBoundary>
      <Layout>
        {/* Left panel has its own error boundary */}
        <ErrorBoundary>
          <AgentTree />
        </ErrorBoundary>

        {/* Right panel has its own error boundary */}
        <ErrorBoundary>
          <AgentDetail />
        </ErrorBoundary>
      </Layout>
    </ErrorBoundary>
  );
}
```

With this setup:
- If `AgentTree` crashes, only the left panel shows an error (right panel continues working)
- If `AgentDetail` crashes, only the right panel shows an error (left panel continues working)
- If `Layout` crashes, the outer boundary catches it

## Integration with Logger

When an error is caught, it's automatically logged to:
- **Memory buffer**: Last 50 errors available via `getRecentErrors()`
- **Debug file**: `~/.cache/gofortress-tui/debug.log` (when `DEBUG=true`)

The logged error includes:
- Error message
- Stack trace
- Component stack (which component tree caused the error)
- Timestamp

### Viewing Debug Logs

```bash
# Enable debug logging
DEBUG=true npm run dev

# In another terminal, tail the log
tail -f ~/.cache/gofortress-tui/debug.log
```

### Accessing Error Logs Programmatically

```tsx
import { getRecentErrors } from "./utils/logger.js";

function ErrorLogViewer() {
  const errors = getRecentErrors();

  return (
    <Box flexDirection="column">
      {errors.map((err, i) => (
        <Text key={i} color="red">
          {err.timestamp}: {err.message}
        </Text>
      ))}
    </Box>
  );
}
```

## Best Practices

### ✅ DO

- Wrap top-level components in error boundaries
- Use nested boundaries to isolate panel failures
- Provide user-friendly fallback messages
- Test error boundaries with intentional errors during development

### ❌ DON'T

- Don't wrap every single component (too granular)
- Don't use error boundaries as a substitute for proper error handling
- Don't ignore errors - check debug logs regularly
- Don't forget to handle async errors separately (boundaries only catch render errors)

## Testing Error Boundaries

Create a test component that throws on demand:

```tsx
import { useState } from "react";
import { Box, Text } from "ink";

function ErrorTest() {
  const [shouldThrow, setShouldThrow] = useState(false);

  if (shouldThrow) {
    throw new Error("Test error triggered");
  }

  return (
    <Box>
      <Text>Press 't' to trigger error</Text>
    </Box>
  );
}

// In your app:
<ErrorBoundary>
  <ErrorTest />
</ErrorBoundary>
```

## Async Error Handling

Error boundaries only catch errors during render. For async operations, handle errors explicitly:

```tsx
import { useState, useEffect } from "react";
import { Text } from "ink";
import { logger } from "./utils/logger.js";

function AsyncComponent() {
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function loadData() {
      try {
        const data = await fetchSomeData();
        // use data...
      } catch (err) {
        logger.error("Async error in loadData", {
          error: err instanceof Error ? err.message : String(err)
        });
        setError(err as Error);
      }
    }
    loadData();
  }, []);

  if (error) {
    return <Text color="red">Failed to load: {error.message}</Text>;
  }

  return <Text>Data loaded successfully</Text>;
}
```

## Common Error Scenarios

### Rendering Errors

Caught by ErrorBoundary:
- Invalid JSX structure
- Missing required props
- Component lifecycle errors
- Uncaught exceptions in render

### NOT Caught by ErrorBoundary

Must handle explicitly:
- Async/await errors
- Event handler errors (onClick, onKeyPress, etc.)
- Promise rejections
- setTimeout/setInterval callback errors

## Example: Full App Integration

```tsx
import { render } from "ink";
import { ErrorBoundary } from "./components/ErrorBoundary.js";
import { App } from "./App.js";
import { logger } from "./utils/logger.js";

// Global error handlers for uncaught errors
process.on("unhandledRejection", (reason) => {
  logger.error("Unhandled promise rejection", { reason: String(reason) });
});

process.on("uncaughtException", (error) => {
  logger.error("Uncaught exception", {
    message: error.message,
    stack: error.stack
  });
  process.exit(1);
});

// Render with top-level error boundary
render(
  <ErrorBoundary>
    <App />
  </ErrorBoundary>
);
```

---

**Related Documentation:**
- [Logger Usage](./logger-usage.md) - How the logger system works
- [Terminal Compatibility](./terminal-compatibility.md) - Terminal testing procedures
- [React Error Boundaries](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary) - Official React docs
