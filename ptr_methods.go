package z

import (
	"math/big"
	"time"
)

// Ptr yields a *T, nil when the input is absent or null.
func (s *EnumSchema[T]) Ptr() *PtrSchema[T] { return Ptr[T](s) }

// Ptr yields a *T, nil when the input is absent or null.
func (s *NumberSchema[T]) Ptr() *PtrSchema[T] { return Ptr[T](s) }

// Ptr yields a **big.Int, nil when the input is absent or null.
func (s *BigIntSchema) Ptr() *PtrSchema[*big.Int] { return Ptr[*big.Int](s) }

// Ptr yields a *bool, nil when the input is absent or null.
func (s *BooleanSchema) Ptr() *PtrSchema[bool] { return Ptr[bool](s) }

// Ptr yields a *time.Time, nil when the input is absent or null.
func (s *DateSchema) Ptr() *PtrSchema[time.Time] { return Ptr[time.Time](s) }

// Ptr yields a *[]byte, nil when the input is absent or null.
func (s *BytesSchema) Ptr() *PtrSchema[[]byte] { return Ptr[[]byte](s) }

// Ptr yields a *T, nil when the input is absent or null.
func (s *LiteralSchema[T]) Ptr() *PtrSchema[T] { return Ptr[T](s) }

// Ptr yields a *T, nil when the input is absent or null.
func (s *RefinedSchema[T]) Ptr() *PtrSchema[T] { return Ptr[T](s) }

// Ptr yields a *string, nil when the input is absent or null.
func (s *StringSchema) Ptr() *PtrSchema[string] { return Ptr[string](s) }

// Ptr yields a *T, nil when the input is absent or null.
func (s *StructSchema[T]) Ptr() *PtrSchema[T] { return Ptr[T](s) }

// Ptr yields a *[]T, nil when the input is absent or null.
func (s *ArraySchema[T]) Ptr() *PtrSchema[[]T] { return Ptr[[]T](s) }

// Ptr yields a *map[K]V, nil when the input is absent or null.
func (s *MapSchema[K, V]) Ptr() *PtrSchema[map[K]V] { return Ptr[map[K]V](s) }

// Ptr yields a *map[K]V, nil when the input is absent or null.
func (s *RecordSchema[K, V]) Ptr() *PtrSchema[map[K]V] { return Ptr[map[K]V](s) }
