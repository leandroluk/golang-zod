package z_test

import (
	"math/big"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-OPT-002 REQ-OPT-003
func TestREQ_OPT_002_003_TokenDecoding(t *testing.T) {
	t.Run("Int64 and Uint64 keep full precision", func(t *testing.T) {
		got, err := z.Int64().ParseJSON([]byte(`9007199254740993`))
		if err != nil || got != 9007199254740993 {
			t.Fatalf("got %d, %v", got, err)
		}
		u, err := z.Uint64().ParseJSON([]byte(`18446744073709551615`))
		if err != nil || u != 18446744073709551615 {
			t.Fatalf("got %d, %v", u, err)
		}
	})

	t.Run("BigInt accepts small and huge integers", func(t *testing.T) {
		b, err := z.BigInt().ParseJSON([]byte(`12345678901234567890123`))
		want, _ := new(big.Int).SetString("12345678901234567890123", 10)
		if err != nil || b.Cmp(want) != 0 {
			t.Fatalf("got %v, %v", b, err)
		}
		b, err = z.BigInt().ParseJSON([]byte(`5`))
		if err != nil || b.Int64() != 5 {
			t.Fatalf("got %v, %v", b, err)
		}
	})

	t.Run("nested documents and Decode from a reader", func(t *testing.T) {
		s := z.Object(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
			s.Field("tags", z.Array(z.String()))
			s.Field("ok", z.Boolean())
			s.Field("n", z.Float64())
		})
		out, err := s.Decode(strings.NewReader(`{"name":"a","tags":["x","y"],"ok":true,"n":1.5}`))
		if err != nil {
			t.Fatal(err)
		}
		if out["name"] != "a" || out["n"] != 1.5 || out["ok"] != true {
			t.Fatalf("unexpected output: %v", out)
		}
	})

	t.Run("null, false and malformed input", func(t *testing.T) {
		if _, err := z.Null().ParseJSON([]byte(`null`)); err != nil {
			t.Fatal(err)
		}
		if _, err := z.Boolean().ParseJSON([]byte(`false`)); err != nil {
			t.Fatal(err)
		}
		for _, bad := range []string{`{`, `[1,`, `{"a":}`, `[1 2]`, ``} {
			if _, err := z.Any().ParseJSON([]byte(bad)); err == nil {
				t.Fatalf("expected error for %q", bad)
			}
		}
	})
}
