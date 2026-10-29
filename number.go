package z

import (
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
)

// NumberConstraint defines all built-in Go numeric types supported by NumberSchema.
type NumberConstraint interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type numCheckKind int

const (
	checkGT numCheckKind = iota
	checkGTE
	checkLT
	checkLTE
	checkMultipleOf
)

type numCheck[T NumberConstraint] struct {
	kind numCheckKind
	val  T
	opts *checkOpts
}

// @spec REQ-PRIM-007
// NumberSchema validates numeric inputs of constraint type T.
type NumberSchema[T NumberConstraint] struct {
	BaseSchema[T]
	checks []numCheck[T]
}

type numberNode[T NumberConstraint] struct {
	schema *NumberSchema[T]
}

func (n *numberNode[T]) kind() nodeKind { return kindNumber }
func (n *numberNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	val, conv := convertToNumber[T](in)
	switch conv {
	case convTooSmall:
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "number",
			Minimum:   minOf[T](),
			Inclusive: true,
			Input:     in,
		})
		var zero T
		return zero, false
	case convTooBig:
		ctx.addIssue(Issue{
			Code:      TooBig,
			Origin:    "number",
			Maximum:   maxOf[T](),
			Inclusive: true,
			Input:     in,
		})
		var zero T
		return zero, false
	case convInvalid:
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "number",
			Input:    in,
		})
		var zero T
		return zero, false
	}

	// Reject NaN and Inf for floats
	if f := float64(val); math.IsNaN(f) || math.IsInf(f, 0) {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "number",
			Input:    in,
		})
		var zero T
		return zero, false
	}

	allOk := true
	for _, ch := range n.schema.checks {
		if err := ctx.checkCancel(); err != nil {
			var zero T
			return zero, false
		}
		mark := ctx.beginCheck(ch.opts)
		switch ch.kind {
		case checkGT:
			if !(val > ch.val) {
				ctx.addIssue(Issue{
					Code:      TooSmall,
					Origin:    "number",
					Minimum:   ch.val,
					Inclusive: false,
					Input:     val,
				})
				allOk = false
			}

		case checkGTE:
			if !(val >= ch.val) {
				ctx.addIssue(Issue{
					Code:      TooSmall,
					Origin:    "number",
					Minimum:   ch.val,
					Inclusive: true,
					Input:     val,
				})
				allOk = false
			}

		case checkLT:
			if !(val < ch.val) {
				ctx.addIssue(Issue{
					Code:      TooBig,
					Origin:    "number",
					Maximum:   ch.val,
					Inclusive: false,
					Input:     val,
				})
				allOk = false
			}

		case checkLTE:
			if !(val <= ch.val) {
				ctx.addIssue(Issue{
					Code:      TooBig,
					Origin:    "number",
					Maximum:   ch.val,
					Inclusive: true,
					Input:     val,
				})
				allOk = false
			}

		case checkMultipleOf:
			if !isMultipleOf(val, ch.val) {
				ctx.addIssue(Issue{
					Code:    NotMultipleOf,
					Divisor: ch.val,
					Input:   val,
				})
				allOk = false
			}
		}

		if ctx.endCheck(mark, ch.opts) {
			var zero T
			return zero, false
		}
		if !allOk && ctx.shouldAbort() {
			break
		}
	}

	if !allOk {
		var zero T
		return zero, false
	}
	return val, true
}

func isMultipleOf[T NumberConstraint](val, divisor T) bool {
	if divisor == 0 {
		return false
	}
	vFloat := float64(val)
	dFloat := float64(divisor)
	remainder := math.Remainder(vFloat, dFloat)
	return math.Abs(remainder) < 1e-9
}

type numConv int

const (
	convOK numConv = iota
	convInvalid
	convTooSmall
	convTooBig
)

// isIntegerKind reports whether T is an integer type (as opposed to float32/float64).
func isIntegerKind[T NumberConstraint]() bool {
	half := 0.5
	return T(half) == 0
}

// isUnsignedKind reports whether T is an unsigned integer type.
func isUnsignedKind[T NumberConstraint]() bool {
	var z T
	return isIntegerKind[T]() && z-1 > z
}

// maxOf returns the largest value of integer type T.
func maxOf[T NumberConstraint]() T {
	var z T
	if isUnsignedKind[T]() {
		return z - 1
	}
	m := T(1)
	for m*2+1 > m {
		m = m*2 + 1
	}
	return m
}

// minOf returns the smallest value of integer type T.
func minOf[T NumberConstraint]() T {
	if isUnsignedKind[T]() {
		return 0
	}
	return -maxOf[T]() - 1
}

// fromInt64 converts an int64 to T, reporting overflow for integer kinds.
func fromInt64[T NumberConstraint](x int64) (T, numConv) {
	t := T(x)
	if !isIntegerKind[T]() {
		return t, convOK
	}
	if (x < 0) != (t < 0) || int64(t) != x {
		if x < 0 {
			return 0, convTooSmall
		}
		return 0, convTooBig
	}
	return t, convOK
}

