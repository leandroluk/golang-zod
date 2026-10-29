package z

import (
	"math/big"
	"time"
)

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
// Mirrors Zod's `.default(() => value)`, e.g. `z.UUID().DefaultFunc(uuid.NewString)`.
func (s *ArraySchema[T]) DefaultFunc(fn func() []T) DefaultSchema[[]T] {
	return NewDefaultFuncSchema[[]T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *EnumSchema[T]) DefaultFunc(fn func() T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *IntersectionSchema) DefaultFunc(fn func() any) DefaultSchema[any] {
	return NewDefaultFuncSchema[any](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *MapSchema[K, V]) DefaultFunc(fn func() map[K]V) DefaultSchema[map[K]V] {
	return NewDefaultFuncSchema[map[K]V](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *NumberSchema[T]) DefaultFunc(fn func() T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *BigIntSchema) DefaultFunc(fn func() *big.Int) DefaultSchema[*big.Int] {
	return NewDefaultFuncSchema[*big.Int](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *ObjectSchema) DefaultFunc(fn func() Dyn) DefaultSchema[Dyn] {
	return NewDefaultFuncSchema[Dyn](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *BooleanSchema) DefaultFunc(fn func() bool) DefaultSchema[bool] {
	return NewDefaultFuncSchema[bool](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *NullSchema) DefaultFunc(fn func() any) DefaultSchema[any] {
	return NewDefaultFuncSchema[any](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *AnyPrimitiveSchema) DefaultFunc(fn func() any) DefaultSchema[any] {
	return NewDefaultFuncSchema[any](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *DateSchema) DefaultFunc(fn func() time.Time) DefaultSchema[time.Time] {
	return NewDefaultFuncSchema[time.Time](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *BytesSchema) DefaultFunc(fn func() []byte) DefaultSchema[[]byte] {
	return NewDefaultFuncSchema[[]byte](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *LiteralSchema[T]) DefaultFunc(fn func() T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *RecordSchema[K, V]) DefaultFunc(fn func() map[K]V) DefaultSchema[map[K]V] {
	return NewDefaultFuncSchema[map[K]V](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *RefinedSchema[T]) DefaultFunc(fn func() T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *SetSchema[T]) DefaultFunc(fn func() map[T]struct{}) DefaultSchema[map[T]struct{}] {
	return NewDefaultFuncSchema[map[T]struct{}](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *StringSchema) DefaultFunc(fn func() string) DefaultSchema[string] {
	return NewDefaultFuncSchema[string](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *StructSchema[T]) DefaultFunc(fn func() T) DefaultSchema[T] {
	return NewDefaultFuncSchema[T](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *TupleSchema) DefaultFunc(fn func() []any) DefaultSchema[[]any] {
	return NewDefaultFuncSchema[[]any](s, fn)
}

// DefaultFunc wraps the schema with a fallback supplier evaluated on every absent input.
func (s *UnionSchema) DefaultFunc(fn func() any) DefaultSchema[any] {
	return NewDefaultFuncSchema[any](s, fn)
}
