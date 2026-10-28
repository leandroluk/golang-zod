package z_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-PIPE-004
func TestCoverDTransform(t *testing.T) {
	tr := z.Transform[string, int](z.String(), func(s string) (int, error) {
		if s == "boom" {
			return 99, errors.New("exploded")
		}
		return len(s), nil
	})

	if v, err := tr.Parse("abc"); err != nil || v != 3 {
		t.Fatalf("got %v %v", v, err)
	}

	v, err := tr.Parse("boom")
	iss := coverDIssues(t, err)
	if v != 0 || len(iss) != 1 || iss[0].Code != z.Custom || iss[0].Message != "exploded" {
		t.Fatalf("v=%v issues=%+v", v, iss)
	}

	_, err = tr.Parse(5)
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.InvalidType {
		t.Fatalf("inner failure must propagate: %+v", iss)
	}

	// inner node yields a value of the wrong type
	mis := z.Transform[string, int](coverDBadString(), func(s string) (int, error) { return 1, nil })
	_, err = mis.Parse("x")
	iss = coverDIssues(t, err)
	if len(iss) != 1 || iss[0].Code != z.Custom || !strings.Contains(iss[0].Message, "transform input type mismatch: expected string, got int") {
		t.Fatalf("unexpected issues: %+v", iss)
	}

	if tr.Input() == nil {
		t.Fatal("Input must expose inner schema")
	}
}

// @spec REQ-PIPE-004
func TestCoverDPipe(t *testing.T) {
	toInt := z.Transform[string, int](z.String(), func(s string) (int, error) { return len(s), nil })
	p := z.Pipe[int, int](toInt, z.Int().Min(3))

	if v, err := p.Parse("abcd"); err != nil || v != 4 {
		t.Fatalf("got %v %v", v, err)
	}
	_, err := p.Parse("ab")
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
		t.Fatalf("right side must validate left output: %+v", iss)
	}
	_, err = p.Parse(10)
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.InvalidType {
		t.Fatalf("left failure must stop pipe: %+v", iss)
	}
	if p.Input() == nil || p.Output() == nil {
		t.Fatal("Input/Output must be exposed")
	}
}

// @spec REQ-PIPE-004
func TestCoverDPreprocess(t *testing.T) {
	s := z.Preprocess[int](func(raw any) any {
		if str, ok := raw.(string); ok {
			return len(str)
		}
		return raw
	}, z.Int())
	if v, err := s.Parse("hello"); err != nil || v != 5 {
		t.Fatalf("got %v %v", v, err)
	}
	if v, err := s.Parse(7); err != nil || v != 7 {
		t.Fatalf("got %v %v", v, err)
	}
	if _, err := s.Parse(true); err == nil {
		t.Fatal("expected failure")
	}
}

// Kinds are consulted when these schemas are object fields.
// @spec REQ-PIPE-003 REQ-PIPE-004 REQ-PIPE-005
func TestCoverDPipelineKindsInObject(t *testing.T) {
	obj := z.Object(func(s *z.Shape) {
		s.Field("tr", z.Transform[string, int](z.String(), func(s string) (int, error) { return len(s), nil }))
		s.Field("pp", z.Pipe[string, string](z.String(), z.String().Min(1)))
		s.Field("rf", z.Refine[string](z.String(), func(string) bool { return true }))
		s.Field("cd", z.Codecs.StringInt())
		s.Field("pr", z.Preprocess[string](func(raw any) any { return raw }, z.String()))
	})
	out, err := obj.Parse(map[string]any{"tr": "abc", "pp": "x", "rf": "y", "cd": "12", "pr": "p"})
	if err != nil {
		t.Fatal(err)
	}
	if out["tr"] != 3 || out["pp"] != "x" || out["rf"] != "y" || out["cd"] != 12 {
		t.Fatalf("unexpected output: %v", out)
	}
	if _, err := z.ToJSONSchema(obj, z.WithUnrepresentable(z.UnrepresentableAny)); err != nil {
		t.Fatal(err)
	}
}

