package z

import (
	"encoding/base64"
	"slices"
	"time"
)

// --- Boolean ---

// @spec REQ-PRIM-001
type BooleanSchema struct {
	BaseSchema[bool]
}

type booleanNode struct{}

func (n *booleanNode) kind() nodeKind { return kindBoolean }
func (n *booleanNode) parse(ctx *parseCtx, in any) (any, bool) {
	if b, ok := in.(bool); ok {
		return b, true
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "boolean",
		Input:    in,
	})
	return false, false
}

// @spec REQ-PRIM-001
// Boolean validates boolean values (true or false).
func Boolean() *BooleanSchema {
	s := &BooleanSchema{
		BaseSchema: NewBaseSchema[bool](&booleanNode{}),
	}
	return s
}

func (s *BooleanSchema) Optional() OptionalSchema[bool] { return NewOptionalSchema[bool](s) }
func (s *BooleanSchema) Nullable() NullableSchema[bool] { return NewNullableSchema[bool](s) }
func (s *BooleanSchema) Nullish() NullishSchema[bool]   { return NewNullishSchema[bool](s) }
func (s *BooleanSchema) Default(v bool) DefaultSchema[bool] {
	return NewDefaultSchema[bool](s, v)
}
func (s *BooleanSchema) Array() *ArraySchema[bool] { return NewArraySchema[bool](s) }
func (s *BooleanSchema) Meta(fn func(m *MetaBuilder)) *BooleanSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// --- Null ---

// @spec REQ-PRIM-001
type NullSchema struct {
	BaseSchema[any]
}

type nullNode struct{}

func (n *nullNode) kind() nodeKind { return kindNullable }
func (n *nullNode) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return nil, true
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "null",
		Input:    in,
	})
	return nil, false
}

// @spec REQ-PRIM-001
// Null validates that the input is JSON null, producing nil output.
func Null() *NullSchema {
	return &NullSchema{
		BaseSchema: NewBaseSchema[any](&nullNode{}),
	}
}

func (s *NullSchema) Optional() OptionalSchema[any]    { return NewOptionalSchema[any](s) }
func (s *NullSchema) Nullable() NullableSchema[any]    { return NewNullableSchema[any](s) }
func (s *NullSchema) Nullish() NullishSchema[any]      { return NewNullishSchema[any](s) }
func (s *NullSchema) Default(v any) DefaultSchema[any] { return NewDefaultSchema[any](s, v) }
func (s *NullSchema) Array() *ArraySchema[any]         { return NewArraySchema[any](s) }
func (s *NullSchema) Meta(fn func(m *MetaBuilder)) *NullSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// --- Any & Unknown ---

// @spec REQ-PRIM-001
type AnyPrimitiveSchema struct {
	BaseSchema[any]
}

type anyNode struct{}

func (n *anyNode) kind() nodeKind { return kindUnknown }
func (n *anyNode) parse(ctx *parseCtx, in any) (any, bool) {
	return in, true
}

// @spec REQ-PRIM-001
// Any accepts any input value without validation.
func Any() *AnyPrimitiveSchema {
	return &AnyPrimitiveSchema{
		BaseSchema: NewBaseSchema[any](&anyNode{}),
	}
}

func (s *AnyPrimitiveSchema) Optional() OptionalSchema[any]    { return NewOptionalSchema[any](s) }
func (s *AnyPrimitiveSchema) Nullable() NullableSchema[any]    { return NewNullableSchema[any](s) }
func (s *AnyPrimitiveSchema) Nullish() NullishSchema[any]      { return NewNullishSchema[any](s) }
func (s *AnyPrimitiveSchema) Default(v any) DefaultSchema[any] { return NewDefaultSchema[any](s, v) }
func (s *AnyPrimitiveSchema) Array() *ArraySchema[any]         { return NewArraySchema[any](s) }
func (s *AnyPrimitiveSchema) Meta(fn func(m *MetaBuilder)) *AnyPrimitiveSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// @spec REQ-PRIM-001
// Unknown accepts any input value, identically to Any.
func Unknown() *AnyPrimitiveSchema {
	return Any()
}

