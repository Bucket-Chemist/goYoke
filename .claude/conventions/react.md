# React Conventions

## System Constraints (CRITICAL)

**This system targets React 18+ with functional components and TypeScript.**

All React code must:
1. Use functional components exclusively - no class components
2. Include proper TypeScript types for all props and state
3. Follow React 18+ patterns (concurrent features, automatic batching)
4. Be compatible with strict mode
5. Never mutate state directly

## Component Patterns

### Component Structure

```typescript
// CORRECT: Complete component with types
interface UserCardProps {
  user: User;
  onEdit?: (user: User) => void;
  className?: string;
}

function UserCard({ user, onEdit, className }: UserCardProps): JSX.Element {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleEdit = useCallback(() => {
    onEdit?.(user);
  }, [onEdit, user]);

  return (
    <div className={className}>
      <h3>{user.name}</h3>
      <button onClick={() => setIsExpanded(!isExpanded)}>
        {isExpanded ? "Collapse" : "Expand"}
      </button>
      {isExpanded && <UserDetails user={user} onEdit={handleEdit} />}
    </div>
  );
}

// WRONG: Missing types
function UserCard({ user, onEdit }) {  // Implicit any
  // ...
}

// WRONG: Props destructuring without interface
function UserCard(props: { user: User; onEdit: Function }) {
  // Don't use Function type - too broad
}
```

### Props Best Practices

```typescript
// CORRECT: Specific prop types
interface ButtonProps {
  variant: "primary" | "secondary" | "danger";
  size?: "small" | "medium" | "large";
  disabled?: boolean;
  onClick: (event: React.MouseEvent<HTMLButtonElement>) => void;
  children: React.ReactNode;
}

// CORRECT: Extending HTML element props
interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

function Input({ label, error, ...inputProps }: InputProps): JSX.Element {
  return (
    <div>
      <label>{label}</label>
      <input {...inputProps} />
      {error && <span className="error">{error}</span>}
    </div>
  );
}

// CORRECT: Component props with generics
interface ListProps<T> {
  items: T[];
  renderItem: (item: T) => React.ReactNode;
  keyExtractor: (item: T) => string;
}

function List<T>({ items, renderItem, keyExtractor }: ListProps<T>) {
  return (
    <ul>
      {items.map(item => (
        <li key={keyExtractor(item)}>{renderItem(item)}</li>
      ))}
    </ul>
  );
}

// Usage
<List
  items={users}
  renderItem={user => <UserCard user={user} />}
  keyExtractor={user => user.id}
/>
```

### Composition Pattern

```typescript
// CORRECT: Compound components with context
interface TabsContextValue {
  activeTab: string;
  setActiveTab: (id: string) => void;
}

const TabsContext = React.createContext<TabsContextValue | null>(null);

function useTabs(): TabsContextValue {
  const context = useContext(TabsContext);
  if (!context) {
    throw new Error("useTabs must be used within Tabs");
  }
  return context;
}

interface TabsProps {
  defaultTab: string;
  children: React.ReactNode;
}

function Tabs({ defaultTab, children }: TabsProps): JSX.Element {
  const [activeTab, setActiveTab] = useState(defaultTab);

  const value = useMemo(
    () => ({ activeTab, setActiveTab }),
    [activeTab]
  );

  return (
    <TabsContext.Provider value={value}>
      <div className="tabs">{children}</div>
    </TabsContext.Provider>
  );
}

interface TabListProps {
  children: React.ReactNode;
}

function TabList({ children }: TabListProps): JSX.Element {
  return <div className="tab-list">{children}</div>;
}

interface TabProps {
  id: string;
  children: React.ReactNode;
}

function Tab({ id, children }: TabProps): JSX.Element {
  const { activeTab, setActiveTab } = useTabs();

  return (
    <button
      className={activeTab === id ? "active" : ""}
      onClick={() => setActiveTab(id)}
    >
      {children}
    </button>
  );
}

interface TabPanelProps {
  id: string;
  children: React.ReactNode;
}

function TabPanel({ id, children }: TabPanelProps): JSX.Element | null {
  const { activeTab } = useTabs();

  if (activeTab !== id) {
    return null;
  }

  return <div className="tab-panel">{children}</div>;
}

// Attach subcomponents
Tabs.List = TabList;
Tabs.Tab = Tab;
Tabs.Panel = TabPanel;

// Usage
<Tabs defaultTab="profile">
  <Tabs.List>
    <Tabs.Tab id="profile">Profile</Tabs.Tab>
    <Tabs.Tab id="settings">Settings</Tabs.Tab>
  </Tabs.List>
  <Tabs.Panel id="profile">Profile content</Tabs.Panel>
  <Tabs.Panel id="settings">Settings content</Tabs.Panel>
</Tabs>
```

