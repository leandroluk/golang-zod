package z

// @spec REQ-COLL-005
// MapSchema validates map key-value collections, returning map[K]V.
type MapSchema[K comparable, V any] struct {
	BaseSchema[map[K]V]
	keySchema Schema[K]
	valSchema Schema[V]
	min       *int
	max       *int
	size      *int
}

type mapNode[K comparable, V any] struct {
	schema *MapSchema[K, V]
}

func (n *mapNode[K, V]) kind() nodeKind { return kindMap }
func (n *mapNode[K, V]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "map",
			Input:    in,
		})
		return nil, false
	}

	rawMap, ok := in.(map[string]any)
	if !ok {
		// Could also be map[K]V or map[any]any
		if direct, ok := in.(map[K]V); ok {
			return direct, true
		}
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "map",
			Input:    in,
		})
		return nil, false
	}

	count := len(rawMap)
	if n.schema.size != nil && count != *n.schema.size {
		ctx.addIssue(Issue{
			Code:      TooSmall,
			Origin:    "map",
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
			Origin:    "map",
			Minimum:   *n.schema.min,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}
	if n.schema.max != nil && count > *n.schema.max {
		ctx.addIssue(Issue{
			Code:      TooBig,
			Origin:    "map",
			Maximum:   *n.schema.max,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}

	res := make(map[K]V, count)
	allOk := true

	for kStr, rawVal := range rawMap {
		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}

		// Validate key
		keyOut, keyOk := n.schema.keySchema.node().parse(ctx, kStr)
		if !keyOk {
			ctx.addIssue(Issue{
				Code:   InvalidKey,
				Origin: "map",
				Input:  kStr,
			})
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		typedKey, ok := keyOut.(K)
		if !ok {
			allOk = false
			continue
		}

		// Validate value
		ctx.pushPath(Property(kStr))
		valOut, valOk := n.schema.valSchema.node().parse(ctx, rawVal)
		ctx.popPath()
		if !valOk {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		if typedVal, ok := valOut.(V); ok {
			res[typedKey] = typedVal
		}
	}

	if !allOk {
		return nil, false
	}
	return res, true
}

// @spec REQ-COLL-005
// Map creates a schema validating key-value map entries.
func Map[K comparable, V any](key Schema[K], val Schema[V]) *MapSchema[K, V] {
	s := &MapSchema[K, V]{
		keySchema: key,
		valSchema: val,
	}
	s.BaseSchema = NewBaseSchema[map[K]V](&mapNode[K, V]{schema: s})
	return s
}

func (s *MapSchema[K, V]) clone() *MapSchema[K, V] {
	c := &MapSchema[K, V]{
		keySchema: s.keySchema,
		valSchema: s.valSchema,
		min:       s.min,
		max:       s.max,
		size:      s.size,
	}
	c.BaseSchema = NewBaseSchema[map[K]V](&mapNode[K, V]{schema: c})
	c.SetMeta(s.GetMeta())
	return c
}

// @spec REQ-COLL-005
// Min enforces a minimum number of entries.
func (s *MapSchema[K, V]) Min(n int) *MapSchema[K, V] {
	c := s.clone()
	c.min = &n
	return c
}

// @spec REQ-COLL-005
// Max enforces a maximum number of entries.
func (s *MapSchema[K, V]) Max(n int) *MapSchema[K, V] {
	c := s.clone()
	c.max = &n
	return c
}

// @spec REQ-COLL-005
// Size enforces an exact number of entries.
func (s *MapSchema[K, V]) Size(n int) *MapSchema[K, V] {
	c := s.clone()
	c.size = &n
	return c
}

// @spec REQ-COLL-005
// NonEmpty enforces that the map contains at least 1 entry.
func (s *MapSchema[K, V]) NonEmpty() *MapSchema[K, V] {
	return s.Min(1)
}

// @spec REQ-COLL-006
func (s *MapSchema[K, V]) Optional() OptionalSchema[map[K]V] {
	return NewOptionalSchema[map[K]V](s)
}
func (s *MapSchema[K, V]) Nullable() NullableSchema[map[K]V] {
	return NewNullableSchema[map[K]V](s)
}
func (s *MapSchema[K, V]) Nullish() NullishSchema[map[K]V] {
	return NewNullishSchema[map[K]V](s)
}
func (s *MapSchema[K, V]) Default(v map[K]V) DefaultSchema[map[K]V] {
	return NewDefaultSchema[map[K]V](s, v)
}
func (s *MapSchema[K, V]) Array() *ArraySchema[map[K]V] {
	return NewArraySchema[map[K]V](s)
}
func (s *MapSchema[K, V]) Meta(fn func(m *MetaBuilder)) *MapSchema[K, V] {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}
