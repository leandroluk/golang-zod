package z_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

func coverAIssues(t *testing.T, err error) []z.Issue {
	t.Helper()
	var ze *z.ZodError
	if !errors.As(err, &ze) {
		t.Fatalf("expected *ZodError, got %T (%v)", err, err)
	}
	return ze.Issues()
}

func coverAOneIssue(t *testing.T, err error) z.Issue {
	t.Helper()
	is := coverAIssues(t, err)
	if len(is) != 1 {
		t.Fatalf("expected exactly one issue, got %d: %v", len(is), is)
	}
	return is[0]
}

// @spec REQ-PRIM-001
func TestCoverAPrimitivesInvalidTypes(t *testing.T) {
	tests := []struct {
		name     string
		parse    func() error
		code     z.IssueCode
		expected string
	}{
		{"boolean", func() error { _, e := z.Boolean().Parse("x"); return e }, z.InvalidType, "boolean"},
		{"null", func() error { _, e := z.Null().Parse(1); return e }, z.InvalidType, "null"},
		{"never", func() error { _, e := z.Never().Parse(1); return e }, z.InvalidType, "never"},
		{"date bad string", func() error { _, e := z.Date().Parse("nope"); return e }, z.InvalidType, "date"},
		{"date bad type", func() error { _, e := z.Date().Parse(5); return e }, z.InvalidType, "date"},
		{"bytes bad base64", func() error { _, e := z.Bytes().Parse("***"); return e }, z.InvalidType, "bytes"},
		{"bytes bad type", func() error { _, e := z.Bytes().Parse(5); return e }, z.InvalidType, "bytes"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			is := coverAOneIssue(t, tc.parse())
			if is.Code != tc.code || is.Expected != tc.expected {
				t.Errorf("got code=%v expected=%q", is.Code, is.Expected)
			}
		})
	}
}

// @spec REQ-PRIM-001
func TestCoverAPrimitivesValidInputs(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if v, err := z.Date().Parse(now); err != nil || !v.Equal(now) {
		t.Errorf("time.Time passthrough: %v %v", v, err)
	}
	if v, err := z.Date().Parse("2024-01-02T03:04:05Z"); err != nil || !v.Equal(now) {
		t.Errorf("RFC3339: %v %v", v, err)
	}
	if v, err := z.Date().Parse("2024-01-02T03:04:05.123456789Z"); err != nil || v.Nanosecond() != 123456789 {
		t.Errorf("RFC3339Nano: %v %v", v, err)
	}
	if v, err := z.Bytes().Parse([]byte("hi")); err != nil || string(v) != "hi" {
		t.Errorf("bytes passthrough: %q %v", v, err)
	}
	if v, err := z.Bytes().Parse("aGk="); err != nil || string(v) != "hi" {
		t.Errorf("bytes base64: %q %v", v, err)
	}
}

// @spec REQ-PRIM-001
func TestCoverALiteral(t *testing.T) {
	s := z.Literal("a", "b")
	if v, err := s.Parse("b"); err != nil || v != "b" {
		t.Fatalf("got %v %v", v, err)
	}
	for _, in := range []any{"c", 1} {
		_, err := s.Parse(in)
		is := coverAOneIssue(t, err)
		if is.Code != z.InvalidValue || len(is.Values) != 2 || is.Values[0] != "a" || is.Values[1] != "b" {
			t.Errorf("unexpected issue %+v", is)
		}
	}
	vals := s.Values()
	vals[0] = "mutated"
	if got := s.Values(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("Values must return a copy, got %v", got)
	}
}