### Render Props Pattern

```typescript
interface MouseTrackerProps {
  children: (position: { x: number; y: number }) => React.ReactNode;
}

function MouseTracker({ children }: MouseTrackerProps): JSX.Element {
  const [position, setPosition] = useState({ x: 0, y: 0 });

  useEffect(() => {
    const handleMouseMove = (event: MouseEvent): void => {
      setPosition({ x: event.clientX, y: event.clientY });
    };

    window.addEventListener("mousemove", handleMouseMove);
    return () => window.removeEventListener("mousemove", handleMouseMove);
  }, []);

  return <>{children(position)}</>;
}

// Usage
<MouseTracker>
  {({ x, y }) => (
    <div>
      Mouse position: {x}, {y}
    </div>
  )}
</MouseTracker>
```

### Higher-Order Components (Use Sparingly)

```typescript
// PREFER: Custom hooks over HOCs
// Only use HOCs for cross-cutting concerns like error boundaries

function withErrorBoundary<P extends object>(
  Component: React.ComponentType<P>
): React.ComponentType<P> {
  return function WithErrorBoundary(props: P): JSX.Element {
    return (
      <ErrorBoundary>
        <Component {...props} />
      </ErrorBoundary>
    );
  };
}

// BETTER: Use hooks instead
function useErrorHandler(): {
  error: Error | null;
  resetError: () => void;
} {
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const handleError = (event: ErrorEvent): void => {
      setError(event.error);
    };

    window.addEventListener("error", handleError);
    return () => window.removeEventListener("error", handleError);
  }, []);

  const resetError = useCallback(() => setError(null), []);

  return { error, resetError };
}
```

## Hooks Mastery

### useState Rules

```typescript
// CORRECT: Proper state initialization
const [count, setCount] = useState(0);
const [user, setUser] = useState<User | null>(null);

// CORRECT: Lazy initialization for expensive computation
const [data, setData] = useState(() => {
  return computeExpensiveValue();
});

// CORRECT: Functional updates for state based on previous state
const incrementCount = () => setCount(prev => prev + 1);

// WRONG: Mutating state
const [items, setItems] = useState<string[]>([]);
items.push("new");  // NEVER mutate state
setItems(items);

// CORRECT: Immutable state updates
setItems(prev => [...prev, "new"]);

// CORRECT: Object state updates
interface FormState {
  name: string;
  email: string;
}

const [form, setForm] = useState<FormState>({ name: "", email: "" });

const updateField = (field: keyof FormState, value: string): void => {
  setForm(prev => ({ ...prev, [field]: value }));
};

// WRONG: Multiple related state values
const [firstName, setFirstName] = useState("");
const [lastName, setLastName] = useState("");
const [email, setEmail] = useState("");

// CORRECT: Single state object
interface UserForm {
  firstName: string;
  lastName: string;
  email: string;
}

const [form, setForm] = useState<UserForm>({
  firstName: "",
  lastName: "",
  email: "",
});
```

### useEffect Rules

```typescript
// CORRECT: Effect with dependencies
useEffect(() => {
  fetchUser(userId);
}, [userId]);  // Re-run when userId changes

// CORRECT: Cleanup function
useEffect(() => {
  const subscription = api.subscribe(userId);

  return () => {
    subscription.unsubscribe();
  };
}, [userId]);

// CORRECT: Multiple effects for different concerns
useEffect(() => {
  // Track page view
  analytics.trackPageView(pathname);
}, [pathname]);

useEffect(() => {
  // Fetch data
  fetchData();
}, [dataId]);

// WRONG: Missing dependencies
useEffect(() => {
  fetchUser(userId);  // userId used but not in deps
}, []);  // ESLint error

// WRONG: Infinite loop
useEffect(() => {
  setCount(count + 1);  // Updates count, triggers effect again
}, [count]);

// CORRECT: Conditional effect execution
useEffect(() => {
  if (shouldFetch) {
    fetchData();
  }
}, [shouldFetch]);

// CORRECT: Abort controller for cleanup
useEffect(() => {
  const controller = new AbortController();

  async function fetchData(): Promise<void> {
    try {
      const response = await fetch(url, { signal: controller.signal });
      const data = await response.json();
      setData(data);
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") {
        return;  // Ignore abort errors
      }
      setError(error);
    }
  }

  fetchData();

  return () => controller.abort();
}, [url]);
```

