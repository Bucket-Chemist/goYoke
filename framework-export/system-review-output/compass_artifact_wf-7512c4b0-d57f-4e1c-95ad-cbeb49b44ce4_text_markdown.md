# Code Review Framework: React 19+ and TypeScript 5.x Best Practices (2024-2025)

**React 19's stable release in December 2024 fundamentally changes React development patterns**, introducing automatic memoization via the React Compiler, native Server Components, and streamlined form handling. Combined with TypeScript 5.5+'s inferred type predicates and isolated declarations, these updates require significant code review criteria adjustments. This guide provides actionable patterns to encourage and anti-patterns to flag, organized for immediate integration into your review process.

---

## React 19: The compiler changes everything

React Compiler reached **stable release (v1.0) in October 2025**, making manual memoization largely unnecessary. The compiler automatically optimizes components at build time, eliminating cascading re-renders and memoizing expensive calculations. Meta's production data shows **12% faster initial loads** and **2.5x faster interactions** in the Quest Store deployment.

**Code review implications are significant.** For projects with React Compiler enabled, flag new instances of `useMemo`, `useCallback`, and `React.memo` as likely unnecessary—the compiler handles these optimizations automatically. However, existing manual memoization should remain in place since removing it can alter compilation output. Projects without the compiler still require manual memoization for performance-critical paths.

The compiler requires components to follow the "Rules of React"—pure components that don't mutate props or state. Flag any component that mutates its inputs directly, as this violates compiler assumptions and will produce incorrect optimizations. The ESLint plugin `eslint-plugin-react-hooks` now includes compiler-specific rules that should be enforced.

---

## Server Components and the client-server boundary

Server Components are now the **default in React 19**—no directive needed. They execute exclusively on the server, enabling direct database access, API credential usage, and zero client-side JavaScript for rendering. Client Components require the `'use client'` directive and are necessary only for interactivity (hooks, event handlers, browser APIs).

**Patterns to encourage in reviews:**

- Server Components for data fetching, static content, and components using large dependencies
- Client boundaries placed as deep in the component tree as possible
- The `use()` hook for conditional context and promise reading (unlike other hooks, it can be called in conditionals)
- Native `<title>`, `<meta>`, and `<link>` tags in components—React 19 automatically hoists these to `<head>`

**Critical anti-patterns to flag:**

- Using `useState`, `useEffect`, or event handlers in Server Components
- Importing Server Components into Client Components (architecturally impossible)
- Missing `'use client'` directive on components using hooks
- Creating promises inside render when using `use()` (causes infinite loops)
- The `use()` hook in try/catch blocks (not supported—use Error Boundaries)

---

## Server Actions replace API routes for mutations

Server Actions allow Client Components to call async server functions directly, integrating seamlessly with forms and React's transition system. Define them with the `'use server'` directive.

The new form hooks provide complete lifecycle management: `useActionState` (formerly `useFormState`) returns state, form action, and pending status; `useFormStatus` provides submission status to child components; `useOptimistic` enables immediate UI updates during async operations.

**Critical pattern for useFormStatus:** It must be called in a child component of the form, not the same component containing the `<form>` element. This is a common mistake that results in `pending` always being `false`. Structure forms with a separate `SubmitButton` component that consumes the status.

---

## Hooks best practices have evolved significantly

The 2024-2025 consensus on hooks reflects both React Compiler adoption and years of community experience with common pitfalls.

**useEffect anti-patterns to flag aggressively:**

| Anti-Pattern | Problem | Better Approach |
|--------------|---------|-----------------|
| Data transformations in useEffect | Unnecessary re-renders | Compute during render |
| Missing/incorrect dependency arrays | Stale closures, infinite loops | Include all dependencies |
| Async operations without cleanup | Memory leaks, race conditions | Use AbortController |
| Objects/arrays as dependencies | Infinite re-renders | Extract values or useMemo |

**When NOT to use useEffect:** data transformations (do during render), initializing state from props (use initial state function), resetting state on prop change (use key prop), subscribing to external stores (use `useSyncExternalStore`).

**useMemo/useCallback decision tree:** Is there a measured performance problem? If no, skip memoization. If yes, is the value expensive (>100ms) or passed to a memoized child? Then memoize. For callbacks, only use `useCallback` when passing to memoized children or when the function is a hook dependency.

**Custom hooks must** start with `use` prefix (enables ESLint enforcement), contain logic only (never return JSX), maintain single responsibility, and have isolated state per call.

---

