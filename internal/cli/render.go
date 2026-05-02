package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/output"
)

func renderJSONIfRequested(cmd *cobra.Command, opts *RootOptions, value any) (bool, error) {
	flag := cmd.Root().PersistentFlags().Lookup("json")
	if flag == nil || !flag.Changed {
		return false, nil
	}
	fields := opts.JSONFields
	if fields == "*" {
		fields = ""
	}
	fields = strings.TrimSpace(fields)
	return true, output.NewJSONRenderer(fields).Render(cmd.OutOrStdout(), value)
}
