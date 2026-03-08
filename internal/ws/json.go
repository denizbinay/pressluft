package ws

import "encoding/json"

func marshalJSON(v any) ([]byte, error) {
	if raw, ok := v.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(v)
}
