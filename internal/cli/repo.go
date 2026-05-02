package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	localrepo "tangled.org/onev.cat/tang/internal/repo"
	"tangled.org/onev.cat/tang/internal/tangled"
)

func newRepoCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "repo", Short: "Manage Tangled repositories"}
	cmd.AddCommand(newRepoViewCommand(opts))
	cmd.AddCommand(newRepoListCommand(opts))
	cmd.AddCommand(newRepoCreateCommand(opts))
	cmd.AddCommand(newRepoCloneCommand())
	return cmd
}

func newRepoViewCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "view [owner/name]",
		Short: "View repository information",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			owner, name, err := repoSelector(cmd, cfg, args)
			if err != nil {
				return err
			}
			repo, err := tangled.NewRepoService(cfg, nil).GetRepo(cmd.Context(), owner, name)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, repo); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s/%s\nKnot: %s\nSSH: %s\nHTTPS: %s\n", repo.Owner, repo.Name, repo.Knot, repo.CloneSSH, repo.CloneHTTPS)
			return err
		},
	}
}

func newRepoListCommand(opts *RootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list [owner]",
		Short: "List repositories",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			} else if session, err := auth.Load(); err == nil {
				owner = session.Handle
			}
			if owner == "" {
				return fmt.Errorf("owner is required when not authenticated")
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			repos, err := tangled.NewRepoService(cfg, nil).ListRepos(cmd.Context(), owner)
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, repos); rendered || err != nil {
				return err
			}
			for _, repo := range repos {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s/%s\t%s\n", repo.Owner, repo.Name, repo.Knot); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newRepoCreateCommand(opts *RootOptions) *cobra.Command {
	var description, knot, defaultBranch string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := auth.Load()
			if err != nil {
				return err
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			repo, err := tangled.NewRepoService(cfg, nil).CreateRepo(cmd.Context(), session, tangled.CreateRepoOptions{
				Name:          args[0],
				Description:   description,
				Knot:          knot,
				DefaultBranch: defaultBranch,
			})
			if err != nil {
				return err
			}
			if rendered, err := renderJSONIfRequested(cmd, opts, repo); rendered || err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created repository %s/%s\n", repo.Owner, repo.Name)
			return err
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "Repository description")
	cmd.Flags().StringVar(&knot, "knot", "", "Knot host")
	cmd.Flags().StringVar(&defaultBranch, "default-branch", "main", "Default branch")
	return cmd
}

func newRepoCloneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <owner/name> [dir]",
		Short: "Clone a repository",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, name, err := splitOwnerRepo(args[0])
			if err != nil {
				return err
			}
			dir := ""
			if len(args) > 1 {
				dir = args[1]
			}
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return tangled.NewRepoService(cfg, nil).Clone(cmd.Context(), owner, name, dir)
		},
	}
}

func repoSelector(cmd *cobra.Command, cfg *config.Config, args []string) (string, string, error) {
	if len(args) > 0 {
		return splitOwnerRepo(args[0])
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	context, err := localrepo.Resolve(cmd.Context(), cwd, cfg)
	if err != nil {
		return "", "", err
	}
	return context.Owner, context.Name, nil
}

func splitOwnerRepo(input string) (string, string, error) {
	parts := strings.Split(strings.Trim(input, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be OWNER/NAME")
	}
	return parts[0], parts[1], nil
}
