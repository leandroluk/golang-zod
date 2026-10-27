# Spec: 01 — Core Engine, Typed Schema, Errors, Formatters & i18n

## Summary
Establishes the foundation of `golang-zod` per [INSIGHT.md](../../../INSIGHT.md): the unexported untyped engine (`node`, `parseCtx`), the sealed generic `Schema[T]` interface with the three input entry points (`Parse`, `ParseJSON`, `Decode`), the explicit presence types `Option[T]` and `Nullable[T]`, the metadata value (`Meta`) set through a builder function (`.Meta(func(m *z.MetaBuilder){...})`, including `Describe`), the 11-code Zod v4 issue taxonomy, `ZodError`, the 3 non-deprecated formatters (`Flatten`, `Treeify`, `Prettify`), and the i18n system (`en`, `pt-BR`, `es`) with global and per-parse locale. Pooled parse contexts via `sync.Pool`.

## Requirements
- REQ-CORE-001: Define the internal engine: `type node interface { parse(ctx *parseCtx, in any) (out any, ok bool); kind() nodeKind }`. Input `in` is JSON-like `any` (`nil`, `bool`, `string`, `json.Number`, `float64`, Go integers, `[]any`, `map[string]any`, `time.Time`, `[]byte`, `*big.Int`). All validation runs on `node`; typed schemas only add a final type assertion.
- REQ-CORE-002: Define the sealed `Schema[T any]` interface with exactly: `Parse(in any, ctx ...context.Context) (T, error)`, `ParseJSON(data []byte, ctx ...context.Context) (T, error)`, `Decode(r io.Reader, ctx ...context.Context) (T, error)`. The context is an **optional trailing variadic parameter** of the standard `context.Context` type (ADR-016/017; no custom alias or wrapper type): `schema.ParseJSON(body)` works with no context; `schema.ParseJSON(body, ctx)` uses it, with any `context.Context` such as `c.UserContext()`, `r.Context()`, `context.Background()`. Only the FIRST value is used; extra values are ignored. With no context the parse uses `context.Background()`: global locale, no cancellation. A nil context value is treated as absent. The interface also has `GetMeta() Meta` (the getter is `GetMeta`, not `Meta`, because the setter `.Meta(m Meta)` of REQ-CORE-010 uses that name and Go has no method overloading) and an unexported `node() node`. The interface MUST NOT contain methods with type parameters (compile error in Go 1.27.1).
- REQ-CORE-003: `ParseJSON` and `Decode` read tokens with `encoding/json/v2` + `jsontext` (baseline: decode into `any` preserving numbers as `json.Number`; feature 09 optimizes to single-pass). Numbers are never converted to `float64` before the target numeric kind is known.
- REQ-CORE-004: Implement `Option[T]` (absence of an object key): `Some(v) Option[T]`, `None[T]() Option[T]`, methods `IsSome() bool`, `Value() (T, bool)`, `OrElse(T) T`, `MarshalJSON` omitting/`null` semantics documented. Implement `Nullable[T]` (explicit JSON `null`; named `Nullable`, not `Null`, because `z.Null()` is the schema constructor and a type and a function cannot share the name in one package): `NullableOf(v) Nullable[T]`, `NullableNullable[T]() Nullable[T]`, `IsNull() bool`, `Value() (T, bool)`. Provide `json` (un)marshal support for both via `encoding/json/v2` interfaces.
- REQ-CORE-005: Define `IssueCode` (string type) with exactly 11 constants: `InvalidType`, `TooBig`, `TooSmall`, `InvalidFormat`, `NotMultipleOf`, `UnrecognizedKeys`, `InvalidUnion`, `InvalidKey`, `InvalidElement`, `InvalidValue`, `Custom` (wire values `invalid_type`, `too_big`, `too_small`, `invalid_format`, `not_multiple_of`, `unrecognized_keys`, `invalid_union`, `invalid_key`, `invalid_element`, `invalid_value`, `custom`).
- REQ-CORE-006: Define `PathSegment` (key `string` or index `int`, with `IsIndex()`), `Issue` (Code, Path, Message, Input, plus the code-specific fields of Zod v4, verified against `core/errors.ts`): `invalid_type`: `Expected` only (Zod v4 has NO `received` field; the received kind appears in the message text); `too_big`/`too_small`: `Origin` (`number`, `int`, `bigint`, `date`, `string`, `array`, `set`, `file`/`bytes`), `Maximum`/`Minimum`, `Inclusive`, `Exact`; `invalid_format`: `Format`, `Pattern` (`regex`), `Prefix` (`starts_with`), `Suffix` (`ends_with`), `Includes` (`includes`), `Algorithm` (`jwt`); `not_multiple_of`: `Divisor`; `unrecognized_keys`: `Keys`; `invalid_union`: `Errors [][]Issue`, `Discriminator`, `Options []any`, `Inclusive` (false when several options matched), `Matches []int`, `Note`; `invalid_key`: `Origin` (`map`/`record`), `Issues []Issue`; `invalid_element`: `Origin` (`map`/`set`), `Key`, `Issues []Issue`; `invalid_value`: `Values []any`; `custom`: `Params map[string]any`. Also `*ZodError` implementing `error` with `Issues() []Issue`. `Issue` values are created only when a rule fails.
- REQ-CORE-007: `ZodError` formatters (only the non-deprecated Zod v4 ones; there is NO `Format()`, Zod's deprecated `formatError` with the `_errors` tree): `Flatten()` (`FormErrors []string`, `FieldErrors map[string][]string`; **only the FIRST path segment is the key** of `FieldErrors`, as in Zod's `flattenError`, so `users.0.email` lands under `users`; issues with an empty path go to `FormErrors`), `Treeify()` (`{Errors, Properties, Items}`; string path segments under `Properties`, numeric ones under `Items`, each node with its own `Errors`), `Prettify()` (one `✖ message` per issue followed, when the path is non-empty, by `  → at path`; issues sorted by path length; path rendered like `users[0].email`).- REQ-CORE-008: Default messages for every issue code, sizing origin (`string`, `array`, `set`, `map`, `bytes`, `number`, `date`), and string format, without custom strings.
- REQ-CORE-009: i18n: `type Locale` with catalogs `LocaleEN`, `LocalePtBR`, `LocaleES` (plural/units: `caracteres`, `elementos`, `bytes`); `SetLocale(Locale)` (global default, stored atomically, safe under concurrent Parse); scoped locale per request via the context (REQ-CORE-013). Custom per-schema message override is feature 06 (REQ-PIPE-007).
- REQ-CORE-010: Metadata is a read-only value `Meta` struct `{ID, Title, Description string; Examples []any; Deprecated bool; Custom map[string]any}`, read back through the getter `GetMeta() Meta` (a method cannot be both `Meta()` and `Meta(...)` in Go). It is SET through a **builder function** on every concrete schema (copy-on-write, Zod's `.meta({...})`): `z.String().Meta(func(m *z.MetaBuilder) { m.Describe("Display name").Examples("Ana", "Bia") })`. `MetaBuilder` is a separate type (not `Meta`) because a Go type cannot have a field and a method with the same name (`Examples`). Its methods, all returning `*MetaBuilder` for chaining: `ID(string)`, `Title(string)`, `Describe(string)` (Zod's `.describe()`, equivalent to `meta({description})`; there is no separate `.Describe()` on schemas), `Examples(...any)`, `Deprecated(bool)`, `Custom(key string, v any)`, `Reset()`. The builder starts from a copy of the schema's CURRENT metadata, so calling `.Meta(...)` again MERGES with it by construction (use `m.Reset()` first to replace). Zod's `.meta()` getter form is `GetMeta()`. Registering the same non-empty `ID` on two different schemas panics at construction, like Zod (which throws on a duplicate `id` in every registry); the JSON Schema exporter relies on unique ids (feature 07).
- REQ-CORE-011: All per-call settings travel in the optional `context.Context` (REQ-CORE-002) under **well-known keys documented as part of the public contract**; there is NO separate options parameter and NO `ParseOption` type (a second variadic is impossible in Go, and `Option` is the presence type `Option[T]`). The **default** keys are plain strings namespaced by the module path, exported as constants so applications and other libraries can set them with plain `context.WithValue` WITHOUT importing the package's helpers: `const LocaleContextKey = "github.com/leandroluk/golang-zod#locale"` (value: a `Locale`, or a `string` tag / `Accept-Language` list) and `const AbortEarlyContextKey = "github.com/leandroluk/golang-zod#abortEarly"` (value: `bool`). The application can REPLACE these keys once at startup (REQ-CORE-016); after that the library reads AND writes the replaced key. In this spec, "the active locale key" means the replaced key if one was configured, otherwise `LocaleContextKey` (same for the abort-early key). Helpers: `func WithAbortEarly(ctx context.Context) context.Context` (nil `ctx` means `context.Background()`) and `WithLocale` (REQ-CORE-013); both just call `context.WithValue` with the active keys. The library reads these keys with `ctx.Value(<const>)` using the raw string (a typed key would not match a plain string set by others); the `go vet`/staticcheck "string context key" warning (SA1029) is suppressed with a comment at the library's own call sites, and documented for application authors. The context is also available to context-aware refinements (feature 06).
- REQ-CORE-012: `parseCtx` recycled via `sync.Pool` (fields: path stack, issues, locale, abortEarly, ctx, direction). Fully reset on `Put`. Parse functions return the literal `nil` `error` on success (never a typed-nil `*ZodError`).
- REQ-CORE-013: **Locale scoped in the context**, with a single public function: `func WithLocale(ctx context.Context, tag string) context.Context` (a nil `ctx` means `context.Background()`). It stores the resolved `Locale` under the active locale key (REQ-CORE-011). Equivalent without the helper (default key): `context.WithValue(ctx, z.LocaleContextKey, "pt-BR")`, or with the literal string `"github.com/leandroluk/golang-zod#locale"`; a `string` value is resolved at parse time exactly like `WithLocale`'s `tag`. `tag` is a language tag OR a raw `Accept-Language` header value (`pt-BR`, `pt-BR,pt;q=0.9,en;q=0.8`): the function picks the best registered locale (case-insensitive; exact match, then language-prefix match, honoring q-values) and returns a derived context carrying it. If `tag` is empty or matches nothing, it returns `ctx` unchanged (so the global default applies). Resolution order for every message of a parse: (1) `ctx.Value(<active locale key>)`, (2) the global `SetLocale` value, (3) `LocaleEN`. Reading is safe for concurrent use. Messages are resolved when the issue is created, so a `*ZodError` already holds the final text for its request language; schemas stay immutable and shared. Custom `.Error(msg)` overrides win over every locale.
- REQ-CORE-014: `func RegisterLocale(tag string, l Locale)` registers a custom locale so `WithLocale` can find it (meant for startup; reads are safe for concurrency). Built-ins `en`, `pt-BR`, `es` are pre-registered. There is NO `LocaleFromAcceptLanguage`, `LocaleFromTag`, `ContextWithLocale` or per-call locale option (one way only, KISS). Why a context and not a callback scope such as `z.WithLocale(tag, func() error {...})`: Go has no goroutine-local storage, so a parse running inside the callback could not find the scoped value; the context is the optional trailing parameter of every parse. **Transport-agnostic:** `WithLocale` works on ANY `context.Context` (HTTP, gRPC, CLI, queue workers, tests); the package imports neither `net/http` nor any framework, and `tag` is just a string (a plain tag like `pt-BR`, or an `Accept-Language` list when the caller happens to have one). Documented examples (not part of the package):
  ```go
  // worker / CLI / test: no HTTP involved
  ctx := z.WithLocale(context.Background(), "es")
  out, err := OrderSchema.ParseJSON(payload, ctx)

  // gRPC: language from metadata
  ctx = z.WithLocale(ctx, metadata.ValueFromIncomingContext(ctx, "accept-language")[0])
  ```
  Fiber example:
  ```go
  app.Use(func(c *fiber.Ctx) error {
      c.SetUserContext(z.WithLocale(c.UserContext(), c.Get("Accept-Language")))
      return c.Next()
  })
  // handler:
  user, err := UserSchema.ParseJSON(c.Body(), c.UserContext()) // messages in the request language
  ```
