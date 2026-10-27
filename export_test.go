package z

import "context"

type testNode struct {
	fn func(ctx any, in any) (out any, ok bool)
}

func (t *testNode) parse(ctx *parseCtx, in any) (out any, ok bool) {
	return t.fn(ctx, in)
}

func (t *testNode) kind() nodeKind {
	return kindString
}

// NewTestSchema exposes schema creation for external package z_test.
func NewTestSchema[T any](fn func(ctx any, in any) (out any, ok bool)) Schema[T] {
	base := NewBaseSchema[T](&testNode{fn: fn})
	return &base
}

func ContextAddIssue(pc any, issue Issue) {
	if p, ok := pc.(*parseCtx); ok {
		p.addIssue(issue)
	}
}

func ContextCheckCancel(pc any) error {
	if p, ok := pc.(*parseCtx); ok {
		return p.checkCancel()
	}
	return nil
}

func AcquireParseCtx(ctx context.Context) any {
	return acquireParseCtx(ctx)
}

func ReleaseParseCtx(pc any) {
	if p, ok := pc.(*parseCtx); ok {
		releaseParseCtx(p)
	}
}

func ContextIssuesCount(pc any) int {
	if p, ok := pc.(*parseCtx); ok {
		return len(p.issues)
	}
	return 0
}

