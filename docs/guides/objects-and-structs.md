# Objects & Structs

`golang-zod` offers two distinct constructors for composite data types:
1. `z.Object`: Dynamic object validation returning `z.Dyn` (`map[string]any`).
2. `z.Struct[T]`: Concrete Go struct validation with zero-reflect runtime performance.

## Dynamic Objects (`z.Object`)

`z.Object` builds schemas using string keys:

```go
schema := z.Object(func(s *z.ObjectBuilder) {
	s.Field("username", z.String().Min(3))
	s.Field("score", z.Int().Default(0))
})
```

### Unknown Key Modes

- **Strip (Default)**: Unrecognized keys are omitted from output map.
- **Strict**: Unrecognized keys cause a validation error (`UnrecognizedKeys`).
- **Loose**: Unrecognized keys are preserved in the output map.
- **Catchall**: Unrecognized keys are validated against a catchall schema.

```go
strictSchema := schema.Strict()
looseSchema  := schema.Loose()
catchallSch  := schema.Catchall(z.String())
```

### Schema Manipulation

```go
// Extend with additional fields
extended := schema.Extend(func(s *z.ObjectBuilder) {
	s.Field("role", z.String())
})

// Pick / Omit fields
picked := schema.Pick("username")
omitted := schema.Omit("score")

// Partial & Required
allOptional := schema.Partial()
allRequired := allOptional.Required()

// Keyof
keyEnum := schema.Keyof() // Enum("username", "score")
```

---

## Concrete Go Structs (`z.Struct[T]`)

`z.Struct[T]` compiles a schema directly bound to a Go struct:

```go
type UserAccount struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname,omitempty"`
	IsPremium bool   `json:"is_premium"`
}

var userAccountSchema = z.Struct[UserAccount](func(s *z.StructBuilder, u *UserAccount) {
	s.Field(&u.ID, z.Int())
	s.Field(&u.Email, z.Email())
	s.Field(&u.Nickname, z.String().Default("guest"))
	s.Field(&u.IsPremium, z.Boolean())
})
```

### Zero-Reflect Runtime Guarantee

- Reflection is used **only once** at application bootstrap when `z.Struct[T]` is initialized to record struct field offsets.
- Runtime parse and validation paths allocate and populate `UserAccount` using pointer arithmetic (`unsafe.Add` confined strictly to `struct_unsafe.go`).
- Reflection is **never** invoked during `Parse`, `ParseJSON`, or `Decode`.

### In-Place Destination Reuse (`ParseTo`)

To eliminate allocations during repetitive JSON payload processing (e.g., in a high-throughput HTTP handler or worker pool):

```go
var user UserAccount
err := userAccountSchema.ParseTo(rawJSON, &user)
```

---

## Records (`z.Record`)

For dictionary mappings where keys and values share uniform types:

```go
// map[string]int
scores := z.Record(z.String(), z.Int())

// Exhaustive enum keys
colorEnum := z.Enum("red", "green", "blue")
colorPalette := z.Record(colorEnum, z.String()) // Requires red, green, and blue keys

// Partial enum record
partialPalette := z.PartialRecord(colorEnum, z.String())
```

## Field Metadata

`.Meta(...)` can be chained on any schema you pass to `Field`, including after `Optional()`, `Nullable()`, `Default()`, `DefaultFunc()`, `Prefault()`, `Catch()` and `Readonly()`:

```go
s.Field(&t.ID, z.UUID().DefaultFunc(newID).Meta(func(m *z.MetaBuilder) {
	m.Description("Entity identifier")
	m.Examples(newID())
}))
```

The metadata is picked up by the JSON Schema and GraphQL exporters.

`Examples` is generic, so a typed slice can be spread directly, e.g. an enum's values:

```go
m.Examples(IdentityAccountStatusEnumValues...)
```

To mix value types in one call, set the type explicitly: `m.Examples[any](1, "x")`.

## Declaring Fields

The schema is the mandatory second argument of `Field` in both `z.Object` and `z.Struct`; there are no chained type helpers such as `s.Field(&u.Name).String()`. Pass any schema, including one defined elsewhere:

```go
s.Field("name", z.String().Min(2))
s.Field(&u.Role, RoleEnumSchema)
```

## Embedded Structs

Like `encoding/json`, an embedded struct without a `json` tag is flattened into its parent. Register it with its own schema as the second argument; that schema runs over the same input and its fields (defaults included) take part in the parent:

```go
type Account struct {
	core.IndexableEntity   // `json:"id"`
	core.CreatableEntity   // `json:"created_at"`
	Email string `json:"email"`
}

var AccountSchema = z.Struct(func(s *z.StructBuilder, t *Account) {
	s.Field(&t.IndexableEntity, core.IndexableEntitySchema)
	s.Field(&t.CreatableEntity, core.CreatableEntitySchema)
	s.Field(&t.Email, z.Email())
})
```

`s.Embed(schema)` is a shorthand that needs no field pointer: the embedded field is the parent's anonymous field whose type matches the schema's type.

```go
var AccountSchema = z.Struct(func(s *z.StructBuilder, t *Account) {
	s.Embed(core.IndexableEntitySchema)
	s.Embed(core.CreatableEntitySchema)
	s.Field(&t.Email, z.Email())
})
```

- The embedded type must be exported; omitting the schema panics at construction.
- The embedded fields are merged into the parent's JSON Schema like Zod's `.extend(other.shape)`: all properties (with their metadata) are listed, and `required` holds the fields that are neither pointers, `Optional` nor defaulted on input (a `Nullable` key is still required, and defaulted keys count as required in the output schema).
- `examples` are merged too: the parent's example objects are laid over the embedded structs' example objects (the i-th with the i-th, the last repeating if one has fewer). A struct with no object example of its own gets one assembled from the first example of each of its fields. This only happens for structs that embed others; plain structs export their own `Meta` examples unchanged.
- Flattened keys appear in `FieldNames`, `FieldSchema`, `Keyof`, `Strict()` and the JSON Schema/GraphQL exporters.
- `Pick`/`Omit` address the parent's own fields only.
- Do not make the embedded schema `Strict()`: it sees the whole parent input.
