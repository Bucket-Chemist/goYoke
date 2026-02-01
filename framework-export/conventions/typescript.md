# TypeScript Conventions

## System Constraints (CRITICAL)

**This system targets TypeScript 5.0+ with strict mode ALWAYS enabled.**

All TypeScript code must:
1. Pass `tsc --noEmit` with zero errors under strict configuration
2. Never use `any` type - prefer `unknown`, proper narrowing, or generics
3. Enable all strict flags in tsconfig.json
4. Use modern ES2022+ features (top-level await, private class fields, etc.)

## Type System Mastery

### Strict Configuration (Non-Negotiable)

```json
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true,
    "noPropertyAccessFromIndexSignature": true,
    "noFallthroughCasesInSwitch": true,
    "forceConsistentCasingInFileNames": true,
    "skipLibCheck": false,
    "allowUnusedLabels": false,
    "allowUnreachableCode": false,
    "verbatimModuleSyntax": true
  }
}
```

### Type vs Interface

**Rule:** Use `type` for unions, intersections, primitives, tuples. Use `interface` for object shapes that may be extended.

```typescript
// CORRECT: Use type for unions and complex compositions
type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

type ID = string | number;
type Coordinates = [number, number];

// CORRECT: Use interface for extensible object shapes
interface User {
  id: string;
  name: string;
  email: string;
}

interface AdminUser extends User {
  permissions: string[];
}

// WRONG: Using interface for unions
interface Status = "pending" | "active";  // Won't compile

// WRONG: Using type when extension is expected
type BaseConfig = { port: number };
type ExtendedConfig = BaseConfig & { host: string };  // Works but interface is clearer
```

### Discriminated Unions

**Always include a discriminant field for tagged unions.**

```typescript
// CORRECT: Discriminated union with exhaustive checking
type ApiResponse<T> =
  | { status: "loading" }
  | { status: "success"; data: T }
  | { status: "error"; error: Error };

function handleResponse<T>(response: ApiResponse<T>): void {
  switch (response.status) {
    case "loading":
      console.log("Loading...");
      break;
    case "success":
      console.log(response.data);  // TypeScript knows data exists
      break;
    case "error":
      console.error(response.error);  // TypeScript knows error exists
      break;
    default:
      // Exhaustiveness check - will error if new status added
      const _exhaustive: never = response;
      throw new Error(`Unhandled status: ${_exhaustive}`);
  }
}

// WRONG: Union without discriminant
type BadResponse<T> =
  | { data: T }
  | { error: Error };
// Can't tell which is which at runtime
```

### Branded Types (Nominal Typing)

Use branded types to prevent mixing semantically different primitive types.

```typescript
// Define branded types
type UserId = string & { readonly __brand: "UserId" };
type Email = string & { readonly __brand: "Email" };
type URL = string & { readonly __brand: "URL" };

// Constructor functions with validation
function createUserId(id: string): UserId {
  if (!id.match(/^usr_[a-zA-Z0-9]+$/)) {
    throw new Error(`Invalid user ID: ${id}`);
  }
  return id as UserId;
}

function createEmail(email: string): Email {
  if (!email.includes("@")) {
    throw new Error(`Invalid email: ${email}`);
  }
  return email as Email;
}

// Usage
function getUserById(id: UserId): User { /* ... */ }

const id = createUserId("usr_123");
const email = createEmail("user@example.com");

getUserById(id);  // OK
getUserById(email);  // Type error - can't pass Email where UserId expected
getUserById("usr_123");  // Type error - must use constructor
```

### Conditional Types

```typescript
// Extract function parameter types
type FirstParameter<T> = T extends (first: infer P, ...args: any[]) => any
  ? P
  : never;

type SecondParameter<T> = T extends (
  first: any,
  second: infer P,
  ...args: any[]
) => any
  ? P
  : never;

// Example usage
function example(a: string, b: number): void {}
type First = FirstParameter<typeof example>;  // string
type Second = SecondParameter<typeof example>;  // number

// Conditional return types
type UnwrapPromise<T> = T extends Promise<infer U> ? U : T;

type A = UnwrapPromise<Promise<string>>;  // string
type B = UnwrapPromise<number>;           // number

// Recursive conditional types
type Awaited<T> = T extends Promise<infer U>
  ? Awaited<U>
  : T;

type Nested = Awaited<Promise<Promise<Promise<string>>>>;  // string
```

