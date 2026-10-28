package z

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"fmt"
	"io"
)

// @spec REQ-CORE-002
// Schema is the sealed generic contract implemented by all golang-zod schemas.
type Schema[T any] interface {
	Parse(in any, ctx ...context.Context) (T, error)
	ParseJSON(data []byte, ctx ...context.Context) (T, error)
	Decode(r io.Reader, ctx ...context.Context) (T, error)
	GetMeta() Meta
	node() node
}

// extractContext extracts the first non-nil context, or defaults to context.Background().
func extractContext(ctxs []context.Context) context.Context {
	for _, c := range ctxs {
		if c != nil {
			return c
		}
	}
	return context.Background()
}

// @spec REQ-CORE-002
// BaseSchema provides a reusable base implementation for typed schemas.
type BaseSchema[T any] struct {
	innerNode node
	meta      Meta
}

// NewBaseSchema creates a new BaseSchema wrapping an untyped engine node.
func NewBaseSchema[T any](n node) BaseSchema[T] {
	return BaseSchema[T]{
		innerNode: n,
	}
}

func (s BaseSchema[T]) node() node {
	if s.meta.errOv != nil {
		return &errorNode{inner: s.innerNode, ov: s.meta.errOv}
	}
	return s.innerNode
}

// @spec REQ-CORE-002
// GetMeta returns the schema's metadata.
func (s BaseSchema[T]) GetMeta() Meta {
	return s.meta
}

// SetMeta updates the schema metadata, validating non-empty ID uniqueness.
func (s *BaseSchema[T]) SetMeta(m Meta) {
	if m.ID != "" {
		if m.lineage == nil {
			m.lineage = &metaLineage{}
		}
		RegisterSchemaID(m.ID, m.lineage)
	}
	s.meta = m
}

// @spec REQ-CORE-002
// @spec REQ-CORE-012
// @spec REQ-CORE-017
func (s BaseSchema[T]) Parse(in any, ctx ...context.Context) (T, error) {
	c := extractContext(ctx)
	if err := c.Err(); err != nil {
		var zero T
		return zero, err
	}

	pc := acquireParseCtx(c)
	defer releaseParseCtx(pc)

	out, ok := s.node().parse(pc, in)
	if !ok || len(pc.issues) > 0 {
		var zero T
		issues := make([]Issue, len(pc.issues))
		copy(issues, pc.issues)
		return zero, NewZodError(issues)
	}

	if out == nil {
		var zero T
		return zero, nil
	}

	if typed, ok := out.(T); ok {
		return typed, nil
	}

	var zero T
	return zero, fmt.Errorf("golang-zod internal: failed to convert %T to output type %T", out, zero)
}

// @spec REQ-CORE-002
// @spec REQ-CORE-003
// @spec REQ-CORE-017
func (s BaseSchema[T]) ParseJSON(data []byte, ctx ...context.Context) (T, error) {
	c := extractContext(ctx)
	if err := c.Err(); err != nil {
		var zero T
		return zero, err
	}

	dec := jsontext.NewDecoder(bytes.NewReader(data))
	raw, err := decodeJSON(dec)
	if err != nil {
		var zero T
		return zero, err
	}

	return s.Parse(raw, c)
}

// @spec REQ-CORE-002
// @spec REQ-CORE-003
// @spec REQ-CORE-017
func (s BaseSchema[T]) Decode(r io.Reader, ctx ...context.Context) (T, error) {
	c := extractContext(ctx)
	if err := c.Err(); err != nil {
		var zero T
		return zero, err
	}

	dec := jsontext.NewDecoder(r)
	raw, err := decodeJSON(dec)
	if err != nil {
		var zero T
		return zero, err
	}

	return s.Parse(raw, c)
}
