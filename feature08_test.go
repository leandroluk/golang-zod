package z_test

import (
	"strings"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-GQL-001 REQ-GQL-002 REQ-GQL-004 REQ-GQL-005
func TestREQ_GQL_001_002_004_005_GraphQL(t *testing.T) {
	t.Run("REQ-GQL-001 & REQ-GQL-002: ToGraphQLSDL with inverted nullability", func(t *testing.T) {
		roleEnum := z.Enum("ADMIN", "USER").Meta(func(m *z.MetaBuilder) {
			m.ID("Role")
		})

		user := z.Object(func(s *z.ObjectBuilder) {
			s.Field("id", z.String())
			s.Field("name", z.String())
			s.Field("role", roleEnum)
			s.Field("bio", z.String().Optional())
			s.Field("tags", z.Array(z.String()))
		}).Meta(func(m *z.MetaBuilder) {
			m.ID("User")
		})

		sdl, err := z.ToGraphQLSDL(user)
		if err != nil {
			t.Fatalf("failed to generate GraphQL SDL: %v", err)
		}

		// Check inverted nullability
		if !strings.Contains(sdl, "name: String!") {
			t.Fatalf("expected required field 'name: String!', got:\n%s", sdl)
		}
		if !strings.Contains(sdl, "bio: String") || strings.Contains(sdl, "bio: String!") {
			t.Fatalf("expected optional field 'bio: String' (without !), got:\n%s", sdl)
		}
		if !strings.Contains(sdl, "tags: [String!]!") {
			t.Fatalf("expected list field 'tags: [String!]!', got:\n%s", sdl)
		}
		if !strings.Contains(sdl, "enum Role") {
			t.Fatalf("expected 'enum Role', got:\n%s", sdl)
		}
	})

	t.Run("REQ-GQL-001: GraphQLInput generates input object", func(t *testing.T) {
		input := z.Object(func(s *z.ObjectBuilder) {
			s.Field("email", z.String())
		}).Meta(func(m *z.MetaBuilder) {
			m.ID("CreateUserInput")
		})

		sdl, err := z.ToGraphQLSDL(input, z.GraphQLInput())
		if err != nil {
			t.Fatalf("failed to generate input SDL: %v", err)
		}
		if !strings.Contains(sdl, "input CreateUserInput") {
			t.Fatalf("expected 'input CreateUserInput', got:\n%s", sdl)
		}
	})

	t.Run("REQ-GQL-004: Recursive schema terminates cleanly", func(t *testing.T) {
		var nodeSchema z.Schema[z.Dyn]
		nodeSchema = z.Lazy(func() z.Schema[z.Dyn] {
			return z.Object(func(s *z.ObjectBuilder) {
				s.Field("name", z.String())
				s.Field("children", z.Array(nodeSchema).Optional())
			}).Meta(func(m *z.MetaBuilder) {
				m.ID("TreeNode")
			})
		})

		sdl, err := z.ToGraphQLSDL(nodeSchema)
		if err != nil {
			t.Fatalf("unexpected error for recursive GraphQL schema: %v", err)
		}
		if !strings.Contains(sdl, "children: [TreeNode!]") {
			t.Fatalf("expected recursive field 'children: [TreeNode!]', got:\n%s", sdl)
		}
	})

	t.Run("REQ-GQL-005: Heterogeneous Tuple returns descriptive error", func(t *testing.T) {
		tuple := z.Tuple(z.String(), z.Int())
		_, err := z.ToGraphQLSDL(tuple)
		if err == nil {
			t.Fatal("expected error for tuple in GraphQL generation")
		}
		if !strings.Contains(err.Error(), "no GraphQL equivalent") {
			t.Fatalf("expected 'no GraphQL equivalent' error message, got: %v", err)
		}
	})
}
