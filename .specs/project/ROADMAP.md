# Roadmap: golang-zod (Pure Zod v4 Port for Go)

> See [INSIGHT.md](../../INSIGHT.md) for the complete mapping of Zod v4 top-level constructors, Go paradigms, struct-tag reflection caching, and concurrency design.

## Milestone 1: Foundation, Value Model, Metadata, Errors & i18n (`M1-foundation`)
- **Feature 01 (`01-core-error-i18n`)**:
  - Runtime `Value` abstraction (`Null`, `Bool`, `Int`, `Uint`, `Float`, `String`, `Bytes`, `Time`, `Array`, `Object`) with `ParseInto` support and standard `encoding/json` interop.
  - Metadata builder: `.Meta(z.Meta{ID, Title, Description, Deprecated, ...})`.
  - Zod v4 11-variant issue taxonomy (`Issue`, `IssueCode`, `ZodError`), path tracking, and all 4 Zod v4 formatters (`Format()`, `Flatten()`, `Treeify()`, `Prettify()`).
  - i18n engine: `Locale` (`en`, `pt-BR`, `es`, custom), default validation messages, global (`z.SetLocale`) and request-scoped (`WithLocale`) locale switching.
  - Base `Schema` AST with `sync.Pool` recycled `ParseContext`, supporting `Parse()`, `SafeParse()`, `ParseInto()`.

## Milestone 2: Primitives, Top-Level String Formats & Coercion (`M2-primitives`)
- **Feature 02 (`02-primitives-formats-coercion`)**:
  - `z.String()` with generic text checks/transforms (`Min`, `Max`, `Length`, `NonEmpty`, `Regex`, `Includes`, `StartsWith`, `EndsWith`, `Lowercase`, `Uppercase`, `Trim`, `Normalize`, `ToLowerCase`, `ToUpperCase`, `Slugify`).
  - Top-level Zod v4 string format constructors:
    - `z.Email()`, `z.URL()`, `z.HTTPURL()`, `z.Hostname()`
    - `z.UUID()`, `z.UUIDv4()`, `z.UUIDv7()`, `z.GUID()`
    - `z.NanoID()`, `z.CUID2()`, `z.ULID()`, `z.XID()`, `z.KSUID()`
    - `z.IPv4()`, `z.IPv6()`, `z.MAC()`, `z.CIDRv4()`, `z.CIDRv6()`
    - `z.Base64()`, `z.Base64URL()`, `z.Hex()`, `z.JWT()`, `z.Emoji()`, `z.E164()`, `z.CreditCard()`, `z.IBAN()`, `z.CurrencyCode()`
    - `z.ISO.Date()`, `z.ISO.Time()`, `z.ISO.DateTime()`, `z.ISO.Duration()`
  - Top-level Zod v4 numeric constructors:
    - `z.Float64()`, `z.Float32()`, `z.Number()`
    - `z.Int()`, `z.Int8()`, `z.Int16()`, `z.Int32()`, `z.Int64()`
    - `z.Uint()`, `z.Uint8()`, `z.Uint16()`, `z.Uint32()`, `z.Uint64()`, `z.BigInt()`
    - Checks: `GT`, `GTE`/`Min`, `LT`, `LTE`/`Max`, `Positive`, `NonNegative`, `Negative`, `NonPositive`, `MultipleOf`.
  - `z.Boolean()`, `z.Null()`, `z.Any()`, `z.Unknown()`, `z.Never()`, `z.Date()`, `z.Bytes()`.
  - `z.Literal(...)`.
  - `z.Coerce.*` (`String`, `Number`, `Int`, `Boolean`, `BigInt`, `Date`).

## Milestone 3: Options (Enums) & Collections (`M3-options-collections`)
- **Feature 03 (`03-options-collections`)**:
  - `z.Options(...)` and `z.OptionsSlice(...)` with `.Options()`, `.Extract(...)`, `.Exclude(...)`.
  - `z.Array(schema)` (`Min`, `Max`, `Length`, `NonEmpty`, `Unique`, `Unwrap`).
  - `z.Tuple(...)` with `.Rest(schema)` and `.Partial()`.
  - `z.Set(schema)` and `z.Map(key_schema, value_schema)`.

## Milestone 4: Objects, Struct-Tag Inspection & Structural Combinators (`M4-objects-combinators`)
- **Feature 04 (`04-objects-records`)**:
  - `z.Object(z.Shape{...})`, `z.StrictObject(...)`, `z.LooseObject(...)`.
  - Type-First `z.Object[T]()` with bootstrap-only cached struct inspection reading `json:"..."` and `zod:"..."` tags.
  - Unknown keys handling: `.Strict()`, `.Loose()`, `.Strip()`, `.Catchall(schema)`.
  - Object manipulation: `.Keyof()`, `.Extend(...)`, `.Pick(...)`, `.Omit(...)`, `.Partial()`, `.Required()`.
  - Records: `z.Record(key, val)`.
- **Feature 05 (`05-unions-intersections-lazy`)**:
  - `z.Union(...)`, `z.DiscriminatedUnion(discriminator, ...)`, `z.XOR(...)`.
  - `z.Intersection(left, right)`.
  - `z.Lazy(func() Schema)` with `sync.Once` memoization for recursive schemas.

## Milestone 5: Modifiers, Refinements, Codecs & Pipelines (`M5-modifiers-pipelines`)
- **Feature 06 (`06-modifiers-refinements-pipelines`)**:
  - Modifiers: `.Optional()`, `.Nullable()`, `.Nullish()`, `.Default(val)`, `.Prefault(val)`, `.Catch(val)`, `.Readonly()`.
  - Validation hooks: `.Check(...)`, `.Refine(...)`, `.SuperRefine(...)`.
  - Bidirectional Codecs & Pipelines: `z.Codec(...)`, `.Transform(...)`, `.Preprocess(...)`, `.Pipe(...)`.

## Milestone 6: JSON Schema & OpenAPI Exporters/Importers (`M6-exporters`)
- **Feature 07 (`07-json-schema-openapi`)**:
  - `ToJSONSchema(opts)` supporting `Draft202012`, `Draft07`, `OpenApi30`, `OpenApi31`.
  - `z.FromJSONSchema([]byte)` runtime importer.
