package z_test

import (
	"encoding/json"
	"math/big"
	"slices"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

type coverAStringer struct{ v int }

func (c coverAStringer) String() string { return "S" + string(rune('0'+c.v)) }

// @spec REQ-PRIM-008
func TestCoverACoerceString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "abc", "abc"},
		{"int", 42, "42"},
		{"int64", int64(-7), "-7"},
		{"float64 integral", float64(3), "3"},
		{"float64 fractional", 1.5, "1.5"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"json.Number", json.Number("12.50"), "12.50"},
		{"fallback fmt", coverAStringer{3}, "S3"},
	}
	s := z.Coerce.String()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.Parse(tc.in)
			if err != nil || got != tc.want {
				t.Errorf("got (%q, %v), want %q", got, err, tc.want)
			}
		})
	}
	// Coerced value is still subject to the string checks.
	_, err := z.Coerce.String().Min(3).Parse(1)
	if is := coverAOneIssue(t, err); is.Code != z.TooSmall || is.Origin != "string" {
		t.Errorf("unexpected issue: %+v", is)
	}
}

// @spec REQ-PRIM-008
func TestCoverACoerceBoolean(t *testing.T) {
	tests := []struct {
		in   any
		want bool
		ok   bool
	}{
		{true, true, true},
		{false, false, true},
		{1, true, true},
		{0, false, true},
		{int64(5), true, true},
		{int64(0), false, true},
		{2.5, true, true},
		{0.0, false, true},
		{"true", true, true},
		{" YES ", true, true},
		{"on", true, true},
		{"1", true, true},
		{"false", false, true},
		{"No", false, true},
		{"off", false, true},
		{"0", false, true},
		{"", false, true},
		{"maybe", false, false},
		{[]int{1}, false, false},
		{nil, false, false},
	}
	s := z.Coerce.Boolean()
	for _, tc := range tests {
		got, err := s.Parse(tc.in)
		if tc.ok {
			if err != nil || got != tc.want {
				t.Errorf("Parse(%#v) = (%v, %v), want %v", tc.in, got, err, tc.want)
			}
			continue
		}
		if is := coverAOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "boolean" {
			t.Errorf("Parse(%#v) issue: %+v", tc.in, is)
		}
	}
}

// @spec REQ-PRIM-008
func TestCoverACoerceNumbers(t *testing.T) {
	if v, err := z.Coerce.Number().Parse(" 1.5 "); err != nil || v != 1.5 {
		t.Errorf("Number string: %v %v", v, err)
	}
	if v, err := z.Coerce.Float().Parse("2"); err != nil || v != 2 {
		t.Errorf("Float string: %v %v", v, err)
	}
	if v, err := z.Coerce.Float().Parse(3.25); err != nil || v != 3.25 {
		t.Errorf("Float passthrough: %v %v", v, err)
	}
	_, err := z.Coerce.Float().Parse("abc")
	if is := coverAOneIssue(t, err); is.Code != z.InvalidType {
		t.Errorf("Float invalid string: %+v", is)
	}

	if v, err := z.Coerce.Int().Parse(" 42 "); err != nil || v != 42 {
		t.Errorf("Int string: %v %v", v, err)
	}
	if v, err := z.Coerce.Int().Parse(7); err != nil || v != 7 {
		t.Errorf("Int passthrough: %v %v", v, err)
	}
	_, err = z.Coerce.Int().Parse("4.5")
	if is := coverAOneIssue(t, err); is.Code != z.InvalidType {
		t.Errorf("Int non-integer string: %+v", is)
	}

	for _, tc := range []struct {
		in   any
		want string
	}{{"123456789012345678901234567890", "123456789012345678901234567890"}, {int64(5), "5"}, {json.Number("77"), "77"}, {9, "9"}} {
		v, err := z.Coerce.BigInt().Parse(tc.in)
		if err != nil || v.String() != tc.want {
			t.Errorf("BigInt(%v) = %v %v", tc.in, v, err)
		}
	}
	if _, err := z.Coerce.BigInt().Parse("x"); err == nil {
		t.Errorf("BigInt must reject non numeric string")
	}
	if v, err := z.Coerce.BigInt().Parse(big.NewInt(3)); err != nil || v.Int64() != 3 {
		t.Errorf("BigInt *big.Int: %v %v", v, err)
	}
}

// @spec REQ-PRIM-008
func TestCoverACoerceDate(t *testing.T) {
	ref := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	tests := []struct {
		name string
		in   any
		want time.Time
	}{
		{"time", ref, ref},
		{"rfc3339", "2024-05-06T07:08:09Z", ref},
		{"millis string", "1714979289000", time.UnixMilli(1714979289000)},
		{"int seconds", 1714979289, time.Unix(1714979289, 0)},
		{"int millis", 1714979289000, time.UnixMilli(1714979289000)},
		{"int64 seconds", int64(1714979289), time.Unix(1714979289, 0)},
		{"int64 millis", int64(1714979289000), time.UnixMilli(1714979289000)},
		{"float64 millis", float64(1714979289000), time.UnixMilli(1714979289000)},
	}
	s := z.Coerce.Date()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.Parse(tc.in)
			if err != nil || !got.Equal(tc.want) {
				t.Errorf("got (%v, %v), want %v", got, err, tc.want)
			}
		})
	}
	for _, in := range []any{"garbage", true, nil} {
		_, err := s.Parse(in)
		if is := coverAOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "date" {
			t.Errorf("Parse(%#v) issue: %+v", in, is)
		}
	}
}

