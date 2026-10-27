# Spec: 04 — Objects (`z.Object`, `z.Struct`) & Records

## Summary
Implements Milestone 4 (objects) per [INSIGHT.md](../../../INSIGHT.md) §1.3, §2.2.1, §3. Two constructors, both taking a **builder function** (no map literals):

```go
// z.Object: string keys, output map[string]any (z.Dyn)
user := z.Object(func(s *z.Shape) {
    s.Field("name",  z.String().Min(2))
    s.Field("email", z.Email())
    s.Field("bio",   z.String().Max(280).Optional())
})

// z.Struct: pointers to fields of T, output T (key = mandatory json tag)
user := z.Struct(func(s *z.StructShape, t *User) {
    s.Field(&t.Name,  z.String().Min(2))
    s.Field(&t.Email, z.Email())
    s.Field(&t.Bio,   z.String().Max(280).Optional()) // field must be z.Option[string]
})
```

Signatures (verified on go1.27.1; inference and mismatch rejection tested):
- `type Dyn = map[string]any`
- `func Object(build func(s *Shape)) *ObjectSchema` — `*ObjectSchema` implements `Schema[Dyn]`.
- `func (s *Shape) Field[F any](name string, sch Schema[F])`
- `func Struct[T any](build func(s *StructShape, t *T)) *StructSchema[T]` — `*StructSchema[T]` implements `Schema[T]`; `T` must be a struct type.
- `func (s *StructShape) Field[F any](p *F, sch Schema[F])` — the schema's output type MUST equal the field type: an `.Optional()` schema on a plain `string` field **does not compile**.
- `z.StrictObject(build)` and `z.LooseObject(build)` have the same signature as `z.Object`.
- `build` runs ONCE at construction. There is no `z.Shape{...}` map literal, no `z.ObjectBy`.

Restrictions are compile-time because each constructor has its own type: `Shape` has only string keys, `StructShape` only pointer keys, and `*ObjectSchema` / `*StructSchema[T]` expose different method sets (below).

