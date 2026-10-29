package z_test

import (
	"testing"

	z "github.com/leandroluk/golang-zod"
)

// @spec REQ-COMB-001 REQ-COMB-002 REQ-COMB-003
func TestREQ_COMB_001_002_003_Union(t *testing.T) {
	t.Run("REQ-COMB-001: Union first-matching candidate wins", func(t *testing.T) {
		u := z.Union(z.String(), z.Int())

		resStr, err := u.Parse("hello")
		if err != nil {
			t.Fatalf("unexpected error on string: %v", err)
		}
		if resStr != "hello" {
			t.Fatalf("expected 'hello', got %v", resStr)
		}

		resInt, err := u.Parse(123)
		if err != nil {
			t.Fatalf("unexpected error on int: %v", err)
		}
		if resInt != 123 {
			t.Fatalf("expected 123, got %v", resInt)
		}

		_, err = u.Parse(true)
		if err == nil {
			t.Fatal("expected error on boolean for Union(String, Int)")
		}
	})

	t.Run("REQ-COMB-003: XOR requires exactly one candidate to match", func(t *testing.T) {
		// Both match positive numbers vs int
		xor := z.XOR(z.Int().GTE(0), z.Int().LTE(10))

		// 15 matches only GTE(0) -> succeeds
		res, err := xor.Parse(15)
		if err != nil {
			t.Fatalf("unexpected error for 15: %v", err)
		}
		if res != 15 {
			t.Fatalf("expected 15, got %v", res)
		}

		// -5 matches only LTE(10) -> succeeds
		res, err = xor.Parse(-5)
		if err != nil {
			t.Fatalf("unexpected error for -5: %v", err)
		}
		if res != -5 {
			t.Fatalf("expected -5, got %v", res)
		}

		// 5 matches both -> must fail XOR
		_, err = xor.Parse(5)
		if err == nil {
			t.Fatal("expected XOR error when both branches match (input=5)")
		}
	})

	t.Run("REQ-COMB-002: DiscriminatedUnion O(1) dispatch", func(t *testing.T) {
		circle := z.Object(func(s *z.ObjectBuilder) {
			s.Field("kind", z.Literal("circle"))
			s.Field("radius", z.Float64())
		})
		square := z.Object(func(s *z.ObjectBuilder) {
			s.Field("kind", z.Literal("square"))
			s.Field("side", z.Float64())
		})

		du := z.DiscriminatedUnion("kind", circle, square)

		cRes, err := du.Parse(map[string]any{
			"kind":   "circle",
			"radius": 5.5,
		})
		if err != nil {
			t.Fatalf("unexpected error on circle: %v", err)
		}
		cMap := cRes.(z.Dyn)
		if cMap["kind"] != "circle" || cMap["radius"] != 5.5 {
			t.Fatalf("unexpected circle result: %v", cMap)
		}

		// Unknown discriminator value
		_, err = du.Parse(map[string]any{
			"kind": "triangle",
		})
		if err == nil {
			t.Fatal("expected error on unknown discriminator value 'triangle'")
		}
	})
}

// @spec REQ-COMB-004 REQ-COMB-005
func TestREQ_COMB_004_005_Intersection(t *testing.T) {
	t.Run("REQ-COMB-004: Intersection merges valid outputs", func(t *testing.T) {
		part1 := z.Object(func(s *z.ObjectBuilder) {
			s.Field("id", z.Int())
		})
		part2 := z.Object(func(s *z.ObjectBuilder) {
			s.Field("name", z.String())
		})

		inter := z.Intersection(part1, part2)

		res, err := inter.Parse(map[string]any{
			"id":   1,
			"name": "Widget",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resMap, ok := res.(z.Dyn)
		if !ok {
			t.Fatalf("expected z.Dyn, got %T", res)
		}
		if resMap["id"] != 1 || resMap["name"] != "Widget" {
			t.Fatalf("unexpected merged map: %v", resMap)
		}
	})

	t.Run("REQ-COMB-005: Intersection conflicts return error", func(t *testing.T) {
		part1 := z.Object(func(s *z.ObjectBuilder) {
			s.Field("version", z.Literal(1))
		})
		part2 := z.Object(func(s *z.ObjectBuilder) {
			s.Field("version", z.Literal(2))
		})

		inter := z.Intersection(part1, part2)

		_, err := inter.Parse(map[string]any{
			"version": 1,
		})
		if err == nil {
			t.Fatal("expected error on conflicting intersection")
		}
	})
}

// @spec REQ-COMB-007
func TestREQ_COMB_007_Lazy(t *testing.T) {
	t.Run("REQ-COMB-007: Lazy recursive schema resolves and memoizes correctly", func(t *testing.T) {
		var categorySchema z.Schema[z.Dyn]
		categorySchema = z.Lazy(func() z.Schema[z.Dyn] {
			return z.Object(func(s *z.ObjectBuilder) {
				s.Field("name", z.String())
				s.Field("subcategories", z.Array(categorySchema))
			})
		})

		tree := map[string]any{
			"name": "Electronics",
			"subcategories": []any{
				map[string]any{
					"name": "Computers",
					"subcategories": []any{
						map[string]any{
							"name":          "Laptops",
							"subcategories": []any{},
						},
					},
				},
				map[string]any{
					"name":          "Cameras",
					"subcategories": []any{},
				},
			},
		}

		res, err := categorySchema.Parse(tree)
		if err != nil {
			t.Fatalf("unexpected error parsing recursive tree: %v", err)
		}
		if res["name"] != "Electronics" {
			t.Fatalf("expected top name Electronics, got %v", res["name"])
		}
		subs := res["subcategories"].([]z.Dyn)
		if len(subs) != 2 {
			t.Fatalf("expected 2 subcategories, got %d", len(subs))
		}
	})
}
