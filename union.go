package z

import (
	"fmt"
	"slices"
)

type unionMode int

const (
	unionModeFirst unionMode = iota
	unionModeXOR
	unionModeDiscriminated
)

// @spec REQ-COMB-001
// @spec REQ-COMB-002
// @spec REQ-COMB-003
// UnionSchema validates input matching one or more candidate schemas, returning any.
type UnionSchema struct {
	BaseSchema[any]
	options        []AnySchema
	mode           unionMode
	discriminator  string
	discrimMap     map[string]ObjectNode
	discrimOptions []any
}

type unionNode struct {
	schema *UnionSchema
}

func (n *unionNode) kind() nodeKind {
	switch n.schema.mode {
	case unionModeXOR:
		return kindUnion
	case unionModeDiscriminated:
		return kindDiscriminatedUnion
	default:
		return kindUnion
	}
}

func (n *unionNode) Options() []AnySchema {
	return n.schema.options
}

func (n *unionNode) Discriminator() string {
	return n.schema.discriminator
}

func (n *unionNode) parse(ctx *parseCtx, in any) (any, bool) {
	if err := ctx.checkCancel(); err != nil {
		return nil, false
	}

	switch n.schema.mode {
	case unionModeDiscriminated:
		rawMap, ok := in.(map[string]any)
		if !ok {
			ctx.addIssue(Issue{
				Code:     InvalidType,
				Expected: "object",
				Input:    in,
			})
			return nil, false
		}

		discVal, present := rawMap[n.schema.discriminator]
		if !present {
			ctx.pushPath(Property(n.schema.discriminator))
			ctx.addIssue(Issue{
				Code:          InvalidUnion,
				Discriminator: n.schema.discriminator,
				Options:       n.schema.discrimOptions,
				Note:          "No matching discriminator",
			})
			ctx.popPath()
			return nil, false
		}

		discStr := fmt.Sprintf("%v", discVal)
		optNode, found := n.schema.discrimMap[discStr]
		if !found {
			ctx.pushPath(Property(n.schema.discriminator))
			ctx.addIssue(Issue{
				Code:          InvalidUnion,
				Discriminator: n.schema.discriminator,
				Options:       n.schema.discrimOptions,
				Note:          "No matching discriminator",
				Input:         discVal,
			})
			ctx.popPath()
			return nil, false
		}

		return optNode.node().parse(ctx, in)

	case unionModeXOR:
		var matchedIndices []int
		var lastOut any
		var allIssues [][]Issue

		for i, opt := range n.schema.options {
			subCtx := acquireParseCtx(ctx.ctx)
			out, ok := opt.node().parse(subCtx, in)
			if ok && len(subCtx.issues) == 0 {
				matchedIndices = append(matchedIndices, i)
				lastOut = out
			} else {
				copied := slices.Clone(subCtx.issues)
				allIssues = append(allIssues, copied)
			}
			releaseParseCtx(subCtx)
		}

		if len(matchedIndices) == 1 {
			return lastOut, true
		}

		if len(matchedIndices) == 0 {
			ctx.addIssue(Issue{
				Code:      InvalidUnion,
				Errors:    allIssues,
				Inclusive: true,
				Input:     in,
			})
			return nil, false
		}

		// Multiple matches in XOR
		ctx.addIssue(Issue{
			Code:      InvalidUnion,
			Inclusive: false,
			Matches:   matchedIndices,
			Message:   "Invalid input: more than one option matched",
			Input:     in,
		})
		return nil, false

	default: // unionModeFirst
		var allIssues [][]Issue
		for _, opt := range n.schema.options {
			subCtx := acquireParseCtx(ctx.ctx)
			out, ok := opt.node().parse(subCtx, in)
			if ok && len(subCtx.issues) == 0 {
				releaseParseCtx(subCtx)
				return out, true
			}
			copied := slices.Clone(subCtx.issues)
			allIssues = append(allIssues, copied)
			releaseParseCtx(subCtx)
		}

		ctx.addIssue(Issue{
			Code:      InvalidUnion,
			Errors:    allIssues,
			Inclusive: true,
			Input:     in,
		})
		return nil, false
	}
}

// @spec REQ-COMB-001
// Union creates a union schema where the first matching candidate wins.
func Union(opts ...AnySchema) *UnionSchema {
	cloned := slices.Clone(opts)
	s := &UnionSchema{
		options: cloned,
		mode:    unionModeFirst,
	}
	s.BaseSchema = NewBaseSchema[any](&unionNode{schema: s})
	return s
}

// @spec REQ-COMB-003
// XOR creates an exclusive union schema where exactly one candidate must succeed.
func XOR(opts ...AnySchema) *UnionSchema {
	cloned := slices.Clone(opts)
	s := &UnionSchema{
		options: cloned,
		mode:    unionModeXOR,
	}
	s.BaseSchema = NewBaseSchema[any](&unionNode{schema: s})
	return s
}

// @spec REQ-COMB-002
// DiscriminatedUnion validates against objects dispatched by a discriminator field value.
func DiscriminatedUnion(discriminator string, opts ...ObjectNode) *UnionSchema {
	discMap := make(map[string]ObjectNode, len(opts))
	var discrimOptions []any

	for _, opt := range opts {
		fieldSch := opt.FieldSchema(discriminator)
		if fieldSch == nil {
			panic(fmt.Sprintf("z.DiscriminatedUnion: option lacks discriminator field %q", discriminator))
		}

		// Extract literal / enum value
		var valStr string
		if lit, ok := fieldSch.(interface{ Values() []string }); ok {
			vals := lit.Values()
			if len(vals) > 0 {
				valStr = vals[0]
			}
		} else if lit, ok := fieldSch.(interface{ Values() []any }); ok {
			vals := lit.Values()
			if len(vals) > 0 {
				valStr = fmt.Sprintf("%v", vals[0])
			}
		}

		if valStr == "" {
			panic(fmt.Sprintf("z.DiscriminatedUnion: discriminator field %q must be a Literal or Enum", discriminator))
		}

		if _, exists := discMap[valStr]; exists {
			panic(fmt.Sprintf("z.DiscriminatedUnion: duplicate discriminator value %q", valStr))
		}

		discMap[valStr] = opt
		discrimOptions = append(discrimOptions, valStr)
	}

	var anyOpts []AnySchema
	for _, o := range opts {
		anyOpts = append(anyOpts, o)
	}

	s := &UnionSchema{
		options:        anyOpts,
		mode:           unionModeDiscriminated,
		discriminator:  discriminator,
		discrimMap:     discMap,
		discrimOptions: discrimOptions,
	}
	s.BaseSchema = NewBaseSchema[any](&unionNode{schema: s})
	return s
}

// @spec REQ-COMB-008
func (s *UnionSchema) Optional() OptionalSchema[any]    { return NewOptionalSchema[any](s) }
func (s *UnionSchema) Nullable() NullableSchema[any]    { return NewNullableSchema[any](s) }
func (s *UnionSchema) Nullish() NullishSchema[any]      { return NewNullishSchema[any](s) }
func (s *UnionSchema) Default(v any) DefaultSchema[any] { return NewDefaultSchema[any](s, v) }
func (s *UnionSchema) Array() *ArraySchema[any]         { return NewArraySchema[any](s) }
func (s *UnionSchema) Meta(fn func(m *MetaBuilder)) *UnionSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return &c
}
