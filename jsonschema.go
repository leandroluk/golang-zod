package z

import (
	"encoding/json/v2"
	"fmt"
	"maps"
	"slices"
)

// @spec REQ-EXP-001
// JSONSchemaTarget defines the output dialect for JSON Schema / OpenAPI.
type JSONSchemaTarget int

const (
	Draft202012 JSONSchemaTarget = iota
	Draft07
	Draft04
	OpenAPI30
	OpenAPI31
)

// @spec REQ-EXP-004
type SchemaIO int

const (
	Output SchemaIO = iota
	Input
)

// @spec REQ-EXP-004
type UnrepresentableAction int

const (
	UnrepresentableThrow UnrepresentableAction = iota
	UnrepresentableAny
)

// @spec REQ-EXP-003
type CyclesAction int

const (
	CyclesRef CyclesAction = iota
	CyclesThrow
)

// @spec REQ-EXP-003
type ReusedAction int

const (
	ReusedInline ReusedAction = iota
	ReusedRef
)

// JSONSchemaOverride is called once for every schema that produces its own JSON Schema object, after the
// object is built and before it is stored in the definitions or returned. It may mutate js in place.
// Transparent wrappers (Optional, Default, Catch, Prefault, Readonly, Pipe, Codec, Lazy, refinements)
// delegate to the schema they wrap and do not trigger a call of their own. path locates the schema in the
// root ("" for the root, ".name" for an object key, "[]" for an array element, ".*" for a record value,
// "|i" for union option i).
type JSONSchemaOverride func(s AnySchema, path string, js map[string]any)

// JSONSchemaOptions configures ToJSONSchema generation.
type JSONSchemaOptions struct {
	Target          JSONSchemaTarget
	IO              SchemaIO
	Unrepresentable UnrepresentableAction
	Cycles          CyclesAction
	Reused          ReusedAction
	URI             func(id string) string
	// Override post-processes each generated schema (see JSONSchemaOverride).
	Override JSONSchemaOverride
	// Metadata supplies extra metadata for a schema from outside the schema itself (Zod's metadata registry).
	// Returned fields are merged over the schema's own Meta: non-empty Title/Description/Examples replace,
	// Deprecated/ReadOnly are OR-ed, Custom keys are merged; ID is never changed.
	Metadata func(s AnySchema) (Meta, bool)
}

// JSONSchemaOption configures options for ToJSONSchema.
type JSONSchemaOption func(*JSONSchemaOptions)

func WithTarget(t JSONSchemaTarget) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Target = t }
}

func WithIO(io SchemaIO) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.IO = io }
}

func WithUnrepresentable(a UnrepresentableAction) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Unrepresentable = a }
}

func WithCycles(c CyclesAction) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Cycles = c }
}

// WithReused selects how a schema used more than once is emitted: ReusedInline (default) repeats it, while
// ReusedRef moves it to the definitions ($defs, definitions or components/schemas) and emits $refs.
// Schemas without a Meta.ID get the generated name __schemaN.
func WithReused(r ReusedAction) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Reused = r }
}

func WithURI(fn func(id string) string) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.URI = fn }
}

// WithOverride registers a post-processing hook (Zod's override), see JSONSchemaOverride.
func WithOverride(fn JSONSchemaOverride) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Override = fn }
}

// WithMetadata registers an external metadata source (Zod's metadata registry), see JSONSchemaOptions.Metadata.
func WithMetadata(fn func(s AnySchema) (Meta, bool)) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Metadata = fn }
}

type schemaGenCtx struct {
	opts   JSONSchemaOptions
	defs   map[string]map[string]any
	active map[node]bool

	// counting is true during the pre-pass of Reused(Ref): it only counts how often each node is reached.
	counting bool
	seen     map[node]int
	// counts holds the pre-pass result; names holds the generated __schemaN names.
	counts map[node]int
	names  map[node]string
}

