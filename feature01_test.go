package z_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-CORE-004
func TestREQ_CORE_004_OptionNullable(t *testing.T) {
	t.Run("Option Some and None", func(t *testing.T) {
		s := z.Some("hello")
		if !s.IsSome() {
			t.Errorf("expected IsSome() == true")
		}
		if v, ok := s.Value(); !ok || v != "hello" {
			t.Errorf("expected Value() == (\"hello\", true), got (%q, %v)", v, ok)
		}
		if s.OrElse("default") != "hello" {
			t.Errorf("expected OrElse == 'hello'")
		}
		if s.IsZero() {
			t.Errorf("expected IsZero() == false for Some")
		}

		n := z.None[string]()
		if n.IsSome() {
			t.Errorf("expected IsSome() == false")
		}
		if v, ok := n.Value(); ok || v != "" {
			t.Errorf("expected Value() == (\"\", false), got (%q, %v)", v, ok)
		}
		if n.OrElse("default") != "default" {
			t.Errorf("expected OrElse == 'default'")
		}
		if !n.IsZero() {
			t.Errorf("expected IsZero() == true for None")
		}
	})

	t.Run("Nullable Value and Null", func(t *testing.T) {
		val := z.NullableOf(42)
		if val.IsNull() {
			t.Errorf("expected IsNull() == false")
		}
		if v, ok := val.Value(); !ok || v != 42 {
			t.Errorf("expected Value() == (42, true), got (%v, %v)", v, ok)
		}
		if val.OrElse(100) != 42 {
			t.Errorf("expected OrElse == 42")
		}

		nullVal := z.NullableNull[int]()
		if !nullVal.IsNull() {
			t.Errorf("expected IsNull() == true")
		}
		if v, ok := nullVal.Value(); ok || v != 0 {
			t.Errorf("expected Value() == (0, false), got (%v, %v)", v, ok)
		}
		if nullVal.OrElse(100) != 100 {
			t.Errorf("expected OrElse == 100")
		}
	})

	t.Run("Option JSON marshaling and unmarshaling", func(t *testing.T) {
		s := z.Some("world")
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}
		if string(data) != `"world"` {
			t.Errorf("expected `\"world\"`, got %s", data)
		}

		var s2 z.Option[string]
		if err := json.Unmarshal(data, &s2); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if v, ok := s2.Value(); !ok || v != "world" {
			t.Errorf("expected 'world', got (%q, %v)", v, ok)
		}

		n := z.None[string]()
		dataNull, err := json.Marshal(n)
		if err != nil {
			t.Fatalf("Marshal None failed: %v", err)
		}
		if string(dataNull) != `null` {
			t.Errorf("expected `null`, got %s", dataNull)
		}
	})

	t.Run("Nullable JSON marshaling and unmarshaling", func(t *testing.T) {
		nullVal := z.NullableNull[string]()
		data, err := json.Marshal(nullVal)
		if err != nil {
			t.Fatalf("Marshal NullableNull failed: %v", err)
		}
		if string(data) != `null` {
			t.Errorf("expected `null`, got %s", data)
		}

		var n2 z.Nullable[string]
		if err := json.Unmarshal([]byte(`null`), &n2); err != nil {
			t.Fatalf("Unmarshal null failed: %v", err)
		}
		if !n2.IsNull() {
			t.Errorf("expected n2.IsNull() == true")
		}

		val := z.NullableOf("test")
		dataVal, err := json.Marshal(val)
		if err != nil {
			t.Fatalf("Marshal NullableOf failed: %v", err)
		}
		if string(dataVal) != `"test"` {
			t.Errorf("expected `\"test\"`, got %s", dataVal)
		}

		var n3 z.Nullable[string]
		if err := json.Unmarshal(dataVal, &n3); err != nil {
			t.Fatalf("Unmarshal test failed: %v", err)
		}
		if v, ok := n3.Value(); !ok || v != "test" {
			t.Errorf("expected 'test', got (%q, %v)", v, ok)
		}
	})

	t.Run("Nullish Option[Nullable[T]] combination", func(t *testing.T) {
		// Nullish: absent (None)
		absent := z.None[z.Nullable[string]]()
		if absent.IsSome() {
			t.Errorf("expected absent.IsSome() == false")
		}

		// Nullish: present and null
		presentNull := z.Some(z.NullableNull[string]())
		if !presentNull.IsSome() {
			t.Errorf("expected presentNull.IsSome() == true")
		}
		if inner, ok := presentNull.Value(); !ok || !inner.IsNull() {
			t.Errorf("expected inner.IsNull() == true")
		}

		// Nullish: present and value
		presentVal := z.Some(z.NullableOf("data"))
		if !presentVal.IsSome() {
			t.Errorf("expected presentVal.IsSome() == true")
		}
		if inner, ok := presentVal.Value(); !ok || inner.IsNull() {
			t.Errorf("expected inner.IsNull() == false")
		} else if v, ok2 := inner.Value(); !ok2 || v != "data" {
			t.Errorf("expected inner.Value() == 'data'")
		}
	})
}

