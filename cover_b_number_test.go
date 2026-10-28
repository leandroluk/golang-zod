package z_test

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// coverBCancelCtx reports no error on the first Err call (so Parse proceeds)
// and a closed Done channel, so in-flight checkCancel calls observe cancellation.
type coverBCancelCtx struct {
	context.Context
	done  chan struct{}
	calls int
}

func coverBNewCancelCtx() *coverBCancelCtx {
	d := make(chan struct{})
	close(d)
	return &coverBCancelCtx{Context: context.Background(), done: d}
}

func (c *coverBCancelCtx) Done() <-chan struct{} { return c.done }
func (c *coverBCancelCtx) Err() error {
	c.calls++
	if c.calls == 1 {
		return nil
	}
	return context.Canceled
}

func coverBIssues(t *testing.T, err error) []z.Issue {
	t.Helper()
	var zerr *z.ZodError
	if !errors.As(err, &zerr) {
		t.Fatalf("expected *ZodError, got %T (%v)", err, err)
	}
	return zerr.Issues()
}

func coverBOneIssue(t *testing.T, err error) z.Issue {
	t.Helper()
	is := coverBIssues(t, err)
	if len(is) != 1 {
		t.Fatalf("expected exactly 1 issue, got %d: %v", len(is), is)
	}
	return is[0]
}

func coverBErrOnly[T any](_ T, err error) error { return err }

type coverBNumCase[T z.NumberConstraint] struct {
	name string
	in   any
	want T
	code z.IssueCode // "" means success
	bnd  any         // expected Minimum/Maximum bound on failure
}

func coverBRunNum[T z.NumberConstraint](t *testing.T, s *z.NumberSchema[T], cases []coverBNumCase[T]) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.Parse(c.in)
			if c.code == "" {
				if err != nil || got != c.want {
					t.Fatalf("want (%v, nil), got (%v, %v)", c.want, got, err)
				}
				return
			}
			is := coverBOneIssue(t, err)
			if is.Code != c.code {
				t.Fatalf("want code %s, got %s", c.code, is.Code)
			}
			switch c.code {
			case z.TooBig:
				if is.Maximum != c.bnd || is.Origin != "number" || !is.Inclusive {
					t.Fatalf("bad TooBig issue: %+v", is)
				}
			case z.TooSmall:
				if is.Minimum != c.bnd || is.Origin != "number" || !is.Inclusive {
					t.Fatalf("bad TooSmall issue: %+v", is)
				}
			case z.InvalidType:
				if is.Expected != "number" {
					t.Fatalf("bad InvalidType issue: %+v", is)
				}
			}
		})
	}
}

