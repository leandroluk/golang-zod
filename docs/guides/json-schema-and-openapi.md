# JSON Schema & OpenAPI

`golang-zod` exports any runtime schema to standards-compliant **JSON Schema** (Draft 2020-12, Draft-07, Draft-04) and **OpenAPI** (3.0 and 3.1) definitions.

## Exporting JSON Schema (`z.ToJSONSchema`)

```go
userSchema := z.Object(func(s *z.ObjectBuilder) {
	s.Field("id", z.UUID())
	s.Field("name", z.String().Min(2))
	s.Field("email", z.Email())
	s.Field("bio", z.String().Optional())
}).Meta(func(m *z.MetaBuilder) {
	m.ID("User").
		Title("User Account").
		Description("Public profile information")
})

// Export to Draft 2020-12 (default)
jsonSchemaBytes, err := z.ToJSONSchema(userSchema)
```

### Specifying Target Dialects

```go
// Draft-07
data07, _ := z.ToJSONSchema(userSchema, z.WithTarget(z.Draft07))

// OpenAPI 3.0 (emits components/schemas and nullable: true)
dataOpenAPI, _ := z.ToJSONSchema(userSchema, z.WithTarget(z.OpenAPI30))

// OpenAPI 3.1
dataOpenAPI31, _ := z.ToJSONSchema(userSchema, z.WithTarget(z.OpenAPI31))
```

### Exporter Options

- `z.WithIO(z.Input | z.Output)`: Select input vs output side for codecs and defaults.
- `z.WithUnrepresentable(z.UnrepresentableThrow | z.UnrepresentableAny)`: Choose whether schemas without JSON Schema counterparts (e.g. `z.Date()`, `z.BigInt()`, `z.Int64()`, `z.Uint64()`, `z.Map()`, `z.Set()`) throw an error or emit `{}`.
- `z.WithCycles(z.CyclesRef | z.CyclesThrow)`: Configure cycle resolution using `$ref` pointers.
- `z.WithReused(z.ReusedRef | z.ReusedInline)`: Deduplicate repeated schemas into `$defs` (default: inline).
- `z.WithOverride(func(s z.AnySchema, path string, js map[string]any))`: Adjust the generated JSON object of each schema before it is stored.
- `z.WithMetadata(func(s z.AnySchema) (z.Meta, bool))`: Supply extra metadata (title, description, examples, ...) merged over the schema's own `Meta`.

Integer schemas (`z.Int()`, `z.Int32()`, `z.Uint8()`, ...) export as `type: "integer"`; Record, Tuple, Struct, Pipe/Codec, Readonly, Default, Catch and Prefault each have their own mapping, and output is deterministic so it can be used in golden-file tests.

---

## Reverse JSON Schema Importer (`z.FromJSONSchema`)

`golang-zod` can dynamically reconstruct validation schemas from external JSON Schema specifications at runtime. It supports local `$ref` (including recursion), `anyOf`/`oneOf`/`allOf`, `const`/`enum`, numeric/string/array/object constraints, `prefixItems` tuples, `nullable`, `integer` and string `format`s. Keywords it cannot reproduce faithfully (remote `$ref`, `patternProperties`, `contains`, `unevaluated*`, ...) return an error instead of being silently ignored:

```go
rawSchema := []byte(`{
  "type": "object",
  "properties": {
    "title": { "type": "string" },
    "pages": { "type": "integer" }
  },
  "required": ["title"]
}`)

dynamicSchema, err := z.FromJSONSchema(rawSchema)
if err != nil {
	panic(err)
}

// Validate input using the reconstructed schema
res, err := dynamicSchema.Parse(map[string]any{
	"title": "The Go Programming Language",
	"pages": 380,
})
```