### useCallback

```typescript
// CORRECT: Memoize callbacks passed to child components
interface ChildProps {
  onUpdate: (value: string) => void;
}

const Child = React.memo(({ onUpdate }: ChildProps) => {
  // Component implementation
});

function Parent(): JSX.Element {
  const [value, setValue] = useState("");

  // Without useCallback, new function created on every render
  // causing Child to re-render even with React.memo
  const handleUpdate = useCallback((newValue: string) => {
    setValue(newValue);
    api.save(newValue);
  }, []);  // Empty deps because setValue is stable

  return <Child onUpdate={handleUpdate} />;
}

// WRONG: Overusing useCallback
function Component(): JSX.Element {
  // Not needed - this callback doesn't go to child components
  const handleClick = useCallback(() => {
    console.log("Clicked");
  }, []);

  return <button onClick={handleClick}>Click</button>;
}

// CORRECT: Don't wrap unless passed to memoized components
function Component(): JSX.Element {
  const handleClick = (): void => {
    console.log("Clicked");
  };

  return <button onClick={handleClick}>Click</button>;
}

// CORRECT: Include all dependencies
function SearchInput({ onSearch }: { onSearch: (q: string) => void }) {
  const [query, setQuery] = useState("");

  const handleSearch = useCallback(() => {
    onSearch(query);
  }, [query, onSearch]);  // Include all used values

  return (
    <input
      value={query}
      onChange={e => setQuery(e.target.value)}
      onKeyDown={e => e.key === "Enter" && handleSearch()}
    />
  );
}
```

### useMemo

```typescript
// CORRECT: Memoize expensive computations
function DataTable({ data }: { data: Item[] }): JSX.Element {
  const sortedData = useMemo(() => {
    return [...data].sort((a, b) => a.name.localeCompare(b.name));
  }, [data]);

  return <Table data={sortedData} />;
}

// WRONG: Memoizing cheap operations
function Component({ count }: { count: number }): JSX.Element {
  const doubled = useMemo(() => count * 2, [count]);  // Unnecessary
  return <div>{doubled}</div>;
}

// CORRECT: Memoize referential equality
function Component({ filters }: { filters: string[] }): JSX.Element {
  // Create stable object reference
  const filterConfig = useMemo(() => ({
    include: filters,
    caseSensitive: true,
  }), [filters]);

  return <FilteredList config={filterConfig} />;
}

// CORRECT: Memoize derived complex state
function useFilteredUsers(users: User[], query: string): User[] {
  return useMemo(() => {
    if (!query) return users;

    const lowerQuery = query.toLowerCase();
    return users.filter(user =>
      user.name.toLowerCase().includes(lowerQuery) ||
      user.email.toLowerCase().includes(lowerQuery)
    );
  }, [users, query]);
}
```

### useRef

```typescript
// CORRECT: DOM reference
function TextInput(): JSX.Element {
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  return <input ref={inputRef} />;
}

// CORRECT: Store mutable value without triggering re-render
function Timer(): JSX.Element {
  const intervalRef = useRef<number | null>(null);
  const [count, setCount] = useState(0);

  useEffect(() => {
    intervalRef.current = window.setInterval(() => {
      setCount(c => c + 1);
    }, 1000);

    return () => {
      if (intervalRef.current !== null) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  return <div>Count: {count}</div>;
}

// CORRECT: Store previous value
function usePrevious<T>(value: T): T | undefined {
  const ref = useRef<T>();

  useEffect(() => {
    ref.current = value;
  }, [value]);

  return ref.current;
}

function Component({ count }: { count: number }): JSX.Element {
  const prevCount = usePrevious(count);

  return (
    <div>
      Current: {count}, Previous: {prevCount ?? "none"}
    </div>
  );
}

// WRONG: Using ref for state that should trigger re-render
function Counter(): JSX.Element {
  const countRef = useRef(0);

  const increment = (): void => {
    countRef.current++;  // Component won't re-render
  };

  return <div>{countRef.current}</div>;  // Stale value
}
```

