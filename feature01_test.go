package z_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

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