### Mapped Types

```typescript
// Make all properties optional recursively
type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends object ? DeepPartial<T[P]> : T[P];
};

// Make all properties readonly recursively
type DeepReadonly<T> = {
  readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

// Make specific properties required
type RequireKeys<T, K extends keyof T> = T & Required<Pick<T, K>>;

interface Config {
  host?: string;
  port?: number;
  debug?: boolean;
}

type ProductionConfig = RequireKeys<Config, "host" | "port">;
// { host: string; port: number; debug?: boolean }

// Remove null and undefined from all properties
type NonNullableProps<T> = {
  [P in keyof T]: NonNullable<T[P]>;
};
```

### Template Literal Types

```typescript
// String pattern matching
type HTTPMethod = "GET" | "POST" | "PUT" | "DELETE";
type Endpoint = `/api/${string}`;
type Route = `${HTTPMethod} ${Endpoint}`;

const route: Route = "GET /api/users";  // OK
const invalid: Route = "PATCH /api/users";  // Error: PATCH not in HTTPMethod

// Derive related types
type EventName = "click" | "focus" | "blur";
type HandlerName = `on${Capitalize<EventName>}`;
// "onClick" | "onFocus" | "onBlur"

// CSS property names
type CSSProperty = "background-color" | "font-size" | "margin-top";
type CSSInJS = {
  [K in CSSProperty as `${K}` | Uncapitalize<K>]: string;
};
// { "background-color": string; "font-size": string; ... }
```

### Utility Types Mastery

```typescript
// Standard utilities
type User = {
  id: string;
  name: string;
  email: string;
  password: string;
  createdAt: Date;
};

// Pick specific properties
type UserPublic = Pick<User, "id" | "name" | "email">;

// Omit specific properties
type UserUpdate = Omit<User, "id" | "createdAt">;

// Make all properties optional
type UserPartial = Partial<User>;

// Make all properties required
type UserRequired = Required<User>;

// Create record type
type UserRoles = Record<string, "admin" | "user" | "guest">;

// Extract return type
function getUser(): User { /* ... */ }
type GetUserReturn = ReturnType<typeof getUser>;  // User

// Extract parameter types
type GetUserParams = Parameters<typeof getUserById>;  // [UserId]

// Custom utility: Nullable
type Nullable<T> = T | null;

// Custom utility: ValueOf (get union of all property values)
type ValueOf<T> = T[keyof T];
type UserValue = ValueOf<User>;  // string | Date
```

## Modern Patterns

### Never Use `any`

```typescript
// WRONG: Using any
function parseJSON(json: string): any {
  return JSON.parse(json);
}

// CORRECT: Use unknown and narrow
function parseJSON(json: string): unknown {
  return JSON.parse(json);
}

// Then narrow the type
function getUser(json: string): User {
  const data = parseJSON(json);

  // Type guard
  if (isUser(data)) {
    return data;
  }
  throw new Error("Invalid user data");
}

function isUser(value: unknown): value is User {
  return (
    typeof value === "object" &&
    value !== null &&
    "id" in value &&
    "name" in value &&
    "email" in value
  );
}

// CORRECT: Use generics
function identity<T>(value: T): T {
  return value;
}

// CORRECT: Accept specific union
function log(value: string | number | boolean): void {
  console.log(value);
}
```

### Type Guards and Narrowing

```typescript
// Built-in type guards
function processValue(value: string | number): string {
  if (typeof value === "string") {
    return value.toUpperCase();  // TypeScript knows it's string
  }
  return value.toFixed(2);  // TypeScript knows it's number
}

// Custom type guards
interface Cat {
  type: "cat";
  meow(): void;
}

interface Dog {
  type: "dog";
  bark(): void;
}

type Animal = Cat | Dog;

function isCat(animal: Animal): animal is Cat {
  return animal.type === "cat";
}

function handleAnimal(animal: Animal): void {
  if (isCat(animal)) {
    animal.meow();  // TypeScript knows it's Cat
  } else {
    animal.bark();  // TypeScript knows it's Dog
  }
}

// Assertion functions
function assertIsString(value: unknown): asserts value is string {
  if (typeof value !== "string") {
    throw new Error(`Expected string, got ${typeof value}`);
  }
}

function processInput(input: unknown): string {
  assertIsString(input);
  return input.toUpperCase();  // TypeScript knows input is string
}
```

