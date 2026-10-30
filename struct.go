package z

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type structSlot struct {
	fieldName   string
	jsonName    string
	schema      AnySchema
	makeBinding func(sch AnySchema) structFieldBinding
	fieldType   reflect.Type
	inline      bool
}

// @spec REQ-OBJ-004
// StructBuilder is the builder passed to z.Struct to bind struct field pointers to schemas.
type StructBuilder struct {
	basePtr uintptr
	tType   reflect.Type
	slots   []*structSlot
}

// @spec REQ-OBJ-004
// Field binds a struct field pointer. Panics on invalid pointers, duplicate JSON names, or unexported fields.
// When called with 1 argument, returns an *StructFieldBuilder with type aliases (.String(), .Int(), etc.).
// When called with 2 arguments, registers the schema directly; its output must be F, or T for a *T field (nil allowed).
func (s *StructBuilder) Field[F any](p *F, sch ...AnySchema) *StructFieldBuilder {
	if p == nil {
		panic("z.Struct: field pointer cannot be nil")
	}

	pVal := reflect.ValueOf(p)
	pPtr := pVal.Pointer()

	if pPtr < s.basePtr || pPtr >= s.basePtr+s.tType.Size() {
		panic("z.Struct: pointer does not point to a field of struct T")
	}

	off := pPtr - s.basePtr

	// Locate top-level field with matching offset
	var matchedField *reflect.StructField
	for i := 0; i < s.tType.NumField(); i++ {
		f := s.tType.Field(i)
		if f.Offset == off {
			matchedField = &f
			break
		}
	}

	if matchedField == nil {
		panic("z.Struct: embedded or nested field paths cannot be registered directly; use a nested z.Struct")
	}

	if !matchedField.IsExported() {
		panic(fmt.Sprintf("z.Struct: field %q must be exported", matchedField.Name))
	}

	expectedType := reflect.TypeOf((*F)(nil)).Elem()
	if matchedField.Type != expectedType {
		panic(fmt.Sprintf("z.Struct: field %q type %v does not match schema type %v", matchedField.Name, matchedField.Type, expectedType))
	}

	jsonTag := matchedField.Tag.Get("json")
	if matchedField.Anonymous && matchedField.Type.Kind() == reflect.Struct && jsonTag == "" {
		// Embedded struct without a json tag: flattened into the parent, so it needs its own schema.
		if len(sch) == 0 || sch[0] == nil {
			panic(fmt.Sprintf("z.Struct: embedded field %q needs a schema: s.Field(&t.%s, schema)", matchedField.Name, matchedField.Name))
		}
		name := matchedField.Name
		slot := &structSlot{
			fieldName: name,
			fieldType: expectedType,
			inline:    true,
			makeBinding: func(sch AnySchema) structFieldBinding {
				b := makeFieldBinding[F]("", off, sch)
				b.key = "\x00" + name
				b.inline = true
				return b
			},
		}
		s.slots = append(s.slots, slot)
		b := &StructFieldBuilder{shape: s, slot: slot}
		b.Schema(sch[0])
		return b
	}
	if jsonTag == "" || jsonTag == "-" {
		panic(fmt.Sprintf("z.Struct: field %q must have a mandatory `json:\"name\"` tag", matchedField.Name))
	}

	jsonName := strings.TrimSpace(strings.Split(jsonTag, ",")[0])
	if jsonName == "" || jsonName == "-" {
		panic(fmt.Sprintf("z.Struct: field %q must specify a valid json name in `json:\"name\"`", matchedField.Name))
	}

	for _, existing := range s.slots {
		if existing.jsonName == jsonName {
			panic(fmt.Sprintf("z.Struct: duplicate json field name %q", jsonName))
		}
	}

	slot := &structSlot{
		fieldName:   matchedField.Name,
		jsonName:    jsonName,
		fieldType:   expectedType,
		makeBinding: func(sch AnySchema) structFieldBinding {
			return makeFieldBinding[F](jsonName, off, sch)
		},
	}
	s.slots = append(s.slots, slot)

	b := &StructFieldBuilder{
		shape: s,
		slot:  slot,
	}
	if len(sch) > 0 && sch[0] != nil {
		b.Schema(sch[0])
	}
	return b
}

// StructFieldBuilder provides fluent methods to declare schemas on a struct field.
type StructFieldBuilder struct {
	shape *StructBuilder
	slot  *structSlot
}

