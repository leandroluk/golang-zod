package z_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type gapZCase struct {
	name   string
	schema string
	accept []string
	reject []string
}

func gapZJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("bad fixture %q: %v", s, err)
	}
	return v
}

func gapZRun(t *testing.T, cases []gapZCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sch, err := z.FromJSONSchema([]byte(tc.schema))
			if err != nil {
				t.Fatalf("FromJSONSchema(%s): %v", tc.schema, err)
			}
			for _, in := range tc.accept {
				if _, err := sch.Parse(gapZJSON(t, in)); err != nil {
					t.Errorf("should accept %s: %v", in, err)
				}
			}
			for _, in := range tc.reject {
				if _, err := sch.Parse(gapZJSON(t, in)); err == nil {
					t.Errorf("should reject %s", in)
				}
			}
		})
	}
}

func gapZErrors(t *testing.T, cases map[string]string) {
	t.Helper()
	for schema, want := range cases {
		sch, err := z.FromJSONSchema([]byte(schema))
		if err == nil || sch != nil {
			t.Errorf("FromJSONSchema(%s) should fail, got %v", schema, sch)
			continue
		}
		if !strings.HasPrefix(err.Error(), "golang-zod: unsupported JSON Schema: ") || !strings.Contains(err.Error(), want) {
			t.Errorf("FromJSONSchema(%s) error = %q, want it to contain %q", schema, err, want)
		}
	}
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaRefs(t *testing.T) {
	tree := `{"$ref":"#/$defs/node","$defs":{"node":{"type":"object","properties":{"v":{"type":"integer"},"kids":{"type":"array","items":{"$ref":"#/$defs/node"}}},"required":["v"]}}}`
	gapZRun(t, []gapZCase{
		{"$defs", `{"type":"object","properties":{"a":{"$ref":"#/$defs/s"}},"required":["a"],"$defs":{"s":{"type":"string","minLength":2}}}`,
			[]string{`{"a":"xy"}`}, []string{`{"a":"x"}`, `{"a":1}`, `{}`}},
		{"definitions", `{"properties":{"a":{"$ref":"#/definitions/s"}},"definitions":{"s":{"type":"boolean"}}}`,
			[]string{`{"a":true}`, `{}`}, []string{`{"a":"x"}`}},
		{"components schemas", `{"$ref":"#/components/schemas/Pet","components":{"schemas":{"Pet":{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}}}}`,
			[]string{`{"name":"rex"}`}, []string{`{}`, `{"name":1}`}},
		{"recursive tree terminates", tree,
			[]string{`{"v":1}`, `{"v":1,"kids":[{"v":2,"kids":[{"v":3}]}]}`}, []string{`{"v":1,"kids":[{"v":"x"}]}`, `{"v":1,"kids":[{"v":2,"kids":[{}]}]}`}},
		{"root recursion", `{"type":"object","properties":{"next":{"$ref":"#"}}}`,
			[]string{`{}`, `{"next":{"next":{}}}`}, []string{`{"next":{"next":1}}`}},
		{"mutual recursion", `{"$ref":"#/$defs/a","$defs":{"a":{"type":"object","properties":{"b":{"$ref":"#/$defs/b"}}},"b":{"type":"object","properties":{"a":{"$ref":"#/$defs/a"}}}}}`,
			[]string{`{"b":{"a":{"b":{}}}}`}, []string{`{"b":{"a":{"b":1}}}`}},
		{"pointer escapes and array index", `{"properties":{"a":{"$ref":"#/$defs/a~1b%20c"},"b":{"$ref":"#/$defs/list/1"},"c":{"$ref":"#/$defs/t~0"}},"$defs":{"a/b c":{"type":"string"},"t~":{"type":"null"},"list":[{"type":"number"},{"type":"integer"}]}}`,
			[]string{`{"a":"x","b":1,"c":null}`}, []string{`{"a":1}`, `{"b":"x"}`, `{"c":1}`}},
		{"siblings are intersected", `{"$ref":"#/$defs/s","maxLength":3,"$defs":{"s":{"type":"string","minLength":2}}}`,
			[]string{`"ab"`, `"abc"`}, []string{`"a"`, `"abcd"`, `1`}},
		{"shared ref reused", `{"properties":{"a":{"$ref":"#/$defs/s"},"b":{"$ref":"#/$defs/s"}},"$defs":{"s":{"type":"string"}}}`,
			[]string{`{"a":"x","b":"y"}`}, []string{`{"b":1}`}},
	})

	gapZErrors(t, map[string]string{
		`{"$ref":"https://example.com/schema.json"}`:              `#/$ref: remote or invalid $ref https://example.com/schema.json is not supported`,
		`{"$ref":"other.json#/a"}`:                                "remote or invalid $ref",
		`{"$ref":5}`:                                              "remote or invalid $ref 5",
		`{"$ref":"#/$defs/missing"}`:                              `$ref "#/$defs/missing" cannot be resolved`,
		`{"$ref":"#/a/0","a":[]}`:                                 `cannot be resolved`,
		`{"$ref":"#/a/-1","a":[1]}`:                               `cannot be resolved`,
		`{"$ref":"#/a/x","a":[1]}`:                                `cannot be resolved`,
		`{"$ref":"#/a/b","a":"str"}`:                              `cannot be resolved`,
		`{"$ref":"#/%zz"}`:                                        `cannot be resolved`,
		`{"$ref":"#foo"}`:                                         `"#foo" is not a JSON pointer`,
		`{"$ref":"#/$defs/a","$defs":{"a":{"$ref":"#/$defs/a"}}}`: `circular $ref "#/$defs/a"`,
		`{"$ref":"#/$defs/a","$defs":{"a":{"allOf":[{"$ref":"#/$defs/a"}]}}}`:              `circular $ref "#/$defs/a"`,
		`{"$ref":"#/$defs/a","$defs":{"a":{"$ref":"#/$defs/b"},"b":{"$ref":"#/$defs/a"}}}`: `circular $ref`,
		`{"$ref":"#/$defs/a","$defs":{"a":3}}`:                                             `schema must be an object or boolean, got float64`,
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaCombinators(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"anyOf", `{"anyOf":[{"type":"string"},{"type":"integer"}]}`, []string{`"a"`, `1`}, []string{`1.5`, `null`, `true`}},
		{"oneOf rejects multiple matches", `{"oneOf":[{"type":"number"},{"type":"integer"}]}`, []string{`1.5`}, []string{`1`, `"x"`}},
		{"oneOf exactly one", `{"oneOf":[{"type":"string"},{"type":"boolean"}]}`, []string{`"x"`, `true`}, []string{`1`, `null`}},
		{"allOf of three", `{"allOf":[{"type":"string"},{"minLength":2},{"maxLength":3}]}`, []string{`"ab"`, `"abc"`}, []string{`"a"`, `"abcd"`, `1`}},
		{"allOf of objects", `{"allOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},{"type":"object","properties":{"b":{"type":"number"}},"required":["b"]}]}`,
			[]string{`{"a":"x","b":1}`}, []string{`{"a":"x"}`, `{"b":1}`, `"x"`}},
		{"combinator with sibling type", `{"type":"string","anyOf":[{"minLength":3},{"pattern":"^x"}]}`, []string{`"abc"`, `"x"`}, []string{`"a"`, `1`}},
		{"not", `{"not":{"type":"string"}}`, []string{`1`, `null`, `{}`}, []string{`"a"`}},
		{"not with sibling type", `{"type":"integer","not":{"const":3}}`, []string{`1`, `4`}, []string{`3`, `"x"`}},
		{"if then else", `{"if":{"type":"string"},"then":{"minLength":2},"else":{"type":"integer"}}`, []string{`"ab"`, `1`}, []string{`"a"`, `1.5`, `null`}},
		{"if then only", `{"if":{"type":"string"},"then":{"minLength":2}}`, []string{`"ab"`, `1`, `null`}, []string{`"a"`}},
		{"if else only", `{"if":{"type":"string"},"else":{"type":"integer"}}`, []string{`"a"`, `1`}, []string{`1.5`}},
		{"boolean schemas", `{"type":"object","properties":{"any":true,"none":false}}`, []string{`{}`, `{"any":1}`}, []string{`{"none":1}`}},
		{"true schema", `true`, []string{`1`, `null`, `"x"`}, nil},
		{"false schema", `false`, nil, []string{`1`, `null`, `"x"`}},
		{"empty schema", `{}`, []string{`1`, `null`, `{}`}, nil},
	})

	gapZErrors(t, map[string]string{
		`{"anyOf":[]}`:       "#/anyOf: anyOf must be a non-empty array",
		`{"oneOf":"x"}`:      "oneOf must be a non-empty array",
		`{"allOf":[]}`:       "allOf must be a non-empty array",
		`{"anyOf":[{},7]}`:   "#/anyOf/1: schema must be an object or boolean, got float64",
		`{"not":7}`:          "#/not: schema must be an object or boolean",
		`{"if":7}`:           "#/if: schema must be an object or boolean",
		`{"if":{},"then":7}`: "#/then: schema must be an object or boolean",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaConstEnum(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"const string", `{"const":"a"}`, []string{`"a"`}, []string{`"b"`, `1`, `null`}},
		{"const number", `{"const":1.5}`, []string{`1.5`}, []string{`2`, `"1.5"`}},
		{"const bool", `{"const":false}`, []string{`false`}, []string{`true`, `0`}},
		{"const null", `{"const":null}`, []string{`null`}, []string{`0`, `""`}},
		{"const object", `{"const":{"a":[1,2]}}`, []string{`{"a":[1,2]}`}, []string{`{"a":[1]}`, `{}`, `null`}},
		{"const array", `{"const":[1,"x"]}`, []string{`[1,"x"]`}, []string{`[1]`, `["x",1]`}},
		{"enum strings", `{"enum":["A","B"]}`, []string{`"A"`, `"B"`}, []string{`"C"`, `1`, `null`}},
		{"enum mixed", `{"enum":[1,"a",null,{"k":true}]}`, []string{`1`, `"a"`, `null`, `{"k":true}`}, []string{`2`, `"z"`, `{}`, `false`}},
		{"enum with type", `{"type":"string","enum":["a","b"]}`, []string{`"a"`}, []string{`"c"`, `1`}},
		{"enum with constraint", `{"enum":[1,2,3],"minimum":2}`, []string{`2`, `3`}, []string{`1`, `4`}},
	})

	enum, _ := z.FromJSONSchema([]byte(`{"enum":[1,"a"]}`))
	if _, err := enum.Parse("zzz"); err == nil || !strings.Contains(err.Error(), "value must be one of the enum values") {
		t.Errorf("enum error = %v", err)
	}
	cst, _ := z.FromJSONSchema([]byte(`{"const":[1]}`))
	if _, err := cst.Parse([]any{2.0}); err == nil || !strings.Contains(err.Error(), "value must equal the const value") {
		t.Errorf("const error = %v", err)
	}

	gapZErrors(t, map[string]string{
		`{"enum":[]}`:  "#/enum: enum must be a non-empty array",
		`{"enum":"a"}`: "enum must be a non-empty array",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaNumbers(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"number inclusive bounds", `{"type":"number","minimum":1,"maximum":3}`, []string{`1`, `2.5`, `3`}, []string{`0.9`, `3.1`, `"2"`}},
		{"number 2019 exclusive bounds", `{"type":"number","exclusiveMinimum":1,"exclusiveMaximum":3}`, []string{`1.1`, `2.9`}, []string{`1`, `3`}},
		{"number draft-04 exclusive bounds", `{"type":"number","minimum":1,"exclusiveMinimum":true,"maximum":3,"exclusiveMaximum":true}`, []string{`2`}, []string{`1`, `3`}},
		{"number draft-04 non-exclusive flags", `{"type":"number","minimum":1,"exclusiveMinimum":false,"maximum":3,"exclusiveMaximum":false}`, []string{`1`, `3`}, []string{`0`, `4`}},
		{"number exclusive flag without bound is ignored", `{"type":"number","exclusiveMinimum":true}`, []string{`-5`, `5`}, []string{`"x"`}},
		{"number multipleOf", `{"type":"number","multipleOf":0.5}`, []string{`0`, `1`, `1.5`}, []string{`0.3`}},
		{"integer", `{"type":"integer"}`, []string{`1`, `-5`, `2.0`}, []string{`1.5`, `"1"`, `null`}},
		{"integer inclusive bounds", `{"type":"integer","minimum":2,"maximum":4}`, []string{`2`, `3`, `4`}, []string{`1`, `5`}},
		{"integer fractional bounds", `{"type":"integer","minimum":1.5,"maximum":3.5}`, []string{`2`, `3`}, []string{`1`, `4`}},
		{"integer exclusive bounds", `{"type":"integer","exclusiveMinimum":1,"exclusiveMaximum":4}`, []string{`2`, `3`}, []string{`1`, `4`}},
		{"integer fractional exclusive bounds", `{"type":"integer","exclusiveMinimum":1.5,"exclusiveMaximum":3.5}`, []string{`2`, `3`}, []string{`1`, `4`}},
		{"integer draft-04 exclusive bounds", `{"type":"integer","minimum":0,"exclusiveMinimum":true,"maximum":10,"exclusiveMaximum":true}`, []string{`1`, `9`}, []string{`0`, `10`}},
		{"integer multipleOf", `{"type":"integer","multipleOf":3}`, []string{`0`, `9`, `-6`}, []string{`4`, `1.5`}},
		{"integer int64 format", `{"type":"integer","format":"int64","minimum":0}`, []string{`0`, `9007199254740992`}, []string{`-1`, `1.5`}},
		{"integer huge bounds saturate", `{"type":"integer","minimum":-1e30,"maximum":1e30}`, []string{`-9007199254740992`, `9007199254740992`}, []string{`0.5`}},
	})

	i64, _ := z.FromJSONSchema([]byte(`{"type":"integer","format":"int64"}`))
	if v, err := i64.Parse(7.0); err != nil || v != int64(7) {
		t.Errorf("int64 format = %#v, %v", v, err)
	}
	i, _ := z.FromJSONSchema([]byte(`{"type":"integer"}`))
	if v, err := i.Parse(7.0); err != nil || v != 7 {
		t.Errorf("integer = %#v, %v", v, err)
	}

	gapZErrors(t, map[string]string{
		`{"type":"number","minimum":"1"}`:     "#/minimum: minimum must be a number",
		`{"type":"number","multipleOf":0}`:    "#/multipleOf: multipleOf must be greater than 0",
		`{"type":"integer","multipleOf":-2}`:  "multipleOf must be greater than 0",
		`{"type":"integer","multipleOf":2.5}`: "multipleOf 2.5 is not supported for integer types",
		`{"maximum":"x"}`:                     "#/maximum: maximum must be a number",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaStrings(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"length and pattern", `{"type":"string","minLength":2,"maxLength":4,"pattern":"^[a-z]+$"}`, []string{`"ab"`, `"abcd"`}, []string{`"a"`, `"abcde"`, `"AB"`, `1`}},
		{"pattern is unanchored", `{"type":"string","pattern":"b"}`, []string{`"abc"`}, []string{`"xyz"`}},
		{"unknown format is ignored", `{"type":"string","format":"made-up"}`, []string{`"anything"`}, []string{`1`}},
		{"non-string format is ignored", `{"type":"string","format":5}`, []string{`"x"`}, nil},
		{"email", `{"type":"string","format":"email"}`, []string{`"a@b.co"`}, []string{`"nope"`, `1`}},
		{"idn-email", `{"format":"idn-email"}`, []string{`"a@b.co"`, `5`}, []string{`"nope"`}},
		{"uuid", `{"type":"string","format":"uuid"}`, []string{`"123e4567-e89b-42d3-a456-426614174000"`}, []string{`"123"`}},
		{"uri", `{"type":"string","format":"uri"}`, []string{`"https://example.com/a"`}, []string{`"not a uri"`}},
		{"url", `{"type":"string","format":"url"}`, []string{`"http://example.com"`}, []string{`"example"`}},
		{"hostname", `{"type":"string","format":"hostname"}`, []string{`"example.com"`}, []string{`"-bad-"`}},
		{"ipv4", `{"type":"string","format":"ipv4"}`, []string{`"192.168.0.1"`}, []string{`"999.1.1.1"`, `"::1"`}},
		{"ipv6", `{"type":"string","format":"ipv6"}`, []string{`"::1"`}, []string{`"1.2.3.4"`}},
		{"date-time", `{"type":"string","format":"date-time"}`, []string{`"2024-01-02T03:04:05Z"`}, []string{`"2024-01-02"`}},
		{"date", `{"type":"string","format":"date"}`, []string{`"2024-01-02"`}, []string{`"2024-13-02"`}},
		{"time", `{"type":"string","format":"time"}`, []string{`"03:04:05"`}, []string{`"25:00:00"`}},
		{"duration", `{"type":"string","format":"duration"}`, []string{`"P1DT2H"`}, []string{`"1 day"`}},
		{"byte", `{"type":"string","format":"byte"}`, []string{`"aGVsbG8="`}, []string{`"@@@"`}},
		{"format with length", `{"type":"string","format":"email","maxLength":8}`, []string{`"a@b.co"`}, []string{`"abcdefg@b.co"`}},
	})

	gapZErrors(t, map[string]string{
		`{"type":"string","minLength":"2"}`:    "#/minLength: minLength must be a number",
		`{"type":"string","maxLength":true}`:   "maxLength must be a number",
		`{"type":"string","pattern":"^(?=a)"}`: `#/pattern: pattern "^(?=a)" is not supported`,
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaArrays(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"items and size", `{"type":"array","items":{"type":"integer"},"minItems":1,"maxItems":2}`, []string{`[1]`, `[1,2]`}, []string{`[]`, `[1,2,3]`, `["a"]`}},
		{"unique items", `{"type":"array","uniqueItems":true}`, []string{`[]`, `[1,2,"1"]`, `[{"a":1},{"a":2}]`}, []string{`[1,1]`, `[{"a":1},{"a":1}]`}},
		{"items false allows only empty", `{"type":"array","items":false}`, []string{`[]`}, []string{`[1]`}},
		{"prefixItems with default rest", `{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"minItems":2}`,
			[]string{`["a",1]`, `["a",1,true]`}, []string{`["a"]`, `[1,1]`, `["a","b"]`, `"a"`}},
		{"prefixItems rest schema", `{"type":"array","prefixItems":[{"type":"string"}],"items":{"type":"boolean"},"minItems":1}`,
			[]string{`["a"]`, `["a",true,false]`}, []string{`["a",1]`, `[]`}},
		{"prefixItems items false", `{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":false,"minItems":2}`,
			[]string{`["a",1]`}, []string{`["a",1,2]`, `["a"]`}},
		{"prefixItems items true", `{"type":"array","prefixItems":[{"type":"string"}],"items":true,"minItems":1}`,
			[]string{`["a",1,null]`}, []string{`[1]`}},
		{"prefixItems shorter arrays allowed without minItems", `{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}],"items":false}`,
			[]string{`[]`, `["a"]`, `["a",1]`}, []string{`[1]`, `["a",1,2]`, `["a","b"]`}},
		{"prefixItems maxItems", `{"type":"array","prefixItems":[{"type":"string"}],"maxItems":2}`,
			[]string{`["a"]`, `["a",1]`}, []string{`["a",1,2]`}},
		{"prefixItems unique", `{"type":"array","prefixItems":[{"type":"integer"}],"uniqueItems":true,"minItems":1}`,
			[]string{`[1,"a"]`}, []string{`[1,1]`}},
		{"legacy items array", `{"type":"array","items":[{"type":"string"},{"type":"integer"}],"minItems":2,"additionalItems":false}`,
			[]string{`["a",1]`}, []string{`["a",1,2]`, `["a"]`, `[1,1]`}},
		{"legacy items array extra schema", `{"type":"array","items":[{"type":"string"}],"additionalItems":{"type":"integer"},"minItems":1}`,
			[]string{`["a",1,2]`}, []string{`["a","b"]`}},
		{"legacy items array extra allowed", `{"type":"array","items":[{"type":"string"}],"minItems":1}`,
			[]string{`["a",1,"x"]`}, []string{`[1]`}},
		{"array without type", `{"items":{"type":"string"}}`, []string{`["a"]`, `1`, `"x"`, `null`}, []string{`[1]`}},
		{"empty prefix", `{"type":"array","prefixItems":[],"items":{"type":"string"}}`, []string{`["a"]`}, []string{`[1]`}},
	})

	gapZErrors(t, map[string]string{
		`{"type":"array","minItems":"1"}`:                   "#/minItems: minItems must be a number",
		`{"type":"array","items":7}`:                        "#/items: schema must be an object or boolean",
		`{"type":"array","prefixItems":[7]}`:                "#/prefixItems/0: schema must be an object or boolean",
		`{"type":"array","prefixItems":[{}],"items":7}`:     "#/items: schema must be an object or boolean",
		`{"type":"array","items":[{}],"additionalItems":7}`: "#/additionalItems: schema must be an object or boolean",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaObjects(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"additionalProperties false is strict", `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"],"additionalProperties":false}`,
			[]string{`{"a":"x"}`}, []string{`{"a":"x","b":1}`, `{}`}},
		{"additionalProperties true is loose", `{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":true}`,
			[]string{`{}`, `{"a":"x","b":1}`}, []string{`{"a":1}`}},
		{"additionalProperties absent accepts extras", `{"type":"object","properties":{"a":{"type":"string"}}}`,
			[]string{`{"a":"x","b":1}`}, []string{`{"a":1}`}},
		{"additionalProperties schema is catchall", `{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":{"type":"integer"}}`,
			[]string{`{"a":"x","b":1,"c":2}`}, []string{`{"b":"x"}`, `{"a":1}`}},
		{"record-like object", `{"type":"object","additionalProperties":{"type":"boolean"}}`,
			[]string{`{}`, `{"x":true}`}, []string{`{"x":1}`, `[]`}},
		{"required without property schema", `{"type":"object","required":["k"]}`,
			[]string{`{"k":1}`, `{"k":null}`}, []string{`{}`, `{"x":1}`}},
		{"property counts", `{"type":"object","minProperties":1,"maxProperties":2}`,
			[]string{`{"a":1}`, `{"a":1,"b":2}`}, []string{`{}`, `{"a":1,"b":2,"c":3}`}},
		{"min properties only", `{"type":"object","minProperties":1}`, []string{`{"a":1}`}, []string{`{}`}},
		{"max properties only", `{"type":"object","maxProperties":1}`, []string{`{}`, `{"a":1}`}, []string{`{"a":1,"b":2}`}},
		{"object without type", `{"properties":{"a":{"type":"string"}},"required":["a"]}`,
			[]string{`{"a":"x"}`, `1`, `"s"`, `null`, `[]`}, []string{`{}`, `{"a":1}`}},
		{"required with non-string entries ignored", `{"type":"object","required":["a",7]}`, []string{`{"a":1}`}, []string{`{}`}},
	})

	gapZErrors(t, map[string]string{
		`{"type":"object","properties":{"a":7}}`:                "#/properties/a: schema must be an object or boolean",
		`{"type":"object","additionalProperties":{"type":"x"}}`: `#/additionalProperties/type: unknown type "x"`,
		`{"type":"object","minProperties":"1"}`:                 "#/minProperties: minProperties must be a number",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaNullableAndTypeLists(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"nullable true", `{"type":"string","nullable":true}`, []string{`"a"`, `null`}, []string{`1`}},
		{"nullable false", `{"type":"string","nullable":false}`, []string{`"a"`}, []string{`null`}},
		{"nullable integer with bounds", `{"type":"integer","minimum":1,"nullable":true}`, []string{`1`, `null`}, []string{`0`}},
		{"nullable enum", `{"enum":["a","b"],"nullable":true}`, []string{`"a"`, `null`}, []string{`"c"`}},
		{"nullable object", `{"type":"object","properties":{"a":{"type":"string","nullable":true}},"required":["a"]}`,
			[]string{`{"a":null}`, `{"a":"x"}`}, []string{`{"a":1}`, `{}`}},
		{"type list with null", `{"type":["string","null"]}`, []string{`"a"`, `null`}, []string{`1`}},
		{"type list keeps constraints", `{"type":["string","null"],"minLength":2}`, []string{`"ab"`, `null`}, []string{`"a"`, `1`}},
		{"type list of several", `{"type":["string","integer"],"minimum":2,"minLength":2}`, []string{`"ab"`, `2`}, []string{`"a"`, `1`, `true`}},
		{"type list only null", `{"type":["null"]}`, []string{`null`}, []string{`0`}},
		{"type null", `{"type":"null"}`, []string{`null`}, []string{`0`, `"x"`}},
		{"empty type list accepts nothing", `{"type":[]}`, nil, []string{`1`, `null`}},
	})

	gapZErrors(t, map[string]string{
		`{"type":"nope"}`:       `#/type: unknown type "nope"`,
		`{"type":5}`:            "#/type: type must be a string or an array of strings",
		`{"type":["string",5]}`: `unknown type ""`,
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaUntypedKeywordsOnlyConstrainTheirCategory(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"minLength", `{"minLength":2}`, []string{`"ab"`, `5`, `null`, `[]`, `{}`, `true`}, []string{`"a"`}},
		{"minimum", `{"minimum":3}`, []string{`3`, `"a"`, `null`}, []string{`2`}},
		{"minItems", `{"minItems":1}`, []string{`[1]`, `"a"`, `{}`}, []string{`[]`}},
		{"properties", `{"properties":{"a":{"type":"integer"}}}`, []string{`{"a":1}`, `"a"`, `[]`}, []string{`{"a":"x"}`}},
		{"two categories", `{"minLength":2,"minimum":3}`, []string{`"ab"`, `3`, `null`}, []string{`"a"`, `2`}},
	})

	sch, err := z.FromJSONSchema([]byte(`{"minLength":2}`))
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []any{5, int8(5), uint16(5), float32(1), struct{}{}, []string{"a"}, map[string]int{}} {
		if _, err := sch.Parse(in); err != nil {
			t.Errorf("non-string %#v should pass minLength: %v", in, err)
		}
	}
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaRejectsUnsupportedKeywords(t *testing.T) {
	for _, kw := range []string{
		"patternProperties", "propertyNames", "contains", "minContains", "maxContains",
		"dependentRequired", "dependentSchemas", "dependencies",
		"unevaluatedProperties", "unevaluatedItems", "$dynamicRef", "$recursiveRef", "$anchor",
	} {
		schema := `{"type":"object","` + kw + `":{}}`
		sch, err := z.FromJSONSchema([]byte(schema))
		want := `golang-zod: unsupported JSON Schema: #: keyword "` + kw + `" is not supported`
		if sch != nil || err == nil || err.Error() != want {
			t.Errorf("%s: got %v, %v; want error %q", kw, sch, err, want)
		}
	}
	// Nested keyword errors carry the path of the offending sub-schema.
	_, err := z.FromJSONSchema([]byte(`{"properties":{"a":{"items":{"contains":{}}}}}`))
	if err == nil || !strings.Contains(err.Error(), `#/properties/a/items: keyword "contains" is not supported`) {
		t.Errorf("nested keyword error = %v", err)
	}
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaIgnoresAnnotations(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"annotations", `{"$schema":"https://json-schema.org/draft/2020-12/schema","$id":"x","title":"T","description":"D","default":"zzz","examples":["a"],"deprecated":true,"readOnly":true,"writeOnly":false,"$comment":"c","x-extra":{"a":1},"discriminator":{"propertyName":"p"},"contentEncoding":"base64","type":"string"}`,
			[]string{`"a"`}, []string{`1`}},
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaInvalidDocuments(t *testing.T) {
	for _, in := range []string{``, `{`, `null1`} {
		sch, err := z.FromJSONSchema([]byte(in))
		if sch != nil || err == nil || !strings.HasPrefix(err.Error(), "golang-zod: failed to unmarshal JSON Schema: ") {
			t.Errorf("FromJSONSchema(%q) = %v, %v", in, sch, err)
		}
	}
	gapZErrors(t, map[string]string{
		`[1]`:  "#: schema must be an object or boolean, got []interface {}",
		`"x"`:  "#: schema must be an object or boolean, got string",
		`null`: "#: schema must be an object or boolean, got <nil>",
		`7`:    "got float64",
	})
}

// @spec REQ-EXP-005
func TestGapZFromJSONSchemaMemberKinds(t *testing.T) {
	gapZRun(t, []gapZCase{
		{"members built from checks and conjunctions", `{"type":"object","required":["c","m"],"properties":{"c":{"const":[1]},"m":{"allOf":[{"type":"string"},{"minLength":2}]},"o":{"not":{"type":"null"}}}}`,
			[]string{`{"c":[1],"m":"ab"}`, `{"c":[1],"m":"ab","o":1}`}, []string{`{"c":[2],"m":"ab"}`, `{"c":[1],"m":"a"}`, `{"c":[1],"m":"ab","o":null}`}},
	})

	sch, err := z.FromJSONSchema([]byte(`{"type":"object","properties":{"a":{"type":"string"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if out, err := z.ToJSONSchema(sch); err != nil || !strings.Contains(string(out), `"a"`) {
		t.Errorf("exporting an imported schema = %s, %v", out, err)
	}

	merged, err := z.FromJSONSchema([]byte(`{"allOf":[{"type":"object","properties":{"a":{"type":"integer"}},"required":["a"]},{"type":"object","properties":{"b":{"type":"string"}},"required":["b"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	out, err := merged.Parse(map[string]any{"a": 1.0, "b": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if m, _ := out.(map[string]any); len(m) != 2 || m["a"] != 1 || m["b"] != "x" {
		t.Errorf("allOf of objects must keep the fields of every member, got %#v", out)
	}
}