## State management: a clear decision framework emerges

The 2024-2025 landscape has consolidated around a clear separation of concerns.

| State Type | Recommended Tool | Bundle Size |
|------------|------------------|-------------|
| Server/API state | TanStack Query | ~40KB |
| Global client state | Zustand or Jotai | ~3KB each |
| Local component state | useState/useReducer | 0KB |
| Form state | React Hook Form or local | Varies |

**Zustand vs Jotai:** Zustand uses a centralized store model with manual selector-based optimization; Jotai uses an atomic model with automatic atom-level optimization. Choose Zustand for interconnected global state, Jotai for many independent pieces needing fine-grained reactivity.

**TanStack Query v5 changes (2024):** `isLoading` renamed to `isPending`, `cacheTime` renamed to `gcTime`. Always separate server state (TanStack Query) from client state (Zustand/Jotai). Use query key factories for organization and implement optimistic updates for responsive UIs.

**Flag in reviews:** storing server state in Zustand/Jotai (use TanStack Query), not using Zustand selectors (causes unnecessary re-renders), Context API for frequently-changing state (causes full subtree re-renders).

---

## Performance patterns beyond memoization

**Concurrent features usage:** `useTransition` wraps low-priority state updates when you control the setter—ideal for tab switching, filtering large lists. `useDeferredValue` wraps values from props when you don't control the update source. Both should be used sparingly, only for complex UI performance issues, not as default patterns.

**Code splitting remains essential:** Use `React.lazy` with `Suspense` at route boundaries. For lists exceeding 100 items, require virtualization (react-window or react-virtualized). Flag components defined inside other components (recreated every render, breaking memoization).

**Context value memoization:** Even with React Compiler, explicitly memoize context values containing objects or arrays to maintain precise control over consumer re-renders.

---

## Testing has shifted toward Vitest and behavior-driven patterns

**Vitest vs Jest:** Vitest offers **4-10x faster execution** with native ESM support and seamless Vite integration. Use Vitest for new Vite-based projects; retain Jest for React Native and legacy codebases requiring stability.

**React Testing Library query priority (enforce in reviews):**

1. `*ByRole` — most accessible, default choice
2. `*ByLabelText` — for form fields
3. `*ByText` — for non-interactive elements
4. `*ByTestId` — escape hatch only

**Required patterns:** Use `screen` object (not render destructuring), prefer `userEvent` over `fireEvent` (simulates real user behavior), use `findBy*` for async content (not `getBy*` with `waitFor`). Test custom hooks via `renderHook` or through components that use them. Use MSW (Mock Service Worker) for API mocking—avoid mocking implementation details.

**Server Component testing** remains challenging; the React team recommends E2E testing for full RSC behavior verification.

---

## TypeScript + React integration patterns

**Component typing consensus:** Plain function declarations with typed props are preferred over `React.FC`, though FC is acceptable in React 18+ (the implicit children issue was fixed). Always explicitly type children as `React.ReactNode` when needed.

**Event handler typing patterns:**
- `React.ChangeEvent<HTMLInputElement>` for change handlers
- `React.FormEvent<HTMLFormElement>` for form submission
- `React.MouseEvent<HTMLButtonElement>` for click handlers
- `React.KeyboardEvent<HTMLInputElement>` for keyboard handlers

**Type-safe context pattern:** Create context with `| undefined` type, then use a custom hook with type guard that throws if context is undefined. This eliminates null checks at every consumption site.

**Generic components** enable type-safe reusable patterns—use for lists, tables, and selection components where item types vary.

---

## TypeScript 5.5+ features for modern codebases

**Inferred type predicates (TS 5.5, June 2024)** eliminate boilerplate: `arr.filter(x => x !== null)` now properly narrows to remove `null` from the resulting type. However, truthiness checks like `x => !!x` do NOT infer predicates for primitives because `0` is falsy but valid. Prefer explicit `x => x !== undefined` checks.

**Isolated declarations mode** enables parallel builds in monorepos by requiring explicit type annotations on all exports. This allows tools like esbuild and swc to generate `.d.ts` files without the full type-checker, achieving up to **3x faster build times**. Enable for libraries and monorepos; ensure all exported functions have explicit return types.

**TypeScript 5.6 (September 2024)** adds compile-time detection of always-truthy or always-nullish conditions and iterator helper method support. **TypeScript 5.7 (November 2024)** adds checks for never-initialized variables and `--rewriteRelativeImportExtensions` for tools running TypeScript directly.

