# Handoff — implementation by Gemini

Specs are complete; no Go code exists yet (only `go.mod`, Go 1.27.1, stdlib only).

## Read first (in order)
1. `INSIGHT.md` §0 (compiler limits verified on go1.27.1) and §2–§3.
2. `.specs/codebase/ARCHITECTURE.md`, `CONVENTIONS.md`, `CONCERNS.md`, `STACK.md`.
3. `.specs/project/ROADMAP.md` for milestone order; then one `.specs/features/<n>/spec.md` at a time.

## Rules that are easy to get wrong
- Methods in an interface cannot have type parameters; `Base[T].Optional() Base[*T]` is an instantiation cycle. Modifiers return wrapper types; each concrete schema declares the modifier set explicitly.
- `.Optional()` = key may be absent; `.Nullable()` = key required, accepts null; `.Nullish()` = both.
- Unknown-key default is strip. No `z.Optional(s)`, no `SafeParse`, no async variants, no `zod:"..."` tags, no `z.Shape{...}` map literal, no `z.Struct`/`z.ObjectBy`.
- Two builder constructors: `z.Object(func(s *z.Shape))` with `s.Field("key", schema)` (output `map[string]any`) and `z.Struct[T](func(s *z.StructShape, t *T))` with `s.Field(&t.Foo, schema)` (output `T`). Struct keys come from the mandatory `json:"name"` tag. There is no `ParseInto`.
- `reflect` only in `struct.go`; `unsafe` only in `struct_unsafe.go`.
- Done = `go vet ./...` + `go test -race ./...` green, tests named `TestREQ_<AREA>_<NNN>_*`, STATE.md updated in the same commit.

## Next step
Implement Milestone 1 (`01-core-error-i18n`).
