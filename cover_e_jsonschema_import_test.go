package z_test

import (
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type coverEImportCase struct {
	name   string
	schema string
	accept []any
	reject []any
}

// @spec REQ-EXP-005
func TestCoverEFromJSONSchema(t *testing.T) {
	obj := map[string]any{}
	cases := []coverEImportCase{
		{"const string", `{"const":"a"}`, []any{"a"}, []any{"b", 1.0, nil}},
		{"const number", `{"const":1}`, []any{1.0}, []any{2.0, "1"}},
		{"enum of strings", `{"enum":["A","B"]}`, []any{"A", "B"}, []any{"C", 1.0}},
		{"enum with non-strings", `{"enum":[1,"a"]}`, []any{1.0, "a"}, []any{"z", nil, obj, 2.0}},
		{"oneOf", `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`, []any{"x", true}, []any{1.0, nil}},
		{"allOf of two objects", `{"allOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},{"type":"object","properties":{"b":{"type":"number"}},"required":["b"]}]}`,
			[]any{map[string]any{"a": "x", "b": 1.0}}, []any{map[string]any{"a": "x"}, map[string]any{"b": 1.0}, "x"}},
		{"allOf with one member", `{"allOf":[{"type":"string"}]}`, []any{"x"}, []any{1.0}},
		{"no type", `{}`, []any{1.0, "x", nil, obj}, nil},
		{"string", `{"type":"string"}`, []any{"x"}, []any{1.0, nil}},
		{"integer", `{"type":"integer"}`, []any{5.0, 5}, []any{"5", nil}},
		{"number", `{"type":"number"}`, []any{1.5, 2.0}, []any{"1"}},
		{"boolean", `{"type":"boolean"}`, []any{true, false}, []any{"true", 1.0}},
		{"array of items", `{"type":"array","items":{"type":"string"}}`, []any{[]any{"a", "b"}, []any{}}, []any{[]any{1.0}, "a"}},
		{"array without items", `{"type":"array"}`, []any{[]any{1.0, "a"}, []any{}}, []any{"a", 1.0}},
		{"array of enum items", `{"type":"array","items":{"enum":["x","y"]}}`, []any{[]any{"x"}}, []any{[]any{"z"}}},
		{"nullable type list picks first non-null", `{"type":["null","string"]}`, []any{"x"}, []any{1.0}},
		{"type list only null", `{"type":["null"]}`, []any{nil}, []any{1.0, "x"}},
		{"null type", `{"type":"null"}`, []any{nil}, []any{1.0, "x"}},
		{"object required and optional", `{"type":"object","properties":{"name":{"type":"string"},"age":{"type":"number"}},"required":["name"]}`,
			[]any{map[string]any{"name": "a"}, map[string]any{"name": "a", "age": 3.0}, map[string]any{"name": "a", "extra": 1.0}},
			[]any{map[string]any{}, map[string]any{"name": 1.0}, map[string]any{"name": "a", "age": "x"}, "a"}},
		{"object additionalProperties false is strict", `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`,
			[]any{map[string]any{"a": "x"}}, []any{map[string]any{"a": "x", "b": 1.0}}},
		{"object additionalProperties true is not strict", `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":true}`,
			[]any{map[string]any{"a": "x", "b": 1.0}}, []any{map[string]any{}}},
		{"object without properties", `{"type":"object"}`, []any{map[string]any{}, map[string]any{"a": 1.0}}, []any{"x"}},
		{"nested object", `{"type":"object","properties":{"o":{"type":"object","properties":{"n":{"type":"integer"}},"required":["n"]}},"required":["o"]}`,
			[]any{map[string]any{"o": map[string]any{"n": 1.0}}}, []any{map[string]any{"o": map[string]any{"n": "x"}}, map[string]any{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch, err := z.FromJSONSchema([]byte(tc.schema))
			if err != nil {
				t.Fatal(err)
			}
			for _, in := range tc.accept {
				if _, err := sch.Parse(in); err != nil {
					t.Errorf("should accept %#v: %v", in, err)
				}
			}
			for _, in := range tc.reject {
				if _, err := sch.Parse(in); err == nil {
					t.Errorf("should reject %#v", in)
				}
			}
		})
	}
}

// @spec REQ-EXP-005
func TestCoverEFromJSONSchemaInvalidJSON(t *testing.T) {
	for _, in := range []string{``, `{`, `null1`} {
		sch, err := z.FromJSONSchema([]byte(in))
		if err == nil || !strings.HasPrefix(err.Error(), "golang-zod: failed to unmarshal JSON Schema: ") || sch != nil {
			t.Errorf("FromJSONSchema(%q) = %v, %v", in, sch, err)
		}
	}
}

// @spec REQ-EXP-006
func TestCoverEJSONSchemaRoundTrip(t *testing.T) {
	type fixture struct {
		name   string
		sch    z.AnySchema
		accept []any
		reject []any
	}
	fixtures := []fixture{
		{"string", z.String(), []any{"a"}, []any{1.0}},
		{"number", z.Float64(), []any{1.5}, []any{"a"}},
		{"boolean", z.Boolean(), []any{true}, []any{"a"}},
		{"literal", z.Literal("a"), []any{"a"}, []any{"b"}},
		{"enum", z.Enum("A", "B"), []any{"A", "B"}, []any{"C"}},
		{"array", z.Array(z.String()), []any{[]any{"a"}}, []any{[]any{1.0}, "a"}},
		{"union", z.Union(z.String(), z.Boolean()), []any{"a", true}, []any{1.0}},
		{"nullable", z.String().Nullable(), []any{"a"}, []any{1.0}},
		{"object", z.StrictObject(func(s *z.Shape) {
			s.Field("name", z.String())
			s.Field("role", z.Enum("ADMIN", "USER"))
			s.Field("tags", z.Array(z.String()).Optional())
		}),
			[]any{map[string]any{"name": "a", "role": "ADMIN"}, map[string]any{"name": "a", "role": "USER", "tags": []any{"x"}}},
			[]any{map[string]any{"name": "a"}, map[string]any{"name": "a", "role": "ROOT"}, map[string]any{"name": "a", "role": "USER", "x": 1.0}}},
		{"intersection", z.Intersection(
			z.LooseObject(func(s *z.Shape) { s.Field("a", z.String()) }),
			z.LooseObject(func(s *z.Shape) { s.Field("b", z.Float64()) })),
			[]any{map[string]any{"a": "x", "b": 1.0}}, []any{map[string]any{"a": "x"}}},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			out, err := z.ToJSONSchema(f.sch)
			if err != nil {
				t.Fatal(err)
			}
			back, err := z.FromJSONSchema(out)
			if err != nil {
				t.Fatalf("FromJSONSchema(%s): %v", out, err)
			}
			for _, in := range f.accept {
				if _, err := back.Parse(in); err != nil {
					t.Errorf("round trip must accept %#v: %v (schema %s)", in, err, out)
				}
			}
			for _, in := range f.reject {
				if _, err := back.Parse(in); err == nil {
					t.Errorf("round trip must reject %#v (schema %s)", in, out)
				}
			}
		})
	}
}