## Requirements
- REQ-OBJ-001: `z.Object`, `z.StrictObject`, `z.LooseObject` (builder with `*Shape`). `z.StrictObject(b)` ≡ `z.Object(b).Strict()`; `z.LooseObject(b)` ≡ `z.Object(b).Loose()`. Duplicate keys panic at construction.
- REQ-OBJ-002: `*ObjectSchema` methods: unknown-key modes — default **strip** (as in Zod), `.Strict()` (`UnrecognizedKeys`), `.Loose()`, `.Strip()`, `.Catchall(schema AnySchema)`; shape ops — `.Extend(build func(s *Shape))`, `.Pick(keys ...string)`, `.Omit(keys ...string)`, `.Partial(keys ...string)` (all keys when empty), `.Required(keys ...string)`, `.Keyof() *EnumSchema[string]`. All return `*ObjectSchema` (same type). `.Extend` inherits the current mode. Unknown key names passed to `Pick/Omit/Partial/Required` panic at construction.
- REQ-OBJ-003: Presence semantics for keys (INSIGHT §2.2.1), both constructors: `.Optional()` key may be absent (omitted from the `Dyn` output / `Option[T]` field; `null` is `InvalidType`); `.Nullable()` key required and accepts `null` (`nil` in `Dyn` / `Nullable[T]` field); `.Nullish()` both (`Option[Nullable[T]]`); any other schema: absent key is `InvalidType` with the key in `Path`.
- REQ-OBJ-004: `z.Struct[T]` construction (the only place where `reflect` runs): `build` is called with `t` pointing to a zero `T`; for each `Field(p, ...)` compute `off = reflect.ValueOf(p).Pointer() - reflect.ValueOf(t).Pointer()`, find the top-level field with that `Offset` AND identical type, and take the key from its **mandatory** `json:"name"` tag (name part; no fallback to the Go field name; `json:"-"` fields cannot be registered). Panic at construction when: `T` is not a struct, the pointer is not a top-level field of `T` (embedded or nested path), the field is unexported, has no `json` tag, or the name is duplicated. A nested struct is a field whose schema is another `z.Struct[Inner]`.
- REQ-OBJ-005: `*StructSchema[T]` runtime: allocate one `T`, validate each registered field, write with a setter built at construction (`unsafe.Add` on the base pointer by the stored offset, typed `*F`, no `reflect`). Unregistered fields keep their zero value. `reflect` is imported ONLY by `struct.go` (construction); `unsafe` ONLY by `struct_unsafe.go` (ADR-010/012). A test scans package sources to enforce both. `Parse`, `ParseJSON` and `Decode` never use `reflect`.
- REQ-OBJ-006: `*StructSchema[T]` methods: `.Strict()`, `.Strip()` (default strip), `.Pick(keys ...string)`, `.Omit(keys ...string)` (same `T`; omitted fields stay zero), `.Keyof() *EnumSchema[string]`. It has NO `.Loose()`, `.Catchall()`, `.Extend()`, `.Partial()` or `.Required()` (a struct has nowhere to store extra keys and cannot change its field set): these are absent, so misuse fails to compile.
- REQ-OBJ-007: **Removed** (there is no `ParseInto`; `z.Struct` replaces it). The ID is kept unused so IDs are never renumbered. Struct field mapping rules live in REQ-OBJ-003/004: `.Optional()` ↔ `Option[T]`, `.Nullable()` ↔ `Nullable[T]`, `.Nullish()` ↔ `Option[Nullable[T]]`, else plain `T` (compile-time checked); `omitempty` and pointers NEVER imply optionality.
- REQ-OBJ-008: Records (semantics verified against Zod's `$ZodRecord`): `z.Record[K ~string | ~int..., V any](key Schema[K], val Schema[V]) *RecordSchema[K,V]` (output `map[K]V`; numeric key schemas validate numeric strings, as in Zod). Default (strict) mode: with an **enum key schema** (`z.Enum`) the record is **exhaustive**: every enum value must be present, a failing key yields `invalid_key` (`Origin: "record"`), and keys outside the enum yield `unrecognized_keys`; with any **other key schema**, each key that fails the key schema yields `invalid_key` with `Path = [key]`. Value failures are re-pathed with the key. A value schema with `.Default()`/`.Optional()` makes that key optional on input. `z.PartialRecord(...)` skips the exhaustiveness check (enum keys may be missing). `z.LooseRecord(...)`: keys that do not match the key schema pass through untouched instead of erroring.
- REQ-OBJ-009: Standard modifier set declared explicitly on `*ObjectSchema`, `*StructSchema[T]`, and records (`.Optional()`, `.Nullable()`, `.Nullish()`, `.Default(v)`, `.Array()`, `.Meta(m)`; see REQ-PRIM-009). Exporters (features 07, 08) read both as objects whose keys are the registered keys, through the exported non-generic interface `ObjectNode` (keys, per-key schemas, unknown-key mode) that both types satisfy.
- REQ-OBJ-010: Tests: both constructors; each panic case of REQ-OBJ-001/004; compile-failure documentation (an `.Optional()` schema bound to a plain `string` field via `StructShape.Field` is rejected by the compiler: keep as a commented example); nested structs; `-race` parallel parses of a shared schema; the source-scan test of REQ-OBJ-005.

## Affected Components
- `object.go` (`Object`, `Shape`, modes, shape ops), `struct.go` (`Struct`, `StructShape`, construction; uses `reflect`), `struct_unsafe.go` (setters; uses `unsafe`), `record.go`
- ⚠️ These are the only files where `reflect` and `unsafe` appear in the core.

## Out of Scope
- `zod:"..."` tags (forbidden); `.merge()` (use `.Extend`); `z.Shape{...}` map literals; `z.ObjectBy`; a single `z.Object` serving both forms (rejected: Go cannot omit a builder parameter or restrict methods by `T`).

## Design notes
- A chained field form `s.Field(&t.Foo).String().Optional()` was rejected: nothing registers the final value of the chain, and Go cannot restrict `.String()` by the field's type. Passing the schema as an argument registers it and gives compile-time type checking. Revisit only with a user decision recorded in STATE.md.
- Verified on go1.27.1: `func(s *Shape)` cannot be passed where `func(s *Shape, t *T)` or `func(s *Shape, t ...*T)` is expected, and a union of the two function types cannot infer `T`; hence two constructors.

## Open Questions
- None.
