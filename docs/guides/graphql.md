# GraphQL SDL Generator

`golang-zod` includes an exporter that compiles schemas into standard GraphQL Schema Definition Language (SDL) strings.

## Exporting GraphQL SDL (`z.ToGraphQLSDL`)

```go
roleEnum := z.Enum("ADMIN", "MEMBER", "GUEST").Meta(func(m *z.MetaBuilder) {
	m.ID("Role")
})

userSchema := z.Object(func(s *z.ObjectBuilder) {
	s.Field("id").UUID()
	s.Field("name").String()
	s.Field("email").Email()
	s.Field("role").Schema(roleEnum)
	s.Field("bio").String().Optional()
	s.Field("skills").Schema(z.Array(z.String()))
}).Meta(func(m *z.MetaBuilder) {
	m.ID("User")
})

sdl, err := z.ToGraphQLSDL(userSchema)
if err != nil {
	panic(err)
}

fmt.Println(sdl)
```

### Output:

```graphql
enum Role {
  ADMIN
  MEMBER
  GUEST
}

type User {
  id: String!
  name: String!
  email: String!
  role: Role!
  bio: String
  skills: [String!]!
}
```

---

## Key Rules & Invariants

### 1. Inverted Nullability

GraphQL fields are nullable by default. `golang-zod` mirrors Zod v4's inverted nullability:
- Required fields emit the non-null modifier `!`.
- `.Optional()`, `.Nullable()`, and `.Nullish()` fields emit standard nullable types without `!`.
- Array elements are non-null by default: `[String!]!`.

### 2. Naming, Unions and Unsupported Constructs

- Type names come from `Meta.ID`; otherwise the parent type name plus the field name in PascalCase (`home_town` becomes `UserHomeTown`). Distinct types that collide get a numeric suffix (`UserAddress2`).
- `z.Union`, `z.XOR` and `z.DiscriminatedUnion` of objects or structs emit `union Name = A | B`.
- Constructs GraphQL cannot represent (intersections, tuples, unions with non-object members) return a descriptive error.

### 3. Input Types (`GraphQLInput()`)

To generate GraphQL `input` definitions rather than `type`:

```go
inputSDL, err := z.ToGraphQLSDL(userSchema, z.GraphQLInput())
```

Emits:

```graphql
input User {
  id: String!
  name: String!
  ...
}
```

### 4. Custom Scalars

Primitives like `z.Date()`, `z.BigInt()`, and `z.Bytes()` automatically declare their respective GraphQL custom scalar definitions (`scalar DateTime`, `scalar BigInt`, `scalar Bytes`).