// @spec REQ-CORE-005
func TestREQ_CORE_005_IssueCode(t *testing.T) {
	expectedCodes := map[z.IssueCode]string{
		z.InvalidType:      "invalid_type",
		z.TooBig:           "too_big",
		z.TooSmall:         "too_small",
		z.InvalidFormat:    "invalid_format",
		z.NotMultipleOf:    "not_multiple_of",
		z.UnrecognizedKeys: "unrecognized_keys",
		z.InvalidUnion:     "invalid_union",
		z.InvalidKey:       "invalid_key",
		z.InvalidElement:   "invalid_element",
		z.InvalidValue:     "invalid_value",
		z.Custom:           "custom",
	}

	if len(expectedCodes) != 11 {
		t.Fatalf("expected exactly 11 issue codes, got %d", len(expectedCodes))
	}

	for code, wire := range expectedCodes {
		if string(code) != wire {
			t.Errorf("code %v has wire value %q, expected %q", code, string(code), wire)
		}
	}
}

// @spec REQ-CORE-006
func TestREQ_CORE_006_IssueAndZodError(t *testing.T) {
	t.Run("PathSegment Property and Index", func(t *testing.T) {
		prop := z.Property("users")
		if prop.IsIndex() {
			t.Errorf("expected prop.IsIndex() == false")
		}
		if prop.Key() != "users" {
			t.Errorf("expected prop.Key() == 'users'")
		}
		if prop.String() != "users" {
			t.Errorf("expected prop.String() == 'users'")
		}

		idx := z.Index(3)
		if !idx.IsIndex() {
			t.Errorf("expected idx.IsIndex() == true")
		}
		if idx.IndexVal() != 3 {
			t.Errorf("expected idx.IndexVal() == 3")
		}
		if idx.String() != "3" {
			t.Errorf("expected idx.String() == '3'")
		}
	})

	t.Run("ZodError Issues and Error interface", func(t *testing.T) {
		issues := []z.Issue{
			{
				Code:     z.InvalidType,
				Path:     []z.PathSegment{z.Property("name")},
				Message:  "Expected string, received number",
				Expected: "string",
			},
		}

		err := z.NewZodError(issues)
		if len(err.Issues()) != 1 {
			t.Fatalf("expected 1 issue, got %d", len(err.Issues()))
		}
		if err.Error() == "" {
			t.Errorf("expected non-empty Error() string")
		}
	})
}

