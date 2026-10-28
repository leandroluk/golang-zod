package z

import (
	"encoding/json"
	"encoding/json/jsontext"
	"strconv"
	"strings"
)

// @spec REQ-OPT-002
// @spec REQ-OPT-003
// decodeJSON reads one JSON value from dec token by token and builds the untyped input tree
// (map[string]any, []any, string, bool, nil, float64 or json.Number).
//
// Numbers are taken from the token text: values that a float64 represents exactly stay float64,
// every other integer (not exactly representable) is kept as json.Number so Int64, Uint64 and BigInt never lose
// precision through float64.
func decodeJSON(dec *jsontext.Decoder) (any, error) {
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	return decodeJSONToken(dec, tok)
}

func decodeJSONToken(dec *jsontext.Decoder, tok jsontext.Token) (any, error) {
	switch tok.Kind() {
	case 'n':
		return nil, nil
	case 't':
		return true, nil
	case 'f':
		return false, nil
	case '"':
		return tok.String(), nil
	case '0':
		return decodeJSONNumber(tok.String())
	case '{':
		obj := make(map[string]any)
		for dec.PeekKind() != '}' {
			kt, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			key := kt.String() // tokens are only valid until the next read
			vt, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			v, err := decodeJSONToken(dec, vt)
			if err != nil {
				return nil, err
			}
			obj[key] = v
		}
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		return obj, nil
	default: // '['
		arr := make([]any, 0)
		for dec.PeekKind() != ']' {
			vt, err := dec.ReadToken()
			if err != nil {
				return nil, err
			}
			v, err := decodeJSONToken(dec, vt)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		return arr, nil
	}
}

func decodeJSONNumber(text string) (any, error) {
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, err
	}
	if !strings.ContainsAny(text, ".eE") && strconv.FormatFloat(f, 'f', -1, 64) != text {
		return json.Number(text), nil
	}
	return f, nil
}