func newSchemaGenCtx(opts JSONSchemaOptions) *schemaGenCtx {
	return &schemaGenCtx{
		opts:   opts,
		defs:   make(map[string]map[string]any),
		active: make(map[node]bool),
		seen:   make(map[node]int),
		names:  make(map[node]string),
	}
}

// nameFor returns the generated definition name of n.
func (ctx *schemaGenCtx) nameFor(n node) string {
	name, ok := ctx.names[n]
	if !ok {
		name = fmt.Sprintf("__schema%d", len(ctx.names))
		ctx.names[n] = name
	}
	return name
}

// @spec REQ-EXP-001
// @spec REQ-EXP-002
// @spec REQ-EXP-003
// @spec REQ-EXP-004
// @spec REQ-EXP-007
func ToJSONSchema(s AnySchema, opts ...JSONSchemaOption) ([]byte, error) {
	cfg := JSONSchemaOptions{
		Target:          Draft202012,
		IO:              Output,
		Unrepresentable: UnrepresentableThrow,
		Cycles:          CyclesRef,
		Reused:          ReusedInline,
	}
	for _, o := range opts {
		o(&cfg)
	}

	ctx := newSchemaGenCtx(cfg)
	if cfg.Reused == ReusedRef {
		// Pre-pass: count how often every node is reached so reused ones can be moved to the definitions.
		pre := newSchemaGenCtx(cfg)
		pre.counting = true
		if _, err := pre.generateNode(s, s.GetMeta(), ""); err != nil {
			return nil, err
		}
		ctx.counts = pre.seen
	}

	root, err := ctx.generateNode(s, s.GetMeta(), "")
	if err != nil {
		return nil, err
	}

	// A root registered in defs under its own name is the very same map: keep a pristine copy in defs so
	// the root-level keywords added below ($schema, $defs, ...) never leak into the definition.
	for id, def := range ctx.defs {
		if fmt.Sprintf("%p", def) == fmt.Sprintf("%p", root) {
			ctx.defs[id] = maps.Clone(root)
		}
	}

	// Add $schema if draft and root is non-empty
	if len(root) > 0 {
		switch cfg.Target {
		case Draft202012:
			root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		case Draft07:
			root["$schema"] = "http://json-schema.org/draft-07/schema#"
		case Draft04:
			root["$schema"] = "http://json-schema.org/draft-04/schema#"
		}
	}

	if len(ctx.defs) > 0 {
		defsKey := "$defs"
		if cfg.Target == Draft07 || cfg.Target == Draft04 {
			defsKey = "definitions"
		} else if cfg.Target == OpenAPI30 {
			defsKey = "components"
		}

		if cfg.Target == OpenAPI30 {
			root[defsKey] = map[string]any{"schemas": ctx.defs}
		} else {
			root[defsKey] = ctx.defs
		}
	}

	return json.Marshal(root, json.Deterministic(true))
}

// unrepresentable applies the Unrepresentable option to a schema that JSON Schema cannot express.
func (ctx *schemaGenCtx) unrepresentable(what, path string) (map[string]any, error) {
	if ctx.opts.Unrepresentable == UnrepresentableThrow {
		return nil, fmt.Errorf("golang-zod: unrepresentable %s at %q", what, path)
	}
	return map[string]any{}, nil
}

// countable reports whether nodes of kind k take part in Reused(Ref): transparent wrappers and
// stateless leaves (any/never/null) are never extracted.
func countable(k nodeKind) bool {
	switch k {
	case kindUnknown, kindOptional, kindNullable, kindNullish, kindDefault, kindPrefault, kindCatch,
		kindReadonly, kindPipe, kindCodec, kindCustom:
		return false
	}
	return true
}

// mergeMeta overlays ext on base following the rules documented on JSONSchemaOptions.Metadata.
func mergeMeta(base, ext Meta) Meta {
	if ext.Title != "" {
		base.Title = ext.Title
	}
	if ext.Description != "" {
		base.Description = ext.Description
	}
	if len(ext.Examples) > 0 {
		base.Examples = ext.Examples
	}
	base.Deprecated = base.Deprecated || ext.Deprecated
	base.ReadOnly = base.ReadOnly || ext.ReadOnly
	if len(ext.Custom) > 0 {
		custom := make(map[string]any, len(base.Custom)+len(ext.Custom))
		maps.Copy(custom, base.Custom)
		maps.Copy(custom, ext.Custom)
		base.Custom = custom
	}
	return base
}