- REQ-CORE-016: **Global configuration** `var Config configNamespace` (like `z.ISO` / `z.Coerce`), with `func (configNamespace) UseContext(o ContextOptions)` and `type ContextOptions struct { LocaleKey any; AbortEarlyKey any }` (the options struct leaves room for future context-scoped settings). **`UseContext` REPLACES the default context keys** so the library shares a key the application already uses: `LocaleKey: "lang"` changes the locale key from `"github.com/leandroluk/golang-zod#locale"` to `"lang"`. After the call, the library reads `ctx.Value("lang")` for the locale and `z.WithLocale` writes to `"lang"`; the old default key is no longer consulted. A nil/zero field keeps that key's default. Values: for the locale key a `Locale` or a `string` (tag or `Accept-Language` list, resolved exactly like `WithLocale`'s argument); for the abort-early key a `bool`. Keys are `any` (not only `string`) because idiomatic Go context keys are unexported typed values; plain strings are supported too. Configuration is meant for startup (`main`), is stored atomically, and each call replaces the previous configuration entirely (call `UseContext(ContextOptions{})` to restore the defaults). Resolution order is the one in REQ-CORE-013. Example:
  ```go
  func main() {
      // before: "github.com/leandroluk/golang-zod#locale"   after: "lang"
      z.Config.UseContext(z.ContextOptions{LocaleKey: "lang"})
  }
  // anywhere in the app: ctx.Value("lang") == "pt-BR" is what z reads
  ```
  Tests: after `UseContext{LocaleKey: "lang"}`, a context holding `"lang" -> "pt-BR"` yields Portuguese with no `WithLocale` call; `z.WithLocale` writes `"lang"` (the old default key is ignored); a typed key holding a `Locale`; `UseContext(ContextOptions{})` restores the default key; `AbortEarlyKey` likewise. There is NO registry keyed by a framework context object and NO `z.Context(c)` / `Done()` lifecycle (see ADR-015).