// @spec REQ-PRIM-007
func TestCoverBNumberIntegerKinds(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		coverBRunNum(t, z.Int(), []coverBNumCase[int]{
			{name: "same type", in: 7, want: 7},
			{name: "int64", in: int64(8), want: 8},
			{name: "integral float", in: 3.0, want: 3},
			{name: "negative integral float", in: -3.0, want: -3},
			{name: "fractional float rejected", in: 1.5, code: z.InvalidType},
			{name: "NaN rejected", in: math.NaN(), code: z.InvalidType},
			{name: "+Inf too big", in: math.Inf(1), code: z.TooBig, bnd: math.MaxInt},
			{name: "-Inf too small", in: math.Inf(-1), code: z.TooSmall, bnd: math.MinInt},
			{name: "string rejected", in: "1", code: z.InvalidType},
			{name: "nil rejected", in: nil, code: z.InvalidType},
			{name: "float above int64", in: 1e19, code: z.TooBig, bnd: math.MaxInt},
			{name: "float below int64", in: -1e19, code: z.TooSmall, bnd: math.MinInt},
			{name: "json.Number ok", in: json.Number("42"), want: 42},
			{name: "json.Number fractional", in: json.Number("1.5"), code: z.InvalidType},
			{name: "json.Number garbage", in: json.Number("abc"), code: z.InvalidType},
			{name: "json.Number overflow", in: json.Number("99999999999999999999"), code: z.TooBig, bnd: math.MaxInt},
			{name: "json.Number underflow", in: json.Number("-99999999999999999999"), code: z.TooSmall, bnd: math.MinInt},
		})
	})
	t.Run("Int8", func(t *testing.T) {
		coverBRunNum(t, z.Int8(), []coverBNumCase[int8]{
			{name: "max", in: 127, want: 127},
			{name: "min", in: -128, want: -128},
			{name: "too big", in: 128, code: z.TooBig, bnd: int8(127)},
			{name: "too small", in: -129, code: z.TooSmall, bnd: int8(-128)},
			{name: "int64 too big", in: int64(300), code: z.TooBig, bnd: int8(127)},
			{name: "float too big", in: 1000.0, code: z.TooBig, bnd: int8(127)},
			{name: "json.Number too big", in: json.Number("200"), code: z.TooBig, bnd: int8(127)},
		})
	})
	t.Run("Int16", func(t *testing.T) {
		coverBRunNum(t, z.Int16(), []coverBNumCase[int16]{
			{name: "ok", in: 300, want: 300},
			{name: "too big", in: 40000, code: z.TooBig, bnd: int16(32767)},
		})
	})
	t.Run("Int32", func(t *testing.T) {
		coverBRunNum(t, z.Int32(), []coverBNumCase[int32]{
			{name: "ok", in: 70000, want: 70000},
			{name: "too small", in: int64(-3000000000), code: z.TooSmall, bnd: int32(math.MinInt32)},
		})
	})
	t.Run("Int64", func(t *testing.T) {
		coverBRunNum(t, z.Int64(), []coverBNumCase[int64]{
			{name: "max via json.Number", in: json.Number("9223372036854775807"), want: math.MaxInt64},
			{name: "min via json.Number", in: json.Number("-9223372036854775808"), want: math.MinInt64},
			{name: "json.Number too big", in: json.Number("9223372036854775808"), code: z.TooBig, bnd: int64(math.MaxInt64)},
			{name: "json.Number too small", in: json.Number("-9223372036854775809"), code: z.TooSmall, bnd: int64(math.MinInt64)},
			{name: "float 2^63 too big", in: 9223372036854775808.0, code: z.TooBig, bnd: int64(math.MaxInt64)},
			{name: "float 2^64 too big", in: 18446744073709551616.0, code: z.TooBig, bnd: int64(math.MaxInt64)},
			{name: "float -2^63 ok", in: -9223372036854775808.0, want: math.MinInt64},
		})
	})
	t.Run("Uint", func(t *testing.T) {
		coverBRunNum(t, z.Uint(), []coverBNumCase[uint]{
			{name: "ok", in: 5, want: 5},
			{name: "negative int", in: -1, code: z.TooSmall, bnd: uint(0)},
			{name: "negative float", in: -2.0, code: z.TooSmall, bnd: uint(0)},
		})
	})
	t.Run("Uint8", func(t *testing.T) {
		coverBRunNum(t, z.Uint8(), []coverBNumCase[uint8]{
			{name: "max", in: 255, want: 255},
			{name: "too big", in: 256, code: z.TooBig, bnd: uint8(255)},
			{name: "json.Number too big", in: json.Number("300"), code: z.TooBig, bnd: uint8(255)},
			{name: "json.Number negative", in: json.Number("-1"), code: z.TooSmall, bnd: uint8(0)},
		})
	})
	t.Run("Uint16", func(t *testing.T) {
		coverBRunNum(t, z.Uint16(), []coverBNumCase[uint16]{
			{name: "ok", in: 65535, want: 65535},
			{name: "too big", in: 65536, code: z.TooBig, bnd: uint16(65535)},
		})
	})
	t.Run("Uint32", func(t *testing.T) {
		coverBRunNum(t, z.Uint32(), []coverBNumCase[uint32]{
			{name: "ok", in: int64(4294967295), want: 4294967295},
			{name: "too big", in: int64(4294967296), code: z.TooBig, bnd: uint32(math.MaxUint32)},
		})
	})
	t.Run("Uint64", func(t *testing.T) {
		coverBRunNum(t, z.Uint64(), []coverBNumCase[uint64]{
			{name: "max via json.Number", in: json.Number("18446744073709551615"), want: math.MaxUint64},
			{name: "json.Number too big", in: json.Number("18446744073709551616"), code: z.TooBig, bnd: uint64(math.MaxUint64)},
			{name: "json.Number negative", in: json.Number("-1"), code: z.TooSmall, bnd: uint64(0)},
			{name: "json.Number garbage", in: json.Number("x"), code: z.InvalidType},
			{name: "json.Number fractional", in: json.Number("1.5"), code: z.InvalidType},
			{name: "float 2^63 ok", in: 9223372036854775808.0, want: 1 << 63},
			{name: "float 2^64 too big", in: 18446744073709551616.0, code: z.TooBig, bnd: uint64(math.MaxUint64)},
			{name: "negative float", in: -1.0, code: z.TooSmall, bnd: uint64(0)},
			{name: "very negative float", in: -1e30, code: z.TooSmall, bnd: uint64(0)},
		})
	})
}

