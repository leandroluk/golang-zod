package z

import (
	"context"
	"sync/atomic"
)

// @spec REQ-CORE-011
// Default well-known context keys.
const (
	LocaleContextKey     = "github.com/leandroluk/golang-zod#locale"
	AbortEarlyContextKey = "github.com/leandroluk/golang-zod#abortEarly"
)

// @spec REQ-CORE-016
// ContextOptions configures the context keys used by the library.
type ContextOptions struct {
	LocaleKey     any
	AbortEarlyKey any
}

type configNamespace struct{}

// @spec REQ-CORE-016
// Config provides global startup configuration for golang-zod.
var Config configNamespace

var activeContextOptions atomic.Pointer[ContextOptions]

func init() {
	opts := ContextOptions{}
	activeContextOptions.Store(&opts)
}

// @spec REQ-CORE-016
// UseContext replaces the default context keys with application-provided keys.
// Calling UseContext(ContextOptions{}) restores the library defaults.
func (configNamespace) UseContext(o ContextOptions) {
	opts := o
	activeContextOptions.Store(&opts)
}

func getActiveLocaleKey() any {
	opts := activeContextOptions.Load()
	if opts != nil && opts.LocaleKey != nil && opts.LocaleKey != "" {
		return opts.LocaleKey
	}
	return LocaleContextKey
}

func getActiveAbortEarlyKey() any {
	opts := activeContextOptions.Load()
	if opts != nil && opts.AbortEarlyKey != nil && opts.AbortEarlyKey != "" {
		return opts.AbortEarlyKey
	}
	return AbortEarlyContextKey
}

// @spec REQ-CORE-011
// WithAbortEarly marks the context to halt schema validation upon the first encountered issue.
// If ctx is nil, context.Background() is used.
func WithAbortEarly(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	key := getActiveAbortEarlyKey()
	// nolint:staticcheck // String context key is by design (ADR-017)
	return context.WithValue(ctx, key, true)
}

func isAbortEarly(ctx context.Context) bool {
	key := getActiveAbortEarlyKey()
	val := ctx.Value(key)
	if val == nil {
		return false
	}
	b, ok := val.(bool)
	return ok && b
}
