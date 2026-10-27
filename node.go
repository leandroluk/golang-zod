package z

import (
	"context"
	"sync"
)

// @spec REQ-CORE-001
type nodeKind int

const (
	kindUnknown nodeKind = iota
	kindString
	kindNumber
	kindBoolean
	kindDate
	kindBytes
	kindBigInt
	kindLiteral
	kindEnum
	kindArray
	kindTuple
	kindSet
	kindMap
	kindObject
	kindStruct
	kindRecord
	kindUnion
	kindDiscriminatedUnion
	kindIntersection
	kindLazy
	kindOptional
	kindNullable
	kindNullish
	kindDefault
	kindPrefault
	kindCatch
	kindReadonly
	kindTransform
	kindPipe
	kindCodec
	kindCustom
)

// @spec REQ-CORE-001
// node is the unexported untyped engine interface for validation and transformation.
type node interface {
	parse(ctx *parseCtx, in any) (out any, ok bool)
	kind() nodeKind
}

type parseDirection int

const (
	directionParse parseDirection = iota
	directionEncode
)

// @spec REQ-CORE-001
// @spec REQ-CORE-012
// @spec REQ-CORE-017
// parseCtx holds ephemeral execution state recycled through sync.Pool.
type parseCtx struct {
	ctx        context.Context
	locale     Locale
	abortEarly bool
	direction  parseDirection
	path       []PathSegment
	issues     []Issue
}

var parseCtxPool = sync.Pool{
	New: func() any {
		return &parseCtx{
			path:   make([]PathSegment, 0, 16),
			issues: make([]Issue, 0, 8),
		}
	},
}

// acquireParseCtx retrieves a clean parseCtx from the pool.
func acquireParseCtx(ctx context.Context) *parseCtx {
	pc := parseCtxPool.Get().(*parseCtx)
	if ctx == nil {
		ctx = context.Background()
	}
	pc.ctx = ctx
	pc.locale = resolveActiveLocale(ctx)
	pc.abortEarly = isAbortEarly(ctx)
	pc.direction = directionParse
	pc.path = pc.path[:0]
	pc.issues = pc.issues[:0]
	return pc
}

// releaseParseCtx resets and returns the parseCtx to the pool.
func releaseParseCtx(pc *parseCtx) {
	if pc == nil {
		return
	}
	pc.ctx = nil
	pc.locale = nil
	pc.abortEarly = false
	pc.direction = directionParse
	pc.path = pc.path[:0]
	pc.issues = pc.issues[:0]
	parseCtxPool.Put(pc)
}

// pushPath pushes a segment onto the context's path.
func (c *parseCtx) pushPath(seg PathSegment) {
	c.path = append(c.path, seg)
}

// popPath pops the last segment from the context's path.
func (c *parseCtx) popPath() {
	if len(c.path) > 0 {
		c.path = c.path[:len(c.path)-1]
	}
}

// addIssue records an issue, resolving its message via the active locale.
func (c *parseCtx) addIssue(issue Issue) {
	if len(issue.Path) == 0 && len(c.path) > 0 {
		issue.Path = make([]PathSegment, len(c.path))
		copy(issue.Path, c.path)
	}
	resolveIssueMessage(&issue, c.locale)
	c.issues = append(c.issues, issue)
}

// shouldAbort reports whether parsing should halt immediately.
func (c *parseCtx) shouldAbort() bool {
	return c.abortEarly && len(c.issues) > 0
}

// @spec REQ-CORE-017
// checkCancel performs a non-blocking check on context cancellation/deadline.
func (c *parseCtx) checkCancel() error {
	if c.ctx == nil || c.ctx.Done() == nil {
		return nil
	}
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
		return nil
	}
}
