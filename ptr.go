package z

// PtrSchema wraps an inner schema so absent or null input yields a nil *T and any other
// input yields a pointer to the parsed value. It fits idiomatic Go structs such as
// `DeletedAt *time.Time` without Option/Nullable wrappers.
type PtrSchema[T any] struct {
	BaseSchema[*T]
	inner Schema[T]
}

type ptrNode[T any] struct {
	inner  node
	schema AnySchema
}

func (n *ptrNode[T]) kind() nodeKind    { return kindNullish }
func (n *ptrNode[T]) Unwrap() AnySchema { return n.schema }
func (n *ptrNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return (*T)(nil), true
	}
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	if typed, ok := out.(T); ok {
		return &typed, true
	}
	return (*T)(nil), false
}

// Ptr wraps inner so the result is a *T, nil when the input is absent or null.
func Ptr[T any](inner Schema[T]) *PtrSchema[T] {
	n := &ptrNode[T]{inner: inner.node(), schema: inner}
	res := &PtrSchema[T]{
		BaseSchema: NewBaseSchema[*T](n),
		inner:      inner,
	}
	inheritSlot(&res.BaseSchema, inner, res)
	return res
}

// Unwrap returns the underlying schema.
func (s *PtrSchema[T]) Unwrap() Schema[T] { return s.inner }
