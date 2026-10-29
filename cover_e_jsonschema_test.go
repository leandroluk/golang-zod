package z_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-EXP-001 REQ-EXP-002 REQ-EXP-003 REQ-EXP-004 REQ-EXP-007
func TestCoverEToJSONSchemaGolden(t *testing.T) {
	if len(coverEJSCases) != len(coverEJSGolden) {
		t.Fatalf("cases (%d) and golden entries (%d) out of sync", len(coverEJSCases), len(coverEJSGolden))
	}
	for _, c := range coverEJSCases {
		t.Run(c.name, func(t *testing.T) {
			want, ok := coverEJSGolden[c.name]
			if !ok {
				t.Fatalf("no golden for %q", c.name)
			}
			got, err := z.ToJSONSchema(c.sch, c.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != want {
				t.Fatalf("output mismatch\n got: %s\nwant: %s", got, want)
			}
			if !json.Valid(got) {
				t.Fatalf("invalid JSON: %s", got)
			}
		})
	}
}

// @spec REQ-EXP-007
func TestCoverEToJSONSchemaDeterministic(t *testing.T) {
	s := coverEStrObj("a", "b", "c", "d", "e", "f", "g", "h", "i", "j")
	first, err := z.ToJSONSchema(s)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		next, _ := z.ToJSONSchema(s)
		if !bytes.Equal(first, next) {
			t.Fatalf("non-deterministic output:\n%s\n%s", first, next)
		}
	}
}

// @spec REQ-EXP-003
func TestCoverEToJSONSchemaRootKeepsSchemaKeyword(t *testing.T) {
	// A root that carries a Meta.ID is also registered in $defs; it must still get $schema, and the
	// definition itself must stay free of root-level keywords.
	sch := coverEStrObj("v").Meta(func(m *z.MetaBuilder) { m.ID("coverE_js_rootid") })
	out, err := z.ToJSONSchema(sch, z.WithTarget(z.Draft07))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["$schema"] != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("root lost $schema: %v", doc)
	}
	def := doc["definitions"].(map[string]any)["coverE_js_rootid"].(map[string]any)
	if _, has := def["$schema"]; has {
		t.Errorf("definition must not carry $schema: %v", def)
	}
	if _, has := def["definitions"]; has {
		t.Errorf("definition must not carry definitions: %v", def)
	}
}

// @spec REQ-EXP-004
func TestCoverEToJSONSchemaUnrepresentable(t *testing.T) {
	cases := []struct {
		name string
		sch  z.AnySchema
		want string
	}{
		{"bigint", z.BigInt(), `unrepresentable node kind`},
		{"date", z.Date(), `unrepresentable node kind`},
		{"map", z.Map(z.String(), z.String()), `unrepresentable node kind`},
		{"set", z.Set(z.String()), `unrepresentable node kind`},
		{"transform", z.Transform(z.String(), func(s string) (int, error) { return len(s), nil }), `unrepresentable node kind`},
		{"array element", z.Array(z.BigInt()), `at "[]"`},
		{"object field", z.Object(func(s *z.ObjectBuilder) { s.Field("n", z.BigInt()) }), `at ".n"`},
		{"catchall", z.Object(func(s *z.ObjectBuilder) {}).Catchall(z.BigInt()), `at ".*"`},
		{"union option", z.Union(z.String(), z.BigInt()), `at "|1"`},
		{"discriminated option", z.DiscriminatedUnion("k",
			z.Object(func(s *z.ObjectBuilder) { s.Field("k", z.Literal("a")); s.Field("v", z.BigInt()) })), `at "|0.v"`},
		{"intersection left", z.Intersection(z.BigInt(), z.String()), `at "&L"`},
		{"intersection right", z.Intersection(z.String(), z.BigInt()), `at "&R"`},
		{"nullable", z.BigInt().Nullable(), `unrepresentable node kind`},
		{"optional", z.BigInt().Optional(), `unrepresentable node kind`},
		{"default", z.BigInt().Default(nil), `unrepresentable node kind`},
		{"lazy", z.Lazy(func() z.Schema[*big.Int] { return z.BigInt() }), `unrepresentable node kind`},
		{"codec output", z.Codec(z.String(), z.BigInt(), func(string) (*big.Int, error) { return nil, nil }, func(*big.Int) (string, error) { return "", nil }), `unrepresentable node kind`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := z.ToJSONSchema(tc.sch)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "golang-zod: ") {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
			// UnrepresentableAny never errors.
			if _, err := z.ToJSONSchema(tc.sch, z.WithUnrepresentable(z.UnrepresentableAny)); err != nil {
				t.Fatalf("UnrepresentableAny must not error: %v", err)
			}
		})
	}
}

// @spec REQ-EXP-003
func TestCoverEToJSONSchemaCyclesThrow(t *testing.T) {
	var node *z.ObjectSchema
	node = z.Object(func(s *z.ObjectBuilder) {
		s.Field("next", z.Lazy(func() z.Schema[z.Dyn] { return node }))
	}).Meta(func(m *z.MetaBuilder) { m.ID("coverE_js_cycle") })

	_, err := z.ToJSONSchema(node, z.WithCycles(z.CyclesThrow))
	if err == nil || !strings.Contains(err.Error(), `cycle detected for schema "coverE_js_cycle"`) {
		t.Fatalf("expected cycle error, got %v", err)
	}
	out, err := z.ToJSONSchema(node, z.WithCycles(z.CyclesRef))
	if err != nil {
		t.Fatalf("CyclesRef must terminate: %v", err)
	}
	var doc map[string]any
	_ = json.Unmarshal(out, &doc)
	next := doc["properties"].(map[string]any)["next"]
	if !reflect.DeepEqual(next, map[string]any{"$ref": "#/$defs/coverE_js_cycle"}) {
		t.Fatalf("expected $ref for the cycle, got %v", next)
	}
}
