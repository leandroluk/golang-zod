package z_test

import (
	"context"
	"reflect"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type CoverCUser struct {
	ID     int              `json:"id"`
	Name   string           `json:"name"`
	Nick   z.Option[string] `json:"nick"`
	Bio    string           `json:"bio"`
	Score  z.Nullable[int]  `json:"score,omitempty"`
	Hidden string           `json:"-"`
	NoTag  string
	secret string
}

type CoverCNested struct {
	X     int `json:"x"`
	Inner struct {
		A int `json:"a"`
		B int `json:"b"`
	} `json:"inner"`
}

func coverCUserSchema() *z.StructSchema[CoverCUser] {
	return z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) {
		s.Field(&u.ID, z.Int())
		s.Field(&u.Name, z.String())
		s.Field(&u.Nick, z.String().Optional())
		s.Field(&u.Bio, z.String().Default("none"))
		s.Field(&u.Score, z.Int().Nullable())
	})
}

// @spec REQ-OBJ-004
func TestCoverCStructConstructionPanics(t *testing.T) {
	var outside int
	cases := []struct {
		name  string
		want  string
		build func()
	}{
		{"non struct", "T must be a struct", func() {
			z.Struct[int](func(s *z.StructShape, v *int) {})
		}},
		{"nil pointer", "field pointer cannot be nil", func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) { s.Field[int](nil, z.Int()) })
		}},
		{"pointer outside struct", "does not point to a field of struct T", func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) { s.Field(&outside, z.Int()) })
		}},
		{"nested field path", "embedded or nested field paths", func() {
			z.Struct[CoverCNested](func(s *z.StructShape, u *CoverCNested) { s.Field(&u.Inner.B, z.Int()) })
		}},
		{"nested field at same offset as parent has mismatched type", "does not match schema type", func() {
			z.Struct[CoverCNested](func(s *z.StructShape, u *CoverCNested) { s.Field(&u.Inner.A, z.Int()) })
		}},
		{"unexported field", "must be exported", func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) { s.Field(&u.secret, z.String()) })
		}},
		{"missing json tag", "mandatory `json:\"name\"` tag", func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) { s.Field(&u.NoTag, z.String()) })
		}},
		{"json dash", "mandatory `json:\"name\"` tag", func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) { s.Field(&u.Hidden, z.String()) })
		}},
		{"empty json name", "must specify a valid json name", func() {
			type T struct {
				V string `json:",omitempty"`
			}
			z.Struct[T](func(s *z.StructShape, u *T) { s.Field(&u.V, z.String()) })
		}},
		{"dash json name with options", "must specify a valid json name", func() {
			type T struct {
				//lint:ignore SA5008 intentionally malformed tag: the constructor must panic on it
				V string `json:"-,"`
			}
			z.Struct[T](func(s *z.StructShape, u *T) { s.Field(&u.V, z.String()) })
		}},
		{"duplicate registration", `duplicate json field name "id"`, func() {
			z.Struct[CoverCUser](func(s *z.StructShape, u *CoverCUser) {
				s.Field(&u.ID, z.Int())
				s.Field(&u.ID, z.Int())
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			coverCMustPanic(t, tc.want, tc.build)
		})
	}
}

