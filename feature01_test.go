package z_test

import (
	"encoding/json/v2"
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
