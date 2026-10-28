# Error Handling & Internationalization (i18n)

`golang-zod` provides a comprehensive error model following Zod v4's exact 11 issue codes, paired with an integrated multilingual translation engine.

## The `ZodError` Structure

Validation failures produce a `*z.ZodError`:

```go
type Issue struct {
	Code      IssueCode      `json:"code"`
	Path      []PathSegment  `json:"path"`
	Message   string         `json:"message"`
	Expected  string         `json:"expected,omitempty"`
	Minimum   any            `json:"minimum,omitempty"`
	Maximum   any            `json:"maximum,omitempty"`
	Inclusive bool           `json:"inclusive,omitempty"`
	Origin    string         `json:"origin,omitempty"`
	Keys      []string       `json:"keys,omitempty"`
	Values    []any          `json:"values,omitempty"`
}
```

### Issue Codes

1. `InvalidType`: Expected type mismatch (e.g., expected `string`, received `int`).
2. `InvalidValue`: Input value rejected by scalar constraints.
3. `TooSmall`: String length, number value, or slice count below minimum.
4. `TooBig`: String length, number value, or slice count exceeding maximum.
5. `UnrecognizedKeys`: Unexpected properties present in `Strict()` mode.
6. `InvalidUnion`: No branches matched in `Union`.
7. `InvalidFormat`: Malformed UUID, email, IP, or ISO timestamp.
8. `NotMultipleOf`: Number is not a multiple of specified factor.
9. `NotFinite`: Non-finite floating point numbers (`NaN`, `Inf`).
10. `Custom`: Custom refinement or transform error.

---

## Error Formatters

`golang-zod` includes 3 built-in formatters:

### 1. `Flatten()`

Produces a dictionary mapping top-level field names to their error messages:

```go
flat := zerr.Flatten()
// map[string][]string{
//     "email": {"Invalid email format"},
//     "age":   {"Number must be greater than or equal to 18"},
// }
```

### 2. `Treeify()`

Builds a formatted multiline ASCII tree depicting errors organized by nesting hierarchy:

```text
✖ 2 validation errors:
  └─ user
     ├─ email: Invalid email address
     └─ profile
        └─ bio: String must contain at least 10 character(s)
```

### 3. `Prettify()`

Generates clean human-readable summaries formatted for logs and console output.

---

## Built-in Internationalization (i18n)

`golang-zod` includes pre-translated locale catalogs:
- `en`: English (Default)
- `pt-BR`: Brazilian Portuguese
- `es`: Spanish

### Setting Global Locale

```go
z.SetLocale(z.LocalePtBR)
```

### Per-Request Scoped Locale (`z.WithLocale`)

In web servers (such as Fiber, Echo, Gin, or standard `net/http`), pass the request's `Accept-Language` header:

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Parse and negotiate Accept-Language header (e.g. "pt-BR,pt;q=0.9,en;q=0.8")
	ctx := z.WithLocale(r.Context(), r.Header.Get("Accept-Language"))

	user, err := userSchema.ParseJSON(body, ctx)
	if err != nil {
		// Issues are translated into Portuguese automatically!
	}
}
```

### Registering Custom Locales

A locale implements `z.Locale` (`Tag() string` and `FormatMessage(issue *z.Issue) string`):

```go
type frenchLocale struct{}

func (frenchLocale) Tag() string { return "fr" }
func (frenchLocale) FormatMessage(issue *z.Issue) string {
	switch issue.Code {
	case z.InvalidType:
		return "Type invalide : attendu " + issue.Expected
	default:
		return z.LocaleEN.FormatMessage(issue)
	}
}

z.RegisterLocale("fr", frenchLocale{})
```

---

## Custom Error Messages

Messages can be customized at three levels. From highest to lowest precedence: the **check**, the **schema**, and a **global function**; the locale catalog is the fallback. A function that returns `""` falls through to the next level.

```go
// Check level: trailing options on String and Number checks
z.String().Min(5, z.Message("too short"))
z.Int().Min(18, z.MessageFunc(func(i *z.Issue) string { return "adults only" }))
z.String().Min(5, z.Abort(true)).Regex(re) // stop evaluating checks if Min fails

// Schema level
z.String().Error("must be text")
z.String().ErrorFunc(func(i *z.Issue) string { return "bad " + i.Expected })

// Global level (application startup)
z.Config.UseError(func(i *z.Issue) string { return "" }) // nil removes it
```

A schema-level message applies only to the issues that schema itself produces, never to the issues of its children.
