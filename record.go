package z

import (
	"slices"
)

type recordMode int

const (
	recordModeDefault recordMode = iota
	recordModePartial
	recordModeLoose
)

// @spec REQ-OBJ-008
// RecordSchema validates dictionary entries matching a key and value schema.
type RecordSchema[K comparable, V any] struct {
	BaseSchema[map[K]V]
	keySchema Schema[K]
	valSchema Schema[V]
	mode      recordMode
}

type recordNode[K comparable, V any] struct {
	schema *RecordSchema[K, V]
}

func (n *recordNode[K, V]) kind() nodeKind { return kindRecord }
func (n *recordNode[K, V]) parse(ctx *parseCtx, in any) (any, bool) {
	if in == nil {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "record",
			Input:    in,
		})
		return nil, false
	}

	rawMap, ok := in.(map[string]any)
	if !ok {
		ctx.addIssue(Issue{
			Code:     InvalidType,
			Expected: "record",
			Input:    in,
		})
		return nil, false
	}

	res := make(map[K]V)
	allOk := true

	// Check if key schema is an Enum (requires exhaustiveness in default mode)
	isEnumKey := n.schema.keySchema.node().kind() == kindEnum
	var expectedEnumKeys []string
	if isEnumKey {
		if enumSch, ok := any(n.schema.keySchema).(interface{ Values() []string }); ok {
			expectedEnumKeys = enumSch.Values()
		}
	}

	seenKeys := make(map[string]struct{})
	var unrecognized []string

	for kStr, rawVal := range rawMap {
		if err := ctx.checkCancel(); err != nil {
			return nil, false
		}

		mark := len(ctx.issues)
		keyOut, keyOk := n.schema.keySchema.node().parse(ctx, kStr)
		if !keyOk {
			if n.schema.mode == recordModeLoose {
				// Non-matching keys are skipped silently: drop the key schema's own issues.
				ctx.issues = ctx.issues[:mark]
				continue
			}
			if isEnumKey {
				// Reported collectively as unrecognized_keys below.
				ctx.issues = ctx.issues[:mark]
				unrecognized = append(unrecognized, kStr)
				allOk = false
				continue
			}
			ctx.pushPath(Property(kStr))
			ctx.addIssue(Issue{
				Code:   InvalidKey,
				Origin: "record",
				Input:  kStr,
			})
			ctx.popPath()
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		seenKeys[kStr] = struct{}{}
		typedKey, ok := keyOut.(K)
		if !ok {
			allOk = false
			continue
		}

		ctx.pushPath(Property(kStr))
		valOut, valOk := n.schema.valSchema.node().parse(ctx, rawVal)
		ctx.popPath()
		if !valOk {
			allOk = false
			if ctx.shouldAbort() {
				break
			}
			continue
		}

		if typedVal, ok := valOut.(V); ok {
			res[typedKey] = typedVal
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

	// Exhaustiveness check for enum keys in default mode
	if isEnumKey && n.schema.mode == recordModeDefault {
		for _, exp := range expectedEnumKeys {
			if _, present := seenKeys[exp]; !present {
				ctx.pushPath(Property(exp))
				ctx.addIssue(Issue{
					Code:     InvalidType,
					Expected: "required",
					Input:    nil,
				})
				ctx.popPath()
				allOk = false
			}
		}
	}

	if !allOk {
		return nil, false
	}
	return res, true
}

// @spec REQ-OBJ-008
// Record creates a record schema mapping keys to values.
func Record[K comparable, V any](key Schema[K], val Schema[V]) *RecordSchema[K, V] {
	s := &RecordSchema[K, V]{
		keySchema: key,
		valSchema: val,
		mode:      recordModeDefault,
	}
	s.BaseSchema = NewBaseSchema[map[K]V](&recordNode[K, V]{schema: s})
	return s
}

// @spec REQ-OBJ-008
// PartialRecord creates a record schema where enum keys are not required to be exhaustive.
func PartialRecord[K comparable, V any](key Schema[K], val Schema[V]) *RecordSchema[K, V] {
	s := Record(key, val)
	s.mode = recordModePartial
	return s
}

// @spec REQ-OBJ-008
// LooseRecord creates a record schema where unmatched keys pass through without error.
func LooseRecord[K comparable, V any](key Schema[K], val Schema[V]) *RecordSchema[K, V] {
	s := Record(key, val)
	s.mode = recordModeLoose
	return s
}

// @spec REQ-OBJ-009
func (s *RecordSchema[K, V]) Optional() OptionalSchema[map[K]V] {
	return NewOptionalSchema[map[K]V](s)
}
func (s *RecordSchema[K, V]) Nullable() NullableSchema[map[K]V] {
	return NewNullableSchema[map[K]V](s)
}
func (s *RecordSchema[K, V]) Nullish() NullishSchema[map[K]V] {
	return NewNullishSchema[map[K]V](s)
}
func (s *RecordSchema[K, V]) Default(v map[K]V) DefaultSchema[map[K]V] {
	return NewDefaultSchema[map[K]V](s, v)
}
func (s *RecordSchema[K, V]) Array() *ArraySchema[map[K]V] {
	return NewArraySchema[map[K]V](s)
}
func (s *RecordSchema[K, V]) Meta(fn func(m *MetaBuilder)) *RecordSchema[K, V] {
	c := Record(s.keySchema, s.valSchema)
	c.mode = s.mode
	b := NewMetaBuilder(s.GetMeta())
	fn(b)
	c.SetMeta(b.Build())
	return c
}
