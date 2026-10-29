package z_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-CORE-006
func TestCoverEZodErrorBasics(t *testing.T) {
	var nilErr *z.ZodError
	if nilErr.Issues() != nil {
		t.Error("nil ZodError must have nil issues")
	}
	if nilErr.Error() != "" {
		t.Error("nil ZodError must have empty message")
	}
	if z.NewZodError(nil).Error() != "" {
		t.Error("empty ZodError must have empty message")
	}

	ze := z.NewZodError([]z.Issue{{Code: z.Custom, Message: "boom", Path: []z.PathSegment{z.Property("a"), z.Index(2)}}})
	if got := ze.Error(); got != "✖ boom\n  → at a[2]" {
		t.Errorf("Error() = %q", got)
	}

	p, i := z.Property("k"), z.Index(7)
	if p.IsIndex() || p.Key() != "k" || p.String() != "k" {
		t.Errorf("bad property segment: %+v", p)
	}
	if !i.IsIndex() || i.IndexVal() != 7 || i.String() != "7" {
		t.Errorf("bad index segment: %+v", i)
	}
}

// @spec REQ-CORE-007
func TestCoverEFormatters(t *testing.T) {
	var nilErr *z.ZodError
	if f := nilErr.Flatten(); len(f.FormErrors) != 0 || len(f.FieldErrors) != 0 || f.FormErrors == nil || f.FieldErrors == nil {
		t.Errorf("nil Flatten must be empty but non-nil: %+v", f)
	}
	if !reflect.DeepEqual(nilErr.Treeify(), z.ErrorTreeNode{}) {
		t.Error("nil Treeify must be empty")
	}
	if nilErr.Prettify() != "" || z.NewZodError(nil).Prettify() != "" {
		t.Error("empty Prettify must be empty string")
	}

	ze := z.NewZodError([]z.Issue{
		{Code: z.TooSmall, Message: "deep", Path: []z.PathSegment{z.Property("users"), z.Index(0), z.Property("email")}},
		{Code: z.InvalidType, Message: "form level"},
		{Code: z.Custom, Message: "users bad", Path: []z.PathSegment{z.Property("users")}},
		{Code: z.InvalidValue, Message: "", Path: []z.PathSegment{z.Property("x")}},
		{Code: z.Custom, Message: "item err", Path: []z.PathSegment{z.Property("users"), z.Index(0)}},
		{Code: z.Custom, Message: "item err 2", Path: []z.PathSegment{z.Property("users"), z.Index(0)}},
	})

	flat := ze.Flatten()
	wantFlat := z.FlattenedErrors{
		FormErrors: []string{"form level"},
		FieldErrors: map[string][]string{
			"users": {"deep", "users bad", "item err", "item err 2"},
			"x":     {""},
		},
	}
	if !reflect.DeepEqual(flat, wantFlat) {
		t.Errorf("Flatten = %+v\nwant %+v", flat, wantFlat)
	}

	tree := ze.Treeify()
	wantTree := z.ErrorTreeNode{
		Errors: []string{"form level"},
		Properties: map[string]z.ErrorTreeNode{
			"users": {
				Errors: []string{"users bad"},
				Items: map[int]z.ErrorTreeNode{
					0: {
						Errors: []string{"item err", "item err 2"},
						Properties: map[string]z.ErrorTreeNode{
							"email": {Errors: []string{"deep"}},
						},
					},
				},
			},
			"x": {Errors: []string{""}},
		},
	}
	if !reflect.DeepEqual(tree, wantTree) {
		t.Errorf("Treeify = %+v\nwant %+v", tree, wantTree)
	}

	// Sorted by path length (stable); empty message falls back to the code.
	want := strings.Join([]string{
		"✖ form level",
		"✖ users bad",
		"  → at users",
		"✖ invalid_value",
		"  → at x",
		"✖ item err",
		"  → at users[0]",
		"✖ item err 2",
		"  → at users[0]",
		"✖ deep",
		"  → at users[0].email",
	}, "\n")
	if got := ze.Prettify(); got != want {
		t.Errorf("Prettify =\n%s\nwant\n%s", got, want)
	}
}

