# Codecs & Pipelines

`golang-zod` features first-class support for transformations, custom refinements, and bidirectional codecs.

## Refinements

Refinements attach domain validation predicates while preserving the schema's underlying Go type `T`.

```go
// Boolean predicate
evenNumber := z.Refine[int](z.Int(), func(n int) bool {
	return n%2 == 0
}, z.RefineMessage("must be an even number"))

// Imperative multi-issue refinement
passwordSchema := z.SuperRefine[string](z.String(), func(pwd string, c *z.RefineCtx) {
	if len(pwd) < 8 {
		c.AddIssue(z.Issue{Code: z.TooSmall, Message: "password must be at least 8 characters"})
	}
	if !strings.ContainsAny(pwd, "!@#$%^&*") {
		c.AddIssue(z.Issue{Code: z.Custom, Message: "password must contain a symbol"})
	}
})

// Context-aware refinement (cancellation / DB check)
uniqueUser := z.RefineWithCtx[string](z.String(), func(ctx context.Context, email string) bool {
	return checkEmailNotInDB(ctx, email)
})
```

---

## Pipelines & Transforms

```go
// Transform converts type T -> U
toUpper := z.Transform[string, string](z.String(), func(s string) (string, error) {
	return strings.ToUpper(s), nil
})

// Preprocess mutates raw input before validation
trimmedJSON := z.Preprocess[int](func(raw any) any {
	if str, ok := raw.(string); ok {
		return len(str)
	}
	return raw
}, z.Int())

// Pipe chains two schemas
pipe := z.Pipe[string, int](z.String(), z.Codecs.StringInt())
```

---

## Bidirectional Codecs (`z.Codec`)

Codecs define symmetric two-way transformations supporting both `Decode` and `Encode`:

```go
codec := z.Codec[string, int](
	z.String(),
	z.Int(),
	func(s string) (int, error) { return strconv.Atoi(s) }, // Decode: string -> int
	func(i int) (string, error) { return strconv.Itoa(i), nil }, // Encode: int -> string
)

// Decode
num, err := codec.DecodeValue("42") // 42

// Encode
str, err := codec.Encode(42) // "42"
```

### Pre-packaged Recipe Codecs (`z.Codecs`)

- `z.Codecs.StringInt()`: Converts string <-> `int`.
- `z.Codecs.StringNumber()`: Converts string <-> `float64`.
- `z.Codecs.Base64Bytes()`: Converts Base64 string <-> `[]byte`.
- `z.Codecs.EpochDate()`: Converts Unix milliseconds <-> `time.Time`.
- `z.Codecs.JSONString()`: Converts JSON string <-> `any`.

### Configurable String Boolean Codec (`z.StringBool`)

```go
sb := z.StringBool(
	z.StringBoolTruthy("yes", "true", "1", "sim"),
	z.StringBoolFalsy("no", "false", "0", "nao"),
	z.StringBoolCaseSensitive(false),
)

val, _ := sb.DecodeValue("SIM") // true
str, _ := sb.Encode(true)        // "yes" (first truthy item)
```

## Inverting a codec

`z.InvertCodec(c)` swaps the directions of a codec: the output schema becomes the input and `Encode`/`Parse` trade places.

```go
inv := z.InvertCodec(intFromString) // *CodecSchema[int, string]
s, _ := inv.Parse(5)                // "5"
```
