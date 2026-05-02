package output

import (
	"encoding/json"
	"testing"
)

func TestFilterFieldsObject(t *testing.T) {
	input := map[string]any{
		"number": 1,
		"title":  "Fix parser",
		"state":  "open",
	}
	got, err := FilterFields(input, []string{"title", "number"})
	if err != nil {
		t.Fatalf("FilterFields returned error: %v", err)
	}

	raw, _ := json.Marshal(got)
	if string(raw) != `{"number":1,"title":"Fix parser"}` {
		t.Fatalf("filtered JSON = %s", raw)
	}
}

func TestFilterFieldsList(t *testing.T) {
	input := []map[string]any{
		{"number": 1, "title": "One", "state": "open"},
		{"number": 2, "title": "Two", "state": "closed"},
	}
	got, err := FilterFields(input, []string{"title"})
	if err != nil {
		t.Fatalf("FilterFields returned error: %v", err)
	}

	raw, _ := json.Marshal(got)
	if string(raw) != `[{"title":"One"},{"title":"Two"}]` {
		t.Fatalf("filtered JSON = %s", raw)
	}
}
