package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type bodyFlags struct {
	body     string
	bodyFile string
}

func readBodyInput(flags bodyFlags, stdin io.Reader) (string, bool, error) {
	count := 0
	if flags.body != "" {
		count++
	}
	if flags.bodyFile != "" {
		count++
	}
	if count > 1 {
		return "", false, errors.New("--body and --body-file are mutually exclusive")
	}
	if flags.body != "" {
		return flags.body, true, nil
	}
	if flags.bodyFile == "" {
		return "", false, nil
	}
	var data []byte
	var err error
	if flags.bodyFile == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(flags.bodyFile) // #nosec G304 -- path is explicit CLI input.
	}
	if err != nil {
		return "", false, fmt.Errorf("read body: %w", err)
	}
	return string(data), true, nil
}
