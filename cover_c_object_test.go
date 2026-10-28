package z_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func coverCObj() *z.ObjectSchema {
	return z.Object(func(s *z.Shape) {
		s.Field("a", z.String())
		s.Field("b", z.Int())
	})
}

// @spec REQ-OBJ-001
func TestCoverCObjectDuplicateFieldPanics(t *testing.T) {
	coverCMustPanic(t, `duplicate field "a"`, func() {
		z.Object(func(s *z.Shape) {
			s.Field("a", z.String())
			s.Field("a", z.Int())
		})
	})
}

// @spec REQ-OBJ-001 REQ-OBJ-003
func TestCoverCObjectParse(t *testing.T) {
	obj := z.Object(func(s *z.Shape) {
		s.Field("req", z.String())
		s.Field("opt", z.String().Optional())
		s.Field("nul", z.String().Nullish())
		s.Field("def", z.String().Default("dflt"))
	})

	t.Run("nil input", func(t *testing.T) {
		_, err := obj.Parse(nil)
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "object" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("non map input", func(t *testing.T) {
		_, err := obj.Parse("x")
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "object" || is[0].Input != "x" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("absent optional/nullish omitted, default applied", func(t *testing.T) {
		out, err := obj.Parse(map[string]any{"req": "r"})
		if err != nil {
			t.Fatal(err)
		}
		want := z.Dyn{"req": "r", "def": "dflt"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got %v want %v", out, want)
		}
	})
	t.Run("present values", func(t *testing.T) {
		out, err := obj.Parse(map[string]any{"req": "r", "opt": "o", "def": "d", "extra": 1})
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := out["opt"].(z.Option[string]); !ok || v.OrElse("") != "o" {
			t.Fatalf("opt: %#v", out["opt"])
		}
		if out["def"] != "d" || out["req"] != "r" {
			t.Fatalf("got %v", out)
		}
		if _, has := out["extra"]; has {
			t.Fatal("unknown key must be stripped")
		}
	})
	t.Run("missing required reports path", func(t *testing.T) {
		_, err := obj.Parse(map[string]any{})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || is[0].Expected != "required" || coverCPath(is[0]) != "req" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("invalid present value reports path", func(t *testing.T) {
		_, err := obj.Parse(map[string]any{"req": 5})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || coverCPath(is[0]) != "req" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("all issues collected without abort early", func(t *testing.T) {
		o := coverCObj()
		_, err := o.Parse(map[string]any{"a": 1, "b": "x"})
		if is := coverCIssues(t, err); len(is) != 2 {
			t.Fatalf("expected 2 issues, got %+v", is)
		}
		_, err = o.Parse(map[string]any{})
		if is := coverCIssues(t, err); len(is) != 2 {
			t.Fatalf("expected 2 issues, got %+v", is)
		}
	})
}

// @spec REQ-OBJ-001 REQ-CORE-017
func TestCoverCObjectAbortEarly(t *testing.T) {
	ctx := z.WithAbortEarly(context.Background())
	o := coverCObj()

	t.Run("missing required stops", func(t *testing.T) {
		_, err := o.Parse(map[string]any{}, ctx)
		if is := coverCIssues(t, err); len(is) != 1 || coverCPath(is[0]) != "a" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("invalid value stops", func(t *testing.T) {
		_, err := o.Parse(map[string]any{"a": 1, "b": "x"}, ctx)
		if is := coverCIssues(t, err); len(is) != 1 || coverCPath(is[0]) != "a" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("catchall failure stops after the first bad key", func(t *testing.T) {
		c := o.Catchall(z.Int())
		_, err := c.Parse(map[string]any{"a": "x", "b": 1, "x1": "bad", "x2": "bad", "x3": "bad"}, ctx)
		is := coverCIssues(t, err)
		if len(is) != 1 {
			t.Fatalf("abort early must yield a single issue, got %+v", is)
		}
		if p := coverCPath(is[0]); p != "x1" && p != "x2" && p != "x3" {
			t.Fatalf("unexpected path %q", p)
		}
	})
}

// @spec REQ-OBJ-001 REQ-CORE-017
func TestCoverCObjectCancelledMidParse(t *testing.T) {
	cctx, cancel := context.WithCancel(context.Background())
	canceller := z.NewTestSchema[string](func(any, any) (any, bool) {
		cancel()
		return "ok", true
	})
	o := z.Object(func(s *z.Shape) {
		s.Field("a", canceller)
		s.Field("b", z.String())
	})
	_, err := o.Parse(map[string]any{"a": "x", "b": "y"}, cctx)
	if err == nil {
		t.Fatal("expected error after cancellation")
	}
}

// @spec REQ-OBJ-001 REQ-OBJ-002
func TestCoverCObjectUnknownKeyModes(t *testing.T) {
	in := func() map[string]any { return map[string]any{"a": "x", "b": 1, "z": 1, "y": 2} }

	t.Run("strict sorts unrecognized keys", func(t *testing.T) {
		_, err := coverCObj().Strict().Parse(in())
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.UnrecognizedKeys || !reflect.DeepEqual(is[0].Keys, []string{"y", "z"}) {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("StrictObject constructor", func(t *testing.T) {
		o := z.StrictObject(func(s *z.Shape) { s.Field("a", z.String()) })
		if _, err := o.Parse(map[string]any{"a": "x", "q": 1}); err == nil {
			t.Fatal("expected strict failure")
		}
	})
	t.Run("loose keeps unknown keys", func(t *testing.T) {
		out, err := coverCObj().Loose().Parse(in())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(out, z.Dyn{"a": "x", "b": 1, "z": 1, "y": 2}) {
			t.Fatalf("got %v", out)
		}
	})
	t.Run("LooseObject constructor", func(t *testing.T) {
		o := z.LooseObject(func(s *z.Shape) { s.Field("a", z.String()) })
		out, err := o.Parse(map[string]any{"a": "x", "q": 1})
		if err != nil || out["q"] != 1 {
			t.Fatalf("got %v %v", out, err)
		}
	})
	t.Run("strip restores default", func(t *testing.T) {
		out, err := coverCObj().Strict().Strip().Parse(in())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(out, z.Dyn{"a": "x", "b": 1}) {
			t.Fatalf("got %v", out)
		}
	})
	t.Run("catchall validates unknown keys", func(t *testing.T) {
		c := coverCObj().Catchall(z.Int())
		out, err := c.Parse(map[string]any{"a": "x", "b": 1, "n": 7})
		if err != nil || out["n"] != 7 {
			t.Fatalf("got %v %v", out, err)
		}
		_, err = c.Parse(map[string]any{"a": "x", "b": 1, "n": "bad"})
		is := coverCIssues(t, err)
		if len(is) != 1 || is[0].Code != z.InvalidType || coverCPath(is[0]) != "n" {
			t.Fatalf("unexpected issues %+v", is)
		}
	})
	t.Run("catchall with nil schema ignores unknown keys", func(t *testing.T) {
		out, err := coverCObj().Catchall(nil).Parse(in())
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(out, z.Dyn{"a": "x", "b": 1}) {
			t.Fatalf("got %v", out)
		}
	})
}

// @spec REQ-OBJ-002 REQ-OBJ-009
func TestCoverCObjectShapeOps(t *testing.T) {
	base := coverCObj()

	t.Run("accessors", func(t *testing.T) {
		if got := base.FieldNames(); !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Fatalf("got %v", got)
		}
		if base.FieldSchema("a") == nil {
			t.Fatal("expected schema for a")
		}
		if base.FieldSchema("nope") != nil {
			t.Fatal("expected nil schema for unknown field")
		}
		if got := base.Keyof().Values(); !reflect.DeepEqual(got, []string{"a", "b"}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("extend replaces and appends, inherits mode, leaves original intact", func(t *testing.T) {
		ext := base.Strict().Extend(func(s *z.Shape) {
			s.Field("a", z.Int())
			s.Field("c", z.Boolean())
		})
		if got := ext.FieldNames(); !reflect.DeepEqual(got, []string{"a", "b", "c"}) {
			t.Fatalf("got %v", got)
		}
		out, err := ext.Parse(map[string]any{"a": 1, "b": 2, "c": true})
		if err != nil || out["a"] != 1 || out["c"] != true {
			t.Fatalf("got %v %v", out, err)
		}
		if _, err = ext.Parse(map[string]any{"a": 1, "b": 2, "c": true, "d": 1}); err == nil {
			t.Fatal("strict mode must be inherited")
		}
		if _, err = base.Parse(map[string]any{"a": "s", "b": 1}); err != nil {
			t.Fatalf("original schema must be unchanged: %v", err)
		}
	})

	t.Run("pick", func(t *testing.T) {
		p := base.Pick("b")
		if got := p.FieldNames(); !reflect.DeepEqual(got, []string{"b"}) {
			t.Fatalf("got %v", got)
		}
		coverCMustPanic(t, `unknown key "zz" in .Pick()`, func() { base.Pick("zz") })
	})

	t.Run("omit", func(t *testing.T) {
		o := base.Omit("a")
		if got := o.FieldNames(); !reflect.DeepEqual(got, []string{"b"}) {
			t.Fatalf("got %v", got)
		}
		coverCMustPanic(t, `unknown key "zz" in .Omit()`, func() { base.Omit("zz") })
	})

	t.Run("partial all keys", func(t *testing.T) {
		p := base.Partial()
		out, err := p.Parse(map[string]any{})
		if err != nil || len(out) != 0 {
			t.Fatalf("got %v %v", out, err)
		}
		out, err = p.Parse(map[string]any{"a": "x"})
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := out["a"].(z.Option[any]); !ok || v.OrElse(nil) != "x" {
			t.Fatalf("a: %#v", out["a"])
		}
		if _, err = p.Parse(map[string]any{"a": 1}); err == nil {
			t.Fatal("inner validation must still apply")
		}
	})

	t.Run("partial selected keys", func(t *testing.T) {
		p := base.Partial("a")
		if _, err := p.Parse(map[string]any{"b": 1}); err != nil {
			t.Fatalf("a must be optional: %v", err)
		}
		_, err := p.Parse(map[string]any{"a": "x"})
		is := coverCIssues(t, err)
		if len(is) != 1 || coverCPath(is[0]) != "b" {
			t.Fatalf("b must remain required: %+v", is)
		}
		coverCMustPanic(t, `unknown key "zz" in .Partial()`, func() { base.Partial("zz") })
	})

	t.Run("partial on already optional field keeps it", func(t *testing.T) {
		o := z.Object(func(s *z.Shape) { s.Field("o", z.String().Optional()) }).Partial()
		out, err := o.Parse(map[string]any{"o": "v"})
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := out["o"].(z.Option[string]); !ok || v.OrElse("") != "v" {
			t.Fatalf("o: %#v", out["o"])
		}
	})

	t.Run("partial wrapper exposes inner meta", func(t *testing.T) {
		withMeta := z.String().Meta(func(m *z.MetaBuilder) { m.Title("coverC title") })
		o := z.Object(func(s *z.Shape) { s.Field("t", withMeta) }).Partial()
		if got := o.FieldSchema("t").GetMeta().Title; got != "coverC title" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("required all keys after partial", func(t *testing.T) {
		r := base.Partial().Required()
		_, err := r.Parse(map[string]any{})
		if is := coverCIssues(t, err); len(is) != 2 {
			t.Fatalf("expected 2 required issues, got %+v", is)
		}
		if _, err = r.Parse(map[string]any{"a": "x", "b": 1}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("required selected keys", func(t *testing.T) {
		r := base.Partial().Required("b")
		_, err := r.Parse(map[string]any{})
		is := coverCIssues(t, err)
		if len(is) != 1 || coverCPath(is[0]) != "b" {
			t.Fatalf("unexpected issues %+v", is)
		}
		coverCMustPanic(t, `unknown key "zz" in .Required()`, func() { base.Required("zz") })
	})

	t.Run("required on non partial field leaves schema as is", func(t *testing.T) {
		r := base.Required("a")
		if _, err := r.Parse(map[string]any{"a": "x", "b": 1}); err != nil {
			t.Fatal(err)
		}
	})
}

// @spec REQ-OBJ-009 REQ-CORE-010 REQ-PIPE-007
func TestCoverCObjectModifiers(t *testing.T) {
	base := coverCObj()
	valid := map[string]any{"a": "x", "b": 1}

	if _, err := base.Optional().Parse(nil); err == nil {
		t.Fatal("Optional rejects null")
	}
	if out, err := base.Optional().Parse(valid); err != nil || !out.IsSome() {
		t.Fatalf("Optional: %v %v", out, err)
	}
	if out, err := base.Nullable().Parse(nil); err != nil || !out.IsNull() {
		t.Fatalf("Nullable: %v %v", out, err)
	}
	if out, err := base.Nullish().Parse(valid); err != nil {
		t.Fatalf("Nullish: %v %v", out, err)
	}
	if out, err := base.Default(z.Dyn{"d": 1}).Parse(nil); err != nil || out["d"] != 1 {
		t.Fatalf("Default: %v %v", out, err)
	}
	arr, err := base.Array().Parse([]any{valid, valid})
	if err != nil || len(arr) != 2 {
		t.Fatalf("Array: %v %v", arr, err)
	}

	withMeta := base.Meta(func(m *z.MetaBuilder) { m.ID("coverC_object_meta").Title("T") })
	if m := withMeta.GetMeta(); m.ID != "coverC_object_meta" || m.Title != "T" {
		t.Fatalf("meta: %+v", m)
	}
	if base.GetMeta().ID != "" {
		t.Fatal("Meta must not mutate the receiver")
	}

	msg := base.Error("custom msg")
	_, err = msg.Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "custom msg" {
		t.Fatalf("got %q", is[0].Message)
	}
	fn := base.ErrorFunc(func(i *z.Issue) string { return "fn:" + string(i.Code) })
	_, err = fn.Parse(1)
	if is := coverCIssues(t, err); is[0].Message != "fn:invalid_type" {
		t.Fatalf("got %q", is[0].Message)
	}
}

// @spec REQ-OBJ-009 REQ-OBJ-002
func TestCoverCObjectNodeAccessorsViaExport(t *testing.T) {
	cat := coverCObj().Catchall(z.Int())
	js, err := z.ToJSONSchema(cat)
	if err != nil || !strings.Contains(string(js), `"additionalProperties":{"type":"integer"}`) {
		t.Fatalf("%s %v", js, err)
	}
	strict, err := z.ToJSONSchema(coverCObj().Strict())
	if err != nil || !strings.Contains(string(strict), `"additionalProperties":false`) {
		t.Fatalf("%s %v", strict, err)
	}
}
