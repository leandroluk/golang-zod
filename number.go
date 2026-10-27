package z

import (
	"encoding/json"
	"math"
	"math/big"
	"slices"
	"strconv"
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
	val, ok := convertToNumber[T](in)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "number",
			Input:    in,
		})
		var zero T
		return zero, false
	}

	// Reject NaN and Inf for floats
	if f, isFloat := any(val).(float64); isFloat && (math.IsNaN(f) || math.IsInf(f, 0)) {
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

func convertToNumber[T NumberConstraint](in any) (T, bool) {
	var zero T
	switch v := in.(type) {
	case T:
		return v, true
	case int:
		return T(v), true
	case int64:
		return T(v), true
	case float64:
		return T(v), true
	case json.Number:
		str := string(v)
		// Check whether T is integer or float
		switch any(zero).(type) {
		case int, int8, int16, int32, int64:
			parsed, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return zero, false
			}
			return T(parsed), true
		case uint, uint8, uint16, uint32, uint64:
			parsed, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return zero, false
			}
			return T(parsed), true
		default:
			parsed, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return zero, false
			}
			return T(parsed), true
		}
	default:
		return zero, false
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
	c.SetMeta(s.GetMeta())
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
func (s *NumberSchema[T]) GT(min T) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkGT, val: min})
	return c
}

// @spec REQ-PRIM-007
// GTE enforces greater-than-or-equal (val >= min).
func (s *NumberSchema[T]) GTE(min T) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkGTE, val: min})
	return c
}

// @spec REQ-PRIM-007
// Min is an alias for GTE matching Zod v4.
func (s *NumberSchema[T]) Min(min T) *NumberSchema[T] {
	return s.GTE(min)
}

// @spec REQ-PRIM-007
// LT enforces strict less-than (val < max).
func (s *NumberSchema[T]) LT(max T) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkLT, val: max})
	return c
}

// @spec REQ-PRIM-007
// LTE enforces less-than-or-equal (val <= max).
func (s *NumberSchema[T]) LTE(max T) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkLTE, val: max})
	return c
}

// @spec REQ-PRIM-007
// Max is an alias for LTE matching Zod v4.
func (s *NumberSchema[T]) Max(max T) *NumberSchema[T] {
	return s.LTE(max)
}

// @spec REQ-PRIM-007
// Positive enforces val > 0.
func (s *NumberSchema[T]) Positive() *NumberSchema[T] {
	return s.GT(0)
}

// @spec REQ-PRIM-007
// NonNegative enforces val >= 0.
func (s *NumberSchema[T]) NonNegative() *NumberSchema[T] {
	return s.GTE(0)
}

// @spec REQ-PRIM-007
// Negative enforces val < 0.
func (s *NumberSchema[T]) Negative() *NumberSchema[T] {
	return s.LT(0)
}

// @spec REQ-PRIM-007
// NonPositive enforces val <= 0.
func (s *NumberSchema[T]) NonPositive() *NumberSchema[T] {
	return s.LTE(0)
}

// @spec REQ-PRIM-007
// MultipleOf checks that value is evenly divisible by divisor.
func (s *NumberSchema[T]) MultipleOf(divisor T) *NumberSchema[T] {
	c := s.clone()
	c.checks = append(c.checks, numCheck[T]{kind: checkMultipleOf, val: divisor})
	return c
}

// @spec REQ-PRIM-009
func (s *NumberSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *NumberSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *NumberSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *NumberSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *NumberSchema[T]) Array() ArraySchema[T] { return NewArraySchema[T](s) }
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
func (s *BigIntSchema) Array() ArraySchema[*big.Int] { return NewArraySchema[*big.Int](s) }
func (s *BigIntSchema) Meta(fn func(m *MetaBuilder)) *BigIntSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return &c
}
