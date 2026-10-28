package z_test

import (
	"errors"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func coverCIssues(t *testing.T, err error) []z.Issue {
	t.Helper()
	var ze *z.ZodError
	if !errors.As(err, &ze) {
		t.Fatalf("expected *z.ZodError, got %T (%v)", err, err)
	}
	return ze.Issues()
}

func coverCPath(is z.Issue) string {
	parts := make([]string, len(is.Path))
	for i, seg := range is.Path {
		parts[i] = seg.String()
	}
	return strings.Join(parts, ".")
}

func coverCMustPanic(t *testing.T, wantSubstr string, fn func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", wantSubstr)
		}
		if msg, _ := r.(string); !strings.Contains(msg, wantSubstr) {
			t.Fatalf("panic %q does not contain %q", r, wantSubstr)
		}
	}()
	fn()
}
