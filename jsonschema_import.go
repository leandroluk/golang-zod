package z

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// jsImportError is the panic payload used internally to abort an import; FromJSONSchema recovers it.
type jsImportError struct{ msg string }

func jsErr(path, format string, args ...any) jsImportError {
	return jsImportError{msg: path + ": " + fmt.Sprintf(format, args...)}
}

// unsupportedJSONSchemaKeywords are keywords that restrict the instance but have no equivalent in
// this library. They are rejected with an error instead of being ignored, because ignoring them
// would make the imported schema accept values the source schema rejects.
var unsupportedJSONSchemaKeywords = []string{
	"patternProperties", "propertyNames", "contains", "minContains", "maxContains",
	"dependentRequired", "dependentSchemas", "dependencies",
	"unevaluatedProperties", "unevaluatedItems", "$dynamicRef", "$recursiveRef", "$anchor",
}

// jsonSchemaCategoryKeywords lists, per instance category, the keywords that only constrain
// instances of that category. They drive the behaviour of schemas without an explicit "type".
var jsonSchemaCategoryKeywords = []struct {
	category string
	keywords []string
}{
	{"string", []string{"minLength", "maxLength", "pattern", "format"}},
	{"number", []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}},
	{"array", []string{"items", "prefixItems", "additionalItems", "minItems", "maxItems", "uniqueItems"}},
	{"object", []string{"properties", "required", "additionalProperties", "minProperties", "maxProperties"}},
}

var jsonSchemaFormats = map[string]func() *StringSchema{
	"email":        Email,
	"idn-email":    Email,
	"uuid":         UUID,
	"uri":          URL,
	"url":          URL,
	"iri":          URL,
	"hostname":     Hostname,
	"idn-hostname": Hostname,
	"ipv4":         IPv4,
	"ipv6":         IPv6,
	"date-time":    func() *StringSchema { return ISO.DateTime() },
	"date":         ISO.Date,
	"time":         ISO.Time,
	"duration":     ISO.Duration,
	"byte":         Base64,
}

type jsRef struct{ sch AnySchema }

type jsImporter struct {
	root map[string]any
	refs map[string]*jsRef
}

// @spec REQ-EXP-005
// FromJSONSchema reconstructs a schema from a JSON Schema document (draft-04 through 2020-12, plus
// the OpenAPI 3.0 "nullable" keyword).
//
// Supported: type (single or list, including "null"), integer (Int, or Int64 for format int64),
// const, enum, format (mapped to the top-level format constructors; unknown formats are
// annotations and ignored), string/number/array constraints (numeric and Draft-04 boolean
// exclusive bounds), prefixItems and array-form items, additionalProperties (false: strict,
// schema: catchall, true: loose, absent: default object), uniqueItems, minProperties and
// maxProperties, anyOf (Union), oneOf (XOR), allOf (all members must match), not, if/then/else, boolean
// schemas, keywords that apply to one instance category when "type" is absent, and local $ref
// ("#", "#/$defs/..", "#/definitions/..", "#/components/schemas/..", any JSON pointer); recursive
// references resolve through Lazy.
//
// Rejected with an error (ignoring them would make the result accept what the source rejects):
// remote or anchor $ref, $ref cycles that never pass through a property or item,
// patternProperties, propertyNames, contains, dependent*, unevaluated*, a multipleOf that is not a
// positive integer for integer types, and ECMA-262 patterns that Go's RE2 engine cannot compile.
// Annotations (title, description, default, examples, deprecated, readOnly, writeOnly, $comment,
// content*, discriminator, x-*) never affect validation and are ignored.
func FromJSONSchema(data []byte) (res Schema[any], err error) {
	var raw any
	if uerr := json.Unmarshal(data, &raw); uerr != nil {
		return nil, fmt.Errorf("golang-zod: failed to unmarshal JSON Schema: %w", uerr)
	}

	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("golang-zod: unsupported JSON Schema: %s", r.(jsImportError).msg)
		}
	}()

	im := &jsImporter{refs: map[string]*jsRef{}}
	im.root, _ = raw.(map[string]any)
	return asAny(im.parse(raw, "#", nil)), nil
}

// asAny re-wraps a schema as Schema[any]. Metadata is deliberately not copied: imported schemas carry
// none, and reading it from a Lazy would resolve the reference before the import has finished.
func asAny(s AnySchema) Schema[any] {
	return NewBaseSchema[any](s.node())
}