// --- Never ---

// @spec REQ-PRIM-001
type NeverSchema struct {
	BaseSchema[any]
}

type neverNode struct{}

func (n *neverNode) kind() nodeKind { return kindUnknown }
func (n *neverNode) parse(ctx *parseCtx, in any) (any, bool) {
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "never",
		Input:    in,
	})
	return nil, false
}

// @spec REQ-PRIM-001
// Never rejects any input, always producing an InvalidType issue.
func Never() *NeverSchema {
	return &NeverSchema{
		BaseSchema: NewBaseSchema[any](&neverNode{}),
	}
}

func (s *NeverSchema) Optional() OptionalSchema[any] { return NewOptionalSchema[any](s) }
func (s *NeverSchema) Nullable() NullableSchema[any] { return NewNullableSchema[any](s) }
func (s *NeverSchema) Nullish() NullishSchema[any]   { return NewNullishSchema[any](s) }
func (s *NeverSchema) Meta(fn func(m *MetaBuilder)) *NeverSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// --- Date ---

// @spec REQ-PRIM-001
type DateSchema struct {
	BaseSchema[time.Time]
}

type dateNode struct {
	min *time.Time
	max *time.Time
}

func (n *dateNode) kind() nodeKind { return kindDate }
func (n *dateNode) parse(ctx *parseCtx, in any) (any, bool) {
	switch v := in.(type) {
	case time.Time:
		return v, n.checkRange(ctx, v)
	case string:
		// RFC 3339 parsing already accepts fractional seconds.
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t, n.checkRange(ctx, t)
		}
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "date",
		Input:    in,
	})
	return time.Time{}, false
}

// @spec REQ-PRIM-001
// Date accepts time.Time instances or RFC 3339 formatted strings.
func Date() *DateSchema {
	return &DateSchema{
		BaseSchema: NewBaseSchema[time.Time](&dateNode{}),
	}
}

func (s *DateSchema) Optional() OptionalSchema[time.Time] {
	return NewOptionalSchema[time.Time](s)
}
func (s *DateSchema) Nullable() NullableSchema[time.Time] {
	return NewNullableSchema[time.Time](s)
}
func (s *DateSchema) Nullish() NullishSchema[time.Time] { return NewNullishSchema[time.Time](s) }
func (s *DateSchema) Default(v time.Time) DefaultSchema[time.Time] {
	return NewDefaultSchema[time.Time](s, v)
}
func (s *DateSchema) Array() *ArraySchema[time.Time] { return NewArraySchema[time.Time](s) }
func (s *DateSchema) Meta(fn func(m *MetaBuilder)) *DateSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// --- Bytes ---

// @spec REQ-PRIM-001
type BytesSchema struct {
	BaseSchema[[]byte]
}

type bytesNode struct{}

func (n *bytesNode) kind() nodeKind { return kindBytes }
func (n *bytesNode) parse(ctx *parseCtx, in any) (any, bool) {
	switch v := in.(type) {
	case []byte:
		return v, true
	case string:
		data, err := base64.StdEncoding.DecodeString(v)
		if err == nil {
			return data, true
		}
	}
	ctx.addIssue(Issue{
		Code:     InvalidType,
		Expected: "bytes",
		Input:    in,
	})
	return nil, false
}

// @spec REQ-PRIM-001
// Bytes accepts []byte slices or base64 encoded strings.
func Bytes() *BytesSchema {
	return &BytesSchema{
		BaseSchema: NewBaseSchema[[]byte](&bytesNode{}),
	}
}