// @spec REQ-OBJ-004 REQ-OBJ-005 REQ-OBJ-003
func TestCoverCStructParse(t *testing.T) {
	sch := coverCUserSchema()

	t.Run("full input", func(t *testing.T) {
		u, err := sch.Parse(map[string]any{"id": 7, "name": "Ann", "nick": "A", "bio": "hi", "score": nil, "ignored": 1})
		if err != nil {
			t.Fatal(err)
		}
		if u.ID != 7 || u.Name != "Ann" || u.Bio != "hi" || u.Nick.OrElse("") != "A" || !u.Score.IsNull() {
			t.Fatalf("unexpected %+v", u)
		}
	})
	t.Run("absent optional stays zero, default applied, unregistered untouched", func(t *testing.T) {
		u, err := sch.Parse(map[string]any{"id": 1, "name": "N", "score": 5})
		if err != nil {
			t.Fatal(err)
		}
		if u.Nick.IsSome() || u.Bio != "none" || u.Hidden != "" || u.NoTag != "" {
			t.Fatalf("unexpected %+v", u)
		}
		if v, ok := u.Score.Value(); !ok || v != 5 {
			t.Fatalf("score %+v", u.Score)
		}
	})
	t.Run("nil input", func(t *testing.T) {
		_, err := sch.Parse(nil)
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "object" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("non map input", func(t *testing.T) {
		_, err := sch.Parse([]any{1})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "object" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("missing required and invalid value", func(t *testing.T) {
		_, err := sch.Parse(map[string]any{"name": 5, "score": 1})
		is := coverCIssues(t, err)
		if len(is) != 2 {
			t.Fatalf("unexpected issues %+v", is)
		}
		if is[0].Expected != "required" || coverCPath(is[0]) != "id" {
			t.Fatalf("unexpected first issue %+v", is[0])
		}
		if is[1].Code != z.InvalidType || coverCPath(is[1]) != "name" {
			t.Fatalf("unexpected second issue %+v", is[1])
		}
	})
	t.Run("parse json", func(t *testing.T) {
		u, err := sch.ParseJSON([]byte(`{"id":3,"name":"J","score":null}`))
		if err != nil || u.ID != 3 || u.Name != "J" {
			t.Fatalf("%+v %v", u, err)
		}
	})
}

// @spec REQ-OBJ-004 REQ-CORE-017
func TestCoverCStructAbortEarlyAndCancel(t *testing.T) {
	sch := coverCUserSchema()
	ctx := z.WithAbortEarly(context.Background())

	t.Run("missing required stops", func(t *testing.T) {
		_, err := sch.Parse(map[string]any{}, ctx)
		if is := coverCIssues(t, err); len(is) != 1 || coverCPath(is[0]) != "id" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("invalid value stops", func(t *testing.T) {
		_, err := sch.Parse(map[string]any{"id": "x", "name": 1}, ctx)
		if is := coverCIssues(t, err); len(is) != 1 || coverCPath(is[0]) != "id" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("cancel between fields", func(t *testing.T) {
		cctx, cancel := context.WithCancel(context.Background())
		canceller := z.NewTestSchema[string](func(any, any) (any, bool) {
			cancel()
			return "ok", true
		})
		type T struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		s := z.Struct[T](func(s *z.StructShape, v *T) {
			s.Field(&v.A, canceller)
			s.Field(&v.B, z.String())
		})
		if _, err := s.Parse(map[string]any{"a": "x", "b": "y"}, cctx); err == nil {
			t.Fatal("expected error after cancellation")
		}
	})
}

// @spec REQ-OBJ-006
func TestCoverCStructStrictStrip(t *testing.T) {
	sch := coverCUserSchema()
	in := map[string]any{"id": 1, "name": "n", "score": 0, "zz": 1, "aa": 2}

	_, err := sch.Strict().Parse(in)
	is := coverCIssues(t, err)
	if len(is) != 1 || is[0].Code != z.UnrecognizedKeys || !reflect.DeepEqual(is[0].Keys, []string{"aa", "zz"}) {
		t.Fatalf("unexpected issues %+v", is)
	}
	if _, err = sch.Strict().Parse(map[string]any{"id": 1, "name": "n", "score": 0}); err != nil {
		t.Fatalf("strict with no extra keys must pass: %v", err)
	}
	if _, err = sch.Strict().Strip().Parse(in); err != nil {
		t.Fatalf("strip must ignore unknown keys: %v", err)
	}
	if _, err = sch.Parse(in); err != nil {
		t.Fatalf("default must strip: %v", err)
	}
}

// @spec REQ-OBJ-006 REQ-OBJ-009
func TestCoverCStructPickOmitKeyof(t *testing.T) {
	sch := coverCUserSchema()

	if got := sch.FieldNames(); !reflect.DeepEqual(got, []string{"id", "name", "nick", "bio", "score"}) {
		t.Fatalf("got %v", got)
	}
	if sch.FieldSchema("id") == nil {
		t.Fatal("expected schema for id")
	}
	if sch.FieldSchema("missing") != nil {
		t.Fatal("expected nil schema for unknown key")
	}
	if got := sch.Keyof().Values(); !reflect.DeepEqual(got, []string{"id", "name", "nick", "bio", "score"}) {
		t.Fatalf("got %v", got)
	}

	picked := sch.Pick("name", "id")
	if got := picked.FieldNames(); !reflect.DeepEqual(got, []string{"name", "id"}) {
		t.Fatalf("got %v", got)
	}
	u, err := picked.Parse(map[string]any{"id": 2, "name": "P"})
	if err != nil || u.ID != 2 || u.Name != "P" {
		t.Fatalf("%+v %v", u, err)
	}
	coverCMustPanic(t, `unknown key "zz" in .Pick()`, func() { sch.Pick("zz") })

	omitted := sch.Omit("id", "score")
	if got := omitted.FieldNames(); !reflect.DeepEqual(got, []string{"name", "nick", "bio"}) {
		t.Fatalf("got %v", got)
	}
	u, err = omitted.Parse(map[string]any{"name": "O"})
	if err != nil || u.ID != 0 || u.Name != "O" || u.Bio != "none" {
		t.Fatalf("omitted fields stay zero: %+v %v", u, err)
	}
	coverCMustPanic(t, `unknown key "zz" in .Omit()`, func() { sch.Omit("zz") })

	if len(sch.FieldNames()) != 5 {
		t.Fatal("Pick/Omit must not mutate the receiver")
	}
}

// @spec REQ-OPT-007a
func TestCoverCStructParseTo(t *testing.T) {
	sch := coverCUserSchema()

	var dst CoverCUser
	if err := sch.ParseTo([]byte(`{"id":9,"name":"To","score":1}`), &dst); err != nil {
		t.Fatal(err)
	}
	if dst.ID != 9 || dst.Name != "To" || dst.Bio != "none" {
		t.Fatalf("unexpected %+v", dst)
	}

	if err := sch.ParseTo([]byte(`{}`), nil); err == nil {
		t.Fatal("nil destination must fail")
	}

	before := dst
	err := sch.ParseTo([]byte(`{"id":"bad"}`), &dst)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if dst != before {
		t.Fatal("destination must not change on error")
	}
	if err = sch.ParseTo([]byte(`{`), &dst); err == nil {
		t.Fatal("expected JSON syntax error")
	}
}

// @spec REQ-OBJ-009 REQ-CORE-010 REQ-PIPE-007 REQ-OBJ-004
func TestCoverCStructModifiersAndNesting(t *testing.T) {
	sch := coverCUserSchema()
	valid := map[string]any{"id": 1, "name": "n", "score": 1}

	if _, err := sch.Optional().Parse(nil); err == nil {
		t.Fatal("Optional rejects null")
	}
	if out, err := sch.Optional().Parse(valid); err != nil || !out.IsSome() {
		t.Fatalf("Optional: %+v %v", out, err)
	}
	if out, err := sch.Nullable().Parse(nil); err != nil || !out.IsNull() {
		t.Fatalf("Nullable: %+v %v", out, err)
	}
	if _, err := sch.Nullish().Parse(valid); err != nil {
		t.Fatalf("Nullish: %v", err)
	}
	if out, err := sch.Default(CoverCUser{ID: 99}).Parse(nil); err != nil || out.ID != 99 {
		t.Fatalf("Default: %+v %v", out, err)
	}
	arr, err := sch.Array().Parse([]any{valid, valid})
	if err != nil || len(arr) != 2 || arr[1].Name != "n" {
		t.Fatalf("Array: %+v %v", arr, err)
	}

	withMeta := sch.Meta(func(m *z.MetaBuilder) { m.ID("coverC_struct_meta").Title("U") })
	if m := withMeta.GetMeta(); m.ID != "coverC_struct_meta" || m.Title != "U" {
		t.Fatalf("meta: %+v", m)
	}
	if sch.GetMeta().ID != "" {
		t.Fatal("Meta must not mutate the receiver")
	}

	_, err = sch.Error("struct msg").Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "struct msg" {
		t.Fatalf("got %q", is[0].Message)
	}
	_, err = sch.ErrorFunc(func(i *z.Issue) string { return "fn" }).Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "fn" {
		t.Fatalf("got %q", is[0].Message)
	}

	// nested struct is a field whose schema is another z.Struct
	type Inner struct {
		V int `json:"v"`
	}
	type Outer struct {
		In Inner `json:"in"`
	}
	inner := z.Struct[Inner](func(s *z.StructShape, i *Inner) { s.Field(&i.V, z.Int()) })
	outer := z.Struct[Outer](func(s *z.StructShape, o *Outer) { s.Field(&o.In, inner) })
	o, err := outer.Parse(map[string]any{"in": map[string]any{"v": 4}})
	if err != nil || o.In.V != 4 {
		t.Fatalf("%+v %v", o, err)
	}
	_, err = outer.Parse(map[string]any{"in": map[string]any{"v": "x"}})
	if is := coverCIssues(t, err); len(is) != 1 || coverCPath(is[0]) != "in.v" {
		t.Fatalf("unexpected issues %+v", is)
	}
}
