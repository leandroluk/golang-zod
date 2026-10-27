# Spec: 05 — Unions, Intersections & Lazy Recursion

## Summary
Implements combinators per [INSIGHT.md](../../../INSIGHT.md):
1. `z.Union(opts ...AnySchema) *UnionSchema` (output `any`): first successful option wins; failure is `InvalidUnion` carrying each option's issues in `Errors`.
2. `z.DiscriminatedUnion(discriminator string, opts ...ObjectNode) *UnionSchema` (`ObjectNode` is the exported non-generic interface satisfied by every `*ObjectSchema[T]`, exposing its keys and per-key schemas): O(1) dispatch by the discriminator literal.
3. `z.XOR(opts ...AnySchema) *UnionSchema`: exactly one option must succeed (zero or many is `InvalidUnion`, with different fields, REQ-COMB-003).
4. `z.Intersection(left, right AnySchema) *IntersectionSchema` (output `any`): both succeed and the results are merged; values that cannot be merged are a runtime ERROR, not an issue (REQ-COMB-004).
5. `z.Lazy[T any](f func() Schema[T]) Schema[T]`: memoized with `sync.OnceValue` so recursive schemas (`var Category = z.Lazy(func() ...)`) neither loop nor overflow.

## Requirements
- REQ-COMB-001: `z.Union` with aggregated `InvalidUnion` issue (`Errors [][]Issue`).
- REQ-COMB-002: `z.DiscriminatedUnion`: builds a map `literal -> option` at construction; panics at construction if an option lacks the discriminator as a `Literal`/`Enum` key or two options collide. An unknown discriminator value yields one `invalid_union` issue with `Errors: []` (empty), `Note: "No matching discriminator"`, `Discriminator` (the key), `Options` (the accepted discriminator values) and `Path = [discriminator]` (as in Zod's `$ZodDiscriminatedUnion`).
- REQ-COMB-003: `z.XOR`: zero matches yields `invalid_union` with `Errors` holding each option's issues; more than one match yields `invalid_union` with `Errors: []`, `Inclusive: false` and `Matches` (the indices of the options that passed), message "Invalid input: more than one option matched"; exactly one match returns that option's value. A plain `z.Union` keeps `Inclusive: true`.
- REQ-COMB-004: `z.Intersection(a, b)`: both sides parse the same input and the results are merged (objects merge by key recursively; equal scalars merge; arrays merge element-wise when equal in length). When the two results cannot be merged (different scalars for the same key, different array lengths), Zod throws `Unmergable intersection` with the error path instead of reporting an issue. Here the parse returns the sentinel-wrapped error `ErrUnmergeableIntersection` (with the JSON path in its message), NOT a `*ZodError`. It is an `error` return, not a `panic` (ADR-021).
- REQ-COMB-005: `z.Lazy` with `sync.OnceValue` memoization; the lazy node pointer is stable per `Lazy(...)` call.
- REQ-COMB-006: `union`, `xor`, `discriminatedUnion`, `intersection`, `lazy` are first-class `node` kinds (not generic "custom") so exporters (features 07, 08) can track the lazy pointer and `Meta.ID` and resolve cycles into `$defs`/`$ref`.
- REQ-COMB-007: Test: a recursive `Category` (`{name, children: Array(Lazy(Category))}`) parses a 3-level tree and exports to JSON Schema and GraphQL without looping.
- REQ-COMB-008: Standard modifier set declared explicitly per type (see REQ-PRIM-009). Because options are heterogeneous, typed `Union` outputs `any`; callers needing a typed result use `.Transform`.

## Affected Components
- `union.go`, `intersection.go`, `lazy.go`; depends on 01–04.

## Out of Scope
- Typed N-ary union generics (`Union2[A,B]`), considered later.

## Open Questions
- None.