func (s *BytesSchema) Optional() OptionalSchema[[]byte] {
	return NewOptionalSchema[[]byte](s)
}
func (s *BytesSchema) Nullable() NullableSchema[[]byte] {
	return NewNullableSchema[[]byte](s)
}
func (s *BytesSchema) Nullish() NullishSchema[[]byte] { return NewNullishSchema[[]byte](s) }
func (s *BytesSchema) Default(v []byte) DefaultSchema[[]byte] {
	return NewDefaultSchema[[]byte](s, v)
}
func (s *BytesSchema) Array() *ArraySchema[[]byte] { return NewArraySchema[[]byte](s) }
func (s *BytesSchema) Meta(fn func(m *MetaBuilder)) *BytesSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// --- Literal ---

// @spec REQ-PRIM-001
type LiteralSchema[T comparable] struct {
	BaseSchema[T]
	values []T
}

type literalNode[T comparable] struct {
	values []T
}

func (n *literalNode[T]) kind() nodeKind { return kindLiteral }
func (n *literalNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if val, ok := in.(T); ok {
		for _, expected := range n.values {
			if val == expected {
				return val, true
			}
		}
	}
	var anyVals []any
	for _, v := range n.values {
		anyVals = append(anyVals, v)
	}
	ctx.addIssue(Issue{
		Code:   InvalidValue,
		Values: anyVals,
		Input:  in,
	})
	var zero T
	return zero, false
}

// @spec REQ-PRIM-001
// Literal matches exact values of a comparable type T.
func Literal[T comparable](v ...T) *LiteralSchema[T] {
	cloned := slices.Clone(v)
	node := &literalNode[T]{values: cloned}
	return &LiteralSchema[T]{
		BaseSchema: NewBaseSchema[T](node),
		values:     cloned,
	}
}

func (s *LiteralSchema[T]) Values() []T {
	return slices.Clone(s.values)
}

func (s *LiteralSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *LiteralSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *LiteralSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *LiteralSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *LiteralSchema[T]) Array() *ArraySchema[T] { return NewArraySchema[T](s) }
func (s *LiteralSchema[T]) Meta(fn func(m *MetaBuilder)) *LiteralSchema[T] {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *BooleanSchema) Error(msg string) *BooleanSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *BooleanSchema) ErrorFunc(fn func(issue *Issue) string) *BooleanSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *NullSchema) Error(msg string) *NullSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *NullSchema) ErrorFunc(fn func(issue *Issue) string) *NullSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *AnyPrimitiveSchema) Error(msg string) *AnyPrimitiveSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *AnyPrimitiveSchema) ErrorFunc(fn func(issue *Issue) string) *AnyPrimitiveSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *NeverSchema) Error(msg string) *NeverSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *NeverSchema) ErrorFunc(fn func(issue *Issue) string) *NeverSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *DateSchema) Error(msg string) *DateSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *DateSchema) ErrorFunc(fn func(issue *Issue) string) *DateSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *BytesSchema) Error(msg string) *BytesSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *BytesSchema) ErrorFunc(fn func(issue *Issue) string) *BytesSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *LiteralSchema[T]) Error(msg string) *LiteralSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *LiteralSchema[T]) ErrorFunc(fn func(issue *Issue) string) *LiteralSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

func (n *dateNode) checkRange(ctx *parseCtx, t time.Time) bool {
	ok := true
	if n.min != nil && t.Before(*n.min) {
		ctx.addIssue(Issue{Code: TooSmall, Origin: "date", Minimum: *n.min, Inclusive: true, Input: t})
		ok = false
	}
	if n.max != nil && t.After(*n.max) {
		ctx.addIssue(Issue{Code: TooBig, Origin: "date", Maximum: *n.max, Inclusive: true, Input: t})
		ok = false
	}
	return ok
}

// Min requires the date to be at or after t (Zod's `z.date().min(t)`).
func (s *DateSchema) Min(t time.Time) *DateSchema {
	c := *s
	n := *s.innerNode.(*dateNode)
	n.min = &t
	c.innerNode = &n
	return &c
}

// Max requires the date to be at or before t (Zod's `z.date().max(t)`).
func (s *DateSchema) Max(t time.Time) *DateSchema {
	c := *s
	n := *s.innerNode.(*dateNode)
	n.max = &t
	c.innerNode = &n
	return &c
}
