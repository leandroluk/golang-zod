package z_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// coverCConst is a schema that ignores its input and yields a fixed value.
func coverCConst(v any) z.Schema[any] {
	return z.NewTestSchema[any](func(any, any) (any, bool) { return v, true })
}

func coverCFail() z.Schema[any] {
	return z.NewTestSchema[any](func(ctx any, in any) (any, bool) {
		z.ContextAddIssue(ctx, z.Issue{Code: z.Custom, Message: "boom"})
		return nil, false
	})
}

func coverCShape(typ string) *z.ObjectSchema {
	return z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal(typ)) })
}

// --- Union / XOR ---

// @spec REQ-COMB-001
func TestCoverCUnionFirstMatch(t *testing.T) {
	u := z.Union(z.Int(), z.String())

	for _, tc := range []struct {
		in   any
		want any
	}{{1, 1}, {"x", "x"}} {
		got, err := u.Parse(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("Parse(%v) = %v, %v", tc.in, got, err)
		}
	}

	_, err := u.Parse(true)
	is := coverCIssues(t, err)
	if len(is) != 1 || is[0].Code != z.InvalidUnion || !is[0].Inclusive || is[0].Input != true {
		t.Fatalf("unexpected issues %+v", is)
	}
	if len(is[0].Errors) != 2 || is[0].Errors[0][0].Code != z.InvalidType || is[0].Errors[1][0].Code != z.InvalidType {
		t.Fatalf("expected one error list per option: %+v", is[0].Errors)
	}
}

// @spec REQ-COMB-003
func TestCoverCXOR(t *testing.T) {
	x := z.XOR(z.Any(), z.String())

	got, err := x.Parse(5)
	if err != nil || got != 5 {
		t.Fatalf("exactly one match: %v %v", got, err)
	}

	_, err = x.Parse("s")
	is := coverCIssues(t, err)
	if len(is) != 1 || is[0].Code != z.InvalidUnion || is[0].Inclusive ||
		!reflect.DeepEqual(is[0].Matches, []int{0, 1}) ||
		is[0].Message != "Invalid input: more than one option matched" {
		t.Fatalf("unexpected issues %+v", is)
	}

	none := z.XOR(z.Int(), z.String())
	_, err = none.Parse(true)
	is = coverCIssues(t, err)
	if len(is) != 1 || is[0].Code != z.InvalidUnion || !is[0].Inclusive || len(is[0].Errors) != 2 {
		t.Fatalf("unexpected issues %+v", is)
	}

	// an option that reports ok together with issues does not count as a match
	partial := z.NewTestSchema[any](func(ctx any, in any) (any, bool) {
		z.ContextAddIssue(ctx, z.Issue{Code: z.Custom, Message: "soft"})
		return in, true
	})
	_, err = z.Union(partial).Parse(1)
	if is = coverCIssues(t, err); len(is[0].Errors) != 1 || is[0].Errors[0][0].Message != "soft" {
		t.Fatalf("unexpected issues %+v", is)
	}
	_, err = z.XOR(partial).Parse(1)
	if is = coverCIssues(t, err); len(is[0].Errors) != 1 || is[0].Errors[0][0].Message != "soft" {
		t.Fatalf("unexpected issues %+v", is)
	}
}

// @spec REQ-COMB-001 REQ-CORE-017
func TestCoverCUnionCancelled(t *testing.T) {
	cctx, cancel := context.WithCancel(context.Background())
	canceller := z.NewTestSchema[any](func(ctx any, in any) (any, bool) {
		cancel()
		return nil, false
	})
	// the outer union's second option is a union that observes the cancellation at entry
	u := z.Union(z.Union(canceller), z.Union(z.String()))
	_, err := u.Parse("x", cctx)
	is := coverCIssues(t, err)
	if len(is) != 1 || is[0].Code != z.InvalidUnion {
		t.Fatalf("unexpected issues %+v", is)
	}
	if len(is[0].Errors) != 2 || len(is[0].Errors[1]) != 0 {
		t.Fatalf("cancelled option must fail without issues: %+v", is[0].Errors)
	}
}

// --- Discriminated union ---

