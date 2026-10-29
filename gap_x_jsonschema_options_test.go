package z_test

import (
	"fmt"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func gapXSharedObj() *z.ObjectSchema {
	return z.Object(func(s *z.ObjectBuilder) { s.Field("v", z.String()) })
}

const gapXObjDef = `{"additionalProperties":false,"properties":{"v":{"type":"string"}},"required":["v"],"type":"object"}`

func gapXTwice(shared z.AnySchema) *z.ObjectSchema {
	return z.Object(func(s *z.ObjectBuilder) {
		s.Field("a", shared)
		s.Field("b", shared)
	})
}

// @spec REQ-EXP-003
func TestGapXReusedRefMovesSharedSchemasToDefs(t *testing.T) {
	ref := gapXOpts(z.WithReused(z.ReusedRef))
	with := func(t z.JSONSchemaTarget) []z.JSONSchemaOption {
		return gapXOpts(z.WithReused(z.ReusedRef), z.WithTarget(t))
	}
	props := func(prefix string) string {
		return `"properties":{"a":{"$ref":"` + prefix + `__schema0"},"b":{"$ref":"` + prefix + `__schema0"}}`
	}
	tail := `,"required":["a","b"],"type":"object"}`
	gapXRun(t, []gapXCase{
		{"draft 2020-12", gapXTwice(gapXSharedObj()), ref,
			`{"$defs":{"__schema0":` + gapXObjDef + `},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
				`"additionalProperties":false,` + props("#/$defs/") + tail},
		{"draft07", gapXTwice(gapXSharedObj()), with(z.Draft07),
			`{"$schema":"http://json-schema.org/draft-07/schema#","additionalProperties":false,` +
				`"definitions":{"__schema0":` + gapXObjDef + `},` + props("#/definitions/") + tail},
		{"draft04", gapXTwice(gapXSharedObj()), with(z.Draft04),
			`{"$schema":"http://json-schema.org/draft-04/schema#","additionalProperties":false,` +
				`"definitions":{"__schema0":` + gapXObjDef + `},` + props("#/definitions/") + tail},
		{"openapi30", gapXTwice(gapXSharedObj()), with(z.OpenAPI30),
			`{"additionalProperties":false,"components":{"schemas":{"__schema0":` + gapXObjDef + `}},` +
				props("#/components/schemas/") + tail},
		{"openapi31", gapXTwice(gapXSharedObj()), with(z.OpenAPI31),
			`{"$defs":{"__schema0":` + gapXObjDef + `},"additionalProperties":false,` +
				props("#/components/schemas/") + tail},
		{"inline is the default", gapXTwice(gapXSharedObj()), nil,
			gapXD2020 + `"additionalProperties":false,"properties":{"a":` + gapXObjDef + `,"b":` + gapXObjDef +
				`},"required":["a","b"],"type":"object"}`},
		{"explicit inline", gapXTwice(gapXSharedObj()), gapXOpts(z.WithReused(z.ReusedInline)),
			gapXD2020 + `"additionalProperties":false,"properties":{"a":` + gapXObjDef + `,"b":` + gapXObjDef +
				`},"required":["a","b"],"type":"object"}`},
		{"used once stays inline", z.Object(func(s *z.ObjectBuilder) { s.Field("a", gapXSharedObj()) }), ref,
			gapXD2020 + `"additionalProperties":false,"properties":{"a":` + gapXObjDef + `},"required":["a"],"type":"object"}`},
		{"URI maps generated names", gapXTwice(gapXSharedObj()),
			gapXOpts(z.WithReused(z.ReusedRef), z.WithURI(func(id string) string { return "https://x.test/" + id })),
			`{"$defs":{"__schema0":` + gapXObjDef + `},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
				`"additionalProperties":false,"properties":{"a":{"$ref":"https://x.test/__schema0"},` +
				`"b":{"$ref":"https://x.test/__schema0"}},"required":["a","b"],"type":"object"}`},
	})
}

// @spec REQ-EXP-003
func TestGapXReusedRefNamingAndNesting(t *testing.T) {
	str := z.String()
	num := z.Int()
	three := z.Object(func(s *z.ObjectBuilder) {
		s.Field("a", str)
		s.Field("b", num)
		s.Field("c", str)
		s.Field("d", num)
		s.Field("e", str)
	})
	gapXRun(t, []gapXCase{
		{"leaf schemas reused three times and twice", three, gapXOpts(z.WithReused(z.ReusedRef)),
			`{"$defs":{"__schema0":{"type":"string"},"__schema1":{"type":"integer"}},` +
				`"$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,` +
				`"properties":{"a":{"$ref":"#/$defs/__schema0"},"b":{"$ref":"#/$defs/__schema1"},` +
				`"c":{"$ref":"#/$defs/__schema0"},"d":{"$ref":"#/$defs/__schema1"},"e":{"$ref":"#/$defs/__schema0"}},` +
				`"required":["a","b","c","d","e"],"type":"object"}`},
	})

	// a reused schema that itself contains a reused schema
	inner := z.String()
	outer := z.Object(func(s *z.ObjectBuilder) {
		s.Field("x", inner)
		s.Field("y", inner)
	})
	root := z.Object(func(s *z.ObjectBuilder) {
		s.Field("p", outer)
		s.Field("q", outer)
	})
	gapXRun(t, []gapXCase{
		{"nested reuse", root, gapXOpts(z.WithReused(z.ReusedRef)),
			`{"$defs":{"__schema0":{"additionalProperties":false,"properties":{"x":{"$ref":"#/$defs/__schema1"},` +
				`"y":{"$ref":"#/$defs/__schema1"}},"required":["x","y"],"type":"object"},"__schema1":{"type":"string"}},` +
				`"$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,` +
				`"properties":{"p":{"$ref":"#/$defs/__schema0"},"q":{"$ref":"#/$defs/__schema0"}},` +
				`"required":["p","q"],"type":"object"}`},
	})
}

// @spec REQ-EXP-003
func TestGapXReusedRefThroughWrappers(t *testing.T) {
	shared := z.String()
	obj := func(a, b z.AnySchema) z.AnySchema {
		return z.Object(func(s *z.ObjectBuilder) {
			s.Field("a", a)
			s.Field("b", b)
		})
	}
	sharedObj := gapXSharedObj()
	ref := gapXOpts(z.WithReused(z.ReusedRef))
	defs := func(d string) string { return `"$defs":{"__schema0":` + d + `},` }
	gapXRun(t, []gapXCase{
		{"default over a reference", obj(shared.Default("x"), shared.Default("y")), ref,
			`{` + defs(`{"type":"string"}`) + `"$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,` +
				`"properties":{"a":{"$ref":"#/$defs/__schema0","default":"x"},"b":{"$ref":"#/$defs/__schema0","default":"y"}},` +
				`"required":["a","b"],"type":"object"}`},
		{"default over a reference draft07", obj(shared.Default("x"), shared.Default("y")),
			gapXOpts(z.WithReused(z.ReusedRef), z.WithTarget(z.Draft07)),
			`{"$schema":"http://json-schema.org/draft-07/schema#","additionalProperties":false,` +
				`"definitions":{"__schema0":{"type":"string"}},` +
				`"properties":{"a":{"allOf":[{"$ref":"#/definitions/__schema0"}],"default":"x"},` +
				`"b":{"allOf":[{"$ref":"#/definitions/__schema0"}],"default":"y"}},"required":["a","b"],"type":"object"}`},
		{"nullable over a reference", obj(sharedObj.Nullable(), sharedObj.Nullable()), ref,
			`{` + defs(gapXObjDef) + `"$schema":"https://json-schema.org/draft/2020-12/schema","additionalProperties":false,` +
				`"properties":{"a":{"anyOf":[{"$ref":"#/$defs/__schema0"},{"type":"null"}]},` +
				`"b":{"anyOf":[{"$ref":"#/$defs/__schema0"},{"type":"null"}]}},"required":["a","b"],"type":"object"}`},
		{"nullable over a reference openapi30", obj(sharedObj.Nullable(), sharedObj.Nullable()),
			gapXOpts(z.WithReused(z.ReusedRef), z.WithTarget(z.OpenAPI30)),
			`{"additionalProperties":false,"components":{"schemas":{"__schema0":` + gapXObjDef + `}},` +
				`"properties":{"a":{"allOf":[{"$ref":"#/components/schemas/__schema0"}],"nullable":true},` +
				`"b":{"allOf":[{"$ref":"#/components/schemas/__schema0"}],"nullable":true}},"required":["a","b"],"type":"object"}`},
		{"nullable over a reference openapi31", obj(sharedObj.Nullable(), sharedObj.Nullable()),
			gapXOpts(z.WithReused(z.ReusedRef), z.WithTarget(z.OpenAPI31)),
			`{"$defs":{"__schema0":` + gapXObjDef + `},"additionalProperties":false,` +
				`"properties":{"a":{"anyOf":[{"$ref":"#/components/schemas/__schema0"},{"type":"null"}]},` +
				`"b":{"anyOf":[{"$ref":"#/components/schemas/__schema0"},{"type":"null"}]}},"required":["a","b"],"type":"object"}`},
		{"pipe codec and optional wrappers are not extracted", obj(z.NewOptionalSchema[int](z.Pipe(z.String(), z.Int())), z.Int().Optional()), ref,
			gapXD2020 + `"additionalProperties":false,"properties":{"a":{"type":"integer"},"b":{"type":"integer"}},"type":"object"}`},
	})
}

// @spec REQ-EXP-003
func TestGapXReusedRefPrePassReportsErrors(t *testing.T) {
	bad := z.Date()
	if _, err := z.ToJSONSchema(gapXTwice(bad), z.WithReused(z.ReusedRef)); err == nil {
		t.Fatal("unrepresentable schema must fail in the pre-pass")
	}
	got, err := z.ToJSONSchema(gapXTwice(bad), z.WithReused(z.ReusedRef), z.WithUnrepresentable(z.UnrepresentableAny))
	want := gapXD2020 + `"additionalProperties":false,"properties":{"a":{},"b":{}},"required":["a","b"],"type":"object"}`
	if err != nil || string(got) != want {
		t.Fatalf("got %s %v", got, err)
	}
}

func gapXTree() z.Schema[z.Dyn] {
	var tree *z.ObjectSchema
	tree = z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String())
		s.Field("kids", z.Array(z.Lazy(func() z.Schema[z.Dyn] { return tree })))
	})
	return tree
}

