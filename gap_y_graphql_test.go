package z_test

import (
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func gapYID(id string, o *z.ObjectSchema) *z.ObjectSchema {
	return o.Meta(func(m *z.MetaBuilder) { m.ID(id) })
}

// @spec REQ-GQL-005
func TestGapYGraphQLIntersectionErrors(t *testing.T) {
	inter := z.Intersection(coverEStrObj("a"), coverEStrObj("b"))
	for name, sch := range map[string]z.AnySchema{
		"root":  inter,
		"field": z.Object(func(s *z.ObjectBuilder) { s.Field("i", inter) }),
	} {
		got, err := z.ToGraphQLSDL(sch)
		if err == nil || err.Error() != "golang-zod: Intersection has no GraphQL equivalent" || got != "" {
			t.Fatalf("%s: got %q, %v", name, got, err)
		}
	}
}

// @spec REQ-GQL-005
func TestGapYGraphQLUnionErrors(t *testing.T) {
	_, err := z.ToGraphQLSDL(z.Union(coverEStrObj("a"), z.String()))
	if err == nil || err.Error() != "golang-zod: Union with non-object member has no GraphQL equivalent" {
		t.Fatalf("non-object union: %v", err)
	}
	_, err = z.ToGraphQLSDL(z.Union(z.Tuple(z.String())))
	if err == nil || err.Error() != "golang-zod: Union with non-object member has no GraphQL equivalent" {
		t.Fatalf("tuple member: %v", err)
	}
	_, err = z.ToGraphQLSDL(z.Union(coverEStrObj("a")), z.GraphQLInput())
	if err == nil || err.Error() != "golang-zod: Union has no GraphQL input equivalent" {
		t.Fatalf("input union: %v", err)
	}
	bad := z.Object(func(s *z.ObjectBuilder) { s.Field("t", z.Tuple(z.String())) })
	_, err = z.ToGraphQLSDL(z.Union(bad))
	if err == nil || err.Error() != "golang-zod: heterogeneous Tuple has no GraphQL equivalent" {
		t.Fatalf("nested tuple error: %v", err)
	}
}

// @spec REQ-GQL-001
func TestGapYGraphQLUnionSDL(t *testing.T) {
	cat := gapYID("gapY_Cat", z.Object(func(s *z.ObjectBuilder) { s.Field("kind", z.Literal("cat")) }))
	dog := gapYID("gapY_Dog", z.Object(func(s *z.ObjectBuilder) { s.Field("kind", z.Literal("dog")) }))
	pet := z.DiscriminatedUnion("kind", cat, dog).Meta(func(m *z.MetaBuilder) { m.ID("gapY_Pet") })
	root := gapYID("gapY_Owner", z.Object(func(s *z.ObjectBuilder) {
		s.Field("pet", pet)
		s.Field("again", pet)
	}))
	got, err := z.ToGraphQLSDL(root)
	want := "type gapY_Cat {\n  kind: String!\n}\n\ntype gapY_Dog {\n  kind: String!\n}\n\ntype gapY_Owner {\n  pet: gapY_Pet!\n  again: gapY_Pet!\n}\n\nunion gapY_Pet = gapY_Cat | gapY_Dog"
	if err != nil || got != want {
		t.Fatalf("got %q, %v\nwant %q", got, err, want)
	}

	got, err = z.ToGraphQLSDL(z.Union(coverEStrObj("a"), coverEStrObj("b")))
	want = "union Query = QueryOption1 | QueryOption2\n\ntype QueryOption1 {\n  a: String!\n}\n\ntype QueryOption2 {\n  b: String!\n}"
	if err != nil || got != want {
		t.Fatalf("got %q, %v\nwant %q", got, err, want)
	}
}

// @spec REQ-GQL-003
func TestGapYGraphQLFallbackNaming(t *testing.T) {
	user := z.Object(func(s *z.ObjectBuilder) {
		s.Field("address", coverEStrObj("street"))
		s.Field("home_town", coverEStrObj("zip"))
		s.Field("tags", z.Array(coverEStrObj("n")))
		s.Field("role", z.Enum("A", "B"))
	}).Meta(func(m *z.MetaBuilder) { m.ID("gapY_User") })
	got, err := z.ToGraphQLSDL(user)
	want := strings.Join([]string{
		"type gapY_User {\n  address: gapY_UserAddress!\n  home_town: gapY_UserHomeTown!\n  tags: [gapY_UserTagsItem!]!\n  role: gapY_UserRole!\n}",
		"type gapY_UserAddress {\n  street: String!\n}",
		"type gapY_UserHomeTown {\n  zip: String!\n}",
		"enum gapY_UserRole {\n  A\n  B\n}",
		"type gapY_UserTagsItem {\n  n: String!\n}",
	}, "\n\n")
	if err != nil || got != want {
		t.Fatalf("got %q, %v\nwant %q", got, err, want)
	}
}

// @spec REQ-GQL-003
func TestGapYGraphQLNameCollisions(t *testing.T) {
	color := z.Enum("R", "G")
	root := z.Object(func(s *z.ObjectBuilder) {
		s.Field("a_b", coverEStrObj("x"))
		s.Field("aB", coverEStrObj("y"))
		s.Field("other", gapYID("gapY_RootAB", coverEStrObj("p")))
		s.Field("c1", color)
		s.Field("c2", color)
		s.Field("dt", z.Date())
	}).Meta(func(m *z.MetaBuilder) { m.ID("gapY_Root") })
	got, err := z.ToGraphQLSDL(root)
	want := strings.Join([]string{
		"type gapY_Root {\n  a_b: gapY_RootAB!\n  aB: gapY_RootAB2!\n  other: gapY_RootAB3!\n  c1: gapY_RootC1!\n  c2: gapY_RootC1!\n  dt: DateTime!\n}",
		"type gapY_RootAB {\n  x: String!\n}",
		"type gapY_RootAB2 {\n  y: String!\n}",
		"type gapY_RootAB3 {\n  p: String!\n}",
		"enum gapY_RootC1 {\n  R\n  G\n}",
		"scalar DateTime",
	}, "\n\n")
	if err != nil || got != want {
		t.Fatalf("got:\n%s\nwant:\n%s\nerr %v", got, want, err)
	}

	// A root named like a built-in scalar gets a suffix.
	got, err = z.ToGraphQLSDL(gapYID("String", coverEStrObj("a")))
	if err != nil || got != "type String2 {\n  a: String!\n}" {
		t.Fatalf("got %q, %v", got, err)
	}

	// A suffixed candidate that is already taken is skipped.
	skip := z.Object(func(s *z.ObjectBuilder) {
		s.Field("a", coverEStrObj("x"))
		s.Field("b", gapYID("QueryA", coverEStrObj("y")))
		s.Field("c", gapYID("QueryA2", coverEStrObj("z")))
	})
	got, err = z.ToGraphQLSDL(skip)
	want = "type Query {\n  a: QueryA!\n  b: QueryA2!\n  c: QueryA22!\n}\n\ntype QueryA {\n  x: String!\n}\n\ntype QueryA2 {\n  y: String!\n}\n\ntype QueryA22 {\n  z: String!\n}"
	if err != nil || got != want {
		t.Fatalf("got %q, %v\nwant %q", got, err, want)
	}
}

// @spec REQ-GQL-002
func TestGapYGraphQLDefaultAndStructFields(t *testing.T) {
	root := z.Object(func(s *z.ObjectBuilder) {
		s.Field("d", z.String().Default("x"))
	})
	got, err := z.ToGraphQLSDL(root)
	if err != nil || got != "type Query {\n  d: String!\n}" {
		t.Fatalf("got %q, %v", got, err)
	}
	type gapYS struct {
		N string `json:"n"`
	}
	st := z.Struct[gapYS](func(s *z.StructBuilder, v *gapYS) { s.Field(&v.N, z.String()) })
	got, err = z.ToGraphQLSDL(z.Union(st))
	if err != nil || got != "union Query = QueryOption1\n\ntype QueryOption1 {\n  n: String!\n}" {
		t.Fatalf("got %q, %v", got, err)
	}
}