// @spec REQ-CORE-007
func TestREQ_CORE_007_Formatters(t *testing.T) {
	issues := []z.Issue{
		{
			Code:    z.InvalidType,
			Path:    nil, // form error
			Message: "Invalid form payload",
		},
		{
			Code:    z.TooSmall,
			Path:    []z.PathSegment{z.Property("users"), z.Index(0), z.Property("email")},
			Message: "Email is too short",
		},
		{
			Code:    z.InvalidFormat,
			Path:    []z.PathSegment{z.Property("users"), z.Index(1), z.Property("name")},
			Message: "Name is invalid",
		},
		{
			Code:    z.InvalidType,
			Path:    []z.PathSegment{z.Property("age")},
			Message: "Age must be a number",
		},
	}

	err := z.NewZodError(issues)

	t.Run("Flatten", func(t *testing.T) {
		flat := err.Flatten()
		if len(flat.FormErrors) != 1 || flat.FormErrors[0] != "Invalid form payload" {
			t.Errorf("unexpected FormErrors: %v", flat.FormErrors)
		}

		// FieldErrors should group by the FIRST segment ("users" and "age")
		if len(flat.FieldErrors["users"]) != 2 {
			t.Errorf("expected 2 errors for 'users', got %v", flat.FieldErrors["users"])
		}
		if len(flat.FieldErrors["age"]) != 1 {
			t.Errorf("expected 1 error for 'age', got %v", flat.FieldErrors["age"])
		}
	})

	t.Run("Treeify", func(t *testing.T) {
		tree := err.Treeify()
		if len(tree.Errors) != 1 || tree.Errors[0] != "Invalid form payload" {
			t.Errorf("root node expected 1 form error, got %v", tree.Errors)
		}

		userProp, ok := tree.Properties["users"]
		if !ok {
			t.Fatalf("expected 'users' property in error tree")
		}
		item0, ok := userProp.Items[0]
		if !ok {
			t.Fatalf("expected item 0 in users property")
		}
		emailProp, ok := item0.Properties["email"]
		if !ok {
			t.Fatalf("expected 'email' property in item 0")
		}
		if len(emailProp.Errors) != 1 || emailProp.Errors[0] != "Email is too short" {
			t.Errorf("expected email error, got %v", emailProp.Errors)
		}
	})

	t.Run("Prettify", func(t *testing.T) {
		pretty := err.Prettify()
		// Root form error (path len 0) must appear before path len 1, which appears before path len 3
		lines := strings.Split(pretty, "\n")
		if len(lines) < 4 {
			t.Fatalf("expected at least 4 lines, got %d:\n%s", len(lines), pretty)
		}
		if lines[0] != "✖ Invalid form payload" {
			t.Errorf("first line should be form error, got %q", lines[0])
		}
		if !strings.Contains(pretty, "  → at users[0].email") {
			t.Errorf("expected formatted path users[0].email in prettify output:\n%s", pretty)
		}
	})
}

// @spec REQ-CORE-008
// @spec REQ-CORE-009
// @spec REQ-CORE-013
// @spec REQ-CORE-014
// @spec REQ-CORE-015
func TestREQ_CORE_008_009_013_014_I18n(t *testing.T) {
	t.Run("Message formatting across built-in locales", func(t *testing.T) {
		issue := z.Issue{
			Code:      z.TooSmall,
			Origin:    "string",
			Minimum:   5,
			Inclusive: true,
		}

		msgEN := z.LocaleEN.FormatMessage(&issue)
		if !strings.Contains(msgEN, "at least 5 character(s)") {
			t.Errorf("unexpected EN message: %q", msgEN)
		}

		msgPT := z.LocalePtBR.FormatMessage(&issue)
		if !strings.Contains(msgPT, "no mínimo 5 caractere(s)") {
			t.Errorf("unexpected PT message: %q", msgPT)
		}

		msgES := z.LocaleES.FormatMessage(&issue)
		if !strings.Contains(msgES, "al menos 5 carácter(es)") {
			t.Errorf("unexpected ES message: %q", msgES)
		}
	})

	t.Run("Accept-Language parsing and negotiation", func(t *testing.T) {
		// Exact match
		if loc := z.ResolveLocale("pt-BR"); loc != z.LocalePtBR {
			t.Errorf("expected LocalePtBR, got %v", loc)
		}

		// Weighted header
		header := "pt-BR,pt;q=0.9,en;q=0.8"
		if loc := z.ResolveLocale(header); loc != z.LocalePtBR {
			t.Errorf("expected LocalePtBR for weighted header, got %v", loc)
		}

		// Prefix fallback: es-MX -> es
		if loc := z.ResolveLocale("es-MX"); loc != z.LocaleES {
			t.Errorf("expected LocaleES for es-MX, got %v", loc)
		}

		// Unknown tag returns nil
		if loc := z.ResolveLocale("xx-YY"); loc != nil {
			t.Errorf("expected nil for unknown locale, got %v", loc)
		}
	})

	t.Run("WithLocale derivation", func(t *testing.T) {
		ctx := context.Background()

		// Unknown language returns ctx unchanged
		ctxSame := z.WithLocale(ctx, "xx-YY")
		if ctxSame != ctx {
			t.Errorf("expected unchanged context for unknown locale")
		}

		// Valid locale attaches value
		ctxPT := z.WithLocale(ctx, "pt-BR")
		val := ctxPT.Value(z.LocaleContextKey)
		if val != z.LocalePtBR {
			t.Errorf("expected LocalePtBR attached to context, got %v", val)
		}
	})

	t.Run("Custom locale registration", func(t *testing.T) {
		// test custom locale
		cl := &testCustomLocale{}
		z.RegisterLocale("fr", cl)

		if loc := z.ResolveLocale("fr-FR"); loc != cl {
			t.Errorf("expected custom registered locale cl, got %v", loc)
		}
	})
}

