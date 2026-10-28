package z_test

import (
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-EXP-001 REQ-EXP-002 REQ-EXP-003 REQ-EXP-004 REQ-EXP-005 REQ-EXP-006
func TestREQ_EXP_001_002_003_004_005_006_JSONSchema(t *testing.T) {
	t.Run("REQ-EXP-001 & REQ-EXP-002: ToJSONSchema Draft 2020-12 and OpenAPI 3.0", func(t *testing.T) {
		user := z.Object(func(s *z.Shape) {
			s.Field("name", z.String())
			s.Field("age", z.Int())
			s.Field("bio", z.String().Optional())
		})

		// Draft 2020-12
		dataDraft, err := z.ToJSONSchema(user, z.WithTarget(z.Draft202012))
		if err != nil {
			t.Fatalf("failed to generate JSON schema Draft202012: %v", err)
		}
		draftStr := string(dataDraft)
		if !strings.Contains(draftStr, "https://json-schema.org/draft/2020-12/schema") {
			t.Fatalf("expected 2020-12 schema url, got: %s", draftStr)
		}
		if !strings.Contains(draftStr, `"properties"`) || !strings.Contains(draftStr, `"required"`) {
			t.Fatalf("missing properties or required: %s", draftStr)
		}

		// OpenAPI 3.0
		dataOpenAPI, err := z.ToJSONSchema(user, z.WithTarget(z.OpenAPI30))
		if err != nil {
			t.Fatalf("failed to generate OpenAPI 3.0 schema: %v", err)
		}
		openAPIStr := string(dataOpenAPI)
		if strings.Contains(openAPIStr, "$schema") {
			t.Fatalf("OpenAPI schema should not contain $schema: %s", openAPIStr)
		}
	})

	t.Run("REQ-EXP-003: Cycle detection and $defs sharing", func(t *testing.T) {
		var catSchema z.Schema[z.Dyn]
		catSchema = z.Lazy(func() z.Schema[z.Dyn] {
			return z.Object(func(s *z.Shape) {
				s.Field("name", z.String())
				s.Field("subs", z.Array(catSchema).Optional())
			}).Meta(func(m *z.MetaBuilder) {
				m.ID("Category")
			})
		})

		data, err := z.ToJSONSchema(catSchema, z.WithCycles(z.CyclesRef))
		if err != nil {
			t.Fatalf("failed to generate schema for recursive tree: %v", err)
		}
		dataStr := string(data)
		if !strings.Contains(dataStr, `"$ref":"#/$defs/Category"`) {
			t.Fatalf("expected $ref to Category, got %s", dataStr)
		}
	})

	t.Run("REQ-EXP-004: Unrepresentable throws by default", func(t *testing.T) {
		dateSchema := z.Date()
		_, err := z.ToJSONSchema(dateSchema)
		if err == nil {
			t.Fatal("expected error for unrepresentable Date schema")
		}

		// UnrepresentableAny allows emitting {}
		data, err := z.ToJSONSchema(dateSchema, z.WithUnrepresentable(z.UnrepresentableAny))
		if err != nil {
			t.Fatalf("unexpected error with UnrepresentableAny: %v", err)
		}
		if string(data) != "{}" {
			t.Fatalf("expected '{}', got: %s", string(data))
		}
	})

	t.Run("REQ-EXP-005 & REQ-EXP-006: FromJSONSchema Round-trip", func(t *testing.T) {
		original := z.Object(func(s *z.Shape) {
			s.Field("title", z.String())
			s.Field("count", z.Int())
			s.Field("active", z.Boolean())
		})

		schemaBytes, err := z.ToJSONSchema(original)
		if err != nil {
			t.Fatalf("ToJSONSchema failed: %v", err)
		}

		imported, err := z.FromJSONSchema(schemaBytes)
		if err != nil {
			t.Fatalf("FromJSONSchema failed: %v", err)
		}

		validPayload := `{"title": "Book", "count": 5, "active": true}`
		_, err = imported.ParseJSON([]byte(validPayload))
		if err != nil {
			t.Fatalf("failed to parse valid payload through imported schema: %v", err)
		}

		invalidPayload := `{"title": "Book", "count": "not-a-number", "active": true}`
		_, err = imported.ParseJSON([]byte(invalidPayload))
		if err == nil {
			t.Fatal("expected validation error on invalid imported schema payload")
		}
	})
}
