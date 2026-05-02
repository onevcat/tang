package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{Version: "v0.1.0", Commit: "abc123"})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "version"); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "v0.1.0") || !strings.Contains(got, "abc123") {
		t.Fatalf("version output = %q", got)
	}
}

func TestHelpIncludesCoreCommands(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{Version: "dev", Commit: "none"})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "--help"); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"version", "completion"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}
