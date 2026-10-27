package z_test

import (
	"errors"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-COLL-001
func TestREQ_COLL_001_Enum(t *testing.T) {
	t.Run("Variadic string enum", func(t *testing.T) {
		roleSchema := z.Enum("admin", "user", "guest")
		v, err := roleSchema.Parse("admin")
		if err != nil || v != "admin" {
			t.Errorf("expected 'admin', got (%q, %v)", v, err)
		}

		vals := roleSchema.Values()
		if len(vals) != 3 || vals[0] != "admin" {
			t.Errorf("unexpected Values(): %v", vals)
		}

		_, err = roleSchema.Parse("superadmin")
		if err == nil {
			t.Errorf("expected error for non-enum value")
		}
		var zerr *z.ZodError
		if errors.As(err, &zerr) {
			if len(zerr.Issues()) != 1 || zerr.Issues()[0].Code != z.InvalidValue {
				t.Errorf("expected InvalidValue issue, got %v", zerr.Issues())
			}
		}
	})

	t.Run("Typed string enum", func(t *testing.T) {
		type Status string
		const (
			Active   Status = "active"
			Inactive Status = "inactive"
		)

		statusSchema := z.Enum(Active, Inactive)
		v, err := statusSchema.Parse("active")
		if err != nil || v != Active {
			t.Errorf("expected Status(active), got (%v, %v)", v, err)
		}
	})

	t.Run("Extract and Exclude", func(t *testing.T) {
		roles := z.Enum("admin", "user", "guest", "bot")

		extracted := roles.Extract("admin", "user")
		if len(extracted.Values()) != 2 {
			t.Errorf("expected 2 values after Extract: %v", extracted.Values())
		}
		if _, err := extracted.Parse("guest"); err == nil {
			t.Errorf("expected failure parsing 'guest' on extracted schema")
		}

		excluded := roles.Exclude("bot")
		if len(excluded.Values()) != 3 {
			t.Errorf("expected 3 values after Exclude: %v", excluded.Values())
		}
		if _, err := excluded.Parse("bot"); err == nil {
			t.Errorf("expected failure parsing 'bot' on excluded schema")
		}
	})
}

// @spec REQ-COLL-002
func TestREQ_COLL_002_Array(t *testing.T) {
	t.Run("Array parsing and sizing checks", func(t *testing.T) {
		s := z.Array(z.String().Min(2)).Min(2).Max(4)

		v, err := s.Parse([]any{"apple", "banana"})
		if err != nil || len(v) != 2 || v[0] != "apple" {
			t.Errorf("expected ['apple', 'banana'], got (%v, %v)", v, err)
		}

		// Length check
		_, err = s.Parse([]any{"a"})
		if err == nil {
			t.Errorf("expected failure for count < Min(2) and element < Min(2)")
		}

		// Sizing check
		_, err = s.Parse([]any{"apple", "banana", "cherry", "date", "elderberry"})
		if err == nil {
			t.Errorf("expected failure for count 5 > Max(4)")
		}

		sExact := z.Array(z.Int()).Length(3)
		if _, err := sExact.Parse([]any{1, 2}); err == nil {
			t.Errorf("expected failure for count 2 != Length(3)")
		}
		if _, err := sExact.Parse([]any{1, 2, 3}); err != nil {
			t.Errorf("expected length 3 to pass: %v", err)
		}
	})

	t.Run("Element and Unwrap accessors", func(t *testing.T) {
		elem := z.String()
		arr := z.Array(elem)
		if arr.Element() != elem || arr.Unwrap() != elem {
			t.Errorf("expected Element and Unwrap to return original element schema")
		}
	})
}

// @spec REQ-COLL-003
func TestREQ_COLL_003_Tuple(t *testing.T) {
	t.Run("Fixed length tuple", func(t *testing.T) {
		tup := z.Tuple(z.String(), z.Int(), z.Boolean())

		v, err := tup.Parse([]any{"Alice", 30, true})
		if err != nil || len(v) != 3 {
			t.Fatalf("expected valid tuple: %v", err)
		}
		if v[0] != "Alice" || v[1] != 30 || v[2] != true {
			t.Errorf("unexpected tuple output: %v", v)
		}

		// Length mismatch
		_, err = tup.Parse([]any{"Alice", 30})
		if err == nil {
			t.Errorf("expected failure for tuple length 2 != 3")
		}

		_, err = tup.Parse([]any{"Alice", 30, true, "extra"})
		if err == nil {
			t.Errorf("expected failure for tuple length 4 != 3")
		}
	})

	t.Run("Tuple with Rest", func(t *testing.T) {
		tup := z.Tuple(z.String(), z.Int()).Rest(z.Boolean())

		v, err := tup.Parse([]any{"Alice", 30, true, false, true})
		if err != nil || len(v) != 5 {
			t.Fatalf("expected valid rest tuple: %v", err)
		}
		if v[2] != true || v[3] != false || v[4] != true {
			t.Errorf("unexpected rest elements: %v", v)
		}
	})
}

// @spec REQ-COLL-004
func TestREQ_COLL_004_Set(t *testing.T) {
	t.Run("Set collapses duplicates and enforces size", func(t *testing.T) {
		setSchema := z.Set(z.String()).Min(2).Max(3)

		// Input contains duplicates: "a", "b", "a" -> collapses to 2 unique items
		v, err := setSchema.Parse([]any{"a", "b", "a"})
		if err != nil {
			t.Fatalf("expected duplicates to collapse without error: %v", err)
		}
		if len(v) != 2 {
			t.Errorf("expected 2 unique items in set, got %d", len(v))
		}
		if _, ok := v["a"]; !ok {
			t.Errorf("expected 'a' in set")
		}
		if _, ok := v["b"]; !ok {
			t.Errorf("expected 'b' in set")
		}

		// Min check: 1 unique item < Min(2)
		_, err = setSchema.Parse([]any{"a", "a", "a"})
		if err == nil {
			t.Errorf("expected failure for 1 unique item < Min(2)")
		}
	})
}

// @spec REQ-COLL-005
func TestREQ_COLL_005_Map(t *testing.T) {
	t.Run("Map key and value validation", func(t *testing.T) {
		mapSchema := z.Map(z.String().Min(3), z.Int().Positive())

		raw := map[string]any{
			"key1": 10,
			"key2": 20,
		}
		v, err := mapSchema.Parse(raw)
		if err != nil || len(v) != 2 {
			t.Fatalf("expected valid map: %v", err)
		}
		if v["key1"] != 10 || v["key2"] != 20 {
			t.Errorf("unexpected map values: %v", v)
		}

		// Invalid key: "ab" length 2 < Min(3)
		badKey := map[string]any{
			"ab": 10,
		}
		_, err = mapSchema.Parse(badKey)
		if err == nil {
			t.Errorf("expected failure on invalid key length")
		}
		var zerr *z.ZodError
		if errors.As(err, &zerr) {
			foundKeyError := false
			for _, iss := range zerr.Issues() {
				if iss.Code == z.InvalidKey {
					foundKeyError = true
					break
				}
			}
			if !foundKeyError {
				t.Errorf("expected InvalidKey issue, got: %v", zerr.Issues())
			}
		}

		// Invalid value: negative int
		badVal := map[string]any{
			"validKey": -5,
		}
		_, err = mapSchema.Parse(badVal)
		if err == nil {
			t.Errorf("expected failure on negative value")
		}
	})
}

// @spec REQ-COLL-006
func TestREQ_COLL_006_Modifiers(t *testing.T) {
	t.Run("Modifiers on Enum", func(t *testing.T) {
		s := z.Enum("a", "b").Optional()
		v, err := s.Parse("a")
		if err != nil || !v.IsSome() || v.OrElse("") != "a" {
			t.Errorf("expected Some('a'): %v", err)
		}
	})

	t.Run("Modifiers on Array", func(t *testing.T) {
		s := z.Array(z.Int()).Default([]int{1, 2})
		v, err := s.Parse(nil)
		if err != nil || len(v) != 2 {
			t.Errorf("expected default array: %v", err)
		}
	})

	t.Run("Modifiers on Tuple", func(t *testing.T) {
		s := z.Tuple(z.String(), z.Int()).Nullable()
		v, err := s.Parse(nil)
		if err != nil || !v.IsNull() {
			t.Errorf("expected null tuple: %v", err)
		}
	})
}
