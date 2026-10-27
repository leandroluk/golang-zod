package z

import (
	"fmt"
)

// @spec REQ-PIPE-004
// TransformSchema applies a mapping function converting T to U, converting errors into Custom issues.
type TransformSchema[T, U any] struct {
	BaseSchema[U]
	inner     Schema[T]
	transform func(v T) (U, error)
}

type transformNode[T, U any] struct {
	schema *TransformSchema[T, U]
}

func (n *transformNode[T, U]) kind() nodeKind { return kindTransform }

func (n *transformNode[T, U]) parse(ctx *parseCtx, in any) (any, bool) {
	out, ok := n.schema.inner.node().parse(ctx, in)
	if !ok {
		return nil, false
	}

	typed, ok := out.(T)
	if !ok {
		ctx.addIssue(Issue{
			Code:    Custom,
			Message: fmt.Sprintf("transform input type mismatch: expected %T, got %T", *new(T), out),
		})
		return nil, false
	}

	res, err := n.schema.transform(typed)
	if err != nil {
		ctx.addIssue(Issue{
			Code:    Custom,
			Message: err.Error(),
		})
		var zero U
		return zero, false
	}

	return res, true
}

// Transform creates a transformed schema applying fn to successful inputs of s.
func Transform[T, U any](s Schema[T], fn func(v T) (U, error)) *TransformSchema[T, U] {
	ts := &TransformSchema[T, U]{
		inner:     s,
		transform: fn,
	}
	ts.BaseSchema = NewBaseSchema[U](&transformNode[T, U]{schema: ts})
	ts.SetMeta(s.GetMeta())
	return ts
}

func (s *TransformSchema[T, U]) Input() Schema[T] {
	return s.inner
}

// @spec REQ-PIPE-004
// PipeSchema chains two schemas, passing the parsed output of left as input to right.
type PipeSchema[T, U any] struct {
	BaseSchema[U]
	left  Schema[T]
	right Schema[U]
}

type pipeNode[T, U any] struct {
	schema *PipeSchema[T, U]
}

func (n *pipeNode[T, U]) kind() nodeKind { return kindPipe }

func (n *pipeNode[T, U]) parse(ctx *parseCtx, in any) (any, bool) {
	leftOut, ok := n.schema.left.node().parse(ctx, in)
	if !ok {
		return nil, false
	}
	return n.schema.right.node().parse(ctx, leftOut)
}

// Pipe chains left and right schemas into a pipeline.
func Pipe[T, U any](left Schema[T], right Schema[U]) *PipeSchema[T, U] {
	ps := &PipeSchema[T, U]{
		left:  left,
		right: right,
	}
	ps.BaseSchema = NewBaseSchema[U](&pipeNode[T, U]{schema: ps})
	ps.SetMeta(right.GetMeta())
	return ps
}

func (s *PipeSchema[T, U]) Input() Schema[T] {
	return s.left
}

func (s *PipeSchema[T, U]) Output() Schema[U] {
	return s.right
}

// @spec REQ-PIPE-004
// PreprocessSchema alters input raw data before passing to inner schema.
type PreprocessSchema[T any] struct {
	BaseSchema[T]
	inner      Schema[T]
	preprocess func(raw any) any
}

type preprocessNode[T any] struct {
	schema *PreprocessSchema[T]
}

func (n *preprocessNode[T]) kind() nodeKind { return kindCustom }

func (n *preprocessNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	transformed := n.schema.preprocess(in)
	return n.schema.inner.node().parse(ctx, transformed)
}

// Preprocess modifies raw input before passing it to the inner schema.
func Preprocess[T any](fn func(raw any) any, s Schema[T]) *PreprocessSchema[T] {
	ps := &PreprocessSchema[T]{
		inner:      s,
		preprocess: fn,
	}
	ps.BaseSchema = NewBaseSchema[T](&preprocessNode[T]{schema: ps})
	ps.SetMeta(s.GetMeta())
	return ps
}
