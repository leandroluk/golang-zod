package z_test

import (
	"reflect"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-GQL-003
func TestREQ_GQL_003_TypeNaming(t *testing.T) {
	user := z.Object(func(s *z.ObjectBuilder) {
		s.Field("address", z.Object(func(a *z.ObjectBuilder) {
			a.Field("city", z.String())
		}))
	}).Meta(func(m *z.MetaBuilder) { m.ID("TraceUser") })

	sdl, err := z.ToGraphQLSDL(user)
	if err != nil {
		t.Fatalf("ToGraphQLSDL: %v", err)
	}
	if !strings.Contains(sdl, "type TraceUser") {
		t.Fatalf("expected Meta.ID type name 'TraceUser', got:\n%s", sdl)
	}
	if !strings.Contains(sdl, "TraceUserAddress") {
		t.Fatalf("expected fallback name 'TraceUserAddress' for unnamed nested object, got:\n%s", sdl)
	}
}

// @spec REQ-PIPE-002
func TestREQ_PIPE_002_WrappersDoNotChainModifiers(t *testing.T) {
	for name, v := range map[string]any{
		"Optional": z.String().Optional(),
		"Nullable": z.String().Nullable(),
	} {
		typ := reflect.TypeOf(v)
		for _, m := range []string{"Parse", "ParseJSON", "Decode", "GetMeta"} {
			if _, ok := typ.MethodByName(m); !ok {
				t.Errorf("%s wrapper must expose %s", name, m)
			}
		}
		if _, ok := typ.MethodByName(name); ok {
			t.Errorf("%s wrapper must not expose another .%s()", name, name)
		}
	}
}

// @spec REQ-PIPE-009
func TestREQ_PIPE_009_NoSafeParseOrAsync(t *testing.T) {
	typ := reflect.TypeOf(z.String())
	for _, m := range []string{"SafeParse", "ParseAsync", "SafeParseAsync"} {
		if _, ok := typ.MethodByName(m); ok {
			t.Errorf("schema must not expose %s", m)
		}
	}
}
