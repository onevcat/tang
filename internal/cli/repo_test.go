package cli

import "testing"

func TestIsExplicitCloneURL(t *testing.T) {
	tests := map[string]bool{
		"git@tangled.org:onev.cat/tang":       true,
		"https://tangled.org/onev.cat/tang":   true,
		"ssh://git@tangled.org/onev.cat/tang": true,
		"onev.cat/tang":                       false,
		"did:plc:abc/tang":                    false,
	}
	for input, want := range tests {
		if got := isExplicitCloneURL(input); got != want {
			t.Fatalf("isExplicitCloneURL(%q) = %v, want %v", input, got, want)
		}
	}
}
