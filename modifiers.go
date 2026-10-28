package z

// @spec REQ-PRIM-009
// @spec REQ-PIPE-001
// OptionalSchema wraps an inner schema to make values optional, returning Option[T].
type OptionalSchema[T any] struct {
	BaseSchema[Option[T]]
	inner Schema[T]
}

type optionalNode[T any] struct {
	inner  node
	schema AnySchema
}

func (n *optionalNode[T]) kind() nodeKind    { return kindOptional }
func (n *optionalNode[T]) Unwrap() AnySchema { return n.schema }
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
	node := &optionalNode[T]{inner: inner.node(), schema: inner}
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
// @spec REQ-PIPE-001
// NullableSchema wraps an inner schema to accept null values, returning Nullable[T].
type NullableSchema[T any] struct {
	BaseSchema[Nullable[T]]
	inner Schema[T]
}

type nullableNode[T any] struct {
	inner  node
	schema AnySchema
}

func (n *nullableNode[T]) kind() nodeKind    { return kindNullable }
func (n *nullableNode[T]) Unwrap() AnySchema { return n.schema }
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
	node := &nullableNode[T]{inner: inner.node(), schema: inner}
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
// @spec REQ-PIPE-001
// NullishSchema wraps an inner schema to accept absence and null, returning Option[Nullable[T]].
type NullishSchema[T any] struct {
	BaseSchema[Option[Nullable[T]]]
	inner Schema[T]
}

type nullishNode[T any] struct {
	inner  node
	schema AnySchema
}

func (n *nullishNode[T]) kind() nodeKind    { return kindNullish }
func (n *nullishNode[T]) Unwrap() AnySchema { return n.schema }
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
	node := &nullishNode[T]{inner: inner.node(), schema: inner}
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
// @spec REQ-PIPE-001
// DefaultSchema wraps an inner schema to apply a fallback value when input is absent.
type DefaultSchema[T any] struct {
	BaseSchema[T]
	inner     Schema[T]
	defaultFn func() T
}

type defaultNode[T any] struct {
	inner     node
	schema    AnySchema
	defaultFn func() T
}

func (n *defaultNode[T]) kind() nodeKind    { return kindDefault }
func (n *defaultNode[T]) Unwrap() AnySchema { return n.schema }
func (n *defaultNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return n.defaultFn(), true
	}
	return n.inner.parse(ctx, in)
}

// NewDefaultSchema creates a DefaultSchema wrapping inner with defaultValue.
func NewDefaultSchema[T any](inner Schema[T], defaultValue T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](inner, func() T { return defaultValue })
}

// NewDefaultFuncSchema creates a DefaultSchema wrapping inner with a dynamic fallback supplier.
func NewDefaultFuncSchema[T any](inner Schema[T], fn func() T) DefaultSchema[T] {
	node := &defaultNode[T]{inner: inner.node(), schema: inner, defaultFn: fn}
	return DefaultSchema[T]{
		BaseSchema: NewBaseSchema[T](node),
		inner:      inner,
		defaultFn:  fn,
	}
}

// Unwrap returns the underlying schema.
func (s DefaultSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PIPE-001
// PrefaultSchema substitutes input before running it through inner validation.
type PrefaultSchema[T any] struct {
	BaseSchema[T]
	inner      Schema[T]
	prefaultFn func() T
}

type prefaultNode[T any] struct {
	schema     AnySchema
	inner      node
	prefaultFn func() T
}

func (n *prefaultNode[T]) kind() nodeKind    { return kindPrefault }
func (n *prefaultNode[T]) Unwrap() AnySchema { return n.schema }
func (n *prefaultNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		in = n.prefaultFn()
	}
	return n.inner.parse(ctx, in)
}

// NewPrefaultSchema creates a PrefaultSchema wrapping inner with prefaultValue.
func NewPrefaultSchema[T any](inner Schema[T], prefaultValue T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](inner, func() T { return prefaultValue })
}

