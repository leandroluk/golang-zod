package z

import (
	"math/big"
	"time"
)

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *ArraySchema[T]) Prefault(v []T) PrefaultSchema[[]T] { return NewPrefaultSchema[[]T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *ArraySchema[T]) PrefaultFunc(fn func() []T) PrefaultSchema[[]T] {
	return NewPrefaultFuncSchema[[]T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *ArraySchema[T]) Catch(fallback []T) CatchSchema[[]T] {
	return NewCatchSchema[[]T](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *ArraySchema[T]) CatchFunc(fn func(issues []Issue) []T) CatchSchema[[]T] {
	return NewCatchFuncSchema[[]T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *ArraySchema[T]) Readonly() ReadonlySchema[[]T] { return NewReadonlySchema[[]T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *EnumSchema[T]) Prefault(v T) PrefaultSchema[T] { return NewPrefaultSchema[T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *EnumSchema[T]) PrefaultFunc(fn func() T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *EnumSchema[T]) Catch(fallback T) CatchSchema[T] { return NewCatchSchema[T](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *EnumSchema[T]) CatchFunc(fn func(issues []Issue) T) CatchSchema[T] {
	return NewCatchFuncSchema[T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *EnumSchema[T]) Readonly() ReadonlySchema[T] { return NewReadonlySchema[T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *IntersectionSchema) Prefault(v any) PrefaultSchema[any] { return NewPrefaultSchema[any](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *IntersectionSchema) PrefaultFunc(fn func() any) PrefaultSchema[any] {
	return NewPrefaultFuncSchema[any](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *IntersectionSchema) Catch(fallback any) CatchSchema[any] {
	return NewCatchSchema[any](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *IntersectionSchema) CatchFunc(fn func(issues []Issue) any) CatchSchema[any] {
	return NewCatchFuncSchema[any](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *IntersectionSchema) Readonly() ReadonlySchema[any] { return NewReadonlySchema[any](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *MapSchema[K, V]) Prefault(v map[K]V) PrefaultSchema[map[K]V] {
	return NewPrefaultSchema[map[K]V](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *MapSchema[K, V]) PrefaultFunc(fn func() map[K]V) PrefaultSchema[map[K]V] {
	return NewPrefaultFuncSchema[map[K]V](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *MapSchema[K, V]) Catch(fallback map[K]V) CatchSchema[map[K]V] {
	return NewCatchSchema[map[K]V](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *MapSchema[K, V]) CatchFunc(fn func(issues []Issue) map[K]V) CatchSchema[map[K]V] {
	return NewCatchFuncSchema[map[K]V](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *MapSchema[K, V]) Readonly() ReadonlySchema[map[K]V] { return NewReadonlySchema[map[K]V](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *NumberSchema[T]) Prefault(v T) PrefaultSchema[T] { return NewPrefaultSchema[T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *NumberSchema[T]) PrefaultFunc(fn func() T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *NumberSchema[T]) Catch(fallback T) CatchSchema[T] { return NewCatchSchema[T](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *NumberSchema[T]) CatchFunc(fn func(issues []Issue) T) CatchSchema[T] {
	return NewCatchFuncSchema[T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *NumberSchema[T]) Readonly() ReadonlySchema[T] { return NewReadonlySchema[T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *BigIntSchema) Prefault(v *big.Int) PrefaultSchema[*big.Int] {
	return NewPrefaultSchema[*big.Int](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *BigIntSchema) PrefaultFunc(fn func() *big.Int) PrefaultSchema[*big.Int] {
	return NewPrefaultFuncSchema[*big.Int](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *BigIntSchema) Catch(fallback *big.Int) CatchSchema[*big.Int] {
	return NewCatchSchema[*big.Int](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *BigIntSchema) CatchFunc(fn func(issues []Issue) *big.Int) CatchSchema[*big.Int] {
	return NewCatchFuncSchema[*big.Int](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *BigIntSchema) Readonly() ReadonlySchema[*big.Int] { return NewReadonlySchema[*big.Int](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *ObjectSchema) Prefault(v Dyn) PrefaultSchema[Dyn] { return NewPrefaultSchema[Dyn](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *ObjectSchema) PrefaultFunc(fn func() Dyn) PrefaultSchema[Dyn] {
	return NewPrefaultFuncSchema[Dyn](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *ObjectSchema) Catch(fallback Dyn) CatchSchema[Dyn] { return NewCatchSchema[Dyn](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *ObjectSchema) CatchFunc(fn func(issues []Issue) Dyn) CatchSchema[Dyn] {
	return NewCatchFuncSchema[Dyn](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *ObjectSchema) Readonly() ReadonlySchema[Dyn] { return NewReadonlySchema[Dyn](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *BooleanSchema) Prefault(v bool) PrefaultSchema[bool] { return NewPrefaultSchema[bool](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *BooleanSchema) PrefaultFunc(fn func() bool) PrefaultSchema[bool] {
	return NewPrefaultFuncSchema[bool](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *BooleanSchema) Catch(fallback bool) CatchSchema[bool] {
	return NewCatchSchema[bool](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *BooleanSchema) CatchFunc(fn func(issues []Issue) bool) CatchSchema[bool] {
	return NewCatchFuncSchema[bool](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *BooleanSchema) Readonly() ReadonlySchema[bool] { return NewReadonlySchema[bool](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *NullSchema) Prefault(v any) PrefaultSchema[any] { return NewPrefaultSchema[any](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *NullSchema) PrefaultFunc(fn func() any) PrefaultSchema[any] {
	return NewPrefaultFuncSchema[any](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *NullSchema) Catch(fallback any) CatchSchema[any] { return NewCatchSchema[any](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *NullSchema) CatchFunc(fn func(issues []Issue) any) CatchSchema[any] {
	return NewCatchFuncSchema[any](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *NullSchema) Readonly() ReadonlySchema[any] { return NewReadonlySchema[any](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *AnyPrimitiveSchema) Prefault(v any) PrefaultSchema[any] { return NewPrefaultSchema[any](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *AnyPrimitiveSchema) PrefaultFunc(fn func() any) PrefaultSchema[any] {
	return NewPrefaultFuncSchema[any](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *AnyPrimitiveSchema) Catch(fallback any) CatchSchema[any] {
	return NewCatchSchema[any](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *AnyPrimitiveSchema) CatchFunc(fn func(issues []Issue) any) CatchSchema[any] {
	return NewCatchFuncSchema[any](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *AnyPrimitiveSchema) Readonly() ReadonlySchema[any] { return NewReadonlySchema[any](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *DateSchema) Prefault(v time.Time) PrefaultSchema[time.Time] {
	return NewPrefaultSchema[time.Time](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *DateSchema) PrefaultFunc(fn func() time.Time) PrefaultSchema[time.Time] {
	return NewPrefaultFuncSchema[time.Time](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *DateSchema) Catch(fallback time.Time) CatchSchema[time.Time] {
	return NewCatchSchema[time.Time](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *DateSchema) CatchFunc(fn func(issues []Issue) time.Time) CatchSchema[time.Time] {
	return NewCatchFuncSchema[time.Time](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *DateSchema) Readonly() ReadonlySchema[time.Time] { return NewReadonlySchema[time.Time](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *BytesSchema) Prefault(v []byte) PrefaultSchema[[]byte] {
	return NewPrefaultSchema[[]byte](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *BytesSchema) PrefaultFunc(fn func() []byte) PrefaultSchema[[]byte] {
	return NewPrefaultFuncSchema[[]byte](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *BytesSchema) Catch(fallback []byte) CatchSchema[[]byte] {
	return NewCatchSchema[[]byte](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *BytesSchema) CatchFunc(fn func(issues []Issue) []byte) CatchSchema[[]byte] {
	return NewCatchFuncSchema[[]byte](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *BytesSchema) Readonly() ReadonlySchema[[]byte] { return NewReadonlySchema[[]byte](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *LiteralSchema[T]) Prefault(v T) PrefaultSchema[T] { return NewPrefaultSchema[T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *LiteralSchema[T]) PrefaultFunc(fn func() T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *LiteralSchema[T]) Catch(fallback T) CatchSchema[T] { return NewCatchSchema[T](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *LiteralSchema[T]) CatchFunc(fn func(issues []Issue) T) CatchSchema[T] {
	return NewCatchFuncSchema[T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *LiteralSchema[T]) Readonly() ReadonlySchema[T] { return NewReadonlySchema[T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *RecordSchema[K, V]) Prefault(v map[K]V) PrefaultSchema[map[K]V] {
	return NewPrefaultSchema[map[K]V](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *RecordSchema[K, V]) PrefaultFunc(fn func() map[K]V) PrefaultSchema[map[K]V] {
	return NewPrefaultFuncSchema[map[K]V](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *RecordSchema[K, V]) Catch(fallback map[K]V) CatchSchema[map[K]V] {
	return NewCatchSchema[map[K]V](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *RecordSchema[K, V]) CatchFunc(fn func(issues []Issue) map[K]V) CatchSchema[map[K]V] {
	return NewCatchFuncSchema[map[K]V](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *RecordSchema[K, V]) Readonly() ReadonlySchema[map[K]V] { return NewReadonlySchema[map[K]V](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *RefinedSchema[T]) Prefault(v T) PrefaultSchema[T] { return NewPrefaultSchema[T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *RefinedSchema[T]) PrefaultFunc(fn func() T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *RefinedSchema[T]) Catch(fallback T) CatchSchema[T] { return NewCatchSchema[T](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *RefinedSchema[T]) CatchFunc(fn func(issues []Issue) T) CatchSchema[T] {
	return NewCatchFuncSchema[T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *RefinedSchema[T]) Readonly() ReadonlySchema[T] { return NewReadonlySchema[T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *SetSchema[T]) Prefault(v map[T]struct{}) PrefaultSchema[map[T]struct{}] {
	return NewPrefaultSchema[map[T]struct{}](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *SetSchema[T]) PrefaultFunc(fn func() map[T]struct{}) PrefaultSchema[map[T]struct{}] {
	return NewPrefaultFuncSchema[map[T]struct{}](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *SetSchema[T]) Catch(fallback map[T]struct{}) CatchSchema[map[T]struct{}] {
	return NewCatchSchema[map[T]struct{}](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *SetSchema[T]) CatchFunc(fn func(issues []Issue) map[T]struct{}) CatchSchema[map[T]struct{}] {
	return NewCatchFuncSchema[map[T]struct{}](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *SetSchema[T]) Readonly() ReadonlySchema[map[T]struct{}] {
	return NewReadonlySchema[map[T]struct{}](s)
}

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *StringSchema) Prefault(v string) PrefaultSchema[string] {
	return NewPrefaultSchema[string](s, v)
}

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *StringSchema) PrefaultFunc(fn func() string) PrefaultSchema[string] {
	return NewPrefaultFuncSchema[string](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *StringSchema) Catch(fallback string) CatchSchema[string] {
	return NewCatchSchema[string](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *StringSchema) CatchFunc(fn func(issues []Issue) string) CatchSchema[string] {
	return NewCatchFuncSchema[string](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *StringSchema) Readonly() ReadonlySchema[string] { return NewReadonlySchema[string](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *StructSchema[T]) Prefault(v T) PrefaultSchema[T] { return NewPrefaultSchema[T](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *StructSchema[T]) PrefaultFunc(fn func() T) PrefaultSchema[T] {
	return NewPrefaultFuncSchema[T](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *StructSchema[T]) Catch(fallback T) CatchSchema[T] { return NewCatchSchema[T](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *StructSchema[T]) CatchFunc(fn func(issues []Issue) T) CatchSchema[T] {
	return NewCatchFuncSchema[T](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *StructSchema[T]) Readonly() ReadonlySchema[T] { return NewReadonlySchema[T](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *TupleSchema) Prefault(v []any) PrefaultSchema[[]any] { return NewPrefaultSchema[[]any](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *TupleSchema) PrefaultFunc(fn func() []any) PrefaultSchema[[]any] {
	return NewPrefaultFuncSchema[[]any](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *TupleSchema) Catch(fallback []any) CatchSchema[[]any] {
	return NewCatchSchema[[]any](s, fallback)
}

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *TupleSchema) CatchFunc(fn func(issues []Issue) []any) CatchSchema[[]any] {
	return NewCatchFuncSchema[[]any](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *TupleSchema) Readonly() ReadonlySchema[[]any] { return NewReadonlySchema[[]any](s) }

// Prefault substitutes v for absent input before it runs through the schema (Zod's `.prefault(v)`).
func (s *UnionSchema) Prefault(v any) PrefaultSchema[any] { return NewPrefaultSchema[any](s, v) }

// PrefaultFunc is Prefault with a supplier evaluated on every absent input.
func (s *UnionSchema) PrefaultFunc(fn func() any) PrefaultSchema[any] {
	return NewPrefaultFuncSchema[any](s, fn)
}

// Catch returns fallback whenever parsing fails (Zod's `.catch(v)`).
func (s *UnionSchema) Catch(fallback any) CatchSchema[any] { return NewCatchSchema[any](s, fallback) }

// CatchFunc is Catch with a supplier that receives the issues that caused the failure.
func (s *UnionSchema) CatchFunc(fn func(issues []Issue) any) CatchSchema[any] {
	return NewCatchFuncSchema[any](s, fn)
}

// Readonly marks the schema read-only (Zod's `.readonly()`).
func (s *UnionSchema) Readonly() ReadonlySchema[any] { return NewReadonlySchema[any](s) }
