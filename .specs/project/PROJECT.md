# Project: golang-zod

Traceability: on

## Vision
A complete, idiomatic Go port of **Zod v4** with maximum API compatibility with `colinhacks/zod`. Validation rules live in Go code (a fluent DSL), never in struct tags. Two builder constructors: `z.Object(func(s *z.Shape))` with `s.Field("name", schema)` (output `map[string]any`), and `z.Struct[T](func(s *z.StructShape, t *T))` with `s.Field(&t.Foo, schema)` (typed output, compile-time checked, no reflection at parse time). A mandatory `json:"name"` tag is the key of every struct field (KISS); There is no `ParseInto`. No `z.Shape{...}` map literal. Target toolchain: **Go 1.27.1** (generic methods available; see `INSIGHT.md` §0 for what the compiler accepts).

## Core Principles
1. **Zod v4 pure API**: all string formats and number kinds top-level (`z.Email()`, `z.UUID()`, `z.ISO.DateTime()`, `z.Int()`); no v3 legacy (`.merge()`, `z.string().email()`).
2. **Compatibility first**: names follow Zod (PascalCase). Unknown-key default is **strip** (as in Zod). `.Optional()` = key may be absent; `.Nullable()` = key required, accepts null; `.Nullish()` = both.
3. **No validation tags**: the only struct tag ever read is `json:"..."` (by `z.Struct`). `zod:"..."` is forbidden.
4. **Fluent modifiers only**: `Optional`, `Nullable`, `Nullish`, `Default`, `Array` are chained methods returning distinct wrapper types (never `Base[*T]`: instantiation cycle). There is no `z.Optional(s)` free function.
5. **Reflection only at `z.Struct` construction**; `Parse`/`ParseJSON`/`Decode` never reflect. `unsafe` only in `struct_unsafe.go`.
6. **Errors as values**: `(T, error)` with `*ZodError`; 11 issue codes; formatters `Flatten`, `Treeify`, `Prettify` (no deprecated `Format`); i18n (`en`, `pt-BR`, `es`).
7. **Concurrency-safe, allocation-conscious**: immutable schemas shared across goroutines; `sync.Pool` for parse contexts; streaming JSON input via `encoding/json/v2` + `jsontext`.
8. **Exporters**: JSON Schema (2020-12, Draft-07, Draft-04), OpenAPI 3.0/3.1, GraphQL SDL, and reverse `z.FromJSONSchema`.
9. **Zero third-party dependencies**: standard library only.
