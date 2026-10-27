# Architecture: golang-zod

## 1. Layout (single package `z`)
```
golang-zod/
├── INSIGHT.md          # design rationale + verified Go 1.27 limits
├── go.mod
├── z.go                # package doc + top-level constructors facade
├── node.go             # internal untyped engine: node interface, parseCtx, pool
├── schema.go           # Schema[T] interface, typed wrapper, input entry points
├── option.go           # Option[T], Nullable[T]
├── meta.go             # Meta
├── error.go            # IssueCode, Issue, PathSegment, ZodError, formatters
├── i18n.go, locale_en.go, locale_ptbr.go, locale_es.go
├── string.go, formats.go, iso.go, number.go, primitives.go, coerce.go
├── enum.go, array.go, tuple.go, set.go, map.go
├── object.go           # Object, Shape, unknown-key modes, shape ops
├── struct.go           # Struct[T], StructShape, construction (reflect, bootstrap only)
├── struct_unsafe.go    # StructSchema[T] field setters (the only file importing unsafe)
├── record.go, union.go, intersection.go, lazy.go
├── modifiers.go        # Optional/Nullable/Nullish/Default/Prefault/Catch/Readonly wrappers
├── refine.go, pipe.go, codec.go
├── jsonschema.go, jsonschema_import.go, graphql.go
└── *_test.go           # package z_test
```

## 2. Two layers
1. **Untyped engine** (unexported): every schema implements `node`:
   ```go
   type node interface {
       parse(ctx *parseCtx, in any) (out any, ok bool) // ok=false => issues already recorded in ctx
       kind() nodeKind // for exporters / introspection
   }
   ```
   Input `any` is decoded JSON-like data: `nil`, `bool`, `string`, `json.Number` / `float64` / any Go integer, `[]any`, `map[string]any`, `time.Time`, `[]byte`, `*big.Int`.
2. **Typed layer**: `Schema[T]` is a sealed generic interface (no generic methods in it):
   ```go
   type Schema[T any] interface {
       Parse(in any, ctx ...context.Context) (T, error)    // context is optional (first value used)
       ParseJSON(data []byte, ctx ...context.Context) (T, error)
       Decode(r io.Reader, ctx ...context.Context) (T, error)
       GetMeta() Meta
       node() node // unexported => sealed
   }
   ```
   Concrete types (`*StringSchema`, `*NumberSchema[T]`, ...) hold a `node` and add typed fluent methods.

## 3. Modifier wrappers
`OptionalSchema[T]`, `NullableSchema[T]`, `NullishSchema[T]`, `DefaultSchema[T]`, `ArraySchema[T]` are **distinct struct types** (they do not expose further `.Optional()`). Every concrete schema type declares the modifier set explicitly (small delegating methods) because Go embedding cannot see the outer type. Do NOT use a back-pointer trick.

## 4. Immutability
Chain methods copy-on-write: `func (s *StringSchema) Min(n int) *StringSchema { c := *s; c.checks = append(slices.Clone(s.checks), ...); return &c }`. Schemas are never mutated after construction; safe to share across goroutines.

## 5. Objects
- Two constructors, both with a builder run once at construction: `z.Object(func(s *z.Shape)) *ObjectSchema` (string keys, output `z.Dyn = map[string]any`, no reflect/unsafe) and `z.Struct[T](func(s *z.StructShape, t *T)) *StructSchema[T]` (pointer keys `&t.Foo`, output `T`).
- `StructShape.Field[F any](p *F, sch Schema[F])` is compile-time checked. Offset + type are matched with `reflect` once at construction (`struct.go`); the key is the mandatory `json` tag; at parse time fields are written by `unsafe.Add` setters (`struct_unsafe.go`), no `reflect`.
- There is no `ParseInto`; typed output comes from `z.Struct`. `json:"name"` is mandatory on every registered field, with no fallback to the field name.
- Different types per constructor, so `*StructSchema[T]` simply lacks `.Loose/.Catchall/.Extend/.Partial/.Required`.

## 6. Error & i18n
`*ZodError` implements `error`; issues carry code-specific fields; messages resolved lazily via the active locale.
