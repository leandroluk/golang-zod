package z

import "sync/atomic"

// errorOverride carries a user-provided message customization: a fixed string or a function.
// A function returning "" falls through to the next precedence level.
type errorOverride struct {
	msg string
	fn  func(issue *Issue) string
}

func (o *errorOverride) resolve(issue *Issue) string {
	if o == nil {
		return ""
	}
	if o.fn != nil {
		return o.fn(issue)
	}
	return o.msg
}

// @spec REQ-PIPE-007
// CheckOption customizes a single check (message, message function, abort).
type CheckOption func(*checkOpts)

type checkOpts struct {
	err   errorOverride
	abort bool
}

func buildCheckOpts(opts []CheckOption) *checkOpts {
	if len(opts) == 0 {
		return nil
	}
	co := &checkOpts{}
	for _, o := range opts {
		o(co)
	}
	return co
}

// @spec REQ-PIPE-007
// Message sets a fixed error message for a check.
func Message(msg string) CheckOption {
	return func(c *checkOpts) { c.err = errorOverride{msg: msg} }
}

// @spec REQ-PIPE-007
// MessageFunc sets an error message function for a check; returning "" falls through.
func MessageFunc(fn func(issue *Issue) string) CheckOption {
	return func(c *checkOpts) { c.err = errorOverride{fn: fn} }
}

// @spec REQ-PIPE-007
// Abort makes the schema stop evaluating further checks when this check fails.
func Abort(abort bool) CheckOption {
	return func(c *checkOpts) { c.abort = abort }
}

var globalErrorFn atomic.Pointer[func(issue *Issue) string]

// @spec REQ-PIPE-007
// UseError installs a global error function (precedence level 4). Passing nil removes it.
func (configNamespace) UseError(fn func(issue *Issue) string) {
	if fn == nil {
		globalErrorFn.Store(nil)
		return
	}
	globalErrorFn.Store(&fn)
}

// errorNode applies a schema-level error override to the issues its inner node produces directly.
type errorNode struct {
	inner node
	ov    *errorOverride
}

func (n *errorNode) kind() nodeKind { return n.inner.kind() }

func (n *errorNode) parse(ctx *parseCtx, in any) (any, bool) {
	prevOv, prevDepth := ctx.schemaErr, ctx.schemaErrDepth
	ctx.schemaErr, ctx.schemaErrDepth = n.ov, ctx.pathLen
	out, ok := n.inner.parse(ctx, in)
	ctx.schemaErr, ctx.schemaErrDepth = prevOv, prevDepth
	return out, ok
}

// unwrapNode strips schema-level error wrappers so exporters can inspect the real node.
func unwrapNode(n node) node {
	for {
		en, ok := n.(*errorNode)
		if !ok {
			return n
		}
		n = en.inner
	}
}

func (c *parseCtx) beginCheck(o *checkOpts) int {
	if o != nil {
		c.checkErr = &o.err
	}
	return len(c.issues)
}

// endCheck clears the check override and reports whether the check failed with Abort set.
func (c *parseCtx) endCheck(mark int, o *checkOpts) bool {
	c.checkErr = nil
	return o != nil && o.abort && len(c.issues) > mark
}
