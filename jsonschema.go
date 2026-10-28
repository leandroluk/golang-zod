package z

import (
	"encoding/json/v2"
	"fmt"
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

// JSONSchemaOptions configures ToJSONSchema generation.
type JSONSchemaOptions struct {
	Target          JSONSchemaTarget
	IO              SchemaIO
	Unrepresentable UnrepresentableAction
	Cycles          CyclesAction
	Reused          ReusedAction
	URI             func(id string) string
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

func WithReused(r ReusedAction) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.Reused = r }
}

func WithURI(fn func(id string) string) JSONSchemaOption {
	return func(o *JSONSchemaOptions) { o.URI = fn }
}

type schemaGenCtx struct {
	opts    JSONSchemaOptions
	defs    map[string]map[string]any
	visiting map[any]string
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

	ctx := &schemaGenCtx{
		opts:     cfg,
		defs:     make(map[string]map[string]any),
		visiting: make(map[any]string),
	}

	root, err := ctx.generateNode(s.node(), s.GetMeta(), "")
	if err != nil {
		return nil, err
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

		// Prevent recursive cycle if root was registered in defs
		for id, def := range ctx.defs {
			delete(def, "$defs")
			delete(def, "definitions")
			delete(def, "components")
			delete(def, "$schema")
			ctx.defs[id] = def
		}

		// If root itself was registered in defs with matching pointer
		cleanRootDef := make(map[string]any, len(root))
		for k, v := range root {
			if k != "$schema" && k != "$defs" && k != "definitions" && k != "components" {
				cleanRootDef[k] = v
			}
		}
		for id, def := range ctx.defs {
			if fmt.Sprintf("%p", def) == fmt.Sprintf("%p", root) {
				ctx.defs[id] = cleanRootDef
			}
		}

		if cfg.Target == OpenAPI30 {
			root[defsKey] = map[string]any{"schemas": ctx.defs}
		} else {
			root[defsKey] = ctx.defs
		}
	}

	return json.Marshal(root)
}

func (ctx *schemaGenCtx) generateNode(n node, meta Meta, path string) (map[string]any, error) {
	if n == nil {
		return map[string]any{}, nil
	}
	n = unwrapNode(n)

	k := n.kind()

	// Check unrepresentable kinds
	switch k {
	case kindBigInt, kindDate, kindMap, kindSet, kindTransform:
		if ctx.opts.Unrepresentable == UnrepresentableThrow {
			return nil, fmt.Errorf("golang-zod: unrepresentable node kind %v at %q", k, path)
		}
		return map[string]any{}, nil
	}

	// Cycle / Def check if ID is present
	if meta.ID != "" {
		if refName, active := ctx.visiting[n]; active {
			if ctx.opts.Cycles == CyclesThrow {
				return nil, fmt.Errorf("golang-zod: cycle detected for schema %q", meta.ID)
			}
			return ctx.makeRef(refName), nil
		}
		ctx.visiting[n] = meta.ID
		defer delete(ctx.visiting, n)
	}

	res := make(map[string]any)

	// Apply Metadata
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

	switch k {
	case kindString:
		res["type"] = "string"
	case kindNumber:
		res["type"] = "number"
	case kindBoolean:
		res["type"] = "boolean"
	case kindBytes:
		res["type"] = "string"
		res["contentEncoding"] = "base64"
	case kindLiteral:
		if lit, ok := n.(interface{ Value() any }); ok {
			res["const"] = lit.Value()
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
			elemSchema, err := ctx.generateNode(arr.Element().node(), arr.Element().GetMeta(), path+"[]")
			if err != nil {
				return nil, err
			}
			res["items"] = elemSchema
		}
	case kindObject:
		res["type"] = "object"
		props := make(map[string]any)
		var required []string

		if obj, ok := n.(interface {
			FieldNames() []string
			FieldSchema(string) AnySchema
			Mode() unknownKeysMode
			CatchallSchema() AnySchema
		}); ok {
			for _, name := range obj.FieldNames() {
				fSch := obj.FieldSchema(name)
				fNode := fSch.node()
				fk := fNode.kind()

				// Determine requiredness: non-optional is required
				if fk != kindOptional && fk != kindNullish {
					required = append(required, name)
				}

				fOut, err := ctx.generateNode(fNode, fSch.GetMeta(), path+"."+name)
				if err != nil {
					return nil, err
				}
				props[name] = fOut
			}

			if len(props) > 0 {
				res["properties"] = props
			}
			if len(required) > 0 {
				slices.Sort(required)
				res["required"] = required
			}

			mode := obj.Mode()
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
				cSch := obj.CatchallSchema()
				if cSch != nil {
					cOut, err := ctx.generateNode(cSch.node(), cSch.GetMeta(), path+".*")
					if err != nil {
						return nil, err
					}
					res["additionalProperties"] = cOut
				}
			}
		}
	case kindUnion:
		if u, ok := n.(interface{ Options() []AnySchema }); ok {
			var oneOf []any
			for i, opt := range u.Options() {
				sub, err := ctx.generateNode(opt.node(), opt.GetMeta(), fmt.Sprintf("%s|%d", path, i))
				if err != nil {
					return nil, err
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
				sub, err := ctx.generateNode(opt.node(), opt.GetMeta(), fmt.Sprintf("%s|%d", path, i))
				if err != nil {
					return nil, err
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
			leftSub, err := ctx.generateNode(left.node(), left.GetMeta(), path+"&L")
			if err != nil {
				return nil, err
			}
			rightSub, err := ctx.generateNode(right.node(), right.GetMeta(), path+"&R")
			if err != nil {
				return nil, err
			}
			res["allOf"] = []any{leftSub, rightSub}
		}
	case kindOptional:
		if opt, ok := n.(interface{ Unwrap() AnySchema }); ok {
			return ctx.generateNode(opt.Unwrap().node(), meta, path)
		}
	case kindNullable:
		if nl, ok := n.(interface{ Unwrap() AnySchema }); ok {
			sub, err := ctx.generateNode(nl.Unwrap().node(), meta, path)
			if err != nil {
				return nil, err
			}
			if ctx.opts.Target == OpenAPI30 {
				sub["nullable"] = true
				return sub, nil
			}
			if tStr, ok := sub["type"].(string); ok {
				sub["type"] = []any{tStr, "null"}
				return sub, nil
			}
			return map[string]any{
				"anyOf": []any{
					sub,
					map[string]any{"type": "null"},
				},
			}, nil
		}
	case kindDefault:
		if df, ok := n.(interface{ Unwrap() AnySchema }); ok {
			return ctx.generateNode(df.Unwrap().node(), meta, path)
		}
	case kindLazy:
		if lz, ok := n.(interface{ Resolve() AnySchema }); ok {
			resolved := lz.Resolve()
			return ctx.generateNode(resolved.node(), resolved.GetMeta(), path)
		}
	case kindCodec:
		if cd, ok := n.(interface{ Schemas() (AnySchema, AnySchema) }); ok {
			inSch, outSch := cd.Schemas()
			targetSch := outSch
			if ctx.opts.IO == Input {
				targetSch = inSch
			}
			if targetSch != nil {
				return ctx.generateNode(targetSch.node(), targetSch.GetMeta(), path)
			}
		}
	default:
		res["type"] = "object"
	}

	if meta.ID != "" {
		ctx.defs[meta.ID] = res
		if ctx.opts.Reused == ReusedRef {
			return ctx.makeRef(meta.ID), nil
		}
	}

	return res, nil
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