// @spec REQ-PRIM-009
func TestCoverAPrimitiveModifiers(t *testing.T) {
	// Boolean
	if o, err := z.Boolean().Optional().Parse(true); err != nil || !o.IsSome() {
		t.Errorf("boolean optional: %v %v", o, err)
	}
	if n, err := z.Boolean().Nullable().Parse(nil); err != nil || !n.IsNull() {
		t.Errorf("boolean nullable: %v %v", n, err)
	}
	if o, err := z.Boolean().Nullish().Parse(true); err != nil {
		t.Errorf("boolean nullish: %v %v", o, err)
	}
	if v, err := z.Boolean().Default(true).Parse(nil); err != nil || !v {
		t.Errorf("boolean default: %v %v", v, err)
	}
	if v, err := z.Boolean().Array().Parse([]any{true, false}); err != nil || len(v) != 2 {
		t.Errorf("boolean array: %v %v", v, err)
	}
	if m := z.Boolean().Meta(func(m *z.MetaBuilder) { m.Title("flag") }).GetMeta(); m.Title != "flag" {
		t.Errorf("boolean meta: %+v", m)
	}

	// Null
	_ = z.Null().Optional()
	_ = z.Null().Nullable()
	_ = z.Null().Nullish()
	if v, err := z.Null().Default(nil).Parse(nil); err != nil || v != nil {
		t.Errorf("null default: %v %v", v, err)
	}
	if v, err := z.Null().Array().Parse([]any{nil}); err != nil || len(v) != 1 {
		t.Errorf("null array: %v %v", v, err)
	}
	if m := z.Null().Meta(func(m *z.MetaBuilder) { m.Title("nil") }).GetMeta(); m.Title != "nil" {
		t.Errorf("null meta: %+v", m)
	}

	// Any
	_ = z.Any().Optional()
	_ = z.Any().Nullable()
	_ = z.Any().Nullish()
	if v, err := z.Any().Default("d").Parse(nil); err != nil || v != "d" {
		t.Errorf("any default: %v %v", v, err)
	}
	if v, err := z.Any().Array().Parse([]any{1, "x"}); err != nil || len(v) != 2 {
		t.Errorf("any array: %v %v", v, err)
	}
	if m := z.Any().Meta(func(m *z.MetaBuilder) { m.Title("anything") }).GetMeta(); m.Title != "anything" {
		t.Errorf("any meta: %+v", m)
	}

	// Never
	_ = z.Never().Nullable()
	_ = z.Never().Nullish()
	_ = z.Never().Optional()
	if m := z.Never().Meta(func(m *z.MetaBuilder) { m.Title("nope") }).GetMeta(); m.Title != "nope" {
		t.Errorf("never meta: %+v", m)
	}

	// Date
	d := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_ = z.Date().Optional()
	_ = z.Date().Nullable()
	_ = z.Date().Nullish()
	if v, err := z.Date().Default(d).Parse(nil); err != nil || !v.Equal(d) {
		t.Errorf("date default: %v %v", v, err)
	}
	if v, err := z.Date().Array().Parse([]any{d}); err != nil || len(v) != 1 {
		t.Errorf("date array: %v %v", v, err)
	}
	if m := z.Date().Meta(func(m *z.MetaBuilder) { m.Title("when") }).GetMeta(); m.Title != "when" {
		t.Errorf("date meta: %+v", m)
	}

	// Bytes
	_ = z.Bytes().Optional()
	_ = z.Bytes().Nullable()
	_ = z.Bytes().Nullish()
	if v, err := z.Bytes().Default([]byte("x")).Parse(nil); err != nil || string(v) != "x" {
		t.Errorf("bytes default: %q %v", v, err)
	}
	if v, err := z.Bytes().Array().Parse([]any{[]byte("a")}); err != nil || len(v) != 1 {
		t.Errorf("bytes array: %v %v", v, err)
	}
	if m := z.Bytes().Meta(func(m *z.MetaBuilder) { m.Title("blob") }).GetMeta(); m.Title != "blob" {
		t.Errorf("bytes meta: %+v", m)
	}

	// Literal
	l := z.Literal(1, 2)
	_ = l.Optional()
	_ = l.Nullable()
	_ = l.Nullish()
	if v, err := l.Default(2).Parse(nil); err != nil || v != 2 {
		t.Errorf("literal default: %v %v", v, err)
	}
	if v, err := l.Array().Parse([]any{1, 2}); err != nil || len(v) != 2 {
		t.Errorf("literal array: %v %v", v, err)
	}
	if m := l.Meta(func(m *z.MetaBuilder) { m.Title("one-or-two") }).GetMeta(); m.Title != "one-or-two" {
		t.Errorf("literal meta: %+v", m)
	}
}

// @spec REQ-PIPE-007
func TestCoverAPrimitiveErrorOverrides(t *testing.T) {
	fn := func(*z.Issue) string { return "fn-msg" }
	tests := []struct {
		name  string
		msg   func() error
		fnMsg func() error
	}{
		{"boolean",
			func() error { _, e := z.Boolean().Error("custom").Parse(1); return e },
			func() error { _, e := z.Boolean().ErrorFunc(fn).Parse(1); return e }},
		{"null",
			func() error { _, e := z.Null().Error("custom").Parse(1); return e },
			func() error { _, e := z.Null().ErrorFunc(fn).Parse(1); return e }},
		{"never",
			func() error { _, e := z.Never().Error("custom").Parse(1); return e },
			func() error { _, e := z.Never().ErrorFunc(fn).Parse(1); return e }},
		{"date",
			func() error { _, e := z.Date().Error("custom").Parse(1); return e },
			func() error { _, e := z.Date().ErrorFunc(fn).Parse(1); return e }},
		{"bytes",
			func() error { _, e := z.Bytes().Error("custom").Parse(1); return e },
			func() error { _, e := z.Bytes().ErrorFunc(fn).Parse(1); return e }},
		{"literal",
			func() error { _, e := z.Literal("a").Error("custom").Parse(1); return e },
			func() error { _, e := z.Literal("a").ErrorFunc(fn).Parse(1); return e }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if m := coverAOneIssue(t, tc.msg()).Message; m != "custom" {
				t.Errorf("Error: got %q", m)
			}
			if m := coverAOneIssue(t, tc.fnMsg()).Message; m != "fn-msg" {
				t.Errorf("ErrorFunc: got %q", m)
			}
		})
	}
	// Any never fails, but the override builders must still return usable schemas.
	if _, err := z.Any().Error("x").Parse(1); err != nil {
		t.Errorf("any with Error must still accept input: %v", err)
	}
	if _, err := z.Any().ErrorFunc(fn).Parse(1); err != nil {
		t.Errorf("any with ErrorFunc must still accept input: %v", err)
	}
	// Overrides must not mutate the original schema.
	base := z.Boolean()
	_ = base.Error("custom")
	_, err := base.Parse(1)
	if m := coverAOneIssue(t, err).Message; m == "custom" {
		t.Errorf("Error mutated original schema")
	}
}

// @spec REQ-PRIM-001
func TestCoverAKindsViaJSONSchema(t *testing.T) {
	// ToJSONSchema inspects every node kind; this exercises kind() of each primitive node.
	schemas := []z.AnySchema{
		z.Boolean(), z.Null(), z.Any(), z.Never(), z.Date(), z.Bytes(), z.Literal("a"),
		z.Coerce.String(), z.Coerce.Boolean(), z.Coerce.Float(), z.Coerce.Int(),
		z.Coerce.BigInt(), z.Coerce.Date(),
	}
	for _, s := range schemas {
		_, _ = z.ToJSONSchema(s, z.WithUnrepresentable(z.UnrepresentableAny))
	}
	b, err := z.ToJSONSchema(z.Boolean())
	if err != nil || len(b) == 0 {
		t.Fatalf("expected boolean JSON schema, got %q %v", b, err)
	}
}
