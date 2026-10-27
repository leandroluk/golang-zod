# JSON Schema & OpenAPI

`golang-zod` exports any runtime schema to standards-compliant **JSON Schema** (Draft 2020-12, Draft-07, Draft-04) and **OpenAPI** (3.0 and 3.1) definitions.

## Exporting JSON Schema (`z.ToJSONSchema`)

```go
userSchema := z.Object(func(s *z.Shape) {
	s.Field("id", z.UUID())
	s.Field("name", z.String().Min(2))
	s.Field("email", z.Email())
	s.Field("bio", z.String().Optional())
}).Meta(func(m *z.MetaBuilder) {
	m.ID("User").
		Title("User Account").
		Describe("Public profile information")
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
- `z.WithUnrepresentable(z.UnrepresentableThrow | z.UnrepresentableAny)`: Choose whether schemas without JSON Schema counterparts (e.g. `z.Date()`, `z.BigInt()`) throw an error or emit `{}`.
- `z.WithCycles(z.CyclesRef | z.CyclesThrow)`: Configure cycle resolution using `$ref` pointers.
- `z.WithReused(z.ReusedRef | z.ReusedInline)`: Deduplicate repeated schemas into `$defs`.

---

## Reverse JSON Schema Importer (`z.FromJSONSchema`)

`golang-zod` can dynamically reconstruct validation schemas from external JSON Schema specifications at runtime:

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