### Custom Hooks

```typescript
// CORRECT: Custom hook for data fetching
interface UseFetchResult<T> {
  data: T | null;
  error: Error | null;
  isLoading: boolean;
  refetch: () => void;
}

function useFetch<T>(url: string): UseFetchResult<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [trigger, setTrigger] = useState(0);

  useEffect(() => {
    const controller = new AbortController();

    async function fetchData(): Promise<void> {
      setIsLoading(true);
      setError(null);

      try {
        const response = await fetch(url, { signal: controller.signal });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const json = await response.json();
        setData(json);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          return;
        }
        setError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setIsLoading(false);
      }
    }

    fetchData();

    return () => controller.abort();
  }, [url, trigger]);

  const refetch = useCallback(() => {
    setTrigger(t => t + 1);
  }, []);

  return { data, error, isLoading, refetch };
}

// CORRECT: Custom hook for form state
interface UseFormOptions<T> {
  initialValues: T;
  onSubmit: (values: T) => void | Promise<void>;
  validate?: (values: T) => Partial<Record<keyof T, string>>;
}

function useForm<T extends Record<string, any>>({
  initialValues,
  onSubmit,
  validate,
}: UseFormOptions<T>) {
  const [values, setValues] = useState<T>(initialValues);
  const [errors, setErrors] = useState<Partial<Record<keyof T, string>>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = useCallback((field: keyof T, value: any) => {
    setValues(prev => ({ ...prev, [field]: value }));
    setErrors(prev => ({ ...prev, [field]: undefined }));
  }, []);

  const handleSubmit = useCallback(async (
    event: React.FormEvent
  ): Promise<void> => {
    event.preventDefault();

    if (validate) {
      const validationErrors = validate(values);
      if (Object.keys(validationErrors).length > 0) {
        setErrors(validationErrors);
        return;
      }
    }

    setIsSubmitting(true);
    try {
      await onSubmit(values);
    } finally {
      setIsSubmitting(false);
    }
  }, [values, validate, onSubmit]);

  const reset = useCallback(() => {
    setValues(initialValues);
    setErrors({});
  }, [initialValues]);

  return {
    values,
    errors,
    isSubmitting,
    handleChange,
    handleSubmit,
    reset,
  };
}

// Usage
function LoginForm(): JSX.Element {
  const { values, errors, isSubmitting, handleChange, handleSubmit } = useForm({
    initialValues: { email: "", password: "" },
    onSubmit: async (values) => {
      await login(values.email, values.password);
    },
    validate: (values) => {
      const errors: Partial<Record<keyof typeof values, string>> = {};
      if (!values.email) errors.email = "Required";
      if (!values.password) errors.password = "Required";
      return errors;
    },
  });

  return (
    <form onSubmit={handleSubmit}>
      <input
        value={values.email}
        onChange={e => handleChange("email", e.target.value)}
      />
      {errors.email && <span>{errors.email}</span>}
      <button disabled={isSubmitting}>Submit</button>
    </form>
  );
}
```

## State Management

### Zustand Patterns

**This project uses Zustand for global state.**

