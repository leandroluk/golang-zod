# Getting Started

`golang-zod` is a pure Go port of TypeScript's Zod v4, designed for macro-free, zero-reflect runtime schema validation, serialization, and API modeling.

## Installation

```bash
go get github.com/leandroluk/golang-zod
```

Requires Go **1.27.1** or newer.

## Basic Usage

Import the library using standard Go package aliasing:

```go
package main

import (
	"fmt"
	z "github.com/leandroluk/golang-zod"
)

func main() {
	// Define schema
	schema := z.Object(func(s *z.Shape) {
		s.Field("name", z.String().Min(2))
		s.Field("age", z.Int().GTE(0))
		s.Field("email", z.Email())
	})

	// Parse input map
	result, err := schema.Parse(map[string]any{
		"name":  "Bob",
		"age":   25,
		"email": "bob@example.com",
	})
	if err != nil {
		fmt.Printf("Validation error: %v\n", err)
		return
	}

	fmt.Printf("Name: %v, Age: %v\n", result["name"], result["age"])
}
```

## Parsing JSON directly

`golang-zod` natively supports streaming JSON byte parsing without manual `json.Unmarshal` boilerplates:

```go
jsonData := []byte(`{"name": "Alice", "age": 30, "email": "alice@example.com"}`)

result, err := schema.ParseJSON(jsonData)
if err != nil {
	panic(err)
}
```

## Error Handling

When validation fails, `golang-zod` returns a structured `*z.ZodError`:

```go
_, err := schema.Parse(map[string]any{
	"name":  "A", // Too short
	"email": "invalid-email",
})

if err != nil {
	if zerr, ok := err.(*z.ZodError); ok {
		for _, iss := range zerr.Issues() {
			fmt.Printf("Path: %s, Code: %s, Message: %s\n",
				iss.PathString(), iss.Code, iss.Message)
		}

		// Or use built-in formatters:
		fmt.Println(zerr.Flatten())
		fmt.Println(zerr.Treeify())
		fmt.Println(zerr.Prettify())
	}
}
```
