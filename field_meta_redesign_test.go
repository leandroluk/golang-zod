package z_test

import (
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type RedesignUser struct {
	ID       string             `json:"id"`
	Email    string             `json:"email"`
	Name     string             `json:"name"`
	Age      int                `json:"age"`
	Score    float64            `json:"score"`
	Active   bool               `json:"active"`
	Nick     z.Option[string]   `json:"nick"`
	Role     string             `json:"role"`
}

func TestObjectFieldAliasesAndChaining(t *testing.T) {
	schema := z.Object(func(s *z.ObjectBuilder) {
		s.Field("id").UUID()
		s.Field("email").Email()
		s.Field("name").String().Min(3)
		s.Field("age").Int().GTE(18)
		s.Field("score").Float64().GTE(0)
		s.Field("active").Boolean()
		s.Field("bio").String().Optional()
		s.Field("tags").Schema(z.Array(z.String()))
	})

	valid := map[string]any{
		"id":     "550e8400-e29b-41d4-a716-446655440000",
		"email":  "dev@example.com",
		"name":   "Alice",
		"age":    25,
		"score":  99.5,
		"active": true,
		"tags":   []any{"admin", "go"},
	}

	out, err := schema.Parse(valid)
	if err != nil {
		t.Fatalf("expected valid parse, got err: %v", err)
	}
	if out["name"] != "Alice" || out["age"] != 25 {
		t.Fatalf("unexpected parsed values: %+v", out)
	}

	// Invalid input should fail appropriately
	invalid := map[string]any{
		"id":     "not-a-uuid",
		"email":  "invalid-email",
		"name":   "Al", // < 3
		"age":    16,   // < 18
		"score":  -1.0, // < 0
		"active": "not-bool",
		"tags":   "not-array",
	}
	_, err = schema.Parse(invalid)
	if err == nil {
		t.Fatal("expected error on invalid input, got nil")
	}
}

func TestStructFieldAliasesAndChaining(t *testing.T) {
	userSchema := z.Struct(func(s *z.StructBuilder, u *RedesignUser) {
		s.Field(&u.ID).UUID()
		s.Field(&u.Email).Email()
		s.Field(&u.Name).String().Min(3)
		s.Field(&u.Age).Int().GTE(18)
		s.Field(&u.Score).Float64().GTE(0)
		s.Field(&u.Active).Boolean()
		s.Field(&u.Nick).Schema(z.String().Optional())
		s.Field(&u.Role).Schema(z.Enum("ADMIN", "USER"))
	})

	validJSON := []byte(`{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"email": "ana@example.com",
		"name": "Ana Silva",
		"age": 28,
		"score": 98.7,
		"active": true,
		"nick": "anita",
		"role": "ADMIN"
	}`)

	u, err := userSchema.ParseJSON(validJSON)
	if err != nil {
		t.Fatalf("expected valid JSON parse, got err: %v", err)
	}
	if u.Name != "Ana Silva" || u.Age != 28 || !u.Active {
		t.Fatalf("unexpected struct values: %+v", u)
	}
	val, ok := u.Nick.Value()
	if !ok || val != "anita" {
		t.Fatalf("unexpected nick: %+v", u.Nick)
	}

	// Absent optional field
	validAbsentNick := []byte(`{
		"id": "550e8400-e29b-41d4-a716-446655440000",
		"email": "ana@example.com",
		"name": "Ana Silva",
		"age": 28,
		"score": 98.7,
		"active": true,
		"role": "USER"
	}`)
	u2, err := userSchema.ParseJSON(validAbsentNick)
	if err != nil {
		t.Fatalf("expected valid parse with absent optional field, got: %v", err)
	}
	if u2.Nick.IsSome() {
		t.Fatalf("expected nick to be None, got: %+v", u2.Nick)
	}
}

func TestMetaAndGetMeta(t *testing.T) {
	s := z.String().Meta(func(m *z.MetaBuilder) {
		m.Title("Username")
		m.Description("Nome de usuário único")
		m.Examples("ana", "bia")
	})

	meta := z.GetMeta(s)
	if meta.Title != "Username" {
		t.Errorf("expected Title %q, got %q", "Username", meta.Title)
	}
	if meta.Description != "Nome de usuário único" {
		t.Errorf("expected Description %q, got %q", "Nome de usuário único", meta.Description)
	}
	if len(meta.Examples) != 2 || meta.Examples[0] != "ana" || meta.Examples[1] != "bia" {
		t.Errorf("unexpected Examples: %+v", meta.Examples)
	}

	// Also verify on Object
	obj := z.Object(func(s *z.ObjectBuilder) {
		s.Field("title").String()
	}).Meta(func(m *z.MetaBuilder) {
		m.Title("ObjectTitle")
		m.Description("ObjectDescription")
	})

	objMeta := z.GetMeta(obj)
	if objMeta.Title != "ObjectTitle" || objMeta.Description != "ObjectDescription" {
		t.Errorf("unexpected object meta: %+v", objMeta)
	}
}
