package z_test

import (
	"math"
	"strconv"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func TestExtrasNaNJSONCustomSuccess(t *testing.T) {
	if _, err := z.NaN().Parse(math.NaN()); err != nil {
		t.Fatal(err)
	}
	if _, err := z.NaN().Parse(1.0); err == nil {
		t.Fatal("expected error")
	}
	if _, err := z.JSON().Parse(map[string]any{"a": []any{1.0, "x", nil, true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := z.JSON().Parse(struct{}{}); err == nil {
		t.Fatal("expected error")
	}
	even := z.CustomFn[int](func(v any) bool { return v.(int)%2 == 0 }, "must be even")
	if _, err := even.Parse(4); err != nil {
		t.Fatal(err)
	}
	if _, err := even.Parse(3); err == nil {
		t.Fatal("expected error")
	}
	if _, err := even.Parse("x"); err == nil {
		t.Fatal("expected type error")
	}
	ok, err := z.Success[int](z.Int()).Parse("nope")
	if err != nil || ok {
		t.Fatalf("success: %v %v", ok, err)
	}
	ok, _ = z.Success[int](z.Int()).Parse(3)
	if !ok {
		t.Fatal("expected true")
	}
}

func TestExtrasNonOptionalTemplateInvertSafeStep(t *testing.T) {
	no := z.NonOptional[string](z.String().Optional())
	if v, err := no.Parse("a"); err != nil || v != "a" {
		t.Fatalf("%q %v", v, err)
	}
	if _, err := no.Parse(nil); err == nil {
		t.Fatal("expected error")
	}

	tl := z.TemplateLiteral("user-", z.Int(), "@", z.Enum("a", "b"))
	if _, err := tl.Parse("user-42@a"); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.Parse("user-x@a"); err == nil {
		t.Fatal("expected error")
	}

	c := z.Codec[string, int](z.String(), z.Int(),
		func(s string) (int, error) { return strconv.Atoi(s) },
		func(i int) (string, error) { return strconv.Itoa(i), nil })
	inv := z.InvertCodec(c)
	if v, err := inv.Parse(5); err != nil || v != "5" {
		t.Fatalf("%q %v", v, err)
	}

	if _, err := z.Int64().Safe().Parse(int64(1) << 60); err == nil {
		t.Fatal("expected too_big")
	}
	if _, err := z.Int8().Safe().Parse(100); err != nil {
		t.Fatal(err)
	}
	if _, err := z.Int().Step(5).Parse(12); err == nil {
		t.Fatal("expected not_multiple_of")
	}
}
