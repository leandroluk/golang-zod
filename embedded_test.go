package z_test

import (
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

type EmbIndexable struct {
	ID string `json:"id"`
}

type EmbCreatable struct {
	CreatedAt time.Time `json:"created_at"`
}

type embEntity struct {
	EmbIndexable
	EmbCreatable
	Name string `json:"name"`
}

func embSchemas() (*z.StructSchema[EmbIndexable], *z.StructSchema[EmbCreatable]) {
	idSch := z.Struct(func(s *z.StructBuilder, t *EmbIndexable) {
		s.Field(&t.ID, z.UUID().DefaultFunc(func() string { return "00000000-0000-4000-8000-000000000001" }))
	})
	crSch := z.Struct(func(s *z.StructBuilder, t *EmbCreatable) {
		s.Field(&t.CreatedAt, z.Date().DefaultFunc(time.Now))
	})
	return idSch, crSch
}

func TestEmbeddedStructsAreFlattened(t *testing.T) {
	idSch, crSch := embSchemas()
	s := z.Struct(func(s *z.StructBuilder, e *embEntity) {
		s.Field(&e.EmbIndexable, idSch)
		s.Field(&e.EmbCreatable, crSch)
		s.Field(&e.Name, z.String())
	}).Strict()

	v, err := s.Parse(map[string]any{"name": "n"})
	if err != nil {
		t.Fatal(err)
	}
	if v.ID == "" || v.CreatedAt.IsZero() || v.Name != "n" {
		t.Fatalf("defaults not applied: %+v", v)
	}
	v, err = s.Parse(map[string]any{"id": "11111111-1111-4111-8111-111111111111", "name": "n", "created_at": "2025-01-01T00:00:00Z"})
	if err != nil || v.ID != "11111111-1111-4111-8111-111111111111" || v.CreatedAt.Year() != 2025 {
		t.Fatalf("%+v %v", v, err)
	}
	if _, err = s.Parse(map[string]any{"name": "n", "id": "bad"}); err == nil {
		t.Fatal("expected invalid uuid error")
	}
	if _, err = s.Parse(map[string]any{"name": "n", "extra": 1}); err == nil {
		t.Fatal("strict must reject unknown keys")
	}
	names := strings.Join(s.FieldNames(), ",")
	if names != "id,created_at,name" {
		t.Fatalf("field names: %s", names)
	}
	if s.FieldSchema("id") == nil {
		t.Fatal("FieldSchema should see inline fields")
	}
	out, err := z.ToJSONSchema(z.Struct(func(s *z.StructBuilder, e *embEntity) {
		s.Field(&e.EmbIndexable, idSch)
		s.Field(&e.Name, z.String())
	}))
	if err != nil || !strings.Contains(string(out), `"id"`) || !strings.Contains(string(out), `"name"`) {
		t.Fatalf("json schema: %s %v", out, err)
	}
}

func TestNilFieldSchemaPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "schema cannot be nil") {
			t.Fatalf("got %v", r)
		}
	}()
	z.Struct(func(s *z.StructBuilder, e *embEntity) {
		s.Field(&e.EmbIndexable, nil)
	})
}

type EmbOnlyBase struct {
	ID string `json:"id"`
}

type EmbOnlyEntity struct {
	EmbOnlyBase
	Name string `json:"name"`
}

type EmbOther struct {
	X int `json:"x"`
}

func TestEmbedFindsFieldBySchemaType(t *testing.T) {
	base := z.Struct(func(s *z.StructBuilder, b *EmbOnlyBase) {
		s.Field(&b.ID, z.UUID().DefaultFunc(func() string { return "00000000-0000-4000-8000-000000000002" }))
	})
	s := z.Struct(func(s *z.StructBuilder, e *EmbOnlyEntity) {
		s.Embed(base)
		s.Field(&e.Name, z.String())
	})
	v, err := s.Parse(map[string]any{"name": "n"})
	if err != nil || v.ID != "00000000-0000-4000-8000-000000000002" || v.Name != "n" {
		t.Fatalf("%+v %v", v, err)
	}
	if got := strings.Join(s.FieldNames(), ","); got != "id,name" {
		t.Fatalf("field names: %s", got)
	}
}

func TestEmbedWithoutMatchingFieldPanics(t *testing.T) {
	other := z.Struct(func(s *z.StructBuilder, b *EmbOther) { s.Field(&b.X, z.Int()) })
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "no embedded field of type") {
			t.Fatalf("got %v", r)
		}
	}()
	z.Struct(func(s *z.StructBuilder, e *EmbOnlyEntity) {
		s.Embed(other)
	})
}