// @spec REQ-PRIM-007
func TestCoverBNumberFloatKinds(t *testing.T) {
	t.Run("Number", func(t *testing.T) {
		coverBRunNum(t, z.Number(), []coverBNumCase[float64]{
			{name: "float", in: 1.5, want: 1.5},
			{name: "int", in: 2, want: 2},
			{name: "int64", in: int64(3), want: 3},
			{name: "json.Number", in: json.Number("2.5"), want: 2.5},
			{name: "json.Number garbage", in: json.Number("zz"), code: z.InvalidType},
			{name: "NaN", in: math.NaN(), code: z.InvalidType},
			{name: "+Inf", in: math.Inf(1), code: z.InvalidType},
			{name: "-Inf", in: math.Inf(-1), code: z.InvalidType},
			{name: "bool", in: true, code: z.InvalidType},
		})
	})
	t.Run("Float64", func(t *testing.T) {
		coverBRunNum(t, z.Float64(), []coverBNumCase[float64]{{name: "ok", in: 0.25, want: 0.25}})
	})
	t.Run("Float32", func(t *testing.T) {
		coverBRunNum(t, z.Float32(), []coverBNumCase[float32]{
			{name: "float", in: 1.5, want: 1.5},
			{name: "native float32", in: float32(2.5), want: 2.5},
			{name: "json.Number", in: json.Number("0.5"), want: 0.5},
			{name: "overflow to Inf rejected", in: 1e300, code: z.InvalidType},
		})
	})
}

