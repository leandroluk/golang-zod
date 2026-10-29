package z_test

import (
	"fmt"

	z "github.com/leandroluk/golang-zod"
)

type coverEJSCase struct {
	name string
	sch  z.AnySchema
	opts []z.JSONSchemaOption
}

func coverEStrObj(fields ...string) *z.ObjectSchema {
	return z.Object(func(s *z.ObjectBuilder) {
		for _, f := range fields {
			s.Field(f, z.String())
		}
	})
}

// coverEJSCases builds every exporter scenario. Schemas with Meta.ID are created once per process.
var coverEJSCases = func() []coverEJSCase {
	var cs []coverEJSCase
	add := func(name string, sch z.AnySchema, opts ...z.JSONSchemaOption) {
		cs = append(cs, coverEJSCase{name, sch, opts})
	}

	// primitives and metadata
	add("string", z.String())
	add("number", z.Float64())
	add("boolean", z.Boolean())
	add("bytes", z.Bytes())
	add("meta", z.String().Meta(func(m *z.MetaBuilder) {
		m.Title("T").Description("D").Deprecated(true).ReadOnly(true).Examples("a", "b")
	}))
	add("literal single", z.Literal("a"))
	add("literal multiple", z.Literal(1, 2, 3))
	add("enum", z.Enum("A", "B"))
	add("array", z.Array(z.Int()))
	add("any nil node", &z.BaseSchema[any]{})
	add("record string to string", z.Record(z.String(), z.String()))

	// objects
	obj := func() *z.ObjectSchema {
		return z.Object(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
			s.Field("age", z.Int().Optional())
			s.Field("nick", z.String().Nullish())
		})
	}
	add("object strip output", obj())
	add("object strip input", obj(), z.WithIO(z.Input))
	add("object strict", obj().Strict())
	add("object strict input", obj().Strict(), z.WithIO(z.Input))
	add("object loose", obj().Loose())
	add("object catchall", obj().Catchall(z.Int()))
	add("object empty", z.Object(func(s *z.ObjectBuilder) {}))
	add("object nested", z.Object(func(s *z.ObjectBuilder) {
		s.Field("child", coverEStrObj("x"))
	}))

	// combinators
	add("union", z.Union(z.String(), z.Float64()))
	add("xor", z.XOR(z.String(), z.Float64()))
	disc := func() *z.UnionSchema {
		return z.DiscriminatedUnion("kind",
			z.Object(func(s *z.ObjectBuilder) { s.Field("kind", z.Literal("a")) }),
			z.Object(func(s *z.ObjectBuilder) { s.Field("kind", z.Literal("b")) }),
		)
	}
	add("discriminated draft", disc())
	add("discriminated openapi30", disc(), z.WithTarget(z.OpenAPI30))
	add("discriminated openapi31", disc(), z.WithTarget(z.OpenAPI31))
	add("intersection", z.Intersection(coverEStrObj("a"), coverEStrObj("b")))

	// wrappers
	add("optional root", z.String().Optional())
	add("nullable string draft", z.String().Nullable())
	add("nullable string openapi30", z.String().Nullable(), z.WithTarget(z.OpenAPI30))
	add("nullable string openapi31", z.String().Nullable(), z.WithTarget(z.OpenAPI31))
	add("nullable literal anyOf", z.Literal("a").Nullable())
	add("nullish string", z.String().Nullish())
	add("default", z.String().Default("x"))
	add("lazy", z.Lazy(func() z.Schema[string] { return z.String() }))
	add("codec output", z.StringBool())
	add("codec input", z.StringBool(), z.WithIO(z.Input))

	// targets
	add("draft07", z.String(), z.WithTarget(z.Draft07))
	add("draft04", z.String(), z.WithTarget(z.Draft04))
	add("openapi30", z.String(), z.WithTarget(z.OpenAPI30))
	add("openapi31", z.String(), z.WithTarget(z.OpenAPI31))

	// unrepresentable
	add("bigint any", z.BigInt(), z.WithUnrepresentable(z.UnrepresentableAny))
	add("date any", z.Date(), z.WithUnrepresentable(z.UnrepresentableAny))

	// ids, defs and refs
	idObj := func(id string) *z.ObjectSchema {
		return coverEStrObj("v").Meta(func(m *z.MetaBuilder) { m.ID(id) })
	}
	add("id draft202012", idObj("coverE_js_a"))
	add("id draft07", idObj("coverE_js_b"), z.WithTarget(z.Draft07))
	add("id draft04", idObj("coverE_js_c"), z.WithTarget(z.Draft04))
	add("id openapi30", idObj("coverE_js_d"), z.WithTarget(z.OpenAPI30))
	add("id openapi31", idObj("coverE_js_e"), z.WithTarget(z.OpenAPI31))
	add("id reused ref", z.Array(idObj("coverE_js_f")), z.WithReused(z.ReusedRef))
	add("id reused ref openapi30", z.Array(idObj("coverE_js_g")), z.WithReused(z.ReusedRef), z.WithTarget(z.OpenAPI30))
	add("id reused ref draft07", z.Array(idObj("coverE_js_h")), z.WithReused(z.ReusedRef), z.WithTarget(z.Draft07))
	add("id reused ref uri", z.Array(idObj("coverE_js_i")), z.WithReused(z.ReusedRef),
		z.WithURI(func(id string) string { return fmt.Sprintf("https://example.com/schemas/%s.json", id) }))

	// recursion
	var tree *z.ObjectSchema
	tree = z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String())
		s.Field("children", z.Array(z.Lazy(func() z.Schema[z.Dyn] { return tree })))
	}).Meta(func(m *z.MetaBuilder) { m.ID("coverE_js_tree") })
	add("recursive ref", tree)
	add("recursive ref openapi30", tree, z.WithTarget(z.OpenAPI30))
	add("recursive ref draft07", tree, z.WithTarget(z.Draft07), z.WithCycles(z.CyclesRef))
	return cs
}()
