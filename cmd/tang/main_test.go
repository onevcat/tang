package main

import (
	"os"
	"testing"
)

func TestMainCommandRunsVersion(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"tang", "version"}
	t.Cleanup(func() { os.Args = oldArgs })
	main()
}