---

## The satisfies operator preserves literal types

Use `satisfies` when you want type validation without losing specific literal types—critical for config objects needing both validation and precise autocomplete:

```typescript
// Type annotation widens to union, losing specificity
const palette: Record<"red"|"green", string|number[]> = { red: [255,0,0], green: "#00ff00" };
palette.green.toUpperCase(); // Error: might be number[]

// satisfies preserves literal types while validating
const palette = { red: [255,0,0], green: "#00ff00" } satisfies Record<"red"|"green", string|number[]>;
palette.green.toUpperCase(); // Works: TS knows it's string
```

Use `satisfies` with `as const` for exhaustive config validation. Don't overuse—simple objects don't benefit from the added verbosity.

---

## Module resolution: match your deployment environment

| Use Case | moduleResolution | module |
|----------|------------------|--------|
| Node.js applications | `nodenext` | `nodenext` |
| Published libraries | `nodenext` | `nodenext` |
| Bundler (Vite, webpack) | `bundler` | `esnext` |
| Bun/tsx | `bundler` | `esnext` |

**Key difference:** `bundler` doesn't require file extensions on relative imports; `nodenext` requires explicit `.js` extensions for ESM compatibility but guarantees Node.js compatibility. For new projects, prefer `"type": "module"` in package.json.

---

## Recommended strict configuration (2024-2025)

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "verbatimModuleSyntax": true,
    "isolatedModules": true,
    "forceConsistentCasingInFileNames": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "target": "es2022"
  }
}
```

**Critical flags:** `noUncheckedIndexedAccess` adds `undefined` to index signatures (catches real bugs); `verbatimModuleSyntax` enforces `import type` syntax (improves tree-shaking); `forceConsistentCasingInFileNames` prevents cross-platform bugs.

---

## TypeScript anti-patterns for code review

**Flag aggressively:**

- **Any `any` usage** — prefer `unknown` with type guards, or generics for truly dynamic cases
- **Excessive type assertions (`as`)** — use type guards instead; double assertions (`as unknown as T`) are red flags
- **Non-null assertions (`!`)** without justification — handle null cases explicitly
- **Union types without discriminants** — require explicit `type` or `status` properties for state unions
- **Overly complex types** — extract deeply nested conditional types into named, documented aliases

**Exhaustiveness enforcement:** Require `never` checks in switch statements on discriminated unions to catch missing cases at compile time:

```typescript
default:
  const _exhaustive: never = result; // Compile error if case missed
```

---

## Consolidated code review checklist

### React 19+ Patterns
- [ ] React Compiler enabled (or appropriate manual memoization justified)
- [ ] Server Components for data fetching; Client Components only where required
- [ ] `useActionState` + `useFormStatus` for forms (status hook in child component)
- [ ] `use()` with stable promises (not created during render)
- [ ] Native metadata tags instead of third-party libraries
- [ ] No hooks in Server Components; no Server Components imported into Client Components

### Hooks and State
- [ ] All useEffect dependencies included; cleanup for subscriptions/timers
- [ ] useMemo/useCallback only where profiled performance benefit exists
- [ ] Server state in TanStack Query; client state in Zustand/Jotai/useState
- [ ] Zustand stores always accessed via selectors

### Performance and Testing
- [ ] Code splitting at route boundaries
- [ ] Lists >100 items virtualized
- [ ] Tests use `*ByRole` queries and `userEvent`
- [ ] No implementation detail testing (state, props directly)

### TypeScript 5.x
- [ ] No `any` without documented justification
- [ ] Type assertions minimal; type guards preferred
- [ ] Explicit return types on exported functions
- [ ] Discriminated unions for state with exhaustiveness checks
- [ ] Module resolution matches deployment environment
- [ ] Strict mode enabled with `noUncheckedIndexedAccess`

---

## Conclusion

The React 19 and TypeScript 5.5+ ecosystem represents a maturation toward **compiler-driven optimization** and **stricter type safety by default**. The most significant shift for code reviewers is recognizing that manual memoization is now often an anti-pattern when React Compiler is enabled, while TypeScript's inferred type predicates and isolated declarations mode demand more explicit typing at module boundaries. Server Components fundamentally change data fetching architecture, and the state management landscape has consolidated around clear separation between server state (TanStack Query) and client state (Zustand/Jotai). Adopt these patterns incrementally—migrate to React 19 via the official codemods, enable TypeScript strict flags progressively, and update ESLint configurations to enforce the new best practices automatically.