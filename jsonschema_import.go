package z

import (
	"encoding/json/v2"
	"fmt"
)

// @spec REQ-EXP-005
// FromJSONSchema dynamically reconstructs a schema from a JSON Schema specification.
func FromJSONSchema(data []byte) (Schema[any], error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("golang-zod: failed to unmarshal JSON Schema: %w", err)
	}

	sch, err := parseJSONSchemaMap(raw)
	if err != nil {
		return nil, err
	}

	res := NewBaseSchema[any](sch.node())
	res.SetMeta(sch.GetMeta())
	return res, nil
}

func parseJSONSchemaMap(raw map[string]any) (AnySchema, error) {
	if constVal, hasConst := raw["const"]; hasConst {
		return Literal(constVal), nil
	}

	if enumVals, hasEnum := raw["enum"].([]any); hasEnum {
		var strVals []string
		for _, ev := range enumVals {
			if s, ok := ev.(string); ok {
				strVals = append(strVals, s)
			}
		}
		if len(strVals) == len(enumVals) {
			return Enum(strVals...), nil
		}
	}

	if oneOf, hasOneOf := raw["oneOf"].([]any); hasOneOf {
		var opts []AnySchema
		for _, o := range oneOf {
			if oMap, ok := o.(map[string]any); ok {
				sub, err := parseJSONSchemaMap(oMap)
				if err != nil {
					return nil, err
				}
				opts = append(opts, sub)
			}
		}
		return Union(opts...), nil
	}

	if allOf, hasAllOf := raw["allOf"].([]any); hasAllOf {
		if len(allOf) == 2 {
			lMap, lOk := allOf[0].(map[string]any)
			rMap, rOk := allOf[1].(map[string]any)
			if lOk && rOk {
				left, err := parseJSONSchemaMap(lMap)
				if err != nil {
					return nil, err
				}
				right, err := parseJSONSchemaMap(rMap)
				if err != nil {
					return nil, err
				}
				return Intersection(left, right), nil
			}
		}
	}

	tVal, hasType := raw["type"]
	if !hasType {
		return Any(), nil
	}

	switch t := tVal.(type) {
	case string:
		switch t {
		case "string":
			return String(), nil
		case "integer":
			return Int(), nil
		case "number":
			return Float64(), nil
		case "boolean":
			return Boolean(), nil
		case "array":
			if items, ok := raw["items"].(map[string]any); ok {
				elem, err := parseJSONSchemaMap(items)
				if err != nil {
					return nil, err
				}
				elemSch := NewBaseSchema[any](elem.node())
				elemSch.SetMeta(elem.GetMeta())
				return Array[any](elemSch), nil
			}
			return Array[any](Any()), nil
		case "object":
			return parseObjectSchemaFromMap(raw)
		}
	case []any:
		// Nullable type representation, e.g. ["string", "null"]
		for _, subType := range t {
			if subStr, ok := subType.(string); ok && subStr != "null" {
				inner, err := parseJSONSchemaMap(map[string]any{"type": subStr})
				if err != nil {
					return nil, err
				}
				return inner, nil
			}
		}
	}

	return Any(), nil
}

func parseObjectSchemaFromMap(raw map[string]any) (AnySchema, error) {
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
				fieldSch, err := parseJSONSchemaMap(pMap)
				if err == nil {
					if _, isReq := requiredMap[k]; !isReq {
						s.Field(k, makeAnyOptional(fieldSch))
					} else {
						s.Field(k, fieldSch)
					}
				}
			}
		}
	})

	if addProps, ok := raw["additionalProperties"].(bool); ok && !addProps {
		return obj.Strict(), nil
	}

	return obj, nil
}