- REQ-CORE-017: **The library respects the `context.Context` it receives** (this is what "context scope" means: the scope is the optional `ctx` value passed to the parse call; there is no implicit/ambient scope because Go has no goroutine-local storage; a parse called without a context uses `context.Background()`). Beyond the locale (REQ-CORE-013/016): (a) a parse checks `ctx.Err()` on entry and then at element boundaries of objects, arrays, sets, maps and records (a non-blocking check of `ctx.Done()`; skipped entirely when `ctx.Done()` is nil, e.g. `context.Background()`, so there is no cost without cancellation); (b) on cancellation or deadline the parse stops and returns the zero value and an error for which `errors.Is(err, context.Canceled)` / `errors.Is(err, context.DeadlineExceeded)` is true (it is NOT a `*ZodError`, because no validation issue occurred); (c) context-aware refinements (`.RefineCtx`, feature 06) receive the same `ctx`; (d) `ParseJSON`/`Decode` also stop reading the input stream on cancellation. Tests: an already-cancelled context returns `context.Canceled` without validating; a deadline expiring during a large array parse returns `context.DeadlineExceeded`; `context.Background()` allocates nothing extra (`-benchmem`).
- REQ-CORE-015: Tests (no HTTP packages imported by the test file for the first two): `WithLocale(context.Background(), "pt-BR")` yields Portuguese messages; a derived child context (`context.WithValue`, `context.WithCancel`) keeps the locale; the locale is isolated between concurrent parses of the same schema with different contexts (`-race`); resolution order of REQ-CORE-013; `WithLocale` with `pt-BR,pt;q=0.9,en;q=0.8`, `es-MX` (prefix match), an unknown language, and an empty header (context returned unchanged).

