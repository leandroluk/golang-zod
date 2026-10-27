# Spec: 07 — JSON Schema, OpenAPI & Reverse Importer

## Summary
Implements Milestone 6 (JSON Schema/OpenAPI) per [INSIGHT.md](../../../INSIGHT.md) §8. Exporters work on the non-generic `node` AST (kinds are first-class: string formats, number kinds/checks, literal, array/tuple/set/map/record, union/xor/discriminated/intersection, optional/nullable/default/readonly wrappers, lazy, transform/pipe/codec, meta).
1. `z.ToJSONSchema(s AnySchema, opts ...JSONSchemaOption) ([]byte, error)`; targets `Draft202012` (default), `Draft07`, `Draft04`, `OpenAPI30` (Zod's `openapi-3.0`), plus `OpenAPI31` (a documented extension, not a Zod target); options mirror Zod's `toJSONSchema`: `IO(Input|Output)` (**default Output**), `Unrepresentable(Throw|Any|func)` (**default Throw**), `Cycles(Ref|Throw)` (default `Ref`), `Reused(Inline|Ref)` (default `Inline`), `OverrideWith(func(...))`, `Metadata(...)`, `URI(func(id string) string)`.
2. `z.FromJSONSchema(data []byte) (AnySchema, error)`: dynamic reconstruction (Mode A schemas).

## Requirements
- REQ-EXP-001: `ToJSONSchema` for Draft 2020-12, Draft-07, Draft-04, OpenAPI 3.0, OpenAPI 3.1 (differences: `prefixItems` vs `items` array, `nullable` vs `type:[..,"null"]`, `$defs` vs `definitions`, `exclusiveMinimum` boolean vs number, `#/components/schemas` refs for OpenAPI).
- REQ-EXP-002: Constraint fidelity: emit `format`, `minLength/maxLength/pattern`, `minimum/maximum/exclusive*/multipleOf`, `oneOf/anyOf/allOf`, `prefixItems`, `additionalProperties`: a default (strip) object emits `additionalProperties: false` in `Output` mode and OMITS it in `Input` mode; `.Strict()`/`z.StrictObject` always emit `false`; `.Loose()`/`z.LooseObject` never emit `false`; a catchall emits its schema. Also `const/enum`, `default`, `readOnly`, `title/description/deprecated/examples` from `Meta`. Required keys come from non-Optional keys. `Nullable` emits `type: ["string","null"]` (or `anyOf` when the inner schema has other constraints); `Optional` emits the inner schema and only drops the key from `required`. How defaults, tuples, records, unions and catchall are emitted is not documented on Zod's JSON Schema page; follow the JSON Schema meaning of each and document the choice in the tests.
- REQ-EXP-003: Cycle detection and `$defs` sharing keyed by `Meta.ID` and by the lazy node pointer; recursive schemas emit `$ref` and terminate (`Cycles(Ref)`, the default); `Cycles(Throw)` returns an error instead. `Reused(Ref)` moves schemas used more than once into `$defs`; `Reused(Inline)` (default) inlines them. `URI(func)` maps ids to URIs for external `$ref`s (identity by default). Two schemas with the same `Meta.ID` are rejected at construction (feature 01).
- REQ-EXP-004: `IO(Input)` vs `IO(Output)` (default `Output`) pick the input/output side of codecs, pipes, defaults and coerced primitives. **Unrepresentable** schemas return an error by default (`Unrepresentable(Throw)`), exactly the set Zod lists: `bigint`, `int64`/`uint64`, `symbol`, `undefined`, `void`, `date`, `map`, `set`, `transform`, `nan` and custom refinements-only schemas (`custom`); `Unrepresentable(Any)` emits `{}` instead and drops the unrepresentable checks; `Unrepresentable(func)` lets the caller decide per schema (it receives the schema, the path and the message). In particular a `z.Date()` and a `.Transform` are NOT silently emitted as a string or `{}`.
- REQ-EXP-005: `z.FromJSONSchema` handles `$ref`, `anyOf/oneOf/allOf`, `const/enum`, constraints, tuples (`prefixItems`), `additionalProperties`, `nullable`, formats mapping to top-level format schemas.
- REQ-EXP-006: Round-trip test: for a corpus of 10 schemas, `FromJSONSchema(ToJSONSchema(s))` accepts and rejects the same fixtures.
- REQ-EXP-007: Output is deterministic (stable key order) so golden-file tests are possible.

## Affected Components
- `jsonschema.go`, `jsonschema_import.go`; depends on all node kinds from features 01–06.

## Out of Scope
- JSON Schema `$dynamicRef`, `if/then/else`, remote `$ref` resolution.

## Open Questions
- None.
