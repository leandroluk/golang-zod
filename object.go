package z

import (
	"fmt"
	"slices"
)

type unknownKeysMode int

const (
	unknownKeysStrip unknownKeysMode = iota
	unknownKeysStrict
	unknownKeysLoose
	unknownKeysCatchall
)

// @spec REQ-OBJ-009
// ObjectNode is satisfied by ObjectSchema and StructSchema, exposing key metadata to exporters.
type ObjectNode interface {
	AnySchema
	FieldNames() []string
	FieldSchema(name string) AnySchema
}

type fieldEntry struct {
	name   string
	schema AnySchema
}

// @spec REQ-OBJ-001
// ObjectBuilder is the builder used by z.Object to define fields with string keys.
type ObjectBuilder struct {
	fields []*fieldEntry
}

// @spec REQ-OBJ-001
// Field registers a named field. Panics on duplicate names.
// When called with 1 argument, returns an *ObjectFieldBuilder with type aliases (.String(), .Int(), etc.).
// When called with 2 arguments (backwards compatibility), registers the schema directly.
func (s *ObjectBuilder) Field(name string, sch ...AnySchema) *ObjectFieldBuilder {
	for _, f := range s.fields {
		if f.name == name {
			panic(fmt.Sprintf("duplicate field %q registered in z.Object", name))
		}
	}
	entry := &fieldEntry{name: name}
	s.fields = append(s.fields, entry)
	b := &ObjectFieldBuilder{
		builder: s,
		entry:   entry,
	}
	if len(sch) > 0 && sch[0] != nil {
		b.Schema(sch[0])
	}
	return b
}

// ObjectFieldBuilder provides fluent methods to declare schemas on an object field.
type ObjectFieldBuilder struct {
	builder *ObjectBuilder
	entry   *fieldEntry
}

func (b *ObjectFieldBuilder) bind(sch AnySchema) {
	b.entry.schema = sch
	if s, ok := any(sch).(interface{ setSlot(*AnySchema) }); ok {
		s.setSlot(&b.entry.schema)
	}
}

