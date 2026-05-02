package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/config"
)

func newConfigCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and write tang configuration",
	}
	cmd.AddCommand(newConfigGetCommand(opts))
	cmd.AddCommand(newConfigSetCommand())
	cmd.AddCommand(newConfigListCommand(opts))
	return cmd
}

func newConfigGetCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			value, err := cfg.Get(args[0])
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, map[string]any{args[0]: value}); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), formatConfigValue(value))
			return err
		},
	}
}

func newConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Set %s\n", args[0])
			return err
		},
	}
}

func newConfigListCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List config values",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			values := cfg.List()
			if rendered, err := renderJSONIfRequested(cmd, opts, values); rendered || err != nil {
				return err
			}
			for _, key := range []string{"knot.hosts", "constellation.url", "appview.url", "clone.protocol", "remote"} {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", key, formatConfigValue(values[key])); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func formatConfigValue(value any) string {
	switch v := value.(type) {
	case []string:
		return strings.Join(v, ",")
	default:
		return fmt.Sprint(v)
	}
}
