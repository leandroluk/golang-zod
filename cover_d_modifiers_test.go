package z_test

import (
	"errors"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func coverDIssues(t *testing.T, err error) []z.Issue {
	t.Helper()
	var ze *z.ZodError
	if !errors.As(err, &ze) {
		t.Fatalf("expected *ZodError, got %T (%v)", err, err)
	}
	return ze.Issues()
}

// coverDBadString yields a Schema[string] whose node returns an int (type mismatch) successfully.
func coverDBadString() z.Schema[string] {
	return z.NewTestSchema[string](func(ctx any, in any) (any, bool) { return 42, true })
}

// @spec REQ-PIPE-001 REQ-PRIM-009
func TestCoverDOptional(t *testing.T) {
	s := z.String().Min(2).Optional()

	v, err := s.Parse("ab")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := v.Value(); !ok || got != "ab" {
		t.Fatalf("got %v", v)
	}

	_, err = s.Parse(nil)
	iss := coverDIssues(t, err)
	if len(iss) != 1 || iss[0].Code != z.InvalidType || iss[0].Expected != "non-null" {
		t.Fatalf("unexpected issues: %+v", iss)
	}

	_, err = s.Parse("a")
	iss = coverDIssues(t, err)
	if len(iss) != 1 || iss[0].Code != z.TooSmall {
		t.Fatalf("unexpected issues: %+v", iss)
	}

	if _, err = z.NewOptionalSchema[string](coverDBadString()).Parse("x"); err == nil {
		t.Fatal("expected failure on inner type mismatch")
	}

	inner := z.String()
	if z.NewOptionalSchema[string](inner).Unwrap() != z.Schema[string](inner) {
		t.Fatal("Unwrap must return the inner schema")
	}
}

// @spec REQ-PIPE-001 REQ-PRIM-009
func TestCoverDNullable(t *testing.T) {
	s := z.NewNullableSchema[string](z.String().Min(2))

	v, err := s.Parse(nil)
	if err != nil || !v.IsNull() {
		t.Fatalf("got %v %v", v, err)
	}
	v, err = s.Parse("ab")
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := v.Value(); !ok || got != "ab" {
		t.Fatalf("got %v", v)
	}
	_, err = s.Parse("a")
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
		t.Fatalf("unexpected issues: %+v", iss)
	}
	if _, err = z.NewNullableSchema[string](coverDBadString()).Parse("x"); err == nil {
		t.Fatal("expected failure on inner type mismatch")
	}
	inner := z.String()
	if z.NewNullableSchema[string](inner).Unwrap() != z.Schema[string](inner) {
		t.Fatal("Unwrap must return the inner schema")
	}
}

// @spec REQ-PIPE-001 REQ-PRIM-009
func TestCoverDNullish(t *testing.T) {
	s := z.NewNullishSchema[string](z.String().Min(2))

	v, err := s.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.Value(); !ok || !n.IsNull() {
		t.Fatalf("got %v", v)
	}
	v, err = s.Parse("ab")
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := v.Value(); !ok || n.OrElse("") != "ab" {
		t.Fatalf("got %v", v)
	}
	_, err = s.Parse("a")
	if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
		t.Fatalf("unexpected issues: %+v", iss)
	}
	if _, err = z.NewNullishSchema[string](coverDBadString()).Parse("x"); err == nil {
		t.Fatal("expected failure on inner type mismatch")
	}
	inner := z.String()
	if z.NewNullishSchema[string](inner).Unwrap() != z.Schema[string](inner) {
		t.Fatal("Unwrap must return the inner schema")
	}
}

