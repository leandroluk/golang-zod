package z

// @spec REQ-COLL-004
// SetSchema validates sets of unique items, returning map[T]struct{}.
type SetSchema[T comparable] struct {
	BaseSchema[map[T]struct{}]
	element Schema[T]
	min     *int
	max     *int
	size    *int
}

type setNode[T comparable] struct {
	schema *SetSchema[T]
}

func (n *setNode[T]) kind() nodeKind { return kindSet }
func (n *setNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "set",
			Input:    in,
		})
		return nil, false
	}

	items, ok := in.([]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "set",
			Input:    in,
		})
		return nil, false
	}

	res := make(map[T]struct{})
	allOk := true

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
			res[typed] = struct{}{}
		}
	}

	if !allOk {
		return nil, false
	}

	// Size checks apply to unique items collapsed
	count := len(res)
	if n.schema.size != nil && count != *n.schema.size {
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "set",
			Minimum:   *n.schema.size,
			Exact:     true,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}
	if n.schema.min != nil && count < *n.schema.min {
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "set",
			Minimum:   *n.schema.min,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}
	if n.schema.max != nil && count > *n.schema.max {
		ctx.addIssue(Issue{
			Code:      TooBig,
			Origin:    "set",
			Maximum:   *n.schema.max,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}

	return res, true
}

// @spec REQ-COLL-004
// Set creates a schema validating a set of comparable items, collapsing duplicates into map[T]struct{}.
func Set[T comparable](item Schema[T]) *SetSchema[T] {
	s := &SetSchema[T]{
		element: item,
	}
	s.BaseSchema = NewBaseSchema[map[T]struct{}](&setNode[T]{schema: s})
	return s
}

func (s *SetSchema[T]) clone() *SetSchema[T] {
	c := &SetSchema[T]{
		element: s.element,
		min:     s.min,
		max:     s.max,
		size:    s.size,
	}
	c.BaseSchema = NewBaseSchema[map[T]struct{}](&setNode[T]{schema: c})
	c.SetMeta(s.GetMeta())
	return c
}

// @spec REQ-COLL-004
// Min enforces a minimum number of unique items.
func (s *SetSchema[T]) Min(n int) *SetSchema[T] {
	c := s.clone()
	c.min = &n
	return c
}

// @spec REQ-COLL-004
// Max enforces a maximum number of unique items.
func (s *SetSchema[T]) Max(n int) *SetSchema[T] {
	c := s.clone()
	c.max = &n
	return c
}

// @spec REQ-COLL-004
// Size enforces an exact number of unique items.
func (s *SetSchema[T]) Size(n int) *SetSchema[T] {
	c := s.clone()
	c.size = &n
	return c
}

// @spec REQ-COLL-004
// NonEmpty enforces that the set contains at least 1 unique item.
func (s *SetSchema[T]) NonEmpty() *SetSchema[T] {
	return s.Min(1)
}

// @spec REQ-COLL-006
func (s *SetSchema[T]) Optional() OptionalSchema[map[T]struct{}] {
	return NewOptionalSchema[map[T]struct{}](s)
}
func (s *SetSchema[T]) Nullable() NullableSchema[map[T]struct{}] {
	return NewNullableSchema[map[T]struct{}](s)
}
func (s *SetSchema[T]) Nullish() NullishSchema[map[T]struct{}] {
	return NewNullishSchema[map[T]struct{}](s)
}
func (s *SetSchema[T]) Default(v map[T]struct{}) DefaultSchema[map[T]struct{}] {
	return NewDefaultSchema[map[T]struct{}](s, v)
}
func (s *SetSchema[T]) Array() *ArraySchema[map[T]struct{}] {
	return NewArraySchema[map[T]struct{}](s)
}
func (s *SetSchema[T]) Meta(fn func(m *MetaBuilder)) *SetSchema[T] {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *SetSchema[T]) Error(msg string) *SetSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *SetSchema[T]) ErrorFunc(fn func(issue *Issue) string) *SetSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
