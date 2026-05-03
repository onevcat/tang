package cli

import "testing"

func TestFormatConfigValueAndRKeyFromURI(t *testing.T) {
	if got := formatConfigValue([]string{"a", "b"}); got != "a,b" {
		t.Fatalf("format slice = %q", got)
	}
	if got := formatConfigValue("value"); got != "value" {
		t.Fatalf("format string = %q", got)
	}
	if got := rkeyFromURI("at://did:plc:a/sh.tangled.publicKey/key1"); got != "key1" {
		t.Fatalf("rkey = %q", got)
	}
}

func TestOptionalCLIString(t *testing.T) {
	if optionalCLIString("") != nil {
		t.Fatal("empty optional string should be nil")
	}
	if got := optionalCLIString("value"); got == nil || *got != "value" {
		t.Fatalf("optional string = %#v", got)
	}
}
