package z_test

import (
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

type ptrEntity struct {
	DeletedAt *time.Time `json:"deleted_at"`
}

func TestPtrStructField(t *testing.T) {
	s := z.Struct(func(s *z.StructBuilder, e *ptrEntity) {
		s.Field(&e.DeletedAt).Date().Ptr()
	})
	v, err := s.Parse(map[string]any{})
	if err != nil || v.DeletedAt != nil {
		t.Fatalf("absent: %+v %v", v, err)
	}
	v, err = s.Parse(map[string]any{"deleted_at": nil})
	if err != nil || v.DeletedAt != nil {
		t.Fatalf("null: %+v %v", v, err)
	}
	v, err = s.Parse(map[string]any{"deleted_at": "2025-01-01T00:00:00Z"})
	if err != nil || v.DeletedAt == nil || v.DeletedAt.Year() != 2025 {
		t.Fatalf("set: %+v %v", v, err)
	}
	if _, err = s.Parse(map[string]any{"deleted_at": "nope"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestStructFieldTypeMismatchPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "use .Ptr()") {
			t.Fatalf("expected mismatch panic, got %v", r)
		}
	}()
	z.Struct(func(s *z.StructBuilder, e *ptrEntity) {
		s.Field(&e.DeletedAt).Date().Optional()
	})
}

func TestPtrFieldAllowsNilWithoutPtrCall(t *testing.T) {
	s := z.Struct(func(s *z.StructBuilder, e *ptrEntity) {
		s.Field(&e.DeletedAt).Date()
	})
	for _, in := range []map[string]any{{}, {"deleted_at": nil}} {
		v, err := s.Parse(in)
		if err != nil || v.DeletedAt != nil {
			t.Fatalf("%v: %+v %v", in, v, err)
		}
	}
	v, err := s.Parse(map[string]any{"deleted_at": "2025-01-01T00:00:00Z"})
	if err != nil || v.DeletedAt == nil || v.DeletedAt.Year() != 2025 {
		t.Fatalf("set: %+v %v", v, err)
	}
	if _, err = s.Parse(map[string]any{"deleted_at": "nope"}); err == nil {
		t.Fatal("expected error")
	}
	ns := z.Struct(func(s *z.StructBuilder, e *ptrStr) { s.Field(&e.Nick).String() })
	out, err := z.ToJSONSchema(ns)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"required"`) {
		t.Fatalf("pointer field must not be required: %s", out)
	}
}

type ptrStr struct {
	Nick *string `json:"nick"`
}

type metaEntity struct {
	ID        string           `json:"id"`
	CreatedAt *time.Time       `json:"created_at"`
	Nick      z.Option[string] `json:"nick"`
}

func TestMetaOnWrappersAndFields(t *testing.T) {
	s := z.Struct(func(s *z.StructBuilder, e *metaEntity) {
		s.Field(&e.ID).UUID().DefaultFunc(func() string { return "x" }).Meta(func(m *z.MetaBuilder) {
			m.Description("the id")
		})
		s.Field(&e.CreatedAt).Date().Meta(func(m *z.MetaBuilder) { m.Description("created") })
		s.Field(&e.Nick).String().Optional().Meta(func(m *z.MetaBuilder) { m.Description("nick") })
	})
	for name, want := range map[string]string{"id": "the id", "created_at": "created", "nick": "nick"} {
		if got := s.FieldSchema(name).GetMeta().Description; got != want {
			t.Fatalf("%s: description %q, want %q", name, got, want)
		}
	}
}