// @spec REQ-PIPE-005
func TestCoverDCodecDecode(t *testing.T) {
	c := z.Codec[string, int](z.String(), z.Int().Min(10),
		func(s string) (int, error) {
			if s == "err" {
				return 0, errors.New("bad input")
			}
			return len(s), nil
		},
		func(i int) (string, error) { return strings.Repeat("x", i), nil },
	)

	if v, err := c.DecodeValue("abcdefghijk"); err != nil || v != 11 {
		t.Fatalf("got %v %v", v, err)
	}
	if v, err := c.Parse("abcdefghijk"); err != nil || v != 11 {
		t.Fatalf("Parse must behave like Decode: %v %v", v, err)
	}
	_, err := c.DecodeValue("err")
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.Custom || iss[0].Message != "bad input" {
		t.Fatalf("unexpected issues: %+v", iss)
	}
	_, err = c.DecodeValue("abc")
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
		t.Fatalf("outSchema must validate decoded value: %+v", iss)
	}
	_, err = c.DecodeValue(5)
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.InvalidType {
		t.Fatalf("inSchema failure: %+v", iss)
	}

	// inSchema produces wrong Go type
	mis := z.Codec[string, int](coverDBadString(), z.Int(),
		func(s string) (int, error) { return 1, nil },
		func(i int) (string, error) { return "", nil })
	_, err = mis.DecodeValue("x")
	if iss := coverDIssues(t, err); len(iss) != 1 || !strings.Contains(iss[0].Message, "codec input type mismatch: expected string, got int") {
		t.Fatalf("unexpected issues: %+v", iss)
	}

	// cancellation observed inside the codec node
	ctx, cancel := context.WithCancel(context.Background())
	gate := z.Refine[string](z.String(), func(string) bool { cancel(); return true })
	piped := z.Pipe[string, int](gate, c)
	_, err = piped.Parse("abcdefghijk", ctx)
	var ze *z.ZodError
	if !errors.As(err, &ze) || len(ze.Issues()) != 0 {
		t.Fatalf("expected silent failure on cancellation, got %v", err)
	}
}