## Usage Examples (reference for the implementer; tests should mirror them)
The context is an **optional trailing parameter**: `schema.ParseJSON(body)` needs nothing; pass a context only where the language or cancellation matters: `schema.ParseJSON(body, ctx)`. There is no implicit/ambient scope (Go has no goroutine-local storage), so the middleware puts the language into the request context and the handler passes that context to the parse when it wants it applied.

```go
// 1. startup (main): optional. Tell z which context key already holds the language,
//    and the default language when nothing is found.
func main() {
    z.SetLocale(z.LocaleEN)
    // optional: share a key your app already uses. This REPLACES the default key
    // "github.com/leandroluk/golang-zod#locale" by "lang" (z reads and writes "lang").
    z.Config.UseContext(z.ContextOptions{LocaleKey: "lang"})
    // ...
}

// 2. Fiber middleware: runs once per request, no cleanup needed.
app.Use(func(c *fiber.Ctx) error {
    // Accept-Language may be empty or unknown: WithLocale then returns the ctx unchanged.
    c.SetUserContext(z.WithLocale(c.UserContext(), c.Get("Accept-Language")))
    return c.Next()
})

// 3. Handler: the context is optional. Without it the parse uses the global locale and never cancels:
//        user, err := UserSchema.ParseJSON(c.Body())
//    With it, messages come out in the request language and the parse stops if the
//    client disconnects or a deadline expires.
func createUser(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
    defer cancel()

    user, err := UserSchema.ParseJSON(c.Body(), ctx)
    if err != nil {
        var zerr *z.ZodError
        switch {
        case errors.As(err, &zerr):
            return c.Status(fiber.StatusUnprocessableEntity).JSON(zerr.Flatten()) // localized messages
        case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
            return c.SendStatus(fiber.StatusRequestTimeout)
        default:
            return err
        }
    }
    _ = user // typed User when UserSchema is a z.Struct
    return c.SendStatus(fiber.StatusCreated)
}

// 4. No HTTP at all: worker, CLI or test.
ctx := z.WithLocale(context.Background(), "pt-BR")
_, err := OrderSchema.ParseJSON(payload, ctx)

// 5. The application already stores the language in the context under its own key.
//    Step 1 REPLACED the default key by "lang" (before: "github.com/leandroluk/golang-zod#locale"),
//    so z reads (and z.WithLocale writes) "lang": no z.WithLocale call is needed.
ctx = context.WithValue(ctx, "lang", "es")
_, err = OrderSchema.ParseJSON(payload, ctx) // Spanish messages

// 5b. Without step 1 (default key), any code (another library, a middleware) can set the
//     documented key directly, with no import of z helpers.
ctx = context.WithValue(ctx, "github.com/leandroluk/golang-zod#locale", "pt-BR")

// 6. Stop at the first issue (setting travels in the context too).
_, err = OrderSchema.ParseJSON(payload, z.WithAbortEarly(ctx))

// 7. No context at all: global locale, no cancellation.
_, err = OrderSchema.ParseJSON(payload)
```

Expected behavior to assert in tests: example 3 returns `422` with Portuguese messages for `Accept-Language: pt-BR`, English for an unknown language, and `408` when the context deadline expires mid-parse; example 5 yields Spanish with no `z.WithLocale` call.

## Affected Components
- `node.go` — engine interface, `parseCtx`, pool
- `schema.go` — `Schema[T]`, entry points, context keys (`LocaleContextKey`, `AbortEarlyContextKey`), `WithAbortEarly`
- `option.go` — `Option[T]`, `Nullable[T]`
- `meta.go`, `error.go`, `i18n.go`, `locale_en.go`, `locale_ptbr.go`, `locale_es.go`

## Out of Scope
- `SafeParse`/`SafeParseAsync`/`Spa` (the `(T, error)` return already covers them) and any `*Async` parse variants (goroutines and `context.Context` replace them).
- JS-only types (`undefined`, `void`, `nan`, `symbol`, `promise`, `function`, `instanceof`).
- A separate `.describe()` method (use `.Meta(func(m *z.MetaBuilder){ m.Describe("...") })`).
- Zod's deprecated `formatError` / `Format()`.

## Open Questions
- None.
