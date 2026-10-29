package z_test

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func firstMessage(t *testing.T, err error) string {
	t.Helper()
	var ze *z.ZodError
	if !errors.As(err, &ze) || len(ze.Issues()) == 0 {
		t.Fatalf("expected ZodError with issues, got %v", err)
	}
	return ze.Issues()[0].Message
}

// @spec REQ-PIPE-007
func TestREQ_PIPE_007_ErrorCustomization(t *testing.T) {
	t.Run("schema-level string and function", func(t *testing.T) {
		_, err := z.String().Error("must be text").Parse(1)
		if got := firstMessage(t, err); got != "must be text" {
			t.Fatalf("got %q", got)
		}
		_, err = z.String().ErrorFunc(func(i *z.Issue) string { return "code=" + string(i.Code) }).Parse(1)
		if got := firstMessage(t, err); got != "code=invalid_type" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("function returning empty falls through to catalog", func(t *testing.T) {
		_, err := z.String().ErrorFunc(func(*z.Issue) string { return "" }).Parse(1)
		if got := firstMessage(t, err); !strings.Contains(got, "string") {
			t.Fatalf("expected catalog message, got %q", got)
		}
	})

	t.Run("check-level beats schema-level", func(t *testing.T) {
		s := z.String().Error("schema").Min(5, z.Message("check"))
		_, err := s.Parse("ab")
		if got := firstMessage(t, err); got != "check" {
			t.Fatalf("got %q", got)
		}
		_, err = s.Parse(1)
		if got := firstMessage(t, err); got != "schema" {
			t.Fatalf("type error should use schema-level, got %q", got)
		}
	})

	t.Run("check MessageFunc and number checks", func(t *testing.T) {
		s := z.Int().Min(3, z.MessageFunc(func(i *z.Issue) string { return "too small" }))
		_, err := s.Parse(1)
		if got := firstMessage(t, err); got != "too small" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("schema-level does not leak to children", func(t *testing.T) {
		obj := z.Object(func(s *z.ObjectBuilder) {
			s.Field("a", z.String())
		}).Error("object-level")
		_, err := obj.Parse(map[string]any{"a": 1})
		if got := firstMessage(t, err); got == "object-level" {
			t.Fatalf("child issue must not use the parent's schema-level message")
		}
		_, err = obj.Parse("not-an-object")
		if got := firstMessage(t, err); got != "object-level" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("global function is below schema level", func(t *testing.T) {
		z.Config.UseError(func(*z.Issue) string { return "global" })
		defer z.Config.UseError(nil)
		_, err := z.String().Parse(1)
		if got := firstMessage(t, err); got != "global" {
			t.Fatalf("got %q", got)
		}
		_, err = z.String().Error("schema").Parse(1)
		if got := firstMessage(t, err); got != "schema" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("Abort stops later checks", func(t *testing.T) {
		s := z.String().Min(5, z.Abort(true)).Regex(regexpDigits)
		_, err := s.Parse("ab")
		var ze *z.ZodError
		if !errors.As(err, &ze) || len(ze.Issues()) != 1 {
			t.Fatalf("expected exactly 1 issue, got %v", err)
		}
	})

	t.Run("exporters see through error wrappers", func(t *testing.T) {
		sch, err := z.ToJSONSchema(z.Object(func(s *z.ObjectBuilder) {
			s.Field("a", z.String().Error("x"))
		}).Error("y"))
		if err != nil || !strings.Contains(string(sch), `"type":"object"`) {
			t.Fatalf("unexpected export: %v %v", sch, err)
		}
	})
}

var regexpDigits = regexp.MustCompile(`^\d+$`)