// @spec REQ-PRIM-007
func TestCoverBNumberChecks(t *testing.T) {
	type tc struct {
		name   string
		schema *z.NumberSchema[int]
		pass   []int
		fail   []int
		code   z.IssueCode
		incl   bool
		bound  int
	}
	cases := []tc{
		{"GT", z.Int().GT(5), []int{6}, []int{5, 4}, z.TooSmall, false, 5},
		{"GTE", z.Int().GTE(5), []int{5, 6}, []int{4}, z.TooSmall, true, 5},
		{"Min", z.Int().Min(5), []int{5}, []int{4}, z.TooSmall, true, 5},
		{"LT", z.Int().LT(5), []int{4}, []int{5, 6}, z.TooBig, false, 5},
		{"LTE", z.Int().LTE(5), []int{5, 4}, []int{6}, z.TooBig, true, 5},
		{"Max", z.Int().Max(5), []int{5}, []int{6}, z.TooBig, true, 5},
		{"Positive", z.Int().Positive(), []int{1}, []int{0, -1}, z.TooSmall, false, 0},
		{"NonNegative", z.Int().NonNegative(), []int{0, 1}, []int{-1}, z.TooSmall, true, 0},
		{"Negative", z.Int().Negative(), []int{-1}, []int{0, 1}, z.TooBig, false, 0},
		{"NonPositive", z.Int().NonPositive(), []int{0, -1}, []int{1}, z.TooBig, true, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, v := range c.pass {
				if got, err := c.schema.Parse(v); err != nil || got != v {
					t.Fatalf("Parse(%d) = (%d, %v), want pass", v, got, err)
				}
			}
			for _, v := range c.fail {
				_, err := c.schema.Parse(v)
				is := coverBOneIssue(t, err)
				if is.Code != c.code || is.Inclusive != c.incl || is.Origin != "number" || is.Input != v {
					t.Fatalf("Parse(%d): unexpected issue %+v", v, is)
				}
				if c.code == z.TooSmall && is.Minimum != c.bound || c.code == z.TooBig && is.Maximum != c.bound {
					t.Fatalf("Parse(%d): wrong bound in %+v", v, is)
				}
			}
		})
	}

	t.Run("MultipleOf", func(t *testing.T) {
		s := z.Int().MultipleOf(3)
		if got, err := s.Parse(9); err != nil || got != 9 {
			t.Fatalf("9 should pass, got (%d, %v)", got, err)
		}
		if got, err := s.Parse(0); err != nil || got != 0 {
			t.Fatalf("0 should pass, got (%d, %v)", got, err)
		}
		is := coverBOneIssue(t, coverBErrOnly(s.Parse(10)))
		if is.Code != z.NotMultipleOf || is.Divisor != 3 || is.Input != 10 {
			t.Fatalf("unexpected issue %+v", is)
		}
		is = coverBOneIssue(t, coverBErrOnly(z.Int().MultipleOf(0).Parse(4)))
		if is.Code != z.NotMultipleOf {
			t.Fatalf("divisor 0 must fail with NotMultipleOf, got %+v", is)
		}
		if got, err := z.Number().MultipleOf(0.1).Parse(0.3); err != nil || got != 0.3 {
			t.Fatalf("0.3 is a multiple of 0.1 within tolerance, got (%v, %v)", got, err)
		}
		if _, err := z.Number().MultipleOf(0.5).Parse(0.3); err == nil {
			t.Fatal("0.3 must not be a multiple of 0.5")
		}
	})

	t.Run("chaining does not mutate the receiver", func(t *testing.T) {
		base := z.Int()
		_ = base.Min(10)
		if _, err := base.Parse(1); err != nil {
			t.Fatalf("base schema must be unchanged: %v", err)
		}
	})

	t.Run("all failing checks are reported", func(t *testing.T) {
		s := z.Int().GT(10).LT(0).MultipleOf(7)
		is := coverBIssues(t, coverBErrOnly(s.Parse(5)))
		if len(is) != 3 || is[0].Code != z.TooSmall || is[1].Code != z.TooBig || is[2].Code != z.NotMultipleOf {
			t.Fatalf("unexpected issues %+v", is)
		}
	})

	t.Run("abort early stops after first failed check", func(t *testing.T) {
		s := z.Int().GT(10).LT(0).MultipleOf(7)
		_, err := s.Parse(5, z.WithAbortEarly(context.Background()))
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall {
			t.Fatalf("unexpected issue %+v", is)
		}
	})

	t.Run("Abort check option stops following checks", func(t *testing.T) {
		s := z.Int().GT(10, z.Abort(true)).LT(0)
		_, err := s.Parse(5)
		if is := coverBOneIssue(t, err); is.Code != z.TooSmall {
			t.Fatalf("unexpected issue %+v", is)
		}
	})

	t.Run("Message check option", func(t *testing.T) {
		s := z.Int().Min(10, z.Message("too low"))
		_, err := s.Parse(1)
		if is := coverBOneIssue(t, err); is.Message != "too low" {
			t.Fatalf("unexpected message %q", is.Message)
		}
	})

	t.Run("cancellation during checks", func(t *testing.T) {
		s := z.Int().Min(1)
		got, err := s.Parse(5, coverBNewCancelCtx())
		if err == nil || got != 0 {
			t.Fatalf("expected failure on cancellation, got (%d, %v)", got, err)
		}
		if is := coverBIssues(t, err); len(is) != 0 {
			t.Fatalf("cancellation must not add issues, got %v", is)
		}
	})
}

