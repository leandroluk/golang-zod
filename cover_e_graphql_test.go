package z_test

import (
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-GQL-001 REQ-GQL-002 REQ-GQL-003
func TestCoverEGraphQLScalarsAndNullability(t *testing.T) {
	shared := coverEStrObj("x")
	user := z.Object(func(s *z.ObjectBuilder) {
		s.Field("s", z.String())
		s.Field("n", z.Float64())
		s.Field("b", z.Boolean())
		s.Field("d", z.Date())
		s.Field("big", z.BigInt())
		s.Field("raw", z.Bytes())
		s.Field("tags", z.Array(z.String()))
		s.Field("maybeTags", z.Array(z.String()).Optional())
		s.Field("maybe", z.String().Optional())
		s.Field("nul", z.Float64().Nullable())
		s.Field("nullish", z.Boolean().Nullish())
		s.Field("color", z.Enum("RED", "GREEN"))
		s.Field("nested", coverEStrObj("y"))
		s.Field("objs", z.Array(coverEStrObj("z")))
		s.Field("a", shared)
		s.Field("b2", shared)
		s.Field("rec", z.Record(z.String(), z.String()))
	}).Meta(func(m *z.MetaBuilder) { m.ID("CoverEUser") })

	want := strings.Join([]string{
		"type CoverEUser {\n" +
			"  s: String!\n" +
			"  n: Float!\n" +
			"  b: Boolean!\n" +
			"  d: DateTime!\n" +
			"  big: BigInt!\n" +
			"  raw: Bytes!\n" +
			"  tags: [String!]!\n" +
			"  maybeTags: [String!]\n" +
			"  maybe: String\n" +
			"  nul: Float\n" +
			"  nullish: Boolean\n" +
			"  color: CoverEUserColor!\n" +
			"  nested: CoverEUserNested!\n" +
			"  objs: [CoverEUserObjsItem!]!\n" +
			"  a: CoverEUserA!\n" +
			"  b2: CoverEUserA!\n" +
			"  rec: String!\n" +
			"}",
		"type CoverEUserA {\n  x: String!\n}",
		"enum CoverEUserColor {\n  RED\n  GREEN\n}",
		"type CoverEUserNested {\n  y: String!\n}",
		"type CoverEUserObjsItem {\n  z: String!\n}",
		"scalar BigInt",
		"scalar Bytes",
		"scalar DateTime",
	}, "\n\n")

	got, err := z.ToGraphQLSDL(user)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("SDL mismatch\n got:\n%s\nwant:\n%s", got, want)
	}

	// Output is deterministic.
	for i := 0; i < 20; i++ {
		again, _ := z.ToGraphQLSDL(user)
		if again != got {
			t.Fatalf("non-deterministic SDL:\n%s\nvs\n%s", got, again)
		}
	}
}

// @spec REQ-GQL-001 REQ-GQL-003
func TestCoverEGraphQLRootNaming(t *testing.T) {
	anon := func() *z.ObjectSchema { return coverEStrObj("a") }

	got, err := z.ToGraphQLSDL(anon())
	if err != nil || got != "type Query {\n  a: String!\n}" {
		t.Errorf("anonymous root = %q, %v", got, err)
	}
	got, err = z.ToGraphQLSDL(anon(), z.GraphQLInput())
	if err != nil || got != "input Input {\n  a: String!\n}" {
		t.Errorf("anonymous input root = %q, %v", got, err)
	}
	named := coverEStrObj("a").Meta(func(m *z.MetaBuilder) { m.ID("CoverENamedInput") })
	got, err = z.ToGraphQLSDL(named, z.GraphQLInput())
	if err != nil || got != "input CoverENamedInput {\n  a: String!\n}" {
		t.Errorf("named input root = %q, %v", got, err)
	}

	// Enum named by Meta.ID wins over the generated hint.
	role := z.Enum("ADMIN", "USER").Meta(func(m *z.MetaBuilder) { m.ID("CoverERole") })
	obj := z.Object(func(s *z.ObjectBuilder) { s.Field("role", role) })
	got, err = z.ToGraphQLSDL(obj)
	if err != nil || got != "enum CoverERole {\n  ADMIN\n  USER\n}\n\ntype Query {\n  role: CoverERole!\n}" {
		t.Errorf("enum by ID = %q, %v", got, err)
	}
	// A nested object named by Meta.ID wins too.
	inner := coverEStrObj("q").Meta(func(m *z.MetaBuilder) { m.ID("CoverEInner") })
	got, err = z.ToGraphQLSDL(z.Object(func(s *z.ObjectBuilder) { s.Field("in", inner) }))
	if err != nil || got != "type CoverEInner {\n  q: String!\n}\n\ntype Query {\n  in: CoverEInner!\n}" {
		t.Errorf("object by ID = %q, %v", got, err)
	}
}

// @spec REQ-GQL-001 REQ-GQL-002
func TestCoverEGraphQLNonObjectRoots(t *testing.T) {
	cases := []struct {
		name string
		sch  z.AnySchema
		want string
	}{
		{"string root has no types", z.String(), ""},
		{"optional root", z.String().Optional(), ""},
		{"nil node", &z.BaseSchema[any]{}, ""},
		{"date root registers scalar", z.Date(), "scalar DateTime"},
		{"enum root", z.Enum("A", "B"), "enum Query {\n  A\n  B\n}"},
		{"array of dates", z.Array(z.Date()), "scalar DateTime"},
		{"lazy root", z.Lazy(func() z.Schema[z.Dyn] { return coverEStrObj("a") }), "type Query {\n  a: String!\n}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := z.ToGraphQLSDL(tc.sch)
			if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

// @spec REQ-GQL-004
func TestCoverEGraphQLRecursion(t *testing.T) {
	var tree *z.ObjectSchema
	tree = z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String())
		s.Field("children", z.Array(z.Lazy(func() z.Schema[z.Dyn] { return tree })))
		s.Field("parent", z.NewOptionalSchema(z.Lazy(func() z.Schema[z.Dyn] { return tree })))
	}).Meta(func(m *z.MetaBuilder) { m.ID("CoverETree") })

	got, err := z.ToGraphQLSDL(tree)
	want := "type CoverETree {\n  name: String!\n  children: [CoverETree!]!\n  parent: CoverETree\n}"
	if err != nil || got != want {
		t.Fatalf("got %q, %v; want %q", got, err, want)
	}
}

// @spec REQ-GQL-005
func TestCoverEGraphQLTupleErrors(t *testing.T) {
	tuple := func() z.AnySchema { return z.Tuple(z.String(), z.Float64()) }
	cases := map[string]z.AnySchema{
		"root":     tuple(),
		"field":    z.Object(func(s *z.ObjectBuilder) { s.Field("t", tuple()) }),
		"array":    z.Array(z.Tuple(z.String())),
		"optional": z.Tuple(z.String()).Optional(),
	}
	for name, sch := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := z.ToGraphQLSDL(sch)
			if err == nil || err.Error() != "golang-zod: heterogeneous Tuple has no GraphQL equivalent" || got != "" {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}