type testCustomLocale struct{}

func (t *testCustomLocale) Tag() string {
	return "fr"
}

func (t *testCustomLocale) FormatMessage(issue *z.Issue) string {
	return "Erreur personnalisée"
}

// @spec REQ-CORE-010
func TestREQ_CORE_010_Meta(t *testing.T) {
	t.Run("MetaBuilder chaining and merging", func(t *testing.T) {
		b := z.NewMetaBuilder(z.Meta{})
		b.ID("UserSchema").
			Title("User Model").
			Describe("Represents an authenticated user").
			Examples(map[string]any{"name": "Alice"}).
			Deprecated(false).
			Custom("version", 1)

		m1 := b.Build()
		if m1.ID != "UserSchema" || m1.Title != "User Model" || m1.Description != "Represents an authenticated user" {
			t.Errorf("unexpected m1: %+v", m1)
		}
		if len(m1.Examples) != 1 || m1.Custom["version"] != 1 {
			t.Errorf("unexpected examples or custom in m1: %+v", m1)
		}

		// Subsequent builder call starts from m1 and merges by default
		b2 := z.NewMetaBuilder(m1)
		b2.Describe("Updated description").
			Examples(map[string]any{"name": "Bob"})

		m2 := b2.Build()
		if m2.ID != "UserSchema" || m2.Title != "User Model" {
			t.Errorf("ID and Title should be preserved: %+v", m2)
		}
		if m2.Description != "Updated description" {
			t.Errorf("Description should be updated: %s", m2.Description)
		}
		if len(m2.Examples) != 2 {
			t.Errorf("Examples should merge: got %d examples", len(m2.Examples))
		}

		// Reset clears all
		b2.Reset()
		mReset := b2.Build()
		if mReset.ID != "" || mReset.Title != "" || len(mReset.Examples) != 0 {
			t.Errorf("Reset should clear all metadata: %+v", mReset)
		}
	})

	t.Run("Duplicate non-empty ID panics", func(t *testing.T) {
		owner1 := &struct{ name string }{"schema1"}
		owner2 := &struct{ name string }{"schema2"}

		z.RegisterSchemaID("UniqueUser", owner1)
		// Registering again for owner1 should not panic
		z.RegisterSchemaID("UniqueUser", owner1)

		// Registering for owner2 must panic
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("expected panic on duplicate schema ID, but did not panic")
			}
		}()

		z.RegisterSchemaID("UniqueUser", owner2)
	})
}

// @spec REQ-CORE-001
// @spec REQ-CORE-012
func TestREQ_CORE_001_012_EngineAndPool(t *testing.T) {
	t.Run("ParseCtx acquisition, issue addition, and clean release", func(t *testing.T) {
		ctx := context.Background()
		pc := z.AcquireParseCtx(ctx)

		z.ContextAddIssue(pc, z.Issue{
			Code:    z.InvalidType,
			Message: "test issue",
		})
		if z.ContextIssuesCount(pc) != 1 {
			t.Errorf("expected 1 issue in pc, got %d", z.ContextIssuesCount(pc))
		}

		z.ReleaseParseCtx(pc)

		// Re-acquire and verify cleanly reset
		pc2 := z.AcquireParseCtx(ctx)
		defer z.ReleaseParseCtx(pc2)
		if z.ContextIssuesCount(pc2) != 0 {
			t.Errorf("expected pool reset to have 0 issues, got %d", z.ContextIssuesCount(pc2))
		}
	})
}

// @spec REQ-CORE-002
// @spec REQ-CORE-003
func TestREQ_CORE_002_003_EntryPoints(t *testing.T) {
	// Simple test schema that validates non-empty string
	schema := z.NewTestSchema[string](func(pc any, in any) (any, bool) {
		s, ok := in.(string)
		if !ok {
			return nil, false
		}
		if len(s) == 0 {
			return nil, false
		}
		return s, true
	})

	t.Run("Parse with and without context", func(t *testing.T) {
		// No context (optional trailing variadic)
		v, err := schema.Parse("hello")
		if err != nil || v != "hello" {
			t.Errorf("expected success with 'hello', got (%q, %v)", v, err)
		}

		// Nil context (treated as absent)
		v, err = schema.Parse("world", nil)
		if err != nil || v != "world" {
			t.Errorf("expected success with nil context, got (%q, %v)", v, err)
		}

		// With valid context
		ctx := context.Background()
		v, err = schema.Parse("active", ctx)
		if err != nil || v != "active" {
			t.Errorf("expected success with active context, got (%q, %v)", v, err)
		}
	})

	t.Run("ParseJSON and Decode", func(t *testing.T) {
		data := []byte(`"json-value"`)
		v, err := schema.ParseJSON(data)
		if err != nil || v != "json-value" {
			t.Errorf("ParseJSON failed: got (%q, %v)", v, err)
		}

		reader := strings.NewReader(`"stream-value"`)
		v, err = schema.Decode(reader)
		if err != nil || v != "stream-value" {
			t.Errorf("Decode failed: got (%q, %v)", v, err)
		}
	})
}

