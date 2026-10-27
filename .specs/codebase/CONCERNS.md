# Architecture Concerns: golang-zod

## 1. Reflection and unsafe
- **Risk**: `reflect` cost on the hot path.
- **Mitigation**: `reflect` runs only once, at construction of `z.Struct`. `Parse`/`ParseJSON`/`Decode` never reflect. There is no `ParseInto`.
- **`z.Struct[T]`** writes fields with `unsafe.Add` at parse time (ADR-010/012). Risk: `unsafe` misuse. Mitigation: confined to `struct_unsafe.go`, offset+type verified at construction, only top-level exported fields with a `json` tag, `go vet` and `-race` tests, no embedded/nested paths.
- **Restrictions are compile-time**: `z.Object` and `z.Struct` are different types, so struct schemas simply do not have `.Loose()`, `.Catchall()`, `.Extend()`, `.Partial()`, `.Required()`. Remaining panics are construction-time configuration errors (bad pointer, missing `json` tag, duplicate name).

## 2. Allocation on the happy path
- **Risk**: `map[string]any` decoding, per-field `Issue` creation, path slices.
- **Mitigation**: pooled `parseCtx`; inline path stack (array, spill to slice only on issue); create `Issue` only on failure; streaming `jsontext` for `ParseJSON`/`Decode` (feature 09). Allocation numbers are measured goals, not guarantees.

## 3. Modifier typing in Go
- **Risk**: `Base[T].Optional() Base[*T]` is an instantiation cycle; interface methods cannot have type parameters.
- **Mitigation**: wrapper types; generic methods only on concrete types (`Transform[U]`); heterogeneous combinators (`Union`, `Intersection`, `Tuple`) output `any`.

## 4. Optional vs Nullable vs zero value
- **Risk**: Go cannot distinguish absent / null / zero.
- **Mitigation**: `Option[T]` (absence) and `Nullable[T]` (null) are explicit types; `omitempty` and pointers never imply optionality.

## 5. Numeric precision
- **Risk**: `float64` loses integers above 2^53.
- **Mitigation**: JSON numbers kept as `json.Number` text until the target kind is known; `Int64`/`Uint64`/`BigInt` parse from text.

## 6. Recursive schemas
- **Risk**: infinite loops in validation and exporters.
- **Mitigation**: `Lazy` memoized with `sync.OnceValue`; exporters track lazy node pointer and `Meta.ID` and emit `$defs`/`$ref`.

## 7. Typed-nil error pitfall
- **Risk**: returning a nil `*ZodError` as `error` is non-nil.
- **Mitigation**: parse functions return the literal `nil` interface on success; internal helpers return `*ZodError` only before the conversion.
