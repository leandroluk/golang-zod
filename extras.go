package z

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// --- Custom ---

// CustomSchema validates input with a user predicate and yields it as T.
type CustomSchema[T any] struct {
	BaseSchema[T]
}

type customNode[T any] struct {
	fn  func(v any) bool
	msg string
}

func (n *customNode[T]) kind() nodeKind { return kindUnknown }
func (n *customNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	typed, isT := in.(T)
	if !isT || !n.fn(in) {
		ctx.addIssue(Issue{Code: Custom, Message: n.msg, Input: in})
		return nil, false
	}
	return typed, true
}

// CustomFn builds a schema from a predicate (Zod's `z.custom<T>(fn)`); input must also be assignable to T.
func CustomFn[T any](fn func(v any) bool, message ...string) *CustomSchema[T] {
	msg := "Invalid input"
	if len(message) > 0 {
		msg = message[0]
	}
	return &CustomSchema[T]{BaseSchema: NewBaseSchema[T](&customNode[T]{fn: fn, msg: msg})}
}

// --- NaN ---

// NaNSchema accepts only float NaN values.
type NaNSchema struct {
	BaseSchema[float64]
}

type nanNode struct{}

func (n *nanNode) kind() nodeKind { return kindUnknown }
func (n *nanNode) parse(ctx *parseCtx, in any) (any, bool) {
	if f, ok := in.(float64); ok && math.IsNaN(f) {
		return f, true
	}
	ctx.addIssue(Issue{Code: InvalidType, Expected: "nan", Input: in})
	return math.NaN(), false
}

// NaN accepts only NaN (Zod's `z.nan()`).
func NaN() *NaNSchema { return &NaNSchema{BaseSchema: NewBaseSchema[float64](&nanNode{})} }

// --- JSON ---

// JSONSchema accepts any JSON-compatible value (null, bool, numbers, string, []any, map[string]any).
type JSONSchema struct {
	BaseSchema[any]
}

type jsonValueNode struct{}

func (n *jsonValueNode) kind() nodeKind { return kindUnknown }
func (n *jsonValueNode) parse(ctx *parseCtx, in any) (any, bool) {
	if isJSONValue(in) {
		return in, true
	}
	ctx.addIssue(Issue{Code: InvalidType, Expected: "json", Input: in})
	return nil, false
}

