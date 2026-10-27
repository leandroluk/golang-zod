# Spec: 08 — GraphQL SDL Generator

## Summary
Implements the GraphQL exporter (parity with `rust-zod` feature 08). `z.ToGraphQLSDL(s AnySchema, opts ...GraphQLOption) (string, error)` generates:
- `type` (output) vs `input` (input objects, selected by `GraphQLInput()`), `enum` from `z.Enum`, `union` from `Union`/`DiscriminatedUnion` of objects, custom `scalar` declarations (`DateTime`, `BigInt`, `Bytes`, `JSON`), and PascalCase naming.
- **Inverted nullability**: GraphQL fields are nullable by default; here a field is `T!` when required and `T` when the schema key is `.Optional()`/`.Nullable()`/`.Nullish()`.

## Requirements
- REQ-GQL-001: Generate `type`/`input`/`enum`/`union`/`scalar` definitions with deterministic output order.
- REQ-GQL-002: Nullability inversion as above, including list element nullability (`[T!]!`).
- REQ-GQL-003: Type naming from `Meta.ID`; fallback PascalCase of parent type + field path (`UserAddress`); collisions get a numeric suffix.
- REQ-GQL-004: Recursive (`Lazy`) types resolve by node pointer without looping.
- REQ-GQL-005: Constructs with no GraphQL equivalent (intersection of non-objects, tuples) return a descriptive error, not a silent drop.

## Affected Components
- `graphql.go`; depends on node kinds from features 01–06.

## Out of Scope
- Resolvers, directives, federation, schema stitching.

## Open Questions
- None.
