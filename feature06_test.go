package z_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-PIPE-001
func TestREQ_PIPE_001_Modifiers(t *testing.T) {
	t.Run("REQ-PIPE-001: Default substitutes only on absent, not null", func(t *testing.T) {
		s := z.String().Default("fallback")

		// Nil input -> fallback
		v, err := s.Parse(nil)
		if err != nil {
			t.Fatalf("unexpected error on nil: %v", err)
		}
		if v != "fallback" {
			t.Fatalf("expected 'fallback', got %q", v)
		}

		// Provided value
		v, err = s.Parse("hello")
		if err != nil {
			t.Fatalf("unexpected error on hello: %v", err)
		}
		if v != "hello" {
			t.Fatalf("expected 'hello', got %q", v)
		}
	})

	t.Run("REQ-PIPE-001: Prefault validates fallback value through inner schema", func(t *testing.T) {
		// Min length 5, prefault "short" should fail inner validation
		failingPrefault := z.NewPrefaultSchema(z.String().Min(5), "abc")
		_, err := failingPrefault.Parse(nil)
		if err == nil {
			t.Fatal("expected prefault 'abc' to fail min length 5")
		}

		// Valid prefault
		validPrefault := z.NewPrefaultSchema(z.String().Min(5), "valid-string")
		v, err := validPrefault.Parse(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != "valid-string" {
			t.Fatalf("expected 'valid-string', got %q", v)
		}
	})

	t.Run("REQ-PIPE-001: Catch recovers on any validation failure", func(t *testing.T) {
		s := z.NewCatchSchema(z.Int().GTE(10), 999)

		v, err := s.Parse(5) // Fails GTE(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 999 {
			t.Fatalf("expected 999, got %d", v)
		}

		v, err = s.Parse("invalid-type")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if v != 999 {
			t.Fatalf("expected 999, got %d", v)
		}
	})

	t.Run("REQ-PIPE-001: Readonly marks schema metadata", func(t *testing.T) {
		s := z.NewReadonlySchema(z.String())
		if !s.GetMeta().ReadOnly {
			t.Fatal("expected ReadOnly to be true")
		}
	})
}

// @spec REQ-PIPE-003
func TestREQ_PIPE_003_Refinements(t *testing.T) {
	t.Run("REQ-PIPE-003: Check, Refine, SuperRefine, RefineCtx", func(t *testing.T) {
		// Refine
		isEven := z.Refine(z.Int(), func(v int) bool {
			return v%2 == 0
		}, z.RefineMessage("number must be even"))

		v, err := isEven.Parse(4)
		if err != nil {
			t.Fatalf("unexpected error on 4: %v", err)
		}
		if v != 4 {
			t.Fatalf("expected 4, got %d", v)
		}

		_, err = isEven.Parse(5)
		if err == nil {
			t.Fatal("expected error for 5 in even refinement")
		}

		// SuperRefine
		super := z.SuperRefine(z.String(), func(v string, c *z.RefineCtx) {
			if !strings.HasPrefix(v, "prefix_") {
				c.AddIssue(z.Issue{
					Code:    z.Custom,
					Message: "must start with prefix_",
				})
			}
		})

		_, err = super.Parse("bad")
		if err == nil {
			t.Fatal("expected SuperRefine error")
		}

		// RefineCtx with context cancellation
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		refineCtx := z.RefineWithCtx(z.Int(), func(c context.Context, v int) bool {
			return true
		})

		_, err = refineCtx.Parse(10, ctx)
		if err == nil {
			t.Fatal("expected cancellation issue on cancelled context")
		}
	})
}

