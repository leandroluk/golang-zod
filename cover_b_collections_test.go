package z_test

import (
	"context"
	"slices"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func coverBAbort() context.Context { return z.WithAbortEarly(context.Background()) }

// coverBBadTypeSchema yields an output whose dynamic type is not string and reports success,
// exercising the defensive type-assertion branches of the collection nodes.
func coverBBadTypeSchema() z.Schema[string] {
	return z.NewTestSchema[string](func(_ any, _ any) (any, bool) { return 42, true })
}

func coverBPathStr(is z.Issue) string {
	s := ""
	for _, p := range is.Path {
		if p.IsIndex() {
			s += "[" + string(rune('0'+p.IndexVal())) + "]"
		} else {
			s += "." + p.Key()
		}
	}
	return s
}

// @spec REQ-COLL-002
func TestCoverBArrayParse(t *testing.T) {
	s := z.Array(z.Int())

	t.Run("valid", func(t *testing.T) {
		got, err := s.Parse([]any{1, 2, 3})
		if err != nil || !slices.Equal(got, []int{1, 2, 3}) {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
	t.Run("nil and non-array are InvalidType", func(t *testing.T) {
		for _, in := range []any{nil, "x", 5, map[string]any{}} {
			_, err := s.Parse(in)
			if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "array" {
				t.Fatalf("input %v: unexpected issue %+v", in, is)
			}
		}
	})
	t.Run("length", func(t *testing.T) {
		_, err := s.Length(2).Parse([]any{1})
		is := coverBOneIssue(t, err)
		if is.Code != z.TooSmall || !is.Exact || is.Minimum != 2 || is.Origin != "array" {
			t.Fatalf("unexpected issue %+v", is)
		}
		if got, err := s.Length(2).Parse([]any{1, 2}); err != nil || len(got) != 2 {
			t.Fatalf("exact length should pass: (%v, %v)", got, err)
		}
	})
	t.Run("min and max", func(t *testing.T) {
		_, err := s.Min(2).Parse([]any{1})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 2 || is.Exact {
			t.Fatalf("unexpected issue %+v", is)
		}
		_, err = s.Max(1).Parse([]any{1, 2})
		if is := coverBOneIssue(t, err); is.Code != z.TooBig || is.Maximum != 1 || !is.Inclusive {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("NonEmpty", func(t *testing.T) {
		_, err := s.NonEmpty().Parse([]any{})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 1 {
			t.Fatalf("unexpected issue %+v", is)
		}
		if _, err := s.NonEmpty().Parse([]any{1}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("element issues carry index paths and are all collected", func(t *testing.T) {
		_, err := s.Parse([]any{1, "a", 3, "b"})
		is := coverBIssues(t, err)
		if len(is) != 2 || coverBPathStr(is[0]) != "[1]" || coverBPathStr(is[1]) != "[3]" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("abort early stops at the first bad element", func(t *testing.T) {
		_, err := s.Parse([]any{"a", "b", "c"}, coverBAbort())
		if is := coverBOneIssue(t, err); coverBPathStr(is) != "[0]" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("length and element issues combine", func(t *testing.T) {
		_, err := s.Min(5).Parse([]any{"a"})
		if is := coverBIssues(t, err); len(is) != 2 || is[0].Code != z.TooSmall || is[1].Code != z.InvalidType {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		_, err := s.Parse([]any{1, 2}, coverBNewCancelCtx())
		if err == nil {
			t.Fatal("expected error on cancellation")
		}
	})
	t.Run("element of unexpected dynamic type leaves zero value", func(t *testing.T) {
		got, err := z.Array(coverBBadTypeSchema()).Parse([]any{"a"})
		if err != nil || len(got) != 1 || got[0] != "" {
			t.Fatalf("got (%q, %v)", got, err)
		}
	})
}

// @spec REQ-COLL-002 REQ-COLL-006 REQ-PIPE-007 REQ-CORE-010
func TestCoverBArrayAccessors(t *testing.T) {
	el := z.Int()
	s := z.NewArraySchema[int](el)
	if s.Element() != z.Schema[int](el) || s.Unwrap() != z.Schema[int](el) {
		t.Fatal("Element/Unwrap must return the element schema")
	}
	if v, err := s.Optional().Parse([]any{1}); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	if v, err := s.Default([]int{9}).Parse(nil); err != nil || !slices.Equal(v, []int{9}) {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	m := s.Min(1).Meta(func(b *z.MetaBuilder) { b.ID("coverB_array_meta") })
	if m.GetMeta().ID != "coverB_array_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	if _, err := m.Parse([]any{}); err == nil {
		t.Fatal("Meta clone must keep constraints")
	}
	_, err := s.Error("arr err").Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "arr err" {
		t.Fatalf("Error: %q", is.Message)
	}
	_, err = s.ErrorFunc(func(*z.Issue) string { return "arr fn" }).Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "arr fn" {
		t.Fatalf("ErrorFunc: %q", is.Message)
	}
}

// @spec REQ-COLL-003
func TestCoverBTupleParse(t *testing.T) {
	s := z.Tuple(z.String(), z.Int())

	t.Run("valid", func(t *testing.T) {
		got, err := s.Parse([]any{"a", 1})
		if err != nil || len(got) != 2 || got[0] != "a" || got[1] != 1 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
	t.Run("nil is InvalidType tuple", func(t *testing.T) {
		_, err := s.Parse(nil)
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "tuple" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("non-array is InvalidType array", func(t *testing.T) {
		_, err := s.Parse("x")
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "array" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("too short", func(t *testing.T) {
		_, err := s.Parse([]any{"a"})
		is := coverBOneIssue(t, err)
		if is.Code != z.TooSmall || !is.Exact || is.Minimum != 2 || is.Origin != "array" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("too long", func(t *testing.T) {
		_, err := s.Parse([]any{"a", 1, 2})
		is := coverBOneIssue(t, err)
		if is.Code != z.TooBig || !is.Exact || is.Maximum != 2 || is.Origin != "array" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("item failures carry index paths", func(t *testing.T) {
		_, err := s.Parse([]any{1, "x"})
		is := coverBIssues(t, err)
		if len(is) != 2 || coverBPathStr(is[0]) != "[0]" || coverBPathStr(is[1]) != "[1]" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("abort early", func(t *testing.T) {
		_, err := s.Parse([]any{1, "x"}, coverBAbort())
		if is := coverBOneIssue(t, err); coverBPathStr(is) != "[0]" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		if _, err := s.Parse([]any{"a", 1}, coverBNewCancelCtx()); err == nil {
			t.Fatal("expected error on cancellation")
		}
	})

	t.Run("rest", func(t *testing.T) {
		r := s.Rest(z.Int())
		got, err := r.Parse([]any{"a", 1, 2, 3})
		if err != nil || len(got) != 4 || got[3] != 3 {
			t.Fatalf("got (%v, %v)", got, err)
		}
		if got, err := r.Parse([]any{"a", 1}); err != nil || len(got) != 2 {
			t.Fatalf("zero extra items should pass: (%v, %v)", got, err)
		}
		_, err = r.Parse([]any{"a"})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Exact || is.Minimum != 2 {
			t.Fatalf("unexpected issue %+v", is)
		}
		_, err = r.Parse([]any{"a", 1, "bad"})
		if is := coverBOneIssue(t, err); coverBPathStr(is) != "[2]" || is.Code != z.InvalidType {
			t.Fatalf("unexpected issue %+v", is)
		}
		if _, err := s.Parse([]any{"a", 1, 2}); err == nil {
			t.Fatal("Rest must not mutate the original tuple")
		}
	})
}

// @spec REQ-COLL-003 REQ-COLL-006 REQ-PIPE-007 REQ-CORE-010
func TestCoverBTupleModifiers(t *testing.T) {
	s := z.Tuple(z.String())
	if v, err := s.Optional().Parse([]any{"a"}); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	if v, err := s.Default([]any{"d"}).Parse(nil); err != nil || len(v) != 1 || v[0] != "d" {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{[]any{"a"}, []any{"b"}}); err != nil || len(v) != 2 || v[1][0] != "b" {
		t.Fatalf("Array = (%v, %v)", v, err)
	}
	r := s.Rest(z.Int())
	m := r.Meta(func(b *z.MetaBuilder) { b.ID("coverB_tuple_meta") })
	if m.GetMeta().ID != "coverB_tuple_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	if _, err := m.Parse([]any{"a", 1, 2}); err != nil {
		t.Fatalf("Meta clone must keep rest: %v", err)
	}
	_, err := s.Error("tup err").Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "tup err" {
		t.Fatalf("Error: %q", is.Message)
	}
	_, err = s.ErrorFunc(func(*z.Issue) string { return "tup fn" }).Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "tup fn" {
		t.Fatalf("ErrorFunc: %q", is.Message)
	}
}

// @spec REQ-COLL-004
func TestCoverBSetParse(t *testing.T) {
	s := z.Set(z.String())

	t.Run("duplicates are collapsed", func(t *testing.T) {
		got, err := s.Parse([]any{"a", "b", "a"})
		if err != nil || len(got) != 2 {
			t.Fatalf("got (%v, %v)", got, err)
		}
		if _, ok := got["a"]; !ok {
			t.Fatalf("missing a in %v", got)
		}
	})
	t.Run("nil and non-array", func(t *testing.T) {
		for _, in := range []any{nil, "x", map[string]any{}} {
			_, err := s.Parse(in)
			if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "set" {
				t.Fatalf("input %v: unexpected issue %+v", in, is)
			}
		}
	})
	t.Run("size counts unique items", func(t *testing.T) {
		if _, err := s.Size(1).Parse([]any{"a", "a"}); err != nil {
			t.Fatalf("duplicates collapse to size 1: %v", err)
		}
		_, err := s.Size(2).Parse([]any{"a", "a"})
		is := coverBOneIssue(t, err)
		if is.Code != z.TooSmall || !is.Exact || is.Minimum != 2 || is.Origin != "set" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("min and max", func(t *testing.T) {
		_, err := s.Min(2).Parse([]any{"a"})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 2 || is.Exact {
			t.Fatalf("unexpected issue %+v", is)
		}
		_, err = s.Max(1).Parse([]any{"a", "b"})
		if is := coverBOneIssue(t, err); is.Code != z.TooBig || is.Maximum != 1 || is.Origin != "set" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("NonEmpty", func(t *testing.T) {
		_, err := s.NonEmpty().Parse([]any{})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 1 {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("element issues and abort early", func(t *testing.T) {
		_, err := s.Parse([]any{"a", 1, 2})
		if is := coverBIssues(t, err); len(is) != 2 || coverBPathStr(is[0]) != "[1]" || coverBPathStr(is[1]) != "[2]" {
			t.Fatalf("unexpected issues %+v", is)
		}
		_, err = s.Parse([]any{1, 2}, coverBAbort())
		if is := coverBOneIssue(t, err); coverBPathStr(is) != "[0]" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		if _, err := s.Parse([]any{"a"}, coverBNewCancelCtx()); err == nil {
			t.Fatal("expected error on cancellation")
		}
	})
	t.Run("element of unexpected dynamic type is skipped", func(t *testing.T) {
		got, err := z.Set(coverBBadTypeSchema()).Parse([]any{"a"})
		if err != nil || len(got) != 0 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
}

// @spec REQ-COLL-004 REQ-COLL-006 REQ-PIPE-007 REQ-CORE-010
func TestCoverBSetModifiers(t *testing.T) {
	s := z.Set(z.String())
	if v, err := s.Optional().Parse([]any{"a"}); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	def := map[string]struct{}{"d": {}}
	if v, err := s.Default(def).Parse(nil); err != nil || len(v) != 1 {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{[]any{"a"}}); err != nil || len(v) != 1 || len(v[0]) != 1 {
		t.Fatalf("Array = (%v, %v)", v, err)
	}
	m := s.Min(1).Meta(func(b *z.MetaBuilder) { b.ID("coverB_set_meta") })
	if m.GetMeta().ID != "coverB_set_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	if _, err := m.Parse([]any{}); err == nil {
		t.Fatal("Meta clone must keep constraints")
	}
	_, err := s.Error("set err").Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "set err" {
		t.Fatalf("Error: %q", is.Message)
	}
	_, err = s.ErrorFunc(func(*z.Issue) string { return "set fn" }).Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "set fn" {
		t.Fatalf("ErrorFunc: %q", is.Message)
	}
}

// @spec REQ-COLL-005
func TestCoverBMapParse(t *testing.T) {
	s := z.Map(z.String(), z.Int())

	t.Run("valid", func(t *testing.T) {
		got, err := s.Parse(map[string]any{"a": 1, "b": 2})
		if err != nil || len(got) != 2 || got["a"] != 1 || got["b"] != 2 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
	t.Run("typed map passes through", func(t *testing.T) {
		in := map[string]int{"x": 9}
		got, err := s.Parse(in)
		if err != nil || got["x"] != 9 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
	t.Run("nil and non-map", func(t *testing.T) {
		for _, in := range []any{nil, "x", []any{}, map[int]int{1: 1}} {
			_, err := s.Parse(in)
			if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "map" {
				t.Fatalf("input %v: unexpected issue %+v", in, is)
			}
		}
	})
	t.Run("size", func(t *testing.T) {
		_, err := s.Size(2).Parse(map[string]any{"a": 1})
		is := coverBOneIssue(t, err)
		if is.Code != z.TooSmall || !is.Exact || is.Minimum != 2 || is.Origin != "map" {
			t.Fatalf("unexpected issue %+v", is)
		}
		if _, err := s.Size(1).Parse(map[string]any{"a": 1}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("min and max", func(t *testing.T) {
		_, err := s.Min(2).Parse(map[string]any{"a": 1})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 2 || is.Exact {
			t.Fatalf("unexpected issue %+v", is)
		}
		_, err = s.Max(1).Parse(map[string]any{"a": 1, "b": 2})
		if is := coverBOneIssue(t, err); is.Code != z.TooBig || is.Maximum != 1 || is.Origin != "map" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("NonEmpty", func(t *testing.T) {
		_, err := s.NonEmpty().Parse(map[string]any{})
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall || is.Minimum != 1 {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("value failure carries the key in the path", func(t *testing.T) {
		_, err := s.Parse(map[string]any{"a": "no"})
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType || coverBPathStr(is) != ".a" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("key failure yields InvalidKey", func(t *testing.T) {
		ks := z.Map(z.String().Min(2), z.Int())
		_, err := ks.Parse(map[string]any{"a": 1})
		is := coverBIssues(t, err)
		if len(is) != 2 || is[0].Code != z.TooSmall || is[1].Code != z.InvalidKey || is[1].Input != "a" || is[1].Origin != "map" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("abort early on key and value failures", func(t *testing.T) {
		ks := z.Map(z.String().Min(2), z.Int())
		_, err := ks.Parse(map[string]any{"a": 1, "b": 2}, coverBAbort())
		if is := coverBIssues(t, err); len(is) != 2 || is[1].Code != z.InvalidKey {
			t.Fatalf("unexpected issues %+v", is)
		}
		_, err = s.Parse(map[string]any{"a": "x", "b": "y"}, coverBAbort())
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("all failures are collected", func(t *testing.T) {
		_, err := s.Parse(map[string]any{"a": "x", "b": "y"})
		if is := coverBIssues(t, err); len(is) != 2 {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		if _, err := s.Parse(map[string]any{"a": 1}, coverBNewCancelCtx()); err == nil {
			t.Fatal("expected error on cancellation")
		}
	})
	t.Run("key of unexpected dynamic type fails", func(t *testing.T) {
		bad := z.Map(coverBBadTypeSchema(), z.Int())
		if _, err := bad.Parse(map[string]any{"a": 1}); err == nil {
			t.Fatal("expected failure")
		}
	})
	t.Run("value of unexpected dynamic type is skipped", func(t *testing.T) {
		bad := z.Map(z.String(), coverBBadTypeSchema())
		got, err := bad.Parse(map[string]any{"a": "x"})
		if err != nil || len(got) != 0 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
}

// @spec REQ-COLL-005 REQ-COLL-006 REQ-CORE-010
func TestCoverBMapModifiers(t *testing.T) {
	s := z.Map(z.String(), z.Int())
	if v, err := s.Optional().Parse(map[string]any{"a": 1}); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	if v, err := s.Default(map[string]int{"d": 4}).Parse(nil); err != nil || v["d"] != 4 {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{map[string]any{"a": 1}}); err != nil || len(v) != 1 || v[0]["a"] != 1 {
		t.Fatalf("Array = (%v, %v)", v, err)
	}
	m := s.Size(1).Meta(func(b *z.MetaBuilder) { b.ID("coverB_map_meta") })
	if m.GetMeta().ID != "coverB_map_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	if _, err := m.Parse(map[string]any{}); err == nil {
		t.Fatal("Meta clone must keep constraints")
	}
}

// @spec REQ-OBJ-008
func TestCoverBRecordParse(t *testing.T) {
	s := z.Record(z.String(), z.Int())

	t.Run("valid", func(t *testing.T) {
		got, err := s.Parse(map[string]any{"a": 1, "b": 2})
		if err != nil || len(got) != 2 || got["b"] != 2 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
	t.Run("nil and non-map", func(t *testing.T) {
		for _, in := range []any{nil, "x", []any{}} {
			_, err := s.Parse(in)
			if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "record" {
				t.Fatalf("input %v: unexpected issue %+v", in, is)
			}
		}
	})
	t.Run("value failure path", func(t *testing.T) {
		_, err := s.Parse(map[string]any{"a": "x"})
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType || coverBPathStr(is) != ".a" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("invalid key", func(t *testing.T) {
		ks := z.Record(z.String().Min(2), z.Int())
		_, err := ks.Parse(map[string]any{"a": 1})
		is := coverBIssues(t, err)
		if len(is) != 2 || is[1].Code != z.InvalidKey || is[1].Origin != "record" || coverBPathStr(is[1]) != ".a" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("abort early", func(t *testing.T) {
		ks := z.Record(z.String().Min(2), z.Int())
		_, err := ks.Parse(map[string]any{"a": 1, "b": 2}, coverBAbort())
		if is := coverBIssues(t, err); len(is) != 2 || is[1].Code != z.InvalidKey {
			t.Fatalf("unexpected issues %+v", is)
		}
		_, err = s.Parse(map[string]any{"a": "x", "b": "y"}, coverBAbort())
		if is := coverBOneIssue(t, err); is.Code != z.InvalidType {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("all value failures collected", func(t *testing.T) {
		_, err := s.Parse(map[string]any{"a": "x", "b": "y"})
		if is := coverBIssues(t, err); len(is) != 2 {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("cancellation", func(t *testing.T) {
		if _, err := s.Parse(map[string]any{"a": 1}, coverBNewCancelCtx()); err == nil {
			t.Fatal("expected error on cancellation")
		}
	})
	t.Run("key and value of unexpected dynamic type", func(t *testing.T) {
		if _, err := z.Record(coverBBadTypeSchema(), z.Int()).Parse(map[string]any{"a": 1}); err == nil {
			t.Fatal("expected failure for key of wrong dynamic type")
		}
		got, err := z.Record(z.String(), coverBBadTypeSchema()).Parse(map[string]any{"a": "x"})
		if err != nil || len(got) != 0 {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
}

// @spec REQ-OBJ-008
func TestCoverBRecordEnumKeys(t *testing.T) {
	keys := z.Enum("a", "b")

	t.Run("default mode requires exhaustive keys", func(t *testing.T) {
		s := z.Record(keys, z.Int())
		if got, err := s.Parse(map[string]any{"a": 1, "b": 2}); err != nil || len(got) != 2 {
			t.Fatalf("got (%v, %v)", got, err)
		}
		_, err := s.Parse(map[string]any{"a": 1})
		is := coverBOneIssue(t, err)
		if is.Code != z.InvalidType || is.Expected != "required" || coverBPathStr(is) != ".b" {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("unknown keys are reported sorted", func(t *testing.T) {
		s := z.Record(keys, z.Int())
		_, err := s.Parse(map[string]any{"a": 1, "b": 2, "d": 3, "c": 4})
		is := coverBOneIssue(t, err)
		if is.Code != z.UnrecognizedKeys || !slices.Equal(is.Keys, []string{"c", "d"}) {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("PartialRecord does not require exhaustiveness", func(t *testing.T) {
		s := z.PartialRecord(keys, z.Int())
		got, err := s.Parse(map[string]any{"a": 1})
		if err != nil || len(got) != 1 || got["a"] != 1 {
			t.Fatalf("got (%v, %v)", got, err)
		}
		_, err = s.Parse(map[string]any{"a": 1, "z": 2})
		if is := coverBOneIssue(t, err); is.Code != z.UnrecognizedKeys || !slices.Equal(is.Keys, []string{"z"}) {
			t.Fatalf("unexpected issue %+v", is)
		}
	})
	t.Run("LooseRecord skips non-matching keys", func(t *testing.T) {
		s := z.LooseRecord(keys, z.Int())
		got, err := s.Parse(map[string]any{"a": 1, "zzz": "ignored"})
		if err != nil || len(got) != 1 || got["a"] != 1 {
			t.Fatalf("got (%v, %v)", got, err)
		}
		lk := z.LooseRecord(z.String().Min(2), z.Int())
		got2, err := lk.Parse(map[string]any{"a": 1, "bb": 2})
		if err != nil || len(got2) != 1 || got2["bb"] != 2 {
			t.Fatalf("got (%v, %v)", got2, err)
		}
	})
}

// @spec REQ-OBJ-008 REQ-OBJ-009 REQ-CORE-010
func TestCoverBRecordModifiers(t *testing.T) {
	s := z.Record(z.String(), z.Int())
	if v, err := s.Optional().Parse(map[string]any{"a": 1}); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	if v, err := s.Default(map[string]int{"d": 4}).Parse(nil); err != nil || v["d"] != 4 {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{map[string]any{"a": 1}}); err != nil || len(v) != 1 || v[0]["a"] != 1 {
		t.Fatalf("Array = (%v, %v)", v, err)
	}

	// Meta preserves the mode (loose records still skip unknown enum keys).
	m := z.LooseRecord(z.Enum("a"), z.Int()).Meta(func(b *z.MetaBuilder) { b.ID("coverB_record_meta") })
	if m.GetMeta().ID != "coverB_record_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	if got, err := m.Parse(map[string]any{"q": 1}); err != nil || len(got) != 0 {
		t.Fatalf("Meta must preserve loose mode: (%v, %v)", got, err)
	}
}

// @spec REQ-EXP-007
func TestCoverBCollectionNodeKinds(t *testing.T) {
	for name, s := range map[string]z.AnySchema{
		"bigint": z.BigInt(),
		"map":    z.Map(z.String(), z.Int()),
		"set":    z.Set(z.String()),
	} {
		if _, err := z.ToJSONSchema(s); err == nil {
			t.Errorf("%s: expected unrepresentable error", name)
		}
	}
	if _, err := z.ToJSONSchema(z.Record(z.String(), z.Int())); err != nil {
		t.Fatalf("record should be representable: %v", err)
	}
}
