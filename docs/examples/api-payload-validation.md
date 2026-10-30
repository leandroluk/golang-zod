# Example: Web API Payload Validation

This recipe shows how to integrate `golang-zod` into a Go HTTP service using standard library `net/http` or frameworks like Fiber, Gin, or Echo with per-request internationalization.

## Complete Example

```go
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	z "github.com/leandroluk/golang-zod"
)

// 1. Define Domain Struct
type CreateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Password string `json:"password"`
}

// 2. Compile Schema with Zero-Reflect Runtime Performance
var CreateUserSchema = z.Struct[CreateUserRequest](func(s *z.StructBuilder, u *CreateUserRequest) {
	s.Field(&u.Name, z.String().Min(2).Trim())
	s.Field(&u.Email, z.Email())
	s.Field(&u.Age, z.Int().GTE(18))
	s.Field(&u.Password, z.String().Min(8))
})

// 3. HTTP Handler with Accept-Language Negotiation
func createUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Negotiate language from client's Accept-Language header
	reqCtx := z.WithLocale(r.Context(), r.Header.Get("Accept-Language"))

	// Validate and parse directly into typed struct
	user, err := CreateUserSchema.ParseJSON(body, reqCtx)
	if err != nil {
		if zerr, ok := err.(*z.ZodError); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   "validation_failed",
				"details": zerr.Flatten(),
			})
			return
		}

		http.Error(w, "Malformed JSON", http.StatusBadRequest)
		return
	}

	// Process valid user
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "user_created",
		"name":   user.Name,
		"email":  user.Email,
	})
}

func main() {
	http.HandleFunc("/api/users", createUserHandler)
	log.Println("Server running at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