func isJSONValue(v any) bool {
	switch t := v.(type) {
	case nil, bool, string,
		int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		return true
	case float64:
		return !math.IsNaN(t) && !math.IsInf(t, 0)
	case []any:
		for _, e := range t {
			if !isJSONValue(e) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, e := range t {
			if !isJSONValue(e) {
				return false
			}
		}
		return true
	}
	return false
}

// JSON accepts any JSON-compatible value (Zod's `z.json()`).
func JSON() *JSONSchema { return &JSONSchema{BaseSchema: NewBaseSchema[any](&jsonValueNode{})} }

// --- Success ---

// SuccessSchema reports whether the inner schema accepted the input, never failing itself.
type SuccessSchema struct {
	BaseSchema[bool]
}

type successNode struct{ inner node }

func (n *successNode) kind() nodeKind { return kindUnknown }
func (n *successNode) parse(ctx *parseCtx, in any) (any, bool) {
	before := len(ctx.issues)
	_, ok := n.inner.parse(ctx, in)
	passed := ok && len(ctx.issues) == before
	ctx.issues = ctx.issues[:before]
	return passed, true
}

// Success wraps s so parsing yields true/false instead of failing (Zod's `z.success(s)`).
func Success[T any](s Schema[T]) *SuccessSchema {
	return &SuccessSchema{BaseSchema: NewBaseSchema[bool](&successNode{inner: s.node()})}
}

// --- NonOptional ---

// NonOptionalSchema unwraps an Option[T] schema and rejects absence.
type NonOptionalSchema[T any] struct {
	BaseSchema[T]
}

type nonOptionalNode[T any] struct{ inner node }

func (n *nonOptionalNode[T]) kind() nodeKind { return kindUnknown }
func (n *nonOptionalNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	if opt, isOpt := out.(Option[T]); isOpt {
		if v, some := opt.Value(); some {
			return v, true
		}
		ctx.addIssue(Issue{Code: InvalidType, Expected: "nonoptional", Input: in})
		return nil, false
	}
	return out, true
}

// NonOptional makes an optional schema required again (Zod's `z.nonoptional(s)`).
func NonOptional[T any](s Schema[Option[T]]) *NonOptionalSchema[T] {
	return &NonOptionalSchema[T]{BaseSchema: NewBaseSchema[T](&nonOptionalNode[T]{inner: s.node()})}
}

// ExactOptional is Optional for Go, where absence (None) is already distinct from null (Zod's `z.exactOptional`).
func ExactOptional[T any](s Schema[T]) OptionalSchema[T] { return NewOptionalSchema[T](s) }

// --- InvertCodec ---

// InvertCodec swaps decode and encode directions of a codec (Zod's `z.invertCodec`).
func InvertCodec[A, B any](c *CodecSchema[A, B]) *CodecSchema[B, A] {
	if c.outSchema == nil {
		panic("golang-zod: InvertCodec requires a codec with an output schema")
	}
	enc := c.encodeFn
	if enc == nil {
		enc = func(B) (A, error) {
			var zero A
			return zero, ErrEncodeUnsupported
		}
	}
	return Codec[B, A](c.outSchema, c.inSchema, enc, c.decodeFn)
}

// --- TemplateLiteral ---

// TemplateLiteral builds a string schema from literal strings and schemas (Zod's `z.templateLiteral([...])`).
// Parts may be string literals, *StringSchema, *BooleanSchema, *EnumSchema[string], *LiteralSchema[string|int]
// and *NumberSchema[int|int64|float64]; any other part panics.
func TemplateLiteral(parts ...any) *StringSchema {
	var b strings.Builder
	b.WriteString("^")
	for _, p := range parts {
		switch t := p.(type) {
		case string:
			b.WriteString(regexp.QuoteMeta(t))
		case *StringSchema:
			b.WriteString(`[\s\S]*?`)
		case *BooleanSchema:
			b.WriteString(`(?:true|false)`)
		case *NumberSchema[int], *NumberSchema[int64]:
			b.WriteString(`-?\d+`)
		case *NumberSchema[float64]:
			b.WriteString(`-?\d+(?:\.\d+)?`)
		case *EnumSchema[string]:
			b.WriteString(alternation(t.Values()))
		case *LiteralSchema[string]:
			b.WriteString(alternation(t.Values()))
		case *LiteralSchema[int]:
			vals := t.Values()
			strs := make([]string, len(vals))
			for i, v := range vals {
				strs[i] = fmt.Sprint(v)
			}
			b.WriteString(alternation(strs))
		default:
			panic(fmt.Sprintf("golang-zod: unsupported TemplateLiteral part %T", p))
		}
	}
	b.WriteString("$")
	re := regexp.MustCompile(b.String())
	return String().WithCustomCheck("template_literal", re.MatchString)
}

func alternation(vals []string) string {
	q := make([]string, len(vals))
	for i, v := range vals {
		q[i] = regexp.QuoteMeta(v)
	}
	return "(?:" + strings.Join(q, "|") + ")"
}

// --- Number extras ---

var maxSafeInteger = float64(1<<53 - 1)

// Step is an alias for MultipleOf matching Zod v4's `.step()`.
func (s *NumberSchema[T]) Step(divisor T, opts ...CheckOption) *NumberSchema[T] {
	return s.MultipleOf(divisor, opts...)
}

// Safe restricts the value to the JS safe integer range ±(2^53-1) (Zod's `.safe()`).
func (s *NumberSchema[T]) Safe(opts ...CheckOption) *NumberSchema[T] {
	safe := maxSafeInteger
	c := s
	if float64(minOf[T]()) < -maxSafeInteger {
		c = c.GTE(T(-safe), opts...)
	}
	if float64(maxOf[T]()) > maxSafeInteger {
		c = c.LTE(T(safe), opts...)
	}
	return c
}

// Finite is a no-op kept for Zod parity: number schemas already reject NaN and ±Inf.
func (s *NumberSchema[T]) Finite() *NumberSchema[T] { return s }