### Type-Only Imports

**Always use type-only imports when importing only types.**

```typescript
// CORRECT: Type-only import
import type { User, Config } from "./types";
import { createUser } from "./user";

// CORRECT: Mixed import
import { type User, createUser } from "./user";

// WRONG: Regular import for types only
import { User, Config } from "./types";  // May cause circular dependency issues
```

### Const Assertions

```typescript
// CORRECT: Use const assertion for literal types
const config = {
  apiUrl: "https://api.example.com",
  timeout: 5000,
  retries: 3,
} as const;

type Config = typeof config;
// {
//   readonly apiUrl: "https://api.example.com";
//   readonly timeout: 5000;
//   readonly retries: 3;
// }

// CORRECT: Use for tuple types
const point = [10, 20] as const;
type Point = typeof point;  // readonly [10, 20]

// CORRECT: Use for enum-like objects
const Status = {
  PENDING: "pending",
  ACTIVE: "active",
  INACTIVE: "inactive",
} as const;

type StatusValue = typeof Status[keyof typeof Status];
// "pending" | "active" | "inactive"

// WRONG: Without const assertion
const badConfig = {
  apiUrl: "https://api.example.com",  // Type: string (too broad)
  timeout: 5000,  // Type: number (too broad)
};
```

### Satisfies Operator (TypeScript 4.9+)

```typescript
// CORRECT: Validate type without widening
type Colors = "red" | "green" | "blue";

const favoriteColors = {
  alice: "red",
  bob: "green",
  charlie: "blue",
} satisfies Record<string, Colors>;

favoriteColors.alice;  // Type: "red" (literal, not Colors)

// WRONG: Using type annotation widens types
const badColors: Record<string, Colors> = {
  alice: "red",  // Type: Colors (widened)
};

// CORRECT: Validate array elements
const endpoints = [
  { path: "/api/users", method: "GET" },
  { path: "/api/posts", method: "POST" },
] satisfies Array<{ path: string; method: "GET" | "POST" }>;

endpoints[0].method;  // Type: "GET" (literal preserved)
```

## Error Handling

### Result Pattern

**Never throw errors for expected failure cases. Use Result type.**

```typescript
type Result<T, E = Error> =
  | { ok: true; value: T }
  | { ok: false; error: E };

// Helper constructors
function Ok<T>(value: T): Result<T, never> {
  return { ok: true, value };
}

function Err<E>(error: E): Result<never, E> {
  return { ok: false, error };
}

// Usage
function parseNumber(input: string): Result<number, string> {
  const num = Number(input);
  if (Number.isNaN(num)) {
    return Err(`Invalid number: ${input}`);
  }
  return Ok(num);
}

// Chain operations
function divide(a: number, b: number): Result<number, string> {
  if (b === 0) {
    return Err("Division by zero");
  }
  return Ok(a / b);
}

function calculate(input: string): Result<number, string> {
  const numResult = parseNumber(input);
  if (!numResult.ok) {
    return numResult;
  }

  return divide(100, numResult.value);
}

// Pattern match on result
const result = calculate("5");
if (result.ok) {
  console.log(`Result: ${result.value}`);
} else {
  console.error(`Error: ${result.error}`);
}
```

### Custom Error Classes

