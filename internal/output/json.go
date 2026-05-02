package output

import (
	"encoding/json"
	"fmt"
)

// FilterFields keeps only the selected top-level fields on an object or object list.
func FilterFields(value any, fields []string) (any, error) {
	if len(fields) == 0 {
		return value, nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	allowed := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		allowed[field] = struct{}{}
	}

	switch v := decoded.(type) {
	case map[string]any:
		return filterMap(v, allowed), nil
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("field filtering requires object values")
			}
			out[i] = filterMap(m, allowed)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("field filtering requires object values")
	}
}

func filterMap(input map[string]any, allowed map[string]struct{}) map[string]any {
	out := make(map[string]any, len(allowed))
	for key, value := range input {
		if _, ok := allowed[key]; ok {
			out[key] = value
		}
	}
	return out
}
