package z_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type coverDOptDoc struct {
	A z.Option[int]             `json:"a,omitzero"`
	B z.Nullable[string]        `json:"b"`
	C z.Option[z.Nullable[int]] `json:"c,omitzero"`
}

// @spec REQ-CORE-004
func TestCoverDOptionBasics(t *testing.T) {
	s := z.Some(5)
	n := z.None[int]()

	if !s.IsSome() || n.IsSome() {
		t.Fatal("IsSome mismatch")
	}
	if v, ok := s.Value(); !ok || v != 5 {
		t.Fatalf("got %v %v", v, ok)
	}
	if v, ok := n.Value(); ok || v != 0 {
		t.Fatalf("got %v %v", v, ok)
	}
	if s.OrElse(9) != 5 || n.OrElse(9) != 9 {
		t.Fatal("OrElse mismatch")
	}
	if s.IsZero() || !n.IsZero() {
		t.Fatal("IsZero mismatch")
	}
	if s.String() != "Some(5)" || n.String() != "None" {
		t.Fatalf("String: %q %q", s.String(), n.String())
	}
}

// @spec REQ-CORE-004
func TestCoverDNullableBasics(t *testing.T) {
	v := z.NullableOf("x")
	n := z.NullableNull[string]()

	if v.IsNull() || !n.IsNull() {
		t.Fatal("IsNull mismatch")
	}
	if got, ok := v.Value(); !ok || got != "x" {
		t.Fatalf("got %v %v", got, ok)
	}
	if got, ok := n.Value(); ok || got != "" {
		t.Fatalf("got %v %v", got, ok)
	}
	if v.OrElse("f") != "x" || n.OrElse("f") != "f" {
		t.Fatal("OrElse mismatch")
	}
	if v.String() != "Nullable(x)" || n.String() != "Null" {
		t.Fatalf("String: %q %q", v.String(), n.String())
	}
}

// @spec REQ-CORE-004
func TestCoverDOptionJSONLegacy(t *testing.T) {
	// MarshalJSON / UnmarshalJSON (encoding/json v1 style) called directly.
	b, err := z.Some(7).MarshalJSON()
	if err != nil || string(b) != "7" {
		t.Fatalf("got %s %v", b, err)
	}
	b, err = z.None[int]().MarshalJSON()
	if err != nil || string(b) != "null" {
		t.Fatalf("got %s %v", b, err)
	}
	if _, err := z.Some(make(chan int)).MarshalJSON(); err == nil {
		t.Fatal("expected marshal error for unsupported inner value")
	}

	var o z.Option[int]
	if err := o.UnmarshalJSON([]byte("12")); err != nil {
		t.Fatal(err)
	}
	if v, ok := o.Value(); !ok || v != 12 {
		t.Fatalf("got %v %v", v, ok)
	}
	if err := o.UnmarshalJSON([]byte(`"nope"`)); err == nil {
		t.Fatal("expected type error")
	}

	var on z.Option[z.Nullable[int]]
	if err := on.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatal(err)
	}
	if inner, ok := on.Value(); !ok || !inner.IsNull() {
		t.Fatalf("null into Option[Nullable] must be Some(Null): %v", on)
	}
}

// @spec REQ-CORE-004
func TestCoverDNullableJSONLegacy(t *testing.T) {
	b, err := z.NullableOf("a").MarshalJSON()
	if err != nil || string(b) != `"a"` {
		t.Fatalf("got %s %v", b, err)
	}
	b, err = z.NullableNull[string]().MarshalJSON()
	if err != nil || string(b) != "null" {
		t.Fatalf("got %s %v", b, err)
	}

	n := z.NullableOf(3)
	if err := n.UnmarshalJSON([]byte("  null ")); err != nil || !n.IsNull() {
		t.Fatalf("got %v %v", n, err)
	}
	if v, _ := n.Value(); v != 0 {
		t.Fatalf("null must reset the value, got %v", v)
	}
	if err := n.UnmarshalJSON([]byte("8")); err != nil || n.IsNull() {
		t.Fatalf("got %v %v", n, err)
	}
	if v, ok := n.Value(); !ok || v != 8 {
		t.Fatalf("got %v %v", v, ok)
	}
	if err := n.UnmarshalJSON([]byte(`"x"`)); err == nil {
		t.Fatal("expected type error")
	}
}

// @spec REQ-CORE-004
func TestCoverDOptionNullableJSONv2RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		doc  coverDOptDoc
		json string
	}{
		{"all absent", coverDOptDoc{B: z.NullableNull[string]()}, `{"b":null}`},
		{"values", coverDOptDoc{A: z.Some(1), B: z.NullableOf("s"), C: z.Some(z.NullableOf(2))}, `{"a":1,"b":"s","c":2}`},
		{"some null", coverDOptDoc{A: z.Some(0), B: z.NullableNull[string](), C: z.Some(z.NullableNull[int]())}, `{"a":0,"b":null,"c":null}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := json.Marshal(c.doc)
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != c.json {
				t.Fatalf("marshal: got %s want %s", out, c.json)
			}
			var back coverDOptDoc
			if err := json.Unmarshal(out, &back); err != nil {
				t.Fatal(err)
			}
			if back != c.doc {
				t.Fatalf("round trip: got %+v want %+v", back, c.doc)
			}
		})
	}

	// MarshalJSONTo: explicitly none (without omitzero) encodes as null.
	out, err := json.Marshal(struct {
		A z.Option[int] `json:"a"`
	}{})
	if err != nil || string(out) != `{"a":null}` {
		t.Fatalf("got %s %v", out, err)
	}

	// Decode errors surface through the streaming interface.
	var bad coverDOptDoc
	if err := json.Unmarshal([]byte(`{"a":"x"}`), &bad); err == nil {
		t.Fatal("expected error decoding string into Option[int]")
	}
	if err := json.Unmarshal([]byte(`{"b":5}`), &bad); err == nil {
		t.Fatal("expected error decoding number into Nullable[string]")
	}

	// Encoding an unsupported inner value reports the error.
	if _, err := json.Marshal(z.Some(make(chan int))); err == nil {
		t.Fatal("expected marshal error")
	}
	if _, err := json.Marshal(z.NullableOf(make(chan int))); err == nil {
		t.Fatal("expected marshal error")
	}
}

// @spec REQ-CORE-004
func TestCoverDNullableUnmarshalJSONFromDecoderErrors(t *testing.T) {
	// A bare `null` token in object-name position makes ReadToken fail after PeekKind reports 'n'.
	dec := jsontext.NewDecoder(strings.NewReader(`{null:1}`))
	if _, err := dec.ReadToken(); err != nil {
		t.Fatal(err)
	}
	var n z.Nullable[int]
	if err := n.UnmarshalJSONFrom(dec); err == nil {
		t.Fatal("expected read error from misplaced null")
	}

	// Malformed value after a non-null peek surfaces the decode error.
	dec = jsontext.NewDecoder(strings.NewReader(`"str"`))
	if err := n.UnmarshalJSONFrom(dec); err == nil {
		t.Fatal("expected type error")
	}
}
