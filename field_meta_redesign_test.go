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
		s.Field("id", z.UUID())
		s.Field("email", z.Email())
		s.Field("name", z.String().Min(3))
		s.Field("age", z.Int().GTE(18))
		s.Field("score", z.Float64().GTE(0))
		s.Field("active", z.Boolean())
		s.Field("bio", z.String().Optional())
		s.Field("tags", z.Array(z.String()))
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
		s.Field(&u.ID, z.UUID())
		s.Field(&u.Email, z.Email())
		s.Field(&u.Name, z.String().Min(3))
		s.Field(&u.Age, z.Int().GTE(18))
		s.Field(&u.Score, z.Float64().GTE(0))
		s.Field(&u.Active, z.Boolean())
		s.Field(&u.Nick, z.String().Optional())
		s.Field(&u.Role, z.Enum("ADMIN", "USER"))
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
		s.Field("title", z.String())
	}).Meta(func(m *z.MetaBuilder) {
		m.Title("ObjectTitle")
		m.Description("ObjectDescription")
	})

	objMeta := z.GetMeta(obj)
	if objMeta.Title != "ObjectTitle" || objMeta.Description != "ObjectDescription" {
		t.Errorf("unexpected object meta: %+v", objMeta)
	}
}

func TestStructDefaultMetaIDAndOverride(t *testing.T) {
	// 1. Default Meta.ID equals struct name
	userSchema := z.Struct(func(s *z.StructBuilder, u *RedesignUser) {
		s.Field(&u.Name, z.String())
	})

	meta := z.GetMeta(userSchema)
	if meta.ID != "RedesignUser" {
		t.Fatalf("expected default Meta.ID %q, got %q", "RedesignUser", meta.ID)
	}

	// 2. Overwriting ID via .Meta(m.ID(...))
	overridden := userSchema.Meta(func(m *z.MetaBuilder) {
		m.ID("CustomUserProfile").Title("User Profile")
	})

	oMeta := z.GetMeta(overridden)
	if oMeta.ID != "CustomUserProfile" {
		t.Fatalf("expected overridden Meta.ID %q, got %q", "CustomUserProfile", oMeta.ID)
	}
	if oMeta.Title != "User Profile" {
		t.Fatalf("expected Title %q, got %q", "User Profile", oMeta.Title)
	}

	// Original receiver must remain unchanged
	if orig := z.GetMeta(userSchema).ID; orig != "RedesignUser" {
		t.Fatalf("expected original Meta.ID to remain %q, got %q", "RedesignUser", orig)
	}
}

type exampleEnum string

func TestExamplesAcceptsTypedSlice(t *testing.T) {
	vals := []exampleEnum{"a", "b"}
	sch := z.Enum(vals...).Meta(func(m *z.MetaBuilder) {
		m.Examples(vals...)
		m.Examples[any](1, "x")
	})
	got := sch.GetMeta().Examples
	if len(got) != 4 || got[0] != exampleEnum("a") || got[2] != 1 {
		t.Fatalf("examples: %#v", got)
	}
}
