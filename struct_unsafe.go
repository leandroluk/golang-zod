package z

import "unsafe"

// @spec REQ-OBJ-005
// structFieldBinding holds a field's JSON name, schema, and an unsafe setter.
type structFieldBinding struct {
	jsonName string
	// key indexes the parsed value map: the json name, or a synthetic key for inline fields.
	key    string
	inline bool
	schema AnySchema
	setter func(base unsafe.Pointer, val any)
}

// @spec REQ-OBJ-005
func makeFieldBinding[F any](jsonName string, offset uintptr, sch AnySchema) structFieldBinding {
	setter := func(base unsafe.Pointer, val any) {
		fieldPtr := (*F)(unsafe.Add(base, offset))
		if typed, ok := val.(F); ok {
			*fieldPtr = typed
		}
	}
	return structFieldBinding{
		jsonName: jsonName,
		key:      jsonName,
		schema:   sch,
		setter:   setter,
	}
}

// @spec REQ-OBJ-005
// allocateAndPopulate creates a new T and writes validated field values using precomputed offsets.
func allocateAndPopulate[T any](bindings []structFieldBinding, values map[string]any) T {
	var target T
	base := unsafe.Pointer(&target)
	for _, b := range bindings {
		if val, exists := values[b.key]; exists {
			b.setter(base, val)
		}
	}
	return target
}