// fromUint64 converts a uint64 to T, reporting overflow for integer kinds.
func fromUint64[T NumberConstraint](u uint64) (T, numConv) {
	t := T(u)
	if isIntegerKind[T]() && (t < 0 || uint64(t) != u) {
		return 0, convTooBig
	}
	return t, convOK
}

// fromFloat64 converts a float64 to T; integer kinds require an integral value in range.
func fromFloat64[T NumberConstraint](f float64) (T, numConv) {
	if !isIntegerKind[T]() {
		return T(f), convOK
	}
	if f != math.Trunc(f) {
		return 0, convInvalid
	}
	switch {
	case f < -(1 << 63):
		return 0, convTooSmall
	case f >= (1 << 64):
		return 0, convTooBig
	case f >= (1 << 63):
		return fromUint64[T](uint64(f))
	}
	return fromInt64[T](int64(f))
}

func convertToNumber[T NumberConstraint](in any) (T, numConv) {
	var zero T
	switch v := in.(type) {
	case T:
		return v, convOK
	case int:
		return fromInt64[T](int64(v))
	case int64:
		return fromInt64[T](v)
	case float64:
		return fromFloat64[T](v)
	case json.Number:
		str := string(v)
		switch {
		case !isIntegerKind[T]():
			parsed, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return zero, convInvalid
			}
			return T(parsed), convOK
		case isUnsignedKind[T]():
			parsed, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				if errors.Is(err, strconv.ErrRange) {
					return zero, convTooBig
				}
				if _, serr := strconv.ParseInt(str, 10, 64); serr == nil {
					return zero, convTooSmall
				}
				return zero, convInvalid
			}
			return fromUint64[T](parsed)
		default:
			parsed, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				if errors.Is(err, strconv.ErrRange) {
					if strings.HasPrefix(str, "-") {
						return zero, convTooSmall
					}
					return zero, convTooBig
				}
				return zero, convInvalid
			}
			return fromInt64[T](parsed)
		}
	default:
		return zero, convInvalid
	}
}

func newNumberSchema[T NumberConstraint]() *NumberSchema[T] {
	s := &NumberSchema[T]{}
	s.BaseSchema = NewBaseSchema[T](&numberNode[T]{schema: s})
	return s
}

func (s *NumberSchema[T]) clone() *NumberSchema[T] {
	c := &NumberSchema[T]{
		checks: slices.Clone(s.checks),
	}
	c.BaseSchema = NewBaseSchema[T](&numberNode[T]{schema: c})
	c.setMeta(s.getMeta())
	c.slot = s.slot
	c.updateSlot(c)
	return c
}

// @spec REQ-PRIM-007
func Number() *NumberSchema[float64] { return newNumberSchema[float64]() }

// @spec REQ-PRIM-007
func Float64() *NumberSchema[float64] { return newNumberSchema[float64]() }

// @spec REQ-PRIM-007
func Float32() *NumberSchema[float32] { return newNumberSchema[float32]() }

// @spec REQ-PRIM-007
func Int() *NumberSchema[int] { return newNumberSchema[int]() }

// @spec REQ-PRIM-007
func Int8() *NumberSchema[int8] { return newNumberSchema[int8]() }

// @spec REQ-PRIM-007
func Int16() *NumberSchema[int16] { return newNumberSchema[int16]() }

// @spec REQ-PRIM-007
func Int32() *NumberSchema[int32] { return newNumberSchema[int32]() }

// @spec REQ-PRIM-007
func Int64() *NumberSchema[int64] { return newNumberSchema[int64]() }

// @spec REQ-PRIM-007
func Uint() *NumberSchema[uint] { return newNumberSchema[uint]() }

// @spec REQ-PRIM-007
func Uint8() *NumberSchema[uint8] { return newNumberSchema[uint8]() }

// @spec REQ-PRIM-007
func Uint16() *NumberSchema[uint16] { return newNumberSchema[uint16]() }

// @spec REQ-PRIM-007
func Uint32() *NumberSchema[uint32] { return newNumberSchema[uint32]() }

// @spec REQ-PRIM-007
func Uint64() *NumberSchema[uint64] { return newNumberSchema[uint64]() }

// @spec REQ-PRIM-007
// GT enforces strict greater-than (val > min).
func (s *NumberSchema[T]) GT(min T, opts ...CheckOption) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkGT, val: min, opts: buildCheckOpts(opts)})
	return c
}

// @spec REQ-PRIM-007
// GTE enforces greater-than-or-equal (val >= min).
func (s *NumberSchema[T]) GTE(min T, opts ...CheckOption) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkGTE, val: min, opts: buildCheckOpts(opts)})
	return c
}

