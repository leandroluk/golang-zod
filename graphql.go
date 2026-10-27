package z

import (
	"fmt"
	"slices"
	"strings"
)

// @spec REQ-GQL-001
// GraphQLOptions configures GraphQL SDL generation.
type GraphQLOptions struct {
	IsInput bool
}

// GraphQLOption configures ToGraphQLSDL options.
type GraphQLOption func(*GraphQLOptions)

// GraphQLInput configures the generator to emit 'input' object types rather than 'type'.
func GraphQLInput() GraphQLOption {
	return func(o *GraphQLOptions) {
		o.IsInput = true
	}
}

type gqlGenCtx struct {
	opts         GraphQLOptions
	types        map[string]string
	visitedNodes map[any]string
}

// @spec REQ-GQL-001
// @spec REQ-GQL-002
// @spec REQ-GQL-003
// @spec REQ-GQL-004
// @spec REQ-GQL-005
// ToGraphQLSDL generates a GraphQL Schema Definition Language (SDL) string from a schema.
func ToGraphQLSDL(s AnySchema, opts ...GraphQLOption) (string, error) {
	cfg := GraphQLOptions{}
	for _, o := range opts {
		o(&cfg)
	}

	ctx := &gqlGenCtx{
		opts:         cfg,
		types:        make(map[string]string),
		visitedNodes: make(map[any]string),
	}

	typeName := "Query"
	if s.GetMeta().ID != "" {
		typeName = s.GetMeta().ID
	} else if cfg.IsInput {
		typeName = "Input"
	}

	typeRef, err := ctx.generateGQL(s.node(), s.GetMeta(), typeName, true)
	if err != nil {
		return "", err
	}

	_ = typeRef

	// Collect and sort all generated type definitions for deterministic output
	var keys []string
	for k := range ctx.types {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(ctx.types[k])
	}

	return sb.String(), nil
}

func (ctx *gqlGenCtx) generateGQL(n node, meta Meta, nameHint string, required bool) (string, error) {
	if n == nil {
		return "String", nil
	}

	k := n.kind()

	switch k {
	case kindTuple:
		return "", fmt.Errorf("golang-zod: heterogeneous Tuple has no GraphQL equivalent")
	case kindOptional, kindNullable, kindNullish:
		if opt, ok := n.(interface{ Unwrap() AnySchema }); ok {
			// Inverted nullability: optional/nullable fields in GraphQL do NOT have the '!' suffix
			return ctx.generateGQL(opt.Unwrap().node(), meta, nameHint, false)
		}
	case kindString:
		return formatReq("String", required), nil
	case kindNumber:
		return formatReq("Float", required), nil
	case kindBoolean:
		return formatReq("Boolean", required), nil
	case kindDate:
		ctx.types["scalar DateTime"] = "scalar DateTime"
		return formatReq("DateTime", required), nil
	case kindBigInt:
		ctx.types["scalar BigInt"] = "scalar BigInt"
		return formatReq("BigInt", required), nil
	case kindBytes:
		ctx.types["scalar Bytes"] = "scalar Bytes"
		return formatReq("Bytes", required), nil
	case kindEnum:
		enumName := nameHint
		if meta.ID != "" {
			enumName = meta.ID
		}
		if en, ok := n.(interface{ Values() []string }); ok {
			vals := en.Values()
			var valLines []string
			for _, v := range vals {
				valLines = append(valLines, "  "+v)
			}
			ctx.types[enumName] = fmt.Sprintf("enum %s {\n%s\n}", enumName, strings.Join(valLines, "\n"))
		}
		return formatReq(enumName, required), nil
	case kindArray:
		if arr, ok := n.(interface{ Element() AnySchema }); ok {
			// List elements are required by default: [T!]!
			elemRef, err := ctx.generateGQL(arr.Element().node(), arr.Element().GetMeta(), nameHint+"Item", true)
			if err != nil {
				return "", err
			}
			return formatReq(fmt.Sprintf("[%s]", elemRef), required), nil
		}
		return formatReq("[String!]", required), nil
	case kindObject:
		objName := nameHint
		if meta.ID != "" {
			objName = meta.ID
		}

		if existing, ok := ctx.visitedNodes[n]; ok {
			return formatReq(existing, required), nil
		}
		ctx.visitedNodes[n] = objName

		keyword := "type"
		if ctx.opts.IsInput {
			keyword = "input"
		}

		if obj, ok := n.(interface {
			FieldNames() []string
			FieldSchema(string) AnySchema
		}); ok {
			var fieldLines []string
			for _, fn := range obj.FieldNames() {
				fSch := obj.FieldSchema(fn)
				fNode := fSch.node()
				fk := fNode.kind()

				// Required unless optional/nullable
				fReq := fk != kindOptional && fk != kindNullish && fk != kindNullable
				fRef, err := ctx.generateGQL(fNode, fSch.GetMeta(), objName+strings.Title(fn), fReq)
				if err != nil {
					return "", err
				}
				fieldLines = append(fieldLines, fmt.Sprintf("  %s: %s", fn, fRef))
			}

			ctx.types[objName] = fmt.Sprintf("%s %s {\n%s\n}", keyword, objName, strings.Join(fieldLines, "\n"))
			return formatReq(objName, required), nil
		}
	case kindLazy:
		if lz, ok := n.(interface{ Resolve() AnySchema }); ok {
			resolved := lz.Resolve()
			return ctx.generateGQL(resolved.node(), resolved.GetMeta(), nameHint, required)
		}
	}

	return formatReq("String", required), nil
}

func formatReq(t string, req bool) string {
	if req {
		return t + "!"
	}
	return t
}
