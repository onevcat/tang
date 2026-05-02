// Package output renders command results.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Mode describes the output format selected for a command.
type Mode string

const (
	// ModeText renders human-readable output.
	ModeText Mode = "text"
	// ModeJSON renders JSON output.
	ModeJSON Mode = "json"
)

// Renderer writes command results to an output stream.
type Renderer interface {
	Render(w io.Writer, value any) error
}

// JSONRenderer renders values as JSON with optional top-level field filtering.
type JSONRenderer struct {
	Fields []string
}

// NewJSONRenderer creates a JSON renderer from a comma-separated field list.
func NewJSONRenderer(fields string) JSONRenderer {
	return JSONRenderer{Fields: parseFields(fields)}
}

// Render writes filtered JSON.
func (r JSONRenderer) Render(w io.Writer, value any) error {
	filtered, err := FilterFields(value, r.Fields)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(filtered)
}

// TextRenderer renders values with fmt.Fprintln.
type TextRenderer struct{}

// Render writes a human-readable value.
func (TextRenderer) Render(w io.Writer, value any) error {
	_, err := fmt.Fprintln(w, value)
	return err
}

func parseFields(fields string) []string {
	if strings.TrimSpace(fields) == "" {
		return nil
	}
	parts := strings.Split(fields, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}