// @spec REQ-PRIM-009 REQ-PIPE-007 REQ-CORE-010
func TestCoverBNumberModifiers(t *testing.T) {
	s := z.Int().Min(1)

	if v, err := s.Optional().Parse(3); err != nil || !v.IsSome() {
		t.Fatalf("Optional(3) = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable(nil) = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(4); err != nil || !v.IsSome() {
		t.Fatalf("Nullish(4) = (%v, %v)", v, err)
	}
	if v, err := s.Default(9).Parse(nil); err != nil || v != 9 {
		t.Fatalf("Default(9)(nil) = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{1, 2}); err != nil || len(v) != 2 || v[1] != 2 {
		t.Fatalf("Array = (%v, %v)", v, err)
	}

	m := s.Meta(func(b *z.MetaBuilder) { b.ID("coverB_number_meta").Title("N") })
	if meta := m.GetMeta(); meta.ID != "coverB_number_meta" || meta.Title != "N" {
		t.Fatalf("unexpected meta %+v", meta)
	}
	if _, err := m.Parse(0); err == nil {
		t.Fatal("Meta clone must keep checks")
	}

	_, err := s.Error("custom number error").Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "custom number error" {
		t.Fatalf("Error: unexpected message %q", is.Message)
	}
	_, err = s.ErrorFunc(func(i *z.Issue) string { return "fn:" + string(i.Code) }).Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "fn:invalid_type" {
		t.Fatalf("ErrorFunc: unexpected message %q", is.Message)
	}
}

// @spec REQ-PRIM-007 REQ-PRIM-009 REQ-PIPE-007 REQ-CORE-010
func TestCoverBBigInt(t *testing.T) {
	huge := "123456789012345678901234567890"
	want, _ := new(big.Int).SetString(huge, 10)
	orig := big.NewInt(77)

	ok := []struct {
		name string
		in   any
		want *big.Int
	}{
		{"*big.Int passthrough", orig, orig},
		{"string", huge, want},
		{"int64", int64(-5), big.NewInt(-5)},
		{"int", 12, big.NewInt(12)},
		{"integral float", 1e3, big.NewInt(1000)},
		{"json.Number", json.Number(huge), want},
	}
	for _, c := range ok {
		t.Run(c.name, func(t *testing.T) {
			got, err := z.BigInt().Parse(c.in)
			if err != nil || got.Cmp(c.want) != 0 {
				t.Fatalf("got (%v, %v), want %v", got, err, c.want)
			}
		})
	}

	bad := []struct {
		name string
		in   any
	}{
		{"bad string", "12x"},
		{"fractional float", 1.5},
		{"+Inf", math.Inf(1)},
		{"bad json.Number", json.Number("1.5")},
		{"bool", true},
		{"nil", nil},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			_, err := z.BigInt().Parse(c.in)
			if is := coverBOneIssue(t, err); is.Code != z.InvalidType || is.Expected != "bigint" {
				t.Fatalf("unexpected issue %+v", is)
			}
		})
	}

	s := z.BigInt()
	if v, err := s.Optional().Parse(5); err != nil || !v.IsSome() {
		t.Fatalf("Optional = (%v, %v)", v, err)
	}
	if v, err := s.Nullable().Parse(nil); err != nil || !v.IsNull() {
		t.Fatalf("Nullable = (%v, %v)", v, err)
	}
	if v, err := s.Nullish().Parse(nil); err != nil || !v.IsSome() {
		t.Fatalf("Nullish = (%v, %v)", v, err)
	}
	if v, err := s.Default(big.NewInt(3)).Parse(nil); err != nil || v.Int64() != 3 {
		t.Fatalf("Default = (%v, %v)", v, err)
	}
	if v, err := s.Array().Parse([]any{1, "2"}); err != nil || len(v) != 2 || v[1].Int64() != 2 {
		t.Fatalf("Array = (%v, %v)", v, err)
	}
	m := s.Meta(func(b *z.MetaBuilder) { b.ID("coverB_bigint_meta") })
	if m.GetMeta().ID != "coverB_bigint_meta" {
		t.Fatalf("unexpected meta %+v", m.GetMeta())
	}
	_, err := s.Error("big err").Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "big err" {
		t.Fatalf("Error: %q", is.Message)
	}
	_, err = s.ErrorFunc(func(*z.Issue) string { return "big fn" }).Parse("x")
	if is := coverBOneIssue(t, err); is.Message != "big fn" {
		t.Fatalf("ErrorFunc: %q", is.Message)
	}
}