```typescript
// Base error class with context
abstract class AppError extends Error {
  abstract readonly code: string;
  readonly timestamp: Date;
  readonly context?: Record<string, unknown>;

  constructor(message: string, context?: Record<string, unknown>) {
    super(message);
    this.name = this.constructor.name;
    this.timestamp = new Date();
    this.context = context;
    Error.captureStackTrace(this, this.constructor);
  }
}

class ValidationError extends AppError {
  readonly code = "VALIDATION_ERROR" as const;

  constructor(
    message: string,
    public readonly field: string,
    public readonly value: unknown
  ) {
    super(message, { field, value });
  }
}

class NotFoundError extends AppError {
  readonly code = "NOT_FOUND" as const;

  constructor(
    message: string,
    public readonly resource: string,
    public readonly id: string
  ) {
    super(message, { resource, id });
  }
}

class UnauthorizedError extends AppError {
  readonly code = "UNAUTHORIZED" as const;
}

// Type guard for error handling
function isAppError(error: unknown): error is AppError {
  return error instanceof AppError;
}

// Usage
try {
  throw new NotFoundError("User not found", "User", "123");
} catch (error) {
  if (error instanceof NotFoundError) {
    console.error(`${error.code}: ${error.message}`);
    console.error(`Resource: ${error.resource}, ID: ${error.id}`);
  } else if (isAppError(error)) {
    console.error(`${error.code}: ${error.message}`);
  } else {
    console.error("Unknown error:", error);
  }
}
```

### Exhaustive Checking

**Always use exhaustive checking for union types.**

```typescript
type Status = "idle" | "loading" | "success" | "error";

function handleStatus(status: Status): string {
  switch (status) {
    case "idle":
      return "Ready";
    case "loading":
      return "Loading...";
    case "success":
      return "Done!";
    case "error":
      return "Failed";
    default:
      // This ensures TypeScript errors if a new status is added
      const _exhaustive: never = status;
      throw new Error(`Unhandled status: ${_exhaustive}`);
  }
}

// Alternative: function-based exhaustive check
function assertUnreachable(value: never): never {
  throw new Error(`Unexpected value: ${value}`);
}

function handleStatus2(status: Status): string {
  switch (status) {
    case "idle":
      return "Ready";
    case "loading":
      return "Loading...";
    case "success":
      return "Done!";
    case "error":
      return "Failed";
    default:
      return assertUnreachable(status);
  }
}
```

## Async Patterns

### Promise Best Practices

```typescript
// CORRECT: Proper error handling
async function fetchUser(id: string): Promise<User> {
  try {
    const response = await fetch(`/api/users/${id}`);
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return await response.json();
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to fetch user: ${error.message}`);
    }
    throw error;
  }
}

// WRONG: Unhandled promise rejection
async function badFetch(id: string): Promise<User> {
  const response = await fetch(`/api/users/${id}`);
  return response.json();  // No error handling
}

// CORRECT: Concurrent requests
async function fetchMultiple(ids: string[]): Promise<User[]> {
  const promises = ids.map(id => fetchUser(id));
  return Promise.all(promises);
}

// CORRECT: Fail-fast with Promise.all
async function fetchWithDependencies(
  userId: string
): Promise<{ user: User; posts: Post[] }> {
  const [user, posts] = await Promise.all([
    fetchUser(userId),
    fetchUserPosts(userId),
  ]);
  return { user, posts };
}

// CORRECT: Continue on partial failure with Promise.allSettled
async function fetchAllUsers(ids: string[]): Promise<User[]> {
  const results = await Promise.allSettled(ids.map(fetchUser));

  return results
    .filter((result): result is PromiseFulfilledResult<User> =>
      result.status === "fulfilled"
    )
    .map(result => result.value);
}
```

### AbortController Pattern

```typescript
// CORRECT: Cancelable fetch with timeout
async function fetchWithTimeout<T>(
  url: string,
  timeoutMs: number,
  signal?: AbortSignal
): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      signal: signal ?? controller.signal,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    return await response.json();
  } finally {
    clearTimeout(timeout);
  }
}

// Usage with external abort signal
const controller = new AbortController();

setTimeout(() => controller.abort(), 5000);

try {
  const data = await fetchWithTimeout<User>(
    "/api/user",
    3000,
    controller.signal
  );
  console.log(data);
} catch (error) {
  if (error instanceof DOMException && error.name === "AbortError") {
    console.log("Request aborted");
  } else {
    console.error("Request failed:", error);
  }
}
```

### Async Iteration

```typescript
// CORRECT: Async generator for pagination
async function* fetchAllPages<T>(
  baseUrl: string
): AsyncGenerator<T[], void, undefined> {
  let page = 1;
  let hasMore = true;

  while (hasMore) {
    const response = await fetch(`${baseUrl}?page=${page}`);
    const data: { items: T[]; hasMore: boolean } = await response.json();

    yield data.items;

    hasMore = data.hasMore;
    page++;
  }
}