// @spec REQ-PIPE-001
func TestCoverDDefaultPrefaultCatchReadonly(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		calls := 0
		s := z.NewDefaultFuncSchema[string](z.String().Min(5), func() string { calls++; return "x" })
		v, err := s.Parse(nil)
		if err != nil || v != "x" || calls != 1 {
			t.Fatalf("got %q %v calls=%d (default must short-circuit)", v, err, calls)
		}
		if v, err = s.Parse("hello"); err != nil || v != "hello" {
			t.Fatalf("got %q %v", v, err)
		}
		if _, err = s.Parse("a"); err == nil {
			t.Fatal("expected failure")
		}
		if z.NewDefaultSchema[string](z.String(), "d").Unwrap() == nil {
			t.Fatal("nil unwrap")
		}
	})
	t.Run("prefault", func(t *testing.T) {
		s := z.NewPrefaultSchema[string](z.String().Min(5), "abc")
		_, err := s.Parse(nil)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.TooSmall {
			t.Fatalf("prefault value must run through inner: %+v", iss)
		}
		s2 := z.NewPrefaultFuncSchema[string](z.String().Min(2), func() string { return "abc" })
		if v, err := s2.Parse(nil); err != nil || v != "abc" {
			t.Fatalf("got %q %v", v, err)
		}
		if v, err := s2.Parse("zzz"); err != nil || v != "zzz" {
			t.Fatalf("got %q %v", v, err)
		}
		if s2.Unwrap() == nil {
			t.Fatal("nil unwrap")
		}
	})
	t.Run("catch", func(t *testing.T) {
		var seen []z.Issue
		s := z.NewCatchFuncSchema[string](z.String().Min(3), func(is []z.Issue) string { seen = is; return "fb" })
		if v, err := s.Parse("abcd"); err != nil || v != "abcd" || seen != nil {
			t.Fatalf("got %q %v %v", v, err, seen)
		}
		if v, err := s.Parse("a"); err != nil || v != "fb" {
			t.Fatalf("got %q %v", v, err)
		}
		if len(seen) != 1 || seen[0].Code != z.TooSmall {
			t.Fatalf("catch must receive issues: %+v", seen)
		}
		// failure that adds no issue (inner returns ok=false silently)
		silent := z.NewTestSchema[string](func(ctx any, in any) (any, bool) { return nil, false })
		seen = []z.Issue{{}}
		if v, err := z.NewCatchFuncSchema[string](silent, func(is []z.Issue) string { seen = is; return "q" }).Parse(1); err != nil || v != "q" || len(seen) != 0 {
			t.Fatalf("got %q %v %v", v, err, seen)
		}
		if v, err := z.NewCatchSchema[string](z.String(), "const").Parse(5); err != nil || v != "const" {
			t.Fatalf("got %q %v", v, err)
		}
		if s.Unwrap() == nil {
			t.Fatal("nil unwrap")
		}
	})
	t.Run("readonly", func(t *testing.T) {
		inner := z.String()
		s := z.NewReadonlySchema[string](inner)
		if !s.GetMeta().ReadOnly {
			t.Fatal("expected ReadOnly meta")
		}
		if v, err := s.Parse("ok"); err != nil || v != "ok" {
			t.Fatalf("got %q %v", v, err)
		}
		if _, err := s.Parse(1); err == nil {
			t.Fatal("expected failure")
		}
		if s.Unwrap() != z.Schema[string](inner) {
			t.Fatal("Unwrap must return inner")
		}
	})
}

// Nodes report their kind when used as object fields and exporters unwrap them.
// @spec REQ-PIPE-001 REQ-PRIM-009
func TestCoverDModifierKindsInObject(t *testing.T) {
	obj := z.Object(func(s *z.Shape) {
		s.Field("opt", z.NewOptionalSchema[string](z.String()))
		s.Field("nul", z.NewNullableSchema[string](z.String()))
		s.Field("nsh", z.NewNullishSchema[string](z.String()))
		s.Field("def", z.NewDefaultSchema[string](z.String(), "d"))
		s.Field("pre", z.NewPrefaultSchema[string](z.String(), "p"))
		s.Field("cat", z.NewCatchSchema[string](z.String(), "c"))
		s.Field("ro", z.NewReadonlySchema[string](z.String()))
	})
	out, err := obj.Parse(map[string]any{"nul": nil, "ro": "r", "pre": nil, "cat": 5})
	if err != nil {
		t.Fatal(err)
	}
	if out["def"] != "d" || out["pre"] != "p" || out["cat"] != "c" || out["ro"] != "r" {
		t.Fatalf("unexpected output: %v", out)
	}

	js, err := z.ToJSONSchema(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(js), `"properties"`) {
		t.Fatalf("unexpected json schema: %s", js)
	}
	sdl, err := z.ToGraphQLSDL(obj)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sdl, "opt: String") || !strings.Contains(sdl, "nsh: String") || !strings.Contains(sdl, "nul: String") {
		t.Fatalf("unexpected sdl: %s", sdl)
	}
}
