package z_test

import (
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

func TestDefaultFuncUUIDEvaluatedPerParse(t *testing.T) {
	n := 0
	s := z.UUID().DefaultFunc(func() string {
		n++
		return "00000000-0000-4000-8000-00000000000" + string(rune('0'+n))
	})
	a, err := s.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Parse(nil)
	if a == b {
		t.Fatalf("expected distinct values, got %q twice", a)
	}
	v, err := s.Parse("11111111-1111-4111-8111-111111111111")
	if err != nil || v != "11111111-1111-4111-8111-111111111111" {
		t.Fatalf("got %q, %v", v, err)
	}
}

func TestZodParityModifierMethods(t *testing.T) {
	if v, err := z.String().Prefault("x").Parse(nil); err != nil || v != "x" {
		t.Fatalf("prefault: %q %v", v, err)
	}
	if v, err := z.Int().Catch(7).Parse("nope"); err != nil || v != 7 {
		t.Fatalf("catch: %v %v", v, err)
	}
	if v, err := z.String().Readonly().Parse("a"); err != nil || v != "a" {
		t.Fatalf("readonly: %q %v", v, err)
	}
}

func TestCUIDAndHash(t *testing.T) {
	if _, err := z.CUID().Parse("cjld2cjxh0000qzrmn831i7rn"); err != nil {
		t.Fatal(err)
	}
	if _, err := z.CUID().Parse("nope"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := z.Hash(z.SHA256).Parse("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); err != nil {
		t.Fatal(err)
	}
	if _, err := z.Hash(z.MD5, z.HashBase64).Parse("1B2M2Y8AsgTpgAmY7PhCfg=="); err != nil {
		t.Fatal(err)
	}
	if _, err := z.Hash(z.SHA1).Parse("abc"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDateMinMax(t *testing.T) {
	lo := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	hi := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	s := z.Date().Min(lo).Max(hi)
	if _, err := s.Parse(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Parse(lo.Add(-time.Hour)); err == nil {
		t.Fatal("expected too_small")
	}
	if _, err := s.Parse("2031-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected too_big")
	}
}
