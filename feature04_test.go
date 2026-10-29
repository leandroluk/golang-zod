package z_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

type TestUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Bio      string `json:"bio,omitempty"`
	IsActive bool   `json:"is_active"`
}

// @spec REQ-OBJ-001 REQ-OBJ-002 REQ-OBJ-003
func TestREQ_OBJ_001_002_003_Objects(t *testing.T) {
	t.Run("REQ-OBJ-001: Object strip by default", func(t *testing.T) {
		schema := z.Object(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
			s.Field("age", z.Int())
		})

		input := map[string]any{
			"name":  "Alice",
			"age":   30,
			"extra": "should be stripped",
		}

		res, err := schema.Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res["name"] != "Alice" || res["age"] != 30 {
			t.Fatalf("mismatched values: %v", res)
		}
		if _, exists := res["extra"]; exists {
			t.Fatalf("expected extra key to be stripped, got: %v", res)
		}
	})

	t.Run("REQ-OBJ-001: StrictObject rejects unknown keys", func(t *testing.T) {
		schema := z.StrictObject(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
		})

		_, err := schema.Parse(map[string]any{
			"name":  "Bob",
			"extra": 123,
		})
		if err == nil {
			t.Fatal("expected error on extra key in strict mode")
		}
		zerr, ok := err.(*z.ZodError)
		if !ok {
			t.Fatalf("expected *ZodError, got %T", err)
		}
		issues := zerr.Issues()
		if len(issues) == 0 || issues[0].Code != z.UnrecognizedKeys {
			t.Fatalf("expected UnrecognizedKeys issue, got %v", issues)
		}
	})

	t.Run("REQ-OBJ-001: LooseObject preserves unknown keys", func(t *testing.T) {
		schema := z.LooseObject(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
		})

		res, err := schema.Parse(map[string]any{
			"name":  "Charlie",
			"extra": 42,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res["extra"] != 42 {
			t.Fatalf("expected extra key to be preserved, got: %v", res)
		}
	})

	t.Run("REQ-OBJ-002: Extend, Pick, Omit, Partial, Required, Keyof", func(t *testing.T) {
		base := z.Object(func(s *z.ObjectBuilder) {
			s.Field("a", z.String())
			s.Field("b", z.Int())
		})

		extended := base.Extend(func(s *z.ObjectBuilder) {
			s.Field("c", z.Boolean())
		})

		res, err := extended.Parse(map[string]any{
			"a": "hello",
			"b": 10,
			"c": true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res["c"] != true {
			t.Fatalf("expected c=true, got %v", res)
		}

		picked := base.Pick("a")
		resPick, err := picked.Parse(map[string]any{"a": "hello", "b": 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, hasB := resPick["b"]; hasB {
			t.Fatal("expected b to be omitted in Pick")
		}

		omitted := base.Omit("b")
		resOmit, err := omitted.Parse(map[string]any{"a": "hello", "b": 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, hasB := resOmit["b"]; hasB {
			t.Fatal("expected b to be omitted in Omit")
		}

		partial := base.Partial()
		resPart, err := partial.Parse(map[string]any{"a": "hello"})
		if err != nil {
			t.Fatalf("unexpected error on partial: %v", err)
		}
		if resPart["a"] != z.Some[any]("hello") {
			t.Fatalf("expected a=Some[any](hello), got %v", resPart["a"])
		}

		required := partial.Required()
		_, err = required.Parse(map[string]any{"a": "hello"})
		if err == nil {
			t.Fatal("expected error on required with missing field 'b'")
		}

		keyof := base.Keyof()
		_, err = keyof.Parse("a")
		if err != nil {
			t.Fatalf("expected 'a' to be valid keyof: %v", err)
		}
		_, err = keyof.Parse("nonexistent")
		if err == nil {
			t.Fatal("expected error for invalid keyof")
		}
	})

	t.Run("REQ-OBJ-003: Catchall schema", func(t *testing.T) {
		schema := z.Object(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
		}).Catchall(z.Int())

		res, err := schema.Parse(map[string]any{
			"name":  "Dave",
			"score": 100,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res["score"] != 100 {
			t.Fatalf("expected score=100, got %v", res)
		}

		_, err = schema.Parse(map[string]any{
			"name":  "Dave",
			"score": "not-an-int",
		})
		if err == nil {
			t.Fatal("expected catchall validation error for string score")
		}
	})
}

// @spec REQ-OBJ-004 REQ-OBJ-006 REQ-OBJ-007
func TestREQ_OBJ_004_006_007_Struct(t *testing.T) {
	t.Run("REQ-OBJ-004 & REQ-OBJ-006: Struct validation with unsafe zero-reflect runtime", func(t *testing.T) {
		userSchema := z.Struct(func(s *z.StructBuilder, u *TestUser) {
			s.Field(&u.ID, z.Int())
			s.Field(&u.Name, z.String())
			s.Field(&u.Email, z.Email())
			s.Field(&u.Bio, z.String().Default("no bio"))
			s.Field(&u.IsActive, z.Boolean())
		})

		input := map[string]any{
			"id":        42,
			"name":      "Leandro",
			"email":     "leandro@example.com",
			"is_active": true,
		}

		user, err := userSchema.Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != 42 || user.Name != "Leandro" || user.Email != "leandro@example.com" || !user.IsActive {
			t.Fatalf("unexpected struct values: %+v", user)
		}
		if user.Bio != "no bio" {
			t.Fatalf("expected default bio 'no bio', got %q", user.Bio)
		}
	})

	t.Run("REQ-OBJ-007: Struct panics on invalid configuration", func(t *testing.T) {
		// Non-struct type
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic on non-struct T")
				}
			}()
			z.Struct(func(s *z.StructBuilder, i *int) {})
		}()

		// Field pointer nil
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic on nil field pointer")
				}
			}()
			z.Struct(func(s *z.StructBuilder, u *TestUser) {
				s.Field(nil, z.Int())
			})
		}()
	})
}

// @spec REQ-OBJ-008
func TestREQ_OBJ_008_Record(t *testing.T) {
	t.Run("REQ-OBJ-008: Record schema validation", func(t *testing.T) {
		rec := z.Record(z.String(), z.Int())

		input := map[string]any{
			"one": 1,
			"two": 2,
		}

		res, err := rec.Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res["one"] != 1 || res["two"] != 2 {
			t.Fatalf("unexpected record: %v", res)
		}

		// Invalid value
		_, err = rec.Parse(map[string]any{
			"one": "not-an-int",
		})
		if err == nil {
			t.Fatal("expected error for non-int record value")
		}
	})

	t.Run("REQ-OBJ-008: Record with Enum keys requires exhaustiveness in default mode", func(t *testing.T) {
		colorEnum := z.Enum("red", "green", "blue")
		rec := z.Record(colorEnum, z.Int())

		// Missing blue
		_, err := rec.Parse(map[string]any{
			"red":   1,
			"green": 2,
		})
		if err == nil {
			t.Fatal("expected error on non-exhaustive enum record")
		}

		// PartialRecord allows partial keys
		partialRec := z.PartialRecord(colorEnum, z.Int())
		res, err := partialRec.Parse(map[string]any{
			"red":   1,
			"green": 2,
		})
		if err != nil {
			t.Fatalf("unexpected error on partial record: %v", err)
		}
		if res["red"] != 1 || res["green"] != 2 {
			t.Fatalf("unexpected partial record: %v", res)
		}
	})
}

// @spec REQ-OBJ-009
func TestREQ_OBJ_009_ArchitecturalInvariants(t *testing.T) {
	t.Run("REQ-OBJ-009: Reflect imported ONLY in struct.go, Unsafe imported ONLY in struct_unsafe.go", func(t *testing.T) {
		fset := token.NewFileSet()
		files, err := os.ReadDir(".")
		if err != nil {
			t.Fatalf("failed to read dir: %v", err)
		}

		for _, f := range files {
			name := f.Name()
			if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}

			path := filepath.Join(".", name)
			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", name, err)
			}

			for _, imp := range node.Imports {
				pkgPath := strings.Trim(imp.Path.Value, `"`)
				if pkgPath == "reflect" && name != "struct.go" {
					t.Fatalf("INVARIANT VIOLATION: reflect imported in %s (ONLY allowed in struct.go)", name)
				}
				if pkgPath == "unsafe" && name != "struct_unsafe.go" {
					t.Fatalf("INVARIANT VIOLATION: unsafe imported in %s (ONLY allowed in struct_unsafe.go)", name)
				}
			}
		}
	})
}