// @spec REQ-EXP-003
func TestGapXRecursiveSchemaWithoutID(t *testing.T) {
	const body = `{"additionalProperties":false,"properties":{"kids":{"items":{"$ref":"#/$defs/__schema0"},"type":"array"},` +
		`"name":{"type":"string"}},"required":["kids","name"],"type":"object"}`
	gapXRun(t, []gapXCase{
		{"inline mode names the cycle", gapXTree(), nil,
			`{"$defs":{"__schema0":` + body + `},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
				`"additionalProperties":false,"properties":{"kids":{"items":{"$ref":"#/$defs/__schema0"},"type":"array"},` +
				`"name":{"type":"string"}},"required":["kids","name"],"type":"object"}`},
		{"ref mode returns a reference", gapXTree(), gapXOpts(z.WithReused(z.ReusedRef)),
			`{"$defs":{"__schema0":` + body + `},"$ref":"#/$defs/__schema0",` +
				`"$schema":"https://json-schema.org/draft/2020-12/schema"}`},
	})
	if _, err := z.ToJSONSchema(gapXTree(), z.WithCycles(z.CyclesThrow)); err == nil ||
		!strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("want cycle error, got %v", err)
	}
	if _, err := z.ToJSONSchema(gapXTree(), z.WithCycles(z.CyclesThrow), z.WithReused(z.ReusedRef)); err != nil {
		// In Ref mode the pre-pass stops at the first repeat and the cycle is reported by the real pass.
		if !strings.Contains(err.Error(), "cycle detected") {
			t.Fatalf("unexpected error: %v", err)
		}
	} else {
		t.Fatal("want cycle error in ref mode")
	}
}

// @spec REQ-EXP-003
func TestGapXLazyReusedWithoutCycle(t *testing.T) {
	lazy := z.Lazy(func() z.Schema[string] { return z.String() })
	root := z.Object(func(s *z.ObjectBuilder) {
		s.Field("a", lazy)
		s.Field("b", lazy)
	})
	gapXRun(t, []gapXCase{
		{"inline", root, nil,
			gapXD2020 + `"additionalProperties":false,"properties":{"a":{"type":"string"},"b":{"type":"string"}},"required":["a","b"],"type":"object"}`},
		{"ref", root, gapXOpts(z.WithReused(z.ReusedRef)),
			`{"$defs":{"__schema0":{"type":"string"}},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
				`"additionalProperties":false,"properties":{"a":{"$ref":"#/$defs/__schema0"},"b":{"$ref":"#/$defs/__schema0"}},` +
				`"required":["a","b"],"type":"object"}`},
	})
}

