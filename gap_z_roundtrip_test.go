package z_test

import (
	"regexp"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type gapZFixture struct {
	name   string
	sch    z.AnySchema
	parse  func(any) error
	accept []string
	reject []string
}

func gapZFix[T any](name string, s z.Schema[T], accept, reject []string) gapZFixture {
	return gapZFixture{name: name, sch: s, accept: accept, reject: reject, parse: func(v any) error {
		_, err := s.Parse(v)
		return err
	}}
}

// @spec REQ-EXP-005 REQ-EXP-006
func TestGapZJSONSchemaRoundTripCorpus(t *testing.T) {
	corpus := []gapZFixture{
		gapZFix("string", z.String(), []string{`"a"`, `""`}, []string{`1`, `null`, `true`}),
		gapZFix("number", z.Float64(), []string{`1`, `2.5`}, []string{`"2"`, `null`}),
		gapZFix("boolean", z.Boolean(), []string{`true`, `false`}, []string{`"true"`, `1`}),
		gapZFix("literal", z.Literal("a"), []string{`"a"`}, []string{`"b"`, `1`}),
		gapZFix("enum", z.Enum("A", "B"), []string{`"A"`, `"B"`}, []string{`"C"`, `1`}),
		gapZFix("array of strings", z.Array(z.String()), []string{`[]`, `["a","b"]`}, []string{`[1]`, `"a"`}),
		gapZFix("union", z.Union(z.String(), z.Boolean()), []string{`"a"`, `true`}, []string{`1`, `null`}),
		gapZFix("nullable", z.String().Nullable(), []string{`"a"`, `null`}, []string{`1`}),
		gapZFix("strict object", z.StrictObject(func(s *z.Shape) {
			s.Field("name", z.String())
			s.Field("role", z.Enum("ADMIN", "USER"))
			s.Field("tags", z.Array(z.String()).Optional())
		}),
			[]string{`{"name":"a","role":"ADMIN"}`, `{"name":"a","role":"USER","tags":["x"]}`},
			[]string{`{"name":"a"}`, `{"name":"a","role":"ROOT"}`, `{"name":"a","role":"USER","x":1}`, `[]`}),
		gapZFix("nested objects", z.Object(func(s *z.Shape) {
			s.Field("user", z.Object(func(u *z.Shape) {
				u.Field("email", z.Email())
				u.Field("age", z.Float64().GTE(0))
			}))
		}),
			[]string{`{"user":{"email":"a@b.co","age":3}}`},
			[]string{`{"user":{"email":"a@b.co","age":"3"}}`, `{"user":{"age":3}}`, `{}`}),
		gapZFix("intersection", z.Intersection(
			z.LooseObject(func(s *z.Shape) { s.Field("a", z.String()) }),
			z.LooseObject(func(s *z.Shape) { s.Field("b", z.Float64()) })),
			[]string{`{"a":"x","b":1}`}, []string{`{"a":"x"}`, `{"b":1}`}),
	}
	if len(corpus) < 10 {
		t.Fatalf("round-trip corpus must hold at least 10 schemas, has %d", len(corpus))
	}

	for _, f := range corpus {
		t.Run(f.name, func(t *testing.T) {
			doc, err := z.ToJSONSchema(f.sch)
			if err != nil {
				t.Fatal(err)
			}
			gapZRoundTrip(t, f, doc)
		})
	}
}

func gapZRoundTrip(t *testing.T, f gapZFixture, doc []byte) {
	t.Helper()
	back, err := z.FromJSONSchema(doc)
	if err != nil {
		t.Fatalf("FromJSONSchema(%s): %v", doc, err)
	}
	for _, in := range f.accept {
		v := gapZJSON(t, in)
		if err := f.parse(v); err != nil {
			t.Fatalf("fixture %s must be accepted by the original schema: %v", in, err)
		}
		if _, err := back.Parse(v); err != nil {
			t.Errorf("imported schema must accept %s (document %s): %v", in, doc, err)
		}
	}
	for _, in := range f.reject {
		v := gapZJSON(t, in)
		if err := f.parse(v); err == nil {
			t.Fatalf("fixture %s must be rejected by the original schema", in)
		}
		if _, err := back.Parse(v); err == nil {
			t.Errorf("imported schema must reject %s (document %s)", in, doc)
		}
	}
}

// The exporter does not render every check yet, so constrained and compound schemas are paired with
// a hand-written JSON Schema document that must behave exactly like the builder-made schema.
//
// @spec REQ-EXP-005 REQ-EXP-006
func TestGapZHandWrittenDocumentsMatchBuilders(t *testing.T) {
	corpus := []struct {
		fixture gapZFixture
		doc     string
	}{
		{gapZFix("string length", z.String().Min(2).Max(4), []string{`"ab"`, `"abcd"`}, []string{`"a"`, `"abcde"`, `1`}),
			`{"type":"string","minLength":2,"maxLength":4}`},
		{gapZFix("string pattern", z.String().Regex(regexp.MustCompile(`^[a-z]+$`)), []string{`"abc"`}, []string{`"ABC"`, `""`, `1`}),
			`{"type":"string","pattern":"^[a-z]+$"}`},
		{gapZFix("email", z.Email(), []string{`"a@b.co"`}, []string{`"nope"`, `1`}),
			`{"type":"string","format":"email"}`},
		{gapZFix("uuid", z.UUID(), []string{`"123e4567-e89b-42d3-a456-426614174000"`}, []string{`"123"`, `1`}),
			`{"type":"string","format":"uuid"}`},
		{gapZFix("number bounds", z.Float64().GTE(1).LTE(3), []string{`1`, `2.5`, `3`}, []string{`0.5`, `3.5`, `"2"`}),
			`{"type":"number","minimum":1,"maximum":3}`},
		{gapZFix("number exclusive bounds", z.Float64().GT(0).LT(1), []string{`0.5`}, []string{`0`, `1`, `"x"`}),
			`{"type":"number","exclusiveMinimum":0,"exclusiveMaximum":1}`},
		{gapZFix("int", z.Int().GTE(0).MultipleOf(2), []string{`0`, `4`}, []string{`-2`, `3`, `1.5`, `"2"`}),
			`{"type":"integer","minimum":0,"multipleOf":2}`},
		{gapZFix("array size", z.Array(z.Float64()).Min(1).Max(2), []string{`[1]`, `[1,2]`}, []string{`[]`, `[1,2,3]`}),
			`{"type":"array","items":{"type":"number"},"minItems":1,"maxItems":2}`},
		{gapZFix("tuple", z.Tuple(z.String(), z.Float64()), []string{`["a",1]`}, []string{`["a"]`, `["a",1,2]`, `[1,"a"]`}),
			`{"type":"array","prefixItems":[{"type":"string"},{"type":"number"}],"minItems":2,"items":false}`},
		{gapZFix("tuple with rest", z.Tuple(z.String()).Rest(z.Float64()), []string{`["a"]`, `["a",1,2]`}, []string{`[]`, `["a","b"]`}),
			`{"type":"array","prefixItems":[{"type":"string"}],"minItems":1,"items":{"type":"number"}}`},
		{gapZFix("record", z.Record(z.String(), z.Float64()), []string{`{}`, `{"a":1}`}, []string{`{"a":"x"}`, `[]`}),
			`{"type":"object","additionalProperties":{"type":"number"}}`},
		{gapZFix("nullable number", z.Float64().Nullable(), []string{`1`, `null`}, []string{`"x"`}),
			`{"type":["number","null"]}`},
		{gapZFix("union of literals", z.Union(z.Literal("a"), z.Literal("b")), []string{`"a"`, `"b"`}, []string{`"c"`, `1`}),
			`{"anyOf":[{"const":"a"},{"const":"b"}]}`},
		{gapZFix("recursive", recursiveTree(), []string{`{"v":1}`, `{"v":1,"kids":[{"v":2}]}`}, []string{`{"v":1,"kids":[{"v":"x"}]}`, `{}`}),
			`{"$ref":"#/$defs/node","$defs":{"node":{"type":"object","properties":{"v":{"type":"number"},"kids":{"type":"array","items":{"$ref":"#/$defs/node"}}},"required":["v"]}}}`},
	}
	for _, c := range corpus {
		t.Run(c.fixture.name, func(t *testing.T) { gapZRoundTrip(t, c.fixture, []byte(c.doc)) })
	}
}

func recursiveTree() z.Schema[z.Dyn] {
	var node z.Schema[z.Dyn]
	node = z.Object(func(s *z.Shape) {
		s.Field("v", z.Float64())
		s.Field("kids", z.Array(z.Lazy(func() z.Schema[z.Dyn] { return node })).Optional())
	})
	return node
}
