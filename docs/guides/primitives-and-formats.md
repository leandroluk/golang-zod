# Primitives & Formats

`golang-zod` adheres strictly to Zod v4 specifications, providing both general scalar schemas and specialized top-level format schemas.

## Core Primitives

```go
z.String()   // string
z.Boolean()  // bool
z.Int()      // int
z.Int32()    // int32
z.Int64()    // int64
z.Uint()     // uint
z.Uint64()   // uint64
z.Float64()  // float64
z.Date()     // time.Time
z.Bytes()    // []byte
z.BigInt()   // *big.Int
z.Any()      // any
```

## String Formats & Constraints

```go
z.String().
	Min(3).
	Max(100).
	Length(10).
	Trim().
	ToLowerCase().
	ToUpperCase().
	Regex(regexp.MustCompile(`^[A-Z0-9]+$`))
```

### Top-Level String Formats (Zod v4 Standard)

In Zod v4, specialized string formats are first-class constructors:

```go
z.Email()       // Valid email address
z.UUID()        // UUID v4 format
z.URL()         // RFC-compliant URL
z.IPv4()        // IPv4 dotted address
z.IPv6()        // IPv6 address
z.CreditCard()  // Valid Luhn checksum number
z.CUID()        // CUID (v1) format
z.CUID2()       // CUID2 format
z.ULID()        // ULID format
z.Base64()      // Standard Base64 encoded string
z.Hex()         // Hexadecimal encoded string
z.Emoji()       // Unicode emoji sequence
z.JWT()         // Three-part JSON Web Token
z.Hash(z.SHA256)                 // Hex digest (also z.MD5, z.SHA1, z.SHA384, z.SHA512)
z.Hash(z.SHA256, z.HashBase64)  // Base64 / base64url encoded digest
```

### ISO Date & Time Formats

```go
z.ISO.DateTime() // Full ISO-8601 UTC / with offset (RFC3339)
z.ISO.Date()     // YYYY-MM-DD
z.ISO.Time()     // HH:MM:SS[.sss]
z.ISO.Duration() // P1Y2M10DT2H30M
```

## Number Constraints

Number schemas support strict sizing and comparison checks:

```go
z.Int().
	GTE(0).        // Greater than or equal to 0 (alias: Min)
	LTE(100).      // Less than or equal to 100 (alias: Max)
	GT(0).         // Greater than 0
	LT(100).       // Less than 100
	MultipleOf(5). // Must be divisible by 5
	Positive().    // > 0
	NonNegative()  // >= 0
```

## Dates

```go
z.Date().Min(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Max(time.Now())
```

## Type Coercion

Convert incoming string or numeric representations into their target types automatically:

```go
z.Coerce.String()   // 123 -> "123"
z.Coerce.Int()      // "42" -> 42
z.Coerce.Float()    // "3.14" -> 3.14 (z.Coerce.Number() is an alias)
z.Coerce.Boolean()  // "true", 1 -> true
z.Coerce.Date()     // "2026-01-01T00:00:00Z" -> time.Time
z.Coerce.BigInt()   // "999999999999999999" -> *big.Int
```

## Additional Zod Parity Schemas

```go
z.NaN()                                   // only NaN
z.JSON()                                  // any JSON-compatible value
z.CustomFn[int](func(v any) bool { ... }) // predicate-based schema (z.custom)
z.Success(z.Int())                        // yields true/false instead of failing
z.NonOptional[string](z.String().Optional())
z.TemplateLiteral("user-", z.Int(), "@", z.Enum("a", "b"))
z.Int().Step(5)                           // alias of MultipleOf
z.Int64().Safe()                          // within ±(2^53-1)
```

`TemplateLiteral` accepts string literals, `z.String()`, `z.Boolean()`, `z.Enum(...)`, `z.Literal(...)` of strings/ints and `z.Int()`/`z.Int64()`/`z.Float64()`.
`z.Float64().Finite()` is a no-op: number schemas already reject `NaN` and `±Inf`.
`String().Normalize()` is intentionally not provided, since Unicode normalization needs `golang.org/x/text` and the module has no dependencies.