// Usage
for await (const users of fetchAllPages<User>("/api/users")) {
  console.log(`Fetched ${users.length} users`);
  users.forEach(processUser);
}

// CORRECT: Async iterator pattern
class DataStream<T> {
  constructor(private readonly source: AsyncIterable<T>) {}

  async *filter(predicate: (item: T) => boolean): AsyncGenerator<T> {
    for await (const item of this.source) {
      if (predicate(item)) {
        yield item;
      }
    }
  }

  async *map<U>(mapper: (item: T) => U): AsyncGenerator<U> {
    for await (const item of this.source) {
      yield mapper(item);
    }
  }

  async collect(): Promise<T[]> {
    const result: T[] = [];
    for await (const item of this.source) {
      result.push(item);
    }
    return result;
  }
}
```

## Naming Conventions

| Element | Convention | Example |
|---------|------------|---------|
| Variable | camelCase | `userName`, `totalCount` |
| Function | camelCase | `fetchUser`, `calculateTotal` |
| Class | PascalCase | `User`, `HttpClient` |
| Interface | PascalCase | `User`, `Config` |
| Type Alias | PascalCase | `Result`, `ApiResponse` |
| Enum | PascalCase | `Status`, `HttpMethod` |
| Enum Member | UPPER_CASE or PascalCase | `PENDING`, `Success` |
| Generic Type Param | Single uppercase letter or PascalCase | `T`, `TKey`, `TValue` |
| Constant | UPPER_SNAKE_CASE | `MAX_RETRIES`, `API_URL` |
| Private Field | #prefix or _prefix | `#apiKey`, `_cache` |
| Boolean | is/has/can prefix | `isLoading`, `hasError`, `canSubmit` |

### Avoid Abbreviations

```typescript
// WRONG: Unclear abbreviations
function procUsr(usr: Usr): void {}

// CORRECT: Full words
function processUser(user: User): void {}

// ACCEPTABLE: Common abbreviations
type ID = string;
type URL = string;
type HTTP = "http" | "https";
```

## Tooling Configuration

