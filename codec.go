package z

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// @spec REQ-PIPE-005
// ErrEncodeUnsupported is returned when attempting to encode through a one-way pipeline or transform.
var ErrEncodeUnsupported = errors.New("golang-zod: encode is unsupported for this transformation")

// @spec REQ-PIPE-005
// CodecSchema defines a bidirectional converter between format A and typed value B.
type CodecSchema[A, B any] struct {
	BaseSchema[B]
	inSchema  Schema[A]
	outSchema Schema[B]
	decodeFn  func(a A) (B, error)
	encodeFn  func(b B) (A, error)
}

type codecNode[A, B any] struct {
	schema *CodecSchema[A, B]
}

func (n *codecNode[A, B]) kind() nodeKind { return kindCodec }

func (n *codecNode[A, B]) parse(ctx *parseCtx, in any) (any, bool) {
	if err := ctx.checkCancel(); err != nil {
		return nil, false
	}

	aOut, ok := n.schema.inSchema.node().parse(ctx, in)
	if !ok {
		return nil, false
	}

	aTyped, ok := aOut.(A)
	if !ok {
		ctx.addIssue(Issue{
			Code:    Custom,
			Message: fmt.Sprintf("codec input type mismatch: expected %T, got %T", *new(A), aOut),
		})
		return nil, false
	}

	bVal, err := n.schema.decodeFn(aTyped)
	if err != nil {
		ctx.addIssue(Issue{
			Code:    Custom,
			Message: err.Error(),
		})
		return nil, false
	}

	// Validate target type B against outSchema if provided
	if n.schema.outSchema != nil {
		bOut, ok := n.schema.outSchema.node().parse(ctx, bVal)
		if !ok {
			return nil, false
		}
		return bOut, true
	}

	return bVal, true
}

// Codec creates a bidirectional codec schema converting between A and B.
func Codec[A, B any](
	in Schema[A],
	out Schema[B],
	decode func(A) (B, error),
	encode func(B) (A, error),
) *CodecSchema[A, B] {
	cs := &CodecSchema[A, B]{
		inSchema:  in,
		outSchema: out,
		decodeFn:  decode,
		encodeFn:  encode,
	}
	cs.BaseSchema = NewBaseSchema[B](&codecNode[A, B]{schema: cs})
	if out != nil {
		cs.SetMeta(out.GetMeta())
	}
	return cs
}

// DecodeValue transforms raw input into type B.
func (s *CodecSchema[A, B]) DecodeValue(in any, ctx ...context.Context) (B, error) {
	return s.Parse(in, ctx...)
}

// Encode converts type B back into representation A.
func (s *CodecSchema[A, B]) Encode(b B, ctx ...context.Context) (A, error) {
	c := extractContext(ctx)
	if err := c.Err(); err != nil {
		var zero A
		return zero, err
	}

	if s.encodeFn == nil {
		var zero A
		return zero, ErrEncodeUnsupported
	}

	// 1. If outSchema exists, check validity of B
	if s.outSchema != nil {
		pc := acquireParseCtx(c)
		_, ok := s.outSchema.node().parse(pc, b)
		hasIssues := len(pc.issues) > 0
		if !ok || hasIssues {
			var issues []Issue
			if hasIssues {
				issues = make([]Issue, len(pc.issues))
				copy(issues, pc.issues)
			}
			releaseParseCtx(pc)
			var zero A
			return zero, NewZodError(issues)
		}
		releaseParseCtx(pc)
	}

	// 2. Encode to A
	aVal, err := s.encodeFn(b)
	if err != nil {
		var zero A
		return zero, err
	}

	// 3. Validate A conforms to inSchema
	pc := acquireParseCtx(c)
	_, ok := s.inSchema.node().parse(pc, aVal)
	hasIssues := len(pc.issues) > 0
	if !ok || hasIssues {
		var issues []Issue
		if hasIssues {
			issues = make([]Issue, len(pc.issues))
			copy(issues, pc.issues)
		}
		releaseParseCtx(pc)
		var zero A
		return zero, NewZodError(issues)
	}
	releaseParseCtx(pc)

	return aVal, nil
}

// @spec REQ-PIPE-006
// StringBoolConfig configures StringBool options.
type StringBoolConfig struct {
	truthy        []string
	falsy         []string
	caseSensitive bool
}