// @spec REQ-PRIM-007
// Min is an alias for GTE matching Zod v4.
func (s *NumberSchema[T]) Min(min T, opts ...CheckOption) *NumberSchema[T] {
	return s.GTE(min, opts...)
}

// @spec REQ-PRIM-007
// LT enforces strict less-than (val < max).
func (s *NumberSchema[T]) LT(max T, opts ...CheckOption) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkLT, val: max, opts: buildCheckOpts(opts)})
	return c
}

// @spec REQ-PRIM-007
// LTE enforces less-than-or-equal (val <= max).
func (s *NumberSchema[T]) LTE(max T, opts ...CheckOption) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkLTE, val: max, opts: buildCheckOpts(opts)})
	return c
}

// @spec REQ-PRIM-007
// Max is an alias for LTE matching Zod v4.
func (s *NumberSchema[T]) Max(max T, opts ...CheckOption) *NumberSchema[T] {
	return s.LTE(max, opts...)
}

// @spec REQ-PRIM-007
// Positive enforces val > 0.
func (s *NumberSchema[T]) Positive(opts ...CheckOption) *NumberSchema[T] {
	return s.GT(0, opts...)
}

// @spec REQ-PRIM-007
// NonNegative enforces val >= 0.
func (s *NumberSchema[T]) NonNegative(opts ...CheckOption) *NumberSchema[T] {
	return s.GTE(0, opts...)
}

// @spec REQ-PRIM-007
// Negative enforces val < 0.
func (s *NumberSchema[T]) Negative(opts ...CheckOption) *NumberSchema[T] {
	return s.LT(0, opts...)
}

// @spec REQ-PRIM-007
// NonPositive enforces val <= 0.
func (s *NumberSchema[T]) NonPositive(opts ...CheckOption) *NumberSchema[T] {
	return s.LTE(0, opts...)
}

// @spec REQ-PRIM-007
// MultipleOf checks that value is evenly divisible by divisor.
func (s *NumberSchema[T]) MultipleOf(divisor T, opts ...CheckOption) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkMultipleOf, val: divisor, opts: buildCheckOpts(opts)})
	return c
}

// @spec REQ-PRIM-009
func (s *NumberSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *NumberSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *NumberSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *NumberSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *NumberSchema[T]) Array() *ArraySchema[T] { return NewArraySchema[T](s) }
func (s *NumberSchema[T]) Meta(fn func(m *MetaBuilder)) *NumberSchema[T] {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// --- BigInt (*big.Int) ---

// @spec REQ-PRIM-007
type BigIntSchema struct {
	BaseSchema[*big.Int]
}

type bigIntNode struct{}

func (n *bigIntNode) kind() nodeKind { return kindBigInt }
func (n *bigIntNode) parse(ctx *parseCtx, in any) (any, bool) {
	switch v := in.(type) {
	case *big.Int:
		return v, true
	case string:
		bi := new(big.Int)
		if _, ok := bi.SetString(v, 10); ok {
			return bi, true
		}
	case int64:
		return big.NewInt(v), true
	case int:
		return big.NewInt(int64(v)), true
	case float64:
		if v == math.Trunc(v) && !math.IsInf(v, 0) {
			bf, _ := new(big.Float).SetFloat64(v).Int(nil)
			return bf, true
		}
	case json.Number:
		bi := new(big.Int)
		if _, ok := bi.SetString(string(v), 10); ok {
			return bi, true
		}
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "bigint",
		Input:    in,
	})
	return nil, false
}

// @spec REQ-PRIM-007
// BigInt accepts arbitrary precision integers (*big.Int, string, json.Number).
func BigInt() *BigIntSchema {
	return &BigIntSchema{
		BaseSchema: NewBaseSchema[*big.Int](&bigIntNode{}),
	}
}

func (s *BigIntSchema) Optional() OptionalSchema[*big.Int] {
	return NewOptionalSchema[*big.Int](s)
}
func (s *BigIntSchema) Nullable() NullableSchema[*big.Int] {
	return NewNullableSchema[*big.Int](s)
}
func (s *BigIntSchema) Nullish() NullishSchema[*big.Int] { return NewNullishSchema[*big.Int](s) }
func (s *BigIntSchema) Default(v *big.Int) DefaultSchema[*big.Int] {
	return NewDefaultSchema[*big.Int](s, v)
}
func (s *BigIntSchema) Array() *ArraySchema[*big.Int] { return NewArraySchema[*big.Int](s) }
func (s *BigIntSchema) Meta(fn func(m *MetaBuilder)) *BigIntSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *NumberSchema[T]) Error(msg string) *NumberSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *NumberSchema[T]) ErrorFunc(fn func(issue *Issue) string) *NumberSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *BigIntSchema) Error(msg string) *BigIntSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *BigIntSchema) ErrorFunc(fn func(issue *Issue) string) *BigIntSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
