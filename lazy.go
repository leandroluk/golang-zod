package z

import (
	"sync"
)

// @spec REQ-COMB-008
// Lazy defines a recursively resolved schema, memoized on first access using sync.OnceValue.
func Lazy[T any](fn func() Schema[T]) Schema[T] {
	memo := sync.OnceValue(fn)
	n := &lazyNode[T]{
		memo: memo,
	}
	s := &lazySchema[T]{
		BaseSchema: NewBaseSchema[T](n),
		nodeRef:    n,
	}
	return s
}

type lazyNode[T any] struct {
	memo func() Schema[T]
}

func (n *lazyNode[T]) kind() nodeKind {
	return kindLazy
}

func (n *lazyNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if err := ctx.checkCancel(); err != nil {
		return nil, false
	}
	resolved := n.memo()
	return resolved.node().parse(ctx, in)
}

type lazySchema[T any] struct {
	BaseSchema[T]
	nodeRef *lazyNode[T]
}

func (s *lazySchema[T]) GetMeta() Meta {
	resolved := s.nodeRef.memo()
	return resolved.GetMeta()
}
