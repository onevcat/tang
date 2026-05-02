// Package cli builds the tang command tree.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// BuildInfo contains values injected at build time.
type BuildInfo struct {
	Version string
	Commit  string
}

// RootOptions contains global options shared by subcommands.
type RootOptions struct {
	JSONFields string
	Repo       string
	PDS        string
}

// NewRootCommand creates the Cobra command tree.
func NewRootCommand(build BuildInfo) *cobra.Command {
	opts := &RootOptions{}
	cmd := &cobra.Command{
		Use:           "tang",
		Short:         "A command-line client for Tangled",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&opts.JSONFields, "json", "", "Output JSON, optionally filtered by comma-separated fields")
	cmd.PersistentFlags().StringVarP(&opts.Repo, "repo", "R", "", "Select another repository using [HOST/]OWNER/REPO")
	cmd.PersistentFlags().StringVar(&opts.PDS, "pds", "", "Override PDS URL for auth and testing")

	cmd.AddCommand(newVersionCommand(build))
	cmd.AddCommand(newCompletionCommand(cmd))
	return cmd
}

func newVersionCommand(build BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "tang version %s (%s)\n", build.Version, build.Commit)
			return err
		},
	}
}

func newCompletionCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletion(out)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletion(out)
			default:
				return fmt.Errorf("unsupported shell %q", args[0])
			}
		},
	}
	return cmd
}

func executeForTest(cmd *cobra.Command, out io.Writer, args ...string) error {
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	return cmd.Execute()
}