### tsconfig.json (Complete)

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "allowImportingTsExtensions": true,
    "allowJs": false,
    "checkJs": false,
    "jsx": "react-jsx",
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "removeComments": true,
    "noEmit": true,
    "importHelpers": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "strict": true,
    "noImplicitAny": true,
    "strictNullChecks": true,
    "strictFunctionTypes": true,
    "strictBindCallApply": true,
    "strictPropertyInitialization": true,
    "noImplicitThis": true,
    "alwaysStrict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true,
    "noFallthroughCasesInSwitch": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "noPropertyAccessFromIndexSignature": true,
    "allowUnusedLabels": false,
    "allowUnreachableCode": false,
    "exactOptionalPropertyTypes": true,
    "skipLibCheck": false,
    "verbatimModuleSyntax": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist", "**/*.test.ts"]
}
```

### ESLint Configuration (typescript-eslint)

```javascript
// eslint.config.js
import eslint from "@eslint/js";
import tseslint from "typescript-eslint";

export default tseslint.config(
  eslint.configs.recommended,
  ...tseslint.configs.strictTypeChecked,
  ...tseslint.configs.stylisticTypeChecked,
  {
    languageOptions: {
      parserOptions: {
        project: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-unsafe-assignment": "error",
      "@typescript-eslint/no-unsafe-member-access": "error",
      "@typescript-eslint/no-unsafe-call": "error",
      "@typescript-eslint/no-unsafe-return": "error",
      "@typescript-eslint/explicit-function-return-type": "error",
      "@typescript-eslint/explicit-module-boundary-types": "error",
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/await-thenable": "error",
      "@typescript-eslint/no-misused-promises": "error",
      "@typescript-eslint/require-await": "error",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
      "@typescript-eslint/consistent-type-imports": [
        "error",
        { prefer: "type-imports" },
      ],
      "@typescript-eslint/consistent-type-exports": "error",
      "@typescript-eslint/no-import-type-side-effects": "error",
    },
  }
);
```

## Sharp Edges

### Type Coercion Gotchas

```typescript
// WRONG: Truthy/falsy assumptions
function getLength(arr: string[] | null): number {
  if (arr) {
    return arr.length;  // What about empty array?
  }
  return 0;
}

// CORRECT: Explicit null check
function getLength(arr: string[] | null): number {
  if (arr === null) {
    return 0;
  }
  return arr.length;
}

// WRONG: Implicit number coercion
const value = "5";
const result = value * 2;  // 10, but type is number (confusing)

// CORRECT: Explicit conversion
const value = "5";
const result = Number(value) * 2;
```

### Module Resolution Pitfalls

```typescript
// WRONG: Implicit any from missing types
import express from "express";  // May be 'any' if @types/express not installed

// CORRECT: Ensure types are installed
// npm install --save-dev @types/express

// WRONG: Circular dependency
// user.ts
import { Post } from "./post";
export interface User {
  posts: Post[];
}

// post.ts
import { User } from "./user";
export interface Post {
  author: User;  // Circular!
}

// CORRECT: Use type-only import or separate types file
// types.ts
export interface User {
  posts: Post[];
}
export interface Post {
  author: User;
}
```

### Declaration Merging Confusion

```typescript
// WRONG: Accidental namespace/interface merge
interface User {
  id: string;
}

namespace User {
  export function create(id: string): User {
    return { id };
  }
}

// This works but is confusing
const user = User.create("123");

// CORRECT: Separate namespace or use class
class UserFactory {
  static create(id: string): User {
    return { id };
  }
}
```

### Index Signature Traps

```typescript
// WRONG: Unsafe object access
interface Config {
  [key: string]: string;
}

const config: Config = { apiUrl: "https://api.example.com" };
const value = config.missingKey;  // Type: string (but actually undefined)

// CORRECT: Enable noUncheckedIndexedAccess
// With this flag, value is: string | undefined

// CORRECT: Use Record with explicit keys
type ConfigKey = "apiUrl" | "timeout" | "retries";
type Config = Record<ConfigKey, string>;

// Or use optional properties
interface Config {
  apiUrl: string;
  timeout?: string;
  retries?: string;
}
```

### Type Parameter Constraints

```typescript
// WRONG: Unconstrained generic loses information
function firstElement<T>(arr: T[]): T | undefined {
  return arr[0];
}

const nums = [1, 2, 3];
const first = firstElement(nums);  // Type: number | undefined (correct)

const value = firstElement([]);  // Type: never | undefined (useless)

// CORRECT: Constrain or provide default
function firstElement<T = never>(arr: T[]): T | undefined {
  return arr[0];
}

// CORRECT: Use extends for constraints
function getProperty<T, K extends keyof T>(obj: T, key: K): T[K] {
  return obj[key];
}

const user = { id: "1", name: "Alice" };
const name = getProperty(user, "name");  // Type: string
getProperty(user, "invalid");  // Type error
```

### Promise Type Issues

```typescript
// WRONG: Promise<void> swallows errors
async function process(): Promise<void> {
  throw new Error("Failed");  // Error silently swallowed if not awaited
}

process();  // No error, promise rejected but not handled

// CORRECT: Mark as floating or always await
void process();  // Explicit void
// or
process().catch(console.error);  // Handle rejection
// or
await process();  // Await the promise

// WRONG: Forgetting await in async function
async function getData(): Promise<number> {
  return fetchNumber();  // Returns Promise<Promise<number>>
}

// CORRECT: Always await promises
async function getData(): Promise<number> {
  return await fetchNumber();
}
```

### Enum Pitfalls

```typescript
// WRONG: Numeric enums allow invalid values
enum Status {
  Idle,
  Loading,
  Success,
}

const status: Status = 99;  // Valid but nonsense

// CORRECT: Use string enums or const objects
enum Status {
  Idle = "idle",
  Loading = "loading",
  Success = "success",
}

const status: Status = 99;  // Type error

// BETTER: Use const object
const Status = {
  Idle: "idle",
  Loading: "loading",
  Success: "success",
} as const;

type Status = typeof Status[keyof typeof Status];
```
