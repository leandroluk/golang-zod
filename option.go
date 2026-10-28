package z

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// @spec REQ-CORE-004
// Option represents the optional presence of a value, distinguishing between
// an absent field (None) and a present field (Some).
// Used as the output of .Optional().
type Option[T any] struct {
	val    T
	isSome bool
}

// @spec REQ-CORE-004
// Some creates an Option containing a value.
func Some[T any](v T) Option[T] {
	return Option[T]{val: v, isSome: true}
}

// @spec REQ-CORE-004
// None creates an empty Option representing an absent value.
func None[T any]() Option[T] {
	return Option[T]{isSome: false}
}

// @spec REQ-CORE-004
// IsSome returns true if the option contains a value.
func (o Option[T]) IsSome() bool {
	return o.isSome
}

// @spec REQ-CORE-004
// Value returns the inner value and true if present, or zero and false if absent.
func (o Option[T]) Value() (T, bool) {
	return o.val, o.isSome
}

// @spec REQ-CORE-004
// OrElse returns the inner value if present, or fallback if absent.
func (o Option[T]) OrElse(fallback T) T {
	if o.isSome {
		return o.val
	}
	return fallback
}

// @spec REQ-CORE-004
// IsZero reports whether the option is empty, allowing encoding/json/v2 `omitzero`
// to omit absent fields when marshaling structs.
func (o Option[T]) IsZero() bool {
	return !o.isSome
}

// @spec REQ-CORE-004
// String formats Option for debugging.
func (o Option[T]) String() string {
	if o.isSome {
		return fmt.Sprintf("Some(%v)", o.val)
	}
	return "None"
}

// @spec REQ-CORE-004
// MarshalJSON encodes the Option to JSON bytes. If present, it marshals the inner
// value. If absent, it marshals as null. Struct fields should typically use the
// `omitzero` tag option to omit absent Option values.
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.isSome {
		return []byte("null"), nil
	}
	return json.Marshal(o.val)
}

// @spec REQ-CORE-004
// UnmarshalJSON decodes JSON bytes into an Option. If data is "null" and T is Nullable,
// it is unmarshaled into T; if T is non-nullable, json.Unmarshal fails as expected.
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	o.val = val
	o.isSome = true
	return nil
}

// @spec REQ-CORE-004
// MarshalJSONTo implements json.MarshalerTo for encoding/json/v2 streaming.
func (o Option[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if !o.isSome {
		return enc.WriteToken(jsontext.Null)
	}
	return json.MarshalEncode(enc, o.val)
}

// @spec REQ-CORE-004
// UnmarshalJSONFrom implements json.UnmarshalerFrom for encoding/json/v2 streaming.
func (o *Option[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var val T
	if err := json.UnmarshalDecode(dec, &val); err != nil {
		return err
	}
	o.val = val
	o.isSome = true
	return nil
}

// @spec REQ-CORE-004
// Nullable represents an explicit JSON null or an actual value.
// Used as the output of .Nullable().
type Nullable[T any] struct {
	val    T
	isNull bool
}

// @spec REQ-CORE-004
// NullableOf creates a Nullable containing a non-null value.
func NullableOf[T any](v T) Nullable[T] {
	return Nullable[T]{val: v, isNull: false}
}

// @spec REQ-CORE-004
// NullableNull creates a Nullable representing explicit null.
func NullableNull[T any]() Nullable[T] {
	return Nullable[T]{isNull: true}
}

// @spec REQ-CORE-004
// IsNull returns true if this represents explicit null.
func (n Nullable[T]) IsNull() bool {
	return n.isNull
}

// @spec REQ-CORE-004
// Value returns the inner value and true if not null, or zero and false if null.
func (n Nullable[T]) Value() (T, bool) {
	return n.val, !n.isNull
}

// @spec REQ-CORE-004
// OrElse returns the inner value if not null, or fallback if null.
func (n Nullable[T]) OrElse(fallback T) T {
	if !n.isNull {
		return n.val
	}
	return fallback
}

// @spec REQ-CORE-004
// String formats Nullable for debugging.
func (n Nullable[T]) String() string {
	if n.isNull {
		return "Null"
	}
	return fmt.Sprintf("Nullable(%v)", n.val)
}

// @spec REQ-CORE-004
// MarshalJSON encodes Nullable to JSON bytes: "null" if null, or marshals the value.
func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.val)
}

// @spec REQ-CORE-004
// UnmarshalJSON decodes JSON bytes into Nullable.
func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		var zero T
		n.val = zero
		n.isNull = true
		return nil
	}
	var val T
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	n.val = val
	n.isNull = false
	return nil
}

// @spec REQ-CORE-004
// MarshalJSONTo implements json.MarshalerTo for encoding/json/v2 streaming.
func (n Nullable[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if n.isNull {
		return enc.WriteToken(jsontext.Null)
	}
	return json.MarshalEncode(enc, n.val)
}

// @spec REQ-CORE-004
// UnmarshalJSONFrom implements json.UnmarshalerFrom for encoding/json/v2 streaming.
func (n *Nullable[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		// PeekKind reported 'n', so the token read here is always the null literal.
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		var zero T
		n.val = zero
		n.isNull = true
		return nil
	}
	var val T
	if err := json.UnmarshalDecode(dec, &val); err != nil {
		return err
	}
	n.val = val
	n.isNull = false
	return nil
}