```typescript
import { create } from "zustand";
import { devtools, persist } from "zustand/middleware";
import { immer } from "zustand/middleware/immer";

// CORRECT: Typed store with actions
interface User {
  id: string;
  name: string;
  email: string;
}

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  updateUser: (updates: Partial<User>) => void;
}

const useAuthStore = create<AuthState>()(
  devtools(
    persist(
      (set) => ({
        user: null,
        isAuthenticated: false,

        login: async (email, password) => {
          const user = await api.login(email, password);
          set({ user, isAuthenticated: true });
        },

        logout: () => {
          set({ user: null, isAuthenticated: false });
        },

        updateUser: (updates) => {
          set((state) => ({
            user: state.user ? { ...state.user, ...updates } : null,
          }));
        },
      }),
      { name: "auth-storage" }
    )
  )
);

// CORRECT: Using immer for complex state updates
interface TodoState {
  todos: Todo[];
  addTodo: (text: string) => void;
  toggleTodo: (id: string) => void;
  removeTodo: (id: string) => void;
}

const useTodoStore = create<TodoState>()(
  immer((set) => ({
    todos: [],

    addTodo: (text) =>
      set((state) => {
        state.todos.push({ id: nanoid(), text, completed: false });
      }),

    toggleTodo: (id) =>
      set((state) => {
        const todo = state.todos.find((t) => t.id === id);
        if (todo) {
          todo.completed = !todo.completed;
        }
      }),

    removeTodo: (id) =>
      set((state) => {
        state.todos = state.todos.filter((t) => t.id !== id);
      }),
  }))
);

// CORRECT: Selective subscription to avoid unnecessary re-renders
function UserProfile(): JSX.Element {
  // Only re-renders when user changes
  const user = useAuthStore((state) => state.user);

  return <div>{user?.name}</div>;
}

// CORRECT: Subscribing to multiple values with shallow comparison
import { shallow } from "zustand/shallow";

function TodoStats(): JSX.Element {
  const { total, completed } = useTodoStore(
    (state) => ({
      total: state.todos.length,
      completed: state.todos.filter((t) => t.completed).length,
    }),
    shallow
  );

  return <div>{completed}/{total} completed</div>;
}

// WRONG: Subscribing to entire store
function TodoList(): JSX.Element {
  const state = useTodoStore();  // Re-renders on ANY state change
  return <div>{state.todos.length}</div>;
}
```

### Context API (Use Sparingly)

```typescript
// CORRECT: Context for dependency injection, not frequent updates
interface ThemeContextValue {
  theme: "light" | "dark";
  toggleTheme: () => void;
}

const ThemeContext = React.createContext<ThemeContextValue | null>(null);

function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
}

function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [theme, setTheme] = useState<"light" | "dark">("light");

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === "light" ? "dark" : "light"));
  }, []);

  const value = useMemo(() => ({ theme, toggleTheme }), [theme, toggleTheme]);

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
}

// WRONG: Context for frequently updating data
// Use Zustand or other state management instead
```

## Performance

### React.memo

```typescript
// CORRECT: Memoize expensive components
interface ListItemProps {
  item: Item;
  onSelect: (id: string) => void;
}

const ListItem = React.memo(({ item, onSelect }: ListItemProps) => {
  return (
    <div onClick={() => onSelect(item.id)}>
      {item.name}
    </div>
  );
});

// CORRECT: Custom comparison function
const ListItem = React.memo(
  ({ item, onSelect }: ListItemProps) => {
    return <div onClick={() => onSelect(item.id)}>{item.name}</div>;
  },
  (prev, next) => {
    // Only re-render if item.id changed
    return prev.item.id === next.item.id;
  }
);

// WRONG: Memoizing everything
const SimpleButton = React.memo(({ onClick }: { onClick: () => void }) => {
  return <button onClick={onClick}>Click</button>;
});
// Not worth the overhead for simple components
```

### Code Splitting

```typescript
// CORRECT: Route-based code splitting
import { lazy, Suspense } from "react";

const Dashboard = lazy(() => import("./pages/Dashboard"));
const Settings = lazy(() => import("./pages/Settings"));

function App(): JSX.Element {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <Routes>
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/settings" element={<Settings />} />
      </Routes>
    </Suspense>
  );
}

// CORRECT: Component-based code splitting
const HeavyChart = lazy(() => import("./components/HeavyChart"));

function Analytics(): JSX.Element {
  const [showChart, setShowChart] = useState(false);

  return (
    <div>
      <button onClick={() => setShowChart(true)}>Show Chart</button>
      {showChart && (
        <Suspense fallback={<div>Loading chart...</div>}>
          <HeavyChart />
        </Suspense>
      )}
    </div>
  );
}
```

### List Virtualization

```typescript
// CORRECT: Use virtual scrolling for long lists
import { FixedSizeList } from "react-window";

interface VirtualListProps {
  items: Item[];
}

function VirtualList({ items }: VirtualListProps): JSX.Element {
  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => (
    <div style={style}>
      {items[index].name}
    </div>
  );

  return (
    <FixedSizeList
      height={600}
      itemCount={items.length}
      itemSize={50}
      width="100%"
    >
      {Row}
    </FixedSizeList>
  );
}
```