// parse converts one schema value. live holds the $refs followed since the last structural
// descent, used to reject reference cycles that never pass through a property or item.
func (im *jsImporter) parse(raw any, path string, live []string) AnySchema {
	switch v := raw.(type) {
	case bool:
		if v {
			return Any()
		}
		return Never()
	case map[string]any:
		return im.build(v, path, live)
	}
	panic(jsErr(path, "schema must be an object or boolean, got %T", raw))
}

func hasAnyKey(m map[string]any, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

func (im *jsImporter) build(m map[string]any, path string, live []string) AnySchema {
	for _, kw := range unsupportedJSONSchemaKeywords {
		if _, ok := m[kw]; ok {
			panic(jsErr(path, "keyword %q is not supported", kw))
		}
	}

	var parts []AnySchema

	if ref, ok := m["$ref"]; ok {
		parts = append(parts, im.ref(ref, path, live))
	}

	switch t := m["type"].(type) {
	case nil:
		for _, c := range jsonSchemaCategoryKeywords {
			if hasAnyKey(m, c.keywords...) {
				other := jsCheck(Any(), "", func(v any) bool { return kindOf(v) != c.category })
				parts = append(parts, Union(im.typed(c.category, m, path), other))
			}
		}
	case string:
		parts = append(parts, im.typed(t, m, path))
	case []any:
		var opts []AnySchema
		for _, e := range t {
			name, _ := e.(string)
			opts = append(opts, im.typed(name, m, path))
		}
		parts = append(parts, Union(opts...))
	default:
		panic(jsErr(path+"/type", "type must be a string or an array of strings"))
	}

	if c, ok := m["const"]; ok {
		parts = append(parts, constSchema(c))
	}
	if e, ok := m["enum"]; ok {
		parts = append(parts, enumSchema(e, path))
	}
	parts = append(parts, im.list(m, "allOf", path, live)...)
	if opts := im.list(m, "anyOf", path, live); opts != nil {
		parts = append(parts, Union(opts...))
	}
	if opts := im.list(m, "oneOf", path, live); opts != nil {
		parts = append(parts, XOR(opts...))
	}
	if n, ok := m["not"]; ok {
		inner := asAny(im.parse(n, path+"/not", live))
		parts = append(parts, jsCheck(Any(), "value must not match the \"not\" schema", func(v any) bool {
			_, err := inner.Parse(v)
			return err != nil
		}))
	}
	if i, ok := m["if"]; ok {
		parts = append(parts, im.conditional(m, i, path, live))
	}

	sch := allOfSchemas(parts)
	if n, _ := m["nullable"].(bool); n {
		sch = Union(sch, Null())
	}
	return sch
}

// list parses a non-empty schema array keyword (allOf/anyOf/oneOf); it returns nil when absent.
func (im *jsImporter) list(m map[string]any, key, path string, live []string) []AnySchema {
	v, ok := m[key]
	if !ok {
		return nil
	}
	items, _ := v.([]any)
	if len(items) == 0 {
		panic(jsErr(path+"/"+key, "%s must be a non-empty array", key))
	}
	out := make([]AnySchema, len(items))
	for i, it := range items {
		out[i] = im.parse(it, fmt.Sprintf("%s/%s/%d", path, key, i), live)
	}
	return out
}

func (im *jsImporter) conditional(m map[string]any, cond any, path string, live []string) AnySchema {
	ifSch := asAny(im.parse(cond, path+"/if", live))
	branch := func(key string) Schema[any] {
		if b, ok := m[key]; ok {
			return asAny(im.parse(b, path+"/"+key, live))
		}
		return Any()
	}
	thenSch, elseSch := branch("then"), branch("else")
	return jsCheck(Any(), "value must satisfy the if/then/else conditions", func(v any) bool {
		if _, err := ifSch.Parse(v); err == nil {
			_, err = thenSch.Parse(v)
			return err == nil
		}
		_, err := elseSch.Parse(v)
		return err == nil
	})
}

func (im *jsImporter) ref(rawRef any, path string, live []string) AnySchema {
	ref, _ := rawRef.(string)
	if !strings.HasPrefix(ref, "#") {
		panic(jsErr(path+"/$ref", "remote or invalid $ref %v is not supported", rawRef))
	}
	if slices.Contains(live, ref) {
		panic(jsErr(path+"/$ref", "circular $ref %q", ref))
	}
	entry, cached := im.refs[ref]
	if !cached {
		entry = &jsRef{}
		im.refs[ref] = entry
		entry.sch = im.parse(im.resolvePointer(ref, path), ref, append(slices.Clone(live), ref))
	}
	return Lazy[any](func() Schema[any] { return asAny(entry.sch) })
}

func (im *jsImporter) resolvePointer(ref, path string) any {
	var cur any = im.root
	ptr := ref[1:]
	if ptr == "" {
		return cur
	}
	if !strings.HasPrefix(ptr, "/") {
		panic(jsErr(path+"/$ref", "$ref %q is not a JSON pointer", ref))
	}
	for _, seg := range strings.Split(ptr[1:], "/") {
		if un, err := url.PathUnescape(seg); err == nil {
			seg = un
		}
		seg = strings.ReplaceAll(strings.ReplaceAll(seg, "~1", "/"), "~0", "~")
		found := false
		switch c := cur.(type) {
		case map[string]any:
			cur, found = c[seg]
		case []any:
			if i, err := strconv.Atoi(seg); err == nil && i >= 0 && i < len(c) {
				cur, found = c[i], true
			}
		}
		if !found {
			panic(jsErr(path+"/$ref", "$ref %q cannot be resolved", ref))
		}
	}
	return cur
}

func kindOf(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "other"
}

func constSchema(c any) AnySchema {
	switch c.(type) {
	case nil:
		return Null()
	case string, float64, bool:
		return Literal(c)
	}
	return jsCheck(Any(), "value must equal the const value", func(v any) bool { return valuesEqual(v, c) })
}

func enumSchema(e any, path string) AnySchema {
	vals, _ := e.([]any)
	if len(vals) == 0 {
		panic(jsErr(path+"/enum", "enum must be a non-empty array"))
	}
	strs := make([]string, 0, len(vals))
	for _, v := range vals {
		if s, ok := v.(string); ok {
			strs = append(strs, s)
		}
	}
	if len(strs) == len(vals) {
		return Enum(strs...)
	}
	return jsCheck(Any(), "value must be one of the enum values", func(v any) bool {
		return slices.ContainsFunc(vals, func(c any) bool { return valuesEqual(v, c) })
	})
}

func (im *jsImporter) typed(name string, m map[string]any, path string) AnySchema {
	switch name {
	case "string":
		return stringSchema(m, path)
	case "number":
		return numberSchema(m, path, false)
	case "integer":
		return numberSchema(m, path, true)
	case "boolean":
		return Boolean()
	case "null":
		return Null()
	case "array":
		return im.arraySchema(m, path)
	case "object":
		return im.objectSchema(m, path)
	}
	panic(jsErr(path+"/type", "unknown type %q", name))
}

// numKeyword reads an optional numeric keyword.
func numKeyword(m map[string]any, key, path string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	f, isNum := v.(float64)
	if !isNum {
		panic(jsErr(path+"/"+key, "%s must be a number", key))
	}
	return f, true
}

func stringSchema(m map[string]any, path string) AnySchema {
	s := String()
	if f, ok := m["format"].(string); ok {
		if ctor, known := jsonSchemaFormats[f]; known {
			s = ctor()
		}
	}
	if n, ok := numKeyword(m, "minLength", path); ok {
		s = s.Min(int(n))
	}
	if n, ok := numKeyword(m, "maxLength", path); ok {
		s = s.Max(int(n))
	}
	if p, ok := m["pattern"]; ok {
		pat, _ := p.(string)
		re, err := regexp.Compile(pat)
		if err != nil {
			panic(jsErr(path+"/pattern", "pattern %q is not supported: %v", p, err))
		}
		s = s.Regex(re)
	}
	return s
}

type jsBound struct {
	v  float64
	ex bool
}

func numberSchema(m map[string]any, path string, integer bool) AnySchema {
	var lows, highs []jsBound
	collect := func(key, exKey string, dst *[]jsBound) {
		if f, ok := numKeyword(m, key, path); ok {
			// Draft-04: exclusiveMinimum/Maximum is a boolean modifying minimum/maximum.
			ex, _ := m[exKey].(bool)
			*dst = append(*dst, jsBound{f, ex})
		}
		if f, ok := m[exKey].(float64); ok {
			*dst = append(*dst, jsBound{f, true})
		}
	}
	collect("minimum", "exclusiveMinimum", &lows)
	collect("maximum", "exclusiveMaximum", &highs)
	mult, hasMult := numKeyword(m, "multipleOf", path)
	if hasMult && mult <= 0 {
		panic(jsErr(path+"/multipleOf", "multipleOf must be greater than 0"))
	}
	if !integer {
		return numberBounds(Float64(), lows, highs, mult, hasMult, false, func(f float64) float64 { return f })
	}
	if hasMult && mult != math.Trunc(mult) {
		panic(jsErr(path+"/multipleOf", "multipleOf %v is not supported for integer types", mult))
	}
	if f, _ := m["format"].(string); f == "int64" {
		return numberBounds(Int64(), lows, highs, mult, hasMult, true, clampInt[int64])
	}
	return numberBounds(Int(), lows, highs, mult, hasMult, true, clampInt[int])
}

// clampInt converts a bound to an integer type, saturating at +-2^53 (the largest range in which
// JSON numbers decoded as float64 are exact integers).
func clampInt[T int | int64](f float64) T {
	return T(math.Max(-(1 << 53), math.Min(1<<53, f)))
}

func numberBounds[T NumberConstraint](s *NumberSchema[T], lows, highs []jsBound, mult float64, hasMult, integer bool, conv func(float64) T) AnySchema {
	for _, b := range lows {
		switch {
		case integer && b.ex:
			s = s.GTE(conv(math.Floor(b.v) + 1))
		case integer:
			s = s.GTE(conv(math.Ceil(b.v)))
		case b.ex:
			s = s.GT(conv(b.v))
		default:
			s = s.GTE(conv(b.v))
		}
	}
	for _, b := range highs {
		switch {
		case integer && b.ex:
			s = s.LTE(conv(math.Ceil(b.v) - 1))
		case integer:
			s = s.LTE(conv(math.Floor(b.v)))
		case b.ex:
			s = s.LT(conv(b.v))
		default:
			s = s.LTE(conv(b.v))
		}
	}
	if hasMult {
		s = s.MultipleOf(conv(mult))
	}
	return s
}

func (im *jsImporter) arraySchema(m map[string]any, path string) AnySchema {
	prefix, _ := m["prefixItems"].([]any)
	rest, restKey := m["items"], "items"
	if legacy, ok := rest.([]any); ok { // Draft-04 .. 2019-09 tuple form.
		prefix, rest, restKey = legacy, m["additionalItems"], "additionalItems"
	}
	minItems, hasMin := numKeyword(m, "minItems", path)
	maxItems, hasMax := numKeyword(m, "maxItems", path)

	var sch AnySchema
	if len(prefix) == 0 {
		var elem AnySchema = Any()
		if rest != nil {
			elem = im.parse(rest, path+"/"+restKey, nil)
		}
		arr := Array[any](asAny(elem))
		if hasMin {
			arr = arr.Min(int(minItems))
		}
		if hasMax {
			arr = arr.Max(int(maxItems))
		}
		sch = arr
	} else {
		sch = im.tupleSchema(prefix, rest, restKey, int(minItems), path)
		if hasMin || hasMax {
			sch = jsCheck(sch, "array length is out of range", func(v any) bool {
				n := len(v.([]any))
				return (!hasMin || n >= int(minItems)) && (!hasMax || n <= int(maxItems))
			})
		}
	}
	if u, _ := m["uniqueItems"].(bool); u {
		sch = jsCheck(sch, "array items must be unique", func(v any) bool {
			arr := v.([]any)
			for i := range arr {
				if slices.ContainsFunc(arr[:i], func(p any) bool { return valuesEqual(p, arr[i]) }) {
					return false
				}
			}
			return true
		})
	}
	return sch
}

// tupleSchema builds the positional schema. JSON Schema lets arrays be shorter than the prefix
// (down to minItems), so shorter exact-length tuples are offered as alternatives.
func (im *jsImporter) tupleSchema(prefix []any, rest any, restKey string, minItems int, path string) AnySchema {
	items := make([]AnySchema, len(prefix))
	for i, p := range prefix {
		items[i] = im.parse(p, fmt.Sprintf("%s/prefixItems/%d", path, i), nil)
	}
	full := Tuple(items...)
	switch r := rest.(type) {
	case nil:
		full = full.Rest(Any())
	case bool:
		if r {
			full = full.Rest(Any())
		}
	default:
		full = full.Rest(im.parse(r, path+"/"+restKey, nil))
	}
	opts := []AnySchema{full}
	for k := len(items) - 1; k >= max(minItems, 0); k-- {
		opts = append(opts, Tuple(items[:k]...))
	}
	if len(opts) == 1 {
		return full
	}
	return Union(opts...)
}

func (im *jsImporter) objectSchema(m map[string]any, path string) AnySchema {
	props, _ := m["properties"].(map[string]any)
	required := map[string]bool{}
	reqList, _ := m["required"].([]any)
	for _, r := range reqList {
		if s, ok := r.(string); ok {
			required[s] = true
		}
	}

	names := make([]string, 0, len(props)+len(required))
	for k := range props {
		names = append(names, k)
	}
	for k := range required {
		if _, ok := props[k]; !ok {
			names = append(names, k)
		}
	}
	slices.Sort(names)

	fields := make([]AnySchema, len(names))
	for i, k := range names {
		var f AnySchema = Any()
		if p, ok := props[k]; ok {
			f = im.parse(p, path+"/properties/"+k, nil)
		}
		if !required[k] {
			f = jsOptional(f)
		}
		fields[i] = f
	}
	obj := Object(func(s *ObjectBuilder) {
		for i, k := range names {
			s.Field(k, fields[i])
		}
	})

	switch ap := m["additionalProperties"].(type) {
	case bool:
		if ap {
			obj = obj.Loose()
		} else {
			obj = obj.Strict()
		}
	case map[string]any:
		obj = obj.Catchall(im.parse(ap, path+"/additionalProperties", nil))
	}

	minProps, hasMin := numKeyword(m, "minProperties", path)
	maxProps, hasMax := numKeyword(m, "maxProperties", path)
	if !hasMin && !hasMax {
		return obj
	}
	return jsCheck(obj, "object property count is out of range", func(in any) bool {
		v := in.(Dyn)
		return (!hasMin || len(v) >= int(minProps)) && (!hasMax || len(v) <= int(maxProps))
	})
}

// jsPredicateNode runs inner and then an extra predicate on the raw input (outputs may be transformed, e.g. integers become int). It is used instead of
// Refine because Refine skips its predicate when the parsed value is nil.
type jsPredicateNode struct {
	inner AnySchema
	msg   string
	fn    func(any) bool
}

func (n *jsPredicateNode) kind() nodeKind { return kindCustom }
func (n *jsPredicateNode) parse(ctx *parseCtx, in any) (any, bool) {
	out, ok := n.inner.node().parse(ctx, in)
	if !ok {
		return nil, false
	}
	if !n.fn(in) {
		ctx.addIssue(Issue{Code: Custom, Message: n.msg, Input: in})
		return nil, false
	}
	return out, true
}

func jsCheck(inner AnySchema, msg string, fn func(any) bool) AnySchema {
	return &jsCheckSchema{BaseSchema: NewBaseSchema[any](&jsPredicateNode{inner: inner, msg: msg, fn: fn})}
}

type jsCheckSchema struct{ BaseSchema[any] }

// allOfSchemas combines schemas that must all accept the same input. Intersection is not used
// because it merges outputs and fails on equal values of different Go types (int vs float64) or
// drops keys of strip-mode objects; JSON Schema allOf is a pure conjunction of validity.
func allOfSchemas(parts []AnySchema) AnySchema {
	switch len(parts) {
	case 0:
		return Any()
	case 1:
		return parts[0]
	}
	return &jsCheckSchema{BaseSchema: NewBaseSchema[any](&jsAllNode{parts: parts})}
}

type jsAllNode struct{ parts []AnySchema }

func (n *jsAllNode) kind() nodeKind { return kindCustom }
func (n *jsAllNode) parse(ctx *parseCtx, in any) (any, bool) {
	var result any
	for i, p := range n.parts {
		out, ok := p.node().parse(ctx, in)
		if !ok {
			return nil, false
		}
		// Objects keep the fields produced by every member (earlier members win on conflicts).
		merged, isMap := result.(map[string]any)
		next, nextIsMap := out.(map[string]any)
		switch {
		case i == 0:
			result = out
		case isMap && nextIsMap:
			merged = maps.Clone(merged)
			for k, v := range next {
				if _, exists := merged[k]; !exists {
					merged[k] = v
				}
			}
			result = merged
		}
	}
	return result, true
}

// jsOptionalNode marks an object member as optional (it may be absent) but, unlike optionalNode,
// validates an explicit null against the member schema: in JSON Schema only absence is optional.
type jsOptionalNode struct{ inner AnySchema }

func (n *jsOptionalNode) kind() nodeKind    { return kindOptional }
func (n *jsOptionalNode) Unwrap() AnySchema { return n.inner }
func (n *jsOptionalNode) parse(ctx *parseCtx, in any) (any, bool) {
	return n.inner.node().parse(ctx, in)
}

func jsOptional(s AnySchema) AnySchema {
	return &anyOptionalWrapper{inner: s, opt: &jsOptionalNode{inner: s}}
}
