# Spec: 03 — Enums & Collections

## Summary
Implements Milestone 3 per [INSIGHT.md](../../../INSIGHT.md) §1.4:
1. `z.Enum[T ~string](v ...T) *EnumSchema[T]` — native variadic (`z.Enum("a","b")`, `z.Enum(roles...)`, `z.Enum(Admin, User)`), with `.Values() []T`, `.Extract(v ...T)`, `.Exclude(v ...T)`.
2. `z.Array[T any](item Schema[T]) *ArraySchema[T]` (output `[]T`) with `.Min(n)`, `.Max(n)`, `.Length(n)`, `.NonEmpty()`, `.Element() Schema[T]`, `.Unwrap() Schema[T]` (same as `.Element()`, both exist in Zod). There is no `.Unique()` (not in Zod v4).
3. `z.Tuple(items ...AnySchema) *TupleSchema` (output `[]any`) with `.Rest(schema AnySchema)`.
4. `z.Set[T comparable](item Schema[T]) *SetSchema[T]` (output `map[T]struct{}`) with `.Min`, `.Max`, `.Size`, `.NonEmpty`.
5. `z.Map[K comparable, V any](key Schema[K], val Schema[V]) *MapSchema[K,V]` (output `map[K]V`) with `.Min`, `.Max`, `.Size`, `.NonEmpty`.

## Requirements
- REQ-COLL-001: `z.Enum` as described. Failure is `InvalidValue` with `Values` set. Mixed-type enums are unsupported (single `T`).
- REQ-COLL-002: `z.Array(item)` plus the fluent `.Array()` modifier on every schema (it returns `*ArraySchema[T]`, a wrapper that is NOT itself arrayable again via `.Array()`; use `z.Array(x.Array())` for nested arrays). Element issues use `InvalidElement` only where Zod does; otherwise the inner issue is re-pathed with the index.
- REQ-COLL-003: `z.Tuple(...)`: fixed-length heterogeneous validation; length mismatch is `TooBig`/`TooSmall` (`Origin: "array"`). `AnySchema` is the exported non-generic interface `interface{ node() node }` satisfied by every `Schema[T]`.
- REQ-COLL-004: `z.Set`: size checks with `Origin: "set"`; duplicates in input are collapsed (Zod semantics), not errors.
- REQ-COLL-005: `z.Map`: key schema failure is `InvalidKey`, value failure is the inner issue with the key in the path.
- REQ-COLL-006: All collection schemas carry the standard modifier set declared explicitly per type (see REQ-PRIM-009).

## Affected Components
- `enum.go`, `array.go`, `tuple.go`, `set.go`, `map.go`; depends on features 01, 02.

## Out of Scope
- `.Unwrap()` aliases beyond `.Element()`; infinite iterators; JS `Set`/`Map` object identity semantics.

## Open Questions
- None.