## Testing

### Component Testing with Vitest + Testing Library

```typescript
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";

// CORRECT: Test user interactions
describe("LoginForm", () => {
  it("calls onSubmit with form values", async () => {
    const onSubmit = vi.fn();
    const user = userEvent.setup();

    render(<LoginForm onSubmit={onSubmit} />);

    await user.type(screen.getByLabelText(/email/i), "test@example.com");
    await user.type(screen.getByLabelText(/password/i), "password123");
    await user.click(screen.getByRole("button", { name: /submit/i }));

    expect(onSubmit).toHaveBeenCalledWith({
      email: "test@example.com",
      password: "password123",
    });
  });

  it("displays validation error", async () => {
    const user = userEvent.setup();
    render(<LoginForm onSubmit={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: /submit/i }));

    expect(screen.getByText(/email is required/i)).toBeInTheDocument();
  });
});

// CORRECT: Test async behavior
describe("UserList", () => {
  it("loads and displays users", async () => {
    const users = [
      { id: "1", name: "Alice" },
      { id: "2", name: "Bob" },
    ];

    vi.mocked(api.getUsers).mockResolvedValue(users);

    render(<UserList />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("Alice")).toBeInTheDocument();
    });

    expect(screen.getByText("Bob")).toBeInTheDocument();
  });
});

// CORRECT: Test custom hooks
import { renderHook } from "@testing-library/react";

describe("useFetch", () => {
  it("fetches data", async () => {
    const data = { id: 1, name: "Test" };
    vi.mocked(fetch).mockResolvedValue(
      new Response(JSON.stringify(data))
    );

    const { result } = renderHook(() => useFetch("/api/data"));

    expect(result.current.isLoading).toBe(true);

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual(data);
    expect(result.current.error).toBeNull();
  });
});
```

## Ink-Specific Patterns

**This project uses Ink for terminal UI components.**

### Basic Ink Components

```typescript
import { Box, Text, useInput, useApp } from "ink";
import { useState } from "react";

// CORRECT: Terminal text display
function Header(): JSX.Element {
  return (
    <Box borderStyle="round" borderColor="cyan" padding={1}>
      <Text bold color="cyan">
        Application Title
      </Text>
    </Box>
  );
}

// CORRECT: Layout with Box
function Dashboard(): JSX.Element {
  return (
    <Box flexDirection="column" padding={1}>
      <Box marginBottom={1}>
        <Text bold>Dashboard</Text>
      </Box>
      <Box borderStyle="single" padding={1}>
        <Text>Content goes here</Text>
      </Box>
    </Box>
  );
}
```

### Ink Input Handling

```typescript
// CORRECT: useInput hook
function Menu(): JSX.Element {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const items = ["Option 1", "Option 2", "Option 3"];
  const { exit } = useApp();

  useInput((input, key) => {
    if (key.upArrow) {
      setSelectedIndex((prev) => Math.max(0, prev - 1));
    }

    if (key.downArrow) {
      setSelectedIndex((prev) => Math.min(items.length - 1, prev + 1));
    }

    if (key.return) {
      console.log(`Selected: ${items[selectedIndex]}`);
    }

    if (input === "q") {
      exit();
    }
  });

  return (
    <Box flexDirection="column">
      {items.map((item, index) => (
        <Text key={item} inverse={index === selectedIndex}>
          {item}
        </Text>
      ))}
    </Box>
  );
}
```

### Ink Constraints

```typescript
// WRONG: Using DOM-specific props
function BadInkComponent(): JSX.Element {
  return (
    <Box className="container">  {/* className doesn't work in Ink */}
      <Text onClick={() => {}}>Click</Text>  {/* No onClick in terminal */}
    </Box>
  );
}

// CORRECT: Ink-specific styling
function GoodInkComponent(): JSX.Element {
  return (
    <Box borderStyle="round" padding={1} borderColor="green">
      <Text color="green" bold>
        Success
      </Text>
    </Box>
  );
}

// CORRECT: Handle terminal dimensions
import { useStdout } from "ink";

function ResponsiveBox(): JSX.Element {
  const { stdout } = useStdout();
  const width = stdout.columns;

  return (
    <Box width={width - 4} borderStyle="single">
      <Text>Responsive content</Text>
    </Box>
  );
}
```