func (ctx *schemaGenCtx) generateNode(s AnySchema, meta Meta, path string) (map[string]any, error) {
	n := s.node()
	if n == nil {
		return map[string]any{}, nil
	}
	n = unwrapNode(n)

	k := n.kind()

	// Check unrepresentable kinds
	switch k {
	case kindBigInt, kindDate, kindMap, kindSet, kindTransform:
		return ctx.unrepresentable(fmt.Sprintf("node kind %v", k), path)
	}

	if !ctx.counting && ctx.opts.Metadata != nil {
		if ext, ok := ctx.opts.Metadata(s); ok {
			meta = mergeMeta(meta, ext)
		}
	}

	// name is the definition name of this node: its Meta.ID, or a generated one for reused/cyclic nodes.
	name := meta.ID
	if ctx.counting {
		if countable(k) {
			ctx.seen[n]++
			if ctx.seen[n] > 1 {
				return map[string]any{}, nil
			}
		}
	} else if name == "" && ctx.counts[n] > 1 {
		name = ctx.nameFor(n)
	}

	// Cycle / Def check: every node is tracked while it is being generated; reaching an active node again
	// closes a cycle (only possible through Lazy) and is emitted as a $ref to a definition.
	if ctx.active[n] {
		if name == "" {
			name = ctx.nameFor(n)
		}
		if ctx.opts.Cycles == CyclesThrow {
			return nil, fmt.Errorf("golang-zod: cycle detected for schema %q", name)
		}
		return ctx.makeRef(name), nil
	}
	if _, done := ctx.defs[name]; done && meta.ID == "" {
		return ctx.makeRef(name), nil
	}
	ctx.active[n] = true
	defer delete(ctx.active, n)

	res, pass, err := ctx.build(n, k, meta, path)
	if err != nil {
		return nil, err
	}
	if !pass && !ctx.counting && ctx.opts.Override != nil {
		ctx.opts.Override(s, path, res)
	}

	if name == "" {
		// A node that turned out to be cyclic was named while it was being generated.
		name = ctx.names[n]
	}
	if name == "" || (pass && meta.ID != "") {
		return res, nil
	}
	ctx.defs[name] = res
	if ctx.opts.Reused == ReusedRef {
		return ctx.makeRef(name), nil
	}
	return res, nil
}

// refSiblingsIgnored reports whether the target ignores keywords next to $ref.
func (ctx *schemaGenCtx) refSiblingsIgnored() bool {
	switch ctx.opts.Target {
	case Draft07, Draft04, OpenAPI30:
		return true
	}
	return false
}

// annotate returns a copy of sub with extra keywords added; a $ref is wrapped in allOf when the target
// ignores siblings of $ref.
func (ctx *schemaGenCtx) annotate(sub, extra map[string]any) map[string]any {
	out := maps.Clone(sub)
	if _, isRef := sub["$ref"]; isRef && ctx.refSiblingsIgnored() {
		out = map[string]any{"allOf": []any{sub}}
	}
	maps.Copy(out, extra)
	return out
}

// fieldRequired reports whether an object key whose schema has kind fk is listed in required.
func (ctx *schemaGenCtx) fieldRequired(fk nodeKind) bool {
	switch fk {
	case kindOptional, kindNullish, kindPrefault, kindCatch:
		return false
	case kindDefault:
		// A default fills the key in the output, but the input may omit it.
		return ctx.opts.IO == Output
	}
	return true
}

