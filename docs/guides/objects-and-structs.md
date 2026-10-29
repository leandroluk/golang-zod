# Objects & Structs

`golang-zod` offers two distinct constructors for composite data types:
1. `z.Object`: Dynamic object validation returning `z.Dyn` (`map[string]any`).
2. `z.Struct[T]`: Concrete Go struct validation with zero-reflect runtime performance.

## Dynamic Objects (`z.Object`)

`z.Object` builds schemas using string keys:

```go
schema := z.Object(func(s *z.ObjectBuilder) {
	s.Field("username").String().Min(3)
	s.Field("score").Int().Default(0)
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
	s.Field("role").String()
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
	s.Field(&u.ID).Int()
	s.Field(&u.Email).Email()
	s.Field(&u.Nickname).String().Default("guest")
	s.Field(&u.IsPremium).Boolean()
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
