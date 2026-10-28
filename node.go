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
// @spec REQ-OPT-001
// parseCtx holds ephemeral execution state recycled through sync.Pool.
type parseCtx struct {
	ctx        context.Context
	locale     Locale
	abortEarly bool
	direction  parseDirection
	pathInline [8]PathSegment
	pathLen    int
	pathSpill  []PathSegment
	issues     []Issue

	schemaErr      *errorOverride
	schemaErrDepth int
	checkErr       *errorOverride
}

var parseCtxPool = sync.Pool{
	New: func() any {
		return &parseCtx{
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
	pc.pathLen = 0
	pc.pathSpill = pc.pathSpill[:0]
	pc.issues = pc.issues[:0]
	pc.schemaErr, pc.schemaErrDepth, pc.checkErr = nil, 0, nil
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
	pc.pathLen = 0
	pc.pathSpill = pc.pathSpill[:0]
	pc.issues = pc.issues[:0]
	pc.schemaErr, pc.schemaErrDepth, pc.checkErr = nil, 0, nil
	parseCtxPool.Put(pc)
}

// pushPath pushes a segment onto the context's path.
func (c *parseCtx) pushPath(seg PathSegment) {
	if c.pathLen < len(c.pathInline) {
		c.pathInline[c.pathLen] = seg
	} else {
		c.pathSpill = append(c.pathSpill, seg)
	}
	c.pathLen++
}

// popPath pops the last segment from the context's path.
func (c *parseCtx) popPath() {
	if c.pathLen > 0 {
		c.pathLen--
		if c.pathLen >= len(c.pathInline) {
			c.pathSpill = c.pathSpill[:len(c.pathSpill)-1]
		}
	}
}

func (c *parseCtx) currentPath() []PathSegment {
	if c.pathLen == 0 {
		return nil
	}
	res := make([]PathSegment, c.pathLen)
	inlineCount := min(c.pathLen, len(c.pathInline))
	copy(res, c.pathInline[:inlineCount])
	if c.pathLen > len(c.pathInline) {
		copy(res[inlineCount:], c.pathSpill)
	}
	return res
}

// addIssue records an issue, resolving its message via the active locale.
func (c *parseCtx) addIssue(issue Issue) {
	if len(issue.Path) == 0 && c.pathLen > 0 {
		issue.Path = c.currentPath()
	}
	if issue.Message == "" {
		issue.Message = c.customMessage(&issue)
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

// customMessage applies the error precedence: check, schema, global (locale catalog is the caller's fallback).
func (c *parseCtx) customMessage(issue *Issue) string {
	if msg := c.checkErr.resolve(issue); msg != "" {
		return msg
	}
	if c.schemaErr != nil && c.pathLen == c.schemaErrDepth {
		if msg := c.schemaErr.resolve(issue); msg != "" {
			return msg
		}
	}
	if g := globalErrorFn.Load(); g != nil {
		return (*g)(issue)
	}
	return ""
}
