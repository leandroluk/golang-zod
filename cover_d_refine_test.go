package z_test

import (
	"context"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// coverDFlipCtx has a nil Done channel (so checkCancel never notices) but Err flips after tripped is set.
type coverDFlipCtx struct {
	context.Context
	tripped bool
}

func (c *coverDFlipCtx) Done() <-chan struct{} { return nil }
func (c *coverDFlipCtx) Err() error {
	if c.tripped {
		return context.Canceled
	}
	return nil
}

// @spec REQ-PIPE-003
func TestCoverDRefineOptions(t *testing.T) {
	even := func(n int) bool { return n%2 == 0 }

	t.Run("defaults to custom code and message", func(t *testing.T) {
		_, err := z.Refine[int](z.Int(), even).Parse(3)
		iss := coverDIssues(t, err)
		if len(iss) != 1 || iss[0].Code != z.Custom || iss[0].Message == "" {
			t.Fatalf("unexpected issues: %+v", iss)
		}
		if v, err := z.Refine[int](z.Int(), even).Parse(4); err != nil || v != 4 {
			t.Fatalf("got %v %v", v, err)
		}
	})
	t.Run("message code path and message func", func(t *testing.T) {
		s := z.Refine[int](z.Int(), even,
			z.RefineMessage("static"),
			z.RefineCode(z.InvalidValue),
			z.RefinePath(z.Property("a"), z.Index(2)),
			z.RefineMessageFunc(func(iss *z.Issue) string { return "dyn:" + string(iss.Code) }),
		)
		_, err := s.Parse(3)
		iss := coverDIssues(t, err)
		if len(iss) != 1 {
			t.Fatalf("unexpected issues: %+v", iss)
		}
		if iss[0].Code != z.InvalidValue || iss[0].Message != "dyn:invalid_value" {
			t.Fatalf("unexpected issue: %+v", iss[0])
		}
		if len(iss[0].Path) != 2 || iss[0].Path[0].Key() != "a" || iss[0].Path[1].IndexVal() != 2 {
			t.Fatalf("unexpected path: %+v", iss[0].Path)
		}
	})
	t.Run("static message", func(t *testing.T) {
		_, err := z.Refine[int](z.Int(), even, z.RefineMessage("must be even")).Parse(1)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Message != "must be even" {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("RefineAbort option", func(t *testing.T) {
		s := z.Refine[int](z.Int(), even, z.RefineAbort(true), z.RefineMessage("m"))
		if _, err := s.Parse(1); err == nil {
			t.Fatal("expected failure")
		}
	})
}

// @spec REQ-PIPE-003
func TestCoverDRefineChaining(t *testing.T) {
	t.Run("all failing steps reported", func(t *testing.T) {
		s := z.NewRefinedSchema[int](z.Int()).
			Refine(func(n int) bool { return n > 10 }, z.RefineMessage("gt10")).
			Check(func(n int) []z.Issue {
				return []z.Issue{{Code: z.Custom, Message: "c1"}, {Code: z.Custom, Message: "c2"}}
			}).
			SuperRefine(func(n int, c *z.RefineCtx) { c.AddIssue(z.Issue{Code: z.Custom, Message: "s1"}) })
		_, err := s.Parse(1)
		iss := coverDIssues(t, err)
		var msgs []string
		for _, i := range iss {
			msgs = append(msgs, i.Message)
		}
		want := []string{"gt10", "c1", "c2", "s1"}
		if len(msgs) != len(want) {
			t.Fatalf("got %v", msgs)
		}
		for i := range want {
			if msgs[i] != want[i] {
				t.Fatalf("got %v want %v", msgs, want)
			}
		}
	})
	t.Run("passing checks", func(t *testing.T) {
		s := z.NewRefinedSchema[int](z.Int()).
			Check(func(n int) []z.Issue { return nil }).
			SuperRefine(func(n int, c *z.RefineCtx) {})
		if v, err := s.Parse(7); err != nil || v != 7 {
			t.Fatalf("got %v %v", v, err)
		}
		if v, err := z.SuperRefine[int](z.Int(), func(n int, c *z.RefineCtx) {}).Parse(1); err != nil || v != 1 {
			t.Fatalf("got %v %v", v, err)
		}
	})
	t.Run("abort early stops at first failing step", func(t *testing.T) {
		calls := 0
		s := z.NewRefinedSchema[int](z.Int()).
			Refine(func(n int) bool { return false }, z.RefineMessage("one")).
			Refine(func(n int) bool { calls++; return false }, z.RefineMessage("two"))
		ctx := context.WithValue(context.Background(), z.AbortEarlyContextKey, true)
		_, err := s.Parse(1, ctx)
		iss := coverDIssues(t, err)
		if len(iss) != 1 || iss[0].Message != "one" || calls != 0 {
			t.Fatalf("issues=%+v calls=%d", iss, calls)
		}
	})
	t.Run("inner failure skips steps", func(t *testing.T) {
		called := false
		s := z.Refine[int](z.Int(), func(int) bool { called = true; return true })
		_, err := s.Parse("x")
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.InvalidType || called {
			t.Fatalf("issues=%+v called=%v", iss, called)
		}
	})
	t.Run("inner type mismatch skips predicate", func(t *testing.T) {
		called := false
		s := z.Refine[string](coverDBadString(), func(string) bool { called = true; return false })
		_, _ = s.Parse("x")
		if called {
			t.Fatal("predicate must not run on mismatched type")
		}
	})
	t.Run("cancelled between steps", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		second := false
		s := z.NewRefinedSchema[int](z.Int()).
			Refine(func(int) bool { cancel(); return true }).
			Refine(func(int) bool { second = true; return true })
		if _, err := s.Parse(1, ctx); err == nil || second {
			t.Fatalf("expected failure without running second step: err=%v second=%v", err, second)
		}
	})
}

// @spec REQ-PIPE-003
func TestCoverDRefineCtx(t *testing.T) {
	t.Run("receives parse context", func(t *testing.T) {
		type key struct{}
		ctx := context.WithValue(context.Background(), key{}, "v")
		var got any
		s := z.RefineWithCtx[int](z.Int(), func(c context.Context, n int) bool { got = c.Value(key{}); return n > 0 })
		if v, err := s.Parse(1, ctx); err != nil || v != 1 || got != "v" {
			t.Fatalf("got %v %v ctxval=%v", v, err, got)
		}
	})
	t.Run("failure with options", func(t *testing.T) {
		s := z.NewRefinedSchema[int](z.Int()).RefineCtx(
			func(context.Context, int) bool { return false },
			z.RefineMessage("static"),
			z.RefineCode(z.InvalidValue),
			z.RefinePath(z.Property("p")),
			z.RefineMessageFunc(func(iss *z.Issue) string { return "dyn" }),
		)
		_, err := s.Parse(1)
		iss := coverDIssues(t, err)
		if len(iss) != 1 || iss[0].Code != z.InvalidValue || iss[0].Message != "dyn" ||
			len(iss[0].Path) != 1 || iss[0].Path[0].Key() != "p" {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("failure with default code", func(t *testing.T) {
		s := z.RefineWithCtx[int](z.Int(), func(context.Context, int) bool { return false }, z.RefineMessage("nope"))
		_, err := s.Parse(1)
		if iss := coverDIssues(t, err); len(iss) != 1 || iss[0].Code != z.Custom || iss[0].Message != "nope" {
			t.Fatalf("unexpected issues: %+v", iss)
		}
	})
	t.Run("context error detected inside step", func(t *testing.T) {
		fc := &coverDFlipCtx{Context: context.Background()}
		ran := false
		s := z.NewRefinedSchema[int](z.Int()).
			Refine(func(int) bool { fc.tripped = true; return true }).
			RefineCtx(func(context.Context, int) bool { ran = true; return true })
		_, err := s.Parse(1, fc)
		iss := coverDIssues(t, err)
		if len(iss) != 1 || iss[0].Code != z.Custom || iss[0].Message != "cancelled" || ran {
			t.Fatalf("issues=%+v ran=%v", iss, ran)
		}
	})
}

// @spec REQ-PIPE-003 REQ-PIPE-001
func TestCoverDRefinedWrappers(t *testing.T) {
	r := z.Refine[string](z.String(), func(s string) bool { return s != "bad" }, z.RefineMessage("bad"))

	opt := r.Optional()
	if v, err := opt.Parse("ok"); err != nil || v.OrElse("") != "ok" {
		t.Fatalf("got %v %v", v, err)
	}
	if _, err := opt.Parse("bad"); err == nil {
		t.Fatal("expected refinement failure through Optional")
	}

	nl := r.Nullable()
	if v, err := nl.Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("got %v %v", v, err)
	}
	nsh := r.Nullish()
	if v, err := nsh.Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("got %v %v", v, err)
	}
	def := r.Default("dflt")
	if v, err := def.Parse(nil); err != nil || v != "dflt" {
		t.Fatalf("got %v %v", v, err)
	}
	arr := r.Array()
	if v, err := arr.Parse([]any{"a", "b"}); err != nil || len(v) != 2 {
		t.Fatalf("got %v %v", v, err)
	}
	_, err := arr.Parse([]any{"a", "bad"})
	iss := coverDIssues(t, err)
	if len(iss) != 1 || iss[0].Message != "bad" || len(iss[0].Path) != 1 || iss[0].Path[0].IndexVal() != 1 {
		t.Fatalf("unexpected issues: %+v", iss)
	}
}