// bound records a numeric lower/upper bound, keeping the stricter one when the keyword is already set.
func (ctx *schemaGenCtx) bound(res map[string]any, lower, exclusive bool, v float64) {
	key, exKey := "minimum", "exclusiveMinimum"
	if !lower {
		key, exKey = "maximum", "exclusiveMaximum"
	}
	if ctx.opts.Target == Draft04 || ctx.opts.Target == OpenAPI30 {
		// exclusiveMinimum/Maximum are booleans modifying minimum/maximum.
		if old, has := res[key].(float64); has && (old == v && !exclusive || lower && old > v || !lower && old < v) {
			return
		}
		res[key] = v
		if exclusive {
			res[exKey] = true
		} else {
			delete(res, exKey)
		}
		return
	}
	if exclusive {
		key = exKey
	}
	if old, has := res[key].(float64); has && (lower && old >= v || !lower && old <= v) {
		return
	}
	res[key] = v
}

// build generates the JSON Schema of one node. pass reports that the result belongs to a wrapped schema
// (or is a final unrepresentable result) and must not be registered as a definition of the wrapper itself.
func (ctx *schemaGenCtx) build(n node, k nodeKind, meta Meta, path string) (res map[string]any, pass bool, err error) {
	res = make(map[string]any)

	// Apply Metadata
	maps.Copy(res, meta.Custom)
	if meta.Title != "" {
		res["title"] = meta.Title
	}
	if meta.Description != "" {
		res["description"] = meta.Description
	}
	if meta.Deprecated {
		res["deprecated"] = true
	}
	if meta.ReadOnly {
		res["readOnly"] = true
	}
	if len(meta.Examples) > 0 {
		res["examples"] = meta.Examples
	}

	// wrap generates the schema wrapped by a transparent wrapper node.
	wrap := func(inner AnySchema, m Meta, p string) {
		res, err = ctx.generateNode(inner, m, p)
		pass = true
	}

	switch k {
	case kindString:
		res["type"] = "string"
	case kindNumber:
		res["type"] = "number"
		if ni, ok := n.(interface{ jsonSchemaNumber() jsNumberInfo }); ok {
			info := ni.jsonSchemaNumber()
			if info.wide {
				res, err = ctx.unrepresentable("int64/uint64", path)
				return res, true, err
			}
			if info.integer {
				res["type"] = "integer"
			}
			for _, c := range info.checks {
				switch c.kind {
				case checkGT:
					ctx.bound(res, true, true, c.val)
				case checkGTE:
					ctx.bound(res, true, false, c.val)
				case checkLT:
					ctx.bound(res, false, true, c.val)
				case checkLTE:
					ctx.bound(res, false, false, c.val)
				case checkMultipleOf:
					res["multipleOf"] = c.val
				}
			}
		}
	case kindBoolean:
		res["type"] = "boolean"
	case kindBytes:
		res["type"] = "string"
		res["contentEncoding"] = "base64"
	case kindUnknown:
		// any/unknown accept everything ({}); never accepts nothing.
		if _, never := n.(*neverNode); never {
			res["not"] = map[string]any{}
		}
	case kindLiteral:
		if lit, ok := n.(interface{ jsonSchemaValues() []any }); ok {
			if vals := lit.jsonSchemaValues(); len(vals) == 1 {
				res["const"] = vals[0]
			} else if len(vals) > 1 {
				res["enum"] = vals
			}
		}
	case kindEnum:
		if en, ok := n.(interface{ Values() []string }); ok {
			vals := en.Values()
			anyVals := make([]any, len(vals))
			for i, v := range vals {
				anyVals[i] = v
			}
			res["enum"] = anyVals
		}
	case kindArray:
		res["type"] = "array"
		if arr, ok := n.(interface{ Element() AnySchema }); ok {
			elemSchema, err := ctx.generateNode(arr.Element(), arr.Element().GetMeta(), path+"[]")
			if err != nil {
				return nil, false, err
			}
			res["items"] = elemSchema
		}
	case kindTuple:
		res["type"] = "array"
		if tp, ok := n.(interface {
			jsonSchemaTuple() ([]AnySchema, AnySchema)
		}); ok {
			items, rest := tp.jsonSchemaTuple()
			prefix := make([]any, len(items))
			for i, it := range items {
				itemSchema, err := ctx.generateNode(it, it.GetMeta(), fmt.Sprintf("%s[%d]", path, i))
				if err != nil {
					return nil, false, err
				}
				prefix[i] = itemSchema
			}
			var restSchema map[string]any
			if rest != nil {
				restSchema, err = ctx.generateNode(rest, rest.GetMeta(), path+"[]")
				if err != nil {
					return nil, false, err
				}
			}
			switch {
			case len(prefix) == 0:
				if restSchema != nil {
					res["items"] = restSchema
				}
			case ctx.opts.Target == OpenAPI30:
				// OpenAPI 3.0 has neither prefixItems nor tuple-form items: every position accepts any option.
				variants := prefix
				if restSchema != nil {
					variants = append(variants, restSchema)
				}
				res["items"] = map[string]any{"anyOf": variants}
			case ctx.opts.Target == Draft202012 || ctx.opts.Target == OpenAPI31:
				res["prefixItems"] = prefix
				if restSchema != nil {
					res["items"] = restSchema
				}
			default:
				res["items"] = prefix
				if restSchema != nil {
					res["additionalItems"] = restSchema
				}
			}
			if len(items) > 0 {
				res["minItems"] = len(items)
			}
			if rest == nil {
				res["maxItems"] = len(items)
			}
		}
	case kindRecord:
		res["type"] = "object"
		if rc, ok := n.(interface{ jsonSchemaRecord() (AnySchema, AnySchema) }); ok {
			key, val := rc.jsonSchemaRecord()
			valSchema, err := ctx.generateNode(val, val.GetMeta(), path+".*")
			if err != nil {
				return nil, false, err
			}
			res["additionalProperties"] = valSchema
			if ctx.opts.Target != Draft04 && ctx.opts.Target != OpenAPI30 {
				keySchema, err := ctx.generateNode(key, key.GetMeta(), path+".#key")
				if err != nil {
					return nil, false, err
				}
				// propertyNames is only informative when the key schema is more than a plain string.
				if len(keySchema) != 1 || keySchema["type"] != "string" {
					res["propertyNames"] = keySchema
				}
			}
		}
	case kindObject, kindStruct:
		res["type"] = "object"
		props := make(map[string]any)
		var required []string

		if obj, ok := n.(interface {
			jsonSchemaObject() ([]jsField, unknownKeysMode, AnySchema)
		}); ok {
			fields, mode, catchall := obj.jsonSchemaObject()
			for _, f := range fields {
				if ctx.fieldRequired(f.schema.node().kind()) {
					required = append(required, f.name)
				}

				fOut, err := ctx.generateNode(f.schema, f.schema.GetMeta(), path+"."+f.name)
				if err != nil {
					return nil, false, err
				}
				props[f.name] = fOut
			}

			if len(props) > 0 {
				res["properties"] = props
			}
			if len(required) > 0 {
				slices.Sort(required)
				res["required"] = required
			}

			switch mode {
			case unknownKeysStrict:
				res["additionalProperties"] = false
			case unknownKeysLoose:
				// omit additionalProperties
			case unknownKeysStrip:
				if ctx.opts.IO == Output {
					res["additionalProperties"] = false
				}
			case unknownKeysCatchall:
				cOut, err := ctx.generateNode(catchall, catchall.GetMeta(), path+".*")
				if err != nil {
					return nil, false, err
				}
				res["additionalProperties"] = cOut
			}
		}
	case kindUnion:
		if u, ok := n.(interface{ Options() []AnySchema }); ok {
			var oneOf []any
			for i, opt := range u.Options() {
				sub, err := ctx.generateNode(opt, opt.GetMeta(), fmt.Sprintf("%s|%d", path, i))
				if err != nil {
					return nil, false, err
				}
				oneOf = append(oneOf, sub)
			}
			res["oneOf"] = oneOf
		}
	case kindDiscriminatedUnion:
		if u, ok := n.(interface {
			Discriminator() string
			Options() []AnySchema
		}); ok {
			var oneOf []any
			for i, opt := range u.Options() {
				sub, err := ctx.generateNode(opt, opt.GetMeta(), fmt.Sprintf("%s|%d", path, i))
				if err != nil {
					return nil, false, err
				}
				oneOf = append(oneOf, sub)
			}
			res["oneOf"] = oneOf
			if ctx.opts.Target == OpenAPI30 || ctx.opts.Target == OpenAPI31 {
				res["discriminator"] = map[string]any{"propertyName": u.Discriminator()}
			}
		}
	case kindIntersection:
		if inter, ok := n.(interface{ Operands() (AnySchema, AnySchema) }); ok {
			left, right := inter.Operands()
			leftSub, err := ctx.generateNode(left, left.GetMeta(), path+"&L")
			if err != nil {
				return nil, false, err
			}
			rightSub, err := ctx.generateNode(right, right.GetMeta(), path+"&R")
			if err != nil {
				return nil, false, err
			}
			res["allOf"] = []any{leftSub, rightSub}
		}
	case kindOptional, kindPrefault, kindCatch, kindCustom:
		// Optional only drops the key from required; Prefault/Catch only change parsing; refinements and
		// preprocessing are not expressible: all of them export the schema they wrap.
		if w, ok := n.(interface{ Unwrap() AnySchema }); ok {
			wrap(w.Unwrap(), meta, path)
		}
	case kindReadonly:
		if w, ok := n.(interface{ Unwrap() AnySchema }); ok {
			ro := meta
			ro.ReadOnly = true
			wrap(w.Unwrap(), ro, path)
		}
	case kindNullable, kindNullish:
		if _, isNull := n.(*nullNode); isNull {
			if ctx.opts.Target == OpenAPI30 {
				res["type"] = "string"
				res["nullable"] = true
				res["enum"] = []any{nil}
			} else {
				res["type"] = "null"
			}
		} else if nl, ok := n.(interface{ Unwrap() AnySchema }); ok {
			inner, err := ctx.generateNode(nl.Unwrap(), meta, path)
			if err != nil {
				return nil, false, err
			}
			if ctx.opts.Target == OpenAPI30 {
				return ctx.annotate(inner, map[string]any{"nullable": true}), true, nil
			}
			if tStr, ok := inner["type"].(string); ok {
				res = maps.Clone(inner)
				res["type"] = []any{tStr, "null"}
				return res, true, nil
			}
			return map[string]any{
				"anyOf": []any{
					inner,
					map[string]any{"type": "null"},
				},
			}, true, nil
		}
	case kindDefault:
		if df, ok := n.(interface {
			Unwrap() AnySchema
			jsonSchemaDefault() any
		}); ok {
			wrap(df.Unwrap(), meta, path)
			if err != nil {
				return nil, false, err
			}
			if d, ok := n.(interface{ isDynamicDefault() bool }); ok && d.isDynamicDefault() {
				// Supplier-based defaults (e.g. generated UUIDs) have no stable value to export.
			} else if v := df.jsonSchemaDefault(); jsonRepresentable(v) {
				res = ctx.annotate(res, map[string]any{"default": v})
			} else if ctx.opts.Unrepresentable == UnrepresentableThrow {
				return nil, false, fmt.Errorf("golang-zod: unrepresentable default value at %q", path)
			}
		}
	case kindLazy:
		if lz, ok := n.(interface{ Resolve() AnySchema }); ok {
			resolved := lz.Resolve()
			wrap(resolved, resolved.GetMeta(), path)
		}
	case kindPipe, kindCodec:
		if p, ok := n.(interface{ Schemas() (AnySchema, AnySchema) }); ok {
			inSch, outSch := p.Schemas()
			targetSch := outSch
			// For the input side, a leading transform contributes nothing: use the schema after it.
			if ctx.opts.IO == Input && unwrapNode(inSch.node()).kind() != kindTransform {
				targetSch = inSch
			}
			// A codec built without an output schema exports as an unconstrained schema.
			if targetSch != nil {
				wrap(targetSch, targetSch.GetMeta(), path)
			}
		}
	}

	return res, pass, err
}