## Sharp Edges

### Stale Closures

```typescript
// WRONG: Stale closure in setInterval
function Timer(): JSX.Element {
  const [count, setCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setCount(count + 1);  // Always uses initial count (0)
    }, 1000);

    return () => clearInterval(interval);
  }, []);  // Empty deps - closure captures initial count

  return <div>{count}</div>;
}

// CORRECT: Use functional update
function Timer(): JSX.Element {
  const [count, setCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setCount((prev) => prev + 1);  // Uses current state
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  return <div>{count}</div>;
}
```

### Dependency Array Issues

```typescript
// WRONG: Missing dependencies
function UserProfile({ userId }: { userId: string }): JSX.Element {
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUser(userId).then(setUser);
  }, []);  // Missing userId - won't refetch on change

  return <div>{user?.name}</div>;
}

// CORRECT: Include all dependencies
useEffect(() => {
  fetchUser(userId).then(setUser);
}, [userId]);

// WRONG: Object/array in dependencies
function Component({ config }: { config: Config }): JSX.Element {
  useEffect(() => {
    applyConfig(config);
  }, [config]);  // New object on every render - infinite loop

  return <div />;
}

// CORRECT: Use primitive dependencies or useMemo
function Component({ config }: { config: Config }): JSX.Element {
  const configJson = JSON.stringify(config);

  useEffect(() => {
    applyConfig(JSON.parse(configJson));
  }, [configJson]);  // String comparison works

  return <div />;
}
```

### Infinite Render Loops

```typescript
// WRONG: State update in render
function Component(): JSX.Element {
  const [count, setCount] = useState(0);

  setCount(count + 1);  // NEVER - infinite loop

  return <div>{count}</div>;
}

// WRONG: useEffect without dependencies
function Component(): JSX.Element {
  const [data, setData] = useState(null);

  useEffect(() => {
    setData({ value: Math.random() });
  });  // No deps array - runs after every render

  return <div>{data?.value}</div>;
}

// CORRECT: Effect with proper dependencies
function Component(): JSX.Element {
  const [data, setData] = useState(null);

  useEffect(() => {
    setData({ value: Math.random() });
  }, []);  // Empty array - runs once

  return <div>{data?.value}</div>;
}
```

### Cleanup Function Gotchas

```typescript
// WRONG: Missing cleanup
function Component(): JSX.Element {
  useEffect(() => {
    const subscription = api.subscribe(handleData);
    // Missing cleanup - memory leak
  }, []);

  return <div />;
}

// CORRECT: Proper cleanup
function Component(): JSX.Element {
  useEffect(() => {
    const subscription = api.subscribe(handleData);

    return () => {
      subscription.unsubscribe();
    };
  }, []);

  return <div />;
}

// WRONG: Async cleanup
function Component(): JSX.Element {
  useEffect(() => {
    return async () => {  // WRONG - cleanup can't be async
      await api.cleanup();
    };
  }, []);

  return <div />;
}

// CORRECT: Handle async in cleanup
function Component(): JSX.Element {
  useEffect(() => {
    return () => {
      void api.cleanup();  // Fire and forget
      // Or use an IIFE if you need to handle the promise
      (async () => {
        await api.cleanup();
      })();
    };
  }, []);

  return <div />;
}
```

### Key Prop Issues

```typescript
// WRONG: Using index as key
function List({ items }: { items: string[] }): JSX.Element {
  return (
    <ul>
      {items.map((item, index) => (
        <li key={index}>{item}</li>  // Breaks on reorder
      ))}
    </ul>
  );
}

// CORRECT: Use stable identifier
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.id}>{item.name}</li>
      ))}
    </ul>
  );
}

// WRONG: Non-unique keys
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item) => (
        <li key={item.category}>{item.name}</li>  // Multiple items per category
      ))}
    </ul>
  );
}

// CORRECT: Compound key if no unique ID
function List({ items }: { items: Item[] }): JSX.Element {
  return (
    <ul>
      {items.map((item, index) => (
        <li key={`${item.category}-${index}`}>{item.name}</li>
      ))}
    </ul>
  );
}
```
