# Code Conventions: golang-zod

## 1. Naming & Style
* **Package**: Single root package `z` (idiomatic import: `github.com/leandroluk/golang-zod/z` or root `github.com/leandroluk/golang-zod` as `z`).
* **Constructors**: Top-level exported functions in PascalCase matching Zod v4: `z.String()`, `z.Email()`, `z.UUID()`, `z.Int()`, `z.Object()`.
* **Method Chaining**: All modifier methods return a new or modified copy: `.Min(2)`, `.Max(100)`, `.Trim()`, `.Optional()`.
* **Errors as Values**: Return native Go `error` / `*ZodError` as the second return value `(T, error)` or nil on success.

## 2. Concurrency & Memory
* Schemas are read-only once constructed.
* Never mutate shared global state during `.Parse()` executions.
* Leverage `sync.Pool` for `ParseContext` and `Issue` buffers.

## 3. Requirement Traceability
* Every feature specification defines `@spec REQ-ID` requirements.
* Code implementations must reference their corresponding requirement in comments:
  ```go
  // @spec REQ-CORE-001
  type Value struct { ... }
  ```
