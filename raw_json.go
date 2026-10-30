package z

import (
	stdjson "encoding/json"
	"fmt"
)

// RawJSON validates any JSON value and yields it as the encoded stdlib json.RawMessage, for
// fields such as `Payload json.RawMessage`. Parsing accepts decoded JSON values (maps, slices,
// numbers, strings, bool, nil) as well as an already-encoded json.RawMessage or []byte holding
// valid JSON; Encode decodes the raw message back into a generic value.
func RawJSON() *CodecSchema[any, stdjson.RawMessage] {
	in := CustomFn[any](func(v any) bool {
		switch t := v.(type) {
		case stdjson.RawMessage:
			return stdjson.Valid(t)
		case []byte:
			return stdjson.Valid(t)
		}
		return isJSONValue(v)
	}, "Invalid JSON value")
	return Codec[any, stdjson.RawMessage](in, nil,
		func(v any) (stdjson.RawMessage, error) {
			switch t := v.(type) {
			case stdjson.RawMessage:
				return append(stdjson.RawMessage(nil), t...), nil
			case []byte:
				return append(stdjson.RawMessage(nil), t...), nil
			}
			b, err := stdjson.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("invalid JSON value: %w", err)
			}
			return b, nil
		},
		func(raw stdjson.RawMessage) (any, error) {
			var v any
			if err := stdjson.Unmarshal(raw, &v); err != nil {
				return nil, err
			}
			return v, nil
		},
	)
}
