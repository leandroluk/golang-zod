package z

import (
	"context"
)

// @spec REQ-PIPE-003
// RefineCtx provides context and issue registration during SuperRefine calls.
type RefineCtx struct {
	pc     *parseCtx
	failed bool
}

// AddIssue records a validation issue onto the refinement context.
func (c *RefineCtx) AddIssue(iss Issue) {
	c.failed = true
	c.pc.addIssue(iss)
}

// RefineConfig configures a Refine check.
type RefineConfig struct {
	Message     string
	MessageFunc func(iss *Issue) string
	Code        IssueCode
	Path        []PathSegment
	Abort       bool
}

// RefineOption configures refinement behavior.
type RefineOption func(*RefineConfig)

// RefineMessage specifies the error message for refinement failure.
func RefineMessage(msg string) RefineOption {
	return func(c *RefineConfig) {
		c.Message = msg
	}
}

// RefineMessageFunc dynamically resolves the error message on failure.
func RefineMessageFunc(fn func(iss *Issue) string) RefineOption {
	return func(c *RefineConfig) {
		c.MessageFunc = fn
	}
}

// RefineCode overrides the IssueCode (default Custom).
func RefineCode(code IssueCode) RefineOption {
	return func(c *RefineConfig) {
		c.Code = code
	}
}

// RefinePath sets the target path segments for the issue.
func RefinePath(path ...PathSegment) RefineOption {
	return func(c *RefineConfig) {
		c.Path = path
	}
}

// RefineAbort configures the refinement to halt execution immediately on failure.
func RefineAbort(abort bool) RefineOption {
	return func(c *RefineConfig) {
		c.Abort = abort
	}
}

type refinementStep[T any] func(pc *parseCtx, v T) bool

// @spec REQ-PIPE-003
// RefinedSchema wraps an inner schema with additional custom refinement passes.
type RefinedSchema[T any] struct {
	BaseSchema[T]
	inner Schema[T]
	steps []refinementStep[T]
}

type refinedNode[T any] struct {
	schema *RefinedSchema[T]
}

func (n *refinedNode[T]) kind() nodeKind { return kindCustom }

func (n *refinedNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	out, ok := n.schema.inner.node().parse(ctx, in)
	if !ok {
		return nil, false
	}

	typed, ok := out.(T)
	if !ok {
		return out, true
	}

	for _, step := range n.schema.steps {
		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}
		if !step(ctx, typed) {
			if ctx.shouldAbort() {
				return nil, false
			}
		}
	}

	if len(ctx.issues) > 0 {
		return nil, false
	}
	return typed, true
}

// NewRefinedSchema wraps inner in a RefinedSchema.
func NewRefinedSchema[T any](inner Schema[T]) *RefinedSchema[T] {
	s := &RefinedSchema[T]{
		inner: inner,
	}
	s.BaseSchema = NewBaseSchema[T](&refinedNode[T]{schema: s})
	s.SetMeta(inner.GetMeta())
	return s
}

// Refine wraps s in a RefinedSchema and attaches a predicate.
func Refine[T any](s Schema[T], fn func(v T) bool, opts ...RefineOption) *RefinedSchema[T] {
	return NewRefinedSchema[T](s).Refine(fn, opts...)
}

// SuperRefine wraps s in a RefinedSchema and attaches an imperative inspector.
func SuperRefine[T any](s Schema[T], fn func(v T, c *RefineCtx)) *RefinedSchema[T] {
	return NewRefinedSchema[T](s).SuperRefine(fn)
}

// RefineWithCtx wraps s in a RefinedSchema and attaches a context-aware predicate.
func RefineWithCtx[T any](s Schema[T], fn func(ctx context.Context, v T) bool, opts ...RefineOption) *RefinedSchema[T] {
	return NewRefinedSchema[T](s).RefineCtx(fn, opts...)
}

// Check registers an issue-returning inspection function.
func (s *RefinedSchema[T]) Check(fn func(v T) []Issue) *RefinedSchema[T] {
	s.steps = append(s.steps, func(pc *parseCtx, v T) bool {
		issues := fn(v)
		if len(issues) > 0 {
			for _, iss := range issues {
				pc.addIssue(iss)
			}
			return false
		}
		return true
	})
	return s
}

// Refine adds a boolean predicate refinement.
func (s *RefinedSchema[T]) Refine(fn func(v T) bool, opts ...RefineOption) *RefinedSchema[T] {
	cfg := RefineConfig{
		Code: Custom,
	}
	for _, o := range opts {
		o(&cfg)
	}

	s.steps = append(s.steps, func(pc *parseCtx, v T) bool {
		if !fn(v) {
			for _, p := range cfg.Path {
				pc.pushPath(p)
			}
			iss := Issue{
				Code:    cfg.Code,
				Message: cfg.Message,
			}
			if cfg.MessageFunc != nil {
				iss.Message = cfg.MessageFunc(&iss)
			}
			pc.addIssue(iss)
			for range cfg.Path {
				pc.popPath()
			}
			return false
		}
		return true
	})
	return s
}

// SuperRefine adds an advanced imperative refinement using RefineCtx.
func (s *RefinedSchema[T]) SuperRefine(fn func(v T, c *RefineCtx)) *RefinedSchema[T] {
	s.steps = append(s.steps, func(pc *parseCtx, v T) bool {
		c := &RefineCtx{pc: pc}
		fn(v, c)
		return !c.failed
	})
	return s
}

// RefineCtx adds a context-aware boolean predicate.
func (s *RefinedSchema[T]) RefineCtx(fn func(ctx context.Context, v T) bool, opts ...RefineOption) *RefinedSchema[T] {
	cfg := RefineConfig{
		Code: Custom,
	}
	for _, o := range opts {
		o(&cfg)
	}

	s.steps = append(s.steps, func(pc *parseCtx, v T) bool {
		if err := pc.ctx.Err(); err != nil {
			pc.addIssue(Issue{
				Code:    Custom,
				Message: "cancelled",
			})
			return false
		}

		if !fn(pc.ctx, v) {
			for _, p := range cfg.Path {
				pc.pushPath(p)
			}
			iss := Issue{
				Code:    cfg.Code,
				Message: cfg.Message,
			}
			if cfg.MessageFunc != nil {
				iss.Message = cfg.MessageFunc(&iss)
			}
			pc.addIssue(iss)
			for range cfg.Path {
				pc.popPath()
			}
			return false
		}
		return true
	})
	return s
}

func (s *RefinedSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *RefinedSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *RefinedSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *RefinedSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *RefinedSchema[T]) Array() *ArraySchema[T] { return NewArraySchema[T](s) }