// @spec REQ-PIPE-004
func TestREQ_PIPE_004_Pipelines(t *testing.T) {
	t.Run("REQ-PIPE-004: Transform, Preprocess, Pipe", func(t *testing.T) {
		// Transform
		toUpper := z.Transform(z.String(), func(s string) (string, error) {
			return strings.ToUpper(s), nil
		})

		res, err := toUpper.Parse("hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != "HELLO" {
			t.Fatalf("expected HELLO, got %q", res)
		}

		// Pipe
		pipe := z.Pipe(z.String(), z.Codecs.StringInt())
		intRes, err := pipe.Parse("12345")
		if err != nil {
			t.Fatalf("unexpected error on pipe: %v", err)
		}
		if intRes != 12345 {
			t.Fatalf("expected 12345, got %d", intRes)
		}

		// Preprocess
		prep := z.Preprocess(func(raw any) any {
			if s, ok := raw.(string); ok {
				return len(s)
			}
			return raw
		}, z.Int())

		pRes, err := prep.Parse("hello world")
		if err != nil {
			t.Fatalf("unexpected error on preprocess: %v", err)
		}
		if pRes != 11 {
			t.Fatalf("expected 11, got %d", pRes)
		}
	})
}

// @spec REQ-PIPE-005 REQ-PIPE-006 REQ-PIPE-008
func TestREQ_PIPE_005_006_008_Codecs(t *testing.T) {
	t.Run("REQ-PIPE-005 & REQ-PIPE-008: Codecs Decode, Encode, and Recipes", func(t *testing.T) {
		// StringInt
		codec := z.Codecs.StringInt()
		val, err := codec.DecodeValue("42")
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if val != 42 {
			t.Fatalf("expected 42, got %d", val)
		}

		encoded, err := codec.Encode(42)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}
		if encoded != "42" {
			t.Fatalf("expected '42', got %q", encoded)
		}

		// Base64Bytes
		b64 := z.Codecs.Base64Bytes()
		bytesVal, err := b64.DecodeValue("aGVsbG8=")
		if err != nil {
			t.Fatalf("decode b64 failed: %v", err)
		}
		if string(bytesVal) != "hello" {
			t.Fatalf("expected hello, got %s", string(bytesVal))
		}

		b64Enc, err := b64.Encode([]byte("hello"))
		if err != nil {
			t.Fatalf("encode b64 failed: %v", err)
		}
		if b64Enc != "aGVsbG8=" {
			t.Fatalf("expected aGVsbG8=, got %s", b64Enc)
		}

		// EpochDate
		epoch := z.Codecs.EpochDate()
		now := time.Now().Truncate(time.Millisecond)
		ts, err := epoch.Encode(now)
		if err != nil {
			t.Fatalf("encode epoch failed: %v", err)
		}
		decodedTime, err := epoch.DecodeValue(ts)
		if err != nil {
			t.Fatalf("decode epoch failed: %v", err)
		}
		if decodedTime.UnixMilli() != now.UnixMilli() {
			t.Fatalf("expected %v, got %v", now, decodedTime)
		}
	})

	t.Run("REQ-PIPE-006: StringBool with options", func(t *testing.T) {
		sb := z.StringBool(
			z.StringBoolTruthy("SIM", "YES"),
			z.StringBoolFalsy("NAO", "NO"),
			z.StringBoolCaseSensitive(false),
		)

		b, err := sb.DecodeValue("sim")
		if err != nil || !b {
			t.Fatalf("expected true, got %v (err=%v)", b, err)
		}

		b, err = sb.DecodeValue("nao")
		if err != nil || b {
			t.Fatalf("expected false, got %v (err=%v)", b, err)
		}

		enc, err := sb.Encode(true)
		if err != nil || enc != "SIM" {
			t.Fatalf("expected 'SIM', got %q", enc)
		}

		enc, err = sb.Encode(false)
		if err != nil || enc != "NAO" {
			t.Fatalf("expected 'NAO', got %q", enc)
		}
	})

	t.Run("REQ-PIPE-005: Encode unsupported error", func(t *testing.T) {
		oneWay := z.Codec(z.String(), z.String(), func(s string) (string, error) {
			return s, nil
		}, nil)

		_, err := oneWay.Encode("hello")
		if !errors.Is(err, z.ErrEncodeUnsupported) {
			t.Fatalf("expected ErrEncodeUnsupported, got %v", err)
		}
	})
}
