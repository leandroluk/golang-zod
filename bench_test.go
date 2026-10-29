package z_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-OPT-001
func TestREQ_OPT_001_InlinePathStackSpill(t *testing.T) {
	// Build a 12-level deeply nested object schema
	var current z.AnySchema = z.Int()
	for i := 11; i >= 0; i-- {
		lvlName := fmt.Sprintf("level_%d", i)
		sub := current
		current = z.Object(func(s *z.ObjectBuilder) {
			s.Field(lvlName, sub)
		})
	}

	// Payload with invalid type at innermost level
	nestedPayload := make(map[string]any)
	currMap := nestedPayload
	for i := 0; i < 11; i++ {
		nextMap := make(map[string]any)
		currMap[fmt.Sprintf("level_%d", i)] = nextMap
		currMap = nextMap
	}
	currMap["level_11"] = "not-an-int"

	sch := current.(z.Schema[z.Dyn])
	_, err := sch.Parse(nestedPayload)
	if err == nil {
		t.Fatal("expected validation error at innermost level")
	}

	zerr := err.(*z.ZodError)
	issues := zerr.Issues()
	if len(issues) == 0 {
		t.Fatal("expected at least one issue")
	}
	if len(issues[0].Path) != 12 {
		t.Fatalf("expected path length 12 after spill, got %d: %v", len(issues[0].Path), issues[0].Path)
	}
}

// @spec REQ-OPT-004
func TestREQ_OPT_004_PoolIsolation_Concurrent(t *testing.T) {
	schema := z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String().Min(3))
		s.Field("age", z.Int().GTE(0))
	})

	var wg sync.WaitGroup
	workers := 20
	iterations := 500

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				if i%2 == 0 {
					// Valid payload
					res, err := schema.Parse(map[string]any{
						"name": fmt.Sprintf("User_%d_%d", workerID, i),
						"age":  i,
					})
					if err != nil {
						t.Errorf("unexpected error on valid parse: %v", err)
						return
					}
					if res["age"] != i {
						t.Errorf("mismatched value in pooled parse")
						return
					}
				} else {
					// Invalid payload
					_, err := schema.Parse(map[string]any{
						"name": "a", // Too short
						"age":  -1,  // Negative
					})
					if err == nil {
						t.Errorf("expected validation error on invalid payload")
						return
					}
				}
			}
		}(w)
	}
	wg.Wait()
}

// @spec REQ-OPT-005
type BenchStruct struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func BenchmarkPrimitiveString(b *testing.B) {
	schema := z.String()
	input := "hello world"

	for b.Loop() {
		_, _ = schema.Parse(input)
	}
}

func BenchmarkEmail(b *testing.B) {
	schema := z.Email()
	input := "test.user@example.com"

	for b.Loop() {
		_, _ = schema.Parse(input)
	}
}

func BenchmarkObjectParse(b *testing.B) {
	schema := z.Object(func(s *z.ObjectBuilder) {
		s.Field("id", z.Int())
		s.Field("name", z.String())
		s.Field("email", z.Email())
	})
	input := map[string]any{
		"id":    101,
		"name":  "Alice",
		"email": "alice@example.com",
	}

	for b.Loop() {
		_, _ = schema.Parse(input)
	}
}

func BenchmarkStructParse(b *testing.B) {
	schema := z.Struct(func(s *z.StructBuilder, u *BenchStruct) {
		s.Field(&u.ID, z.Int())
		s.Field(&u.Name, z.String())
		s.Field(&u.Email, z.Email())
	})
	input := map[string]any{
		"id":    101,
		"name":  "Alice",
		"email": "alice@example.com",
	}

	for b.Loop() {
		_, _ = schema.Parse(input)
	}
}

func BenchmarkStructParseTo(b *testing.B) {
	schema := z.Struct(func(s *z.StructBuilder, u *BenchStruct) {
		s.Field(&u.ID, z.Int())
		s.Field(&u.Name, z.String())
		s.Field(&u.Email, z.Email())
	})
	jsonData := []byte(`{"id": 101, "name": "Alice", "email": "alice@example.com"}`)
	var dst BenchStruct

	for b.Loop() {
		_ = schema.ParseTo(jsonData, &dst, context.Background())
	}
}