// @spec REQ-COMB-002
func TestCoverCDiscriminatedUnionParse(t *testing.T) {
	cat := z.Object(func(s *z.ObjectBuilder) {
		s.Field("type", z.Literal("cat"))
		s.Field("lives", z.Int())
	})
	type Dog struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	dog := z.Struct[Dog](func(s *z.StructBuilder, d *Dog) {
		s.Field(&d.Type, z.Literal("dog"))
		s.Field(&d.Name, z.String())
	})
	bird := z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Enum("bird", "parrot")) })
	num := z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal[any](1)) })

	u := z.DiscriminatedUnion("type", cat, dog, bird, num)

	t.Run("dispatches to object", func(t *testing.T) {
		got, err := u.Parse(map[string]any{"type": "cat", "lives": 9})
		if err != nil || !reflect.DeepEqual(got, z.Dyn{"type": "cat", "lives": 9}) {
			t.Fatalf("%v %v", got, err)
		}
	})
	t.Run("dispatches to struct", func(t *testing.T) {
		got, err := u.Parse(map[string]any{"type": "dog", "name": "Rex"})
		if err != nil || got != (Dog{Type: "dog", Name: "Rex"}) {
			t.Fatalf("%v %v", got, err)
		}
	})
	t.Run("enum registers its first value", func(t *testing.T) {
		if _, err := u.Parse(map[string]any{"type": "bird"}); err != nil {
			t.Fatal(err)
		}
		if _, err := u.Parse(map[string]any{"type": "parrot"}); err == nil {
			t.Fatal("only the first enum value is registered")
		}
	})
	t.Run("literal over any renders value", func(t *testing.T) {
		if _, err := u.Parse(map[string]any{"type": 1}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("option failure is reported with its path", func(t *testing.T) {
		_, err := u.Parse(map[string]any{"type": "cat", "lives": "x"})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || coverCPath(is[0]) != "lives" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("non object input", func(t *testing.T) {
		_, err := u.Parse("cat")
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "object" || is[0].Input != "cat" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("missing discriminator", func(t *testing.T) {
		_, err := u.Parse(map[string]any{"lives": 1})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidUnion || is[0].Note != "No matching discriminator" ||
			is[0].Discriminator != "type" || coverCPath(is[0]) != "type" || is[0].Input != nil ||
			!reflect.DeepEqual(is[0].Options, []any{"cat", "dog", "bird", "1"}) {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("unknown discriminator value", func(t *testing.T) {
		_, err := u.Parse(map[string]any{"type": "fish"})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidUnion || is[0].Note != "No matching discriminator" ||
			is[0].Input != "fish" || coverCPath(is[0]) != "type" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("as object field uses discriminated kind", func(t *testing.T) {
		wrap := z.Object(func(s *z.ObjectBuilder) { s.Field("pet", u) })
		out, err := wrap.Parse(map[string]any{"pet": map[string]any{"type": "bird"}})
		if err != nil || out["pet"] == nil {
			t.Fatalf("%v %v", out, err)
		}
	})
}

// @spec REQ-COMB-002
func TestCoverCDiscriminatedUnionConstructionPanics(t *testing.T) {
	cases := []struct {
		name string
		want string
		fn   func()
	}{
		{"option lacks discriminator", `option lacks discriminator field "type"`, func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("other", z.String()) }))
		}},
		{"discriminator is not a literal or enum", "must be a Literal or Enum", func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.String()) }))
		}},
		{"literal with int values is not usable", "must be a Literal or Enum", func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal(1)) }))
		}},
		{"literal without values", "must be a Literal or Enum", func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal[string]()) }))
		}},
		{"literal over any without values", "must be a Literal or Enum", func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal[any]()) }))
		}},
		{"empty literal value", "must be a Literal or Enum", func() {
			z.DiscriminatedUnion("type", z.Object(func(s *z.ObjectBuilder) { s.Field("type", z.Literal("")) }))
		}},
		{"duplicate discriminator value", `duplicate discriminator value "a"`, func() {
			z.DiscriminatedUnion("type", coverCShape("a"), coverCShape("a"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { coverCMustPanic(t, tc.want, tc.fn) })
	}
}

// --- Union modifiers ---

// @spec REQ-COMB-008 REQ-CORE-010 REQ-PIPE-007
func TestCoverCUnionModifiers(t *testing.T) {
	u := z.Union(z.Int(), z.String())

	if _, err := u.Optional().Parse(nil); err == nil {
		t.Fatal("Optional rejects null")
	}
	if out, err := u.Optional().Parse(1); err != nil || !out.IsSome() {
		t.Fatalf("Optional: %v %v", out, err)
	}
	if out, err := u.Nullable().Parse(nil); err != nil || !out.IsNull() {
		t.Fatalf("Nullable: %v %v", out, err)
	}
	if _, err := u.Nullish().Parse("s"); err != nil {
		t.Fatalf("Nullish: %v", err)
	}
	if out, err := u.Default("dflt").Parse(nil); err != nil || out != "dflt" {
		t.Fatalf("Default: %v %v", out, err)
	}
	arr, err := u.Array().Parse([]any{1, "a"})
	if err != nil || !reflect.DeepEqual(arr, []any{1, "a"}) {
		t.Fatalf("Array: %v %v", arr, err)
	}

	if m := u.Meta(func(m *z.MetaBuilder) { m.ID("coverC_union_meta") }).GetMeta(); m.ID != "coverC_union_meta" {
		t.Fatalf("meta: %+v", m)
	}
	if u.GetMeta().ID != "" {
		t.Fatal("Meta must not mutate the receiver")
	}

	_, err = u.Error("union msg").Parse(true)
	if is := coverCIssues(t, err); is[0].Message != "union msg" {
		t.Fatalf("got %q", is[0].Message)
	}
	_, err = u.ErrorFunc(func(*z.Issue) string { return "union fn" }).Parse(true)
	if is := coverCIssues(t, err); is[0].Message != "union fn" {
		t.Fatalf("got %q", is[0].Message)
	}

	// kind() for plain/XOR unions when used as an object field
	wrap := z.Object(func(s *z.ObjectBuilder) {
		s.Field("u", u)
		s.Field("x", z.XOR(z.Int(), z.String()))
	})
	if _, err = wrap.Parse(map[string]any{"u": 1, "x": "s"}); err != nil {
		t.Fatal(err)
	}
}

// @spec REQ-COMB-006 REQ-COMB-008
func TestCoverCUnionExporterAccessors(t *testing.T) {
	plain, err := z.ToJSONSchema(z.Union(z.Int(), z.String()))
	if err != nil || !strings.Contains(string(plain), "oneOf") {
		t.Fatalf("%s %v", plain, err)
	}
	disc, err := z.ToJSONSchema(z.DiscriminatedUnion("type", coverCShape("a"), coverCShape("b")), z.WithTarget(z.OpenAPI31))
	if err != nil || !strings.Contains(string(disc), `"propertyName":"type"`) {
		t.Fatalf("%s %v", disc, err)
	}
}

// --- Intersection ---

// @spec REQ-COMB-004
func TestCoverCIntersectionObjects(t *testing.T) {
	left := z.Object(func(s *z.ObjectBuilder) { s.Field("id", z.Int()) })
	right := z.Object(func(s *z.ObjectBuilder) { s.Field("name", z.String()) })
	inter := z.Intersection(left, right)

	got, err := inter.Parse(map[string]any{"id": 1, "name": "n"})
	if err != nil || !reflect.DeepEqual(got, z.Dyn{"id": 1, "name": "n"}) {
		t.Fatalf("%v %v", got, err)
	}

	t.Run("left failure propagates", func(t *testing.T) {
		_, err := inter.Parse(map[string]any{"id": "x", "name": "n"})
		is := coverCIssues(t, err)
		if len(is) != 1 || coverCPath(is[0]) != "id" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("right failure propagates", func(t *testing.T) {
		_, err := inter.Parse(map[string]any{"id": 1, "name": 5})
		is := coverCIssues(t, err)
		if len(is) != 1 || coverCPath(is[0]) != "name" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("same key with equal values merges", func(t *testing.T) {
		both := z.Intersection(
			z.Object(func(s *z.ObjectBuilder) { s.Field("k", z.Int()) }),
			z.Object(func(s *z.ObjectBuilder) { s.Field("k", z.Int()) }),
		)
		got, err := both.Parse(map[string]any{"k": 3})
		if err != nil || !reflect.DeepEqual(got, z.Dyn{"k": 3}) {
			t.Fatalf("%v %v", got, err)
		}
	})
	t.Run("conflict is an unmergeable custom issue with path", func(t *testing.T) {
		conflict := z.Intersection(
			z.Object(func(s *z.ObjectBuilder) { s.Field("k", z.Int()) }),
			z.Object(func(s *z.ObjectBuilder) { s.Field("k", z.Int().Default(0)) }),
		)
		if _, err := conflict.Parse(map[string]any{"k": 3}); err != nil {
			t.Fatalf("identical values must merge: %v", err)
		}
		bad := z.Intersection(coverCConst(z.Dyn{"k": 1}), coverCConst(z.Dyn{"k": 2}))
		_, err := bad.Parse(nil)
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.Custom ||
			!strings.HasPrefix(is[0].Message, z.ErrUnmergeableIntersection.Error()+": ") ||
			!strings.Contains(is[0].Message, `path "k"`) {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
}

// @spec REQ-COMB-004
func TestCoverCIntersectionMergeValues(t *testing.T) {
	type pair struct {
		name  string
		a, b  any
		want  any
		fails string // substring of the conflict message; empty when merge succeeds
	}
	cases := []pair{
		{name: "nil and nil", a: nil, b: nil, want: nil},
		{name: "nil vs value", a: nil, b: 1, fails: "nil vs non-nil"},
		{name: "value vs nil", a: 1, b: nil, fails: "nil vs non-nil"},

		{name: "equal ints", a: 1, b: 1, want: 1},
		{name: "equal strings", a: "s", b: "s", want: "s"},
		{name: "different ints", a: 1, b: 2, fails: "conflicting scalar values (1 != 2)"},
		{name: "same number different types", a: 1, b: int64(1), fails: "conflicting scalar values"},
		{name: "scalar vs slice", a: 1, b: []string{"a"}, fails: "conflicting scalar values"},

		{name: "disjoint maps merge", a: z.Dyn{"a": 1}, b: z.Dyn{"b": 2}, want: z.Dyn{"a": 1, "b": 2}},
		{name: "nested maps merge", a: z.Dyn{"o": z.Dyn{"a": 1}}, b: z.Dyn{"o": z.Dyn{"b": 2}}, want: z.Dyn{"o": z.Dyn{"a": 1, "b": 2}}},
		{name: "nested map conflict path", a: z.Dyn{"o": z.Dyn{"a": 1}}, b: z.Dyn{"o": z.Dyn{"a": 2}}, fails: `path "o.a"`},
		{name: "map vs scalar", a: z.Dyn{"a": 1}, b: 1, fails: "conflicting scalar values"},

		{name: "any slices merge elementwise", a: []any{1, z.Dyn{"a": 1}}, b: []any{1, z.Dyn{"b": 2}}, want: []any{1, z.Dyn{"a": 1, "b": 2}}},
		{name: "empty any slices", a: []any{}, b: []any{}, want: []any{}},
		{name: "any slices length mismatch", a: []any{1}, b: []any{1, 2}, fails: "array lengths differ (1 != 2)"},
		{name: "any slice element conflict path", a: z.Dyn{"l": []any{1, 2}}, b: z.Dyn{"l": []any{1, 3}}, fails: `path "l[1]"`},
		{name: "any slice root element conflict path", a: []any{1}, b: []any{2}, fails: `path "[0]"`},

		{name: "equal string slices", a: []string{"a"}, b: []string{"a"}, want: []string{"a"}},
		{name: "different string slices", a: []string{"a"}, b: []string{"b"}, fails: "conflicting scalar values"},
		{name: "string slice vs int slice", a: []string{"a"}, b: []int{1}, fails: "conflicting scalar values"},
		{name: "equal int slices", a: []int{1, 2}, b: []int{1, 2}, want: []int{1, 2}},
		{name: "different int slices", a: []int{1}, b: []int{1, 2}, fails: "conflicting scalar values"},
		{name: "equal int64 slices", a: []int64{1}, b: []int64{1}, want: []int64{1}},
		{name: "different int64 slices", a: []int64{1}, b: []int64{2}, fails: "conflicting scalar values"},
		{name: "equal float slices", a: []float64{1.5}, b: []float64{1.5}, want: []float64{1.5}},
		{name: "different float slices", a: []float64{1.5}, b: []float64{2.5}, fails: "conflicting scalar values"},
		{name: "equal bool slices", a: []bool{true}, b: []bool{true}, want: []bool{true}},
		{name: "different bool slices", a: []bool{true}, b: []bool{false}, fails: "conflicting scalar values"},

		{name: "map slices merge", a: []z.Dyn{{"a": 1}, {"x": 1}}, b: []z.Dyn{{"b": 2}, {"x": 1}}, want: []z.Dyn{{"a": 1, "b": 2}, {"x": 1}}},
		{name: "map slices length mismatch", a: []z.Dyn{{}}, b: []z.Dyn{}, fails: "array lengths differ (1 != 0)"},
		{name: "map slice vs other type", a: []z.Dyn{{}}, b: []string{"a"}, fails: "conflicting values"},
		{name: "map slice element conflict path", a: z.Dyn{"l": []z.Dyn{{"a": 1}}}, b: z.Dyn{"l": []z.Dyn{{"a": 2}}}, fails: `path "l[0].a"`},

		{name: "equal nested slices (uncomparable)", a: [][]int{{1}, {2}}, b: [][]int{{1}, {2}}, want: [][]int{{1}, {2}}},
		{name: "different nested slices", a: [][]int{{1}}, b: [][]int{{2}}, fails: "conflicting scalar values"},
		{name: "equal typed maps (uncomparable)", a: map[string]int{"a": 1}, b: map[string]int{"a": 1}, want: map[string]int{"a": 1}},
		{name: "different typed maps", a: map[string]int{"a": 1}, b: map[string]int{"a": 2}, fails: "conflicting scalar values"},
		{name: "uncomparable types with same rendering differ", a: []int{1}, b: []int32{1}, fails: "conflicting scalar values"},
		{name: "equal struct values", a: struct{ V int }{1}, b: struct{ V int }{1}, want: struct{ V int }{1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch := z.Intersection(coverCConst(tc.a), coverCConst(tc.b))
			var (
				got any
				err error
			)
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("merge panicked: %v", r)
					}
				}()
				got, err = sch.Parse(1)
			}()
			if tc.fails == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("got %#v want %#v", got, tc.want)
				}
				return
			}
			is := coverCIssues(t, err)
			if len(is) != 1 || is[0].Code != z.Custom ||
				!strings.HasPrefix(is[0].Message, "unmergeable intersection: conflict at path ") ||
				!strings.Contains(is[0].Message, tc.fails) {
				t.Fatalf("want conflict containing %q, got %+v", tc.fails, is)
			}
		})
	}
}

// @spec REQ-COMB-004
func TestCoverCIntersectionRealArrays(t *testing.T) {
	obj := func(k string, sch z.AnySchema) *z.ObjectSchema {
		return z.Object(func(s *z.ObjectBuilder) { s.Field(k, sch) })
	}
	ints := z.Array(z.Int())
	inter := z.Intersection(obj("l", ints), obj("l", z.Array(z.Int())))

	got, err := inter.Parse(map[string]any{"l": []any{1, 2}})
	if err != nil || !reflect.DeepEqual(got, z.Dyn{"l": []int{1, 2}}) {
		t.Fatalf("%v %v", got, err)
	}

	objs := z.Array(coverCShape("a"))
	inter = z.Intersection(obj("l", objs), obj("l", z.Array(coverCShape("a"))))
	got, err = inter.Parse(map[string]any{"l": []any{map[string]any{"type": "a"}}})
	if err != nil || !reflect.DeepEqual(got, z.Dyn{"l": []z.Dyn{{"type": "a"}}}) {
		t.Fatalf("%v %v", got, err)
	}
}

// @spec REQ-COMB-004 REQ-CORE-017
func TestCoverCIntersectionCancelled(t *testing.T) {
	newCanceller := func(cancel context.CancelFunc) z.Schema[any] {
		return z.NewTestSchema[any](func(any, any) (any, bool) {
			cancel()
			return 1, true
		})
	}

	t.Run("cancelled before left parse", func(t *testing.T) {
		cctx, cancel := context.WithCancel(context.Background())
		// inner intersection cancels; outer right operand observes it at entry
		inner := z.Intersection(newCanceller(cancel), coverCConst(1))
		outer := z.Intersection(inner, z.Intersection(coverCConst(1), coverCConst(1)))
		_, err := outer.Parse(nil, cctx)
		if err == nil {
			t.Fatal("expected cancellation to fail the parse")
		}
		if is := coverCIssues(t, err); len(is) != 0 {
			t.Fatalf("cancellation yields no issues, got %+v", is)
		}
	})
	t.Run("right operand fails", func(t *testing.T) {
		_, err := z.Intersection(coverCConst(1), coverCFail()).Parse(1)
		if is := coverCIssues(t, err); len(is) != 1 || is[0].Message != "boom" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
}

// @spec REQ-COMB-008 REQ-CORE-010 REQ-PIPE-007 REQ-COMB-006
func TestCoverCIntersectionModifiers(t *testing.T) {
	left := z.Object(func(s *z.ObjectBuilder) { s.Field("a", z.Int()) })
	right := z.Object(func(s *z.ObjectBuilder) { s.Field("b", z.Int()) })
	in := z.Intersection(left, right)
	valid := map[string]any{"a": 1, "b": 2}

	if _, err := in.Optional().Parse(nil); err == nil {
		t.Fatal("Optional rejects null")
	}
	if out, err := in.Optional().Parse(valid); err != nil || !out.IsSome() {
		t.Fatalf("Optional: %v %v", out, err)
	}
	if out, err := in.Nullable().Parse(nil); err != nil || !out.IsNull() {
		t.Fatalf("Nullable: %v %v", out, err)
	}
	if _, err := in.Nullish().Parse(valid); err != nil {
		t.Fatalf("Nullish: %v", err)
	}
	if out, err := in.Default("d").Parse(nil); err != nil || out != "d" {
		t.Fatalf("Default: %v %v", out, err)
	}
	arr, err := in.Array().Parse([]any{valid})
	if err != nil || len(arr) != 1 {
		t.Fatalf("Array: %v %v", arr, err)
	}
	if m := in.Meta(func(m *z.MetaBuilder) { m.ID("coverC_inter_meta") }).GetMeta(); m.ID != "coverC_inter_meta" {
		t.Fatalf("meta: %+v", m)
	}
	if in.GetMeta().ID != "" {
		t.Fatal("Meta must not mutate the receiver")
	}
	_, err = in.Error("inter msg").Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "inter msg" {
		t.Fatalf("got %q", is[0].Message)
	}
	_, err = in.ErrorFunc(func(*z.Issue) string { return "inter fn" }).Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "inter fn" {
		t.Fatalf("got %q", is[0].Message)
	}

	js, err := z.ToJSONSchema(in)
	if err != nil || !strings.Contains(string(js), "allOf") {
		t.Fatalf("%s %v", js, err)
	}
}

// --- Lazy ---

// @spec REQ-COMB-005 REQ-COMB-007
func TestCoverCLazy(t *testing.T) {
	calls := 0
	lz := z.Lazy(func() z.Schema[z.Dyn] {
		calls++
		return z.Object(func(s *z.ObjectBuilder) { s.Field("n", z.Int()) }).
			Meta(func(m *z.MetaBuilder) { m.ID("coverC_lazy_target").Title("lazy target") })
	})

	if calls != 0 {
		t.Fatal("Lazy must not resolve at construction")
	}
	for i := 0; i < 3; i++ {
		got, err := lz.Parse(map[string]any{"n": i})
		if err != nil || got["n"] != i {
			t.Fatalf("%v %v", got, err)
		}
	}
	if _, err := lz.Parse(map[string]any{"n": "x"}); err == nil {
		t.Fatal("resolved schema validation must apply")
	}
	if m := lz.GetMeta(); m.ID != "coverC_lazy_target" || m.Title != "lazy target" {
		t.Fatalf("meta must come from the resolved schema: %+v", m)
	}
	if calls != 1 {
		t.Fatalf("resolver must be memoized, called %d times", calls)
	}

	js, err := z.ToJSONSchema(lz)
	if err != nil || !strings.Contains(string(js), `"n"`) {
		t.Fatalf("%s %v", js, err)
	}
}

// @spec REQ-COMB-005 REQ-CORE-017
func TestCoverCLazyCancelled(t *testing.T) {
	cctx, cancel := context.WithCancel(context.Background())
	canceller := z.NewTestSchema[any](func(any, any) (any, bool) {
		cancel()
		return 1, true
	})
	resolved := false
	lz := z.Lazy(func() z.Schema[any] {
		resolved = true
		return z.Any()
	})
	_, err := z.Intersection(canceller, lz).Parse(1, cctx)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if resolved {
		t.Fatal("cancelled parse must not resolve the lazy schema")
	}
}