func (b *StructFieldBuilder) bind(sch AnySchema) {
	b.slot.schema = sch
	if s, ok := any(sch).(interface{ setSlot(*AnySchema) }); ok {
		s.setSlot(&b.slot.schema)
	}
}

func (b *StructFieldBuilder) Schema(sch AnySchema) AnySchema {
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) String() *StringSchema {
	sch := String()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Int() *NumberSchema[int] {
	sch := Int()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Int8() *NumberSchema[int8] {
	sch := Int8()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Int16() *NumberSchema[int16] {
	sch := Int16()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Int32() *NumberSchema[int32] {
	sch := Int32()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Int64() *NumberSchema[int64] {
	sch := Int64()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Uint() *NumberSchema[uint] {
	sch := Uint()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Uint8() *NumberSchema[uint8] {
	sch := Uint8()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Uint16() *NumberSchema[uint16] {
	sch := Uint16()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Uint32() *NumberSchema[uint32] {
	sch := Uint32()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Uint64() *NumberSchema[uint64] {
	sch := Uint64()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Float32() *NumberSchema[float32] {
	sch := Float32()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Float64() *NumberSchema[float64] {
	sch := Float64()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) BigInt() *BigIntSchema {
	sch := BigInt()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Boolean() *BooleanSchema {
	sch := Boolean()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Bool() *BooleanSchema {
	return b.Boolean()
}

func (b *StructFieldBuilder) Date() *DateSchema {
	sch := Date()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Bytes() *BytesSchema {
	sch := Bytes()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Email() *StringSchema {
	sch := Email()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) UUID() *StringSchema {
	sch := UUID()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) URL() *StringSchema {
	sch := URL()
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Enum(values ...string) *EnumSchema[string] {
	sch := Enum(values...)
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Literal(val any) AnySchema {
	sch := Literal(val)
	b.bind(sch)
	return sch
}

func (b *StructFieldBuilder) Any() *AnyPrimitiveSchema {
	sch := Any()
	b.bind(sch)
	return sch
}

// @spec REQ-OBJ-004
// StructSchema validates JSON objects and constructs typed struct T instances without reflect at parse time.
type StructSchema[T any] struct {
	BaseSchema[T]
	fields   []structFieldBinding
	isStrict bool
}

type structNode[T any] struct {
	schema *StructSchema[T]
}

func (n *structNode[T]) kind() nodeKind                    { return kindStruct }
func (n *structNode[T]) FieldNames() []string              { return n.schema.FieldNames() }
func (n *structNode[T]) FieldSchema(name string) AnySchema { return n.schema.FieldSchema(name) }
func (n *structNode[T]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "object",
			Input:    in,
		})
		var zero T
		return zero, false
	}

	rawMap, ok := in.(map[string]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "object",
			Input:    in,
		})
		var zero T
		return zero, false
	}

	fieldValues := make(map[string]any, len(n.schema.fields))
	seenKeys := make(map[string]struct{}, len(n.schema.fields))
	allOk := true

	for _, f := range n.schema.fields {
		if f.inline {
			if err := ctx.checkCancel(); err != nil {
				var zero T
				return zero, false
			}
			for _, name := range inlineNames(f.schema) {
				seenKeys[name] = struct{}{}
			}
			out, ok := f.schema.node().parse(ctx, rawMap)
			if !ok {
				allOk = false
				if ctx.shouldAbort() {
					break
				}
				continue
			}
			fieldValues[f.key] = out
			continue
		}
		seenKeys[f.jsonName] = struct{}{}
		rawVal, present := rawMap[f.jsonName]

		if err := ctx.checkCancel(); err != nil {
			var zero T
			return zero, false
		}

		k := f.schema.node().kind()

		if !present {
			if k == kindOptional || k == kindNullish {
				// Absent optional struct fields remain their zero Option value
				continue
			}
			if k == kindDefault || k == kindPrefault || k == kindCatch {
				out, ok := parseAbsent(ctx, f.schema.node())
				if ok {
					fieldValues[f.key] = out
				}
				continue
			}

			// Required key missing
			ctx.pushPath(Property(f.jsonName))
			ctx.addIssue(Issue{
				Code:     InvalidType,
				Expected: "required",
				Input:    nil,
			})
			ctx.popPath()
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		// Field is present
		ctx.pushPath(Property(f.jsonName))
		out, ok := f.schema.node().parse(ctx, rawVal)
		ctx.popPath()

		if !ok {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		fieldValues[f.key] = out
	}

	if !allOk && ctx.shouldAbort() {
		var zero T
		return zero, false
	}

	// Unknown key handling
	if n.schema.isStrict {
		var unrecognized []string
		for k := range rawMap {
			if _, known := seenKeys[k]; !known {
				unrecognized = append(unrecognized, k)
			}
		}
		if len(unrecognized) > 0 {
			slices.Sort(unrecognized)
			ctx.addIssue(Issue{
				Code: UnrecognizedKeys,
				Keys: unrecognized,
			})
			allOk = false
		}
	}

	if !allOk {
		var zero T
		return zero, false
	}

	// Write fields via precompiled unsafe setters
	target := allocateAndPopulate[T](n.schema.fields, fieldValues)
	return target, true
}

// @spec REQ-OBJ-004
// Struct constructs a schema targeting a concrete Go struct type T.
func Struct[T any](build func(s *StructBuilder, t *T)) *StructSchema[T] {
	tType := reflect.TypeOf((*T)(nil)).Elem()
	if tType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("z.Struct[T]: T must be a struct, got %s", tType.Kind()))
	}

	var sample T
	sampleVal := reflect.ValueOf(&sample)
	basePtr := sampleVal.Pointer()

	shape := &StructBuilder{
		basePtr: basePtr,
		tType:   tType,
	}

	build(shape, &sample)

	fields := make([]structFieldBinding, 0, len(shape.slots))
	for _, slot := range shape.slots {
		if slot.schema == nil {
			panic(fmt.Sprintf("z.Struct: field %q registered without schema", slot.fieldName))
		}
		if out := schemaOutputType(slot.schema); out != nil && out != slot.fieldType && slot.fieldType.Kind() == reflect.Pointer && slot.fieldType.Elem() == out {
			// A *T field with a T schema: nil is allowed (absent or null), as in Go pointer semantics.
			slot.schema = &autoPtrSchema{inner: slot.schema, ptrType: slot.fieldType}
		} else if out != nil && out != slot.fieldType {
			panic(fmt.Sprintf("z.Struct: field %q has type %v but its schema produces %v (use .Ptr() for pointer fields, or declare the field as that type)", slot.fieldName, slot.fieldType, out))
		}
		fields = append(fields, slot.makeBinding(slot.schema))
	}

	s := &StructSchema[T]{
		fields:   fields,
		isStrict: false,
	}
	s.BaseSchema = NewBaseSchema[T](&structNode[T]{schema: s})
	if name := tType.Name(); name != "" {
		s.meta.ID = name
		s.meta.isDefaultID = true
	}
	return s
}

func (s *StructSchema[T]) clone() *StructSchema[T] {
	c := &StructSchema[T]{
		fields:   slices.Clone(s.fields),
		isStrict: s.isStrict,
	}
	c.BaseSchema = NewBaseSchema[T](&structNode[T]{schema: c})
	c.setMeta(s.getMeta())
	c.slot = s.slot
	c.updateSlot(c)
	return c
}

// @spec REQ-OBJ-006
// Strict configures the struct schema to reject unrecognized JSON keys.
func (s *StructSchema[T]) Strict() *StructSchema[T] {
	c := s.clone()
	c.isStrict = true
	return c
}

// @spec REQ-OBJ-006
// Strip restores default unknown key stripping for the struct schema.
func (s *StructSchema[T]) Strip() *StructSchema[T] {
	c := s.clone()
	c.isStrict = false
	return c
}

// @spec REQ-OBJ-006
// Pick retains only specified field keys. Panics if an unknown key is specified.
func (s *StructSchema[T]) Pick(keys ...string) *StructSchema[T] {
	c := s.clone()
	var picked []structFieldBinding
	for _, key := range keys {
		found := false
		for _, f := range s.fields {
			if f.jsonName == key {
				picked = append(picked, f)
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("z.Struct: unknown key %q in .Pick()", key))
		}
	}
	c.fields = picked
	return c
}

// @spec REQ-OBJ-006
// Omit excludes specified field keys. Panics if an unknown key is specified.
func (s *StructSchema[T]) Omit(keys ...string) *StructSchema[T] {
	c := s.clone()
	for _, key := range keys {
		found := false
		for _, f := range s.fields {
			if f.jsonName == key {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("z.Struct: unknown key %q in .Omit()", key))
		}
	}

	var kept []structFieldBinding
	for _, f := range s.fields {
		omit := false
		for _, key := range keys {
			if f.jsonName == key {
				omit = true
				break
			}
		}
		if !omit {
			kept = append(kept, f)
		}
	}
	c.fields = kept
	return c
}

// @spec REQ-OBJ-006
// Keyof returns an EnumSchema of registered JSON field keys.
func (s *StructSchema[T]) Keyof() *EnumSchema[string] {
	names := s.FieldNames()
	return Enum(names...)
}

// @spec REQ-OPT-007a
// ParseTo parses JSON data directly into an existing struct pointer, enabling zero-allocation reuse.
func (s *StructSchema[T]) ParseTo(data []byte, dst *T, ctx ...context.Context) error {
	if dst == nil {
		return fmt.Errorf("golang-zod: ParseTo destination pointer cannot be nil")
	}
	val, err := s.ParseJSON(data, ctx...)
	if err != nil {
		return err
	}
	*dst = val
	return nil
}

// @spec REQ-OBJ-009
func (s *StructSchema[T]) FieldNames() []string {
	names := make([]string, 0, len(s.fields))
	for _, f := range s.fields {
		if f.inline {
			names = append(names, inlineNames(f.schema)...)
			continue
		}
		names = append(names, f.jsonName)
	}
	return names
}

// @spec REQ-OBJ-009
func (s *StructSchema[T]) FieldSchema(name string) AnySchema {
	for _, f := range s.fields {
		if f.inline {
			if obj, ok := f.schema.node().(interface{ FieldSchema(string) AnySchema }); ok {
				if sub := obj.FieldSchema(name); sub != nil {
					return sub
				}
			}
			continue
		}
		if f.jsonName == name {
			return f.schema
		}
	}
	return nil
}

// @spec REQ-OBJ-009
func (s *StructSchema[T]) Optional() OptionalSchema[T] { return NewOptionalSchema[T](s) }
func (s *StructSchema[T]) Nullable() NullableSchema[T] { return NewNullableSchema[T](s) }
func (s *StructSchema[T]) Nullish() NullishSchema[T]   { return NewNullishSchema[T](s) }
func (s *StructSchema[T]) Default(v T) DefaultSchema[T] {
	return NewDefaultSchema[T](s, v)
}
func (s *StructSchema[T]) Array() *ArraySchema[T] { return NewArraySchema[T](s) }
func (s *StructSchema[T]) Meta(fn func(m *MetaBuilder)) *StructSchema[T] {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *StructSchema[T]) Error(msg string) *StructSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *StructSchema[T]) ErrorFunc(fn func(issue *Issue) string) *StructSchema[T] {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}

// schemaOutputType returns the Go type a schema's Parse yields, or nil when it cannot be determined.
func schemaOutputType(sch AnySchema) reflect.Type {
	m, ok := reflect.TypeOf(sch).MethodByName("Parse")
	if !ok || m.Type.NumOut() != 2 {
		return nil
	}
	return m.Type.Out(0)
}

// autoPtrSchema adapts a schema producing T to a struct field declared as *T:
// absent or null input yields nil, anything else a pointer to the parsed value.
type autoPtrSchema struct {
	inner   AnySchema
	ptrType reflect.Type
}

func (s *autoPtrSchema) GetMeta() Meta { return s.inner.GetMeta() }
func (s *autoPtrSchema) node() node {
	return &autoPtrNode{inner: s.inner.node(), schema: s.inner, ptrType: s.ptrType}
}

type autoPtrNode struct {
	inner   node
	schema  AnySchema
	ptrType reflect.Type
}

func (n *autoPtrNode) kind() nodeKind    { return kindNullish }
func (n *autoPtrNode) Unwrap() AnySchema { return n.schema }
func (n *autoPtrNode) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		return reflect.Zero(n.ptrType).Interface(), true
	}
	out, ok := n.inner.parse(ctx, in)
	if !ok {
		return nil, false
	}
	p := reflect.New(n.ptrType.Elem())
	p.Elem().Set(reflect.ValueOf(out))
	return p.Interface(), true
}


// inlineNames lists the JSON keys contributed by an inline (embedded) struct schema.
func inlineNames(sch AnySchema) []string {
	if obj, ok := sch.node().(interface{ FieldNames() []string }); ok {
		return obj.FieldNames()
	}
	return nil
}
