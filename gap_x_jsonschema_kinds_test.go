package z_test

import (
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

const (
	gapXD2020 = `{"$schema":"https://json-schema.org/draft/2020-12/schema",`
	gapXD07   = `{"$schema":"http://json-schema.org/draft-07/schema#",`
	gapXD04   = `{"$schema":"http://json-schema.org/draft-04/schema#",`
)

type gapXCase struct {
	name string
	sch  z.AnySchema
	opts []z.JSONSchemaOption
	want string
}

func gapXRun(t *testing.T, cases []gapXCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := z.ToJSONSchema(c.sch, c.opts...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}

func gapXOpts(o ...z.JSONSchemaOption) []z.JSONSchemaOption { return o }

// @spec REQ-EXP-002
func TestGapXIntegerNumberTypes(t *testing.T) {
	gapXRun(t, []gapXCase{
		{"int", z.Int(), nil, gapXD2020 + `"type":"integer"}`},
		{"int8", z.Int8(), nil, gapXD2020 + `"type":"integer"}`},
		{"int16", z.Int16(), nil, gapXD2020 + `"type":"integer"}`},
		{"int32", z.Int32(), nil, gapXD2020 + `"type":"integer"}`},
		{"uint", z.Uint(), nil, gapXD2020 + `"type":"integer"}`},
		{"uint8", z.Uint8(), nil, gapXD2020 + `"type":"integer"}`},
		{"uint16", z.Uint16(), nil, gapXD2020 + `"type":"integer"}`},
		{"uint32", z.Uint32(), nil, gapXD2020 + `"type":"integer"}`},
		{"float32", z.Float32(), nil, gapXD2020 + `"type":"number"}`},
		{"float64", z.Float64(), nil, gapXD2020 + `"type":"number"}`},
		{"coerce int", z.Coerce.Int(), nil, gapXD2020 + `"type":"integer"}`},
		{"coerce float", z.Coerce.Float(), nil, gapXD2020 + `"type":"number"}`},
		{"int input", z.Int(), gapXOpts(z.WithIO(z.Input)), gapXD2020 + `"type":"integer"}`},
		{"int draft04", z.Int(), gapXOpts(z.WithTarget(z.Draft04)), gapXD04 + `"type":"integer"}`},
		{"int openapi30", z.Int(), gapXOpts(z.WithTarget(z.OpenAPI30)), `{"type":"integer"}`},
		{"int nullable", z.Int().Nullable(), nil, gapXD2020 + `"type":["integer","null"]}`},
	})
}

// @spec REQ-EXP-002
func TestGapXNumberConstraints(t *testing.T) {
	d07 := gapXOpts(z.WithTarget(z.Draft07))
	d04 := gapXOpts(z.WithTarget(z.Draft04))
	o30 := gapXOpts(z.WithTarget(z.OpenAPI30))
	gapXRun(t, []gapXCase{
		{"all kinds", z.Int().GT(1).LTE(10).MultipleOf(2), nil,
			gapXD2020 + `"exclusiveMinimum":1,"maximum":10,"multipleOf":2,"type":"integer"}`},
		{"gte lt", z.Float64().GTE(0.5).LT(9.5), nil,
			gapXD2020 + `"exclusiveMaximum":9.5,"minimum":0.5,"type":"number"}`},
		{"draft07 numeric exclusive", z.Int().GT(1).LT(9), d07,
			gapXD07 + `"exclusiveMaximum":9,"exclusiveMinimum":1,"type":"integer"}`},
		{"draft04 boolean exclusive", z.Int().GT(1).LT(9), d04,
			gapXD04 + `"exclusiveMaximum":true,"exclusiveMinimum":true,"maximum":9,"minimum":1,"type":"integer"}`},
		{"openapi30 boolean exclusive", z.Int().GT(1).LT(9), o30,
			`{"exclusiveMaximum":true,"exclusiveMinimum":true,"maximum":9,"minimum":1,"type":"integer"}`},
		{"openapi31 numeric exclusive", z.Int().GT(1).LT(9), gapXOpts(z.WithTarget(z.OpenAPI31)),
			`{"exclusiveMaximum":9,"exclusiveMinimum":1,"type":"integer"}`},
		{"positive and nonnegative", z.Int().Positive().Max(5), nil,
			gapXD2020 + `"exclusiveMinimum":0,"maximum":5,"type":"integer"}`},

		// the stricter of repeated checks wins
		{"min then tighter min", z.Int().GTE(3).GTE(5), nil, gapXD2020 + `"minimum":5,"type":"integer"}`},
		{"min then looser min", z.Int().GTE(5).GTE(3), nil, gapXD2020 + `"minimum":5,"type":"integer"}`},
		{"gt then looser gt", z.Int().GT(7).GT(5), nil, gapXD2020 + `"exclusiveMinimum":7,"type":"integer"}`},
		{"max then tighter max", z.Int().LTE(7).LTE(5), nil, gapXD2020 + `"maximum":5,"type":"integer"}`},
		{"max then looser max", z.Int().LTE(5).LTE(7), nil, gapXD2020 + `"maximum":5,"type":"integer"}`},
		{"legacy min then tighter", z.Int().GTE(3).GTE(5), d04, gapXD04 + `"minimum":5,"type":"integer"}`},
		{"legacy min then looser", z.Int().GTE(5).GTE(3), d04, gapXD04 + `"minimum":5,"type":"integer"}`},
		{"legacy max then tighter", z.Int().LTE(7).LTE(5), d04, gapXD04 + `"maximum":5,"type":"integer"}`},
		{"legacy max then looser", z.Int().LTE(5).LTE(7), d04, gapXD04 + `"maximum":5,"type":"integer"}`},
		{"legacy gt then gte same value stays exclusive", z.Int().GT(5).GTE(5), d04,
			gapXD04 + `"exclusiveMinimum":true,"minimum":5,"type":"integer"}`},
		{"legacy gte then gt same value becomes exclusive", z.Int().GTE(5).GT(5), d04,
			gapXD04 + `"exclusiveMinimum":true,"minimum":5,"type":"integer"}`},
		{"legacy gt then tighter gte drops exclusive", z.Int().GT(3).GTE(5), d04,
			gapXD04 + `"minimum":5,"type":"integer"}`},
	})
}

// @spec REQ-EXP-004
func TestGapXInt64AndUint64AreUnrepresentable(t *testing.T) {
	for name, sch := range map[string]z.AnySchema{
		"int64":  z.Int64(),
		"uint64": z.Uint64(),
		"bigint": z.BigInt(),
		"nested": z.Object(func(s *z.ObjectBuilder) { s.Field("n", z.Int64()) }),
	} {
		t.Run(name+" throws", func(t *testing.T) {
			_, err := z.ToJSONSchema(sch)
			if err == nil || !strings.Contains(err.Error(), "unrepresentable") {
				t.Fatalf("want unrepresentable error, got %v", err)
			}
		})
	}
	gapXRun(t, []gapXCase{
		{"int64 any", z.Int64(), gapXOpts(z.WithUnrepresentable(z.UnrepresentableAny)), `{}`},
		{"uint64 any", z.Uint64(), gapXOpts(z.WithUnrepresentable(z.UnrepresentableAny)), `{}`},
		{"nested int64 any", z.Object(func(s *z.ObjectBuilder) { s.Field("n", z.Int64()) }),
			gapXOpts(z.WithUnrepresentable(z.UnrepresentableAny)),
			gapXD2020 + `"additionalProperties":false,"properties":{"n":{}},"required":["n"],"type":"object"}`},
	})
}

// @spec REQ-EXP-004
func TestGapXOtherUnrepresentableKinds(t *testing.T) {
	cases := map[string]z.AnySchema{
		"date": z.Date(),
		"set":  z.Set(z.String()),
		"map":  z.Map(z.String(), z.Int()),
		"transform": z.Transform(z.String(), func(v string) (int, error) {
			return len(v), nil
		}),
		"pipe into transform": z.Pipe(z.String(), z.Transform(z.String(), func(v string) (int, error) {
			return len(v), nil
		})),
		"default of a date": z.Object(func(s *z.ObjectBuilder) { s.Field("d", z.Date().Default(time.Time{})) }),
	}
	for name, sch := range cases {
		t.Run(name+" throws", func(t *testing.T) {
			if _, err := z.ToJSONSchema(sch); err == nil {
				t.Fatal("want error")
			}
		})
		t.Run(name+" any", func(t *testing.T) {
			if _, err := z.ToJSONSchema(sch, z.WithUnrepresentable(z.UnrepresentableAny)); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// @spec REQ-EXP-002
func TestGapXRecordExport(t *testing.T) {
	gapXRun(t, []gapXCase{
		{"string keys", z.Record(z.String(), z.Int()), nil,
			gapXD2020 + `"additionalProperties":{"type":"integer"},"type":"object"}`},
		{"enum keys use propertyNames", z.Record(z.Enum("a", "b"), z.Int()), nil,
			gapXD2020 + `"additionalProperties":{"type":"integer"},"propertyNames":{"enum":["a","b"]},"type":"object"}`},
		{"enum keys draft07", z.Record(z.Enum("a", "b"), z.String()), gapXOpts(z.WithTarget(z.Draft07)),
			gapXD07 + `"additionalProperties":{"type":"string"},"propertyNames":{"enum":["a","b"]},"type":"object"}`},
		{"enum keys openapi31", z.Record(z.Enum("a"), z.String()), gapXOpts(z.WithTarget(z.OpenAPI31)),
			`{"additionalProperties":{"type":"string"},"propertyNames":{"enum":["a"]},"type":"object"}`},
		{"enum keys draft04 has no propertyNames", z.Record(z.Enum("a", "b"), z.String()), gapXOpts(z.WithTarget(z.Draft04)),
			gapXD04 + `"additionalProperties":{"type":"string"},"type":"object"}`},
		{"enum keys openapi30 has no propertyNames", z.Record(z.Enum("a", "b"), z.String()), gapXOpts(z.WithTarget(z.OpenAPI30)),
			`{"additionalProperties":{"type":"string"},"type":"object"}`},
		{"nested record value", z.Record(z.String(), z.Record(z.String(), z.Boolean())), nil,
			gapXD2020 + `"additionalProperties":{"additionalProperties":{"type":"boolean"},"type":"object"},"type":"object"}`},
		{"record in an object", z.Object(func(s *z.ObjectBuilder) { s.Field("tags", z.Record(z.String(), z.String())) }), nil,
			gapXD2020 + `"additionalProperties":false,"properties":{"tags":{"additionalProperties":{"type":"string"},"type":"object"}},"required":["tags"],"type":"object"}`},
	})
	if _, err := z.ToJSONSchema(z.Record(z.String(), z.Date())); err == nil {
		t.Fatal("unrepresentable value schema must fail")
	}
	if _, err := z.ToJSONSchema(z.Record(z.Date(), z.String())); err == nil {
		t.Fatal("unrepresentable key schema must fail")
	}
	if _, err := z.ToJSONSchema(z.Record(z.Date(), z.String()), z.WithTarget(z.Draft04)); err != nil {
		t.Fatalf("draft04 never exports the key schema: %v", err)
	}
}

// @spec REQ-EXP-001
func TestGapXTupleExport(t *testing.T) {
	two := func() *z.TupleSchema { return z.Tuple(z.String(), z.Int()) }
	withRest := func() *z.TupleSchema { return two().Rest(z.Boolean()) }
	gapXRun(t, []gapXCase{
		{"2020-12", two(), nil,
			gapXD2020 + `"maxItems":2,"minItems":2,"prefixItems":[{"type":"string"},{"type":"integer"}],"type":"array"}`},
		{"2020-12 rest", withRest(), nil,
			gapXD2020 + `"items":{"type":"boolean"},"minItems":2,"prefixItems":[{"type":"string"},{"type":"integer"}],"type":"array"}`},
		{"openapi31 rest", withRest(), gapXOpts(z.WithTarget(z.OpenAPI31)),
			`{"items":{"type":"boolean"},"minItems":2,"prefixItems":[{"type":"string"},{"type":"integer"}],"type":"array"}`},
		{"draft07", two(), gapXOpts(z.WithTarget(z.Draft07)),
			gapXD07 + `"items":[{"type":"string"},{"type":"integer"}],"maxItems":2,"minItems":2,"type":"array"}`},
		{"draft07 rest", withRest(), gapXOpts(z.WithTarget(z.Draft07)),
			gapXD07 + `"additionalItems":{"type":"boolean"},"items":[{"type":"string"},{"type":"integer"}],"minItems":2,"type":"array"}`},
		{"draft04", two(), gapXOpts(z.WithTarget(z.Draft04)),
			gapXD04 + `"items":[{"type":"string"},{"type":"integer"}],"maxItems":2,"minItems":2,"type":"array"}`},
		{"openapi30", two(), gapXOpts(z.WithTarget(z.OpenAPI30)),
			`{"items":{"anyOf":[{"type":"string"},{"type":"integer"}]},"maxItems":2,"minItems":2,"type":"array"}`},
		{"openapi30 rest", withRest(), gapXOpts(z.WithTarget(z.OpenAPI30)),
			`{"items":{"anyOf":[{"type":"string"},{"type":"integer"},{"type":"boolean"}]},"minItems":2,"type":"array"}`},
		{"empty", z.Tuple(), nil, gapXD2020 + `"maxItems":0,"type":"array"}`},
		{"empty with rest", z.Tuple().Rest(z.String()), nil, gapXD2020 + `"items":{"type":"string"},"type":"array"}`},
		{"nullable items", z.Tuple(z.String().Nullable()), nil,
			gapXD2020 + `"maxItems":1,"minItems":1,"prefixItems":[{"type":["string","null"]}],"type":"array"}`},
	})
	if _, err := z.ToJSONSchema(z.Tuple(z.String(), z.Date())); err == nil {
		t.Fatal("unrepresentable item must fail")
	}
	if _, err := z.ToJSONSchema(z.Tuple(z.String()).Rest(z.Date())); err == nil {
		t.Fatal("unrepresentable rest must fail")
	}
}

type gapXUser struct {
	Name string  `json:"name"`
	Age  int     `json:"age"`
	Nick *string `json:"nick"`
	Bio  string  `json:"bio"`
	Tag  string  `json:"tag"`
	Role string  `json:"role"`
}

func gapXUserSchema() *z.StructSchema[gapXUser] {
	return z.Struct(func(s *z.StructBuilder, u *gapXUser) {
		s.Field(&u.Name, z.String())
		s.Field(&u.Age, z.Int().Min(0))
		s.Field(&u.Bio, z.String().Default("none"))
		s.Field(&u.Tag, z.NewCatchSchema(z.String(), "x"))
		s.Field(&u.Role, z.NewPrefaultSchema(z.String(), "guest"))
	})
}

// @spec REQ-EXP-002
func TestGapXStructExport(t *testing.T) {
	props := `"properties":{"age":{"minimum":0,"type":"integer"},"bio":{"default":"none","type":"string"},` +
		`"name":{"type":"string"},"role":{"type":"string"},"tag":{"type":"string"}}`
	gapXRun(t, []gapXCase{
		{"strip output", gapXUserSchema(), nil,
			gapXD2020 + `"additionalProperties":false,` + props + `,"required":["age","bio","name"],"type":"object"}`},
		{"strip input", gapXUserSchema(), gapXOpts(z.WithIO(z.Input)),
			gapXD2020 + props + `,"required":["age","name"],"type":"object"}`},
		{"strict input", gapXUserSchema().Strict(), gapXOpts(z.WithIO(z.Input)),
			gapXD2020 + `"additionalProperties":false,` + props + `,"required":["age","name"],"type":"object"}`},
		{"empty struct", z.Struct[struct{}](func(s *z.StructBuilder, u *struct{}) {}), nil,
			gapXD2020 + `"additionalProperties":false,"type":"object"}`},
	})
}

// @spec REQ-EXP-002
func TestGapXObjectRequiredness(t *testing.T) {
	obj := func() *z.ObjectSchema {
		return z.Object(func(s *z.ObjectBuilder) {
			s.Field("plain", z.String())
			s.Field("opt", z.String().Optional())
			s.Field("def", z.String().Default("d"))
			s.Field("catch", z.NewCatchSchema(z.String(), "c"))
			s.Field("pre", z.NewPrefaultSchema(z.String(), "p"))
		})
	}
	props := `"properties":{"catch":{"type":"string"},"def":{"default":"d","type":"string"},"opt":{"type":"string"},` +
		`"plain":{"type":"string"},"pre":{"type":"string"}}`
	gapXRun(t, []gapXCase{
		{"output", obj(), nil,
			gapXD2020 + `"additionalProperties":false,` + props + `,"required":["def","plain"],"type":"object"}`},
		{"input", obj(), gapXOpts(z.WithIO(z.Input)),
			gapXD2020 + props + `,"required":["plain"],"type":"object"}`},
	})
}

// @spec REQ-EXP-002
func TestGapXPipeCodecAndWrapperKinds(t *testing.T) {
	toInt := z.Transform(z.String(), func(v string) (int, error) { return len(v), nil })
	in := gapXOpts(z.WithIO(z.Input))
	gapXRun(t, []gapXCase{
		{"pipe output uses right", z.Pipe(z.String(), z.Int()), nil, gapXD2020 + `"type":"integer"}`},
		{"pipe input uses left", z.Pipe(z.String(), z.Int()), in, gapXD2020 + `"type":"string"}`},
		{"pipe input skips leading transform", z.Pipe(toInt, z.Int().Min(1)), in,
			gapXD2020 + `"minimum":1,"type":"integer"}`},
		{"pipe output of transform then schema", z.Pipe(toInt, z.Int().Min(1)), nil,
			gapXD2020 + `"minimum":1,"type":"integer"}`},
		{"codec output", z.StringBool(), nil, gapXD2020 + `"type":"boolean"}`},
		{"codec input", z.StringBool(), in, gapXD2020 + `"type":"string"}`},
		{"readonly", z.NewReadonlySchema(z.String()), nil, gapXD2020 + `"readOnly":true,"type":"string"}`},
		{"readonly object field", z.Object(func(s *z.ObjectBuilder) {
			s.Field("id", z.NewReadonlySchema(z.Int()))
		}), nil, gapXD2020 + `"additionalProperties":false,"properties":{"id":{"readOnly":true,"type":"integer"}},"required":["id"],"type":"object"}`},
		{"catch", z.NewCatchSchema(z.String(), "x"), nil, gapXD2020 + `"type":"string"}`},
		{"prefault", z.NewPrefaultSchema(z.Int(), 1), nil, gapXD2020 + `"type":"integer"}`},
		{"optional", z.Int().Optional(), nil, gapXD2020 + `"type":"integer"}`},
		{"refine exports the refined schema", z.Refine(z.String(), func(v string) bool { return v != "" }), nil,
			gapXD2020 + `"type":"string"}`},
		{"preprocess exports the inner schema", z.Preprocess(func(raw any) any { return raw }, z.Int()), nil,
			gapXD2020 + `"type":"integer"}`},
	})
}

// @spec REQ-EXP-002
func TestGapXDefaultValues(t *testing.T) {
	gapXRun(t, []gapXCase{
		{"string", z.String().Default("x"), nil, gapXD2020 + `"default":"x","type":"string"}`},
		{"number", z.Int().Default(3), nil, gapXD2020 + `"default":3,"type":"integer"}`},
		{"bool", z.Boolean().Default(false), nil, gapXD2020 + `"default":false,"type":"boolean"}`},
		{"input mode keeps default", z.String().Default("x"), gapXOpts(z.WithIO(z.Input)),
			gapXD2020 + `"default":"x","type":"string"}`},
		{"openapi30", z.String().Default("x"), gapXOpts(z.WithTarget(z.OpenAPI30)), `{"default":"x","type":"string"}`},
		{"nullable default", z.NewDefaultSchema(z.String().Nullable(), z.NullableOf("x")), nil,
			gapXD2020 + `"default":"x","type":["string","null"]}`},
		{"unrepresentable value dropped under Any", z.Any().Default(make(chan int)),
			gapXOpts(z.WithUnrepresentable(z.UnrepresentableAny)), `{}`},
	})
	if _, err := z.ToJSONSchema(z.Any().Default(make(chan int))); err == nil {
		t.Fatal("non JSON default must fail under Throw")
	}
}

// @spec REQ-EXP-002
func TestGapXAnyNeverNull(t *testing.T) {
	gapXRun(t, []gapXCase{
		{"any", z.Any(), nil, `{}`},
		{"unknown", z.Unknown(), nil, `{}`},
		{"never", z.Never(), nil, gapXD2020 + `"not":{}}`},
		{"null", z.Null(), nil, gapXD2020 + `"type":"null"}`},
		{"null openapi30", z.Null(), gapXOpts(z.WithTarget(z.OpenAPI30)), `{"enum":[null],"nullable":true,"type":"string"}`},
		{"null openapi31", z.Null(), gapXOpts(z.WithTarget(z.OpenAPI31)), `{"type":"null"}`},
		{"never field", z.Object(func(s *z.ObjectBuilder) { s.Field("n", z.Never()) }), nil,
			gapXD2020 + `"additionalProperties":false,"properties":{"n":{"not":{}}},"required":["n"],"type":"object"}`},
	})
}

// @spec REQ-EXP-007
func TestGapXExportIsDeterministic(t *testing.T) {
	sch := z.Object(func(s *z.ObjectBuilder) {
		s.Field("rec", z.Record(z.Enum("a", "b", "c"), z.Int().Min(1)))
		s.Field("tup", z.Tuple(z.String(), z.Int()).Rest(z.Boolean()))
		s.Field("u", gapXUserSchema())
		s.Field("ro", z.NewReadonlySchema(z.String()))
		s.Field("d", z.String().Default("x"))
	})
	first, err := z.ToJSONSchema(sch, z.WithReused(z.ReusedRef))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 25; i++ {
		again, err := z.ToJSONSchema(sch, z.WithReused(z.ReusedRef))
		if err != nil || string(again) != string(first) {
			t.Fatalf("run %d differs:\n%s\n%s", i, first, again)
		}
	}
}