type coverAColor string

// @spec REQ-COLL-001
func TestCoverAEnumParse(t *testing.T) {
	s := z.Enum("red", "green")
	if v, err := s.Parse("green"); err != nil || v != "green" {
		t.Fatalf("got %v %v", v, err)
	}
	_, err := s.Parse("blue")
	is := coverAOneIssue(t, err)
	if is.Code != z.InvalidValue || !slices.Equal(is.Values, []any{"red", "green"}) {
		t.Errorf("unexpected issue %+v", is)
	}
	_, err = s.Parse(5)
	if is := coverAOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "string" {
		t.Errorf("unexpected issue %+v", is)
	}

	// Named string type: both plain strings and already-typed values are accepted.
	c := z.Enum(coverAColor("red"), coverAColor("blue"))
	if v, err := c.Parse(coverAColor("blue")); err != nil || v != coverAColor("blue") {
		t.Errorf("typed input: %v %v", v, err)
	}
	if v, err := c.Parse("red"); err != nil || v != coverAColor("red") {
		t.Errorf("string input: %v %v", v, err)
	}
	if _, err := c.Parse(coverAColor("pink")); err == nil {
		t.Errorf("typed invalid value must fail")
	}
}

// @spec REQ-COLL-001
func TestCoverAEnumExtractExclude(t *testing.T) {
	s := z.Enum("a", "b", "c", "d")
	if got := s.Values(); !slices.Equal(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("Values: %v", got)
	}
	got := s.Values()
	got[0] = "zzz"
	if s.Values()[0] != "a" {
		t.Errorf("Values must return a copy")
	}
	if got := s.Extract("b", "d", "x").Values(); !slices.Equal(got, []string{"b", "d"}) {
		t.Errorf("Extract: %v", got)
	}
	if got := s.Exclude("b", "d").Values(); !slices.Equal(got, []string{"a", "c"}) {
		t.Errorf("Exclude: %v", got)
	}
	if _, err := s.Exclude("a").Parse("a"); err == nil {
		t.Errorf("excluded value must fail")
	}
	if _, err := s.Extract("a").Parse("b"); err == nil {
		t.Errorf("non extracted value must fail")
	}
	// Extract/Exclude preserve metadata.
	m := s.Meta(func(b *z.MetaBuilder) { b.Title("letters") })
	if m.Extract("a").GetMeta().Title != "letters" || m.Exclude("a").GetMeta().Title != "letters" {
		t.Errorf("meta not preserved by Extract/Exclude")
	}
}

// @spec REQ-COLL-006
func TestCoverAEnumModifiers(t *testing.T) {
	s := z.Enum("x", "y")
	if o, err := s.Optional().Parse("x"); err != nil || !o.IsSome() {
		t.Errorf("optional: %v %v", o, err)
	}
	if n, err := s.Nullable().Parse(nil); err != nil || !n.IsNull() {
		t.Errorf("nullable: %v %v", n, err)
	}
	_ = s.Nullish()
	if v, err := s.Default("y").Parse(nil); err != nil || v != "y" {
		t.Errorf("default: %v %v", v, err)
	}
	if v, err := s.Array().Parse([]any{"x", "y"}); err != nil || len(v) != 2 {
		t.Errorf("array: %v %v", v, err)
	}
	m := s.Meta(func(b *z.MetaBuilder) { b.Title("xy").Describe("two") })
	if mm := m.GetMeta(); mm.Title != "xy" || mm.Description != "two" {
		t.Errorf("meta: %+v", mm)
	}
	if _, err := m.Parse("y"); err != nil {
		t.Errorf("meta clone must still validate: %v", err)
	}
}

// @spec REQ-PIPE-007
func TestCoverAEnumErrorOverrides(t *testing.T) {
	s := z.Enum("x")
	_, err := s.Error("custom enum").Parse("q")
	if m := coverAOneIssue(t, err).Message; m != "custom enum" {
		t.Errorf("Error: %q", m)
	}
	_, err = s.ErrorFunc(func(*z.Issue) string { return "fn enum" }).Parse("q")
	if m := coverAOneIssue(t, err).Message; m != "fn enum" {
		t.Errorf("ErrorFunc: %q", m)
	}
}

// @spec REQ-COLL-001
func TestCoverAEnumJSONSchema(t *testing.T) {
	b, err := z.ToJSONSchema(z.Enum("p", "q"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	enum, _ := doc["enum"].([]any)
	if len(enum) != 2 || enum[0] != "p" || enum[1] != "q" {
		t.Errorf("unexpected schema: %s", b)
	}
}
