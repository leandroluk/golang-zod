# Roadmap: golang-zod (Pure Zod v4 Port for Go)

> Master design: [INSIGHT.md](../../INSIGHT.md). Each feature has `.specs/features/<slug>/spec.md` with `REQ-*` IDs. Implement milestones in order; each depends on the previous.

## Milestone 1: Foundation, Errors, i18n, Engine (`M1-foundation`)
- **Feature 01 (`01-core-error-i18n`)**: untyped execution engine (`node`), typed `Schema[T]`, `ParseContext` + `sync.Pool`, input entry points (`Parse`, `ParseJSON`, `Decode`), `Option[T]`/`Nullable[T]`, `Meta`, 11 issue codes, `ZodError`, the 3 formatters (`Flatten`, `Treeify`, `Prettify`), i18n catalogs (`en`, `pt-BR`, `es`).

## Milestone 2: Primitives, String Formats, Numbers, Coercion (`M2-primitives`)
- **Feature 02 (`02-primitives-formats-coercion`)**: `String`, `Boolean`, `Null`, `Any`, `Unknown`, `Never`, `Date`, `Bytes`, `Literal`; text checks/transforms; top-level formats and `z.ISO.*`; numeric constructors and checks; `z.Coerce.*`.

## Milestone 3: Enums & Collections (`M3-enums-collections`)
- **Feature 03 (`03-enums-collections`)**: `z.Enum(...)` (+ `Values/Extract/Exclude`), `Array` (constructor and `.Array()` modifier), `Tuple`, `Set`, `Map`.

## Milestone 4: Objects, Records, Combinators (`M4-objects-combinators`)
- **Feature 04 (`04-objects-records`)**: `z.Object(func(s *z.Shape))` (string keys, `map[string]any`), `z.Struct[T](func(s *z.StructShape, t *T))` (`&t.Foo` keys, typed `T`), `z.StrictObject`, `z.LooseObject` (struct keys use the mandatory `json` tag; no `ParseInto`), unknown-key modes, shape ops, `Record`.
- **Feature 05 (`05-unions-intersections-lazy`)**: `Union`, `DiscriminatedUnion`, `XOR`, `Intersection`, `Lazy`.

## Milestone 5: Modifiers, Refinements, Codecs, Pipelines (`M5-modifiers-pipelines`)
- **Feature 06 (`06-modifiers-refinements-pipelines`)**: `.Optional/.Nullable/.Nullish/.Default/.Prefault/.Catch/.Readonly`, `.Check/.Refine/.SuperRefine`, `.Transform/.Preprocess/.Pipe`, `z.Codec`, `z.StringBool`, error customization.

## Milestone 6: Exporters (`M6-exporters`)
- **Feature 07 (`07-json-schema-openapi`)**: `ToJSONSchema`, `z.FromJSONSchema`.
- **Feature 08 (`08-graphql-builder`)**: `ToGraphQLSDL`.

## Milestone 7: Performance (`M7-optimizations`)
- **Feature 09 (`09-optimizations`)**: streaming `jsontext` parsing, inline path stack, allocation budgets, benchmark harness.
