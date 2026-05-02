package cli

import (
	"strings"
	"testing"
)

func TestReadBodyInputFromString(t *testing.T) {
	body, ok, err := readBodyInput(bodyFlags{body: "hello"}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("readBodyInput error = %v", err)
	}
	if !ok || body != "hello" {
		t.Fatalf("body=%q ok=%v", body, ok)
	}
}

func TestReadBodyInputMutuallyExclusive(t *testing.T) {
	_, _, err := readBodyInput(bodyFlags{body: "hello", bodyFile: "-"}, strings.NewReader("stdin"))
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "--body and --body-file are mutually exclusive" {
		t.Fatalf("error = %v", err)
	}
}