// jsonRepresentable reports whether v can be serialized as a JSON value.
func jsonRepresentable(v any) bool {
	_, err := json.Marshal(v)
	return err == nil
}

func (ctx *schemaGenCtx) makeRef(id string) map[string]any {
	prefix := "#/$defs/"
	switch ctx.opts.Target {
	case Draft07, Draft04:
		prefix = "#/definitions/"
	case OpenAPI30, OpenAPI31:
		prefix = "#/components/schemas/"
	}
	uri := prefix + id
	if ctx.opts.URI != nil {
		uri = ctx.opts.URI(id)
	}
	return map[string]any{"$ref": uri}
}

// jsField is an object key and its schema as seen by the exporter.
type jsField struct {
	name   string
	schema AnySchema
}

// jsNumberInfo describes a number node for the exporter.
type jsNumberInfo struct {
	integer bool
	wide    bool // int64/uint64: not representable (REQ-EXP-004)
	checks  []jsNumberCheck
}

type jsNumberCheck struct {
	kind numCheckKind
	val  float64
}

func (n *numberNode[T]) jsonSchemaNumber() jsNumberInfo {
	var zero T
	_, isInt64 := any(zero).(int64)
	_, isUint64 := any(zero).(uint64)
	info := jsNumberInfo{integer: isIntegerKind[T](), wide: isInt64 || isUint64}
	for _, c := range n.schema.checks {
		info.checks = append(info.checks, jsNumberCheck{kind: c.kind, val: float64(c.val)})
	}
	return info
}