func (b *ObjectFieldBuilder) Schema(sch AnySchema) AnySchema {
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) String() *StringSchema {
	sch := String()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Int() *NumberSchema[int] {
	sch := Int()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Int8() *NumberSchema[int8] {
	sch := Int8()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Int16() *NumberSchema[int16] {
	sch := Int16()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Int32() *NumberSchema[int32] {
	sch := Int32()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Int64() *NumberSchema[int64] {
	sch := Int64()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Uint() *NumberSchema[uint] {
	sch := Uint()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Uint8() *NumberSchema[uint8] {
	sch := Uint8()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Uint16() *NumberSchema[uint16] {
	sch := Uint16()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Uint32() *NumberSchema[uint32] {
	sch := Uint32()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Uint64() *NumberSchema[uint64] {
	sch := Uint64()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Float32() *NumberSchema[float32] {
	sch := Float32()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Float64() *NumberSchema[float64] {
	sch := Float64()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) BigInt() *BigIntSchema {
	sch := BigInt()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Boolean() *BooleanSchema {
	sch := Boolean()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Bool() *BooleanSchema {
	return b.Boolean()
}

func (b *ObjectFieldBuilder) Date() *DateSchema {
	sch := Date()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Bytes() *BytesSchema {
	sch := Bytes()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Email() *StringSchema {
	sch := Email()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) UUID() *StringSchema {
	sch := UUID()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) URL() *StringSchema {
	sch := URL()
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Enum(values ...string) *EnumSchema[string] {
	sch := Enum(values...)
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Literal(val any) AnySchema {
	sch := Literal(val)
	b.bind(sch)
	return sch
}

func (b *ObjectFieldBuilder) Any() *AnyPrimitiveSchema {
	sch := Any()
	b.bind(sch)
	return sch
}

// @spec REQ-OBJ-001
// ObjectSchema validates dynamic object maps (map[string]any).
type ObjectSchema struct {
	BaseSchema[Dyn]
	fields      []fieldEntry
	mode        unknownKeysMode
	catchallSch AnySchema
}

type objectNode struct {
	schema *ObjectSchema
}

func (n *objectNode) kind() nodeKind                    { return kindObject }
func (n *objectNode) FieldNames() []string              { return n.schema.FieldNames() }
func (n *objectNode) FieldSchema(name string) AnySchema { return n.schema.FieldSchema(name) }
func (n *objectNode) Mode() unknownKeysMode             { return n.schema.mode }
func (n *objectNode) CatchallSchema() AnySchema         { return n.schema.catchallSch }
func (n *objectNode) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "object",
			Input:    in,
		})
		return nil, false
	}

	rawMap, ok := in.(map[string]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "object",
			Input:    in,
		})
		return nil, false
	}

	result := make(Dyn)
	allOk := true
	seenKeys := make(map[string]struct{}, len(n.schema.fields))

	// 1. Validate registered fields
	for _, f := range n.schema.fields {
		seenKeys[f.name] = struct{}{}
		rawVal, present := rawMap[f.name]

		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}

		k := f.schema.node().kind()

		if !present {
			// Check optionality
			if k == kindOptional || k == kindNullish {
				// Absent optional keys are omitted from Dyn
				continue
			}
			if k == kindDefault || k == kindPrefault || k == kindCatch {
				// Default schema supplies fallback on absence
				out, ok := parseAbsent(ctx, f.schema.node())
				if ok {
					result[f.name] = out
				}
				continue
			}

			// Required key missing
			ctx.pushPath(Property(f.name))
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

		// Key is present
		ctx.pushPath(Property(f.name))
		out, ok := f.schema.node().parse(ctx, rawVal)
		ctx.popPath()

		if !ok {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		result[f.name] = out
	}

	if !allOk && ctx.shouldAbort() {
		return nil, false
	}

	// 2. Handle unknown keys
	var unrecognized []string
unknownKeys:
	for k, v := range rawMap {
		if _, known := seenKeys[k]; known {
			continue
		}

		switch n.schema.mode {
		case unknownKeysStrip:
			// Stripped (Zod default)
		case unknownKeysStrict:
			unrecognized = append(unrecognized, k)
		case unknownKeysLoose:
			result[k] = v
		case unknownKeysCatchall:
			if n.schema.catchallSch != nil {
				ctx.pushPath(Property(k))
				out, ok := n.schema.catchallSch.node().parse(ctx, v)
				ctx.popPath()
				if !ok {
					allOk = false
					if ctx.shouldAbort() {
						break unknownKeys
					}
					continue
				}
				result[k] = out
			}
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

	if !allOk {
		return nil, false
	}
	return result, true
}

// @spec REQ-OBJ-001
// Object creates an object schema with strip mode by default.
func Object(build func(s *ObjectBuilder)) *ObjectSchema {
	s := &ObjectBuilder{}
	build(s)

	fields := make([]fieldEntry, 0, len(s.fields))
	for _, f := range s.fields {
		if f.schema == nil {
			panic(fmt.Sprintf("z.Object: field %q registered without schema", f.name))
		}
		fields = append(fields, *f)
	}

	obj := &ObjectSchema{
		fields: fields,
		mode:   unknownKeysStrip,
	}
	obj.BaseSchema = NewBaseSchema[Dyn](&objectNode{schema: obj})
	return obj
}

// @spec REQ-OBJ-001
// StrictObject creates an object schema in strict mode (rejects unknown keys).
func StrictObject(build func(s *ObjectBuilder)) *ObjectSchema {
	return Object(build).Strict()
}

// @spec REQ-OBJ-001
// LooseObject creates an object schema in loose mode (preserves unknown keys).
func LooseObject(build func(s *ObjectBuilder)) *ObjectSchema {
	return Object(build).Loose()
}

func (s *ObjectSchema) clone() *ObjectSchema {
	c := &ObjectSchema{
		fields:      slices.Clone(s.fields),
		mode:        s.mode,
		catchallSch: s.catchallSch,
	}
	c.BaseSchema = NewBaseSchema[Dyn](&objectNode{schema: c})
	c.setMeta(s.getMeta())
	c.slot = s.slot
	c.updateSlot(c)
	return c
}

// @spec REQ-OBJ-002
// Strict rejects unrecognized keys.
func (s *ObjectSchema) Strict() *ObjectSchema {
	c := s.clone()
	c.mode = unknownKeysStrict
	return c
}

// @spec REQ-OBJ-002
// Loose preserves unrecognized keys in the parsed result.
func (s *ObjectSchema) Loose() *ObjectSchema {
	c := s.clone()
	c.mode = unknownKeysLoose
	return c
}

// @spec REQ-OBJ-002
// Strip restores default unknown key stripping.
func (s *ObjectSchema) Strip() *ObjectSchema {
	c := s.clone()
	c.mode = unknownKeysStrip
	return c
}

// @spec REQ-OBJ-002
// Catchall validates unrecognized keys against catchallSchema.
func (s *ObjectSchema) Catchall(catchallSchema AnySchema) *ObjectSchema {
	c := s.clone()
	c.mode = unknownKeysCatchall
	c.catchallSch = catchallSchema
	return c
}

// @spec REQ-OBJ-002
// Extend merges new fields into the object schema, inheriting unknown key mode.
func (s *ObjectSchema) Extend(build func(shape *ObjectBuilder)) *ObjectSchema {
	c := s.clone()
	sub := &ObjectBuilder{}
	build(sub)

	// Replace existing or append
	for _, newF := range sub.fields {
		if newF.schema == nil {
			panic(fmt.Sprintf("z.Object: field %q registered without schema in Extend", newF.name))
		}
		found := false
		for i, oldF := range c.fields {
			if oldF.name == newF.name {
				c.fields[i] = *newF
				found = true
				break
			}
		}
		if !found {
			c.fields = append(c.fields, *newF)
		}
	}
	c.BaseSchema = NewBaseSchema[Dyn](&objectNode{schema: c})
	c.setMeta(s.getMeta())
	return c
}

// @spec REQ-OBJ-002
// Pick retains only the named keys. Panics if an unknown key name is specified.
func (s *ObjectSchema) Pick(keys ...string) *ObjectSchema {
	c := s.clone()
	var picked []fieldEntry
	for _, key := range keys {
		found := false
		for _, f := range s.fields {
			if f.name == key {
				picked = append(picked, f)
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("unknown key %q in .Pick()", key))
		}
	}
	c.fields = picked
	return c
}

// @spec REQ-OBJ-002
// Omit excludes the named keys. Panics if an unknown key name is specified.
func (s *ObjectSchema) Omit(keys ...string) *ObjectSchema {
	c := s.clone()
	for _, key := range keys {
		found := false
		for _, f := range s.fields {
			if f.name == key {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("unknown key %q in .Omit()", key))
		}
	}

	var kept []fieldEntry
	for _, f := range s.fields {
		omit := false
		for _, key := range keys {
			if f.name == key {
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

// @spec REQ-OBJ-002
// Partial marks all (or specified) keys as optional.
func (s *ObjectSchema) Partial(keys ...string) *ObjectSchema {
	c := s.clone()
	targetKeys := make(map[string]struct{})
	for _, k := range keys {
		found := false
		for _, f := range s.fields {
			if f.name == k {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("unknown key %q in .Partial()", k))
		}
		targetKeys[k] = struct{}{}
	}

	for i, f := range c.fields {
		if len(keys) == 0 {
			c.fields[i].schema = makeAnyOptional(f.schema)
		} else if _, match := targetKeys[f.name]; match {
			c.fields[i].schema = makeAnyOptional(f.schema)
		}
	}
	return c
}

// @spec REQ-OBJ-002
// Required marks all (or specified) keys as required.
func (s *ObjectSchema) Required(keys ...string) *ObjectSchema {
	c := s.clone()
	targetKeys := make(map[string]struct{})
	for _, k := range keys {
		found := false
		for _, f := range s.fields {
			if f.name == k {
				found = true
				break
			}
		}
		if !found {
			panic(fmt.Sprintf("unknown key %q in .Required()", k))
		}
		targetKeys[k] = struct{}{}
	}

	for i, f := range c.fields {
		if len(keys) == 0 || containsKey(targetKeys, f.name) {
			c.fields[i].schema = unwrapOptional(f.schema)
		}
	}
	return c
}

func containsKey(m map[string]struct{}, k string) bool {
	_, ok := m[k]
	return ok
}

// @spec REQ-OBJ-002
// Keyof returns an EnumSchema of the object's keys.
func (s *ObjectSchema) Keyof() *EnumSchema[string] {
	names := s.FieldNames()
	return Enum(names...)
}

// @spec REQ-OBJ-009
func (s *ObjectSchema) FieldNames() []string {
	names := make([]string, len(s.fields))
	for i, f := range s.fields {
		names[i] = f.name
	}
	return names
}

// @spec REQ-OBJ-009
func (s *ObjectSchema) FieldSchema(name string) AnySchema {
	for _, f := range s.fields {
		if f.name == name {
			return f.schema
		}
	}
	return nil
}

type anyOptionalWrapper struct {
	inner AnySchema
	opt   node
}

func (w *anyOptionalWrapper) GetMeta() Meta { return w.inner.GetMeta() }
func (w *anyOptionalWrapper) node() node    { return w.opt }

func makeAnyOptional(s AnySchema) AnySchema {
	if s.node().kind() == kindOptional {
		return s
	}
	opt := &optionalNode[any]{inner: s.node(), schema: s}
	return &anyOptionalWrapper{inner: s, opt: opt}
}

func unwrapOptional(s AnySchema) AnySchema {
	if w, ok := s.(*anyOptionalWrapper); ok {
		return w.inner
	}
	return s
}

// @spec REQ-OBJ-009
func (s *ObjectSchema) Optional() OptionalSchema[Dyn] { return NewOptionalSchema[Dyn](s) }
func (s *ObjectSchema) Nullable() NullableSchema[Dyn] { return NewNullableSchema[Dyn](s) }
func (s *ObjectSchema) Nullish() NullishSchema[Dyn]   { return NewNullishSchema[Dyn](s) }
func (s *ObjectSchema) Default(v Dyn) DefaultSchema[Dyn] {
	return NewDefaultSchema[Dyn](s, v)
}
func (s *ObjectSchema) Array() *ArraySchema[Dyn] { return NewArraySchema[Dyn](s) }
func (s *ObjectSchema) Meta(fn func(m *MetaBuilder)) *ObjectSchema {
	c := s.clone()
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}

// @spec REQ-PIPE-007
// Error sets a schema-level error message.
func (s *ObjectSchema) Error(msg string) *ObjectSchema {
	c := *s
	c.meta.errOv = &errorOverride{msg: msg}
	return &c
}

// @spec REQ-PIPE-007
// ErrorFunc sets a schema-level error function; returning "" falls through to the next level.
func (s *ObjectSchema) ErrorFunc(fn func(issue *Issue) string) *ObjectSchema {
	c := *s
	c.meta.errOv = &errorOverride{fn: fn}
	return &c
}
