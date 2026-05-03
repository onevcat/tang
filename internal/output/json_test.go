package output

import (
	"bytes"
	"encoding/json"
	"strings"
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

func TestFilterFieldsRejectsNonObjects(t *testing.T) {
	if _, err := FilterFields("value", []string{"title"}); err == nil {
		t.Fatal("expected scalar filter error")
	}
	if _, err := FilterFields([]string{"value"}, []string{"title"}); err == nil {
		t.Fatal("expected list element filter error")
	}
	if _, err := FilterFields(func() {}, []string{"title"}); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestJSONRendererParsesAndRendersFields(t *testing.T) {
	renderer := NewJSONRenderer(" title, number, ")
	var out bytes.Buffer
	err := renderer.Render(&out, map[string]any{
		"title":  "Fix parser",
		"number": 2,
		"state":  "open",
	})
	if err != nil {
		t.Fatalf("Render error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, `"title": "Fix parser"`) || !strings.Contains(got, `"number": 2`) || strings.Contains(got, "state") {
		t.Fatalf("rendered JSON = %s", got)
	}
}

func TestTextRendererWritesLine(t *testing.T) {
	var out bytes.Buffer
	if err := (TextRenderer{}).Render(&out, "hello"); err != nil {
		t.Fatalf("Render error = %v", err)
	}
	if out.String() != "hello\n" {
		t.Fatalf("text output = %q", out.String())
	}
}
