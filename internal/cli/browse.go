package cli

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/repo"
	"tangled.org/onev.cat/tang/internal/tangled"
)

func newBrowseCommand(opts *RootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Open Tangled pages in a browser",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, context, err := browseContext(cmd)
			if err != nil {
				return err
			}
			return openBrowserForCLI(repoURL(cfg, context))
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "issue <issue>",
		Short: "Open an issue in a browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, context, err := browseContext(cmd)
			if err != nil {
				return err
			}
			service := tangled.NewIssueService(cfg, nil)
			repoURI, err := tangled.BuildRepoATURI(cmd.Context(), context)
			if err != nil {
				return err
			}
			issues, err := service.ListIssues(cmd.Context(), repoURI, tangled.IssueListOptions{State: "all", Limit: 100})
			if err != nil {
				return err
			}
			issue, err := tangled.ResolveIssueIdentifier(args[0], issues)
			if err != nil {
				return err
			}
			return openBrowserForCLI(issueURL(cfg, context, issue))
		},
	})
	_ = opts
	return cmd
}

func browseContext(cmd *cobra.Command) (*config.Config, *repo.RepositoryContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	context, err := repo.Resolve(cmd.Context(), ".", cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, context, nil
}

func repoURL(cfg *config.Config, context *repo.RepositoryContext) string {
	return strings.TrimRight(cfg.AppView.URL, "/") + "/" + url.PathEscape(context.Owner) + "/" + url.PathEscape(context.Name)
}

func issueURL(cfg *config.Config, context *repo.RepositoryContext, issue tangled.Issue) string {
	base := strings.TrimRight(cfg.AppView.URL, "/")
	number := issue.Number
	if number <= 0 {
		number = 1
	}
	return base + "/" + url.PathEscape(context.Owner) + "/" + url.PathEscape(context.Name) + "/issues/" + strconv.Itoa(number)
}

func openBrowser(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target) // #nosec G204 -- browser opener is fixed; URL is a Tangled page.
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target) // #nosec G204 -- browser opener is fixed; URL is a Tangled page.
	default:
		cmd = exec.Command("xdg-open", target) // #nosec G204 -- browser opener is fixed; URL is a Tangled page.
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	return nil
}