// @spec REQ-EXP-003
func TestGapXReusedRefKeepsIDSemantics(t *testing.T) {
	idObj := gapXSharedObj().Meta(func(m *z.MetaBuilder) { m.ID("gapX_reused_id") })
	got, err := z.ToJSONSchema(gapXTwice(idObj), z.WithReused(z.ReusedRef))
	want := `{"$defs":{"gapX_reused_id":` + gapXObjDef + `},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
		`"additionalProperties":false,"properties":{"a":{"$ref":"#/$defs/gapX_reused_id"},` +
		`"b":{"$ref":"#/$defs/gapX_reused_id"}},"required":["a","b"],"type":"object"}`
	if err != nil || string(got) != want {
		t.Fatalf("\n got: %s\nwant: %s\n err: %v", got, want, err)
	}
}

// @spec REQ-EXP-001
func TestGapXOverrideWith(t *testing.T) {
	var seen []string
	sch := z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", z.String())
		s.Field("tags", z.Array(z.String()).Optional())
		s.Field("n", z.Int().Default(1))
	})
	got, err := z.ToJSONSchema(sch, z.WithOverride(func(s z.AnySchema, path string, js map[string]any) {
		seen = append(seen, fmt.Sprintf("%q", path))
		js["x-path"] = path
	}))
	want := gapXD2020 + `"additionalProperties":false,"properties":{` +
		`"n":{"default":1,"type":"integer","x-path":".n"},` +
		`"name":{"type":"string","x-path":".name"},` +
		`"tags":{"items":{"type":"string","x-path":".tags[]"},"type":"array","x-path":".tags"}},` +
		`"required":["n","name"],"type":"object","x-path":""}`
	if err != nil || string(got) != want {
		t.Fatalf("\n got: %s\nwant: %s\n err: %v", got, want, err)
	}
	// called once per schema that builds its own JSON object (wrappers delegate), never during the pre-pass
	if len(seen) != 5 {
		t.Fatalf("override calls: %v", seen)
	}
	seen = nil
	if _, err := z.ToJSONSchema(sch, z.WithReused(z.ReusedRef), z.WithOverride(func(z.AnySchema, string, map[string]any) {
		seen = append(seen, "x")
	})); err != nil || len(seen) != 5 {
		t.Fatalf("override calls with Reused(Ref): %v %v", seen, err)
	}
}

// @spec REQ-EXP-001
func TestGapXOverrideAppliesToDefinitions(t *testing.T) {
	got, err := z.ToJSONSchema(gapXTwice(gapXSharedObj()), z.WithReused(z.ReusedRef),
		z.WithOverride(func(s z.AnySchema, path string, js map[string]any) {
			if _, isObject := js["properties"]; isObject {
				js["x-obj"] = true
			}
		}))
	want := `{"$defs":{"__schema0":{"additionalProperties":false,"properties":{"v":{"type":"string"}},` +
		`"required":["v"],"type":"object","x-obj":true}},"$schema":"https://json-schema.org/draft/2020-12/schema",` +
		`"additionalProperties":false,"properties":{"a":{"$ref":"#/$defs/__schema0"},"b":{"$ref":"#/$defs/__schema0"}},` +
		`"required":["a","b"],"type":"object","x-obj":true}`
	if err != nil || string(got) != want {
		t.Fatalf("\n got: %s\nwant: %s\n err: %v", got, want, err)
	}
}

// @spec REQ-EXP-002
func TestGapXMetadataOption(t *testing.T) {
	name := z.String().Meta(func(m *z.MetaBuilder) { m.Title("own").Description("own desc").Custom("x-own", 1) })
	plain := z.String()
	flags := z.Int()
	sch := z.Object(func(s *z.ObjectBuilder) {
		s.Field("name", name)
		s.Field("plain", plain)
		s.Field("flags", flags)
		s.Field("untouched", z.Boolean())
	})
	registry := map[z.AnySchema]z.Meta{
		name:  {Title: "ext", Examples: []any{"a"}, Custom: map[string]any{"x-ext": true}},
		plain: {Description: "from registry"},
		flags: {Deprecated: true, ReadOnly: true},
	}
	got, err := z.ToJSONSchema(sch, z.WithMetadata(func(s z.AnySchema) (z.Meta, bool) {
		m, ok := registry[s]
		return m, ok
	}))
	want := gapXD2020 + `"additionalProperties":false,"properties":{` +
		`"flags":{"deprecated":true,"readOnly":true,"type":"integer"},` +
		`"name":{"description":"own desc","examples":["a"],"title":"ext","type":"string","x-ext":true,"x-own":1},` +
		`"plain":{"description":"from registry","type":"string"},` +
		`"untouched":{"type":"boolean"}},"required":["flags","name","plain","untouched"],"type":"object"}`
	if err != nil || string(got) != want {
		t.Fatalf("\n got: %s\nwant: %s\n err: %v", got, want, err)
	}
}

// @spec REQ-EXP-002
func TestGapXOwnCustomMetaIsEmitted(t *testing.T) {
	sch := z.String().Meta(func(m *z.MetaBuilder) { m.Custom("x-a", "b").Custom("type", "ignored") })
	got, err := z.ToJSONSchema(sch)
	if err != nil || string(got) != gapXD2020+`"type":"string","x-a":"b"}` {
		t.Fatalf("got %s %v", got, err)
	}
}
