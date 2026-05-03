package cli

import (
	"os"
	"path/filepath"
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

func TestReadBodyInputFromFileAndEmpty(t *testing.T) {
	body, ok, err := readBodyInput(bodyFlags{}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("readBodyInput empty error = %v", err)
	}
	if ok || body != "" {
		t.Fatalf("empty body=%q ok=%v", body, ok)
	}

	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte("from file"), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	body, ok, err = readBodyInput(bodyFlags{bodyFile: path}, strings.NewReader(""))
	if err != nil {
		t.Fatalf("readBodyInput file error = %v", err)
	}
	if !ok || body != "from file" {
		t.Fatalf("file body=%q ok=%v", body, ok)
	}

	body, ok, err = readBodyInput(bodyFlags{bodyFile: "-"}, strings.NewReader("from stdin"))
	if err != nil {
		t.Fatalf("readBodyInput stdin error = %v", err)
	}
	if !ok || body != "from stdin" {
		t.Fatalf("stdin body=%q ok=%v", body, ok)
	}
}
