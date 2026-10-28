package z_test

import (
	"context"
	"errors"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-CORE-008 REQ-CORE-009
func TestCoverELocaleCatalogs(t *testing.T) {
	for _, tc := range coverELocaleCases {
		t.Run(tc.name, func(t *testing.T) {
			is := tc.issue
			if got := z.LocaleEN.FormatMessage(&is); got != tc.en {
				t.Errorf("en: got %q want %q", got, tc.en)
			}
			if got := z.LocaleES.FormatMessage(&is); got != tc.es {
				t.Errorf("es: got %q want %q", got, tc.es)
			}
			if got := z.LocalePtBR.FormatMessage(&is); got != tc.pt {
				t.Errorf("pt-BR: got %q want %q", got, tc.pt)
			}
		})
	}
}

// @spec REQ-CORE-009
func TestCoverELocaleTags(t *testing.T) {
	if z.LocaleEN.Tag() != "en" || z.LocaleES.Tag() != "es" || z.LocalePtBR.Tag() != "pt-BR" {
		t.Fatalf("unexpected tags: %q %q %q", z.LocaleEN.Tag(), z.LocaleES.Tag(), z.LocalePtBR.Tag())
	}
}

type coverECustomLocale struct{}

func (coverECustomLocale) Tag() string                     { return "coverE-xx" }
func (coverECustomLocale) FormatMessage(i *z.Issue) string { return "xx:" + string(i.Code) }

func coverEFirstMessage(t *testing.T, ctx context.Context) string {
	t.Helper()
	_, err := z.String().Parse(1, ctx)
	var ze *z.ZodError
	if !errors.As(err, &ze) {
		t.Fatalf("expected ZodError, got %v", err)
	}
	return ze.Issues()[0].Message
}

// @spec REQ-CORE-009
func TestCoverESetGetLocale(t *testing.T) {
	prev := z.GetLocale()
	t.Cleanup(func() { z.SetLocale(prev) })

	z.SetLocale(z.LocalePtBR)
	if z.GetLocale() != z.LocalePtBR {
		t.Fatal("SetLocale(pt-BR) not applied")
	}
	if got := coverEFirstMessage(t, context.Background()); got != "Esperado string" {
		t.Fatalf("global locale not used in parse: %q", got)
	}

	z.SetLocale(nil)
	if z.GetLocale() != z.LocaleEN {
		t.Fatal("SetLocale(nil) must fall back to LocaleEN")
	}
	if got := coverEFirstMessage(t, context.Background()); got != "Expected string" {
		t.Fatalf("got %q", got)
	}
}

// @spec REQ-CORE-013 REQ-CORE-014
func TestCoverEResolveLocale(t *testing.T) {
	z.RegisterLocale("  coverE-XX ", coverECustomLocale{})
	cases := []struct {
		name string
		in   string
		want z.Locale
	}{
		{"exact", "pt-BR", z.LocalePtBR},
		{"case-insensitive", "PT-br", z.LocalePtBR},
		{"language only", "es", z.LocaleES},
		{"dash prefix", "es-MX", z.LocaleES},
		{"unregistered region falls to base", "en-GB", z.LocaleEN},
		{"registered-tag prefix scan", "p", z.LocalePtBR},
		{"q ordering", "en;q=0.5, pt-BR;q=0.9", z.LocalePtBR},
		{"q ordering reversed input", "pt-BR;q=0.2,es;q=0.8", z.LocaleES},
		{"default q is 1", "en,pt;q=0.5", z.LocaleEN},
		{"invalid q ignored", "es;q=abc", z.LocaleES},
		{"unrelated params ignored", "es;level=1", z.LocaleES},
		{"empty parts skipped", " , ,es", z.LocaleES},
		{"fallback to second preference", "zz,pt-BR;q=0.5", z.LocalePtBR},
		{"custom registered locale", "coverE-xx", coverECustomLocale{}},
		{"unknown", "zz", nil},
		{"empty", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := z.ResolveLocale(tc.in); got != tc.want {
				t.Fatalf("ResolveLocale(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// @spec REQ-CORE-013 REQ-CORE-015
func TestCoverEWithLocale(t *testing.T) {
	base := context.Background()

	if got := coverEFirstMessage(t, z.WithLocale(base, "pt-BR,pt;q=0.9,en;q=0.8")); got != "Esperado string" {
		t.Errorf("pt-BR: %q", got)
	}
	//nolint:staticcheck // nil ctx is part of the contract
	if got := coverEFirstMessage(t, z.WithLocale(nil, "es-MX")); got != "Se esperaba string" {
		t.Errorf("nil ctx + es-MX: %q", got)
	}
	if z.WithLocale(base, "") != base || z.WithLocale(base, "   ") != base {
		t.Error("empty tag must return ctx unchanged")
	}
	if z.WithLocale(base, "zz") != base {
		t.Error("unmatched tag must return ctx unchanged")
	}
	if got := coverEFirstMessage(t, z.WithLocale(base, "zz")); got != "Expected string" {
		t.Errorf("unmatched tag should use global default: %q", got)
	}

	// Derived contexts keep the locale.
	child, cancel := context.WithCancel(z.WithLocale(base, "es"))
	defer cancel()
	if got := coverEFirstMessage(t, child); got != "Se esperaba string" {
		t.Errorf("child ctx: %q", got)
	}

	// A plain string stored under the well-known key is resolved like WithLocale's tag.
	strCtx := context.WithValue(base, z.LocaleContextKey, "pt-BR") //nolint:staticcheck // string key by design
	if got := coverEFirstMessage(t, strCtx); got != "Esperado string" {
		t.Errorf("string value: %q", got)
	}
	// An unresolvable string and a value of another type fall through to the global locale.
	badStr := context.WithValue(base, z.LocaleContextKey, "zz") //nolint:staticcheck
	if got := coverEFirstMessage(t, badStr); got != "Expected string" {
		t.Errorf("unresolvable string: %q", got)
	}
	badType := context.WithValue(base, z.LocaleContextKey, 42) //nolint:staticcheck
	if got := coverEFirstMessage(t, badType); got != "Expected string" {
		t.Errorf("wrong type: %q", got)
	}
	// A Locale value is used directly.
	locCtx := context.WithValue(base, z.LocaleContextKey, z.LocalePtBR) //nolint:staticcheck
	if got := coverEFirstMessage(t, locCtx); got != "Esperado string" {
		t.Errorf("Locale value: %q", got)
	}
}

type coverELocaleKey struct{}
type coverEAbortKey struct{}

// @spec REQ-CORE-016 REQ-CORE-011
func TestCoverEConfigUseContext(t *testing.T) {
	t.Cleanup(func() { z.Config.UseContext(z.ContextOptions{}) })

	z.Config.UseContext(z.ContextOptions{LocaleKey: coverELocaleKey{}, AbortEarlyKey: coverEAbortKey{}})

	// WithLocale / WithAbortEarly write to the replaced keys.
	ctx := z.WithLocale(context.Background(), "pt-BR")
	if ctx.Value(coverELocaleKey{}) != z.LocalePtBR {
		t.Fatalf("locale not stored under custom key")
	}
	if ctx.Value(z.LocaleContextKey) != nil {
		t.Fatalf("default key must not be written")
	}
	ctx = z.WithAbortEarly(ctx)
	if ctx.Value(coverEAbortKey{}) != true {
		t.Fatalf("abort-early not stored under custom key")
	}

	sch := z.Object(func(s *z.Shape) {
		s.Field("a", z.String())
		s.Field("b", z.String())
	})
	_, err := sch.Parse(map[string]any{"a": 1, "b": 2}, ctx)
	var ze *z.ZodError
	if !errors.As(err, &ze) || len(ze.Issues()) != 1 || ze.Issues()[0].Message != "Esperado string" {
		t.Fatalf("custom keys not honored: %v", err)
	}

	// The old default key is no longer consulted.
	old := context.WithValue(context.Background(), z.AbortEarlyContextKey, true) //nolint:staticcheck
	_, err = sch.Parse(map[string]any{"a": 1, "b": 2}, old)
	if !errors.As(err, &ze) || len(ze.Issues()) != 2 {
		t.Fatalf("default key should be ignored after UseContext: %v", err)
	}

	// Empty-string keys keep defaults.
	z.Config.UseContext(z.ContextOptions{LocaleKey: "", AbortEarlyKey: ""})
	if z.WithLocale(context.Background(), "es").Value(z.LocaleContextKey) != z.LocaleES {
		t.Fatal("empty LocaleKey must keep the default key")
	}
	if z.WithAbortEarly(context.Background()).Value(z.AbortEarlyContextKey) != true {
		t.Fatal("empty AbortEarlyKey must keep the default key")
	}
}

// @spec REQ-CORE-011
func TestCoverEAbortEarly(t *testing.T) {
	sch := z.Object(func(s *z.Shape) {
		s.Field("a", z.String())
		s.Field("b", z.String())
	})
	in := map[string]any{"a": 1, "b": 2}
	var ze *z.ZodError

	_, err := sch.Parse(in, z.WithAbortEarly(nil)) //nolint:staticcheck // nil ctx is part of the contract
	if !errors.As(err, &ze) || len(ze.Issues()) != 1 {
		t.Fatalf("abort early should stop at first issue: %v", err)
	}
	_, err = sch.Parse(in)
	if !errors.As(err, &ze) || len(ze.Issues()) != 2 {
		t.Fatalf("without abort early expected 2 issues: %v", err)
	}
	// A non-bool value under the key does not enable abort-early.
	ctx := context.WithValue(context.Background(), z.AbortEarlyContextKey, "yes") //nolint:staticcheck
	_, err = sch.Parse(in, ctx)
	if !errors.As(err, &ze) || len(ze.Issues()) != 2 {
		t.Fatalf("non-bool abort value must be ignored: %v", err)
	}
}
