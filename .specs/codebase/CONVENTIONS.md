# Code Conventions: golang-zod

## Naming & style
- Single package `z`; exported API PascalCase matching Zod (`z.String()`, `z.Email()`, `z.Int()`).
- Initialisms uppercase: `GT`, `GTE`, `LT`, `LTE`, `URL`, `UUID`, `IPv4`, `HTTPURL`, `JWT`.
- Zod method aliases are kept as in Zod: `GTE`/`Min`, `LTE`/`Max`, `z.StrictObject`/`.Strict()`, `z.LooseObject`/`.Loose()`. No other duplicate forms.
- Checks that preserve `T` return the same concrete type (fluent). Modifiers that change `T` return wrapper types.
- Constructors that can fail on programmer error (invalid struct, bad regex) have a `Must*` variant only where a build-time error exists; schema construction otherwise never returns an error.

## Errors
- `(T, error)`; failures are `*ZodError`; use `errors.As(err, &zerr)`.
- Never return a typed-nil `*ZodError` as `error`.
- No `SafeParse` (Result is already `(T, error)`), no async variants (goroutines + `context.Context` instead).

## Concurrency & memory
- Schemas immutable after construction; chain methods copy.
- Never mutate shared globals during `Parse` (locale is read via atomic pointer).
- `parseCtx` from `sync.Pool`; reset fully on `Put`.

## Tests
- External test package `z_test`; table-driven; `go test -race ./...` must pass.
- Every behavior in a `REQ-*` has at least one test named `Test<REQID>_<what>` (e.g. `TestREQ_OBJ_002_Strict`).

## Requirement traceability
Implementations reference requirements in comments:
```go
// @spec REQ-CORE-003
type Issue struct { ... }
```
