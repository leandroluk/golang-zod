package z

import (
	"slices"
)

// AnySchema is satisfied by any schema regardless of its generic type parameter.
type AnySchema interface {
	GetMeta() Meta
	node() node
}

// @spec REQ-COLL-003
// TupleSchema validates fixed-length heterogeneous sequences of items, returning []any.
type TupleSchema struct {
	BaseSchema[[]any]
	items []AnySchema
	rest  AnySchema
}

type tupleNode struct {
	schema *TupleSchema
}

func (n *tupleNode) kind() nodeKind { return kindTuple }
func (n *tupleNode) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "tuple",
			Input:    in,
		})
		return nil, false
	}

	arr, ok := in.([]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "array",
			Input:    in,
		})
		return nil, false
	}

	itemCount := len(n.schema.items)
	inputLen := len(arr)

	if n.schema.rest == nil {
		if inputLen < itemCount {
			ctx.addIssue(Issue{
				Code:      TooSmall,
				Origin:    "array",
				Minimum:   itemCount,
				Exact:     true,
				Inclusive: true,
				Input:     in,
			})
			return nil, false
		}
		if inputLen > itemCount {
			ctx.addIssue(Issue{
				Code:      TooBig,
				Origin:    "array",
				Maximum:   itemCount,
				Exact:     true,
				Inclusive: true,
				Input:     in,
			})
			return nil, false
		}
	} else {
		if inputLen < itemCount {
			ctx.addIssue(Issue{
				Code:      TooSmall,
				Origin:    "array",
				Minimum:   itemCount,
				Inclusive: true,
				Input:     in,
			})
			return nil, false
		}
	}

	res := make([]any, inputLen)
	allOk := true

	for i := 0; i < inputLen; i++ {
		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}
		ctx.pushPath(Index(i))
		var schemaNode node
		if i < itemCount {
			schemaNode = n.schema.items[i].node()
		} else {
			schemaNode = n.schema.rest.node()
		}

		out, ok := schemaNode.parse(ctx, arr[i])
		ctx.popPath()
		if !ok {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}
		res[i] = out
	}

	if !allOk {
		return nil, false
	}
	return res, true
}

// @spec REQ-COLL-003
// Tuple creates a heterogeneous fixed-length schema from positional item schemas.
func Tuple(items ...AnySchema) *TupleSchema {
	cloned := slices.Clone(items)
	s := &TupleSchema{
		items: cloned,
	}
	s.BaseSchema = NewBaseSchema[[]any](&tupleNode{schema: s})
	return s
}

func (s *TupleSchema) clone() *TupleSchema {
	c := &TupleSchema{
		items: slices.Clone(s.items),
		rest:  s.rest,
	}
	c.BaseSchema = NewBaseSchema[[]any](&tupleNode{schema: c})
	c.SetMeta(s.GetMeta())
	return c
}

// @spec REQ-COLL-003
// Rest allows additional trailing elements validated against restSchema.
func (s *TupleSchema) Rest(restSchema AnySchema) *TupleSchema {
	c := s.clone()
	c.rest = restSchema
	return c
}

// @spec REQ-COLL-006
func (s *TupleSchema) Optional() OptionalSchema[[]any] { return NewOptionalSchema[[]any](s) }
func (s *TupleSchema) Nullable() NullableSchema[[]any] { return NewNullableSchema[[]any](s) }
func (s *TupleSchema) Nullish() NullishSchema[[]any]   { return NewNullishSchema[[]any](s) }
func (s *TupleSchema) Default(v []any) DefaultSchema[[]any] {
	return NewDefaultSchema[[]any](s, v)
}
func (s *TupleSchema) Array() *ArraySchema[[]any] { return NewArraySchema[[]any](s) }
func (s *TupleSchema) Meta(fn func(m *MetaBuilder)) *TupleSchema {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *TupleSchema) Error(msg string) *TupleSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *TupleSchema) ErrorFunc(fn func(issue *Issue) string) *TupleSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
