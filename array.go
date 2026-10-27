package z

// @spec REQ-COLL-002
// ArraySchema validates and parses array collections of element type T.
type ArraySchema[T any] struct {
	BaseSchema[[]T]
	element Schema[T]
	min     *int
	max     *int
	length  *int
}

type arrayNode[T any] struct {
	schema *ArraySchema[T]
}

func (n *arrayNode[T]) kind() nodeKind { return kindArray }
func (n *arrayNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "array",
			Input:    in,
		})
		return nil, false
	}

	items, ok := in.([]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "array",
			Input:    in,
		})
		return nil, false
	}

	count := len(items)
	allOk := true

	if n.schema.length != nil && count != *n.schema.length {
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "array",
			Minimum:   *n.schema.length,
			Exact:     true,
			Inclusive: true,
			Input:     in,
		})
		allOk = false
	}
	if n.schema.min != nil && count < *n.schema.min {
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "array",
			Minimum:   *n.schema.min,
			Inclusive: true,
			Input:     in,
		})
		allOk = false
	}
	if n.schema.max != nil && count > *n.schema.max {
		ctx.addIssue(Issue{
			Code:      TooBig,
			Origin:    "array",
			Maximum:   *n.schema.max,
			Inclusive: true,
			Input:     in,
		})
		allOk = false
	}

	res := make([]T, count)
	for i, item := range items {
		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}
		ctx.pushPath(Index(i))
		out, ok := n.schema.element.node().parse(ctx, item)
		ctx.popPath()
		if !ok {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}
		if typed, ok := out.(T); ok {
			res[i] = typed
		}
	}

	if !allOk {
		return nil, false
	}
	return res, true
}

// @spec REQ-COLL-002
// Array creates a schema validating array inputs against the given element schema.
func Array[T any](item Schema[T]) *ArraySchema[T] {
	s := &ArraySchema[T]{
		element: item,
	}
	s.BaseSchema = NewBaseSchema[[]T](&arrayNode[T]{schema: s})
	return s
}

// NewArraySchema creates an ArraySchema pointer for modifier chaining.
func NewArraySchema[T any](element Schema[T]) *ArraySchema[T] {
	return Array(element)
}

func (s *ArraySchema[T]) clone() *ArraySchema[T] {
	c := &ArraySchema[T]{
		element: s.element,
		min:     s.min,
		max:     s.max,
		length:  s.length,
	}
	c.BaseSchema = NewBaseSchema[[]T](&arrayNode[T]{schema: c})
	c.SetMeta(s.GetMeta())
	return c
}

// @spec REQ-COLL-002
// Min enforces a minimum number of elements.
func (s *ArraySchema[T]) Min(n int) *ArraySchema[T] {
	c := s.clone()
	c.min = &n
	return c
}

// @spec REQ-COLL-002
// Max enforces a maximum number of elements.
func (s *ArraySchema[T]) Max(n int) *ArraySchema[T] {
	c := s.clone()
	c.max = &n
	return c
}

// @spec REQ-COLL-002
// Length enforces an exact number of elements.
func (s *ArraySchema[T]) Length(n int) *ArraySchema[T] {
	c := s.clone()
	c.length = &n
	return c
}

// @spec REQ-COLL-002
// NonEmpty enforces that the array contains at least 1 element.
func (s *ArraySchema[T]) NonEmpty() *ArraySchema[T] {
	return s.Min(1)
}

// @spec REQ-COLL-002
// Element returns the underlying element schema.
func (s *ArraySchema[T]) Element() Schema[T] {
	return s.element
}

// @spec REQ-COLL-002
// Unwrap returns the underlying element schema (matching Zod v4).
func (s *ArraySchema[T]) Unwrap() Schema[T] {
	return s.element
}

// @spec REQ-COLL-006
func (s *ArraySchema[T]) Optional() OptionalSchema[[]T] { return NewOptionalSchema[[]T](s) }
func (s *ArraySchema[T]) Nullable() NullableSchema[[]T] { return NewNullableSchema[[]T](s) }
func (s *ArraySchema[T]) Nullish() NullishSchema[[]T]   { return NewNullishSchema[[]T](s) }
func (s *ArraySchema[T]) Default(v []T) DefaultSchema[[]T] {
	return NewDefaultSchema[[]T](s, v)
}
func (s *ArraySchema[T]) Meta(fn func(m *MetaBuilder)) *ArraySchema[T] {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}