// NewPrefaultFuncSchema creates a PrefaultSchema wrapping inner with a dynamic supplier.
func NewPrefaultFuncSchema[T any](inner Schema[T], fn func() T) PrefaultSchema[T] {
	node := &prefaultNode[T]{inner: inner.node(), schema: inner, prefaultFn: fn}
	return PrefaultSchema[T]{
		BaseSchema: NewBaseSchema[T](node),
		inner:      inner,
		prefaultFn: fn,
	}
}

// Unwrap returns the underlying schema.
func (s PrefaultSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PIPE-001
// CatchSchema catches validation failure and supplies a fallback value.
type CatchSchema[T any] struct {
	BaseSchema[T]
	inner   Schema[T]
	catchFn func(issues []Issue) T
}

type catchNode[T any] struct {
	schema  AnySchema
	inner   node
	catchFn func(issues []Issue) T
}

func (n *catchNode[T]) kind() nodeKind    { return kindCatch }
func (n *catchNode[T]) Unwrap() AnySchema { return n.schema }
func (n *catchNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	initialIssuesLen := len(ctx.issues)
	out, ok := n.inner.parse(ctx, in)
	if ok && len(ctx.issues) == initialIssuesLen {
		return out, true
	}

	// Capture newly added issues for catchFn, then rewind
	var newIssues []Issue
	if len(ctx.issues) > initialIssuesLen {
		newIssues = append(newIssues, ctx.issues[initialIssuesLen:]...)
		ctx.issues = ctx.issues[:initialIssuesLen]
	}

	return n.catchFn(newIssues), true
}

// NewCatchSchema creates a CatchSchema with fixed fallback.
func NewCatchSchema[T any](inner Schema[T], fallback T) CatchSchema[T] {
	return NewCatchFuncSchema[T](inner, func([]Issue) T { return fallback })
}

// NewCatchFuncSchema creates a CatchSchema with dynamic issue-aware fallback.
func NewCatchFuncSchema[T any](inner Schema[T], fn func(issues []Issue) T) CatchSchema[T] {
	node := &catchNode[T]{inner: inner.node(), schema: inner, catchFn: fn}
	return CatchSchema[T]{
		BaseSchema: NewBaseSchema[T](node),
		inner:      inner,
		catchFn:    fn,
	}
}

// Unwrap returns the underlying schema.
func (s CatchSchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// @spec REQ-PIPE-001
// ReadonlySchema marks a schema as readOnly for documentation and JSON schema export.
type ReadonlySchema[T any] struct {
	BaseSchema[T]
	inner Schema[T]
}

type readonlyNode[T any] struct {
	schema AnySchema
	inner  node
}

func (n *readonlyNode[T]) kind() nodeKind    { return kindReadonly }
func (n *readonlyNode[T]) Unwrap() AnySchema { return n.schema }
func (n *readonlyNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	return n.inner.parse(ctx, in)
}

// NewReadonlySchema creates a ReadonlySchema wrapping inner.
func NewReadonlySchema[T any](inner Schema[T]) ReadonlySchema[T] {
	node := &readonlyNode[T]{inner: inner.node(), schema: inner}
	s := ReadonlySchema[T]{
		BaseSchema: NewBaseSchema[T](node),
		inner:      inner,
	}
	m := inner.GetMeta()
	m.ReadOnly = true
	s.SetMeta(m)
	return s
}

// Unwrap returns the underlying schema.
func (s ReadonlySchema[T]) Unwrap() Schema[T] {
	return s.inner
}

// parseAbsent evaluates a Default/Prefault/Catch node for an object key that is absent.
// Catch yields its fallback directly (absence is never treated as a null input).
func parseAbsent(ctx *parseCtx, n node) (any, bool) {
	if c, ok := n.(interface{ absent() any }); ok {
		return c.absent(), true
	}
	return n.parse(ctx, nil)
}

func (n *catchNode[T]) absent() any {
	return n.catchFn([]Issue{{Code: InvalidType, Expected: "required"}})
}
