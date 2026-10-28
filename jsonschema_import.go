package z

import (
	"encoding/json/v2"
	"fmt"
)

// @spec REQ-EXP-005
// FromJSONSchema dynamically reconstructs a schema from a JSON Schema specification.
// Unsupported or unknown constructs degrade to z.Any() rather than failing; only malformed JSON is an error.
func FromJSONSchema(data []byte) (Schema[any], error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("golang-zod: failed to unmarshal JSON Schema: %w", err)
	}

	sch := parseJSONSchemaMap(raw)

	res := NewBaseSchema[any](sch.node())
	res.SetMeta(sch.GetMeta())
	return res, nil
}

func parseJSONSchemaMap(raw map[string]any) AnySchema {
	if constVal, hasConst := raw["const"]; hasConst {
		return Literal(constVal)
	}

	if enumVals, hasEnum := raw["enum"].([]any); hasEnum {
		var strVals []string
		for _, ev := range enumVals {
			if s, ok := ev.(string); ok {
				strVals = append(strVals, s)
			}
		}
		if len(strVals) == len(enumVals) {
			return Enum(strVals...)
		}
	}

	if oneOf, hasOneOf := raw["oneOf"].([]any); hasOneOf {
		var opts []AnySchema
		for _, o := range oneOf {
			if oMap, ok := o.(map[string]any); ok {
				opts = append(opts, parseJSONSchemaMap(oMap))
			}
		}
		return Union(opts...)
	}

	if allOf, hasAllOf := raw["allOf"].([]any); hasAllOf && len(allOf) == 2 {
		lMap, lOk := allOf[0].(map[string]any)
		rMap, rOk := allOf[1].(map[string]any)
		if lOk && rOk {
			return Intersection(parseJSONSchemaMap(lMap), parseJSONSchemaMap(rMap))
		}
	}

	switch t := raw["type"].(type) {
	case string:
		switch t {
		case "string":
			return String()
		case "integer":
			return Int()
		case "number":
			return Float64()
		case "boolean":
			return Boolean()
		case "array":
			if items, ok := raw["items"].(map[string]any); ok {
				elem := parseJSONSchemaMap(items)
				elemSch := NewBaseSchema[any](elem.node())
				elemSch.SetMeta(elem.GetMeta())
				return Array[any](elemSch)
			}
			return Array[any](Any())
		case "object":
			return parseObjectSchemaFromMap(raw)
		}
	case []any:
		// Nullable type representation, e.g. ["string", "null"]: the first non-null type wins.
		for _, subType := range t {
			if subStr, ok := subType.(string); ok && subStr != "null" {
				return parseJSONSchemaMap(map[string]any{"type": subStr})
			}
		}
	}

	return Any()
}

func parseObjectSchemaFromMap(raw map[string]any) AnySchema {
	props, _ := raw["properties"].(map[string]any)
	reqSlice, _ := raw["required"].([]any)
	requiredMap := make(map[string]struct{})
	for _, r := range reqSlice {
		if s, ok := r.(string); ok {
			requiredMap[s] = struct{}{}
		}
	}

	obj := Object(func(s *Shape) {
		for k, v := range props {
			if pMap, ok := v.(map[string]any); ok {
				fieldSch := parseJSONSchemaMap(pMap)
				if _, isReq := requiredMap[k]; !isReq {
					s.Field(k, makeAnyOptional(fieldSch))
				} else {
					s.Field(k, fieldSch)
				}
			}
		}
	})

	if addProps, ok := raw["additionalProperties"].(bool); ok && !addProps {
		return obj.Strict()
	}

	return obj
}
