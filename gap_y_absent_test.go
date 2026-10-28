package z_test

import (
	"reflect"
	"testing"

	z "github.com/leandroluk/golang-zod"
)

func gapYAbsentObject() *z.ObjectSchema {
	return z.Object(func(s *z.Shape) {
		s.Field("d", z.String().Default("dv"))
		s.Field("p", z.NewPrefaultSchema[string](z.String().Min(2), "pv"))
		s.Field("c", z.NewCatchSchema[string](z.String(), "cv"))
		s.Field("req", z.String())
	})
}

// @spec REQ-PIPE-001
func TestGapYObjectAbsentDefaultPrefaultCatch(t *testing.T) {
	got, err := gapYAbsentObject().Parse(map[string]any{"req": "r"})
	want := z.Dyn{"d": "dv", "p": "pv", "c": "cv", "req": "r"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
}

// @spec REQ-PIPE-001
func TestGapYObjectPresentValues(t *testing.T) {
	got, err := gapYAbsentObject().Parse(map[string]any{"req": "r", "p": "given", "c": 5})
	want := z.Dyn{"d": "dv", "p": "given", "c": "cv", "req": "r"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
	_, err = gapYAbsentObject().Parse(map[string]any{"req": "r", "p": "x"})
	zerr, ok := err.(*z.ZodError)
	if !ok || len(zerr.Issues()) != 1 || zerr.Issues()[0].Code != z.TooSmall || zerr.Issues()[0].Path[0] != z.Property("p") {
		t.Fatalf("err = %v", err)
	}
}

// @spec REQ-PIPE-001
func TestGapYObjectPrefaultInnerFailure(t *testing.T) {
	obj := z.Object(func(s *z.Shape) {
		s.Field("p", z.NewPrefaultSchema[string](z.String().Min(5), "ab"))
	})
	_, err := obj.Parse(map[string]any{})
	zerr, ok := err.(*z.ZodError)
	if !ok || len(zerr.Issues()) != 1 || zerr.Issues()[0].Code != z.TooSmall {
		t.Fatalf("prefault value must run through the inner schema, err = %v", err)
	}
}

// @spec REQ-PIPE-001
func TestGapYCatchAbsentNullableInner(t *testing.T) {
	obj := z.Object(func(s *z.Shape) {
		s.Field("c", z.NewCatchSchema[z.Nullable[string]](z.String().Nullable(), z.NullableOf("cv")))
	})
	got, err := obj.Parse(map[string]any{})
	if err != nil || !reflect.DeepEqual(got, z.Dyn{"c": z.NullableOf("cv")}) {
		t.Fatalf("got %v, %v", got, err)
	}
}

type gapYStructT struct {
	D string `json:"d"`
	P string `json:"p"`
	C string `json:"c"`
	R string `json:"r"`
}

func gapYStruct() *z.StructSchema[gapYStructT] {
	return z.Struct[gapYStructT](func(s *z.StructShape, v *gapYStructT) {
		s.Field(&v.D, z.String().Default("dv"))
		s.Field(&v.P, z.NewPrefaultSchema[string](z.String().Min(2), "pv"))
		s.Field(&v.C, z.NewCatchSchema[string](z.String(), "cv"))
		s.Field(&v.R, z.String())
	})
}

// @spec REQ-PIPE-001
func TestGapYStructAbsentDefaultPrefaultCatch(t *testing.T) {
	got, err := gapYStruct().Parse(map[string]any{"r": "r"})
	if err != nil || got != (gapYStructT{D: "dv", P: "pv", C: "cv", R: "r"}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	got, err = gapYStruct().Parse(map[string]any{"r": "r", "p": "given", "c": 1, "d": "x"})
	if err != nil || got != (gapYStructT{D: "x", P: "given", C: "cv", R: "r"}) {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err = gapYStruct().Parse(map[string]any{"r": "r", "p": "x"}); err == nil {
		t.Fatal("expected issue for present invalid prefault field")
	}
}