func (n *coercedFloatNode) jsonSchemaNumber() jsNumberInfo {
	return n.inner.(interface{ jsonSchemaNumber() jsNumberInfo }).jsonSchemaNumber()
}

func (n *coercedIntNode) jsonSchemaNumber() jsNumberInfo {
	return n.inner.(interface{ jsonSchemaNumber() jsNumberInfo }).jsonSchemaNumber()
}

func (n *objectNode) jsonSchemaObject() ([]jsField, unknownKeysMode, AnySchema) {
	fields := make([]jsField, len(n.schema.fields))
	for i, f := range n.schema.fields {
		fields[i] = jsField(f)
	}
	return fields, n.Mode(), n.CatchallSchema()
}

func (n *structNode[T]) jsonSchemaObject() ([]jsField, unknownKeysMode, AnySchema) {
	fields := make([]jsField, len(n.schema.fields))
	fields = fields[:0]
	for _, f := range n.schema.fields {
		if f.inline {
			if obj, ok := f.schema.node().(interface {
				jsonSchemaObject() ([]jsField, unknownKeysMode, AnySchema)
			}); ok {
				sub, _, _ := obj.jsonSchemaObject()
				fields = append(fields, sub...)
			}
			continue
		}
		fields = append(fields, jsField{name: f.jsonName, schema: f.schema})
	}
	mode := unknownKeysStrip
	if n.schema.isStrict {
		mode = unknownKeysStrict
	}
	return fields, mode, nil
}

func (n *tupleNode) jsonSchemaTuple() ([]AnySchema, AnySchema) {
	return n.schema.items, n.schema.rest
}

func (n *recordNode[K, V]) jsonSchemaRecord() (AnySchema, AnySchema) {
	return n.schema.keySchema, n.schema.valSchema
}

func (n *defaultNode[T]) jsonSchemaDefault() any { return n.defaultFn() }

func (n *pipeNode[T, U]) Schemas() (AnySchema, AnySchema) { return n.schema.left, n.schema.right }

func (n *refinedNode[T]) Unwrap() AnySchema { return n.schema.inner }

func (n *preprocessNode[T]) Unwrap() AnySchema { return n.schema.inner }

// jsonSchemaValues exposes the literal's accepted values to the JSON Schema exporter.
func (n *literalNode[T]) jsonSchemaValues() []any {
	vals := make([]any, len(n.values))
	for i, v := range n.values {
		vals[i] = v
	}
	return vals
}