// @spec REQ-CORE-002 REQ-CORE-003 REQ-CORE-017
func TestCoverESchemaEntryPoints(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	sch := z.String()
	if _, err := sch.Decode(strings.NewReader(`"hi"`), cancelled); !errors.Is(err, context.Canceled) {
		t.Errorf("Decode with cancelled ctx: %v", err)
	}
	if _, err := sch.ParseJSON([]byte(`"hi"`), cancelled); !errors.Is(err, context.Canceled) {
		t.Errorf("ParseJSON with cancelled ctx: %v", err)
	}
	if _, err := sch.Parse("hi", cancelled); !errors.Is(err, context.Canceled) {
		t.Errorf("Parse with cancelled ctx: %v", err)
	}

	if v, err := sch.Decode(strings.NewReader(`"hi"`)); err != nil || v != "hi" {
		t.Errorf("Decode = %q, %v", v, err)
	}
	if _, err := sch.Decode(strings.NewReader(`{"a":`)); err == nil {
		t.Error("Decode of truncated JSON must fail")
	}
	if _, err := sch.ParseJSON([]byte(`[`)); err == nil {
		t.Error("ParseJSON of invalid JSON must fail")
	}
	var ze *z.ZodError
	if _, err := sch.Decode(strings.NewReader(`1`)); !errors.As(err, &ze) {
		t.Errorf("Decode of wrong type must yield ZodError, got %v", err)
	}
	// nil context values are treated as absent.
	if v, err := sch.Parse("x", nil); err != nil || v != "x" { //nolint:staticcheck
		t.Errorf("Parse with nil ctx = %q, %v", v, err)
	}
}

// @spec REQ-CORE-002
func TestCoverEInternalTypeMismatch(t *testing.T) {
	bad := z.NewTestSchema[string](func(ctx any, in any) (any, bool) { return 123, true })
	_, err := bad.Parse("x")
	if err == nil || !strings.Contains(err.Error(), "failed to convert int to output type string") {
		t.Fatalf("expected conversion error, got %v", err)
	}
	var ze *z.ZodError
	if errors.As(err, &ze) {
		t.Fatal("conversion error must not be a ZodError")
	}

	nilOut := z.NewTestSchema[string](func(ctx any, in any) (any, bool) { return nil, true })
	if v, err := nilOut.Parse("x"); err != nil || v != "" {
		t.Fatalf("nil output must yield zero value, got %q %v", v, err)
	}
}

// @spec REQ-CORE-001 REQ-CORE-012 REQ-CORE-017
func TestCoverEParseCtxHelpers(t *testing.T) {
	// nil context is replaced by Background.
	pc := z.AcquireParseCtx(nil) //nolint:staticcheck
	if err := z.ContextCheckCancel(pc); err != nil {
		t.Errorf("background ctx cannot be cancelled: %v", err)
	}
	z.ReleaseParseCtx(pc)

	ctx, cancel := context.WithCancel(context.Background())
	pc = z.AcquireParseCtx(ctx)
	if err := z.ContextCheckCancel(pc); err != nil {
		t.Errorf("live ctx: %v", err)
	}
	cancel()
	if err := z.ContextCheckCancel(pc); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx: %v", err)
	}
	z.ReleaseParseCtx(pc)

	dctx, dcancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer dcancel()
	pc = z.AcquireParseCtx(dctx)
	if err := z.ContextCheckCancel(pc); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expired ctx: %v", err)
	}
	z.ReleaseParseCtx(pc)

	// Not a parseCtx: helpers are no-ops.
	z.ContextAddIssue("nope", z.Issue{})
	if z.ContextCheckCancel("nope") != nil || z.ContextIssuesCount("nope") != 0 {
		t.Error("foreign values must be ignored")
	}
	z.ReleaseParseCtx("nope")
	z.ReleaseParseCtx(nil)
}

