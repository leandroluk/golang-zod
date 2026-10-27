package z

// @spec REQ-PRIM-009
// OptionalSchema wraps an inner schema to make values optional, returning Option[T].
type OptionalSchema[T any] struct {
	BaseSchema[Option[T]]
	inner Schema[T]
}

type optionalNode[T any] struct {
	inner node
}

func (n *optionalNode[T]) kind() nodeKind { return kindOptional }
func (n *optionalNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		// When input is nil in direct parse, Optional rejects null per Zod v4 unless Nullable
		// But in object key context, absent keys produce None
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "non-null",
		})
		return nil, false
	}
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	if typed, ok := out.(T); ok {
		return Some(typed), true
	}
	return None[T](), false
}

// NewOptionalSchema creates an OptionalSchema wrapping inner.
func NewOptionalSchema[T any](inner Schema[T]) OptionalSchema[T] {
	node := &optionalNode[T]{inner: inner.node()}
	return OptionalSchema[T]{
		BaseSchema: NewBaseSchema[Option[T]](node),
		inner:      inner,
	}
}

// Unwrap returns the underlying schema.
func (s OptionalSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PRIM-009
// NullableSchema wraps an inner schema to accept null values, returning Nullable[T].
type NullableSchema[T any] struct {
	BaseSchema[Nullable[T]]
	inner Schema[T]
}

type nullableNode[T any] struct {
	inner node
}

func (n *nullableNode[T]) kind() nodeKind { return kindNullable }
func (n *nullableNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return NullableNull[T](), true
	}
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	if typed, ok := out.(T); ok {
		return NullableOf(typed), true
	}
	return NullableNull[T](), false
}

// NewNullableSchema creates a NullableSchema wrapping inner.
func NewNullableSchema[T any](inner Schema[T]) NullableSchema[T] {
	node := &nullableNode[T]{inner: inner.node()}
	return NullableSchema[T]{
		BaseSchema: NewBaseSchema[Nullable[T]](node),
		inner:      inner,
	}
}

// Unwrap returns the underlying schema.
func (s NullableSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PRIM-009
// NullishSchema wraps an inner schema to accept absence and null, returning Option[Nullable[T]].
type NullishSchema[T any] struct {
	BaseSchema[Option[Nullable[T]]]
	inner Schema[T]
}

type nullishNode[T any] struct {
	inner node
}

func (n *nullishNode[T]) kind() nodeKind { return kindNullish }
func (n *nullishNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return Some(NullableNull[T]()), true
	}
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	if typed, ok := out.(T); ok {
		return Some(NullableOf(typed)), true
	}
	return None[Nullable[T]](), false
}

// NewNullishSchema creates a NullishSchema wrapping inner.
func NewNullishSchema[T any](inner Schema[T]) NullishSchema[T] {
	node := &nullishNode[T]{inner: inner.node()}
	return NullishSchema[T]{
		BaseSchema: NewBaseSchema[Option[Nullable[T]]](node),
		inner:      inner,
	}
}

// Unwrap returns the underlying schema.
func (s NullishSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PRIM-009
// DefaultSchema wraps an inner schema to apply a fallback value when input is absent.
type DefaultSchema[T any] struct {
	BaseSchema[T]
	inner        Schema[T]
	defaultValue T
}

type defaultNode[T any] struct {
	inner        node
	defaultValue T
}

func (n *defaultNode[T]) kind() nodeKind { return kindDefault }
func (n *defaultNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return n.defaultValue, true
	}
	return n.inner.parse(ctx, in)
}

// NewDefaultSchema creates a DefaultSchema wrapping inner with defaultValue.
func NewDefaultSchema[T any](inner Schema[T], defaultValue T) DefaultSchema[T] {
	node := &defaultNode[T]{inner: inner.node(), defaultValue: defaultValue}
	return DefaultSchema[T]{
		BaseSchema:   NewBaseSchema[T](node),
		inner:        inner,
		defaultValue: defaultValue,
	}
}

// Unwrap returns the underlying schema.
func (s DefaultSchema[T]) Unwrap() Schema[T] {
	return s.inner
}