// @spec REQ-PIPE-005
func TestCoverDCodecEncode(t *testing.T) {
	newCodec := func(out z.Schema[int], enc func(int) (string, error)) *z.CodecSchema[string, int] {
		return z.Codec[string, int](z.String().Min(2), out,
			func(s string) (int, error) { return len(s), nil }, enc)
	}
	repeat := func(i int) (string, error) { return strings.Repeat("x", i), nil }

	t.Run("success", func(t *testing.T) {
		c := newCodec(z.Int().Min(1), repeat)
		if s, err := c.Encode(3); err != nil || s != "xxx" {
			t.Fatalf("got %q %v", s, err)
		}
		// nil outSchema skips the first validation pass
		c2 := newCodec(nil, repeat)
		if s, err := c2.Encode(3); err != nil || s != "xxx" {
			t.Fatalf("got %q %v", s, err)
		}
	})
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := newCodec(nil, repeat).Encode(3, ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("one-way codec", func(t *testing.T) {
		if _, err := newCodec(nil, nil).Encode(3); !errors.Is(err, z.ErrEncodeUnsupported) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("out schema rejects value", func(t *testing.T) {
		_, err := newCodec(z.Int().Min(5), repeat).Encode(3)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("out schema fails silently", func(t *testing.T) {
		silent := z.NewTestSchema[int](func(ctx any, in any) (any, bool) { return nil, false })
		_, err := newCodec(silent, repeat).Encode(3)
		if iss := coverDIssues(t, err); len(iss) != 0 {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("encode function error", func(t *testing.T) {
		boom := errors.New("encode boom")
		_, err := newCodec(nil, func(int) (string, error) { return "", boom }).Encode(3)
		if !errors.Is(err, boom) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("in schema rejects encoded value", func(t *testing.T) {
		_, err := newCodec(nil, repeat).Encode(1) // "x" violates Min(2)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("in schema fails silently", func(t *testing.T) {
		silent := z.NewTestSchema[string](func(ctx any, in any) (any, bool) { return nil, false })
		c := z.Codec[string, int](silent, nil, func(s string) (int, error) { return 0, nil }, repeat)
		_, err := c.Encode(2)
		if iss := coverDIssues(t, err); len(iss) != 0 {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
}

// @spec REQ-PIPE-006
func TestCoverDStringBool(t *testing.T) {
	type tc struct {
		name string
		opts []z.StringBoolOption
		in   string
		want bool
		fail bool
	}
	cases := []tc{
		{name: "default true upper", in: "YES", want: true},
		{name: "default true on", in: "on", want: true},
		{name: "default false", in: "0", want: false},
		{name: "default false mixed", in: "Off", want: false},
		{name: "default invalid", in: "maybe", fail: true},
		{name: "custom truthy", opts: []z.StringBoolOption{z.StringBoolTruthy("Sim")}, in: "sim", want: true},
		{name: "custom truthy replaces default", opts: []z.StringBoolOption{z.StringBoolTruthy("sim")}, in: "yes", fail: true},
		{name: "custom falsy", opts: []z.StringBoolOption{z.StringBoolFalsy("Nao")}, in: "NAO", want: false},
		{name: "case sensitive miss", opts: []z.StringBoolOption{z.StringBoolCaseSensitive(true)}, in: "TRUE", fail: true},
		{name: "case sensitive hit", opts: []z.StringBoolOption{z.StringBoolCaseSensitive(true)}, in: "true", want: true},
		{name: "case sensitive falsy hit", opts: []z.StringBoolOption{z.StringBoolCaseSensitive(true)}, in: "no", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v, err := z.StringBool(c.opts...).Parse(c.in)
			if c.fail {
				if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.Custom || !strings.Contains(iss[0].Message, "invalid boolean string") {
					t.Fatalf("unexpected issues: %+v", iss)
				}
				return
			}
			if err != nil || v != c.want {
				t.Fatalf("got %v %v", v, err)
			}
		})
	}

	enc := []struct {
		name string
		opts []z.StringBoolOption
		in   bool
		want string
	}{
		{"default true", nil, true, "true"},
		{"default false", nil, false, "false"},
		{"custom first truthy", []z.StringBoolOption{z.StringBoolTruthy("sim", "s")}, true, "sim"},
		{"custom first falsy", []z.StringBoolOption{z.StringBoolFalsy("nao", "n")}, false, "nao"},
		{"empty truthy falls back", []z.StringBoolOption{z.StringBoolTruthy()}, true, "true"},
		{"empty falsy falls back", []z.StringBoolOption{z.StringBoolFalsy()}, false, "false"},
	}
	for _, e := range enc {
		t.Run("encode "+e.name, func(t *testing.T) {
			got, err := z.StringBool(e.opts...).Encode(e.in)
			if err != nil || got != e.want {
				t.Fatalf("got %q %v", got, err)
			}
		})
	}
}

// @spec REQ-PIPE-008
func TestCoverDCodecRecipes(t *testing.T) {
	t.Run("JSONString", func(t *testing.T) {
		c := z.Codecs.JSONString()
		v, err := c.Parse(`{"a":[1,2]}`)
		if err != nil {
			t.Fatal(err)
		}
		m, ok := v.(map[string]any)
		if !ok || len(m["a"].([]any)) != 2 {
			t.Fatalf("got %#v", v)
		}
		_, err = c.Parse(`{nope`)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.Custom {
			t.Fatalf("unexpected issues: %+v", iss)
		}
		s, err := c.Encode(map[string]any{"k": "v"})
		if err != nil || s != `{"k":"v"}` {
			t.Fatalf("got %q %v", s, err)
		}
		if _, err := c.Encode(make(chan int)); err == nil {
			t.Fatal("expected marshal error")
		}
		if _, err := c.Encode(nil); err == nil || strings.Contains(err.Error(), "unsupported") {
			// nil marshals to "null", which is valid and must succeed
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
		}
	})
	t.Run("Base64Bytes", func(t *testing.T) {
		c := z.Codecs.Base64Bytes()
		v, err := c.Parse("aGk=")
		if err != nil || string(v) != "hi" {
			t.Fatalf("got %q %v", v, err)
		}
		if _, err := c.Parse("***"); err == nil {
			t.Fatal("expected error")
		}
		if s, err := c.Encode([]byte("hi")); err != nil || s != "aGk=" {
			t.Fatalf("got %q %v", s, err)
		}
	})
	t.Run("StringInt", func(t *testing.T) {
		c := z.Codecs.StringInt()
		if v, err := c.Parse("42"); err != nil || v != 42 {
			t.Fatalf("got %v %v", v, err)
		}
		if _, err := c.Parse("4x"); err == nil {
			t.Fatal("expected error")
		}
		if s, err := c.Encode(7); err != nil || s != "7" {
			t.Fatalf("got %q %v", s, err)
		}
	})
	t.Run("StringNumber", func(t *testing.T) {
		c := z.Codecs.StringNumber()
		if v, err := c.Parse("1.5"); err != nil || v != 1.5 {
			t.Fatalf("got %v %v", v, err)
		}
		if _, err := c.Parse("abc"); err == nil {
			t.Fatal("expected error")
		}
		if s, err := c.Encode(2.25); err != nil || s != "2.25" {
			t.Fatalf("got %q %v", s, err)
		}
	})
	t.Run("EpochDate", func(t *testing.T) {
		c := z.Codecs.EpochDate()
		v, err := c.Parse(int64(1_700_000_000))
		if err != nil || v.Unix() != 1_700_000_000 {
			t.Fatalf("seconds: got %v %v", v, err)
		}
		v, err = c.Parse(int64(1_700_000_000_123))
		if err != nil || v.UnixMilli() != 1_700_000_000_123 {
			t.Fatalf("millis: got %v %v", v, err)
		}
		if _, err := c.Parse("x"); err == nil {
			t.Fatal("expected error")
		}
		ms, err := c.Encode(time.UnixMilli(1_700_000_000_123))
		if err != nil || ms != 1_700_000_000_123 {
			t.Fatalf("got %v %v", ms, err)
		}
	})
}

// @spec REQ-PIPE-005 REQ-PIPE-008
func TestCoverDCodecJSONSchema(t *testing.T) {
	c := z.Codecs.StringInt()
	in, err := z.ToJSONSchema(c, z.WithIO(z.Input))
	if err != nil || !strings.Contains(string(in), `"string"`) {
		t.Fatalf("input schema: %s %v", in, err)
	}
	out, err := z.ToJSONSchema(c, z.WithIO(z.Output))
	if err != nil || strings.Contains(string(out), `"string"`) {
		t.Fatalf("output schema: %s %v", out, err)
	}
	// codec without out schema still exports (falls through without type)
	if _, err := z.ToJSONSchema(z.Codecs.JSONString()); err != nil {
		t.Fatal(err)
	}
}