// @spec REQ-CORE-011
// @spec REQ-CORE-016
func TestREQ_CORE_011_016_ContextConfiguration(t *testing.T) {
	defer z.Config.UseContext(z.ContextOptions{}) // restore default

	t.Run("WithAbortEarly helper", func(t *testing.T) {
		ctx := z.WithAbortEarly(context.Background())
		val := ctx.Value(z.AbortEarlyContextKey)
		if b, ok := val.(bool); !ok || !b {
			t.Errorf("expected true under AbortEarlyContextKey, got %v", val)
		}
	})

	t.Run("UseContext replaces keys and restores", func(t *testing.T) {
		customLocaleKey := "app.locale"
		customAbortKey := "app.abort"

		z.Config.UseContext(z.ContextOptions{
			LocaleKey:     customLocaleKey,
			AbortEarlyKey: customAbortKey,
		})

		ctx := context.WithValue(context.Background(), customLocaleKey, "pt-BR")
		// Resolving active locale should pick up pt-BR directly from the custom key
		ctxWithDerived := z.WithLocale(ctx, "pt-BR")
		val := ctxWithDerived.Value(customLocaleKey)
		if val != z.LocalePtBR {
			t.Errorf("expected LocalePtBR written to custom key %q, got %v", customLocaleKey, val)
		}

		// Restoring default
		z.Config.UseContext(z.ContextOptions{})
		defaultCtx := z.WithLocale(context.Background(), "pt-BR")
		if defaultCtx.Value(z.LocaleContextKey) != z.LocalePtBR {
			t.Errorf("expected default key restored")
		}
	})
}

// @spec REQ-CORE-017
func TestREQ_CORE_017_CancellationAndDeadlines(t *testing.T) {
	schema := z.NewTestSchema[string](func(pc any, in any) (any, bool) {
		return in, true
	})

	t.Run("Cancelled context returns context.Canceled directly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := schema.Parse("test", ctx)
		if err == nil {
			t.Fatalf("expected error on cancelled context, got nil")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected errors.Is(err, context.Canceled) to be true, got %v", err)
		}

		var zerr *z.ZodError
		if errors.As(err, &zerr) {
			t.Errorf("cancellation error must NOT be a *ZodError")
		}
	})

	t.Run("Deadline exceeded context returns context.DeadlineExceeded", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Second))
		defer cancel()

		_, err := schema.ParseJSON([]byte(`"test"`), ctx)
		if err == nil {
			t.Fatalf("expected error on expired deadline, got nil")
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected errors.Is(err, context.DeadlineExceeded) to be true, got %v", err)
		}
	})
}

// @spec REQ-CORE-015
func TestREQ_CORE_015_Concurrency_Race(t *testing.T) {
	// Schema that generates an error
	schema := z.NewTestSchema[string](func(pc any, in any) (any, bool) {
		return nil, false
	})

	var wg sync.WaitGroup
	localesToTest := []struct {
		tag      string
		expected string
	}{
		{"en", "Validation error"},
		{"pt-BR", "Erro de validação"},
		{"es", "Error de validación"},
	}

	for i := 0; i < 30; i++ {
		for _, item := range localesToTest {
			wg.Add(1)
			go func(tag, expected string) {
				defer wg.Done()
				ctx := z.WithLocale(context.Background(), tag)
				_, err := schema.Parse("input", ctx)
				if err == nil {
					t.Errorf("expected validation failure")
					return
				}
				var zerr *z.ZodError
				if !errors.As(err, &zerr) {
					t.Errorf("expected *ZodError")
					return
				}
			}(item.tag, item.expected)
		}
	}

	wg.Wait()
}
