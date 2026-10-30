package z

import (
	"errors"
	"fmt"
	"slices"
)

// @spec REQ-COMB-004
// ErrUnmergeableIntersection is returned when two valid intersection results conflict.
var ErrUnmergeableIntersection = errors.New("unmergeable intersection")

// @spec REQ-COMB-004
// IntersectionSchema parses input with two schemas and merges their results.
type IntersectionSchema struct {
	BaseSchema[any]
	left  AnySchema
	right AnySchema
}

type intersectionNode struct {
	schema *IntersectionSchema
}

func (n *intersectionNode) kind() nodeKind                   { return kindIntersection }
func (n *intersectionNode) Operands() (AnySchema, AnySchema) { return n.schema.left, n.schema.right }
func (n *intersectionNode) parse(ctx *parseCtx, in any) (any, bool) {
	if err := ctx.checkCancel(); err != nil {
		return nil, false
	}

	leftOut, leftOk := n.schema.left.node().parse(ctx, in)
	if !leftOk {
		return nil, false
	}

	rightOut, rightOk := n.schema.right.node().parse(ctx, in)
	if !rightOk {
		return nil, false
	}

	merged, err := mergeValues(leftOut, rightOut, "")
	if err != nil {
		// ADR-021: conflict returns error, not *ZodError
		// Record error onto parseCtx or fail
		ctx.addIssue(Issue{
			Code:    Custom,
			Message: fmt.Sprintf("%v: %v", ErrUnmergeableIntersection, err),
		})
		return nil, false
	}

	return merged, true
}

func mergeValues(a, b any, path string) (any, error) {
	if a == nil && b == nil {
		return nil, nil
	}
	if a == nil || b == nil {
		return nil, fmt.Errorf("conflict at path %q: nil vs non-nil", path)
	}

	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)
	if aIsMap && bIsMap {
		res := make(map[string]any, len(aMap)+len(bMap))
		for k, v := range aMap {
			res[k] = v
		}
		for k, v := range bMap {
			if existing, exists := res[k]; exists {
				subPath := k
				if path != "" {
					subPath = path + "." + k
				}
				mergedVal, err := mergeValues(existing, v, subPath)
				if err != nil {
					return nil, err
				}
				res[k] = mergedVal
			} else {
				res[k] = v
			}
		}
		return res, nil
	}

	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)
	if aIsArr && bIsArr {
		if len(aArr) != len(bArr) {
			return nil, fmt.Errorf("conflict at path %q: array lengths differ (%d != %d)", path, len(aArr), len(bArr))
		}
		res := make([]any, len(aArr))
		for i := range aArr {
			subPath := fmt.Sprintf("%s[%d]", path, i)
			mergedElem, err := mergeValues(aArr[i], bArr[i], subPath)
			if err != nil {
				return nil, err
			}
			res[i] = mergedElem
		}
		return res, nil
	}

	if aMaps, ok := a.([]map[string]any); ok {
		bMaps, ok := b.([]map[string]any)
		if !ok {
			return nil, fmt.Errorf("conflict at path %q: conflicting values (%T != %T)", path, a, b)
		}
		if len(aMaps) != len(bMaps) {
			return nil, fmt.Errorf("conflict at path %q: array lengths differ (%d != %d)", path, len(aMaps), len(bMaps))
		}
		res := make([]map[string]any, len(aMaps))
		for i := range aMaps {
			mergedElem, err := mergeValues(aMaps[i], bMaps[i], fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			res[i] = mergedElem.(map[string]any)
		}
		return res, nil
	}

	if valuesEqual(a, b) {
		return a, nil
	}

	return nil, fmt.Errorf("conflict at path %q: conflicting scalar values (%v != %v)", path, a, b)
}

// valuesEqual compares two non-nil values without panicking on uncomparable
// dynamic types (slices, maps, funcs). Common scalar and slice types use direct
// comparison; anything else falls back to comparing type and Go-syntax rendering.
func valuesEqual(a, b any) bool {
	switch x := a.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return a == b
	case []string:
		y, ok := b.([]string)
		return ok && slices.Equal(x, y)
	case []int:
		y, ok := b.([]int)
		return ok && slices.Equal(x, y)
	case []int64:
		y, ok := b.([]int64)
		return ok && slices.Equal(x, y)
	case []float64:
		y, ok := b.([]float64)
		return ok && slices.Equal(x, y)
	case []bool:
		y, ok := b.([]bool)
		return ok && slices.Equal(x, y)
	default:
		return fmt.Sprintf("%T|%#v", a, a) == fmt.Sprintf("%T|%#v", b, b)
	}
}

// @spec REQ-COMB-004
// Intersection creates a schema requiring both left and right to succeed, merging outputs.
func Intersection(left, right AnySchema) *IntersectionSchema {
	s := &IntersectionSchema{
		left:  left,
		right: right,
	}
	s.BaseSchema = NewBaseSchema[any](&intersectionNode{schema: s})
	return s
}

// @spec REQ-COMB-008
func (s *IntersectionSchema) Optional() OptionalSchema[any] { return NewOptionalSchema[any](s) }
func (s *IntersectionSchema) Nullable() NullableSchema[any] { return NewNullableSchema[any](s) }
func (s *IntersectionSchema) Nullish() NullishSchema[any]   { return NewNullishSchema[any](s) }
func (s *IntersectionSchema) Default(v any) DefaultSchema[any] {
	return NewDefaultSchema[any](s, v)
}
func (s *IntersectionSchema) Array() *ArraySchema[any] { return NewArraySchema[any](s) }
func (s *IntersectionSchema) Meta(fn func(m *MetaBuilder)) *IntersectionSchema {
	c := *s
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	c.updateSlot(&c)
	return &c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *IntersectionSchema) Error(msg string) *IntersectionSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *IntersectionSchema) ErrorFunc(fn func(issue *Issue) string) *IntersectionSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