// @spec REQ-CORE-006 REQ-CORE-012
func TestCoverEDeepPathSpill(t *testing.T) {
	// Paths deeper than the inline buffer (8 segments) must be preserved in order.
	const depth = 11
	var sch z.AnySchema = z.String()
	for i := 0; i < depth; i++ {
		inner := sch
		sch = z.Object(func(s *z.ObjectBuilder) { s.Field(fmt.Sprintf("f%d", i), inner) })
	}
	// Matching input: outermost key is f{depth-1}.
	in := any(5)
	for i := 0; i < depth; i++ {
		in = map[string]any{fmt.Sprintf("f%d", i): in}
	}

	obj := sch.(interface {
		Parse(any, ...context.Context) (z.Dyn, error)
	})
	_, err := obj.Parse(in)
	var ze *z.ZodError
	if !errors.As(err, &ze) || len(ze.Issues()) != 1 {
		t.Fatalf("expected one issue, got %v", err)
	}
	path := ze.Issues()[0].Path
	if len(path) != depth {
		t.Fatalf("path length = %d, want %d", len(path), depth)
	}
	for i, seg := range path {
		if want := fmt.Sprintf("f%d", depth-1-i); seg.Key() != want {
			t.Errorf("path[%d] = %q, want %q", i, seg.Key(), want)
		}
	}
}

// @spec REQ-CORE-010
func TestCoverEMetaBuilder(t *testing.T) {
	sch := z.String().Meta(func(m *z.MetaBuilder) {
		m.Title("T").Describe("D").Examples("a").Examples("b").ReadOnly(true).Deprecated(true).Custom("k", 1).Custom("j", 2)
	})
	m := sch.GetMeta()
	if m.Title != "T" || m.Description != "D" || !reflect.DeepEqual(m.Examples, []any{"a", "b"}) || !m.ReadOnly || !m.Deprecated ||
		!reflect.DeepEqual(m.Custom, map[string]any{"k": 1, "j": 2}) {
		t.Fatalf("unexpected meta: %+v", m)
	}

	// The builder copies current metadata: mutating a derived builder never leaks into the source.
	derived := sch.Meta(func(b *z.MetaBuilder) { b.Examples("c").Custom("k", 99).ReadOnly(false) })
	if got := derived.GetMeta(); !reflect.DeepEqual(got.Examples, []any{"a", "b", "c"}) || got.Custom["k"] != 99 || got.ReadOnly {
		t.Fatalf("derived meta wrong: %+v", got)
	}
	if got := sch.GetMeta(); len(got.Examples) != 2 || got.Custom["k"] != 1 || !got.ReadOnly {
		t.Fatalf("source meta mutated: %+v", got)
	}
}

// @spec REQ-CORE-010
func TestCoverESchemaIDLineage(t *testing.T) {
	base := z.String().Meta(func(m *z.MetaBuilder) { m.ID("coverE_lineage_base") })

	// Cloning through checks and modifiers of the same schema lineage must not panic.
	clone := base.Min(1).Max(10)
	if clone.GetMeta().ID != "coverE_lineage_base" {
		t.Fatalf("clone lost ID: %+v", clone.GetMeta())
	}
	if _, err := clone.Parse(""); err == nil {
		t.Fatal("clone must keep its own checks")
	}
	if _, err := base.Parse(""); err != nil {
		t.Fatal("base must be unaffected by clone checks")
	}
	re := clone.Meta(func(m *z.MetaBuilder) { m.Title("again").ID("coverE_lineage_base") })
	if re.GetMeta().Title != "again" {
		t.Fatal("re-applying the same ID on the same lineage must work")
	}

	mustPanic := func(name string, fn func()) {
		t.Helper()
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("%s: expected duplicate ID panic", name)
				return
			}
			if s, _ := r.(string); !strings.Contains(s, `duplicate schema ID "coverE_lineage_base"`) {
				t.Errorf("%s: unexpected panic %v", name, r)
			}
		}()
		fn()
	}
	// A different schema claiming the same ID still panics.
	mustPanic("other schema", func() {
		z.String().Meta(func(m *z.MetaBuilder) { m.ID("coverE_lineage_base") })
	})
	// A clone that renames its ID gets a fresh registration, which clashes if taken.
	mustPanic("rename into taken ID", func() {
		other := z.String().Meta(func(m *z.MetaBuilder) { m.ID("coverE_lineage_other") })
		other.Meta(func(m *z.MetaBuilder) { m.ID("coverE_lineage_base") })
	})
	// Reset discards the lineage as well.
	mustPanic("reset then reuse", func() {
		base.Meta(func(m *z.MetaBuilder) { m.Reset().ID("coverE_lineage_base") })
	})
}
