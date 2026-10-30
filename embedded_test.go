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

type EmbExpBase struct {
	ID   string  `json:"id"`
	Note *string `json:"note"`
}

type EmbExpEntity struct {
	EmbExpBase
	Name string           `json:"name"`
	Nick z.Option[string] `json:"nick"`
}

// Embedding merges the embedded schema's properties and requiredness into the parent's export,
// like Zod's `.extend(other.shape)`.
func TestEmbedMergesIntoJSONSchema(t *testing.T) {
	base := z.Struct(func(s *z.StructBuilder, b *EmbExpBase) {
		s.Field(&b.ID, z.UUID().DefaultFunc(func() string { return "x" }).Meta(func(m *z.MetaBuilder) { m.Description("the id") }))
		s.Field(&b.Note, z.String().Meta(func(m *z.MetaBuilder) { m.Description("note") }))
	})
	s := z.Struct(func(s *z.StructBuilder, e *EmbExpEntity) {
		s.Embed(base)
		s.Field(&e.Name, z.String())
		s.Field(&e.Nick, z.String().Optional())
	})
	in, err := z.ToJSONSchema(s, z.WithIO(z.Input))
	if err != nil {
		t.Fatal(err)
	}
	out, err := z.ToJSONSchema(s, z.WithIO(z.Output))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"id":{"description":"the id"`, `"note":{"description":"note"`, `"name"`, `"nick"`} {
		if !strings.Contains(string(in), want) || !strings.Contains(string(out), want) {
			t.Fatalf("missing %s", want)
		}
	}
	// Input: id has a default and note is a pointer, so only name is required.
	if !strings.Contains(string(in), `"required":["name"]`) {
		t.Fatalf("input required: %s", in)
	}
	// Output: id is always filled by its default; pointer and optional fields stay optional.
	if !strings.Contains(string(out), `"required":["id","name"]`) {
		t.Fatalf("output required: %s", out)
	}
}

type EmbExBase struct {
	ID string `json:"id"`
}

type EmbExEntity struct {
	EmbExBase
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestEmbedMergesExamples(t *testing.T) {
	base := z.Struct(func(s *z.StructBuilder, b *EmbExBase) {
		s.Field(&b.ID, z.String().Meta(func(m *z.MetaBuilder) { m.Examples("abc") }))
	})
	// Field-level examples only: one object is assembled from embedded and own fields.
	fromFields := z.Struct(func(s *z.StructBuilder, e *EmbExEntity) {
		s.Embed(base)
		s.Field(&e.Name, z.String().Meta(func(m *z.MetaBuilder) { m.Examples("Ana") }))
		s.Field(&e.Age, z.Int())
	})
	out, err := z.ToJSONSchema(fromFields)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"examples":[{"id":"abc","name":"Ana"}]`) {
		t.Fatalf("assembled examples: %s", out)
	}
	// An explicit object example on the parent overrides on top of the embedded one.
	explicit := z.Struct(func(s *z.StructBuilder, e *EmbExEntity) {
		s.Embed(base)
		s.Field(&e.Name, z.String())
		s.Field(&e.Age, z.Int())
	}).Meta(func(m *z.MetaBuilder) { m.Examples(map[string]any{"name": "Bia", "age": 30}) })
	out, err = z.ToJSONSchema(explicit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"examples":[{"age":30,"id":"abc","name":"Bia"}]`) {
		t.Fatalf("explicit examples: %s", out)
	}
}
