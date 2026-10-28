# Spec: 06 — Modifiers, Refinements, Pipelines & Codecs

## Summary
Implements Milestone 5 per [INSIGHT.md](../../../INSIGHT.md) §2, §6:
1. **Presence & fallback modifiers (chained methods only; no `z.Optional(s)`)**, each returning a distinct wrapper type: `.Optional() OptionalSchema[T]` (output `Option[T]`), `.Nullable() NullableSchema[T]` (output `Nullable[T]`), `.Nullish() NullishSchema[T]` (output `Option[Nullable[T]]`), `.Default(v T)`, `.Prefault(v T)`, `.Catch(v T)`, `.Readonly()`.
2. **Refinements** (preserve `T`): `.Check(fn func(v T) []Issue)`, `.Refine(fn func(v T) bool, opts ...RefineOption)`, `.SuperRefine(fn func(v T, c *RefineCtx))`, plus context-aware `.RefineCtx(fn func(ctx context.Context, v T) bool)`.
3. **Transforms & pipelines**: `.Transform[U any](f func(T) (U, error)) Schema[U]` (generic method on concrete types only), `z.Preprocess[T any](f func(any) any, s Schema[T]) Schema[T]`, `.Pipe[U any](next Schema[U]) Schema[U]`, `.Input()`/`.Output()` on pipes.
4. **Codecs**: `z.Codec[A, B any](in Schema[A], out Schema[B], decode func(A) (B, error), encode func(B) (A, error)) *CodecSchema[A,B]` with `.Decode(in any, ctx ...context.Context) (B, error)` and `.Encode(v B, ctx ...context.Context) (A, error)` (same locale/context rules as parsing, REQ-CORE-013).
5. `z.StringBool(opts ...StringBoolOption) Schema[bool]` with truthy/falsy/case-sensitive options.
6. **Error customization**: `.Error(msg string)`, and `_With` variants `MinWith(n, func(e *CheckErr))` style (`.Message(msg)`, `.Abort(bool)`) on checks.

## Requirements
- REQ-PIPE-001: Implement the wrappers and semantics of INSIGHT §2.2.1: `.Optional()` absent key ok / `null` invalid; `.Nullable()` key required / `null` ok; `.Nullish()` both. `.Default(v)` substitutes ONLY when the input is **absent** (Zod's `undefined`), never when it is `null`: `null` goes to the inner schema (and fails unless the schema is `.Nullable()`/`.Nullish()`); it short-circuits, returning `v` WITHOUT running it through the inner schema; `.Prefault(v)` also applies only when the input is absent but substitutes `v` as the *pre-parse input*, so `v` is parsed by the inner schema (and transforms/checks run); `.Catch(v)` substitutes on any failure; `.Readonly()` is a marker (JSON Schema `readOnly`). A function form (`DefaultFunc(func() T)`, `CatchFunc(func(issue *Issue) T)`) mirrors Zod's value-or-function arguments. These verified rules come from Zod's `$ZodDefault`/`$ZodPrefault` (`payload.value === undefined`).
- REQ-PIPE-002: Wrapper types expose `Parse/ParseJSON/Decode/GetMeta` and DO NOT expose another `.Optional()`/`.Nullable()` (prevents the instantiation cycle). Checks such as `.Min()` must be applied BEFORE `.Optional()`.
- REQ-PIPE-003: `.Check`, `.Refine`, `.SuperRefine`, `.RefineCtx` on every concrete schema type (explicit methods). `.RefineCtx` receives the optional `ctx` passed to the parse call (`context.Background()` when none was passed); if it was cancelled, parse returns an issue of code `Custom` with message from locale (`cancelled`).
- REQ-PIPE-004: `.Transform[U]` (generic method on concrete types; permitted in Go 1.27.1), `z.Preprocess`, `.Pipe[U]`. Transform errors become `Custom` issues.
- REQ-PIPE-005: `z.Codec` with `.Decode` and `.Encode`; encode traverses pipes in reverse and propagates through objects/arrays/unions/optional wrappers. One-way steps (`.Transform`, `Preprocess`) return an encode error (`ErrEncodeUnsupported`), never a panic or `Issue` (Zod throws a runtime error here, not a `ZodError`). `.Default`, `.Prefault` and `.Catch` apply only in the decode direction (encoding an absent value against a default fails). Refinements and checks run in both directions: encode makes a first pass for type conformance and a second for refinements. Mutating transforms such as `.Trim()` apply in both directions. `Parse` on a codec behaves exactly like `Decode`.
- REQ-PIPE-006: `z.StringBool` with `Truthy(...)`, `Falsy(...)`, `CaseSensitive()` options; case-insensitive by default; on encode it emits `"true"`/`"false"` by default, or the FIRST element of the custom truthy/falsy list.
- REQ-PIPE-007: Error customization follows Zod's `error` parameter, which is a string OR a function: `.Error(msg string)` and `.ErrorFunc(func(issue *Issue) string)` (returning `""` falls through to the next level) on a schema, and the same on each String/Number check via trailing options `z.Message(msg)`, `z.MessageFunc(fn)`, `z.Abort(bool)` (e.g. `z.String().Min(5, z.Message("too short"))`; Array/Set size bounds use the schema-level error); a parse-level override via the context key (feature 01) is NOT part of this feature. Precedence (highest to lowest), as in Zod: (1) check-level error, (2) schema-level error, (3) per-parse error function, (4) global error function (`z.Config.UseError(func(issue *Issue) string)`), (5) locale catalog (feature 01).
- REQ-PIPE-008: Provide recipe codecs in `z.Codecs`: `JSONString`, `Base64Bytes`, `StringInt`, `StringNumber`, `EpochDate`.
- REQ-PIPE-009: No `SafeParse` or async variants. Concurrency is plain goroutines; cancellation and request scope (locale) via the optional trailing `ctx` parameter of every parse call.

## Affected Components
- `modifiers.go`, `refine.go`, `pipe.go`, `codec.go`; touches every concrete schema type (modifier method set).

## Out of Scope
- Fluent `.Optional().Optional()`, `z.Optional(s)` free function, JS `parseAsync`.

## Open Questions
- Whether to add `.Min()` forwarding on wrappers (default: no; check-before-wrap rule).