// StringBoolOption configures a StringBool codec.
type StringBoolOption func(*StringBoolConfig)

// StringBoolTruthy specifies custom strings accepted as true.
func StringBoolTruthy(values ...string) StringBoolOption {
	return func(c *StringBoolConfig) {
		c.truthy = values
	}
}

// StringBoolFalsy specifies custom strings accepted as false.
func StringBoolFalsy(values ...string) StringBoolOption {
	return func(c *StringBoolConfig) {
		c.falsy = values
	}
}

// StringBoolCaseSensitive toggles case-sensitivity (default false).
func StringBoolCaseSensitive(cs bool) StringBoolOption {
	return func(c *StringBoolConfig) {
		c.caseSensitive = cs
	}
}

// StringBool creates a codec parsing booleans from strings with configurable truthy/falsy sets.
func StringBool(opts ...StringBoolOption) *CodecSchema[string, bool] {
	cfg := StringBoolConfig{
		truthy:        []string{"true", "1", "yes", "y", "on"},
		falsy:         []string{"false", "0", "no", "n", "off"},
		caseSensitive: false,
	}
	for _, o := range opts {
		o(&cfg)
	}

	decode := func(s string) (bool, error) {
		cmp := s
		if !cfg.caseSensitive {
			cmp = strings.ToLower(s)
		}

		for _, t := range cfg.truthy {
			candidate := t
			if !cfg.caseSensitive {
				candidate = strings.ToLower(t)
			}
			if cmp == candidate {
				return true, nil
			}
		}

		for _, f := range cfg.falsy {
			candidate := f
			if !cfg.caseSensitive {
				candidate = strings.ToLower(f)
			}
			if cmp == candidate {
				return false, nil
			}
		}

		return false, fmt.Errorf("invalid boolean string %q", s)
	}

	encode := func(b bool) (string, error) {
		if b {
			if len(cfg.truthy) > 0 {
				return cfg.truthy[0], nil
			}
			return "true", nil
		}
		if len(cfg.falsy) > 0 {
			return cfg.falsy[0], nil
		}
		return "false", nil
	}

	return Codec[string, bool](String(), Boolean(), decode, encode)
}

// @spec REQ-PIPE-008
// CodecRecipes provides built-in recipe codecs.
type CodecRecipes struct{}

// Codecs provides standard recipe codecs.
var Codecs = CodecRecipes{}

// JSONString creates a codec converting raw JSON string into any.
func (CodecRecipes) JSONString() *CodecSchema[string, any] {
	decode := func(s string) (any, error) {
		var out any
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	encode := func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return Codec[string, any](String(), nil, decode, encode)
}

// Base64Bytes creates a codec converting base64 encoded strings to []byte and vice-versa.
func (CodecRecipes) Base64Bytes() *CodecSchema[string, []byte] {
	decode := func(s string) ([]byte, error) {
		return base64.StdEncoding.DecodeString(s)
	}
	encode := func(b []byte) (string, error) {
		return base64.StdEncoding.EncodeToString(b), nil
	}
	return Codec[string, []byte](String(), Bytes(), decode, encode)
}

// StringInt creates a codec converting string numbers into int.
func (CodecRecipes) StringInt() *CodecSchema[string, int] {
	decode := func(s string) (int, error) {
		return strconv.Atoi(s)
	}
	encode := func(i int) (string, error) {
		return strconv.Itoa(i), nil
	}
	return Codec[string, int](String(), Int(), decode, encode)
}

// StringNumber creates a codec converting string numbers into float64.
func (CodecRecipes) StringNumber() *CodecSchema[string, float64] {
	decode := func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}
	encode := func(f float64) (string, error) {
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	return Codec[string, float64](String(), Float64(), decode, encode)
}

// EpochDate creates a codec converting unix timestamp milliseconds or seconds into time.Time.
func (CodecRecipes) EpochDate() *CodecSchema[int64, time.Time] {
	decode := func(ts int64) (time.Time, error) {
		if ts < 100_000_000_000 {
			return time.Unix(ts, 0), nil
		}
		return time.UnixMilli(ts), nil
	}
	encode := func(t time.Time) (int64, error) {
		return t.UnixMilli(), nil
	}
	return Codec[int64, time.Time](Int64(), Date(), decode, encode)
}
