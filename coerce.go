package z

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type coerceNamespace struct{}

// @spec REQ-PRIM-008
// Coerce provides explicit type coercion schemas matching Zod v4 z.coerce.
var Coerce coerceNamespace

// @spec REQ-PRIM-008
// String coerces numbers, booleans, and other values into string before validation.
func (coerceNamespace) String() *StringSchema {
	s := String()
	origNode := s.node()
	coercedNode := &coercedStringNode{inner: origNode}
	s.BaseSchema = NewBaseSchema[string](coercedNode)
	return s
}

type coercedStringNode struct {
	inner node
}

func (n *coercedStringNode) kind() nodeKind { return kindString }
func (n *coercedStringNode) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return n.inner.parse(ctx, "")
	}
	switch v := in.(type) {
	case string:
		return n.inner.parse(ctx, v)
	case int:
		return n.inner.parse(ctx, strconv.Itoa(v))
	case int64:
		return n.inner.parse(ctx, strconv.FormatInt(v, 10))
	case float64:
		return n.inner.parse(ctx, strconv.FormatFloat(v, 'f', -1, 64))
	case bool:
		return n.inner.parse(ctx, strconv.FormatBool(v))
	case json.Number:
		return n.inner.parse(ctx, string(v))
	default:
		return n.inner.parse(ctx, fmt.Sprintf("%v", v))
	}
}

// @spec REQ-PRIM-008
// Boolean coerces strings, numbers, and bools into bool.
func (coerceNamespace) Boolean() *BooleanSchema {
	node := &coercedBoolNode{}
	return &BooleanSchema{
		BaseSchema: NewBaseSchema[bool](node),
	}
}

type coercedBoolNode struct{}

func (n *coercedBoolNode) kind() nodeKind { return kindBoolean }
func (n *coercedBoolNode) parse(ctx *parseCtx, in any) (any, bool) {
	switch v := in.(type) {
	case bool:
		return v, true
	case int:
		return v != 0, true
	case int64:
		return v != 0, true
	case float64:
		return v != 0, true
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		switch lower {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off", "":
			return false, true
		}
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "boolean",
		Input:    in,
	})
	return false, false
}

// @spec REQ-PRIM-008
// Number coerces string or numeric input into float64.
func (coerceNamespace) Number() *NumberSchema[float64] {
	return (coerceNamespace{}).Float()
}

// @spec REQ-PRIM-008
// Float coerces string or numeric input into float64.
func (coerceNamespace) Float() *NumberSchema[float64] {
	s := Float64()
	orig := s.node()
	node := &coercedFloatNode{inner: orig}
	s.BaseSchema = NewBaseSchema[float64](node)
	return s
}

type coercedFloatNode struct {
	inner node
}

func (n *coercedFloatNode) kind() nodeKind { return kindNumber }
func (n *coercedFloatNode) parse(ctx *parseCtx, in any) (any, bool) {
	if s, ok := in.(string); ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return n.inner.parse(ctx, f)
		}
	}
	return n.inner.parse(ctx, in)
}

// @spec REQ-PRIM-008
// Int coerces string or numeric input into int.
func (coerceNamespace) Int() *NumberSchema[int] {
	s := Int()
	orig := s.node()
	node := &coercedIntNode{inner: orig}
	s.BaseSchema = NewBaseSchema[int](node)
	return s
}

type coercedIntNode struct {
	inner node
}

func (n *coercedIntNode) kind() nodeKind { return kindNumber }
func (n *coercedIntNode) parse(ctx *parseCtx, in any) (any, bool) {
	if s, ok := in.(string); ok {
		if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n.inner.parse(ctx, i)
		}
	}
	return n.inner.parse(ctx, in)
}

// @spec REQ-PRIM-008
// BigInt coerces string, json.Number or integers into *big.Int.
func (coerceNamespace) BigInt() *BigIntSchema {
	return BigInt()
}

// @spec REQ-PRIM-008
// Date coerces RFC 3339 strings or numeric epoch timestamps (seconds/milliseconds) into time.Time.
func (coerceNamespace) Date() *DateSchema {
	node := &coercedDateNode{}
	return &DateSchema{
		BaseSchema: NewBaseSchema[time.Time](node),
	}
}

type coercedDateNode struct{}

func (n *coercedDateNode) kind() nodeKind { return kindDate }
func (n *coercedDateNode) parse(ctx *parseCtx, in any) (any, bool) {
	switch v := in.(type) {
	case time.Time:
		return v, true
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, true
		}
		if millis, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.UnixMilli(millis), true
		}
	case int:
		i64 := int64(v)
		if i64 < 100_000_000_000 {
			return time.Unix(i64, 0), true
		}
		return time.UnixMilli(i64), true
	case int64:
		if v < 100_000_000_000 {
			return time.Unix(v, 0), true
		}
		return time.UnixMilli(v), true
	case float64:
		return time.UnixMilli(int64(v)), true
	}

	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "date",
		Input:    in,
	})
	return time.Time{}, false
}
