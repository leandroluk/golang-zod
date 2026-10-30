# Type Conversion & Binding

`golang-zod` bridges untyped external data (JSON, form inputs, dynamic maps) and Go's strong static type system safely.

## Presence Modifiers

Go does not have a native `undefined` type. `golang-zod` defines precise presence types in `z.Option[T]` and `z.Nullable[T]`.

| Modifier      | Return Type             | Missing Key        | `null` Value          |
| :------------ | :---------------------- | :----------------- | :-------------------- |
| `.Optional()` | `z.Option[T]`           | `None[T]()`        | InvalidType Error     |
| `.Nullable()` | `z.Nullable[T]`         | Required           | `NullableNull[T]()`   |
| `.Nullish()`  | `z.Option[Nullable[T]]` | `None`             | `Some(NullableNull)`  |
| `.Default(v)` | `T`                     | Evaluates fallback | Fails unless Nullable |
| `.DefaultFunc(fn)` | `T`                 | Calls `fn` on every absent value | Fails unless Nullable |
| `.Prefault(v)` / `.PrefaultFunc(fn)` | `T`      | Replaces input, then validates it | Fails unless Nullable |
| `.Catch(v)` / `.CatchFunc(fn)` | `T`            | Fallback on any failure | Fallback |
| `.Readonly()` | `T`                     | Same as inner      | Same as inner         |

### Dynamic defaults

`Default` takes a fixed value; `DefaultFunc` takes a supplier that runs on every absent input, which is what you want for generated values (Zod's `.default(() => ...)`):

```go
id := z.UUID().DefaultFunc(func() string { return uuid.NewString() })

createdAt := z.Date().DefaultFunc(time.Now)
```

Supplier-based defaults are not exported as a `default` keyword in JSON Schema, since they have no stable value.

### Pointer fields

A struct field declared as a pointer, such as `DeletedAt *time.Time`, already allows `nil`: pass the plain schema and an absent or `null` key yields `nil`, anything else a pointer to the parsed value. No `Optional()` needed.

```go
s.Field(&e.DeletedAt, z.Date())
```

For a `*T` field the schema must produce `T`.

`z.Ptr(s)` / `.Ptr()` do the same explicitly. `Optional()` yields `z.Option[T]` and `Nullable()` yields `z.Nullable[T]`, so a field using them must be declared with that type; `z.Struct` panics at construction when a field type does not match what its schema produces.

### Unpacking `Option[T]` and `Nullable[T]`

```go
opt := z.Some("hello")
if val, ok := opt.Value(); ok {
	_ = val // "hello"
}

// Or with fallback
val := opt.OrElse("fallback")
```

## Combinators

### Union (First Match)

```go
// Matches string first, otherwise integer
union := z.Union(z.String(), z.Int())
```

### XOR (Mutually Exclusive)

```go
// Exactly ONE candidate must succeed
xor := z.XOR(z.Int().GTE(0), z.Int().LTE(10))
```

### Discriminated Union (O(1) Map Dispatch)

```go
circle := z.Object(func(s *z.ObjectBuilder) {
	s.Field("kind", z.Literal("circle"))
	s.Field("radius", z.Float64())
})

square := z.Object(func(s *z.ObjectBuilder) {
	s.Field("kind", z.Literal("square"))
	s.Field("side", z.Float64())
})

shapeUnion := z.DiscriminatedUnion("kind", circle, square)
```

### Intersection & Recursive Schemas (`z.Lazy`)

```go
// Intersection merges two object schemas
both := z.Intersection(partA, partB)

// Recursive tree structures
var treeSchema z.Schema[z.Dyn]
treeSchema = z.Lazy(func() z.Schema[z.Dyn] {
	return z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String())
		s.Field("children", z.Array(treeSchema).Optional())
	})
})
```
