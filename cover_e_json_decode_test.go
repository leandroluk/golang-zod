package z_test

import (
	"encoding/json"
	"reflect"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-OPT-002 REQ-OPT-003
func TestCoverEJSONDecodeValues(t *testing.T) {
	cases := []struct {
		name string
		json string
		want any
	}{
		{"null", `null`, nil},
		{"true", `true`, true},
		{"false", `false`, false},
		{"string", `"héllo"`, "héllo"},
		{"small int stays float64", `42`, float64(42)},
		{"fraction", `1.5`, 1.5},
		{"exponent", `1e3`, float64(1000)},
		{"exact large power of two", `9007199254740992`, float64(9007199254740992)},
		{"imprecise int keeps json.Number", `9007199254740993`, json.Number("9007199254740993")},
		{"max uint64 keeps json.Number", `18446744073709551615`, json.Number("18446744073709551615")},
		{"negative imprecise", `-9007199254740993`, json.Number("-9007199254740993")},
		{"empty object", `{}`, map[string]any{}},
		{"empty array", `[]`, []any{}},
		{"nested", `{"a":[1,{"b":null}],"c":"x"}`, map[string]any{"a": []any{float64(1), map[string]any{"b": nil}}, "c": "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := z.Any().ParseJSON([]byte(tc.json))
			if err != nil {
				t.Fatalf("ParseJSON(%s): %v", tc.json, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseJSON(%s) = %#v, want %#v", tc.json, got, tc.want)
			}
		})
	}
}

// @spec REQ-OPT-002 REQ-OPT-003
func TestCoverEJSONDecodeErrors(t *testing.T) {
	bad := []string{
		``, `{`, `[`, `{"a"`, `{"a":`, `{"a":1`, `[1`, `[1,`, `{"a":1,}`, `[1,]`, `{"a":1]`, `[1}`,
		`tru`, `"abc`, `{"a":1e999}`, `[1e999]`, `1e999`, `{"a":[1e999]}`, `[{"a":1e999}]`, `{"a":{"b"}}`, `[[`,
	}
	for _, in := range bad {
		t.Run(in, func(t *testing.T) {
			if _, err := z.Any().ParseJSON([]byte(in)); err == nil {
				t.Fatalf("ParseJSON(%q) must fail", in)
			}
		})
	}
}
