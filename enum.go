package z

import (
	"slices"
)

// @spec REQ-COLL-001
// EnumSchema validates that input matches one of the specified enum values.
type EnumSchema[T ~string] struct {
	BaseSchema[T]
	values []T
}

type enumNode[T ~string] struct {
	schema *EnumSchema[T]
}

func (n *enumNode[T]) kind() nodeKind { return kindEnum }
func (n *enumNode[T]) Values() []string {
	strs := make([]string, len(n.schema.values))
	for i, v := range n.schema.values {
		strs[i] = string(v)
	}
	return strs
}
func (n *enumNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	str, ok := in.(string)
	if !ok {
		// Also accept T directly if already typed
		if typed, ok := in.(T); ok {
			str = string(typed)
		} else {
			ctx.addIssue(Issue{
				Code:     InvalidType,
				Expected: "string",
				Input:    in,
			})
			var zero T
			return zero, false
		}
	}

	val := T(str)
	for _, expected := range n.schema.values {
		if val == expected {
			return val, true
		}
	}

	var anyVals []any
	for _, v := range n.schema.values {
		anyVals = append(anyVals, string(v))
	}

	ctx.addIssue(Issue{
		Code:   InvalidValue,
		Values: anyVals,
		Input:  in,
	})
	var zero T
	return zero, false
}

// @spec REQ-COLL-001
// Enum creates a schema validating against a fixed set of string enum values.
func Enum[T ~string](v ...T) *EnumSchema[T] {
	cloned := slices.Clone(v)
	s := &EnumSchema[T]{values: cloned}
	s.BaseSchema = NewBaseSchema[T](&enumNode[T]{schema: s})
	return s
}

// @spec REQ-COLL-001
// Values returns a slice of all valid enum values.
func (s *EnumSchema[T]) Values() []T {
	return slices.Clone(s.values)
}

// @spec REQ-COLL-001
// Extract returns a new EnumSchema keeping only the specified values.
func (s *EnumSchema[T]) Extract(v ...T) *EnumSchema[T] {
	var kept []T
	for _, val := range s.values {
		for _, target := range v {
			if val == target {
				kept = append(kept, val)
				break
			}
		}
	}
	res := Enum(kept...)
	res.SetMeta(s.GetMeta())
	return res
}

// @spec REQ-COLL-001
// Exclude returns a new EnumSchema removing the specified values.
func (s *EnumSchema[T]) Exclude(v ...T) *EnumSchema[T] {
	var kept []T
	for _, val := range s.values {
		excluded := false
		for _, target := range v {
			if val == target {
				excluded = true
				break
			}
		}
		if !excluded {
			kept = append(kept, val)
		}
	}
	res := Enum(kept...)
	res.SetMeta(s.GetMeta())
	return res
}

// @spec REQ-COLL-006
func (s *EnumSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *EnumSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *EnumSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *EnumSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *EnumSchema[T]) Array() *ArraySchema[T] { return NewArraySchema[T](s) }
func (s *EnumSchema[T]) Meta(fn func(m *MetaBuilder)) *EnumSchema[T] {
	c := Enum(s.values...)
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *EnumSchema[T]) Error(msg string) *EnumSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *EnumSchema[T]) ErrorFunc(fn func(issue *Issue) string) *EnumSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
